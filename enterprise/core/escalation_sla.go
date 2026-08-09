package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Escalation Engine ──────────────────────────────────────────────────
// Phase V: The escalation engine automatically escalates overdue
// activities through a configurable chain. The chain is reusable
// and role-aware.

// runEscalationSweep checks for overdue workflow instances and escalates them.
func runEscalationSweep() {
	now := time.Now()

	workflowMu.RLock()
	overdue := make([]WorkflowInstance, 0)
	for _, inst := range workflowInstances {
		if inst.Status != "running" && inst.Status != "waiting_task" && inst.Status != "waiting_approval" {
			continue
		}
		if inst.DueAt == "" {
			continue
		}
		due, err := time.Parse(time.RFC3339, inst.DueAt)
		if err != nil {
			continue
		}
		if now.After(due) {
			overdue = append(overdue, inst)
		}
	}
	workflowMu.RUnlock()

	for _, inst := range overdue {
		escalateWorkflowInstance(inst)
	}
}

// escalateWorkflowInstance escalates an overdue workflow instance.
func escalateWorkflowInstance(inst WorkflowInstance) {
	// Get escalation policy from workflow definition
	def, ok := getWorkflowDefinition(inst.WorkflowID)
	if !ok || def.Escalation == nil || !def.Escalation.Enabled {
		// No escalation policy - mark as escalated with a generic notification
		escalateGeneric(inst)
		return
	}

	policy := def.Escalation
	if len(policy.Levels) == 0 {
		escalateGeneric(inst)
		return
	}

	// Determine current escalation level based on overdue time
	overdueHours := 0
	if due, err := time.Parse(time.RFC3339, inst.DueAt); err == nil {
		overdueHours = int(time.Since(due).Hours())
	}

	level := 0
	for _, l := range policy.Levels {
		if overdueHours >= l.AfterHours {
			level = l.Level
		}
	}

	if level > inst.EscalationLevel {
		// Find the matching level
		for _, l := range policy.Levels {
			if l.Level == level {
				applyEscalationLevel(inst, l)
				break
			}
		}
	}
}

func escalateGeneric(inst WorkflowInstance) {
	// Notify assignee about overdue
	if inst.Assignee != "" {
		_, _ = createNotificationRecord(Notification{
			UserID:         inst.Assignee,
			Title:          "Workflow Overdue",
			Body:           fmt.Sprintf("Workflow %s is overdue", inst.WorkflowName),
			Priority:       "high",
			Category:       "escalation",
			SourceApp:      "enterprise",
			SourceEntity:   "workflow",
			SourceEntityID: inst.ID,
			DeepLink:       "/action-centre/workflows/" + inst.ID,
			Metadata: map[string]interface{}{
				"workflow_id": inst.WorkflowID, "instance_id": inst.ID,
				"escalation_level": inst.EscalationLevel,
			},
		})
	}
}

// applyEscalationLevel executes an escalation level action.
func applyEscalationLevel(inst WorkflowInstance, level EscalationLevel) {
	// Increment escalation level
	inst.EscalationLevel = level.Level
	inst.Status = "escalated"
	updateWorkflowInstance(inst)

	// Execute actions
	recipient := level.NotifyUser
	if recipient == "" && level.NotifyRole != "" {
		// In production this would resolve the role to users
		// For now, use a role-based fallback
		recipient = getEnvValue("STATGATE_ROLE_USER_" + level.NotifyRole)
	}

	if recipient != "" {
		_, _ = createNotificationRecord(Notification{
			UserID:         recipient,
			Title:          "Workflow Escalated",
			Body:           fmt.Sprintf("Workflow %s has been escalated (Level %d): %s", inst.WorkflowName, level.Level, level.Message),
			Priority:       "critical",
			Category:       "escalation",
			SourceApp:      "enterprise",
			SourceEntity:   "workflow",
			SourceEntityID: inst.ID,
			DeepLink:       "/action-centre/workflows/" + inst.ID,
			Metadata: map[string]interface{}{
				"workflow_id": inst.WorkflowID, "instance_id": inst.ID,
				"escalation_level": level.Level,
			},
		})
	}

	// Record audit
	recordAudit("workflow.escalated", "enterprise", inst.Assignee, map[string]interface{}{
		"instance_id": inst.ID, "workflow_id": inst.WorkflowID,
		"escalation_level": level.Level, "notify_role": level.NotifyRole,
		"notify_user": level.NotifyUser, "message": level.Message,
	})

	// Publish analytics event
	publishAnalyticsEvent("workflow.escalated", map[string]interface{}{
		"instance_id": inst.ID, "workflow_id": inst.WorkflowID,
		"escalation_level": level.Level, "message": level.Message,
	})
}

