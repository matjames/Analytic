package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── Phase IV: Automated Tests for Analytical Functionality ────────
// Covers: KPI calculations, aggregations, permissions, data freshness,
// event-driven updates, dashboard rendering, export permissions,
// search results, data lineage, alert generation.

// setupTestRouter builds a test router with all routes registered.
func setupTestRouter() *ginTestRouter {
	// Register Phase IV bootstraps
	bootstrapDataLayer()
	bootstrapKPIs()
	bootstrapAlertRules()
	bootstrapAnomalyRules()
	bootstrapAnalyticsDashboards()

	// Register Phase V bootstraps
	bootstrapWorkflowTemplates()
	bootstrapEscalationPolicies()
	bootstrapSLAs()

	r := setupRouter()
	registerRoutes(r)
	return &ginTestRouter{engine: r}
}

type ginTestRouter struct {
	engine http.Handler
}

func (g *ginTestRouter) do(method, path, body string) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	g.engine.ServeHTTP(w, req)
	return w
}

// ─── KPI Engine Tests ─────────────────────────────────────────────

func TestKPIDefinitionsBootstrapped(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/kpis", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count int             `json:"count"`
		KPIs  []KPIDefinition `json:"kpis"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count == 0 {
		t.Fatal("expected at least one KPI definition")
	}
}

func TestKPIGetReturnsLiveValue(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/kpis/kpi_active_projects", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp KPIValue
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.KPIID != "kpi_active_projects" {
		t.Fatalf("expected kpi_active_projects, got %s", resp.KPIID)
	}
	if resp.ComputedAt == "" {
		t.Fatal("expected computed_at timestamp")
	}
	if resp.Lineage == nil {
		t.Fatal("expected lineage to be attached")
	}
}

func TestKPIDrillDown(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/kpis/kpi_project_progress/drilldown", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		KPIID     string         `json:"kpi_id"`
		Breakdown []KPIComponent `json:"breakdown"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.KPIID == "" {
		t.Fatal("expected kpi_id")
	}
}

func TestKPIStatusEvaluation(t *testing.T) {
	// Test below threshold
	val := KPIValue{KPIID: "kpi_survey_completion_rate", Value: 50}
	status := evaluateKPIStatus(val)
	if status == "" {
		t.Fatal("expected a status evaluation")
	}
}

// ─── Data Layer Tests ─────────────────────────────────────────────

func TestEnterpriseRecordIngestion(t *testing.T) {
	rec := EnterpriseRecord{
		SourceApp:    "test",
		SourceEntity: "record",
		SourceID:     "rec-1",
		TenantID:     "test-tenant",
		ProjectID:    "PRJ-1",
		Timestamp:    nowUTC(),
	}
	ingestEnterpriseRecord(rec)

	fetched, ok := fetchEnterpriseRecord("test", "record", "rec-1")
	if !ok {
		t.Fatal("expected record to be found after ingestion")
	}
	if fetched.ProjectID != "PRJ-1" {
		t.Fatalf("expected project PRJ-1, got %s", fetched.ProjectID)
	}
}

func TestDatasetsRegistered(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/datasets", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count == 0 {
		t.Fatal("expected at least one dataset")
	}
}

func TestDataFreshness(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/freshness", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count == 0 {
		t.Fatal("expected freshness data")
	}
}

// ─── Data Quality Tests ───────────────────────────────────────────

func TestDataQualityReport(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/quality", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp DataQualityReport
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.GeneratedAt == "" {
		t.Fatal("expected generated_at timestamp")
	}
}

func TestDataQualityOutlierDetection(t *testing.T) {
	// Values with a clear outlier (1000 among tightly-clustered 10s)
	values := []float64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 1000}
	mean, stdDev := computeMeanStdDev(values)
	if stdDev == 0 {
		t.Fatal("expected non-zero std deviation")
	}
	outliers := 0
	for _, v := range values {
		if abs(v-mean) > 3*stdDev {
			outliers++
		}
	}
	if outliers < 1 {
		t.Fatal("expected at least one outlier detected")
	}
}

// ─── Anomaly Detection Tests ──────────────────────────────────────

func TestAnomalyRulesBootstrapped(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/anomalies/rules", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count == 0 {
		t.Fatal("expected at least one anomaly rule")
	}
}

