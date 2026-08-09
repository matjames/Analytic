package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Anomaly Detection Foundation ──────────────────────────────────
// Rule-based anomaly detection for Phase IV. The architecture allows
// future statistical and AI-based anomaly detection by separating
// detection rules from the detection engine.

var (
	anomalyMu      sync.RWMutex
	anomalyRules   = make(map[string]AnomalyRule)
	anomalyRecords = make(map[string]AnomalyRecord)
)

// ─── Anomaly Rule Registry ─────────────────────────────────────────

func registerAnomalyRule(rule AnomalyRule) {
	anomalyMu.Lock()
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("anr_%d", time.Now().UnixNano())
	}
	if rule.CreatedAt == "" {
		rule.CreatedAt = nowUTC()
	}
	anomalyRules[rule.ID] = rule
	anomalyMu.Unlock()
}

func listAnomalyRules() []AnomalyRule {
	anomalyMu.RLock()
	defer anomalyMu.RUnlock()
	out := make([]AnomalyRule, 0, len(anomalyRules))
	for _, rule := range anomalyRules {
		out = append(out, rule)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func getAnomalyRule(id string) (AnomalyRule, bool) {
	anomalyMu.RLock()
	defer anomalyMu.RUnlock()
	rule, ok := anomalyRules[id]
	return rule, ok
}

// bootstrapAnomalyRules registers initial rule-based detection rules.
func bootstrapAnomalyRules() {
	now := nowUTC()
	rules := []AnomalyRule{
		{
			ID: "anr_submission_drop", Name: "Sudden Submission Drop",
			Description: "Detects a sudden drop in field submissions",
			Type:        "drop", Metric: "submission_volume",
			WindowMinutes: 1440, ThresholdPct: 50, Severity: "high",
			Enabled: true, CreatedAt: now,
		},
		{
			ID: "anr_ticket_spike", Name: "Unexpected Ticket Increase",
			Description: "Detects unexpected increases in HelpDesk tickets",
			Type:        "spike", Metric: "ticket_volume",
			WindowMinutes: 1440, ThresholdPct: 100, Severity: "medium",
			Enabled: true, CreatedAt: now,
		},
		{
			ID: "anr_facility_inactive", Name: "Unusually Inactive Facility",
			Description: "Detects facilities with no recent submissions",
			Type:        "inactivity", Metric: "facility_reporting",
			WindowMinutes: 4320, ThresholdPct: 0, Severity: "high",
			Enabled: true, CreatedAt: now,
		},
		{
			ID: "anr_project_delay", Name: "Sudden Project Delay",
			Description: "Detects projects that have stalled or regressed",
			Type:        "delay", Metric: "project_progress",
			WindowMinutes: 10080, ThresholdPct: 10, Severity: "high",
			Enabled: true, CreatedAt: now,
		},
		{
			ID: "anr_research_activity", Name: "Unusual Research Activity",
			Description: "Detects unusual patterns in research activity",
			Type:        "pattern", Metric: "research_activity",
			WindowMinutes: 4320, ThresholdPct: 50, Severity: "medium",
			Enabled: true, CreatedAt: now,
		},
		{
			ID: "anr_reporting_pattern", Name: "Unexpected Reporting Pattern",
			Description: "Detects unexpected reporting patterns",
			Type:        "pattern", Metric: "reporting_pattern",
			WindowMinutes: 1440, ThresholdPct: 50, Severity: "low",
			Enabled: true, CreatedAt: now,
		},
	}

	for _, rule := range rules {
		registerAnomalyRule(rule)
	}
}

// ─── Anomaly Detection Engine ──────────────────────────────────────
// runAnomalyDetection executes all enabled anomaly rules.

func runAnomalyDetection() {
	rules := listAnomalyRules()
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		detectAnomaly(rule)
	}
}

func detectAnomaly(rule AnomalyRule) {
	switch rule.Metric {
	case "submission_volume":
		detectSubmissionVolumeAnomaly(rule)
	case "ticket_volume":
		detectTicketVolumeAnomaly(rule)
	case "facility_reporting":
		detectFacilityInactivityAnomaly(rule)
	case "project_progress":
		detectProjectDelayAnomaly(rule)
	}
}

// detectSubmissionVolumeAnomaly detects drops in submission volume.
func detectSubmissionVolumeAnomaly(rule AnomalyRule) {
	windowStart := time.Now().Add(-time.Duration(rule.WindowMinutes) * time.Minute)
	prevWindowStart := windowStart.Add(-time.Duration(rule.WindowMinutes) * time.Minute)

	currentCount := int64(0)
	prevCount := int64(0)

	dataLayerMu.RLock()
	for _, rec := range enterpriseRecords {
		if rec.SourceApp != "statcollect" || rec.SourceEntity != "submission" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, rec.Timestamp); err == nil {
			if t.After(prevWindowStart) && t.Before(windowStart) {
				prevCount++
			}
			if t.After(windowStart) {
				currentCount++
			}
		}
	}
	dataLayerMu.RUnlock()

	if prevCount < 3 {
		return
	}
	dropPct := (float64(prevCount-currentCount) / float64(prevCount)) * 100
	if dropPct > rule.ThresholdPct {
		recordAnomaly(AnomalyRecord{
			ID:            fmt.Sprintf("anm_%d", time.Now().UnixNano()),
			RuleID:        rule.ID,
			RuleName:      rule.Name,
			Type:          rule.Type,
			Severity:      rule.Severity,
			Description:   fmt.Sprintf("Submission volume dropped by %.1f%% compared to the previous period", dropPct),
			CurrentValue:  float64(currentCount),
			ExpectedValue: float64(prevCount),
			WindowStart:   prevWindowStart.Format(time.RFC3339),
			WindowEnd:     windowStart.Format(time.RFC3339),
			Status:        "detected",
			DetectedAt:    nowUTC(),
		})
	}
}

