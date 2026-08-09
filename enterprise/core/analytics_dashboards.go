package main

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Enterprise Analytics Dashboards ────────────────────────────────
// Multi-level dashboards: Executive, Management, Project, Field.
// All values computed from live data — never hardcoded.

var (
	dashMu              sync.RWMutex
	analyticsDashboards = make(map[string]AnalyticsDashboard)
)

// bootstrapAnalyticsDashboards registers the dashboard definitions.
func bootstrapAnalyticsDashboards() {
	now := nowUTC()
	dashboards := []AnalyticsDashboard{
		{
			ID: "dash_executive", Type: "executive", Name: "Executive Dashboard",
			Description: "Organization performance overview for executives",
			Scopes:      []string{"organization"},
			Sections: []DashboardSection{
				{ID: "sec_kpis", Title: "Key Performance Indicators", Type: "kpis", Items: []string{"kpi_active_projects", "kpi_active_research", "kpi_submissions_received", "kpi_open_tickets", "kpi_approval_pending"}},
				{ID: "sec_quality", Title: "Data Quality", Type: "quality", Items: []string{"kpi_data_quality_score"}},
				{ID: "sec_alerts", Title: "Important Alerts", Type: "alerts"},
				{ID: "sec_timeline", Title: "Recent Activity", Type: "timeline"},
				{ID: "sec_projects", Title: "Project Portfolio", Type: "tables"},
				{ID: "sec_operations", Title: "Field Operations", Type: "tables"},
			},
			CreatedAt: now,
		},
		{
			ID: "dash_management", Type: "management", Name: "Management Dashboard",
			Description: "Projects, teams, tasks, approvals and performance for managers",
			Scopes:      []string{"organization"},
			Sections: []DashboardSection{
				{ID: "sec_kpis", Title: "Performance KPIs", Type: "kpis", Items: []string{"kpi_project_progress", "kpi_survey_completion_rate", "kpi_meeting_deadlines"}},
				{ID: "sec_tasks", Title: "Tasks & Teams", Type: "tables"},
				{ID: "sec_approvals", Title: "Pending Approvals", Type: "tables"},
				{ID: "sec_timeline", Title: "Team Activity", Type: "timeline"},
			},
			CreatedAt: now,
		},
		{
			ID: "dash_project", Type: "project", Name: "Project Dashboard",
			Description: "Project progress, tasks, surveys, team and risks",
			Scopes:      []string{"project"},
			Sections: []DashboardSection{
				{ID: "sec_kpis", Title: "Project KPIs", Type: "kpis", Items: []string{"kpi_project_progress", "kpi_survey_completion_rate", "kpi_data_quality_score"}},
				{ID: "sec_tasks", Title: "Project Tasks", Type: "tables"},
				{ID: "sec_quality", Title: "Data Quality", Type: "quality"},
				{ID: "sec_anomalies", Title: "Anomaly Detection", Type: "alerts"},
				{ID: "sec_timeline", Title: "Project Timeline", Type: "timeline"},
			},
			CreatedAt: now,
		},
		{
			ID: "dash_field", Type: "field", Name: "Field Operations Dashboard",
			Description: "Facilities, enumerators, survey progress and geographic coverage",
			Scopes:      []string{"organization"},
			Sections: []DashboardSection{
				{ID: "sec_kpis", Title: "Field KPIs", Type: "kpis", Items: []string{"kpi_field_staff_active", "kpi_submissions_received", "kpi_survey_completion_rate"}},
				{ID: "sec_map", Title: "Geographic Coverage", Type: "maps"},
				{ID: "sec_quality", Title: "Field Data Quality", Type: "quality"},
				{ID: "sec_anomalies", Title: "Field Anomalies", Type: "alerts"},
				{ID: "sec_facilities", Title: "Facilities", Type: "tables"},
			},
			CreatedAt: now,
		},
	}

	for _, d := range dashboards {
		analyticsDashboards[d.ID] = d
	}
}

