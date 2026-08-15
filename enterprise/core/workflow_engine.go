package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── Workflow Engine ──────────────────────────────────────────────────
// Phase V: The centralized workflow engine transforms analytical
// intelligence into controlled organizational action. Workflows are
// reusable across applications, auditable, permission-aware,
// recoverable and human-supervised where necessary.

var (
	workflowMu          sync.RWMutex
	workflowDefinitions = make(map[string]WorkflowDefinition)
	workflowInstances   = make(map[string]WorkflowInstance)
	workflowExecutions  = make(map[string]WorkflowExecution)
	escalationPolicies  = make(map[string]EscalationPolicy)
	approvalRequests    = make(map[string]ApprovalRequest)
	enterpriseTasks     = make(map[string]EnterpriseTask)
	slaConfigs          = make(map[string]SLAConfig)
	slaBreaches         = make(map[string]SLABreach)
	decisionRecords     = make(map[string]DecisionRecord)
	aiRecommendations   = make(map[string]AIRecommendation)
)

// ─── Registry ─────────────────────────────────────────────────────────

func registerWorkflowDefinition(wf WorkflowDefinition) {
	workflowMu.Lock()
	defer workflowMu.Unlock()
	if wf.ID == "" {
		wf.ID = fmt.Sprintf("wf_%d", time.Now().UnixNano())
	}
	if wf.CreatedAt == "" {
		wf.CreatedAt = nowUTC()
	}
	wf.UpdatedAt = nowUTC()
	workflowDefinitions[wf.ID] = wf
}

func getWorkflowDefinition(id string) (WorkflowDefinition, bool) {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	wf, ok := workflowDefinitions[id]
	return wf, ok
}

