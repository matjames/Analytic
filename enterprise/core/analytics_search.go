package main

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// ─── Analytics Search ──────────────────────────────────────────────
// Enterprise Search discovers dashboards, KPIs, reports, datasets,
// alerts and analytical outputs.

func analyticsSearch(query string) []AnalyticsSearchResult {
	if strings.TrimSpace(query) == "" {
		return []AnalyticsSearchResult{}
	}
	q := strings.ToLower(query)
	results := []AnalyticsSearchResult{}

	// Search KPI definitions
	for _, def := range listKPIDefinitions() {
		score := searchScore(def.Name+" "+def.Description+" "+def.Category, q)
		if score > 0 {
			results = append(results, AnalyticsSearchResult{
				Type: "kpi", ID: def.ID, Title: def.Name,
				Description: def.Description, Source: "enterprise",
				Score: score + 5, URL: "/analytics/kpi/" + def.ID,
				Meta: map[string]interface{}{
					"category": def.Category, "source_app": def.SourceApp,
					"scope": def.Scope, "formula": def.Formula,
				},
			})
		}
	}

	// Search dashboards
	for _, d := range listAnalyticsDashboards() {
		score := searchScore(d.Name+" "+d.Description+" "+d.Type, q)
		if score > 0 {
			results = append(results, AnalyticsSearchResult{
				Type: "dashboard", ID: d.ID, Title: d.Name,
				Description: d.Description, Source: "enterprise",
				Score: score + 4, URL: "/analytics/dashboards/" + d.ID,
				Meta: map[string]interface{}{"type": d.Type},
			})
		}
	}

	// Search report definitions
	for _, r := range listReportDefinitions() {
		score := searchScore(r.Name+" "+r.Description+" "+r.DataSource, q)
		if score > 0 {
			results = append(results, AnalyticsSearchResult{
				Type: "report", ID: r.ID, Title: r.Name,
				Description: r.Description, Source: "enterprise",
				Score: score + 3, URL: "/analytics/reports/" + r.ID,
				Meta: map[string]interface{}{
					"data_source": r.DataSource, "source_app": r.SourceApp,
				},
			})
		}
	}

	// Search datasets
	for _, ds := range listDatasets() {
		score := searchScore(ds.Name+" "+ds.Description+" "+ds.SourceApp, q)
		if score > 0 {
			results = append(results, AnalyticsSearchResult{
				Type: "dataset", ID: ds.ID, Title: ds.Name,
				Description: ds.Description, Source: "enterprise",
				Score: score + 2, URL: "/analytics/datasets/" + ds.ID,
				Meta: map[string]interface{}{
					"source_app": ds.SourceApp, "record_count": ds.RecordCount,
				},
			})
		}
	}

	// Search alert rules
	for _, rule := range listAlertRules() {
		score := searchScore(rule.Name+" "+rule.Description, q)
		if score > 0 {
			results = append(results, AnalyticsSearchResult{
				Type: "alert_rule", ID: rule.ID, Title: rule.Name,
				Description: rule.Description, Source: "enterprise",
				Score: score + 1, URL: "/analytics/alerts/rules",
				Meta: map[string]interface{}{"metric": rule.Metric},
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > 20 {
		results = results[:20]
	}
	return results
}

// searchScore returns a relevance score for a query against text.
func searchScore(text, query string) int {
	lower := strings.ToLower(text)
	score := 0
	if strings.Contains(lower, query) {
		score += 10
	}
	// Token-based matching
	for _, token := range strings.Fields(query) {
		if strings.Contains(lower, token) {
			score += 3
		}
	}
	// Exact title match
	if lower == query {
		score += 15
	}
	return score
}

// handleAnalyticsSearch is the API handler for analytics-specific search.
func handleAnalyticsSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(400, gin.H{"error": "q parameter required"})
		return
	}
	results := analyticsSearch(query)
	c.JSON(200, gin.H{"query": query, "count": len(results), "results": results})
}