func listAnalyticsDashboards() []AnalyticsDashboard {
	dashMu.RLock()
	defer dashMu.RUnlock()
	out := make([]AnalyticsDashboard, 0, len(analyticsDashboards))
	for _, d := range analyticsDashboards {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

func getAnalyticsDashboard(id string) (AnalyticsDashboard, bool) {
	dashMu.RLock()
	defer dashMu.RUnlock()
	d, ok := analyticsDashboards[id]
	return d, ok
}

// ─── Dashboard Data Assembly ───────────────────────────────────────
// fetchDashboardData assembles live data for each dashboard type.

func fetchDashboardData(dashType, scopeID, userID string) map[string]interface{} {
	data := map[string]interface{}{
		"type":      dashType,
		"timestamp": nowUTC(),
		"user_id":   userID,
		"scope_id":  scopeID,
	}

	switch dashType {
	case "executive":
		data["kpis"] = fetchExecutiveKPIs()
		data["quality"] = fetchQualitySummary(scopeID)
		data["alerts"] = fetchRecentAlerts(10)
		data["timeline"] = fetchRecentTimeline(10)
		data["projects"] = fetchProjectPortfolio()
		data["operations"] = fetchFieldOperations()
	case "management":
		data["kpis"] = fetchManagementKPIs(scopeID)
		data["tasks"] = fetchTasksForDashboard(scopeID)
		data["approvals"] = fetchPendingApprovals()
		data["timeline"] = fetchRecentTimeline(15)
	case "project":
		data["kpis"] = fetchProjectKPIs(scopeID)
		data["tasks"] = fetchTasksForProject(scopeID)
		data["quality"] = fetchQualitySummary(scopeID)
		data["anomalies"] = fetchRecentAnomalies(5)
		data["timeline"] = fetchTimelineForProject(scopeID)
	case "field":
		data["kpis"] = fetchFieldKPIs()
		data["geography"] = fetchGeoCoverage()
		data["quality"] = fetchQualitySummary(scopeID)
		data["anomalies"] = fetchRecentAnomalies(5)
		data["facilities"] = fetchFacilitiesSummary()
	}

	return data
}

// ─── KPI Aggregators ───────────────────────────────────────────────

func fetchExecutiveKPIs() []map[string]interface{} {
	kpis := make([]map[string]interface{}, 0)
	for _, kpiID := range []string{"kpi_active_projects", "kpi_active_research", "kpi_submissions_received", "kpi_open_tickets", "kpi_approval_pending", "kpi_meeting_deadlines"} {
		if def, ok := getKPIDefinition(kpiID); ok {
			val := computeKPI(def, "organization", "")
			kpis = append(kpis, kpiToMap(val))
		}
	}
	return kpis
}

func fetchManagementKPIs(scopeID string) []map[string]interface{} {
	kpis := make([]map[string]interface{}, 0)
	for _, kpiID := range []string{"kpi_project_progress", "kpi_survey_completion_rate", "kpi_meeting_deadlines", "kpi_approval_pending"} {
		if def, ok := getKPIDefinition(kpiID); ok {
			val := computeKPI(def, "organization", scopeID)
			kpis = append(kpis, kpiToMap(val))
		}
	}
	return kpis
}

func fetchProjectKPIs(projectID string) []map[string]interface{} {
	kpis := make([]map[string]interface{}, 0)
	for _, kpiID := range []string{"kpi_project_progress", "kpi_survey_completion_rate", "kpi_data_quality_score"} {
		if def, ok := getKPIDefinition(kpiID); ok {
			val := computeKPI(def, "project", projectID)
			kpis = append(kpis, kpiToMap(val))
		}
	}
	return kpis
}

func fetchFieldKPIs() []map[string]interface{} {
	kpis := make([]map[string]interface{}, 0)
	for _, kpiID := range []string{"kpi_field_staff_active", "kpi_submissions_received", "kpi_survey_completion_rate"} {
		if def, ok := getKPIDefinition(kpiID); ok {
			val := computeKPI(def, "organization", "")
			kpis = append(kpis, kpiToMap(val))
		}
	}
	return kpis
}

func kpiToMap(val KPIValue) map[string]interface{} {
	return map[string]interface{}{
		"kpi_id": val.KPIID, "kpi_name": val.KPI, "value": val.Value,
		"display_value": val.DisplayValue, "unit": val.Unit, "status": val.Status,
		"source_app": val.SourceApp, "period": val.Period, "computed_at": val.ComputedAt,
	}
}

// ─── Data Aggregators ──────────────────────────────────────────────

func fetchQualitySummary(scopeID string) map[string]interface{} {
	report := generateDataQualityReport("project", scopeID, "")
	return map[string]interface{}{
		"score":               report.Score,
		"completeness_score":  report.CompletenessScore,
		"duplicate_count":     report.DuplicateCount,
		"missing_value_count": report.MissingValueCount,
		"invalid_date_count":  report.InvalidDateCount,
		"invalid_coord_count": report.InvalidCoordCount,
		"outlier_count":       report.OutlierCount,
		"total_records":       report.TotalRecords,
		"generated_at":        report.GeneratedAt,
		"issues":              report.Issues,
	}
}

func fetchRecentAlerts(limit int) []map[string]interface{} {
	alertMu.RLock()
	alerts := make([]map[string]interface{}, 0, len(analyticalAlerts))
	for _, a := range analyticalAlerts {
		alerts = append(alerts, alertToMap(a))
	}
	alertMu.RUnlock()
	sort.Slice(alerts, func(i, j int) bool {
		ti, _ := alerts[i]["created_at"].(string)
		tj, _ := alerts[j]["created_at"].(string)
		return ti > tj
	})
	if len(alerts) > limit {
		alerts = alerts[:limit]
	}
	return alerts
}

func fetchRecentTimeline(limit int) []map[string]interface{} {
	entries := fetchTimeline(limit)
	out := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		out = append(out, map[string]interface{}{
			"id": e.ID, "user": e.User, "application": e.Application,
			"entity": e.Entity, "entity_id": e.EntityID, "action": e.Action,
			"description": e.Description, "timestamp": e.Timestamp, "project_id": e.ProjectID,
		})
	}
	return out
}

func fetchTimelineForProject(projectID string) []map[string]interface{} {
	entries := fetchTimeline(100)
	out := make([]map[string]interface{}, 0)
	for _, e := range entries {
		if e.ProjectID == projectID || e.EntityID == projectID {
			out = append(out, map[string]interface{}{
				"id": e.ID, "user": e.User, "application": e.Application,
				"entity": e.Entity, "entity_id": e.EntityID, "action": e.Action,
				"description": e.Description, "timestamp": e.Timestamp,
			})
		}
	}
	return out
}

func fetchProjectPortfolio() []map[string]interface{} {
	projects := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects"); data != nil {
		for _, p := range data {
			progress := 0.0
			if v, ok := p["progress"].(float64); ok {
				progress = v
			}
			projects = append(projects, map[string]interface{}{
				"id": p["id"], "name": p["name"], "stage": p["stage"],
				"owner": p["owner"], "progress": progress, "source": "pms",
			})
		}
	}
	return projects
}

