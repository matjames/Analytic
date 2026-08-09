package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Alert Engine ───────────────────────────────────────────────────
// Analytical conditions generate alerts. Alerts are configurable,
// permission-aware, prioritized, traceable and linked to the
// affected entity.

var (
	alertMu          sync.RWMutex
	alertRules       = make(map[string]AlertRule)
	analyticalAlerts = make(map[string]AnalyticalAlert)
)

// ─── Alert Rule Registry ───────────────────────────────────────────

func registerAlertRule(rule AlertRule) {
	alertMu.Lock()
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("ar_%d", time.Now().UnixNano())
	}
	if rule.CreatedAt == "" {
		rule.CreatedAt = nowUTC()
	}
	alertRules[rule.ID] = rule
	alertMu.Unlock()
}

func getAlertRule(id string) (AlertRule, bool) {
	alertMu.RLock()
	defer alertMu.RUnlock()
	rule, ok := alertRules[id]
	return rule, ok
}

func listAlertRules() []AlertRule {
	alertMu.RLock()
	defer alertMu.RUnlock()
	out := make([]AlertRule, 0, len(alertRules))
	for _, rule := range alertRules {
		out = append(out, rule)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// bootstrapAlertRules registers initial alert rules.
// These are definitions/thresholds — not hardcoded values.
func bootstrapAlertRules() {
	now := nowUTC()
	rules := []AlertRule{
		{
			ID: "ar_survey_reporting_low", Name: "Survey Reporting Below Threshold",
			Description: "Triggers when survey completion rate falls below the configured threshold",
			Metric:      "survey_completion_rate", Condition: "below", Threshold: 70,
			Priority: "high", Enabled: true,
			NotifTitle: "Survey reporting below target",
			NotifBody:  "Survey completion rate has fallen below the configured threshold.",
			CreatedAt:  now,
		},
		{
			ID: "ar_milestone_overdue", Name: "Project Milestone Overdue",
			Description: "Triggers when a project milestone is overdue",
			Metric:      "milestone_overdue", Condition: "overdue", Threshold: 0,
			Priority: "high", Enabled: true,
			NotifTitle: "Milestone overdue",
			NotifBody:  "A project milestone has passed without completion.",
			CreatedAt:  now,
		},
		{
			ID: "ar_data_quality_low", Name: "Data Quality Below Acceptable",
			Description: "Triggers when data quality score falls below acceptable threshold",
			Metric:      "data_quality_score", Condition: "below", Threshold: 80,
			Priority: "medium", Enabled: true,
			NotifTitle: "Data quality issue detected",
			NotifBody:  "Data quality has fallen below the acceptable threshold.",
			CreatedAt:  now,
		},
		{
			ID: "ar_helpdesk_sla", Name: "HelpDesk SLA Approaching Breach",
			Description: "Triggers when a HelpDesk ticket is approaching SLA breach",
			Metric:      "helpdesk_sla", Condition: "expiring", Threshold: 24,
			Priority: "high", Enabled: true,
			NotifTitle: "HelpDesk SLA approaching breach",
			NotifBody:  "A HelpDesk ticket is approaching its SLA deadline.",
			CreatedAt:  now,
		},
		{
			ID: "ar_research_approval_expiring", Name: "Research Approval Expiring",
			Description: "Triggers when a research approval is about to expire",
			Metric:      "research_approval", Condition: "expiring", Threshold: 7,
			Priority: "medium", Enabled: true,
			NotifTitle: "Research approval expiring",
			NotifBody:  "A research approval is approaching its expiry date.",
			CreatedAt:  now,
		},
		{
			ID: "ar_ticket_volume_spike", Name: "Ticket Volume Spike",
			Description: "Triggers when ticket volume increases unexpectedly",
			Metric:      "ticket_volume", Condition: "above", Threshold: 10,
			Priority: "medium", Enabled: true,
			NotifTitle: "Unexpected ticket volume increase",
			NotifBody:  "HelpDesk ticket volume has increased significantly.",
			CreatedAt:  now,
		},
		{
			ID: "ar_submission_drop", Name: "Submission Volume Drop",
			Description: "Triggers when submission volume drops significantly",
			Metric:      "submission_volume", Condition: "below", Threshold: 5,
			Priority: "high", Enabled: true,
			NotifTitle: "Field submissions dropped",
			NotifBody:  "Field data submission volume has dropped significantly.",
			CreatedAt:  now,
		},
		{
			ID: "ar_project_delay", Name: "Project Behind Schedule",
			Description: "Triggers when a project falls behind schedule",
			Metric:      "project_progress", Condition: "below", Threshold: 50,
			Priority: "high", Enabled: true,
			NotifTitle: "Project behind schedule",
			NotifBody:  "A project may be falling behind its expected progress.",
			CreatedAt:  now,
		},
	}

	for _, rule := range rules {
		registerAlertRule(rule)
	}
}

// ─── Alert Generation ──────────────────────────────────────────────
// evaluateAlertRule checks whether an alert condition is met.

func evaluateAlertRule(rule AlertRule, metricValue float64, scope string) bool {
	if !rule.Enabled {
		return false
	}
	switch rule.Condition {
	case "below":
		return metricValue < rule.Threshold
	case "above":
		return metricValue > rule.Threshold
	case "equals":
		return metricValue == rule.Threshold
	default:
		return false
	}
}

// createAnalyticalAlert records a new alert and delivers notifications.
func createAnalyticalAlert(rule AlertRule, value float64, scope, scopeID, entityType, entityID string, extra map[string]interface{}) AnalyticalAlert {
	alert := AnalyticalAlert{
		ID:         fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		Title:      rule.NotifTitle,
		Message:    rule.NotifBody,
		Priority:   rule.Priority,
		Status:     "open",
		Metric:     rule.Metric,
		Value:      value,
		Threshold:  rule.Threshold,
		Scope:      scope,
		ScopeID:    scopeID,
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  nowUTC(),
		Actions: []AlertAction{
			{Label: "View Analysis", Type: "deep_link", URL: "/analytics/kpi/" + rule.Metric},
			{Label: "Open StatChat", Type: "chat", URL: getEnv("STATCHAT_UI_URL", "http://localhost:3009")},
			{Label: "Create Ticket", Type: "ticket", URL: getEnv("HELPDESK_UI_URL", "http://localhost:3005") + "/tickets/new", Meta: map[string]interface{}{"title": rule.NotifTitle, "priority": rule.Priority}},
		},
		Metadata: extra,
	}
	if extra != nil {
		if pid, ok := extra["project_id"].(string); ok {
			alert.ProjectID = pid
		}
		if v, ok := extra["deep_link"].(string); ok && alert.DeepLink == "" {
			alert.DeepLink = v
		}
	}

	alertMu.Lock()
	analyticalAlerts[alert.ID] = alert
	alertMu.Unlock()

	// Persist to Redis
	persistAnalyticalAlert(alert)

	// Create enterprise notification for recipients
	notifyAlert(rule, alert)

	// Record audit
	recordAudit("alert.created", "enterprise", "", map[string]interface{}{
		"alert_id": alert.ID, "rule_id": rule.ID, "metric": rule.Metric,
		"value": value, "priority": rule.Priority, "scope": scope, "scope_id": scopeID,
	})

	// ── Phase V: Trigger workflows from analytical alerts ──
	processWorkflowForAlert(alert)

	return alert
}

func persistAnalyticalAlert(alert AnalyticalAlert) {
	if redisClient == nil {
		return
	}
	data, _ := json.Marshal(alert)
	ctx := context.Background()
	key := fmt.Sprintf("statgate:alerts:%s", alert.ID)
	redisClient.Set(ctx, key, string(data), 7*24*time.Hour)
	redisClient.LPush(ctx, "statgate:alerts:index", alert.ID)
	redisClient.LTrim(ctx, "statgate:alerts:index", 0, 999)
}

// notifyAlert delivers the alert as an enterprise notification.
func notifyAlert(rule AlertRule, alert AnalyticalAlert) {
	recipients := rule.Recipients
	if len(recipients) == 0 {
		// Default: send to admin and alert-aware users.
		// We use the X-Admin user convention, matching existing permission grants.
		if alert.ScopeID != "" {
			recipients = []string{alert.ScopeID}
		}
	}
	for _, recipient := range recipients {
		if recipient == "" {
			continue
		}
		actions := make([]NotificationAction, 0, len(alert.Actions))
		for _, a := range alert.Actions {
			actions = append(actions, NotificationAction{Label: a.Label, URL: a.URL, Type: a.Type})
		}
		_, _ = createNotificationRecord(Notification{
			UserID:         recipient,
			Title:          alert.Title,
			Body:           fmt.Sprintf("%s (Value: %.1f, Threshold: %.1f)", alert.Message, alert.Value, alert.Threshold),
			Priority:       alert.Priority,
			Category:       "analytics",
			SourceApp:      "enterprise",
			SourceEntity:   "alert",
			SourceEntityID: alert.ID,
			DeepLink:       alert.DeepLink,
			Actions:        actions,
			Metadata:       alert.Metadata,
		})
	}
}

// ─── Event-Driven Alert Processing ─────────────────────────────────
// processAlertRulesForEvent evaluates applicable rules when events arrive.
func processAlertRulesForEvent(ev DomainEvent) []map[string]interface{} {
	triggered := []map[string]interface{}{}

	// Check metric-specific rules based on event type
	switch ev.EventType {
	case "submission.received", "field_data.submitted":
		// Evaluate submission volume drop by inspecting enterprise layer
		checkSubmissionVolumeRules(&triggered, ev)
	case "ticket.raised", "ticket.updated":
		checkTicketVolumeRules(&triggered, ev)
	case "project.created", "project.updated":
		checkProjectProgressRule(&triggered, ev)
	case "submission.approved", "submission.rejected":
		checkDataQualityRule(&triggered, ev)
	}

	return triggered
}

func checkSubmissionVolumeRules(triggered *[]map[string]interface{}, ev DomainEvent) {
	rules := listAlertRules()
	for _, rule := range rules {
		if rule.Metric != "submission_volume" || !rule.Enabled {
			continue
		}
		// Count submissions in the last time window vs expected
		windowStart := time.Now().Add(-time.Duration(rule.Threshold) * time.Hour)
		recentCount := int64(0)
		dataLayerMu.RLock()
		for _, rec := range enterpriseRecords {
			if rec.SourceApp == "statcollect" && rec.SourceEntity == "submission" {
				if t, err := time.Parse(time.RFC3339, rec.Timestamp); err == nil {
					if t.After(windowStart) {
						recentCount++
					}
				}
			}
		}
		dataLayerMu.RUnlock()

		// Compare with the running average (if we have enough data)
		totalCount := int64(0)
		dataLayerMu.RLock()
		for _, rec := range enterpriseRecords {
			if rec.SourceApp == "statcollect" && rec.SourceEntity == "submission" {
				totalCount++
			}
		}
		dataLayerMu.RUnlock()

		if totalCount >= 3 && float64(recentCount) <= rule.Threshold {
			alert := createAnalyticalAlert(rule, float64(recentCount), "project", ev.ProjectID, ev.ObjectType, ev.ObjectID, map[string]interface{}{
				"project_id": ev.ProjectID, "event_type": ev.EventType,
				"recent_count": recentCount, "total_count": totalCount,
			})
			*triggered = append(*triggered, alertToMap(alert))
		}
	}
}

func checkTicketVolumeRules(triggered *[]map[string]interface{}, ev DomainEvent) {
	rules := listAlertRules()
	for _, rule := range rules {
		if rule.Metric != "ticket_volume" || !rule.Enabled {
			continue
		}
		windowStart := time.Now().Add(-24 * time.Hour)
		recentCount := int64(0)
		dataLayerMu.RLock()
		for _, rec := range enterpriseRecords {
			if rec.SourceApp == "helpdesk" && rec.SourceEntity == "ticket" {
				if t, err := time.Parse(time.RFC3339, rec.Timestamp); err == nil {
					if t.After(windowStart) {
						recentCount++
					}
				}
			}
		}
		dataLayerMu.RUnlock()

		if recentCount > int64(rule.Threshold) {
			alert := createAnalyticalAlert(rule, float64(recentCount), "organization", "", "ticket", ev.ObjectID, map[string]interface{}{
				"recent_count": recentCount, "event_type": ev.EventType,
			})
			*triggered = append(*triggered, alertToMap(alert))
		}
	}
}

func checkProjectProgressRule(triggered *[]map[string]interface{}, ev DomainEvent) {
	rules := listAlertRules()
	for _, rule := range rules {
		if rule.Metric != "project_progress" || !rule.Enabled {
			continue
		}
		// Fetch the project progress
		projectID := ev.ProjectID
		if projectID == "" {
			if v, ok := ev.Payload["project_id"].(string); ok {
				projectID = v
			}
		}
		if projectID == "" {
			projectID = ev.ObjectID
		}
		var progress float64
		if data := fetchJSON(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects/" + projectID); data != nil {
			if v, ok := data["progress"].(float64); ok {
				progress = v
			}
		}
		if progress > 0 && progress < rule.Threshold {
			alert := createAnalyticalAlert(rule, progress, "project", projectID, "project", projectID, map[string]interface{}{
				"project_id": projectID, "progress": progress,
			})
			*triggered = append(*triggered, alertToMap(alert))
		}
	}
}

func checkDataQualityRule(triggered *[]map[string]interface{}, ev DomainEvent) {
	rules := listAlertRules()
	for _, rule := range rules {
		if rule.Metric != "data_quality_score" || !rule.Enabled {
			continue
		}
		qc := runQuickQualityCheck(ev)
		if v, ok := qc["score"].(float64); ok && v < rule.Threshold {
			alert := createAnalyticalAlert(rule, v, "project", ev.ProjectID, ev.ObjectType, ev.ObjectID, map[string]interface{}{
				"project_id": ev.ProjectID, "event_type": ev.EventType,
				"quality_issues": qc["issues"],
			})
			*triggered = append(*triggered, alertToMap(alert))
		}
	}
}

func alertToMap(alert AnalyticalAlert) map[string]interface{} {
	return map[string]interface{}{
		"id": alert.ID, "rule_id": alert.RuleID, "rule_name": alert.RuleName,
		"title": alert.Title, "message": alert.Message, "priority": alert.Priority,
		"status": alert.Status, "metric": alert.Metric, "value": alert.Value,
		"threshold": alert.Threshold, "scope": alert.Scope, "scope_id": alert.ScopeID,
		"project_id": alert.ProjectID, "deep_link": alert.DeepLink,
		"actions": alert.Actions, "created_at": alert.CreatedAt,
	}
}

// ─── Scheduled Alert Sweep ─────────────────────────────────────────
// Periodic evaluation of KPI-based alert rules.

func runScheduledAlertSweep() {
	// Survey completion rate below threshold
	checkKPIBasedRule("ar_survey_reporting_low", "kpi_survey_completion_rate")
	checkKPIBasedRule("ar_data_quality_low", "kpi_data_quality_score")
	checkKPIBasedRule("ar_project_delay", "kpi_project_progress")
	checkKPIBasedRule("ar_ticket_volume_spike", "kpi_open_tickets")
}

func checkKPIBasedRule(ruleID, kpiID string) {
	rule, ok := getAlertRule(ruleID)
	if !ok || !rule.Enabled {
		return
	}
	def, ok := getKPIDefinition(kpiID)
	if !ok {
		return
	}
	value := computeKPI(def, def.Scope, "")
	if evaluateAlertRule(rule, value.Value, def.Scope) {
		alert := createAnalyticalAlert(rule, value.Value, def.Scope, "", def.Scope, "", map[string]interface{}{
			"kpi_id": kpiID, "kpi_name": def.Name,
		})
		publishAnalyticsEvent("alert.triggered", alertToMap(alert))
	}
}

// ─── API Handlers ──────────────────────────────────────────────────

func handleListAlertRules(c *gin.Context) {
	rules := listAlertRules()
	c.JSON(200, gin.H{"count": len(rules), "rules": rules})
}

func handleCreateAlertRule(c *gin.Context) {
	var rule AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(400, gin.H{"error": "invalid alert rule", "detail": err.Error()})
		return
	}
	registerAlertRule(rule)
	recordAudit("alert_rule.create", "enterprise", "", map[string]interface{}{
		"rule_id": rule.ID, "rule_name": rule.Name,
	})
	c.JSON(201, rule)
}

func handleListAlerts(c *gin.Context) {
	status := c.Query("status")
	priority := c.Query("priority")
	scope := c.Query("scope")
	scopeID := c.Query("scope_id")
	limit := parseIntDefault(c.Query("limit"), 50)

	alertMu.RLock()
	alerts := make([]AnalyticalAlert, 0, len(analyticalAlerts))
	for _, a := range analyticalAlerts {
		if status != "" && a.Status != status {
			continue
		}
		if priority != "" && a.Priority != priority {
			continue
		}
		if scope != "" && a.Scope != scope {
			continue
		}
		if scopeID != "" && a.ScopeID != scopeID {
			continue
		}
		alerts = append(alerts, a)
	}
	alertMu.RUnlock()

	sort.Slice(alerts, func(i, j int) bool { return alerts[i].CreatedAt > alerts[j].CreatedAt })
	if len(alerts) > limit {
		alerts = alerts[:limit]
	}
	c.JSON(200, gin.H{"count": len(alerts), "alerts": alerts})
}

func handleGetAlert(c *gin.Context) {
	id := c.Param("id")
	alertMu.RLock()
	alert, ok := analyticalAlerts[id]
	alertMu.RUnlock()
	if !ok {
		c.JSON(404, gin.H{"error": "alert not found"})
		return
	}
	c.JSON(200, alert)
}

func handleAcknowledgeAlert(c *gin.Context) {
	id := c.Param("id")
	alertMu.Lock()
	if alert, ok := analyticalAlerts[id]; ok {
		alert.Status = "acknowledged"
		analyticalAlerts[id] = alert
		alertMu.Unlock()
		c.JSON(200, gin.H{"status": "acknowledged", "id": id})
		return
	}
	alertMu.Unlock()
	c.JSON(404, gin.H{"error": "alert not found"})
}

func handleResolveAlert(c *gin.Context) {
	id := c.Param("id")
	alertMu.Lock()
	if alert, ok := analyticalAlerts[id]; ok {
		alert.Status = "resolved"
		alert.ResolvedAt = nowUTC()
		analyticalAlerts[id] = alert
		alertMu.Unlock()
		c.JSON(200, gin.H{"status": "resolved", "id": id})
		return
	}
	alertMu.Unlock()
	c.JSON(404, gin.H{"error": "alert not found"})
}

// handleAlertActions performs an operational action from an alert.
// POST /api/analytics/alerts/:id/actions
func handleAlertActions(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Action     string   `json:"action"`
		Recipients []string `json:"recipients,omitempty"`
		Notes      string   `json:"notes,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid action request"})
		return
	}

	alertMu.RLock()
	alert, ok := analyticalAlerts[id]
	alertMu.RUnlock()
	if !ok {
		c.JSON(404, gin.H{"error": "alert not found"})
		return
	}

	switch body.Action {
	case "create_ticket":
		// Create HelpDesk ticket via API
		created, err := createHelpDeskTicket(alert, body.Notes)
		if err != nil {
			c.JSON(502, gin.H{"error": "failed to create ticket", "detail": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ticket_created", "ticket_id": created, "alert_id": id})
	case "notify":
		// Send notification to recipients
		recipients := body.Recipients
		if len(recipients) == 0 {
			recipients = []string{alert.ScopeID}
		}
		for _, r := range recipients {
			if r == "" {
				continue
			}
			_, _ = createNotificationRecord(Notification{
				UserID:         r,
				Title:          alert.Title,
				Body:           alert.Message,
				Priority:       alert.Priority,
				Category:       "analytics",
				SourceApp:      "enterprise",
				SourceEntity:   "alert",
				SourceEntityID: alert.ID,
				DeepLink:       alert.DeepLink,
			})
		}
		c.JSON(200, gin.H{"status": "notified", "recipients": recipients})
	case "open_chat":
		c.JSON(200, gin.H{"status": "chat_opened", "url": getEnv("STATCHAT_UI_URL", "http://localhost:3009")})
	default:
		c.JSON(400, gin.H{"error": "unsupported action", "supported": []string{"create_ticket", "notify", "open_chat"}})
	}
}

// createHelpDeskTicket creates a ticket in HelpDesk via its API.
func createHelpDeskTicket(alert AnalyticalAlert, notes string) (string, error) {
	payload := map[string]interface{}{
		"title":       alert.Title,
		"description": fmt.Sprintf("%s\n\nAlert ID: %s\nMetric: %s (value: %.1f)\nNotes: %s", alert.Message, alert.ID, alert.Metric, alert.Value, notes),
		"priority":    mapAlertPriority(alert.Priority),
		"category":    "analytics",
		"source":      "enterprise-analytics",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	ticketID, err := postJSON(getEnv("HELPDESK_API_URL", "http://localhost:5006")+"/api/tickets", data)
	if err != nil {
		return "", err
	}
	return ticketID, nil
}

func mapAlertPriority(priority string) string {
	switch priority {
	case "critical":
		return "urgent"
	case "high":
		return "high"
	case "medium":
		return "normal"
	default:
		return "low"
	}
}

// postJSON posts a JSON payload and returns the extracted "id" from the response.
func postJSON(url string, body []byte) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytesReader(body))
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
	if v, ok := result["ticket_id"].(string); ok {
		return v, nil
	}
	return "", nil
}

func bytesReader(b []byte) io.Reader {
	return &byteReader{data: b}
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