func TestAnomalyDetectionRun(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("POST", "/api/analytics/anomalies/run", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ─── Alert Engine Tests ───────────────────────────────────────────

func TestAlertRulesBootstrapped(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/alerts/rules", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count == 0 {
		t.Fatal("expected at least one alert rule")
	}
}

func TestAlertRuleEvaluation(t *testing.T) {
	rule := AlertRule{ID: "test-rule", Enabled: true, Condition: "below", Threshold: 50}
	if !evaluateAlertRule(rule, 30, "test") {
		t.Fatal("expected alert to trigger for value below threshold")
	}
	if evaluateAlertRule(rule, 60, "test") {
		t.Fatal("expected no alert for value above threshold")
	}
}

func TestAlertCreation(t *testing.T) {
	rule := AlertRule{
		ID: "test-alert-rule", Name: "Test Alert", Metric: "test_metric",
		Condition: "below", Threshold: 50, Priority: "high", Enabled: true,
		NotifTitle: "Test Alert Title", NotifBody: "Test alert body",
	}
	alert := createAnalyticalAlert(rule, 30, "project", "PRJ-1", "project", "PRJ-1", nil)
	if alert.ID == "" {
		t.Fatal("expected alert ID")
	}
	if alert.Status != "open" {
		t.Fatalf("expected open status, got %s", alert.Status)
	}
	if len(alert.Actions) == 0 {
		t.Fatal("expected alert actions")
	}
}

// ─── Dashboard Tests ──────────────────────────────────────────────

func TestDashboardsBootstrapped(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/dashboards", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count      int                  `json:"count"`
		Dashboards []AnalyticsDashboard `json:"dashboards"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count < 4 {
		t.Fatalf("expected at least 4 dashboards, got %d", resp.Count)
	}
}

func TestExecutiveDashboardData(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/dashboards/dash_executive", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["kpis"]; !ok {
		t.Fatal("expected kpis in dashboard data")
	}
	if _, ok := resp["dashboard"]; !ok {
		t.Fatal("expected dashboard definition in response")
	}
}

// ─── Search Tests ─────────────────────────────────────────────────

func TestAnalyticsSearch(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/search?q=survey", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count   int                     `json:"count"`
		Results []AnalyticsSearchResult `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count == 0 {
		t.Fatal("expected at least one search result for 'survey'")
	}
}

// ─── Lineage Tests ────────────────────────────────────────────────

func TestKPILineage(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/lineage/kpi/kpi_active_projects", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp DataLineage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.SourceApp == "" {
		t.Fatal("expected source app in lineage")
	}
	if resp.ResponsibleSystem == "" {
		t.Fatal("expected responsible system in lineage")
	}
}

func TestLineageOverview(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/lineage", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count == 0 {
		t.Fatal("expected lineage overview entries")
	}
}

// ─── Export Tests ─────────────────────────────────────────────────

func TestExportCreation(t *testing.T) {
	ts := setupTestRouter()
	body := `{"dataset": "test", "format": "json", "requested_by": "test-user"}`
	w := ts.do("POST", "/api/analytics/exports", body)
	if w.Code != 202 {
		t.Fatalf("expected 202, got %d", w.Code)
	}
}

// ─── Cross-App Intelligence Tests ─────────────────────────────────

func TestCrossAppIntelligence(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/intelligence", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["synthesis"]; !ok {
		t.Fatal("expected synthesis in intelligence response")
	}
}

// ─── AI-Ready Tests ───────────────────────────────────────────────

func TestAIAnalyticsCatalog(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/analytics/ai/catalog", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["resources"]; !ok {
		t.Fatal("expected resources in AI catalog")
	}
}

// ─── Helper Tests ─────────────────────────────────────────────────

func TestGPSCoordinateParsing(t *testing.T) {
	lat, lng := parseGPSCoordinates("0.3136, 32.5811")
	if lat != 0.3136 || lng != 32.5811 {
		t.Fatalf("expected (0.3136, 32.5811), got (%f, %f)", lat, lng)
	}
}

func TestAbs(t *testing.T) {
	if abs(-5) != 5 {
		t.Fatal("expected abs(-5) = 5")
	}
	if abs(5) != 5 {
		t.Fatal("expected abs(5) = 5")
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
