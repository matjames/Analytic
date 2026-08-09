package main

import (
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Data Quality Intelligence ─────────────────────────────────────
// StatGate identifies data-quality problems automatically:
// missing values, duplicate submissions, impossible dates,
// invalid coordinates, outliers, inconsistent categories,
// unexpected values, missing identifiers, duplicate records,
// abnormally low reporting.

var (
	dqMu      sync.RWMutex
	dqReports = make(map[string]*DataQualityReport)
)

// ─── Quality Check Engine ──────────────────────────────────────────

// generateDataQualityReport computes a quality report for a scope.
func generateDataQualityReport(scope, scopeID, tenantID string) *DataQualityReport {
	report := &DataQualityReport{
		ID:          "dq_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Scope:       scope,
		ScopeID:     scopeID,
		TenantID:    tenantID,
		Score:       100,
		Issues:      []DataQualityIssue{},
		GeneratedAt: nowUTC(),
	}

	// Collect enterprise records for the scope
	records := fetchEnterpriseRecords("", "", "", "", "", 5000)

	// Filter by scope
	filtered := make([]EnterpriseRecord, 0, len(records))
	for _, rec := range records {
		if scopeID == "" || rec.ProjectID == scopeID {
			filtered = append(filtered, rec)
		}
	}
	report.TotalRecords = int64(len(filtered))

	if len(filtered) == 0 {
		report.LowReportingFlag = true
		report.Issues = append(report.Issues, DataQualityIssue{
			Type: "low_reporting", Severity: "warning",
			Message: "No records have been received for this scope.",
			Entity:  scope, EntityID: scopeID,
		})
		dqMu.Lock()
		dqReports[report.ID] = report
		dqMu.Unlock()
		return report
	}

	// ── Duplicate Detection ──
	seen := make(map[string][]string) // source_entity -> source_ids
	duplicates := 0
	for _, rec := range filtered {
		if rec.SourceID == "" {
			continue
		}
		if prev, ok := seen[rec.SourceApp+":"+rec.SourceEntity]; ok {
			for _, p := range prev {
				if p == rec.SourceID {
					duplicates++
					report.Issues = append(report.Issues, DataQualityIssue{
						Type: "duplicate_record", Severity: "warning",
						Message: "Duplicate record found", Entity: rec.SourceEntity,
						EntityID: rec.SourceID, Field: "source_id",
						Details: map[string]interface{}{"source_app": rec.SourceApp},
					})
					break
				}
			}
		}
		seen[rec.SourceApp+":"+rec.SourceEntity] = append(seen[rec.SourceApp+":"+rec.SourceEntity], rec.SourceID)
	}
	report.DuplicateCount = int64(duplicates)

	// ── Missing & Invalid Value Checks ──
	missingValues := 0
	invalidDates := 0
	invalidCoords := 0
	missingIDs := 0
	outliers := 0
	inconsistent := 0

	values := make([]float64, 0, len(filtered))

	for _, rec := range filtered {
		// Missing identifiers
		if rec.SourceID == "" {
			missingIDs++
			report.Issues = append(report.Issues, DataQualityIssue{
				Type: "missing_identifier", Severity: "critical",
				Message: "Record missing source identifier", Entity: rec.SourceEntity,
				EntityID: rec.ID, Field: "source_id",
			})
		}

		// Invalid dates
		if rec.Timestamp != "" {
			if _, err := time.Parse(time.RFC3339, rec.Timestamp); err != nil {
				invalidDates++
				report.Issues = append(report.Issues, DataQualityIssue{
					Type: "invalid_date", Severity: "warning",
					Message: "Invalid timestamp format", Entity: rec.SourceEntity,
					EntityID: rec.SourceID, Field: "timestamp",
					Details: map[string]interface{}{"value": rec.Timestamp},
				})
			}
		}

		// Invalid coordinates
		if rec.GeoLat != 0 || rec.GeoLng != 0 {
			if math.Abs(rec.GeoLat) > 90 || math.Abs(rec.GeoLng) > 180 {
				invalidCoords++
				report.Issues = append(report.Issues, DataQualityIssue{
					Type: "invalid_coordinates", Severity: "warning",
					Message: "GPS coordinates outside valid range", Entity: rec.SourceEntity,
					EntityID: rec.SourceID, Field: "gps_coordinates",
					Details: map[string]interface{}{"lat": rec.GeoLat, "lng": rec.GeoLng},
				})
			}
		}

		// Missing values in metadata
		if rec.Metadata != nil {
			if v, ok := rec.Metadata["name"].(string); ok && v == "" {
				missingValues++
			}
			// Check for empty enumerator
			if v, ok := rec.Metadata["enumerator_id"].(string); ok && v == "" {
				missingValues++
				report.Issues = append(report.Issues, DataQualityIssue{
					Type: "missing_value", Severity: "warning",
					Message: "Missing enumerator identifier", Entity: rec.SourceEntity,
					EntityID: rec.SourceID, Field: "enumerator_id",
				})
			}
			// Status consistency
			if v, ok := rec.Metadata["status"].(string); ok {
				known := map[string]bool{"received": true, "approved": true, "rejected": true, "active": true, "in_progress": true, "completed": true, "closed": true}
				if !known[v] && v != "" {
					inconsistent++
				}
			}
		}

		// Numeric values for outlier detection
		if rec.Value != 0 {
			values = append(values, rec.Value)
		}
	}

	report.MissingValueCount = int64(missingValues)
	report.InvalidDateCount = int64(invalidDates)
	report.InvalidCoordCount = int64(invalidCoords)
	report.MissingIdentifierCount = int64(missingIDs)
	report.InconsistentCount = int64(inconsistent)

	// ── Outlier Detection (simple std-dev based) ──
	if len(values) >= 5 {
		mean, stdDev := computeMeanStdDev(values)
		if stdDev > 0 {
			for _, v := range values {
				if math.Abs(v-mean) > 3*stdDev {
					outliers++
				}
			}
		}
	}
	report.OutlierCount = int64(outliers)

	// ── Completeness Score ──
	totalChecks := float64(report.TotalRecords * 5)
	problems := float64(report.MissingValueCount + report.InvalidDateCount + report.MissingIdentifierCount + report.InvalidCoordCount)
	completeness := 100.0
	if totalChecks > 0 {
		completeness = 100.0 - (problems / totalChecks * 100)
	}
	report.CompletenessScore = math.Round(completeness*10) / 10

	// ── Overall Score ──
	score := 100.0
	score -= float64(report.DuplicateCount) * 8
	score -= float64(report.InvalidDateCount) * 6
	score -= float64(report.InvalidCoordCount) * 6
	score -= float64(report.OutlierCount) * 4
	score -= float64(report.InconsistentCount) * 5
	score -= float64(report.MissingIdentifierCount) * 10
	score -= float64(report.MissingValueCount) * 3
	if score < 0 {
		score = 0
	}
	report.Score = math.Round(score*10) / 10

	if report.TotalRecords < 3 {
		report.LowReportingFlag = true
	}

	dqMu.Lock()
	dqReports[report.ID] = report
	dqMu.Unlock()

	return report
}

func computeMeanStdDev(values []float64) (mean, stdDev float64) {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))
	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))
	stdDev = math.Sqrt(variance)
	return mean, stdDev
}

