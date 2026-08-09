package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

// ─── Data Lineage ──────────────────────────────────────────────────
// Every analytical result is traceable: "Where did this number come from?"
// For important metrics, StatGate provides: source application, source
// dataset, calculation, reporting period, filters, last update,
// responsible system.

// buildLineageForKPI returns a lineage record for a specific KPI value.
func buildLineageForKPI(kpiID string) (*DataLineage, error) {
	def, ok := getKPIDefinition(kpiID)
	if !ok {
		return nil, fmt.Errorf("KPI definition not found: %s", kpiID)
	}
	value := computeKPI(def, def.Scope, "")
	return buildLineage(def, value), nil
}

// handleKPIlineage returns lineage for a KPI.
func handleKPILineage(c *gin.Context) {
	kpiID := c.Param("id")
	lineage, err := buildLineageForKPI(kpiID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, lineage)
}

// handleDashboardLineage returns lineage for all KPIs on a dashboard.
func handleDashboardLineage(c *gin.Context) {
	dashID := c.Param("id")
	dash, ok := getAnalyticsDashboard(dashID)
	if !ok {
		c.JSON(404, gin.H{"error": "dashboard not found"})
		return
	}

	lineages := []map[string]interface{}{}
	for _, section := range dash.Sections {
		for _, item := range section.Items {
			if def, ok := getKPIDefinition(item); ok {
				value := computeKPI(def, def.Scope, "")
				lineages = append(lineages, map[string]interface{}{
					"kpi_id":   def.ID,
					"kpi_name": def.Name,
					"value":    value.Value,
					"display":  value.DisplayValue,
					"unit":     value.Unit,
					"lineage":  value.Lineage,
				})
			}
		}
	}

	c.JSON(200, gin.H{
		"dashboard_id":   dash.ID,
		"dashboard_name": dash.Name,
		"count":          len(lineages),
		"lineages":       lineages,
		"generated_at":   nowUTC(),
	})
}

// handleRecordLineage returns the provenance chain for an enterprise record.
func handleRecordLineage(c *gin.Context) {
	sourceApp := c.Param("source")
	sourceEntity := c.Param("entity")
	sourceID := c.Param("id")

	rec, ok := fetchEnterpriseRecord(sourceApp, sourceEntity, sourceID)
	if !ok {
		c.JSON(404, gin.H{"error": "record not found"})
		return
	}

	// Build provenance chain
	provenance := map[string]interface{}{
		"record_id":          rec.ID,
		"source_application": rec.SourceApp,
		"source_entity":      rec.SourceEntity,
		"source_id":          rec.SourceID,
		"tenant_id":          rec.TenantID,
		"organization_id":    rec.OrgID,
		"project_id":         rec.ProjectID,
		"event_type":         rec.EventType,
		"correlation_id":     rec.Correlation,
		"data_version":       rec.DataVersion,
		"timestamp":          rec.Timestamp,
		"received_at":        rec.ReceivedAt,
		"responsible_system": "statgate-enterprise-core",
		"original_system":    "System of record: " + rec.SourceApp,
	}

	// Include related timeline events
	timelineCount := 0
	entries := fetchTimeline(100)
	for _, e := range entries {
		if e.Entity == rec.SourceEntity && e.EntityID == rec.SourceID {
			timelineCount++
		}
	}
	provenance["related_timeline_events"] = timelineCount

	// Include correlation chain if present
	if rec.Correlation != "" {
		corrEntries := fetchCorrelationEvents(rec.Correlation)
		provenance["correlation_chain"] = corrEntries
	}

	c.JSON(200, provenance)
}

// fetchCorrelationEvents returns events sharing a correlation ID.
func fetchCorrelationEvents(correlation string) []map[string]interface{} {
	if redisClient == nil || correlation == "" {
		return []map[string]interface{}{}
	}
	key := fmt.Sprintf("statgate:events:correlation:%s", correlation)
	raw, _ := redisClient.LRange(contextForLineage(), key, 0, 99).Result()
	events := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		var ev map[string]interface{}
		if err := json.Unmarshal([]byte(item), &ev); err == nil {
			events = append(events, ev)
		}
	}
	return events
}

// handleLineageOverview provides an overview of lineage for all active KPIs.
func handleLineageOverview(c *gin.Context) {
	defs := listKPIDefinitions()
	overview := make([]map[string]interface{}, 0, len(defs))
	for _, def := range defs {
		value := computeKPI(def, def.Scope, "")
		overview = append(overview, map[string]interface{}{
			"kpi_id":             def.ID,
			"kpi_name":           def.Name,
			"formula":            def.Formula,
			"source_app":         def.SourceApp,
			"reporting_period":   fmt.Sprintf("%s → %s", value.PeriodStart, value.PeriodEnd),
			"last_updated":       value.ComputedAt,
			"responsible_system": "statgate-enterprise-core",
			"raw_record_count":   value.Lineage.RawRecordCount,
			"status":             value.Status,
			"value":              value.DisplayValue,
		})
	}
	c.JSON(200, gin.H{"count": len(overview), "lineages": overview})
}

// contextForLineage returns a background context for Redis operations.
func contextForLineage() context.Context {
	return context.Background()
}