// ─── SLA Management ──────────────────────────────────────────────────
// SLA management supports response, resolution, approval, review and
// project SLAs. SLA breaches automatically generate alerts and
// escalation workflows.

func registerSLAConfig(sla SLAConfig) {
	workflowMu.Lock()
	defer workflowMu.Unlock()
	if sla.ID == "" {
		sla.ID = fmt.Sprintf("sla_%d", time.Now().UnixNano())
	}
	if sla.CreatedAt == "" {
		sla.CreatedAt = nowUTC()
	}
	slaConfigs[sla.ID] = sla
}

func listSLAConfigs() []SLAConfig {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	out := make([]SLAConfig, 0, len(slaConfigs))
	for _, sla := range slaConfigs {
		out = append(out, sla)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// bootstrapSLAs registers initial SLA configurations.
func bootstrapSLAs() {
	now := nowUTC()
	configs := []SLAConfig{
		{
			ID: "sla_helpdesk_response", Name: "HelpDesk Response SLA",
			Type: "response", TargetHours: 4, WarningHours: 3,
			Enabled: true, CreatedAt: now,
		},
		{
			ID: "sla_helpdesk_resolution", Name: "HelpDesk Resolution SLA",
			Type: "resolution", TargetHours: 48, WarningHours: 36,
			Enabled: true, CreatedAt: now,
		},
		{
			ID: "sla_approval", Name: "Approval SLA",
			Type: "approval", TargetHours: 72, WarningHours: 48,
			Enabled: true, CreatedAt: now,
		},
		{
			ID: "sla_research_review", Name: "Research Review SLA",
			Type: "review", TargetHours: 120, WarningHours: 96,
			Enabled: true, CreatedAt: now,
		},
		{
			ID: "sla_project", Name: "Project Milestone SLA",
			Type: "project", TargetHours: 168, WarningHours: 120,
			Enabled: true, CreatedAt: now,
		},
	}
	for _, sla := range configs {
		registerSLAConfig(sla)
	}
}

// runSLACompliance checks for SLA breaches.
func runSLACompliance() {
	slaConfigs := listSLAConfigs()

	// Check HelpDesk resolution SLA
	for _, sla := range slaConfigs {
		if !sla.Enabled {
			continue
		}
		switch sla.Type {
		case "resolution":
			checkHelpDeskResolutionSLA(sla)
		case "approval":
			checkApprovalSLA(sla)
		}
	}
}

func checkHelpDeskResolutionSLA(sla SLAConfig) {
	// Fetch open tickets from HelpDesk
	tickets := fetchJSONArray(getEnv("HELPDESK_API_URL", "http://localhost:5006") + "/api/tickets?status=open&limit=100")
	for _, ticket := range tickets {
		id, _ := ticket["id"].(string)
		if id == "" {
			continue
		}
		createdAt, _ := ticket["created_at"].(string)
		if createdAt == "" {
			continue
		}
		createdTime, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			continue
		}
		ageHours := time.Since(createdTime).Hours()

		// Check for warning
		if ageHours >= float64(sla.WarningHours) && ageHours < float64(sla.TargetHours) {
			createSLAWarning(sla, "ticket", id, createdAt)
		}
		// Check for breach
		if ageHours >= float64(sla.TargetHours) {
			createSLABreach(sla, "ticket", id, createdAt)
		}
	}
}

func checkApprovalSLA(sla SLAConfig) {
	workflowMu.RLock()
	pending := make([]ApprovalRequest, 0)
	for _, req := range approvalRequests {
		if req.Status == "pending" && req.ExpiresAt != "" {
			if expiry, err := time.Parse(time.RFC3339, req.ExpiresAt); err == nil {
				hoursLeft := time.Until(expiry).Hours()
				if hoursLeft <= float64(sla.WarningHours) && hoursLeft > 0 {
					pending = append(pending, req)
				}
			}
		}
	}
	workflowMu.RUnlock()

	for _, req := range pending {
		createSLAWarning(sla, "approval", req.ID, req.CreatedAt)
	}
}

func createSLAWarning(sla SLAConfig, entityType, entityID, startedAt string) {
	breachID := fmt.Sprintf("sla_%s_%s", sla.ID, entityID)

	workflowMu.RLock()
	existing, exists := slaBreaches[breachID]
	workflowMu.RUnlock()
	if exists && existing.Status != "resolved" {
		// Already warned or breached
		return
	}

	dueAt := startedAt
	if t, err := time.Parse(time.RFC3339, startedAt); err == nil {
		dueAt = t.Add(time.Duration(sla.TargetHours) * time.Hour).Format(time.RFC3339)
	}

	breach := SLABreach{
		ID:         breachID,
		SLAID:      sla.ID,
		SLAName:    sla.Name,
		Type:       sla.Type,
		EntityType: entityType,
		EntityID:   entityID,
		Status:     "warning",
		StartedAt:  startedAt,
		DueAt:      dueAt,
		Metadata: map[string]interface{}{
			"warning_hours": sla.WarningHours, "target_hours": sla.TargetHours,
		},
	}
	workflowMu.Lock()
	slaBreaches[breachID] = breach
	workflowMu.Unlock()

	// Create alert notification
	notifySLA("SLA Warning", fmt.Sprintf("SLA %s is approaching the deadline for %s %s", sla.Name, entityType, entityID), "high", sla, entityType, entityID)
}

func createSLABreach(sla SLAConfig, entityType, entityID, startedAt string) {
	breachID := fmt.Sprintf("sla_%s_%s", sla.ID, entityID)

	workflowMu.RLock()
	existing, exists := slaBreaches[breachID]
	workflowMu.RUnlock()
	if exists && existing.Status == "breached" {
		return
	}

	dueAt := startedAt
	if t, err := time.Parse(time.RFC3339, startedAt); err == nil {
		dueAt = t.Add(time.Duration(sla.TargetHours) * time.Hour).Format(time.RFC3339)
	}

	breach := SLABreach{
		ID:         breachID,
		SLAID:      sla.ID,
		SLAName:    sla.Name,
		Type:       sla.Type,
		EntityType: entityType,
		EntityID:   entityID,
		Status:     "breached",
		StartedAt:  startedAt,
		BreachedAt: nowUTC(),
		DueAt:      dueAt,
		Metadata: map[string]interface{}{
			"target_hours": sla.TargetHours,
		},
	}
	workflowMu.Lock()
	slaBreaches[breachID] = breach
	workflowMu.Unlock()

	// Create alert notification
	notifySLA("SLA Breached", fmt.Sprintf("SLA %s has been breached for %s %s", sla.Name, entityType, entityID), "critical", sla, entityType, entityID)

	// Trigger escalation workflow
	triggerSLAEscalation(sla, breach)

	// Publish analytics event
	publishAnalyticsEvent("sla.breached", map[string]interface{}{
		"sla_id": sla.ID, "sla_name": sla.Name, "type": sla.Type,
		"entity_type": entityType, "entity_id": entityID,
	})
}

func notifySLA(title, body, priority string, sla SLAConfig, entityType, entityID string) {
	// Notify appropriate users
	recipients := []string{}
	// Default: notify admin and SLA-aware users in production
	if v := getEnvValue("STATGATE_SLA_NOTIFY"); v != "" {
		recipients = append(recipients, v)
	}

	for _, recipient := range recipients {
		if recipient == "" {
			continue
		}
		_, _ = createNotificationRecord(Notification{
			UserID:         recipient,
			Title:          title,
			Body:           body,
			Priority:       priority,
			Category:       "sla",
			SourceApp:      "enterprise",
			SourceEntity:   entityType,
			SourceEntityID: entityID,
			Metadata: map[string]interface{}{
				"sla_id": sla.ID, "sla_name": sla.Name, "type": sla.Type,
			},
		})
	}
}

func triggerSLAEscalation(sla SLAConfig, breach SLABreach) {
	// Trigger the HelpDesk escalation workflow if applicable
	if sla.Type == "resolution" && sla.EscalationPolicy != "" {
		if policy, ok := getEscalationPolicy(sla.EscalationPolicy); ok && policy.Enabled {
			// Convert SLA breach into a workflow trigger
			publishEvent(DomainEvent{
				ID:         fmt.Sprintf("evt_%d", time.Now().UnixNano()),
				EventType:  "sla.breached",
				Source:     "enterprise",
				ObjectType: "sla",
				ObjectID:   breach.ID,
				Payload: map[string]interface{}{
					"sla_id": sla.ID, "sla_name": sla.Name, "entity_type": breach.EntityType,
					"entity_id": breach.EntityID,
				},
				Timestamp: nowUTC(),
			})
		}
	}
}

// ─── API Handlers ─────────────────────────────────────────────────

func handleListEscalationPolicies(c *gin.Context) {
	policies := listEscalationPolicies()
	c.JSON(200, gin.H{"count": len(policies), "policies": policies})
}

func handleCreateEscalationPolicy(c *gin.Context) {
	var policy EscalationPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(400, gin.H{"error": "invalid escalation policy", "detail": err.Error()})
		return
	}
	registerEscalationPolicy(policy)
	c.JSON(201, policy)
}

