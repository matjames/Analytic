package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Phase VIII: Enterprise Command Centre ────────────────────────────────
// The Command Centre aggregates all Enterprise Core capabilities into a
// single, role-aware response that answers:
//
//   What is happening?    What requires my attention?  What is changing?
//   What is at risk?      What evidence supports this? What decisions are required?
//   What should I do?     What happened after I acted?
//
// This file does NOT duplicate any engine. It aggregates from existing
// functions defined in workspace.go, action_centre.go, analytics_alerts.go,
// analytics_anomaly.go, analytics_kpi.go, decisions.go, workflow_engine.go.

// ─── Models ───────────────────────────────────────────────────────────────

type SituationSummary struct {
	ActiveProjects   int `json:"active_projects"`
	ActiveResearch   int `json:"active_research"`
	ActiveSurveys    int `json:"active_surveys"`
	OpenTickets      int `json:"open_tickets"`
	OpenRisks        int `json:"open_risks"`
	ActiveWorkflows  int `json:"active_workflows"`
	OpenDecisions    int `json:"open_decisions"`
	PendingApprovals int `json:"pending_approvals"`
}

type AttentionItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Source      string `json:"source"`
	DeepLink    string `json:"deep_link,omitempty"`
	DueAt       string `json:"due_at,omitempty"`
	Description string `json:"description,omitempty"`
}

type IntelligenceSummary struct {
	ActiveAlerts      int `json:"active_alerts"`
	AnomaliesLast24h  int `json:"anomalies_last_24h"`
	DataQualityIssues int `json:"data_quality_issues"`
	KPICount          int `json:"kpi_count"`
}

// ─── API Handlers ─────────────────────────────────────────────────────────

// handleCommandCentreSummary returns the master command centre payload.
func handleCommandCentreSummary(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	userRole := c.Query("role")
	if userRole == "" {
		userRole = c.GetHeader("X-User-Role")
	}
	if userRole == "" {
		userRole = "analyst"
	}
	if userID == "" {
		c.JSON(400, gin.H{"error": "user_id required"})
		return
	}

	type fetch struct {
		key string
		val interface{}
	}
	ch := make(chan fetch, 16)
	var wg sync.WaitGroup

	funcs := []struct {
		key string
		fn  func() interface{}
	}{
		{"situation", func() interface{} { return buildSituation(userID) }},
		{"intelligence", func() interface{} { return buildIntelligence() }},
		{"service_health", func() interface{} { return checkAllServices() }},
		{"notifications", func() interface{} { return fetchMyNotifications(userID) }},
		{"activity", func() interface{} { return fetchMyActivity(userID) }},
		{"kpis", func() interface{} { return listKPIsForRole(userRole, 12) }},
		{"alerts", func() interface{} { return listActiveAlerts(10) }},
		{"decisions", func() interface{} { return buildDecisionSummary(userID) }},
		{"tasks", func() interface{} { return fetchMyTasks(userID) }},
		{"approvals", func() interface{} { return fetchMyApprovals(userID) }},
		{"projects", func() interface{} { return fetchMyProjects(userID) }},
		{"research", func() interface{} { return fetchMyResearch(userID) }},
		{"surveys", func() interface{} { return fetchMySurveys(userID) }},
		{"workflows", func() interface{} { return buildWorkflowSummary() }},
		{"risks", func() interface{} { return fetchGovernanceRisks(10) }},
	}

	for _, f := range funcs {
		wg.Add(1)
		go func(key string, fn func() interface{}) {
			defer wg.Done()
			ch <- fetch{key, fn()}
		}(f.key, f.fn)
	}
	go func() { wg.Wait(); close(ch) }()

	data := make(map[string]interface{}, 16)
	for f := range ch {
		data[f.key] = f.val
	}

	// Build attention items
	attention := buildAttentionItems(userID)
	data["attention"] = gin.H{
		"total_items":    len(attention),
		"critical_count": countByPriority(attention, "critical"),
		"high_count":     countByPriority(attention, "high"),
		"items":          attention,
	}
	data["user_id"] = userID
	data["user_role"] = userRole
	data["timestamp"] = nowUTC()

	recordAudit("command_centre.summary", "command_centre", userID, map[string]interface{}{"role": userRole})
	c.JSON(200, data)
}