func detectTicketVolumeAnomaly(rule AnomalyRule) {
	windowStart := time.Now().Add(-time.Duration(rule.WindowMinutes) * time.Minute)
	prevWindowStart := windowStart.Add(-time.Duration(rule.WindowMinutes) * time.Minute)

	currentCount := int64(0)
	prevCount := int64(0)

	dataLayerMu.RLock()
	for _, rec := range enterpriseRecords {
		if rec.SourceApp != "helpdesk" || rec.SourceEntity != "ticket" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, rec.Timestamp); err == nil {
			if t.After(prevWindowStart) && t.Before(windowStart) {
				prevCount++
			}
			if t.After(windowStart) {
				currentCount++
			}
		}
	}
	dataLayerMu.RUnlock()

	if prevCount < 2 {
		return
	}
	increasePct := (float64(currentCount-prevCount) / float64(prevCount)) * 100
	if currentCount > prevCount && increasePct > rule.ThresholdPct {
		recordAnomaly(AnomalyRecord{
			ID:            fmt.Sprintf("anm_%d", time.Now().UnixNano()),
			RuleID:        rule.ID,
			RuleName:      rule.Name,
			Type:          rule.Type,
			Severity:      rule.Severity,
			Description:   fmt.Sprintf("Ticket volume increased by %.1f%% compared to the previous period", increasePct),
			CurrentValue:  float64(currentCount),
			ExpectedValue: float64(prevCount),
			WindowStart:   prevWindowStart.Format(time.RFC3339),
			WindowEnd:     windowStart.Format(time.RFC3339),
			Status:        "detected",
			DetectedAt:    nowUTC(),
		})
	}
}

func detectFacilityInactivityAnomaly(rule AnomalyRule) {
	inactiveCutoff := time.Now().Add(-time.Duration(rule.WindowMinutes) * time.Minute)

	facilityActivity := map[string]time.Time{}
	dataLayerMu.RLock()
	for _, rec := range enterpriseRecords {
		if rec.FacilityID == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, rec.Timestamp); err == nil {
			if existing, ok := facilityActivity[rec.FacilityID]; !ok || t.After(existing) {
				facilityActivity[rec.FacilityID] = t
			}
		}
	}
	dataLayerMu.RUnlock()

	for facilityID, lastActive := range facilityActivity {
		if lastActive.Before(inactiveCutoff) {
			recordAnomaly(AnomalyRecord{
				ID:            fmt.Sprintf("anm_%d", time.Now().UnixNano()),
				RuleID:        rule.ID,
				RuleName:      rule.Name,
				Type:          rule.Type,
				Severity:      rule.Severity,
				Scope:         "facility",
				ScopeID:       facilityID,
				Description:   fmt.Sprintf("Facility %s has been inactive since %s", facilityID, lastActive.Format(time.RFC3339)),
				CurrentValue:  float64(time.Since(lastActive).Hours()),
				ExpectedValue: 0,
				WindowStart:   inactiveCutoff.Format(time.RFC3339),
				WindowEnd:     nowUTC(),
				Status:        "detected",
				DetectedAt:    nowUTC(),
			})
		}
	}
}