func handleListSLAs(c *gin.Context) {
	configs := listSLAConfigs()
	c.JSON(200, gin.H{"count": len(configs), "slas": configs})
}

func handleCreateSLA(c *gin.Context) {
	var sla SLAConfig
	if err := c.ShouldBindJSON(&sla); err != nil {
		c.JSON(400, gin.H{"error": "invalid SLA configuration", "detail": err.Error()})
		return
	}
	registerSLAConfig(sla)
	c.JSON(201, sla)
}

func handleListSLABreaches(c *gin.Context) {
	status := c.Query("status")
	workflowMu.RLock()
	breaches := make([]SLABreach, 0, len(slaBreaches))
	for _, b := range slaBreaches {
		if status != "" && b.Status != status {
			continue
		}
		breaches = append(breaches, b)
	}
	workflowMu.RUnlock()
	sort.Slice(breaches, func(i, j int) bool { return breaches[i].StartedAt > breaches[j].StartedAt })
	c.JSON(200, gin.H{"count": len(breaches), "breaches": breaches})
}

func handleResolveSLABreach(c *gin.Context) {
	id := c.Param("id")
	workflowMu.Lock()
	if breach, ok := slaBreaches[id]; ok {
		breach.Status = "resolved"
		slaBreaches[id] = breach
		workflowMu.Unlock()
		c.JSON(200, gin.H{"status": "resolved", "id": id})
		return
	}
	workflowMu.Unlock()
	c.JSON(404, gin.H{"error": "SLA breach not found"})
}

