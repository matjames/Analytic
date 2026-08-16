package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/matjames/statgate-lib/health"
	"github.com/matjames/statgate-lib/metrics"
	"statdata-backend/internal/api"
	"statdata-backend/internal/events"
	"statdata-backend/internal/pipeline"
	"statdata-backend/internal/science"
	"statdata-backend/internal/search"
	"statdata-backend/internal/store"
)

var (
	testMetricsOnce sync.Once
	testMetricsInst *metrics.Metrics
)

func getTestMetrics() *metrics.Metrics {
	testMetricsOnce.Do(func() {
		testMetricsInst = metrics.NewMetrics("statdata_test")
	})
	return testMetricsInst
}

func setupTestServer() *httptest.Server {
	memStore := store.NewMemStore()
	qualityEngine := pipeline.NewQualityEngine(memStore)
	lineageTracker := pipeline.NewLineageTracker(memStore)
	pipelineEngine := pipeline.NewPipelineEngine(memStore, qualityEngine, lineageTracker)

	notebookRunner := science.NewNotebookRunner(memStore)
	experimentTracker := science.NewExperimentTracker(memStore)
	modelRegistry := science.NewModelRegistry(memStore)
	clusterOrchestrator := science.NewClusterOrchestrator(memStore)

	indexer := search.NewUniversalIndexer(memStore)
	searchEngine := search.NewSearchEngine(memStore, indexer)

	eventWorker := events.NewEventWorker(nil, memStore, pipelineEngine, lineageTracker, indexer, "test-node")

	handlers := api.NewHandlers(
		memStore, pipelineEngine, qualityEngine, lineageTracker,
		notebookRunner, experimentTracker, modelRegistry, clusterOrchestrator,
		searchEngine, indexer, eventWorker,
	)

	promMetrics := getTestMetrics()
	healthChecker := health.NewChecker("statdata_test", nil)

	router := api.SetupRouter(api.RouterConfig{
		Handlers:      handlers,
		HealthChecker: healthChecker,
		Metrics:       promMetrics,
		AuthValidator: nil, // Open routes for testing
		CORSOrigin:    "*",
		Env:           "development",
	})

	return httptest.NewServer(router)
}

func TestHealthAndMetricsEndpoints(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// 1. /health
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /health, got %d", resp.StatusCode)
	}

	// 2. /ready
	resp, err = http.Get(ts.URL + "/ready")
	if err != nil {
		t.Fatalf("GET /ready failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /ready, got %d", resp.StatusCode)
	}

	// 3. /metrics
	resp, err = http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /metrics, got %d", resp.StatusCode)
	}
}

func TestCatalogAndSchemaValidationAPI(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// 1. GET /api/v1/data/catalog/datasets
	resp, err := http.Get(ts.URL + "/api/v1/data/catalog/datasets")
	if err != nil {
		t.Fatalf("GET /api/v1/data/catalog/datasets failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	// 2. POST /api/v1/data/schemas/validate (Valid payload)
	validateReq := map[string]interface{}{
		"subject": "national_census_demographics",
		"payload": map[string]interface{}{
			"district":   "Kigali Urban",
			"population": 1340000,
			"median_age": 22.4,
		},
	}
	body, _ := json.Marshal(validateReq)
	resp, err = http.Post(ts.URL+"/api/v1/data/schemas/validate", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("POST /api/v1/data/schemas/validate failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	var valResult map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&valResult)
	if valResult["valid"] != true {
		t.Errorf("Expected payload to be valid according to schema, got %v", valResult)
	}

	// 3. POST /api/v1/search/query (Search API)
	searchReq := map[string]interface{}{
		"query": "Census demographics population",
		"alpha": 0.5,
		"limit": 5,
	}
	searchBody, _ := json.Marshal(searchReq)
	resp, err = http.Post(ts.URL+"/api/v1/search/query", "application/json", bytes.NewBuffer(searchBody))
	if err != nil {
		t.Fatalf("POST /api/v1/search/query failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	var searchResult map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&searchResult)
	if hits, ok := searchResult["total_hits"].(float64); !ok || hits <= 0 {
		t.Errorf("Expected positive total_hits in search response, got %v", searchResult)
	}
}

func TestObjectLinksAndCrossAppHooks(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// POST /api/v1/data/links
	linkReq := map[string]interface{}{
		"source_type":   "DATASET",
		"source_id":     "ds-census-2026",
		"target_type":   "PMS_PROJECT",
		"target_id":     "pms-proj-national-census",
		"relation_type": "INPUT_DATASET",
		"tenant_id":     "default",
	}
	body, _ := json.Marshal(linkReq)
	resp, err := http.Post(ts.URL+"/api/v1/data/links", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("POST /api/v1/data/links failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected 201 Created, got %d", resp.StatusCode)
	}

	// GET /api/v1/data/links?source_type=DATASET&source_id=ds-census-2026
	resp, err = http.Get(ts.URL + "/api/v1/data/links?source_type=DATASET&source_id=ds-census-2026")
	if err != nil {
		t.Fatalf("GET /api/v1/data/links failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}
}
