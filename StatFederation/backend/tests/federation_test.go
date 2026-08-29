package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/matjames/statgate-lib/events"
	"github.com/matjames/statgate-lib/health"
	"github.com/matjames/statgate-lib/metrics"

	"statfederation-backend/internal/api"
	"statfederation-backend/internal/diplomacy"
	"statfederation-backend/internal/engine"
	fedevents "statfederation-backend/internal/events"
	"statfederation-backend/internal/models"
	"statfederation-backend/internal/store"
)

var (
	testMetricsOnce sync.Once
	testMetricsInst *metrics.Metrics
)

func getTestMetrics() *metrics.Metrics {
	testMetricsOnce.Do(func() {
		testMetricsInst = metrics.NewMetrics("statfederation_test")
	})
	return testMetricsInst
}

func setupTestApp() (*api.Handlers, http.Handler, store.Store) {
	memStore := store.NewMemStore()
	fedEngine := engine.NewFederationEngine(memStore)
	queryRouter := engine.NewQueryRouter(memStore, fedEngine)
	harmonizer := engine.NewHarmonizer(memStore)
	diplomacyGW := diplomacy.NewDiplomacyGateway(memStore)
	compliance := diplomacy.NewComplianceEvaluator(memStore)
	fedSearch := diplomacy.NewFederatedSearchHub(memStore)
	pushProtocol := engine.NewPushProtocol(memStore)

	eventBus, _ := events.NewEventBus(events.Config{
		Source:  "statfederation-test",
		Channel: events.DefaultChannel,
	})
	eventWorker := fedevents.NewEventWorker(eventBus, memStore, "NODE-TEST-HQ")

	handlers := api.NewHandlers(
		memStore, fedEngine, queryRouter, harmonizer,
		diplomacyGW, compliance, fedSearch, eventWorker, pushProtocol,
	)

	promMetrics := getTestMetrics()
	healthChecker := health.NewChecker("statfederation_test", nil)

	router := api.SetupRouter(api.RouterConfig{
		Handlers:      handlers,
		HealthChecker: healthChecker,
		Metrics:       promMetrics,
		AuthValidator: nil, // Open in test mode
		CORSOrigin:    "*",
		Env:           "test",
	})

	return handlers, router, memStore
}

// 1. Test NSS Node Registry & Probing
func TestNSSNodeRegistryAndProbe(t *testing.T) {
	_, router, s := setupTestApp()
	ctx := context.Background()

	// Verify pre-seeded nodes
	nodes, err := s.ListNodes(ctx, "default", "", "")
	if err != nil || len(nodes) < 3 {
		t.Fatalf("Expected at least 3 seeded nodes, got %d (err: %v)", len(nodes), err)
	}

	// Register new District Node via HTTP
	newNode := models.FederatedNode{
		Name:         "Gulu District Statistical Directorate",
		Code:         "NSS-DIST-GULU",
		NodeType:     models.NodeTypeDistrict,
		Jurisdiction: "NATIONAL",
		EndpointURL:  "https://gulu.dist.gov.statgate/api",
		TenantID:     "tenant-alpha",
	}
	body, _ := json.Marshal(newNode)

	req, _ := http.NewRequest("POST", "/api/v1/federation/nodes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on node registration, got %d: %s", w.Code, w.Body.String())
	}

	// Probe Nodes Endpoint
	reqProbe, _ := http.NewRequest("POST", "/api/v1/federation/nodes/probe", nil)
	wProbe := httptest.NewRecorder()
	router.ServeHTTP(wProbe, reqProbe)

	if wProbe.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on node probe, got %d: %s", wProbe.Code, wProbe.Body.String())
	}
}

