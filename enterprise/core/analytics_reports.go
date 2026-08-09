package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Enterprise Report Builder ─────────────────────────────────────
// Authorized users create reports from enterprise data with:
// data source selection, filters, date ranges, grouping, aggregations,
// charts, tables, narrative sections, branding, export and scheduling.
// Reports are generated from live data.

var (
	reportDefMu   sync.RWMutex
	reportDefs    = make(map[string]ReportDefinition)
	reportRunMu   sync.RWMutex
	reportRuns    = make(map[string]ReportRun)
	scheduledRuns = make(map[string]time.Time)
)

// ─── Report Definition Registry ────────────────────────────────────

func registerReportDefinition(def ReportDefinition) {
	reportDefMu.Lock()
	if def.ID == "" {
		def.ID = fmt.Sprintf("rptd_%d", time.Now().UnixNano())
	}
	if def.CreatedAt == "" {
		def.CreatedAt = nowUTC()
	}
	def.UpdatedAt = nowUTC()
	reportDefs[def.ID] = def
	reportDefMu.Unlock()
}

func getReportDefinition(id string) (ReportDefinition, bool) {
	reportDefMu.RLock()
	defer reportDefMu.RUnlock()
	def, ok := reportDefs[id]
	return def, ok
}

func listReportDefinitions() []ReportDefinition {
	reportDefMu.RLock()
	defer reportDefMu.RUnlock()
	out := make([]ReportDefinition, 0, len(reportDefs))
	for _, def := range reportDefs {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ─── Report Execution ──────────────────────────────────────────────

func executeReport(def ReportDefinition, runID, genBy, schedule string) {
	run := ReportRun{
		ID:         runID,
		ReportID:   def.ID,
		ReportName: def.Name,
		Status:     "running",
		GenBy:      genBy,
		Schedule:   schedule,
		CreatedAt:  nowUTC(),
	}

	reportRunMu.Lock()
	reportRuns[run.ID] = run
	reportRunMu.Unlock()

	// Build dataset from live data
	dataSummary, records := buildReportDataset(def)

	reportRunMu.Lock()
	run.Status = "completed"
	run.CompletedAt = nowUTC()
	run.RecordCount = int64(len(records))
	run.DataSummary = dataSummary
	run.URL = fmt.Sprintf("/api/analytics/reports/runs/%s/data", run.ID)
	reportRuns[run.ID] = run
	reportRunMu.Unlock()

	// Publish event for report generation
	publishEvent(DomainEvent{
		EventType:  "report.generated",
		Source:     "enterprise",
		ObjectType: "report",
		ObjectID:   run.ID,
		Actor:      genBy,
		Payload: map[string]interface{}{
			"title": def.Name, "report_id": def.ID, "format": def.ExportFormat,
			"record_count": len(records), "schedule": schedule,
		},
	})

	// Create notification for the creator
	if genBy != "" {
		_, _ = createNotificationRecord(Notification{
			UserID:         genBy,
			Title:          "Report Ready",
			Body:           fmt.Sprintf("Report %s has been generated with %d records.", def.Name, len(records)),
			Priority:       "medium",
			Category:       "report",
			SourceApp:      "enterprise",
			SourceEntity:   "report",
			SourceEntityID: run.ID,
			DeepLink:       run.URL,
			Metadata: map[string]interface{}{
				"report_id": def.ID, "record_count": len(records),
			},
		})
	}

	// Deliver via scheduled channel if configured
	if schedule != "" && def.Schedule != nil {
		deliverScheduledReport(def, run)
	}
}

// buildReportDataset assembles live data for a report definition.
func buildReportDataset(def ReportDefinition) (map[string]interface{}, []map[string]interface{}) {
	summary := map[string]interface{}{
		"report_name":  def.Name,
		"data_source":  def.DataSource,
		"source_app":   def.SourceApp,
		"grouping":     def.Grouping,
		"aggregation":  def.Aggregation,
		"chart_type":   def.ChartType,
		"generated_at": nowUTC(),
		"narrative":    def.Narrative,
		"branding":     def.Branding,
		"denormalized": []map[string]interface{}{},
	}

	records := []map[string]interface{}{}

	switch def.DataSource {
	case "kpis":
		// Report over all KPI definitions
		defs := listKPIDefinitions()
		for _, k := range defs {
			val := computeKPI(k, "organization", "")
			records = append(records, map[string]interface{}{
				"kpi_id":        k.ID,
				"kpi_name":      k.Name,
				"category":      k.Category,
				"value":         val.Value,
				"display_value": val.DisplayValue,
				"unit":          val.Unit,
				"status":        val.Status,
				"source_app":    k.SourceApp,
				"period":        val.Period,
				"computed_at":   val.ComputedAt,
			})
		}
		summary["record_count"] = len(records)
	case "submissions":
		// Report over StatCollect submissions from enterprise layer
		records = enterpriseRecordsToMap("statcollect", "submission")
		summary["record_count"] = len(records)
	case "projects":
		records = enterpriseRecordsToMap("pms", "project")
		summary["record_count"] = len(records)
	case "research":
		records = enterpriseRecordsToMap("rms", "research")
		summary["record_count"] = len(records)
	case "tickets":
		records = enterpriseRecordsToMap("helpdesk", "ticket")
		summary["record_count"] = len(records)
	case "facilities":
		records = enterpriseRecordsToMap("registry", "facility")
		summary["record_count"] = len(records)
	case "activity":
		// Report over timeline activity
		entries := fetchTimeline(200)
		for _, e := range entries {
			records = append(records, map[string]interface{}{
				"id": e.ID, "user": e.User, "application": e.Application,
				"entity": e.Entity, "entity_id": e.EntityID, "action": e.Action,
				"description": e.Description, "project_id": e.ProjectID,
				"timestamp": e.Timestamp,
			})
		}
		summary["record_count"] = len(records)
	case "quality":
		report := generateDataQualityReport("project", "", "")
		records = append(records, map[string]interface{}{
			"score":               report.Score,
			"completeness_score":  report.CompletenessScore,
			"duplicate_count":     report.DuplicateCount,
			"missing_value_count": report.MissingValueCount,
			"invalid_date_count":  report.InvalidDateCount,
			"invalid_coord_count": report.InvalidCoordCount,
			"outlier_count":       report.OutlierCount,
			"total_records":       report.TotalRecords,
			"low_reporting_flag":  report.LowReportingFlag,
			"generated_at":        report.GeneratedAt,
		})
		summary["record_count"] = len(records)
	default:
		// Generic: use source_app/entity from filters
		source := def.SourceApp
		entity := ""
		if v, ok := def.Filters["entity"].(string); ok {
			entity = v
		}
		records = enterpriseRecordsToMap(source, entity)
		summary["record_count"] = len(records)
	}

	// Apply filters
	records = applyReportFilters(records, def.Filters)

	// Apply grouping
	if def.Grouping != "" {
		grouped := groupReportRecords(records, def.Grouping)
		summary["groups"] = grouped
	}

	// Date range filtering
	if def.DateRange.Start != "" || def.DateRange.End != "" {
		records = filterByDateRange(records, def.DateRange)
		summary["date_range"] = map[string]interface{}{
			"start": def.DateRange.Start, "end": def.DateRange.End, "type": def.DateRange.Type,
		}
	}

	summary["records"] = records
	summary["filtered_count"] = len(records)
	return summary, records
}

func enterpriseRecordsToMap(sourceApp, entity string) []map[string]interface{} {
	records := []map[string]interface{}{}
	dataLayerMu.RLock()
	for _, rec := range enterpriseRecords {
		if sourceApp != "" && rec.SourceApp != sourceApp {
			continue
		}
		if entity != "" && rec.SourceEntity != entity {
			continue
		}
		records = append(records, map[string]interface{}{
			"id":              rec.ID,
			"source_app":      rec.SourceApp,
			"source_entity":   rec.SourceEntity,
			"source_id":       rec.SourceID,
			"project_id":      rec.ProjectID,
			"organization_id": rec.OrgID,
			"region":          rec.Region,
			"district":        rec.District,
			"status":          rec.Status,
			"event_type":      rec.EventType,
			"timestamp":       rec.Timestamp,
			"received_at":     rec.ReceivedAt,
			"correlation_id":  rec.Correlation,
			"value":           rec.Value,
			"geo_lat":         rec.GeoLat,
			"geo_lng":         rec.GeoLng,
			"metadata":        rec.Metadata,
		})
	}
	dataLayerMu.RUnlock()
	return records
}

func applyReportFilters(records []map[string]interface{}, filters map[string]interface{}) []map[string]interface{} {
	if filters == nil {
		return records
	}
	out := []map[string]interface{}{}
	for _, rec := range records {
		match := true
		for key, expected := range filters {
			if key == "entity" || key == "period" {
				continue
			}
			if v, ok := rec[key]; ok {
				if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", expected) {
					match = false
					break
				}
			}
		}
		if match {
			out = append(out, rec)
		}
	}
	return out
}

func groupReportRecords(records []map[string]interface{}, grouping string) map[string]int {
	groups := map[string]int{}
	for _, rec := range records {
		key := "unknown"
		if v, ok := rec[grouping].(string); ok && v != "" {
			key = v
		}
		groups[key]++
	}
	return groups
}

func filterByDateRange(records []map[string]interface{}, dr DateRange) []map[string]interface{} {
	out := []map[string]interface{}{}
	for _, rec := range records {
		ts, ok := rec["timestamp"].(string)
		if !ok {
			out = append(out, rec)
			continue
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			out = append(out, rec)
			continue
		}
		if dr.Start != "" {
			if start, err := time.Parse(time.RFC3339, dr.Start); err == nil && t.Before(start) {
				continue
			}
		}
		if dr.End != "" {
			if end, err := time.Parse(time.RFC3339, dr.End); err == nil && t.After(end) {
				continue
			}
		}
		out = append(out, rec)
	}
	return out
}

// ─── Scheduled Reports ─────────────────────────────────────────────
// runScheduledReports evaluates scheduled report definitions and
// generates runs when due.

func runScheduledReports() {
	defs := listReportDefinitions()
	for _, def := range defs {
		if def.Schedule == nil || !def.Schedule.Enabled {
			continue
		}
		lastRun, hasRun := scheduledRuns[def.ID]
		if hasRun && time.Since(lastRun) < time.Hour {
			continue // Avoid duplicate runs within an hour
		}
		switch def.Schedule.Frequency {
		case "daily":
			if !hasRun || time.Since(lastRun) >= 24*time.Hour {
				runID := fmt.Sprintf("rptrun_%d", time.Now().UnixNano())
				go executeReport(def, runID, "scheduler", "daily")
				scheduledRuns[def.ID] = time.Now()
			}
		case "weekly":
			if !hasRun || time.Since(lastRun) >= 7*24*time.Hour {
				runID := fmt.Sprintf("rptrun_%d", time.Now().UnixNano())
				go executeReport(def, runID, "scheduler", "weekly")
				scheduledRuns[def.ID] = time.Now()
			}
		case "monthly":
			if !hasRun || time.Since(lastRun) >= 30*24*time.Hour {
				runID := fmt.Sprintf("rptrun_%d", time.Now().UnixNano())
				go executeReport(def, runID, "scheduler", "monthly")
				scheduledRuns[def.ID] = time.Now()
			}
		}
	}
}

func deliverScheduledReport(def ReportDefinition, run ReportRun) {
	if def.Schedule == nil {
		return
	}
	channel := def.Schedule.Channel
	if channel == "" {
		channel = "workspace"
	}
	switch channel {
	case "notification":
		for _, r := range def.Schedule.Recipients {
			if r == "" {
				continue
			}
			_, _ = createNotificationRecord(Notification{
				UserID:         r,
				Title:          fmt.Sprintf("Scheduled Report: %s", def.Name),
				Body:           fmt.Sprintf("Your scheduled report is ready with %d records.", run.RecordCount),
				Priority:       "medium",
				Category:       "report",
				SourceApp:      "enterprise",
				SourceEntity:   "report",
				SourceEntityID: run.ID,
				DeepLink:       run.URL,
				Metadata: map[string]interface{}{
					"schedule": channel, "report_id": def.ID,
				},
			})
		}
	case "statchat":
		// Post to StatChat via API (best-effort)
		msg := map[string]interface{}{
			"message":  fmt.Sprintf("📊 **Scheduled Report:** %s (%d records). %s", def.Name, run.RecordCount, run.URL),
			"sender":   "statgate-analytics",
			"source":   "analytics",
			"metadata": map[string]interface{}{"report_id": def.ID, "run_id": run.ID},
		}
		data, _ := json.Marshal(msg)
		_, _ = postJSON(getEnv("STATCHAT_API_URL", "http://localhost:4000")+"/v1/messages", data)
	case "email":
		// Email delivery requires configuration; fall back to notification.
		for _, r := range def.Schedule.Recipients {
			if r == "" {
				continue
			}
			_, _ = createNotificationRecord(Notification{
				UserID:         r,
				Title:          fmt.Sprintf("Email Report: %s", def.Name),
				Body:           fmt.Sprintf("Report %s is attached for delivery. %d records.", def.Name, run.RecordCount),
				Priority:       "medium",
				Category:       "report",
				SourceApp:      "enterprise",
				SourceEntity:   "report",
				SourceEntityID: run.ID,
				DeepLink:       run.URL,
			})
		}
	default: // workspace
		for _, r := range def.Schedule.Recipients {
			if r == "" {
				continue
			}
			_, _ = createNotificationRecord(Notification{
				UserID:         r,
				Title:          fmt.Sprintf("Workspace Report: %s", def.Name),
				Body:           fmt.Sprintf("Report %s is available in your workspace. %d records.", def.Name, run.RecordCount),
				Priority:       "low",
				Category:       "report",
				SourceApp:      "enterprise",
				SourceEntity:   "report",
				SourceEntityID: run.ID,
				DeepLink:       run.URL,
			})
		}
	}
}

// ─── API Handlers ──────────────────────────────────────────────────

func handleListReportDefinitions(c *gin.Context) {
	defs := listReportDefinitions()
	c.JSON(200, gin.H{"count": len(defs), "reports": defs})
}

func handleGetReportDefinition(c *gin.Context) {
	id := c.Param("id")
	def, ok := getReportDefinition(id)
	if !ok {
		c.JSON(404, gin.H{"error": "report definition not found"})
		return
	}
	c.JSON(200, def)
}

func handleCreateReportDefinition(c *gin.Context) {
	var def ReportDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		c.JSON(400, gin.H{"error": "invalid report definition", "detail": err.Error()})
		return
	}
	if def.CreatedBy == "" {
		def.CreatedBy = c.GetHeader("X-User-ID")
	}
	if def.ExportFormat == "" {
		def.ExportFormat = "json"
	}
	registerReportDefinition(def)
	recordAudit("report.create", "enterprise", def.CreatedBy, map[string]interface{}{
		"report_id": def.ID, "name": def.Name, "data_source": def.DataSource,
	})
	c.JSON(201, def)
}