// ─── API Handlers ──────────────────────────────────────────────────

func handleDataQualityReport(c *gin.Context) {
	scopeID := c.Query("scope_id")
	scope := c.Query("scope")
	if scope == "" {
		scope = "project"
	}
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		tenantID = getEnvValue("STATGATE_TENANT_ID")
		if tenantID == "" {
			tenantID = "statgate"
		}
	}
	report := generateDataQualityReport(scope, scopeID, tenantID)
	c.JSON(200, report)
}

func handleListDataQualityReports(c *gin.Context) {
	dqMu.RLock()
	reports := make([]*DataQualityReport, 0, len(dqReports))
	for _, r := range dqReports {
		reports = append(reports, r)
	}
	dqMu.RUnlock()
	sort.Slice(reports, func(i, j int) bool { return reports[i].GeneratedAt > reports[j].GeneratedAt })
	if len(reports) > 50 {
		reports = reports[:50]
	}
	c.JSON(200, gin.H{"count": len(reports), "reports": reports})
}

func handleDataQualityIssues(c *gin.Context) {
	scopeID := c.Query("scope_id")
	scope := c.Query("scope")
	if scope == "" {
		scope = "project"
	}
	report := generateDataQualityReport(scope, scopeID, "")
	c.JSON(200, gin.H{"issues": report.Issues, "count": len(report.Issues), "score": report.Score})
}
