package main

import (
	"github.com/gin-gonic/gin"
)

// ─── AI-Ready Analytics ────────────────────────────────────────────
// Clean interfaces for future AI services. The AI layer receives
// structured information — not screens. AIs can consume:
// Enterprise KPIs, Trends, Datasets, Data-quality indicators,
// Events, Reports, Research outputs, Project activity.

// handleAIAnalyticsCatalog provides a structured catalog for AI consumption.
func handleAIAnalyticsCatalog(c *gin.Context) {
	c.JSON(200, gin.H{
		"catalog_version": "1.0",
		"last_built":      nowUTC(),
		"resources": []map[string]interface{}{
			{"type": "kpis", "endpoint": "/api/analytics/ai/kpis", "description": "All KPI definitions with live computed values"},
			{"type": "trends", "endpoint": "/api/analytics/ai/trends?days=30", "description": "Time series trends from enterprise records"},
			{"type": "datasets", "endpoint": "/api/analytics/datasets", "description": "Registered enterprise datasets"},
			{"type": "data_quality", "endpoint": "/api/analytics/ai/quality", "description": "Data quality indicators across scopes"},
			{"type": "events", "endpoint": "/api/analytics/ai/events?limit=100", "description": "Recent enterprise events for context"},
			{"type": "reports", "endpoint": "/api/analytics/ai/reports", "description": "Report definitions and run summaries"},
			{"type": "anomalies", "endpoint": "/api/analytics/ai/anomalies", "description": "Detected anomalies for attention"},
			{"type": "alerts", "endpoint": "/api/analytics/ai/alerts", "description": "Active analytical alerts"},
		},
	})
}

// handleAIAnalyticsKPIs provides all KPI definitions with live values.
func handleAIAnalyticsKPIs(c *gin.Context) {
	defs := listKPIDefinitions()
	kpis := make([]map[string]interface{}, 0, len(defs))
	for _, def := range defs {
		val := computeKPI(def, "organization", "")
		kpis = append(kpis, map[string]interface{}{
			"kpi_id":        def.ID,
			"kpi_name":      def.Name,
			"category":      def.Category,
			"formula":       def.Formula,
			"source_app":    def.SourceApp,
			"scope":         def.Scope,
			"period":        def.Period,
			"value":         val.Value,
			"display_value": val.DisplayValue,
			"unit":          val.Unit,
			"status":        val.Status,
			"computed_at":   val.ComputedAt,
			"lineage":       val.Lineage,
		})
	}
	c.JSON(200, gin.H{"count": len(kpis), "kpis": kpis, "generated_at": nowUTC()})
}

// handleAIAnalyticsTrends provides time-series trends for AI consumption.
func handleAIAnalyticsTrends(c *gin.Context) {
	days := parseIntDefault(c.Query("days"), 30)
	source := c.Query("source")
	metric := c.Query("metric")

	// If no metric specified, provide overall activity trend
	trend := fetchTrendData(metric, source, days)
	c.JSON(200, gin.H{
		"metric":       metric,
		"source":       source,
		"days":         days,
		"points":       trend,
		"generated_at": nowUTC(),
	})
}

// handleAIAnalyticsQuality provides data quality indicators structured for AI.
func handleAIAnalyticsQuality(c *gin.Context) {
	report := generateDataQualityReport("project", "", "")
	c.JSON(200, gin.H{
		"quality_score":            report.Score,
		"completeness_score":       report.CompletenessScore,
		"duplicate_count":          report.DuplicateCount,
		"missing_value_count":      report.MissingValueCount,
		"invalid_date_count":       report.InvalidDateCount,
		"invalid_coord_count":      report.InvalidCoordCount,
		"outlier_count":            report.OutlierCount,
		"inconsistent_count":       report.InconsistentCount,
		"missing_identifier_count": report.MissingIdentifierCount,
		"low_reporting_flag":       report.LowReportingFlag,
		"total_records":            report.TotalRecords,
		"issues":                   report.Issues,
		"generated_at":             report.GeneratedAt,
	})
}

// handleAIAnalyticsEvents provides recent events structured for AI.
func handleAIAnalyticsEvents(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 100)
	entries := fetchTimeline(limit)
	events := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		events = append(events, map[string]interface{}{
			"id": e.ID, "user": e.User, "application": e.Application,
			"entity": e.Entity, "entity_id": e.EntityID, "action": e.Action,
			"description": e.Description, "project_id": e.ProjectID,
			"timestamp": e.Timestamp,
		})
	}
	c.JSON(200, gin.H{"count": len(events), "events": events, "generated_at": nowUTC()})
}

// handleAIAnalyticsReports provides report definitions for AI consumption.
func handleAIAnalyticsReports(c *gin.Context) {
	defs := listReportDefinitions()
	c.JSON(200, gin.H{"count": len(defs), "reports": defs})
}

// handleAIAnalyticsAnomalies provides detected anomalies for AI.
func handleAIAnalyticsAnomalies(c *gin.Context) {
	anomalyMu.RLock()
	anomalies := make([]map[string]interface{}, 0, len(anomalyRecords))
	for _, a := range anomalyRecords {
		anomalies = append(anomalies, map[string]interface{}{
			"anomaly_id":     a.ID,
			"rule_id":        a.RuleID,
			"rule_name":      a.RuleName,
			"type":           a.Type,
			"severity":       a.Severity,
			"scope":          a.Scope,
			"scope_id":       a.ScopeID,
			"project_id":     a.ProjectID,
			"description":    a.Description,
			"current_value":  a.CurrentValue,
			"expected_value": a.ExpectedValue,
			"status":         a.Status,
			"detected_at":    a.DetectedAt,
		})
	}
	anomalyMu.RUnlock()
	c.JSON(200, gin.H{"count": len(anomalies), "anomalies": anomalies})
}

// handleAIAnalyticsAlerts provides active alerts for AI.
func handleAIAnalyticsAlerts(c *gin.Context) {
	alertMu.RLock()
	alerts := make([]map[string]interface{}, 0, len(analyticalAlerts))
	for _, a := range analyticalAlerts {
		if a.Status == "resolved" {
			continue
		}
		alerts = append(alerts, map[string]interface{}{
			"alert_id":   a.ID,
			"rule_id":    a.RuleID,
			"rule_name":  a.RuleName,
			"title":      a.Title,
			"message":    a.Message,
			"priority":   a.Priority,
			"status":     a.Status,
			"metric":     a.Metric,
			"value":      a.Value,
			"threshold":  a.Threshold,
			"scope":      a.Scope,
			"scope_id":   a.ScopeID,
			"project_id": a.ProjectID,
			"created_at": a.CreatedAt,
			"actions":    a.Actions,
		})
	}
	alertMu.RUnlock()
	c.JSON(200, gin.H{"count": len(alerts), "alerts": alerts})
}

// handleAIAnalyticsProjectActivity provides project activity for AI.
func handleAIAnalyticsProjectActivity(c *gin.Context) {
	projectID := c.Query("project_id")
	if projectID == "" {
		c.JSON(400, gin.H{"error": "project_id required"})
		return
	}
	entries := fetchTimeline(200)
	activity := make([]map[string]interface{}, 0)
	for _, e := range entries {
		if e.ProjectID == projectID || e.EntityID == projectID {
			activity = append(activity, map[string]interface{}{
				"id": e.ID, "user": e.User, "application": e.Application,
				"entity": e.Entity, "entity_id": e.EntityID, "action": e.Action,
				"description": e.Description, "timestamp": e.Timestamp,
			})
		}
	}
	c.JSON(200, gin.H{"project_id": projectID, "count": len(activity), "activity": activity})
}