func handleUpdateReportDefinition(c *gin.Context) {
	id := c.Param("id")
	def, ok := getReportDefinition(id)
	if !ok {
		c.JSON(404, gin.H{"error": "report definition not found"})
		return
	}
	var updates ReportDefinition
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(400, gin.H{"error": "invalid report definition"})
		return
	}
	if updates.Name != "" {
		def.Name = updates.Name
	}
	if updates.Description != "" {
		def.Description = updates.Description
	}
	if updates.DataSource != "" {
		def.DataSource = updates.DataSource
	}
	if updates.SourceApp != "" {
		def.SourceApp = updates.SourceApp
	}
	if updates.Filters != nil {
		def.Filters = updates.Filters
	}
	if updates.Grouping != "" {
		def.Grouping = updates.Grouping
	}
	if updates.Aggregation != "" {
		def.Aggregation = updates.Aggregation
	}
	if updates.ChartType != "" {
		def.ChartType = updates.ChartType
	}
	if updates.Narrative != "" {
		def.Narrative = updates.Narrative
	}
	if updates.ExportFormat != "" {
		def.ExportFormat = updates.ExportFormat
	}
	if updates.Schedule != nil {
		def.Schedule = updates.Schedule
	}
	def.UpdatedAt = nowUTC()
	registerReportDefinition(def)
	c.JSON(200, def)
}