func fetchFieldOperations() []map[string]interface{} {
	ops := []map[string]interface{}{}
	// Registry facilities
	if facilities := fetchJSONArray(getEnv("REGISTRY_API_URL", "http://localhost:9090") + "/api/facilities?limit=100"); facilities != nil {
		for _, f := range facilities {
			ops = append(ops, map[string]interface{}{
				"id": f["id"], "name": f["name"], "type": "facility",
				"region": f["region"], "source": "registry",
			})
		}
	}
	// HelpDesk open tickets
	if data := fetchJSON(getEnv("HELPDESK_API_URL", "http://localhost:5006") + "/api/dashboard"); data != nil {
		if v, ok := data["openTickets"].(float64); ok {
			ops = append(ops, map[string]interface{}{"type": "open_tickets", "value": v, "source": "helpdesk"})
		}
	}
	return ops
}

func fetchTasksForDashboard(scopeID string) []map[string]interface{} {
	return fetchTasksForProject(scopeID)
}

func fetchTasksForProject(projectID string) []map[string]interface{} {
	tasks := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/tasks?limit=50"); data != nil {
		for _, t := range data {
			if projectID == "" {
				tasks = append(tasks, taskToMap(t, "pms"))
			} else {
				if pid, ok := t["project_id"].(string); ok && pid == projectID {
					tasks = append(tasks, taskToMap(t, "pms"))
				}
			}
		}
	}
	return tasks
}