func listWorkflowDefinitions(status string) []WorkflowDefinition {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	out := make([]WorkflowDefinition, 0, len(workflowDefinitions))
	for _, wf := range workflowDefinitions {
		if status != "" && wf.Status != status {
			continue
		}
		out = append(out, wf)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func registerEscalationPolicy(p EscalationPolicy) {
	workflowMu.Lock()
	defer workflowMu.Unlock()
	if p.ID == "" {
		p.ID = fmt.Sprintf("esc_%d", time.Now().UnixNano())
	}
	if p.CreatedAt == "" {
		p.CreatedAt = nowUTC()
	}
	escalationPolicies[p.ID] = p
}

func getEscalationPolicy(id string) (EscalationPolicy, bool) {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	p, ok := escalationPolicies[id]
	return p, ok
}

func listEscalationPolicies() []EscalationPolicy {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	out := make([]EscalationPolicy, 0, len(escalationPolicies))
	for _, p := range escalationPolicies {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ─── Workflow Triggering ──────────────────────────────────────────────
// processWorkflowForEvent evaluates domain events against active workflow
// definitions and starts instances where triggers match.

func processWorkflowForEvent(ev DomainEvent) {
	defs := listWorkflowDefinitions("active")
	for _, def := range defs {
		if def.Trigger.Type != "event" {
			continue
		}
		if def.Trigger.EventType != "" && def.Trigger.EventType != ev.EventType {
			continue
		}
		if def.Trigger.SourceApp != "" && def.Trigger.SourceApp != ev.Source {
			continue
		}

		// Evaluate workflow-level conditions
		if !evaluateWorkflowConditions(def, ev) {
			continue
		}

		// Start the workflow
		_, err := startWorkflow(def, "event", ev.ID, ev.ObjectType, ev.ObjectID, ev.ProjectID, ev.OrgID, ev.Actor, ev.Payload)
		if err != nil {
			log.Printf("workflow: failed to start %s for event %s: %v", def.ID, ev.EventType, err)
		}
	}
}

// processWorkflowForAlert starts workflows when analytical alerts fire.
func processWorkflowForAlert(alert AnalyticalAlert) {
	defs := listWorkflowDefinitions("active")
	for _, def := range defs {
		if def.Trigger.Type != "alert" {
			continue
		}
		if def.Trigger.AlertRuleID != "" && def.Trigger.AlertRuleID != alert.RuleID {
			continue
		}

		context := map[string]interface{}{
			"alert_id": alert.ID, "rule_id": alert.RuleID, "rule_name": alert.RuleName,
			"metric": alert.Metric, "value": alert.Value, "threshold": alert.Threshold,
			"priority": alert.Priority, "scope": alert.Scope, "scope_id": alert.ScopeID,
			"project_id": alert.ProjectID, "entity_type": alert.EntityType, "entity_id": alert.EntityID,
		}

		_, err := startWorkflow(def, "alert", alert.ID, alert.EntityType, alert.EntityID, alert.ProjectID, "", "", context)
		if err != nil {
			log.Printf("workflow: failed to start %s for alert %s: %v", def.ID, alert.ID, err)
		}
	}
}

// processWorkflowForAnomaly starts workflows when anomalies are detected.
func processWorkflowForAnomaly(anomaly AnomalyRecord) {
	defs := listWorkflowDefinitions("active")
	for _, def := range defs {
		if def.Trigger.Type != "anomaly" {
			continue
		}
		if def.Trigger.AnomalyRule != "" && def.Trigger.AnomalyRule != anomaly.RuleID {
			continue
		}

		context := map[string]interface{}{
			"anomaly_id": anomaly.ID, "rule_id": anomaly.RuleID, "rule_name": anomaly.RuleName,
			"type": anomaly.Type, "severity": anomaly.Severity, "scope": anomaly.Scope,
			"scope_id": anomaly.ScopeID, "project_id": anomaly.ProjectID,
			"description": anomaly.Description, "current_value": anomaly.CurrentValue,
			"expected_value": anomaly.ExpectedValue,
		}

		_, err := startWorkflow(def, "anomaly", anomaly.ID, anomaly.Scope, anomaly.ScopeID, anomaly.ProjectID, "", "", context)
		if err != nil {
			log.Printf("workflow: failed to start %s for anomaly %s: %v", def.ID, anomaly.ID, err)
		}
	}
}

// evaluateWorkflowConditions checks all configured conditions against the event.
func evaluateWorkflowConditions(def WorkflowDefinition, ev DomainEvent) bool {
	if len(def.Conditions) == 0 {
		return true
	}
	context := ev.Payload
	if context == nil {
		context = make(map[string]interface{})
	}
	context["event_type"] = ev.EventType
	context["source"] = ev.Source
	context["object_type"] = ev.ObjectType
	context["object_id"] = ev.ObjectID
	context["project_id"] = ev.ProjectID
	context["organization_id"] = ev.OrgID
	context["actor"] = ev.Actor

	for _, cond := range def.Conditions {
		if !evaluateWorkflowCondition(cond.Field, cond.Operator, cond.Value, context) {
			return false
		}
	}
	return true
}

func evaluateWorkflowCondition(field, operator string, value interface{}, context map[string]interface{}) bool {
	// Get the field value from context
	fieldValue, exists := context[field]
	if !exists {
		// Try nested lookup with dot notation
		parts := strings.Split(field, ".")
		current := interface{}(context)
		for _, part := range parts {
			m, ok := current.(map[string]interface{})
			if !ok {
				return false
			}
			current, exists = m[part]
			if !exists {
				return false
			}
		}
		fieldValue = current
	}

	switch operator {
	case "equals":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", value)
	case "not_equals":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", value)
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", fieldValue), fmt.Sprintf("%v", value))
	case "exists":
		return exists
	case "not_exists":
		return !exists
	case "in":
		if list, ok := value.([]interface{}); ok {
			for _, item := range list {
				if fmt.Sprintf("%v", item) == fmt.Sprintf("%v", fieldValue) {
					return true
				}
			}
		}
		return false
	case "greater_than":
		fv, ok1 := toFloat(fieldValue)
		vv, ok2 := toFloat(value)
		return ok1 && ok2 && fv > vv
	case "less_than":
		fv, ok1 := toFloat(fieldValue)
		vv, ok2 := toFloat(value)
		return ok1 && ok2 && fv < vv
	default:
		return true
	}
}

// ─── Workflow Instance Management ─────────────────────────────────────

func startWorkflow(def WorkflowDefinition, triggerType, triggerID, entityType, entityID, projectID, orgID, actor string, context map[string]interface{}) (*WorkflowInstance, error) {
	instance := WorkflowInstance{
		ID:             fmt.Sprintf("wfi_%d", time.Now().UnixNano()),
		WorkflowID:     def.ID,
		WorkflowName:   def.Name,
		Status:         "pending",
		TriggerEventID: triggerID,
		TriggerType:    triggerType,
		EntityType:     entityType,
		EntityID:       entityID,
		ProjectID:      projectID,
		OrgID:          orgID,
		Scope:          def.Scope,
		ScopeID:        def.ScopeID,
		StartedAt:      nowUTC(),
		Context:        context,
		Metadata:       def.Metadata,
	}

	// Determine assignee
	instance.Assignee = resolveAssignee(def.Assignment, actor, context)

	// Determine due date
	if def.Deadline != nil {
		due := time.Now().Add(time.Duration(def.Deadline.DurationHours) * time.Hour)
		instance.DueAt = due.UTC().Format(time.RFC3339)
	}

	workflowMu.Lock()
	workflowInstances[instance.ID] = instance
	workflowMu.Unlock()

	// Persist to Redis
	persistWorkflowInstance(instance)

	// Record audit trail
	recordAudit("workflow.started", "enterprise", actor, map[string]interface{}{
		"workflow_id": def.ID, "workflow_name": def.Name, "instance_id": instance.ID,
		"trigger_type": triggerType, "trigger_id": triggerID, "entity_type": entityType,
		"entity_id": entityID, "project_id": projectID,
	})

	// Create timeline entry
	recordTimelineEntry(TimelineEntry{
		ID:          fmt.Sprintf("tl_%d", time.Now().UnixNano()),
		User:        actor,
		Application: "enterprise",
		Entity:      "workflow",
		EntityID:    instance.ID,
		Action:      "workflow.started",
		Description: fmt.Sprintf("Workflow %s started", def.Name),
		ProjectID:   projectID,
		OrgID:       orgID,
		Metadata: map[string]interface{}{
			"workflow_id": def.ID, "trigger_type": triggerType, "trigger_id": triggerID,
		},
		Timestamp: nowUTC(),
	})

	// Publish analytics broadcast
	publishAnalyticsEvent("workflow.started", map[string]interface{}{
		"workflow_id": def.ID, "workflow_name": def.Name, "instance_id": instance.ID,
		"entity_type": entityType, "entity_id": entityID, "project_id": projectID,
	})

	// Execute workflow steps
	go executeWorkflowSteps(instance)

	return &instance, nil
}

func resolveAssignee(assignment WorkflowAssignment, actor string, context map[string]interface{}) string {
	switch assignment.Type {
	case "user":
		if assignment.UserID != "" {
			return assignment.UserID
		}
	case "project_owner":
		if actor != "" {
			return actor
		}
		if owner, ok := context["owner"].(string); ok {
			return owner
		}
		if uid, ok := context["user_id"].(string); ok {
			return uid
		}
	case "manager", "supervisor":
		// In production this would query the organizational hierarchy.
		// For now, use the assigner or fallback.
		if assignment.Fallback != "" {
			return assignment.Fallback
		}
	case "auto":
		// Auto-assignment logic would evaluate workload in production.
		if actor != "" {
			return actor
		}
	}
	if assignment.Fallback != "" {
		return assignment.Fallback
	}
	return ""
}

func persistWorkflowInstance(instance WorkflowInstance) {
	if redisClient == nil {
		return
	}
	data, _ := json.Marshal(instance)
	ctx := context.Background()
	key := fmt.Sprintf("statgate:workflow:instance:%s", instance.ID)
	redisClient.Set(ctx, key, string(data), 30*24*time.Hour)
	redisClient.LPush(ctx, "statgate:workflow:instances", instance.ID)
	redisClient.LTrim(ctx, "statgate:workflow:instances", 0, 999)
}

func updateWorkflowInstance(instance WorkflowInstance) {
	workflowMu.Lock()
	workflowInstances[instance.ID] = instance
	workflowMu.Unlock()
	persistWorkflowInstance(instance)
}

func getWorkflowInstance(id string) (WorkflowInstance, bool) {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	instance, ok := workflowInstances[id]
	return instance, ok
}

func listWorkflowInstances(status string) []WorkflowInstance {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	out := make([]WorkflowInstance, 0, len(workflowInstances))
	for _, inst := range workflowInstances {
		if status != "" && inst.Status != status {
			continue
		}
		out = append(out, inst)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt > out[j].StartedAt })
	return out
}

// ─── Workflow Step Execution ──────────────────────────────────────────

func executeWorkflowSteps(instance WorkflowInstance) {
	wf, ok := getWorkflowDefinition(instance.WorkflowID)
	if !ok {
		failWorkflowInstance(instance.ID, "workflow definition not found")
		return
	}

	// Mark as running
	instance.Status = "running"
	updateWorkflowInstance(instance)

	steps := wf.Steps
	sort.Slice(steps, func(i, j int) bool { return steps[i].Order < steps[j].Order })

	for _, step := range steps {
		// Check if workflow was cancelled or completed while running
		current, _ := getWorkflowInstance(instance.ID)
		if current.Status == "cancelled" || current.Status == "completed" || current.Status == "failed" {
			return
		}

		// Execute the step
		result := executeWorkflowStep(instance, wf, step)
		if !result {
			// Step failed - check retry
			if step.RetryCount > 0 {
				for attempt := 1; attempt <= step.RetryCount; attempt++ {
					execution := recordStepFailure(instance, wf, step, attempt)
					result = executeWorkflowStep(instance, wf, step)
					if result {
						markStepSucceeded(execution)
						break
					}
				}
			}
			if !result {
				execution := recordStepFailure(instance, wf, step, 0)
				if step.NextOnFail != "" {
					// Follow failure path
					instance.CurrentStep = step.NextOnFail
					updateWorkflowInstance(instance)
					continue
				}
				failWorkflowInstance(instance.ID, fmt.Sprintf("step %s failed: %s", step.Name, execution.Error))
				return
			}
		}

		// Update instance
		instance.CurrentStep = step.ID
		if step.Deadline != "" {
			instance.DueAt = step.Deadline
		}
		updateWorkflowInstance(instance)

		// Check if workflow requires waiting for approval
		if step.Type == "approval" {
			instance.Status = "waiting_approval"
			updateWorkflowInstance(instance)
			// Don't continue - wait for approval to complete
			// The approval completion handler will continue the workflow
			return
		}

		// Check if workflow requires waiting for task completion
		if step.Type == "task" {
			instance.Status = "waiting_task"
			updateWorkflowInstance(instance)
			// Task completion handler will continue
			return
		}

		// Check if step is a wait
		if step.Type == "wait" {
			if step.TimeoutSecs > 0 {
				time.Sleep(time.Duration(step.TimeoutSecs) * time.Second)
			}
		}
	}

	// All steps completed
	completeWorkflowInstance(instance.ID)
}

func executeWorkflowStep(instance WorkflowInstance, wf WorkflowDefinition, step WorkflowStep) bool {
	// Create execution record
	execution := WorkflowExecution{
		ID:            fmt.Sprintf("wfe_%d", time.Now().UnixNano()),
		InstanceID:    instance.ID,
		WorkflowID:    wf.ID,
		StepID:        step.ID,
		StepName:      step.Name,
		Action:        step.Action,
		Status:        "running",
		Timestamp:     nowUTC(),
		CorrelationID: instance.CorrelationID,
	}
	workflowMu.Lock()
	workflowExecutions[execution.ID] = execution
	workflowMu.Unlock()

	// Execute the action
	var err error
	switch step.Type {
	case "action":
		err = performWorkflowAction(instance, wf, step)
	case "approval":
		err = createApprovalForWorkflow(instance, wf, step)
	case "task":
		err = createTaskForWorkflow(instance, wf, step)
	case "notification":
		err = sendWorkflowNotification(instance, wf, step)
	case "condition":
		// Conditions are evaluated as branches
		if evaluateStepConditions(step, instance) {
			return true
		}
		return false
	case "wait":
		return true
	default:
		err = performWorkflowAction(instance, wf, step)
	}

	if err != nil {
		execution.Status = "failed"
		execution.Error = err.Error()
		execution.CompletedAt = nowUTC()
		workflowMu.Lock()
		workflowExecutions[execution.ID] = execution
		workflowMu.Unlock()
		recordAudit("workflow.step_failed", "enterprise", instance.Assignee, map[string]interface{}{
			"instance_id": instance.ID, "step_id": step.ID, "step_name": step.Name,
			"error": err.Error(), "workflow_id": wf.ID,
		})
		return false
	}

	execution.Status = "succeeded"
	execution.CompletedAt = nowUTC()
	workflowMu.Lock()
	workflowExecutions[execution.ID] = execution
	workflowMu.Unlock()

	recordAudit("workflow.step_succeeded", "enterprise", instance.Assignee, map[string]interface{}{
		"instance_id": instance.ID, "step_id": step.ID, "step_name": step.Name,
		"workflow_id": wf.ID, "action": step.Action,
	})
	return true
}

func recordStepFailure(instance WorkflowInstance, wf WorkflowDefinition, step WorkflowStep, attempt int) WorkflowExecution {
	execution := WorkflowExecution{
		ID:            fmt.Sprintf("wfe_%d", time.Now().UnixNano()),
		InstanceID:    instance.ID,
		WorkflowID:    wf.ID,
		StepID:        step.ID,
		StepName:      step.Name,
		Action:        step.Action,
		Status:        "retrying",
		RetryCount:    attempt,
		Timestamp:     nowUTC(),
		CorrelationID: instance.CorrelationID,
	}
	workflowMu.Lock()
	workflowExecutions[execution.ID] = execution
	workflowMu.Unlock()
	return execution
}

func markStepSucceeded(execution WorkflowExecution) {
	execution.Status = "succeeded"
	execution.CompletedAt = nowUTC()
	workflowMu.Lock()
	workflowExecutions[execution.ID] = execution
	workflowMu.Unlock()
}

func evaluateStepConditions(step WorkflowStep, inst WorkflowInstance) bool {
	// Steps without explicit conditions always execute
	if len(step.Params) == 0 {
		return true
	}
	// Check for a "condition" parameter
	if condRaw, ok := step.Params["condition"].(map[string]interface{}); ok {
		field, _ := condRaw["field"].(string)
		operator, _ := condRaw["operator"].(string)
		value := condRaw["value"]
		return evaluateWorkflowCondition(field, operator, value, inst.Context)
	}
	return true
}

// ─── Workflow Actions ─────────────────────────────────────────────────

func performWorkflowAction(instance WorkflowInstance, wf WorkflowDefinition, step WorkflowStep) error {
	switch step.Action {
	case "create_task":
		return createTaskForWorkflow(instance, wf, step)
	case "create_ticket":
		return createHelpDeskTicketForWorkflow(instance, wf, step)
	case "notify":
		return sendWorkflowNotification(instance, wf, step)
	case "create_chat":
		return openStatChatForWorkflow(instance, wf, step)
	case "create_calendar":
		return createCalendarForWorkflow(instance, wf, step)
	case "create_decision":
		return createDecisionForWorkflow(instance, wf, step)
	case "create_approval":
		return createApprovalForWorkflow(instance, wf, step)
	case "update_entity":
		return workflowUpdateEntity(instance, wf, step)
	case "wait":
		return workflowWait(instance, wf, step)
	default:
		// Generic action - log and continue
		recordAudit("workflow.action", "enterprise", instance.Assignee, map[string]interface{}{
			"instance_id": instance.ID, "workflow_id": wf.ID, "step_id": step.ID,
			"action": step.Action, "status": "executed",
		})
		return nil
	}
}

// createTaskForWorkflow creates an enterprise task from a workflow step.
func createTaskForWorkflow(inst WorkflowInstance, def WorkflowDefinition, step WorkflowStep) error {
	task := EnterpriseTask{
		ID:             fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Title:          getParamString(step.Params, "title", fmt.Sprintf("Task: %s", step.Name)),
		Description:    getParamString(step.Params, "description", def.Description),
		Status:         "pending",
		Priority:       getParamString(step.Params, "priority", "medium"),
		Assignee:       getParamString(step.Params, "assignee", inst.Assignee),
		Assigner:       inst.Assignee,
		SourceApp:      "enterprise",
		SourceEntity:   "workflow",
		SourceEntityID: inst.ID,
		ProjectID:      inst.ProjectID,
		OrgID:          inst.OrgID,
		WorkflowID:     def.ID,
		InstanceID:     inst.ID,
		CorrelationID:  inst.CorrelationID,
		Metadata:       step.Params,
		CreatedAt:      nowUTC(),
		UpdatedAt:      nowUTC(),
	}
	if due := getParamString(step.Params, "due_at", ""); due != "" {
		task.DueAt = due
	} else if inst.DueAt != "" {
		task.DueAt = inst.DueAt
	}
	createTaskRecord(task)
	return nil
}

// createHelpDeskTicketForWorkflow creates a HelpDesk ticket via its API.
func createHelpDeskTicketForWorkflow(inst WorkflowInstance, def WorkflowDefinition, step WorkflowStep) error {
	title := getParamString(step.Params, "title", fmt.Sprintf("Workflow: %s", def.Name))
	description := getParamString(step.Params, "description", def.Description)
	priority := getParamString(step.Params, "priority", "normal")

	payload := map[string]interface{}{
		"title":       title,
		"description": fmt.Sprintf("%s\n\nWorkflow: %s\nInstance: %s", description, def.Name, inst.ID),
		"priority":    priority,
		"category":    getParamString(step.Params, "category", "workflow"),
		"source":      "enterprise-workflow",
		"metadata": map[string]interface{}{
			"workflow_id": def.ID, "instance_id": inst.ID, "correlation_id": inst.CorrelationID,
			"project_id": inst.ProjectID,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = postJSON(getEnv("HELPDESK_API_URL", "http://localhost:5006")+"/api/tickets", data)
	return err
}

// sendWorkflowNotification sends a notification to the assignee or specified recipients.
func sendWorkflowNotification(inst WorkflowInstance, def WorkflowDefinition, step WorkflowStep) error {
	title := getParamString(step.Params, "title", fmt.Sprintf("Workflow: %s", def.Name))
	body := getParamString(step.Params, "body", def.Description)
	priority := getParamString(step.Params, "priority", "medium")

	recipients := getParamStringSlice(step.Params, "recipients")
	if len(recipients) == 0 && inst.Assignee != "" {
		recipients = []string{inst.Assignee}
	}
	if len(recipients) == 0 {
		return nil
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
			Category:       "workflow",
			SourceApp:      "enterprise",
			SourceEntity:   "workflow",
			SourceEntityID: inst.ID,
			DeepLink:       "/action-centre/workflows/" + inst.ID,
			Metadata: map[string]interface{}{
				"workflow_id": def.ID, "instance_id": inst.ID, "correlation_id": inst.CorrelationID,
			},
		})
	}
	return nil
}

// openStatChatForWorkflow creates a StatChat conversation via its API.
func openStatChatForWorkflow(inst WorkflowInstance, def WorkflowDefinition, step WorkflowStep) error {
	title := getParamString(step.Params, "title", fmt.Sprintf("Workflow: %s", def.Name))
	participants := getParamStringSlice(step.Params, "participants")
	if len(participants) == 0 && inst.Assignee != "" {
		participants = []string{inst.Assignee}
	}

	payload := map[string]interface{}{
		"title":        title,
		"participants": participants,
		"source":       "enterprise-workflow",
		"metadata": map[string]interface{}{
			"workflow_id": def.ID, "instance_id": inst.ID, "correlation_id": inst.CorrelationID,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = postJSON(getEnv("STATCHAT_API_URL", "http://localhost:4000")+"/v1/chats", data)
	return err
}

// createCalendarForWorkflow creates a calendar event.
func createCalendarForWorkflow(inst WorkflowInstance, def WorkflowDefinition, step WorkflowStep) error {
	title := getParamString(step.Params, "title", fmt.Sprintf("Workflow: %s", def.Name))
	start := getParamString(step.Params, "start", nowUTC())
	end := getParamString(step.Params, "end", start)
	participants := getParamStringSlice(step.Params, "participants")
	if len(participants) == 0 && inst.Assignee != "" {
		participants = []string{inst.Assignee}
	}

	event := CalendarEvent{
		ID:           fmt.Sprintf("cal_%d", time.Now().UnixNano()),
		Title:        title,
		Description:  getParamString(step.Params, "description", step.Name),
		Start:        start,
		End:          end,
		Category:     getParamString(step.Params, "category", "workflow"),
		SourceApp:    "enterprise",
		EntityID:     inst.ID,
		ProjectID:    inst.ProjectID,
		Participants: participants,
		Metadata: map[string]interface{}{
			"workflow_id": def.ID, "instance_id": inst.ID, "correlation_id": inst.CorrelationID,
		},
		CreatedAt: nowUTC(),
	}
	createCalendarEventRecord(event)
	return nil
}

// createDecisionForWorkflow creates a decision record.
func createDecisionForWorkflow(inst WorkflowInstance, def WorkflowDefinition, step WorkflowStep) error {
	decision := DecisionRecord{
		ID:                fmt.Sprintf("dec_%d", time.Now().UnixNano()),
		Decision:          getParamString(step.Params, "decision", step.Name),
		Date:              nowUTC(),
		DecisionMaker:     inst.Assignee,
		Context:           getParamString(step.Params, "context", def.Description),
		RelatedProject:    inst.ProjectID,
		ActionRequired:    getParamString(step.Params, "action_required", ""),
		ResponsiblePerson: inst.Assignee,
		Deadline:          getParamString(step.Params, "deadline", ""),
		Status:            "open",
		WorkflowID:        def.ID,
		InstanceID:        inst.ID,
		CorrelationID:     inst.CorrelationID,
		CreatedAt:         nowUTC(),
		UpdatedAt:         nowUTC(),
	}
	createDecisionRecord(decision)
	return nil
}

// createApprovalForWorkflow creates an approval request.
func createApprovalForWorkflow(inst WorkflowInstance, def WorkflowDefinition, step WorkflowStep) error {
	approvers := getParamStringSlice(step.Params, "approvers")
	if len(approvers) == 0 && def.Approval != nil {
		approvers = def.Approval.Approvers
	}
	if len(approvers) == 0 {
		return fmt.Errorf("no approvers configured for approval step %s", step.Name)
	}

	req := ApprovalRequest{
		ID:          fmt.Sprintf("apr_%d", time.Now().UnixNano()),
		Title:       getParamString(step.Params, "title", fmt.Sprintf("Approval: %s", step.Name)),
		Description: getParamString(step.Params, "description", def.Description),
		Type:        "single",
		Status:      "pending",
		EntityType:  inst.EntityType,
		EntityID:    inst.EntityID,
		ProjectID:   inst.ProjectID,
		OrgID:       inst.OrgID,
		Requester:   inst.Assignee,
		Approvers:   approvers,
		Context: map[string]interface{}{
			"workflow_id": def.ID, "instance_id": inst.ID, "correlation_id": inst.CorrelationID,
			"step_id": step.ID, "step_name": step.Name,
		},
		WorkflowID: def.ID,
		InstanceID: inst.ID,
		CreatedAt:  nowUTC(),
		UpdatedAt:  nowUTC(),
	}
	if def.Approval != nil {
		req.Type = def.Approval.Type
		req.MinApprovals = def.Approval.MinApprovals
		if def.Approval.ExpiryHours > 0 {
			req.ExpiresAt = time.Now().UTC().Add(time.Duration(def.Approval.ExpiryHours) * time.Hour).Format(time.RFC3339)
		}
	}
	if len(req.Approvers) > 0 {
		req.CurrentApprover = req.Approvers[0]
	}
	createApprovalRequest(req)

	// Notify approvers
	for _, approver := range approvers {
		if approver == "" {
			continue
		}
		_, _ = createNotificationRecord(Notification{
			UserID:         approver,
			Title:          "Approval Required",
			Body:           fmt.Sprintf("%s: %s", req.Title, req.Description),
			Priority:       "high",
			Category:       "approval",
			SourceApp:      "enterprise",
			SourceEntity:   "approval",
			SourceEntityID: req.ID,
			DeepLink:       "/action-centre/approvals/" + req.ID,
			Metadata: map[string]interface{}{
				"approval_id": req.ID, "workflow_id": def.ID, "instance_id": inst.ID,
				"correlation_id": inst.CorrelationID,
			},
		})
	}
	return nil
}

// workflowUpdateEntity updates an entity in a target application via its API.
func workflowUpdateEntity(inst WorkflowInstance, def WorkflowDefinition, step WorkflowStep) error {
	targetApp := step.TargetApp
	if targetApp == "" {
		return fmt.Errorf("update_entity requires target_app")
	}
	entityType := getParamString(step.Params, "entity_type", inst.EntityType)
	entityID := getParamString(step.Params, "entity_id", inst.EntityID)
	if entityID == "" {
		return fmt.Errorf("update_entity requires entity_id")
	}
	updates := map[string]interface{}{}
	if params, ok := step.Params["updates"].(map[string]interface{}); ok {
		updates = params
	}
	if len(updates) == 0 {
		return nil
	}

	var baseURL string
	switch targetApp {
	case "pms":
		baseURL = getEnv("PMS_API_URL", "http://localhost:8091")
	case "rms":
		baseURL = getEnv("RMS_API_URL", "http://localhost:8092")
	case "helpdesk":
		baseURL = getEnv("HELPDESK_API_URL", "http://localhost:5006")
	default:
		return fmt.Errorf("unsupported target app: %s", targetApp)
	}

	payload, err := json.Marshal(updates)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/%s/%s", baseURL, entityType, entityID)
	_, err = putJSON(url, payload)
	return err
}

// workflowWait waits for a specified duration.
func workflowWait(inst WorkflowInstance, def WorkflowDefinition, step WorkflowStep) error {
	seconds := getParamInt(step.Params, "seconds", 0)
	if seconds > 0 {
		time.Sleep(time.Duration(seconds) * time.Second)
	}
	return nil
}

// ─── Workflow Completion & Failure ─────────────────────────────────

func completeWorkflowInstance(instanceID string) {
	workflowMu.Lock()
	inst, ok := workflowInstances[instanceID]
	if !ok {
		workflowMu.Unlock()
		return
	}
	inst.Status = "completed"
	inst.CompletedAt = nowUTC()
	workflowInstances[instanceID] = inst
	workflowMu.Unlock()
	persistWorkflowInstance(inst)

	// Record audit
	recordAudit("workflow.completed", "enterprise", inst.Assignee, map[string]interface{}{
		"instance_id": inst.ID, "workflow_id": inst.WorkflowID, "workflow_name": inst.WorkflowName,
	})

	// Record timeline
	recordTimelineEntry(TimelineEntry{
		ID:          fmt.Sprintf("tl_%d", time.Now().UnixNano()),
		User:        inst.Assignee,
		Application: "enterprise",
		Entity:      "workflow",
		EntityID:    inst.ID,
		Action:      "workflow.completed",
		Description: fmt.Sprintf("Workflow %s completed", inst.WorkflowName),
		ProjectID:   inst.ProjectID,
		OrgID:       inst.OrgID,
		Metadata: map[string]interface{}{
			"workflow_id": inst.WorkflowID, "correlation_id": inst.CorrelationID,
		},
		Timestamp: nowUTC(),
	})

	// Publish analytics event
	publishAnalyticsEvent("workflow.completed", map[string]interface{}{
		"instance_id": inst.ID, "workflow_id": inst.WorkflowID, "workflow_name": inst.WorkflowName,
		"correlation_id": inst.CorrelationID, "project_id": inst.ProjectID,
	})
}

func failWorkflowInstance(instanceID, reason string) {
	workflowMu.Lock()
	inst, ok := workflowInstances[instanceID]
	if !ok {
		workflowMu.Unlock()
		return
	}
	inst.Status = "failed"
	inst.Error = reason
	workflowInstances[instanceID] = inst
	workflowMu.Unlock()
	persistWorkflowInstance(inst)

	// Record audit
	recordAudit("workflow.failed", "enterprise", inst.Assignee, map[string]interface{}{
		"instance_id": inst.ID, "workflow_id": inst.WorkflowID, "workflow_name": inst.WorkflowName,
		"error": reason, "correlation_id": inst.CorrelationID,
	})

	// Record dead-letter for recovery
	if redisClient != nil {
		entry := map[string]interface{}{
			"instance_id":    inst.ID,
			"workflow_id":    inst.WorkflowID,
			"workflow_name":  inst.WorkflowName,
			"error":          reason,
			"correlation_id": inst.CorrelationID,
			"timestamp":      nowUTC(),
		}
		data, _ := json.Marshal(entry)
		ctx := context.Background()
		redisClient.LPush(ctx, "statgate:workflow:dead-letter", string(data))
		redisClient.LTrim(ctx, "statgate:workflow:dead-letter", 0, 999)
	}

	// Notify assignee
	if inst.Assignee != "" {
		_, _ = createNotificationRecord(Notification{
			UserID:         inst.Assignee,
			Title:          "Workflow Failed",
			Body:           fmt.Sprintf("Workflow %s failed: %s", inst.WorkflowName, reason),
			Priority:       "high",
			Category:       "workflow",
			SourceApp:      "enterprise",
			SourceEntity:   "workflow",
			SourceEntityID: inst.ID,
			DeepLink:       "/action-centre/workflows/" + inst.ID,
			Metadata: map[string]interface{}{
				"workflow_id": inst.WorkflowID, "instance_id": inst.ID, "correlation_id": inst.CorrelationID,
			},
		})
	}

	// Publish analytics event
	publishAnalyticsEvent("workflow.failed", map[string]interface{}{
		"instance_id": inst.ID, "workflow_id": inst.WorkflowID, "workflow_name": inst.WorkflowName,
		"error": reason, "correlation_id": inst.CorrelationID,
	})
}

func updateWorkflowStatus(instanceID, status, currentStep string) {
	workflowMu.Lock()
	if inst, ok := workflowInstances[instanceID]; ok {
		inst.Status = status
		if currentStep != "" {
			inst.CurrentStep = currentStep
		}
		if status == "completed" {
			inst.CompletedAt = nowUTC()
		}
		workflowInstances[instanceID] = inst
		workflowMu.Unlock()
		persistWorkflowInstance(inst)
		return
	}
	workflowMu.Unlock()
}

// ─── Helper Functions ─────────────────────────────────────────────

func getParamString(params map[string]interface{}, key, def string) string {
	if v, ok := params[key].(string); ok && v != "" {
		return v
	}
	return def
}

func getParamStringSlice(params map[string]interface{}, key string) []string {
	if v, ok := params[key].([]interface{}); ok {
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	if v, ok := params[key].([]string); ok {
		return v
	}
	return nil
}

func getParamInt(params map[string]interface{}, key string, def int) int {
	if v, ok := params[key].(float64); ok {
		return int(v)
	}
	if v, ok := params[key].(int); ok {
		return v
	}
	if v, ok := params[key].(string); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func fetchMetricValue(metric, projectID string) float64 {
	def, ok := getKPIDefinition(metric)
	if !ok {
		return 0
	}
	val := computeKPI(def, def.Scope, projectID)
	return val.Value
}

func recordWorkflowExecution(exec WorkflowExecution) {
	workflowMu.Lock()
	workflowExecutions[exec.ID] = exec
	workflowMu.Unlock()
}

func recordTimelineEntry(entry TimelineEntry) {
	if redisClient == nil {
		return
	}
	data, _ := json.Marshal(entry)
	ctx := context.Background()
	redisClient.LPush(ctx, "statgate:timeline", string(data))
	redisClient.LTrim(ctx, "statgate:timeline", 0, 4999)
}

func putJSON(url string, body []byte) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", nil
	}
	if v, ok := result["id"].(string); ok {
		return v, nil
	}
	return "", nil
}

// ─── Workflow Templates ────────────────────────────────────────────
// bootstrapWorkflowTemplates registers reusable workflow templates.

func bootstrapWorkflowTemplates() {
	now := nowUTC()

	// ── Project Onboarding ──
	registerWorkflowDefinition(WorkflowDefinition{
		ID:          "wf_project_onboarding",
		Name:        "Project Onboarding",
		Description: "Initializes a new project: workspace, permissions, files, calendar, team, notifications",
		Category:    "project",
		Version:     1,
		Status:      "active",
		Trigger: WorkflowTrigger{
			Type:      "event",
			EventType: "project.created",
			SourceApp: "pms",
		},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "Notify Project Team", Type: "action", Action: "notify", Order: 1,
				Params: map[string]interface{}{
					"title": "Project Created", "body": "A new project has been created and is being onboarded.",
					"priority": "high",
				}},
			{ID: "s2", Name: "Create Project Calendar", Type: "action", Action: "create_calendar", Order: 2,
				Params: map[string]interface{}{
					"title": "Project Kickoff", "category": "milestone",
				}},
			{ID: "s3", Name: "Create Onboarding Task", Type: "action", Action: "create_task", Order: 3,
				Params: map[string]interface{}{
					"title": "Complete Project Onboarding", "description": "Set up project workspace, permissions and files.",
					"priority": "high",
				}},
		},
		Assignment: WorkflowAssignment{Type: "project_owner", AllowReassign: true},
		Deadline:   &WorkflowDeadline{DurationHours: 48, From: "start", ReminderHours: 24},
		Completion: &WorkflowCompletion{Type: "all_steps"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// ── Survey Launch ──
	registerWorkflowDefinition(WorkflowDefinition{
		ID:          "wf_survey_launch",
		Name:        "Survey Launch",
		Description: "Publishes a survey, assigns field workers, notifies teams and creates monitoring",
		Category:    "survey",
		Version:     1,
		Status:      "active",
		Trigger: WorkflowTrigger{
			Type:      "event",
			EventType: "survey.published",
			SourceApp: "pms",
		},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "Notify Field Teams", Type: "action", Action: "notify", Order: 1,
				Params: map[string]interface{}{
					"title": "Survey Published", "body": "A survey has been published. Field teams should begin data collection.",
					"priority": "high",
				}},
			{ID: "s2", Name: "Create Field Assignment", Type: "action", Action: "create_task", Order: 2,
				Params: map[string]interface{}{
					"title": "Begin Field Data Collection", "description": "Collect field data for the published survey.",
					"priority": "high",
				}},
			{ID: "s3", Name: "Create Monitoring Calendar", Type: "action", Action: "create_calendar", Order: 3,
				Params: map[string]interface{}{
					"title": "Survey Monitoring Review", "category": "milestone",
				}},
		},
		Assignment: WorkflowAssignment{Type: "manager", AllowReassign: true},
		Deadline:   &WorkflowDeadline{DurationHours: 72, From: "start", ReminderHours: 24},
		Completion: &WorkflowCompletion{Type: "all_steps"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// ── Data Quality Intervention ──
	registerWorkflowDefinition(WorkflowDefinition{
		ID:          "wf_data_quality_intervention",
		Name:        "Data Quality Intervention",
		Description: "Creates a corrective action when data quality issues are detected",
		Category:    "data_quality",
		Version:     1,
		Status:      "active",
		Trigger: WorkflowTrigger{
			Type:        "alert",
			AlertRuleID: "ar_data_quality_low",
		},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "Create Quality Alert", Type: "action", Action: "notify", Order: 1,
				Params: map[string]interface{}{
					"title": "Data Quality Issue Detected", "body": "Data quality has fallen below acceptable levels.",
					"priority": "high",
				}},
			{ID: "s2", Name: "Assign Quality Reviewer", Type: "action", Action: "create_task", Order: 2,
				Params: map[string]interface{}{
					"title": "Review Data Quality Issues", "description": "Investigate and correct data quality issues.",
					"priority": "high",
				}},
			{ID: "s3", Name: "Create Corrective Action", Type: "action", Action: "create_decision", Order: 3,
				Params: map[string]interface{}{
					"decision": "Data Quality Corrective Action", "context": "Data quality below threshold",
					"action_required": "Correct data quality issues and validate",
				}},
		},
		Assignment: WorkflowAssignment{Type: "manager", AllowReassign: true},
		Deadline:   &WorkflowDeadline{DurationHours: 48, From: "start", ReminderHours: 12},
		Completion: &WorkflowCompletion{Type: "metric_improved", Metric: "data_quality_score", Threshold: 80, Operator: "above"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// ── HelpDesk Escalation ──
	registerWorkflowDefinition(WorkflowDefinition{
		ID:          "wf_helpdesk_escalation",
		Name:        "HelpDesk Escalation",
		Description: "Escalates HelpDesk tickets approaching SLA breach",
		Category:    "helpdesk",
		Version:     1,
		Status:      "active",
		Trigger: WorkflowTrigger{
			Type:        "alert",
			AlertRuleID: "ar_helpdesk_sla",
		},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "Notify Ticket Assignee", Type: "action", Action: "notify", Order: 1,
				Params: map[string]interface{}{
					"title": "SLA Warning", "body": "A HelpDesk ticket is approaching its SLA deadline.",
					"priority": "high",
				}},
			{ID: "s2", Name: "Create Resolution Task", Type: "action", Action: "create_task", Order: 2,
				Params: map[string]interface{}{
					"title": "Resolve Ticket Before SLA Breach", "description": "Resolve the HelpDesk ticket before SLA breach.",
					"priority": "high",
				}},
			{ID: "s3", Name: "Escalate to Manager", Type: "action", Action: "notify", Order: 3,
				Params: map[string]interface{}{
					"title": "SLA Escalation", "body": "Ticket requires manager attention to avoid SLA breach.",
					"priority": "critical",
				}},
		},
		Assignment: WorkflowAssignment{Type: "manager", AllowReassign: true},
		Deadline:   &WorkflowDeadline{DurationHours: 24, From: "start", ReminderHours: 6},
		Completion: &WorkflowCompletion{Type: "all_steps"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// ── Research Approval ──
	registerWorkflowDefinition(WorkflowDefinition{
		ID:          "wf_research_approval",
		Name:        "Research Approval",
		Description: "Reviews and approves research submissions, then initializes the project",
		Category:    "research",
		Version:     1,
		Status:      "active",
		Trigger: WorkflowTrigger{
			Type:      "event",
			EventType: "research.stage_changed",
			SourceApp: "rms",
		},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "Review Research", Type: "approval", Action: "create_approval", Order: 1,
				Params: map[string]interface{}{
					"title": "Research Approval Required", "description": "Review the research submission for approval.",
				}},
			{ID: "s2", Name: "Notify Research Team", Type: "action", Action: "notify", Order: 2,
				Params: map[string]interface{}{
					"title": "Research Approved", "body": "Research has been approved. Project initialization can begin.",
					"priority": "high",
				}},
			{ID: "s3", Name: "Create Research Project Task", Type: "action", Action: "create_task", Order: 3,
				Params: map[string]interface{}{
					"title": "Initialize Research Project", "description": "Set up the research project workspace and team.",
					"priority": "high",
				}},
		},
		Assignment: WorkflowAssignment{Type: "manager", AllowReassign: true},
		Approval: &ApprovalConfig{
			Type: "single", MinApprovals: 1, ExpiryHours: 72,
			AllowReject: true, AllowChanges: true,
		},
		Deadline:   &WorkflowDeadline{DurationHours: 72, From: "start", ReminderHours: 24},
		Completion: &WorkflowCompletion{Type: "all_steps"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// ── Submission Decline Intervention (Closed-Loop) ──
	registerWorkflowDefinition(WorkflowDefinition{
		ID:          "wf_submission_decline_intervention",
		Name:        "Submission Decline Intervention",
		Description: "Closed-loop intervention when field submissions decline: detect, understand, decide, assign, act, monitor, verify, close",
		Category:    "data_quality",
		Version:     1,
		Status:      "active",
		Trigger: WorkflowTrigger{
			Type:        "alert",
			AlertRuleID: "ar_submission_drop",
		},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "Notify Responsible Team", Type: "action", Action: "notify", Order: 1,
				Params: map[string]interface{}{
					"title": "Field Submissions Declining", "body": "Field data submission volume has dropped significantly. Review affected facilities.",
					"priority": "high",
				}},
			{ID: "s2", Name: "Create Corrective Action", Type: "action", Action: "create_decision", Order: 2,
				Params: map[string]interface{}{
					"decision": "Field Submission Intervention", "context": "Submission volume below threshold",
					"action_required": "Investigate cause of reporting drop and implement corrective actions.",
				}},
			{ID: "s3", Name: "Assign District Team Task", Type: "action", Action: "create_task", Order: 3,
				Params: map[string]interface{}{
					"title": "Investigate Field Submission Decline", "description": "Review affected facilities and identify root cause of declining submission volume.",
					"priority": "high",
				}},
			{ID: "s4", Name: "Create HelpDesk Ticket if Technical", Type: "action", Action: "create_ticket", Order: 4,
				Params: map[string]interface{}{
					"title":       "Technical Issue: Field Submissions Declining",
					"description": "Create HelpDesk ticket if the decline is due to technical issues.",
				}},
			{ID: "s5", Name: "Open StatChat Coordination", Type: "action", Action: "create_chat", Order: 5,
				Params: map[string]interface{}{
					"title": "Field Submission Decline Coordination",
				}},
			{ID: "s6", Name: "Monitor Reporting Improvement", Type: "wait", Action: "wait", Order: 6,
				TimeoutSecs: 86400,
				Params: map[string]interface{}{
					"description": "Wait for the reporting period to pass, then verify improvement.",
				}},
			{ID: "s7", Name: "Verify Improvement and Close", Type: "action", Action: "notify", Order: 7,
				Params: map[string]interface{}{
					"title": "Corrective Action Closed", "body": "The field submission intervention has been completed. Verify reporting improvement.",
					"priority": "medium",
				}},
		},
		Assignment: WorkflowAssignment{Type: "manager", AllowReassign: true},
		Deadline:   &WorkflowDeadline{DurationHours: 168, From: "start", ReminderHours: 24},
		Completion: &WorkflowCompletion{
			Type:        "metric_improved",
			Metric:      "kpi_survey_completion_rate",
			Operator:    "greater_than",
			Threshold:   70,
			VerifyAfter: "7d",
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	// ── Phase VIII: Policy Approval Workflow ──
	registerWorkflowDefinition(WorkflowDefinition{
		ID:          "wf_policy_approval",
		Name:        "Policy Review & Approval",
		Description: "Multi-stage institutional policy review: department review -> compliance/legal review -> executive board approval -> publish",
		Category:    "governance",
		Version:     1,
		Status:      "active",
		Trigger: WorkflowTrigger{
			Type:      "event",
			EventType: "policy.created",
			SourceApp: "statgovernance",
		},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "Departmental Review", Type: "action", Action: "notify", Order: 1,
				Params: map[string]interface{}{
					"title": "New Policy Requiring Review", "body": "A new institutional policy has been drafted and requires review.",
					"priority": "high",
				}},
			{ID: "s2", Name: "Compliance & Legal Review", Type: "action", Action: "create_task", Order: 2,
				Params: map[string]interface{}{
					"title": "Perform Policy Compliance Analysis", "description": "Verify policy against Data Protection Act and institutional standards.",
					"priority": "high",
				}},
			{ID: "s3", Name: "Executive Board Approval", Type: "approval", Action: "create_approval", Order: 3,
				Params: map[string]interface{}{
					"title": "Policy Formal Determination & Approval", "description": "Review and grant institutional approval to the policy.",
				}},
		},
		Assignment: WorkflowAssignment{Type: "manager", AllowReassign: true},
		Approval: &ApprovalConfig{
			Type: "single", MinApprovals: 1, ExpiryHours: 120,
			AllowReject: true, AllowChanges: true,
		},
		Deadline:   &WorkflowDeadline{DurationHours: 120, From: "start", ReminderHours: 24},
		Completion: &WorkflowCompletion{Type: "all_steps"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// ── Phase VIII: Risk Escalation Workflow ──
	registerWorkflowDefinition(WorkflowDefinition{
		ID:          "wf_risk_escalation",
		Name:        "Risk Escalation & Treatment",
		Description: "Escalates Critical/High institutional risks to Management Board, opens StatChat coordination, and provisions mitigation tasks",
		Category:    "governance",
		Version:     1,
		Status:      "active",
		Trigger: WorkflowTrigger{
			Type:      "event",
			EventType: "risk.escalated",
			SourceApp: "statgovernance",
		},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "Notify Executive Board", Type: "action", Action: "notify", Order: 1,
				Params: map[string]interface{}{
					"title": "Critical Risk Escalated", "body": "A critical risk has been escalated for executive treatment determination.",
					"priority": "critical",
				}},
			{ID: "s2", Name: "Open StatChat Coordination", Type: "action", Action: "create_chat", Order: 2,
				Params: map[string]interface{}{
					"title": "Critical Risk Escalation Coordination",
				}},
			{ID: "s3", Name: "Create Treatment Action Task", Type: "action", Action: "create_task", Order: 3,
				Params: map[string]interface{}{
					"title": "Implement Risk Mitigation Controls", "description": "Execute risk treatment action plan and verify residual impact reduction.",
					"priority": "critical",
				}},
		},
		Assignment: WorkflowAssignment{Type: "manager", AllowReassign: true},
		Deadline:   &WorkflowDeadline{DurationHours: 48, From: "start", ReminderHours: 12},
		Completion: &WorkflowCompletion{Type: "all_steps"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// ── Phase VIII: Compliance Finding Resolution Workflow ──
	registerWorkflowDefinition(WorkflowDefinition{
		ID:          "wf_compliance_finding_resolution",
		Name:        "Compliance Finding Resolution (CAPA)",
		Description: "Tracks audit non-conformances from finding creation -> task remediation -> evidence submission -> verification -> closure",
		Category:    "governance",
		Version:     1,
		Status:      "active",
		Trigger: WorkflowTrigger{
			Type:      "event",
			EventType: "compliance.finding.created",
			SourceApp: "statgovernance",
		},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "Assign Remediation Task", Type: "action", Action: "create_task", Order: 1,
				Params: map[string]interface{}{
					"title": "Resolve Audit Finding (CAPA)", "description": "Implement corrective actions and gather verification evidence.",
					"priority": "high",
				}},
			{ID: "s2", Name: "Audit Verification Review", Type: "action", Action: "create_decision", Order: 2,
				Params: map[string]interface{}{
					"decision": "Finding Remediation Verification", "context": "Verification of CAPA evidence by lead auditor",
					"action_required": "Verify remediation effectiveness and record closure determination",
				}},
		},
		Assignment: WorkflowAssignment{Type: "manager", AllowReassign: true},
		Deadline:   &WorkflowDeadline{DurationHours: 168, From: "start", ReminderHours: 24},
		Completion: &WorkflowCompletion{Type: "all_steps"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	// ── Phase VIII: Control Failure Remediation Workflow ──
	registerWorkflowDefinition(WorkflowDefinition{
		ID:          "wf_control_failure_remediation",
		Name:        "Control Failure Remediation",
		Description: "Triggers on automated or manual control failure: evaluates risk impact, assigns emergency remediation, and schedules retest",
		Category:    "governance",
		Version:     1,
		Status:      "active",
		Trigger: WorkflowTrigger{
			Type:      "event",
			EventType: "control.failed",
			SourceApp: "statgovernance",
		},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "Alert Control Owner", Type: "action", Action: "notify", Order: 1,
				Params: map[string]interface{}{
					"title": "Internal Control Failure Detected", "body": "A key institutional control probe has reported failure.",
					"priority": "critical",
				}},
			{ID: "s2", Name: "Create Emergency Remediation Task", Type: "action", Action: "create_task", Order: 2,
				Params: map[string]interface{}{
					"title": "Fix Failed Control Mechanism", "description": "Restore control effectiveness and prepare evidence for re-testing.",
					"priority": "critical",
				}},
		},
		Assignment: WorkflowAssignment{Type: "manager", AllowReassign: true},
		Deadline:   &WorkflowDeadline{DurationHours: 24, From: "start", ReminderHours: 6},
		Completion: &WorkflowCompletion{Type: "all_steps"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})
}