func handleGenerateReportFromDef(c *gin.Context) {
	id := c.Param("id")
	def, ok := getReportDefinition(id)
	if !ok {
		c.JSON(404, gin.H{"error": "report definition not found"})
		return
	}
	genBy := c.GetHeader("X-User-ID")
	runID := fmt.Sprintf("rptrun_%d", time.Now().UnixNano())
	go executeReport(def, runID, genBy, "")
	c.JSON(202, gin.H{"run_id": runID, "report_id": def.ID, "status": "queued"})
}

func handleListReportRuns(c *gin.Context) {
	reportID := c.Query("report_id")
	status := c.Query("status")
	limit := parseIntDefault(c.Query("limit"), 50)

	reportRunMu.RLock()
	runs := make([]ReportRun, 0, len(reportRuns))
	for _, r := range reportRuns {
		if reportID != "" && r.ReportID != reportID {
			continue
		}
		if status != "" && r.Status != status {
			continue
		}
		runs = append(runs, r)
	}
	reportRunMu.RUnlock()

	sort.Slice(runs, func(i, j int) bool { return runs[i].CreatedAt > runs[j].CreatedAt })
	if len(runs) > limit {
		runs = runs[:limit]
	}
	c.JSON(200, gin.H{"count": len(runs), "runs": runs})
}

func handleGetReportRun(c *gin.Context) {
	id := c.Param("runId")
	reportRunMu.RLock()
	run, ok := reportRuns[id]
	reportRunMu.RUnlock()
	if !ok {
		c.JSON(404, gin.H{"error": "report run not found"})
		return
	}
	c.JSON(200, run)
}

func handleGetReportRunData(c *gin.Context) {
	id := c.Param("runId")
	reportRunMu.RLock()
	run, ok := reportRuns[id]
	reportRunMu.RUnlock()
	if !ok {
		c.JSON(404, gin.H{"error": "report run not found"})
		return
	}
	if run.Status != "completed" {
		c.JSON(400, gin.H{"error": "report not ready", "status": run.Status})
		return
	}
	c.JSON(200, run.DataSummary)
}

// handleCreateReportFromBuilder handles the report builder POST.
func handleCreateReportFromBuilder(c *gin.Context) {
	handleCreateReportDefinition(c)
}