func taskToMap(t map[string]interface{}, source string) map[string]interface{} {
	return map[string]interface{}{
		"id": t["id"], "title": t["title"], "status": t["status"],
		"project_id": t["project_id"], "due": t["end_date"], "source": source,
	}
}

func fetchPendingApprovals() []map[string]interface{} {
	approvals := []map[string]interface{}{}
	if projects := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects"); projects != nil {
		for _, p := range projects {
			if stage, ok := p["stage"].(string); ok && stage == "Approval" {
				approvals = append(approvals, map[string]interface{}{
					"id": p["id"], "title": p["name"], "type": "project",
					"status": "pending", "source": "pms",
				})
			}
		}
	}
	if research := fetchJSONArray(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/research"); research != nil {
		for _, r := range research {
			if stage, ok := r["stage"].(string); ok {
				if stage == "Proposal" || stage == "Ethics Submission" || stage == "Funding Approval" {
					approvals = append(approvals, map[string]interface{}{
						"id": r["id"], "title": r["name"], "type": "research",
						"status": "pending", "source": "rms",
					})
				}
			}
		}
	}
	return approvals
}

func fetchRecentAnomalies(limit int) []map[string]interface{} {
	anomalyMu.RLock()
	anomalies := make([]map[string]interface{}, 0, len(anomalyRecords))
	for _, a := range anomalyRecords {
		anomalies = append(anomalies, map[string]interface{}{
			"id": a.ID, "rule_id": a.RuleID, "rule_name": a.RuleName,
			"type": a.Type, "severity": a.Severity, "description": a.Description,
			"current_value": a.CurrentValue, "expected_value": a.ExpectedValue,
			"detected_at": a.DetectedAt, "scope": a.Scope, "scope_id": a.ScopeID,
		})
	}
	anomalyMu.RUnlock()
	sort.Slice(anomalies, func(i, j int) bool {
		ti, _ := anomalies[i]["detected_at"].(string)
		tj, _ := anomalies[j]["detected_at"].(string)
		return ti > tj
	})
	if len(anomalies) > limit {
		anomalies = anomalies[:limit]
	}
	return anomalies
}

func fetchGeoCoverage() map[string]interface{} {
	// Real GPS points from enterprise records
	points := []map[string]interface{}{}
	dataLayerMu.RLock()
	for _, rec := range enterpriseRecords {
		if rec.GeoLat != 0 || rec.GeoLng != 0 {
			points = append(points, map[string]interface{}{
				"lat": rec.GeoLat, "lng": rec.GeoLng,
				"source": rec.SourceApp, "entity": rec.SourceEntity,
				"source_id": rec.SourceID, "status": rec.Status,
			})
		}
	}
	dataLayerMu.RUnlock()

	// Region breakdown from enterprise records
	regions := map[string]int{}
	dataLayerMu.RLock()
	for _, rec := range enterpriseRecords {
		if rec.Region != "" {
			regions[rec.Region]++
		}
	}
	dataLayerMu.RUnlock()

	regionList := make([]map[string]interface{}, 0, len(regions))
	for region, count := range regions {
		regionList = append(regionList, map[string]interface{}{"region": region, "count": count})
	}
	sort.Slice(regionList, func(i, j int) bool {
		ci, _ := regionList[i]["count"].(int)
		cj, _ := regionList[j]["count"].(int)
		return ci > cj
	})

	return map[string]interface{}{
		"points":  points,
		"regions": regionList,
		"total":   len(points),
		"source":  "enterprise-layer",
	}
}