func TestWorkspaceScopedFederationNodes(t *testing.T) {
	_, router, _ := setupTestApp()

	invalidReq, _ := http.NewRequest("GET", "/api/v1/federation/nodes", nil)
	invalidReq.Header.Set("X-Workspace-ID", "bad-workspace-id")
	invalid := httptest.NewRecorder()
	router.ServeHTTP(invalid, invalidReq)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("Expected invalid workspace header to return 400, got %d", invalid.Code)
	}

	node := models.FederatedNode{
		Name:         "Workspace Federation Node",
		Code:         "NSS-WORKSPACE-1",
		NodeType:     models.NodeTypeDistrict,
		Jurisdiction: "NATIONAL",
		EndpointURL:  "https://workspace.dist.gov.statgate/api",
	}
	body, _ := json.Marshal(node)
	registerReq, _ := http.NewRequest("POST", "/api/v1/federation/nodes", bytes.NewBuffer(body))
	registerReq.Header.Set("Content-Type", "application/json")
	registerReq.Header.Set("X-Workspace-ID", "workspace_one")
	registered := httptest.NewRecorder()
	router.ServeHTTP(registered, registerReq)
	if registered.Code != http.StatusCreated {
		t.Fatalf("Expected workspace node registration to return 201, got %d: %s", registered.Code, registered.Body.String())
	}

	for _, workspace := range []string{"workspace_one", "workspace_two"} {
		listReq, _ := http.NewRequest("GET", "/api/v1/federation/nodes", nil)
		listReq.Header.Set("X-Workspace-ID", workspace)
		listed := httptest.NewRecorder()
		router.ServeHTTP(listed, listReq)
		if listed.Code != http.StatusOK {
			t.Fatalf("Expected workspace node list to return 200, got %d", listed.Code)
		}
		var response struct {
			Nodes []models.FederatedNode `json:"nodes"`
		}
		if err := json.Unmarshal(listed.Body.Bytes(), &response); err != nil {
			t.Fatalf("Decode workspace node list: %v", err)
		}
		if workspace == "workspace_one" && len(response.Nodes) != 1 {
			t.Fatalf("Expected one node in workspace_one, got %d", len(response.Nodes))
		}
		if workspace == "workspace_two" && len(response.Nodes) != 0 {
			t.Fatalf("Expected no nodes in workspace_two, got %d", len(response.Nodes))
		}
	}
}

