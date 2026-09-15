package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	globalStore = NewMemStore()

	r := gin.New()
	r.GET("/health", HealthHandler)
	r.GET("/ready", ReadyHandler)
	r.GET("/readyz", ReadyHandler)

	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		c.Set("tenant_id", "tenant-alpha")
		c.Set("user_id", "sre-test-admin")
		c.Next()
	})
	{
		v1.GET("/summary", SummaryHandler)
		obs := v1.Group("/observability")
		{
			obs.GET("/telemetry", ListTelemetryHandler)
			obs.POST("/telemetry/scrape", ScrapeTelemetryHandler)
			obs.GET("/cmdb", ListCMDBHandler)
			obs.GET("/alerts", ListAlertsHandler)
			obs.POST("/alerts", CreateAlertHandler)
			obs.GET("/status-page", ListStatusPageHandler)
		}
		devops := v1.Group("/devops")
		{
			devops.GET("/pipelines", ListPipelinesHandler)
			devops.POST("/pipelines/trigger", TriggerPipelineHandler)
			devops.GET("/deployments", ListDeploymentsHandler)
			devops.POST("/deployments", CreateDeploymentHandler)
			devops.POST("/deployments/:id/rollback", RollbackDeploymentHandler)
		}
		cloud := v1.Group("/cloud")
		{
			cloud.GET("/clusters", ListClustersHandler)
			cloud.GET("/costs", ListCostsHandler)
		}
		sre := v1.Group("/sre")
		{
			sre.GET("/readiness", ListReadinessHandler)
			sre.GET("/slos", ListSLOsHandler)
			sre.POST("/runbooks/execute", ExecuteRunbookHandler)
			sre.POST("/chaos/simulate", SimulateChaosDrillHandler)
		}
	}
	return r
}

func TestHealthEndpoint(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestWorkspaceHeaderAcceptsPlatformWorkspaceIDs(t *testing.T) {
	if !validWorkspaceID("ws-1788255280007546969") {
		t.Fatal("expected platform workspace ID to pass validation")
	}
	if validWorkspaceID("workspace with spaces") {
		t.Fatal("expected whitespace workspace ID to fail validation")
	}
}

func TestTelemetryAggregation(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("GET", "/api/v1/observability/telemetry", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var res struct {
		Count     int                `json:"count"`
		Telemetry []ServiceTelemetry `json:"telemetry"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)

	if res.Count < 5 {
		t.Fatalf("Expected at least 5 service telemetry records, got %d", res.Count)
	}
}

func TestPipelineTriggerAndList(t *testing.T) {
	r := setupTestRouter()

	pipeReq := map[string]interface{}{
		"repo_name":  "statgate/rms",
		"branch":     "feat/doi-crossref",
		"commit_sha": "a1b2c3d",
		"commit_msg": "feat: add CrossRef XML exporter",
		"author":     "alice.sre",
	}
	b, _ := json.Marshal(pipeReq)
	req, _ := http.NewRequest("POST", "/api/v1/devops/pipelines/trigger", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected 202 Accepted for pipeline trigger, got %d", w.Code)
	}

	var created PipelineRun
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if created.Status != "RUNNING" {
		t.Fatalf("Expected RUNNING status, got %s", created.Status)
	}
}

func TestDeploymentAndRollback(t *testing.T) {
	r := setupTestRouter()

	// 1. Create deployment
	depReq := map[string]interface{}{
		"service_name":   "PMS",
		"version":        "v2.1.0-rc1",
		"environment":    "PRODUCTION_SOVEREIGN",
		"strategy":       "CANARY",
		"target_cluster": "cls-gov-kampala-01",
		"deployed_by":    "deploy-bot",
	}
	b, _ := json.Marshal(depReq)
	req, _ := http.NewRequest("POST", "/api/v1/devops/deployments", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for deployment, got %d", w.Code)
	}

	var dep DeploymentRecord
	_ = json.Unmarshal(w.Body.Bytes(), &dep)

	// 2. Rollback deployment
	rollReq := map[string]interface{}{
		"target_version": "v2.0.8-stable",
	}
	rb, _ := json.Marshal(rollReq)
	rreq, _ := http.NewRequest("POST", "/api/v1/devops/deployments/"+dep.ID+"/rollback", bytes.NewBuffer(rb))
	rreq.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, rreq)

	if rw.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for rollback, got %d", rw.Code)
	}
}

func TestSovereignClustersAndCost(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/cloud/clusters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for clusters, got %d", w.Code)
	}

	req2, _ := http.NewRequest("GET", "/api/v1/cloud/costs", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected 200 for costs, got %d", w2.Code)
	}
}

func TestSREReadinessAndChaosDrill(t *testing.T) {
	r := setupTestRouter()

	// 1. Verify readiness checks
	req, _ := http.NewRequest("GET", "/api/v1/sre/readiness", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for readiness checks, got %d", w.Code)
	}

	// 2. Execute automated runbook
	runReq := map[string]interface{}{
		"runbook_name":   "Auto-Remediate Redis Memory Fragmentation",
		"target_service": "Redis Event Mesh",
		"trigger_reason": "Memory fragmentation ratio > 1.8",
	}
	rb, _ := json.Marshal(runReq)
	req2, _ := http.NewRequest("POST", "/api/v1/sre/runbooks/execute", bytes.NewBuffer(rb))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("Expected 200 for runbook execution, got %d", w2.Code)
	}

	// 3. Simulate Chaos drill
	chaosReq := map[string]interface{}{
		"scenario":       "REGION_FAILOVER",
		"target_service": "StatTrust Gateway",
	}
	cb, _ := json.Marshal(chaosReq)
	req3, _ := http.NewRequest("POST", "/api/v1/sre/chaos/simulate", bytes.NewBuffer(cb))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("Expected 200 for chaos simulation, got %d", w3.Code)
	}
}

func TestCMDBAndAlertCenter(t *testing.T) {
	r := setupTestRouter()

	// 1. CMDB List
	req, _ := http.NewRequest("GET", "/api/v1/observability/cmdb", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for CMDB, got %d", w.Code)
	}

	// 2. Alert creation
	alertReq := map[string]interface{}{
		"title":       "Database CPU Utilization > 85% on Primary Node",
		"severity":    "P2_HIGH",
		"source_app":  "PostgreSQL Sovereign Cluster",
		"description": "Auto-scaling replica provisioned in Entebbe DR.",
	}
	b, _ := json.Marshal(alertReq)
	req2, _ := http.NewRequest("POST", "/api/v1/observability/alerts", bytes.NewBuffer(b))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for alert, got %d", w2.Code)
	}
}