func detectProjectDelayAnomaly(rule AnomalyRule) {
	cutoff := time.Now().Add(-time.Duration(rule.WindowMinutes) * time.Minute)

	dataLayerMu.RLock()
	projectRecords := make(map[string]EnterpriseRecord)
	for _, rec := range enterpriseRecords {
		if rec.ProjectID == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, rec.Timestamp); err == nil {
			if existing, ok := projectRecords[rec.ProjectID]; !ok || t.After(existing.TimestampTime()) {
				projectRecords[rec.ProjectID] = rec
			}
		}
	}
	dataLayerMu.RUnlock()

	for projectID := range projectRecords {
		if data := fetchJSON(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects/" + projectID); data != nil {
			if lastUpdatedStr, ok := data["updated_time"].(string); ok {
				if lastUpdated, err := time.Parse(time.RFC3339, lastUpdatedStr); err == nil {
					if lastUpdated.Before(cutoff) {
						recordAnomaly(AnomalyRecord{
							ID:            fmt.Sprintf("anm_%d", time.Now().UnixNano()),
							RuleID:        rule.ID,
							RuleName:      rule.Name,
							Type:          rule.Type,
							Severity:      rule.Severity,
							Scope:         "project",
							ScopeID:       projectID,
							ProjectID:     projectID,
							Description:   fmt.Sprintf("Project %s has not been updated since %s", projectID, lastUpdated.Format(time.RFC3339)),
							CurrentValue:  float64(time.Since(lastUpdated).Hours()),
							ExpectedValue: 0,
							WindowStart:   cutoff.Format(time.RFC3339),
							WindowEnd:     nowUTC(),
							Status:        "detected",
							DetectedAt:    nowUTC(),
						})
					}
				}
			}
		}
	}
}

// recordAnomaly stores an anomaly and raises an alert.
func recordAnomaly(anomaly AnomalyRecord) {
	anomalyMu.Lock()
	anomalyRecords[anomaly.ID] = anomaly
	anomalyMu.Unlock()

	if redisClient != nil {
		data, _ := json.Marshal(anomaly)
		ctx := context.Background()
		key := fmt.Sprintf("statgate:anomalies:%s", anomaly.ID)
		redisClient.Set(ctx, key, string(data), 30*24*time.Hour)
		redisClient.LPush(ctx, "statgate:anomalies:index", anomaly.ID)
		redisClient.LTrim(ctx, "statgate:anomalies:index", 0, 999)
	}

	publishAnalyticsEvent("anomaly.detected", map[string]interface{}{
		"anomaly_id": anomaly.ID, "rule_id": anomaly.RuleID, "rule_name": anomaly.RuleName,
		"type": anomaly.Type, "severity": anomaly.Severity, "scope": anomaly.Scope,
		"scope_id": anomaly.ScopeID, "project_id": anomaly.ProjectID,
		"description": anomaly.Description, "current_value": anomaly.CurrentValue,
		"expected_value": anomaly.ExpectedValue, "detected_at": anomaly.DetectedAt,
	})

	recordAudit("anomaly.detected", "enterprise", "", map[string]interface{}{
		"anomaly_id": anomaly.ID, "rule_id": anomaly.RuleID, "type": anomaly.Type,
		"severity": anomaly.Severity, "scope": anomaly.Scope, "scope_id": anomaly.ScopeID,
	})
}

// ─── Helpers ───────────────────────────────────────────────────────

func (r EnterpriseRecord) TimestampTime() time.Time {
	t, _ := time.Parse(time.RFC3339, r.Timestamp)
	return t
}

// ─── API Handlers ──────────────────────────────────────────────────

func handleListAnomalyRules(c *gin.Context) {
	rules := listAnomalyRules()
	c.JSON(200, gin.H{"count": len(rules), "rules": rules})
}

func handleRunAnomalyDetection(c *gin.Context) {
	runAnomalyDetection()
	c.JSON(200, gin.H{"status": "completed", "timestamp": nowUTC()})
}

func handleListAnomalies(c *gin.Context) {
	severity := c.Query("severity")
	status := c.Query("status")
	limit := parseIntDefault(c.Query("limit"), 50)

	anomalyMu.RLock()
	anomalies := make([]AnomalyRecord, 0, len(anomalyRecords))
	for _, a := range anomalyRecords {
		if severity != "" && a.Severity != severity {
			continue
		}
		if status != "" && a.Status != status {
			continue
		}
		anomalies = append(anomalies, a)
	}
	anomalyMu.RUnlock()

	sort.Slice(anomalies, func(i, j int) bool { return anomalies[i].DetectedAt > anomalies[j].DetectedAt })
	if len(anomalies) > limit {
		anomalies = anomalies[:limit]
	}
	c.JSON(200, gin.H{"count": len(anomalies), "anomalies": anomalies})
}

func handleResolveAnomaly(c *gin.Context) {
	id := c.Param("id")
	anomalyMu.Lock()
	if anomaly, ok := anomalyRecords[id]; ok {
		anomaly.Status = "resolved"
		anomalyRecords[id] = anomaly
		anomalyMu.Unlock()
		c.JSON(200, gin.H{"status": "resolved", "id": id})
		return
	}
	anomalyMu.Unlock()
	c.JSON(404, gin.H{"error": "anomaly not found"})
}

// math used for potential future statistical detection
func zScore(value, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}
	return (value - mean) / stdDev
}