// ─── Calendar Event Record ─────────────────────────────────────────

// createCalendarEventRecord stores a calendar event.
func createCalendarEventRecord(event CalendarEvent) {
	if event.ID == "" {
		event.ID = fmt.Sprintf("cal_%d", time.Now().UnixNano())
	}
	if event.CreatedAt == "" {
		event.CreatedAt = nowUTC()
	}
	if redisClient != nil {
		data, _ := json.Marshal(event)
		ctx := context.Background()
		redisClient.RPush(ctx, "statgate:calendar", string(data))
		redisClient.LTrim(ctx, "statgate:calendar", 0, 1999)
	}
}

// ─── Escalation Policy Bootstrapping ─────────────────────────────

func bootstrapEscalationPolicies() {
	now := nowUTC()

	// Standard escalation chain: Deadline approaching → Reminder → Supervisor → Manager → Director
	registerEscalationPolicy(EscalationPolicy{
		ID:          "esc_standard_overdue",
		Name:        "Standard Overdue Escalation",
		Description: "Escalates overdue activities through supervisor, manager and director chain",
		Enabled:     true,
		Levels: []EscalationLevel{
			{Level: 1, AfterHours: 24, NotifyRole: "supervisor", Action: "notify",
				Message: "Activity is overdue. Supervisor notification sent."},
			{Level: 2, AfterHours: 48, NotifyRole: "manager", Action: "notify",
				Message: "Activity remains overdue. Manager notification sent."},
			{Level: 3, AfterHours: 72, NotifyRole: "director", Action: "critical",
				Message: "Critical escalation. Director notification sent."},
		},
		CreatedAt: now,
	})

	// Critical escalation for high-priority items
	registerEscalationPolicy(EscalationPolicy{
		ID:          "esc_critical",
		Name:        "Critical Overdue Escalation",
		Description: "Critical escalation for high-priority overdue activities",
		Enabled:     true,
		Levels: []EscalationLevel{
			{Level: 1, AfterHours: 12, NotifyRole: "manager", Action: "notify",
				Message: "Critical activity overdue. Manager notification sent."},
			{Level: 2, AfterHours: 24, NotifyRole: "director", Action: "critical",
				Message: "Critical escalation. Director notification sent."},
		},
		CreatedAt: now,
	})
}

// log-level helper
var _ = log.Printf