func fetchFacilitiesSummary() []map[string]interface{} {
	facilities := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("REGISTRY_API_URL", "http://localhost:9090") + "/api/facilities?limit=100"); data != nil {
		for _, f := range data {
			facilities = append(facilities, map[string]interface{}{
				"id": f["id"], "name": f["name"], "region": f["region"],
				"district": f["district"], "status": f["status"], "source": "registry",
			})
		}
	}
	return facilities
}

// fetchTrendData provides time-series data for charts.
func fetchTrendData(metric, sourceApp string, days int) []map[string]interface{} {
	trend := []map[string]interface{}{}
	if days <= 0 {
		days = 30
	}

	// Build time buckets
	buckets := make(map[string]int)
	dataLayerMu.RLock()
	for _, rec := range enterpriseRecords {
		if sourceApp != "" && rec.SourceApp != sourceApp {
			continue
		}
		if metric != "" && rec.SourceEntity != metric {
			continue
		}
		if t, err := time.Parse(time.RFC3339, rec.Timestamp); err == nil {
			if time.Since(t) <= time.Duration(days)*24*time.Hour {
				key := t.Format("2006-01-02")
				buckets[key]++
			}
		}
	}
	dataLayerMu.RUnlock()

	// Fill in all days
	for i := days - 1; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i)
		key := d.Format("2006-01-02")
		trend = append(trend, map[string]interface{}{
			"period": key, "value": buckets[key], "source": sourceApp,
		})
	}
	return trend
}

// computeTrendDelta computes % change vs previous period (real data).
func computeTrendDelta(metric, sourceApp string, days int) float64 {
	current := 0
	previous := 0
	dataLayerMu.RLock()
	for _, rec := range enterpriseRecords {
		if sourceApp != "" && rec.SourceApp != sourceApp {
			continue
		}
		if metric != "" && rec.SourceEntity != metric {
			continue
		}
		if t, err := time.Parse(time.RFC3339, rec.Timestamp); err == nil {
			if time.Since(t) <= time.Duration(days)*24*time.Hour {
				current++
			} else if time.Since(t) <= time.Duration(days*2)*24*time.Hour {
				previous++
			}
		}
	}
	dataLayerMu.RUnlock()

	if previous == 0 {
		return 0
	}
	return math.Round((float64(current-previous)/float64(previous))*1000) / 10
}

// ─── API Handlers ──────────────────────────────────────────────────

func handleListAnalyticsDashboards(c *gin.Context) {
	dashboards := listAnalyticsDashboards()
	c.JSON(200, gin.H{"count": len(dashboards), "dashboards": dashboards})
}

func handleGetAnalyticsDashboard(c *gin.Context) {
	id := c.Param("id")
	dash, ok := getAnalyticsDashboard(id)
	if !ok {
		c.JSON(404, gin.H{"error": "dashboard not found"})
		return
	}
	scopeID := c.Query("scope_id")
	userID := c.Query("user_id")
	data := fetchDashboardData(dash.Type, scopeID, userID)
	data["dashboard"] = dash
	c.JSON(200, data)
}

// handleDashboardData returns raw dashboard data for a type without definition.
func handleDashboardData(c *gin.Context) {
	dashType := c.Query("type")
	if dashType == "" {
		dashType = "executive"
	}
	valid := map[string]bool{"executive": true, "management": true, "project": true, "field": true}
	if !valid[dashType] {
		c.JSON(400, gin.H{"error": "invalid dashboard type", "valid": []string{"executive", "management", "project", "field"}})
		return
	}
	scopeID := c.Query("scope_id")
	userID := c.Query("user_id")
	data := fetchDashboardData(dashType, scopeID, userID)
	c.JSON(200, data)
}