// handleExecutiveView returns the executive command centre view.
func handleExecutiveView(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	if userID == "" {
		c.JSON(400, gin.H{"error": "user_id required"})
		return
	}

	type fetch struct{ key string; val interface{} }
	ch := make(chan fetch, 12)
	var wg sync.WaitGroup

	funcs := []struct {
		key string
		fn  func() interface{}
	}{
		{"kpis", func() interface{} { return listKPIsForRole("executive", 20) }},
		{"alerts", func() interface{} { return listActiveAlerts(20) }},
		{"anomalies", func() interface{} { return listRecentAnomalies(20) }},
		{"risks", func() interface{} { return fetchGovernanceRisks(20) }},
		{"decisions", func() interface{} { return buildDecisionSummary(userID) }},
		{"approvals", func() interface{} { return fetchMyApprovals(userID) }},
		{"projects", func() interface{} { return fetchAllProjects(20) }},
		{"research", func() interface{} { return fetchAllResearch(20) }},
		{"service_health", func() interface{} { return checkAllServices() }},
		{"governance", func() interface{} { return fetchGovernanceSummary() }},
		{"activity", func() interface{} { return fetchMyActivity(userID) }},
		{"workflows", func() interface{} { return buildWorkflowSummary() }},
	}

	for _, f := range funcs {
		wg.Add(1)
		go func(key string, fn func() interface{}) {
			defer wg.Done()
			ch <- fetch{key, fn()}
		}(f.key, f.fn)
	}
	go func() { wg.Wait(); close(ch) }()

	data := make(map[string]interface{}, 14)
	for f := range ch {
		data[f.key] = f.val
	}
	data["user_id"] = userID
	data["timestamp"] = nowUTC()
	data["situation"] = buildSituation(userID)

	recordAudit("command_centre.executive", "command_centre", userID, nil)
	c.JSON(200, data)
}

// handleOperationsView returns the operational command centre view.
func handleOperationsView(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	if userID == "" {
		c.JSON(400, gin.H{"error": "user_id required"})
		return
	}

	type fetch struct{ key string; val interface{} }
	ch := make(chan fetch, 10)
	var wg sync.WaitGroup

	funcs := []struct {
		key string
		fn  func() interface{}
	}{
		{"tasks", func() interface{} { return fetchMyTasks(userID) }},
		{"approvals", func() interface{} { return fetchMyApprovals(userID) }},
		{"tickets", func() interface{} { return fetchMyTickets(userID) }},
		{"alerts", func() interface{} { return listActiveAlerts(20) }},
		{"workflows", func() interface{} { return buildWorkflowSummary() }},
		{"overdue_tasks", func() interface{} { return fetchOverdueTasks(userID) }},
		{"data_quality", func() interface{} { return userDataQualityIssues(userID) }},
		{"notifications", func() interface{} { return fetchMyNotifications(userID) }},
		{"activity", func() interface{} { return fetchMyActivity(userID) }},
		{"surveys", func() interface{} { return fetchMySurveys(userID) }},
	}

	for _, f := range funcs {
		wg.Add(1)
		go func(key string, fn func() interface{}) {
			defer wg.Done()
			ch <- fetch{key, fn()}
		}(f.key, f.fn)
	}
	go func() { wg.Wait(); close(ch) }()

	data := make(map[string]interface{}, 12)
	for f := range ch {
		data[f.key] = f.val
	}
	data["user_id"] = userID
	data["timestamp"] = nowUTC()
	c.JSON(200, data)
}