// 2. Test Data Sharing Agreement (DSA) Lifecycle & Validation
func TestDataSharingAgreementLifecycle(t *testing.T) {
	_, router, s := setupTestApp()
	ctx := context.Background()

	// Create new draft DSA
	dsa := models.DataSharingAgreement{
		Title:                 "Agricultural Census Microdata Exchange Agreement",
		ProviderNodeID:        "node-moh-002",
		ConsumerNodeID:        "node-nss-001",
		AccessTier:            "INTER_AGENCY",
		PermittedDomains:      []string{"agriculture", "food_security"},
		ClassificationAllowed: "CONFIDENTIAL",
		Purpose:               "National Crop Forecasting Synthesis",
		ValidFrom:             time.Now().UTC(),
		ValidUntil:            time.Now().UTC().Add(180 * 24 * time.Hour),
		RateLimitPerMin:       60,
		DailyQuota:            5000,
		TenantID:              "default",
	}

	body, _ := json.Marshal(dsa)
	req, _ := http.NewRequest("POST", "/api/v1/federation/dsa", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on DSA creation, got %d", w.Code)
	}

	var resp struct {
		DSA models.DataSharingAgreement `json:"dsa"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	dsaID := resp.DSA.ID

	// Approve DSA
	reqApprove, _ := http.NewRequest("POST", "/api/v1/federation/dsa/"+dsaID+"/approve", nil)
	wApprove := httptest.NewRecorder()
	router.ServeHTTP(wApprove, reqApprove)

	if wApprove.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on DSA approval, got %d", wApprove.Code)
	}

	// Verify active status in store
	approvedDSA, err := s.GetDSAByID(ctx, dsaID)
	if err != nil || approvedDSA.Status != models.DSAStatusActive {
		t.Fatalf("Expected DSA status ACTIVE, got %v (err: %v)", approvedDSA.Status, err)
	}
}

// 3. Test Distributed Query Router Scatter-Gather Execution
func TestDistributedQueryRouter(t *testing.T) {
	_, router, s := setupTestApp()
	ctx := context.Background()

	// Dispatch query to pre-seeded nodes
	queryReq := models.DistributedQueryRequest{
		QueryName:    "National Health Surveillance Aggregate Q2",
		TargetDomain: "health",
		FilterCriteria: map[string]interface{}{
			"year":   2026,
			"period": "Q2",
		},
		Aggregations:   []string{"SUM(cases)", "AVG(positivity_rate)"},
		TimeoutSeconds: 5,
	}

	body, _ := json.Marshal(queryReq)
	req, _ := http.NewRequest("POST", "/api/v1/federation/queries/dispatch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on query dispatch, got %d: %s", w.Code, w.Body.String())
	}

	var qRes engine.AggregatedQueryResult
	if err := json.Unmarshal(w.Body.Bytes(), &qRes); err != nil {
		t.Fatalf("Failed to parse aggregated query response: %v", err)
	}

	if qRes.TotalNodesTargeted == 0 || qRes.SuccessfulNodes == 0 {
		t.Fatalf("Expected successful node execution, got targeted: %d, success: %d", qRes.TotalNodesTargeted, qRes.SuccessfulNodes)
	}

	if len(qRes.HarmonizedRecords) == 0 {
		t.Fatalf("Expected harmonized data records in result, got 0")
	}

	// Verify query record logged
	qRec, err := s.GetQueryByID(ctx, qRes.QueryID)
	if err != nil || qRec.TotalRecordsRetrieved == 0 {
		t.Fatalf("Expected recorded query in ledger, got: %v (err: %v)", qRec, err)
	}

	// Verify object_links cross-module entry created
	links, err := s.GetObjectLinks(ctx, "distributed_query", qRes.QueryID, "default")
	if err != nil || len(links) == 0 {
		t.Fatalf("Expected object_link created for distributed query, got %d", len(links))
	}
}

// 4. Test Metadata Harmonization Engine (SDMX / DDI)
func TestMetadataHarmonizer(t *testing.T) {
	_, router, _ := setupTestApp()

	rawAgencyData := map[string]interface{}{
		"district":    "Gulu",
		"sex":         "FEMALE",
		"age":         "15-49",
		"mortality":   18.4,
		"facility_id": "FAC-GULU-001",
		"year":        2026,
	}

	payload := map[string]interface{}{
		"source_agency":      "MOH",
		"standard_framework": "SDMX_2.1",
		"raw_record":         rawAgencyData,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/federation/metadata/harmonize", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on metadata harmonization, got %d: %s", w.Code, w.Body.String())
	}

	var res engine.HarmonizationResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("Failed to parse harmonization response: %v", err)
	}

	if res.HarmonizedRecord["DIM_SEX"] != "FEMALE" {
		t.Errorf("Expected 'sex' mapped to 'DIM_SEX', got %v", res.HarmonizedRecord["DIM_SEX"])
	}
	if res.HarmonizedRecord["DIM_GEO_ADMIN2"] != "Gulu" {
		t.Errorf("Expected 'district' mapped to 'DIM_GEO_ADMIN2', got %v", res.HarmonizedRecord["DIM_GEO_ADMIN2"])
	}
	if res.HarmonizedRecord["DIM_TIME_PERIOD"] != float64(2026) {
		t.Errorf("Expected 'year' mapped to 'DIM_TIME_PERIOD', got %v", res.HarmonizedRecord["DIM_TIME_PERIOD"])
	}
}

// 5. Test Global Diplomacy Gateway & Multilateral Submissions (SDG / AU)
func TestDiplomacyGatewayReports(t *testing.T) {
	_, router, s := setupTestApp()
	ctx := context.Background()

	// Generate SDG submission
	reqSDG, _ := http.NewRequest("POST", "/api/v1/diplomacy/reports/sdg", bytes.NewBufferString(`{"period":"2026-ANNUAL"}`))
	reqSDG.Header.Set("Content-Type", "application/json")
	wSDG := httptest.NewRecorder()
	router.ServeHTTP(wSDG, reqSDG)

	if wSDG.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on SDG report, got %d: %s", wSDG.Code, wSDG.Body.String())
	}

	var sdgResp struct {
		Report models.InternationalReport `json:"report"`
	}
	_ = json.Unmarshal(wSDG.Body.Bytes(), &sdgResp)

	if sdgResp.Report.DestinationBody != "UN_DESA" || sdgResp.Report.SubmissionHash == "" {
		t.Errorf("Invalid SDG submission report: %+v", sdgResp.Report)
	}

	// Verify stored report
	savedReport, err := s.GetReportByID(ctx, sdgResp.Report.ID)
	if err != nil || savedReport.Status != "TRANSMITTED" {
		t.Errorf("Expected report status TRANSMITTED, got %v (err: %v)", savedReport, err)
	}

	// Generate AU Agenda 2063 Report
	reqAU, _ := http.NewRequest("POST", "/api/v1/diplomacy/reports/au", bytes.NewBufferString(`{"period":"2026-Q2"}`))
	reqAU.Header.Set("Content-Type", "application/json")
	wAU := httptest.NewRecorder()
	router.ServeHTTP(wAU, reqAU)

	if wAU.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on AU report, got %d: %s", wAU.Code, wAU.Body.String())
	}
}

// 6. Test Transboundary Policy Compliance & PII Anonymization
func TestTransboundaryPolicyCompliance(t *testing.T) {
	_, router, s := setupTestApp()
	ctx := context.Background()

	// Scenario A: Sovereignty Violation (Raw Microdata to Global)
	reqDenied := map[string]interface{}{
		"source_jurisdiction": "NATIONAL",
		"target_jurisdiction": "GLOBAL",
		"resource_type":       "RAW_MICRODATA",
		"resource_id":         "census_microdata_sample_2026",
		"payload": map[string]interface{}{
			"sample_id": "REC-9918",
			"income":    1500000,
		},
	}
	bodyDenied, _ := json.Marshal(reqDenied)
	req1, _ := http.NewRequest("POST", "/api/v1/diplomacy/compliance/evaluate", bytes.NewBuffer(bodyDenied))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	var resDenied diplomacy.ComplianceEvaluationResult
	_ = json.Unmarshal(w1.Body.Bytes(), &resDenied)
	if resDenied.Decision != diplomacy.DecisionDenied {
		t.Errorf("Expected DENIED decision for raw microdata egress, got %s", resDenied.Decision)
	}

	// Scenario B: Allowed with Automated PII Masking
	reqMasked := map[string]interface{}{
		"source_jurisdiction": "NATIONAL",
		"target_jurisdiction": "EAST_AFRICA",
		"resource_type":       "AGGREGATE_TABLE",
		"resource_id":         "district_health_summary",
		"payload": map[string]interface{}{
			"respondent_name": "Jane Doe",
			"national_id":     "CM991029384",
			"district":        "Kampala",
			"total_visits":    42,
		},
	}
	bodyMasked, _ := json.Marshal(reqMasked)
	req2, _ := http.NewRequest("POST", "/api/v1/diplomacy/compliance/evaluate", bytes.NewBuffer(bodyMasked))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	var resMasked diplomacy.ComplianceEvaluationResult
	_ = json.Unmarshal(w2.Body.Bytes(), &resMasked)
	if resMasked.Decision != diplomacy.DecisionRedactedAndAllowed {
		t.Errorf("Expected REDACTED_AND_ALLOWED, got %s", resMasked.Decision)
	}
	if resMasked.SanitizedPayload["national_id"] != "[REDACTED_SOVEREIGN_PII]" {
		t.Errorf("Expected PII redaction on national_id, got %v", resMasked.SanitizedPayload["national_id"])
	}

	// Verify compliance audit logs logged
	logs, err := s.ListComplianceLogs(ctx, "default", 10)
	if err != nil || len(logs) < 2 {
		t.Errorf("Expected at least 2 audit logs, got %d (err: %v)", len(logs), err)
	}
}

// 7. Test Federated Search Hub
func TestFederatedSearchHub(t *testing.T) {
	_, router, _ := setupTestApp()

	searchReq := diplomacy.SearchQuery{
		Keyword: "AfCFTA",
	}
	body, _ := json.Marshal(searchReq)
	req, _ := http.NewRequest("POST", "/api/v1/diplomacy/search", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on federated search, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Results []*models.FederatedSearchResult `json:"results"`
		Count   int                             `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Count == 0 {
		t.Errorf("Expected search results for AfCFTA keyword, got 0")
	}
}

// 8. Test System Endpoints: Health, Ready, Live, Metrics
func TestSystemEndpoints(t *testing.T) {
	_, router, _ := setupTestApp()

	// /health
	reqH, _ := http.NewRequest("GET", "/health", nil)
	wH := httptest.NewRecorder()
	router.ServeHTTP(wH, reqH)
	if wH.Code != http.StatusOK {
		t.Errorf("Expected 200 on /health, got %d", wH.Code)
	}

	// /ready
	reqR, _ := http.NewRequest("GET", "/ready", nil)
	wR := httptest.NewRecorder()
	router.ServeHTTP(wR, reqR)
	if wR.Code != http.StatusOK {
		t.Errorf("Expected 200 on /ready, got %d", wR.Code)
	}

	// /metrics
	reqM, _ := http.NewRequest("GET", "/metrics", nil)
	wM := httptest.NewRecorder()
	router.ServeHTTP(wM, reqM)
	if wM.Code != http.StatusOK {
		t.Errorf("Expected 200 on /metrics, got %d", wM.Code)
	}
}

// 9. Test Indicator Push & Ingest Protocol
func TestIndicatorPushAndIngestProtocol(t *testing.T) {
	_, router, s := setupTestApp()
	ctx := context.Background()

	// 1. Ingest an SDMX dataset
	sdmxPayload := map[string]interface{}{
		"header": map[string]interface{}{
			"id":        "push-test-001",
			"test":      true,
			"prepared":  time.Now().UTC().Format(time.RFC3339),
			"dataSetID": "DS-SDG-2026",
			"sender": map[string]interface{}{
				"id":   "node-moh-002",
				"name": "Ministry of Health",
			},
		},
		"dataSets": []map[string]interface{}{
			{
				"action": "Replace",
				"observations": map[string]interface{}{
					"IND-HEALTH-IMMUNIZATION": map[string]interface{}{
						"code":          "IND-HEALTH-IMMUNIZATION",
						"title":         "DTP3 Immunization Coverage Rate",
						"domain":        "health",
						"value":         89.4,
						"unit":          "Percentage",
						"freq":          "ANNUAL",
						"sdmxDimension": "HEALTH_IMMUN_DTP3",
					},
				},
			},
		},
	}
	body, _ := json.Marshal(sdmxPayload)
	req, _ := http.NewRequest("POST", "/api/v1/federation/indicators/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Source-Node", "node-moh-002")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on indicator ingest, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the ingested indicator is now in the store
	ind, err := s.GetIndicatorByCode(ctx, "IND-HEALTH-IMMUNIZATION")
	if err != nil || ind == nil {
		t.Fatalf("Expected ingested indicator in store, got error: %v", err)
	}
	if ind.CurrentValue == nil || *ind.CurrentValue != 89.4 {
		t.Errorf("Expected indicator value 89.4, got %v", ind.CurrentValue)
	}
}