// handleCrossAppIntelligence provides cross-application signal aggregation.
// Combines PMS, StatCollect, HelpDesk and StatChat signals for the "bigger picture".
func handleCrossAppIntelligence(c *gin.Context) {
	projectID := c.Query("project_id")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		tenantID = getEnvValue("STATGATE_TENANT_ID")
		if tenantID == "" {
			tenantID = "statgate"
		}
	}

	signals := map[string]interface{}{}

	// PMS: project status
	if projectID != "" {
		if data := fetchJSON(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects/" + projectID); data != nil {
			progress, _ := data["progress"].(float64)
			stage, _ := data["stage"].(string)
			signals["pms"] = map[string]interface{}{
				"project_behind_schedule": progress < 50,
				"progress":                progress,
				"stage":                   stage,
				"source":                  "pms",
			}
		}
	}

	// StatCollect: submission trend
	submissionTrend := fetchTrendData("submission", "statcollect", 14)
	subCount := 0
	for _, s := range submissionTrend {
		if v, ok := s["value"].(int); ok {
			subCount += v
		}
	}
	delta := computeTrendDelta("submission", "statcollect", 7)
	signals["statcollect"] = map[string]interface{}{
		"reporting_dropped": delta < -20,
		"submissions":       subCount,
		"trend_delta_pct":   delta,
		"source":            "statcollect",
	}

	// HelpDesk: field-related tickets
	ticketCount := 0
	if tickets := fetchJSONArray(getEnv("HELPDESK_API_URL", "http://localhost:5006") + "/api/tickets?limit=50"); tickets != nil {
		for _, t := range tickets {
			category, _ := t["category"].(string)
			if category == "field" || category == "operations" || category == "" {
				ticketCount++
			}
		}
	}
	signals["helpdesk"] = map[string]interface{}{
		"open_field_tickets": ticketCount,
		"source":             "helpdesk",
	}

	// StatChat: field discussions
	chatSignals := []map[string]interface{}{}
	if messages := fetchJSONArray(getEnv("STATCHAT_API_URL", "http://localhost:4000") + "/v1/messages?limit=20"); messages != nil {
		keywords := []string{"connectivity", "offline", "network", "field", "breakdown"}
		for _, m := range messages {
			text, _ := m["message"].(string)
			for _, kw := range keywords {
				if contains(text, kw) {
					chatSignals = append(chatSignals, map[string]interface{}{
						"sender": m["sender"], "message": text,
						"keyword": kw, "timestamp": m["created_at"],
					})
					break
				}
			}
		}
	}
	signals["statchat"] = map[string]interface{}{
		"field_discussions": chatSignals,
		"source":            "statchat",
	}

	// Data quality
	dq := quickDataQualitySnapshot(projectID)
	signals["quality"] = dq

	// Analytical synthesis
	riskLevel := "low"
	riskPoints := 0
	if v, ok := signals["pms"].(map[string]interface{}); ok {
		if behind, _ := v["project_behind_schedule"].(bool); behind {
			riskPoints += 2
		}
	}
	if v, ok := signals["statcollect"].(map[string]interface{}); ok {
		if dropped, _ := v["reporting_dropped"].(bool); dropped {
			riskPoints += 2
		}
	}
	if v, ok := signals["helpdesk"].(map[string]interface{}); ok {
		if tickets, _ := v["open_field_tickets"].(int); tickets > 3 {
			riskPoints += 1
		}
	}
	if dq["score"].(float64) < 80 {
		riskPoints += 1
	}
	if riskPoints >= 4 {
		riskLevel = "critical"
	} else if riskPoints >= 2 {
		riskLevel = "elevated"
	}

	signals["synthesis"] = map[string]interface{}{
		"risk_level":     riskLevel,
		"risk_points":    riskPoints,
		"active_signals": riskPoints,
		"timestamp":      nowUTC(),
	}

	c.JSON(200, signals)
}

func quickDataQualitySnapshot(scopeID string) map[string]interface{} {
	report := generateDataQualityReport("project", scopeID, "")
	return map[string]interface{}{
		"score": report.Score, "issues": report.Issues,
		"total_records": report.TotalRecords, "generated_at": report.GeneratedAt,
	}
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// needFmt keeps fmt imported (used for formatting)
func needFmt(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