// handleFieldView returns the field command centre view.
func handleFieldView(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	if userID == "" {
		c.JSON(400, gin.H{"error": "user_id required"})
		return
	}

	type fetch struct{ key string; val interface{} }
	ch := make(chan fetch, 5)
	var wg sync.WaitGroup

	funcs := []struct {
		key string
		fn  func() interface{}
	}{
		{"facilities", func() interface{} { return fetchFacilities(50) }},
		{"surveys", func() interface{} { return fetchAllSurveys(20) }},
		{"data_quality", func() interface{} { return userDataQualityIssues(userID) }},
		{"alerts", func() interface{} { return listActiveAlerts(20) }},
		{"activity", func() interface{} { return fetchMyActivity(userID) }},
	}

	for _, f := range funcs {
		wg.Add(1)
		go func(key string, fn func() interface{}) {
			defer wg.Done()
			ch <- fetch{key, fn()}
		}(f.key, f.fn)
	}
	go func() { wg.Wait(); close(ch) }()

	data := make(map[string]interface{}, 7)
	for f := range ch {
		data[f.key] = f.val
	}
	data["user_id"] = userID
	data["timestamp"] = nowUTC()
	c.JSON(200, data)
}

// handleProjectPortfolio returns the project portfolio view.
func handleProjectPortfolio(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	projects := fetchAllProjects(50)
	c.JSON(200, gin.H{
		"user_id":   userID,
		"timestamp": nowUTC(),
		"projects":  projects,
		"count":     len(projects),
	})
}

// handleResearchPortfolio returns the research portfolio view.
func handleResearchPortfolio(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	research := fetchAllResearch(50)
	c.JSON(200, gin.H{
		"user_id":   userID,
		"timestamp": nowUTC(),
		"research":  research,
		"count":     len(research),
	})
}

// handleGovernanceView returns the governance command centre view.
func handleGovernanceView(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}

	type fetch struct{ key string; val interface{} }
	ch := make(chan fetch, 4)
	var wg sync.WaitGroup

	funcs := []struct {
		key string
		fn  func() interface{}
	}{
		{"risks", func() interface{} { return fetchGovernanceRisks(20) }},
		{"controls", func() interface{} { return fetchGovernanceControls(20) }},
		{"findings", func() interface{} { return fetchGovernanceFindings(20) }},
		{"summary", func() interface{} { return fetchGovernanceSummary() }},
	}

	for _, f := range funcs {
		wg.Add(1)
		go func(key string, fn func() interface{}) {
			defer wg.Done()
			ch <- fetch{key, fn()}
		}(f.key, f.fn)
	}
	go func() { wg.Wait(); close(ch) }()

	data := make(map[string]interface{}, 6)
	for f := range ch {
		data[f.key] = f.val
	}
	data["user_id"] = userID
	data["timestamp"] = nowUTC()
	c.JSON(200, data)
}

// handleDecisionCentreView returns the decision centre view.
func handleDecisionCentreView(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	if userID == "" {
		c.JSON(400, gin.H{"error": "user_id required"})
		return
	}

	awaiting := listDecisionRecords("open", userID, 20)
	created := listDecisionRecordsBy(userID, 20)
	recent := listDecisionRecords("", "", 10)
	underReview := listDecisionRecords("under_review", "", 10)
	recs := listAIRecommendations("")
	pendingApprovals := listApprovalRequests(userID, "pending")
	allMyApprovals := listApprovalRequests(userID, "")

	c.JSON(200, gin.H{
		"user_id":               userID,
		"timestamp":             nowUTC(),
		"awaiting_me":           awaiting,
		"created_by_me":         created,
		"recent":                recent,
		"under_review":          underReview,
		"ai_recommendations":    recs,
		"approvals_awaiting_me": pendingApprovals,
		"my_approvals":          allMyApprovals,
	})
}

// handleOutcomeMonitor returns the outcome monitoring view.
func handleOutcomeMonitor(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	actioned := listDecisionRecords("actioned", "", 20)
	monitoring := listDecisionRecords("monitoring", "", 20)
	closed := listDecisionRecords("closed", "", 20)

	c.JSON(200, gin.H{
		"user_id":    userID,
		"timestamp":  nowUTC(),
		"actioned":   actioned,
		"monitoring": monitoring,
		"closed":     closed,
	})
}

// handleDataIntelligenceView returns the data intelligence view.
func handleDataIntelligenceView(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}

	type fetch struct{ key string; val interface{} }
	ch := make(chan fetch, 3)
	var wg sync.WaitGroup

	funcs := []struct {
		key string
		fn  func() interface{}
	}{
		{"kpis", func() interface{} { return listKPIsForRole("analyst", 20) }},
		{"alerts", func() interface{} { return listActiveAlerts(20) }},
		{"anomalies", func() interface{} { return listRecentAnomalies(20) }},
	}

	for _, f := range funcs {
		wg.Add(1)
		go func(key string, fn func() interface{}) {
			defer wg.Done()
			ch <- fetch{key, fn()}
		}(f.key, f.fn)
	}
	go func() { wg.Wait(); close(ch) }()

	data := make(map[string]interface{}, 5)
	for f := range ch {
		data[f.key] = f.val
	}
	data["user_id"] = userID
	data["timestamp"] = nowUTC()
	c.JSON(200, data)
}

// handleCommandCentreServiceHealth returns service health panel data.
func handleCommandCentreServiceHealth(c *gin.Context) {
	services := checkAllServices()
	healthy, degraded, unreachable := 0, 0, 0
	for _, s := range services {
		switch s.Status {
		case "healthy":
			healthy++
		case "degraded":
			degraded++
		default:
			unreachable++
		}
	}
	overall := "healthy"
	if unreachable > 0 || degraded > 0 {
		overall = "degraded"
	}
	c.JSON(200, gin.H{
		"services":  services,
		"summary":   gin.H{"healthy": healthy, "degraded": degraded, "unreachable": unreachable},
		"timestamp": nowUTC(),
		"overall":   overall,
	})
}

// ─── Internal aggregation helpers ─────────────────────────────────────────

func buildSituation(userID string) SituationSummary {
	projects := fetchMyProjects(userID)
	research := fetchMyResearch(userID)
	surveys := fetchMySurveys(userID)
	tickets := fetchMyTickets(userID)
	risks := fetchGovernanceRisks(100)
	instances := listWorkflowInstances("running")
	decisions := listDecisionRecords("open", "", 100)
	approvals := fetchMyApprovals(userID)

	return SituationSummary{
		ActiveProjects:   len(projects),
		ActiveResearch:   len(research),
		ActiveSurveys:    len(surveys),
		OpenTickets:      len(tickets),
		OpenRisks:        len(risks),
		ActiveWorkflows:  len(instances),
		OpenDecisions:    len(decisions),
		PendingApprovals: len(approvals),
	}
}

func buildIntelligence() IntelligenceSummary {
	alerts := listActiveAlerts(1000)
	anomalies := listRecentAnomalies(1000)
	quality := listDataQualityIssuesAll()
	kpis := listKPIDefinitions()
	return IntelligenceSummary{
		ActiveAlerts:      len(alerts),
		AnomaliesLast24h:  len(anomalies),
		DataQualityIssues: len(quality),
		KPICount:          len(kpis),
	}
}

func buildAttentionItems(userID string) []AttentionItem {
	items := []AttentionItem{}

	for _, t := range fetchMyTasks(userID) {
		id := fmt.Sprintf("%v", t["id"])
		source := fmt.Sprintf("%v", t["source"])
		items = append(items, AttentionItem{
			ID: id, Title: fmt.Sprintf("%v", t["title"]),
			Type: "task", Priority: "medium", Source: source,
			DeepLink: ccDeepLink("task", source, id),
		})
	}

	for _, a := range fetchMyApprovals(userID) {
		id := fmt.Sprintf("%v", a["id"])
		source := fmt.Sprintf("%v", a["source"])
		items = append(items, AttentionItem{
			ID: id, Title: fmt.Sprintf("%v", a["title"]),
			Type: "approval", Priority: "high", Source: source,
			DeepLink: ccDeepLink("approval", source, id),
		})
	}

	for _, al := range listActiveAlerts(10) {
		id := fmt.Sprintf("%v", al["id"])
		items = append(items, AttentionItem{
			ID: id, Title: fmt.Sprintf("%v", al["name"]),
			Type: "alert", Priority: fmt.Sprintf("%v", al["severity"]),
			Source:   "enterprise",
			DeepLink: ccDeepLink("alert", "enterprise", id),
		})
	}

	for _, d := range listDecisionRecords("open", userID, 5) {
		items = append(items, AttentionItem{
			ID: d.ID, Title: d.Decision, Type: "decision", Priority: "high",
			Source: "enterprise", Description: d.ActionRequired,
			DeepLink: ccDeepLink("decision", "enterprise", d.ID),
		})
	}

	for _, f := range fetchGovernanceFindings(5) {
		id := fmt.Sprintf("%v", f["id"])
		title := fmt.Sprintf("%v", f["title"])
		severity := fmt.Sprintf("%v", f["severity"])
		items = append(items, AttentionItem{
			ID: id, Title: "Governance: " + title,
			Type: "governance", Priority: severityToPriority(severity),
			Source: "statgovernance",
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return ccPriorityWeight(items[i].Priority) > ccPriorityWeight(items[j].Priority)
	})
	return items
}

func buildDecisionSummary(userID string) map[string]interface{} {
	open := listDecisionRecords("open", userID, 10)
	recent := listDecisionRecords("", "", 10)
	return map[string]interface{}{
		"open":   open,
		"recent": recent,
		"count":  len(open),
	}
}

func buildWorkflowSummary() map[string]interface{} {
	running := listWorkflowInstances("running")
	failed := listWorkflowInstances("failed")
	return map[string]interface{}{
		"running_count": len(running),
		"failed_count":  len(failed),
		"running":       running,
		"failed":        failed,
	}
}

func listKPIsForRole(role string, limit int) []KPIDefinition {
	all := listKPIDefinitions()
	if len(all) <= limit {
		return all
	}
	return all[:limit]
}

func listActiveAlerts(limit int) []map[string]interface{} {
	alertMu.RLock()
	defer alertMu.RUnlock()
	out := make([]map[string]interface{}, 0)
	for _, al := range analyticalAlerts {
		if al.Status == "active" || al.Status == "acknowledged" {
			out = append(out, alertToMap(al))
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out
}

func listRecentAnomalies(limit int) []map[string]interface{} {
	anomalyMu.RLock()
	defer anomalyMu.RUnlock()
	cutoff := time.Now().Add(-24 * time.Hour)
	out := make([]map[string]interface{}, 0)
	for _, an := range anomalyRecords {
		if t, err := time.Parse(time.RFC3339, an.DetectedAt); err == nil && t.After(cutoff) {
			out = append(out, map[string]interface{}{
				"id": an.ID, "rule_id": an.RuleID, "type": an.Type,
				"severity": an.Severity, "detected_at": an.DetectedAt,
				"description": an.Description, "status": an.Status,
			})
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out
}

func listDataQualityIssuesAll() []map[string]interface{} {
	if data := fetchJSONArray(getEnv("ANALYTICS_API_URL", "http://localhost:5000") + "/api/quality/issues?limit=100"); data != nil {
		return data
	}
	return []map[string]interface{}{}
}

func fetchGovernanceRisks(limit int) []map[string]interface{} {
	url := fmt.Sprintf("%s/api/risks?limit=%d", getEnv("STATGOVERNANCE_API_URL", "http://localhost:8093"), limit)
	if data := fetchJSONArray(url); data != nil {
		return data
	}
	return []map[string]interface{}{}
}

func fetchGovernanceControls(limit int) []map[string]interface{} {
	url := fmt.Sprintf("%s/api/controls?limit=%d", getEnv("STATGOVERNANCE_API_URL", "http://localhost:8093"), limit)
	if data := fetchJSONArray(url); data != nil {
		return data
	}
	return []map[string]interface{}{}
}

func fetchGovernanceFindings(limit int) []map[string]interface{} {
	url := fmt.Sprintf("%s/api/findings?limit=%d", getEnv("STATGOVERNANCE_API_URL", "http://localhost:8093"), limit)
	if data := fetchJSONArray(url); data != nil {
		return data
	}
	return []map[string]interface{}{}
}

func fetchGovernanceSummary() map[string]interface{} {
	if obj := fetchJSON(getEnv("STATGOVERNANCE_API_URL", "http://localhost:8093") + "/api/dashboard"); obj != nil {
		return obj
	}
	return map[string]interface{}{"status": "unavailable"}
}

func fetchAllProjects(limit int) []map[string]interface{} {
	url := fmt.Sprintf("%s/api/projects?limit=%d", getEnv("PMS_API_URL", "http://localhost:8091"), limit)
	if data := fetchJSONArray(url); data != nil {
		return data
	}
	return []map[string]interface{}{}
}

func fetchAllResearch(limit int) []map[string]interface{} {
	url := fmt.Sprintf("%s/api/research?limit=%d", getEnv("RMS_API_URL", "http://localhost:8092"), limit)
	if data := fetchJSONArray(url); data != nil {
		return data
	}
	return []map[string]interface{}{}
}

func fetchAllSurveys(limit int) []map[string]interface{} {
	url := fmt.Sprintf("%s/api/surveys?limit=%d", getEnv("PMS_API_URL", "http://localhost:8091"), limit)
	if data := fetchJSONArray(url); data != nil {
		return data
	}
	return []map[string]interface{}{}
}

func fetchFacilities(limit int) []map[string]interface{} {
	url := fmt.Sprintf("%s/api/facilities?limit=%d", getEnv("REGISTRY_API_URL", "http://localhost:9090"), limit)
	if data := fetchJSONArray(url); data != nil {
		return data
	}
	return []map[string]interface{}{}
}

func fetchOverdueTasks(userID string) []map[string]interface{} {
	tasks := listEnterpriseTasks(userID, "", "", 50)
	out := []map[string]interface{}{}
	now := time.Now()
	for _, t := range tasks {
		if t.DueAt == "" || t.Status == "completed" {
			continue
		}
		if due, err := time.Parse(time.RFC3339, t.DueAt); err == nil && now.After(due) {
			source := t.SourceApp
			if source == "" {
				source = "enterprise"
			}
			out = append(out, map[string]interface{}{
				"id": t.ID, "title": t.Title, "status": t.Status,
				"due_at": t.DueAt, "source": source, "priority": t.Priority,
			})
		}
	}
	return out
}

func listDecisionRecordsBy(userID string, limit int) []DecisionRecord {
	all := listDecisionRecords("", "", 1000)
	out := []DecisionRecord{}
	for _, d := range all {
		if d.DecisionMaker == userID {
			out = append(out, d)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func countByPriority(items []AttentionItem, priority string) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(item.Priority, priority) {
			count++
		}
	}
	return count
}

func ccDeepLink(entityType, source, id string) string {
	switch source {
	case "pms":
		return fmt.Sprintf("http://localhost:3010/%s/%s", entityType, id)
	case "rms":
		return fmt.Sprintf("http://localhost:3011/%s/%s", entityType, id)
	case "helpdesk":
		return fmt.Sprintf("http://localhost:3005/%s/%s", entityType, id)
	case "statgovernance":
		return fmt.Sprintf("http://localhost:3012/%s/%s", entityType, id)
	default:
		return ""
	}
}

func ccPriorityWeight(p string) int {
	switch strings.ToLower(p) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	}
	return 0
}

func severityToPriority(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	}
	return "low"
}
