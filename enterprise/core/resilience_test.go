package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupResilienceTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Set("user_id", "test_admin")
		c.Set("tenant_id", "statgate")
		c.Next()
	})

	initResilienceEngine()
	initDrillEngine()
	initIncidentDetector()
	initBackupAssurance()
	initIntegrityEngine()
	initRunbooks()
	initEvidenceVault()

	api := r.Group("/api")
	{
		resilience := api.Group("/resilience")
		{
			resilience.GET("/overview", handleResilienceOverview)
			resilience.GET("/services", handleListResilienceServices)
			resilience.GET("/services/:id", handleGetResilienceService)
			resilience.PUT("/services/:id", handleUpdateResilienceService)
			resilience.GET("/score", handleResilienceScore)
		}

		recovery := api.Group("/recovery")
		{
			recovery.GET("/drills", handleListDrills)
			recovery.POST("/drills", handleCreateDrill)
			recovery.GET("/drills/:id", handleGetDrill)
			recovery.POST("/drills/:id/start", handleStartDrill)
		}

		incidents := api.Group("/incidents")
		{
			incidents.GET("", handleListIncidents)
			incidents.POST("", handleCreateIncident)
			incidents.GET("/:id", handleGetIncident)
			incidents.POST("/:id/acknowledge", handleAcknowledgeIncident)
			incidents.POST("/:id/mitigate", handleMitigateIncident)
			incidents.POST("/:id/resolve", handleResolveIncident)
			incidents.POST("/:id/close", handleCloseIncident)
			incidents.POST("/:id/actions", handleAddIncidentAction)
		}

		backups := api.Group("/backups")
		{
			backups.GET("", handleListBackups)
			backups.POST("", handleCreateBackupRecord)
			backups.GET("/:id", handleGetBackup)
			backups.POST("/:id/verify", handleVerifyBackup)
			backups.POST("/:id/restore-test", handleRunRestoreTest)
		}

		integrity := api.Group("/integrity")
		{
			integrity.GET("", handleGetIntegrity)
			integrity.POST("/run", handleRunIntegrity)
			integrity.GET("/results", handleGetIntegrityResults)
		}

		runbooks := api.Group("/runbooks")
		{
			runbooks.GET("", handleListRunbooks)
			runbooks.GET("/:id", handleGetRunbook)
			runbooks.POST("/:id/execute", handleExecuteRunbook)
		}

		evidence := api.Group("/evidence")
		{
			evidence.GET("", handleListEvidence)
			evidence.GET("/:id", handleGetEvidence)
		}
	}
	return r
}

// ─── 1. Resilience Overview & Score Tests ─────────────────────────

func TestResilienceOverviewScoreCalculation(t *testing.T) {
	r := setupResilienceTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/resilience/overview", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var overview ResilienceScoreOverview
	if err := json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
		t.Fatalf("failed to decode overview: %v", err)
	}

	if overview.CompositeScore < 80 || overview.CompositeScore > 100 {
		t.Errorf("expected composite score between 80 and 100, got %d", overview.CompositeScore)
	}
	if overview.TotalServices != 8 {
		t.Errorf("expected 8 total governed services, got %d", overview.TotalServices)
	}
	if overview.AvailabilityScore <= 0 {
		t.Errorf("expected positive availability score, got %d", overview.AvailabilityScore)
	}
}

func TestResilienceServicesListAndGet(t *testing.T) {
	r := setupResilienceTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/resilience/services", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Test Get Enterprise Core profile
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/resilience/services/enterprise-core", nil)
	r.ServeHTTP(w2, req2)

	if w2.Code != 200 {
		t.Fatalf("expected status 200 for enterprise-core, got %d", w2.Code)
	}

	var profile ServiceResilienceProfile
	_ = json.Unmarshal(w2.Body.Bytes(), &profile)
	if profile.CriticalityTier != Tier0_MissionCritical {
		t.Errorf("expected Tier 0 for enterprise-core, got %s", profile.CriticalityTier)
	}
	if profile.RTOTargetSec != 60 {
		t.Errorf("expected RTO target 60s, got %d", profile.RTOTargetSec)
	}
}

// ─── 2. Recovery Drill Tests ──────────────────────────────────────

func TestRecoveryDrillsExecutionAndRTO(t *testing.T) {
	r := setupResilienceTestRouter()

	// 1. List drills
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/recovery/drills", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. Start Drill 1
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/recovery/drills/drill_std_001/start", nil)
	r.ServeHTTP(w2, req2)

	if w2.Code != 200 {
		t.Fatalf("expected 200 for drill start, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["status"] != "completed" {
		t.Errorf("expected status=completed, got %v", resp["status"])
	}
	drillObj := resp["drill"].(map[string]interface{})
	if drillObj["drill_result"] != "SUCCESS" {
		t.Errorf("expected drill_result=SUCCESS, got %v", drillObj["drill_result"])
	}
}

// ─── 3. Incident State Machine Tests ──────────────────────────────

func TestIncidentStateMachineTransitions(t *testing.T) {
	r := setupResilienceTestRouter()

	// 1. Create Incident
	newInc := Incident{
		Title:       "Test Database Socket Latency",
		Severity:    SeverityCritical,
		ServiceID:   "registry",
		ServiceName: "Registry",
	}
	body, _ := json.Marshal(newInc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/incidents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201 created, got %d", w.Code)
	}

	var created Incident
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	incID := created.ID

	// 2. Acknowledge
	wAck := httptest.NewRecorder()
	reqAck, _ := http.NewRequest("POST", "/api/incidents/"+incID+"/acknowledge", nil)
	r.ServeHTTP(wAck, reqAck)
	if wAck.Code != 200 {
		t.Errorf("expected 200 for ack, got %d", wAck.Code)
	}

	// 3. Mitigate
	wMit := httptest.NewRecorder()
	reqMit, _ := http.NewRequest("POST", "/api/incidents/"+incID+"/mitigate", nil)
	r.ServeHTTP(wMit, reqMit)
	if wMit.Code != 200 {
		t.Errorf("expected 200 for mitigate, got %d", wMit.Code)
	}

	// 4. Resolve
	wRes := httptest.NewRecorder()
	reqRes, _ := http.NewRequest("POST", "/api/incidents/"+incID+"/resolve", bytes.NewBufferString(`{"reason":"Socket recycled"}`))
	reqRes.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wRes, reqRes)
	if wRes.Code != 200 {
		t.Errorf("expected 200 for resolve, got %d", wRes.Code)
	}

	// 5. Close
	wClose := httptest.NewRecorder()
	reqClose, _ := http.NewRequest("POST", "/api/incidents/"+incID+"/close", nil)
	r.ServeHTTP(wClose, reqClose)
	if wClose.Code != 200 {
		t.Errorf("expected 200 for close, got %d", wClose.Code)
	}
}

// ─── 4. Backup Assurance Tests ────────────────────────────────────

func TestBackupAssuranceVerifyAndRestoreTest(t *testing.T) {
	r := setupResilienceTestRouter()

	// 1. Verify backup
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/backups/bk_core_snap_001/verify", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. Sandbox Restore test
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/backups/bk_core_snap_001/restore-test", nil)
	r.ServeHTTP(w2, req2)

	if w2.Code != 200 {
		t.Fatalf("expected 200 for restore-test, got %d", w2.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["status"] != "restore_tested" {
		t.Errorf("expected status=restore_tested, got %v", resp["status"])
	}
}

// ─── 5. Data Integrity Engine Tests ───────────────────────────────

func TestIntegrityEngineSuiteAndScoring(t *testing.T) {
	r := setupResilienceTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/integrity/run", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	score := int(resp["integrity_score"].(float64))
	if score < 90 || score > 100 {
		t.Errorf("expected integrity score >= 90, got %d", score)
	}
}

// ─── 6. Runbooks Execution Tests ──────────────────────────────────

func TestRunbooksCatalogAndExecution(t *testing.T) {
	r := setupResilienceTestRouter()

	// 1. List
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/runbooks", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. Execute Redis Runbook
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/runbooks/rb_redis_recovery/execute", bytes.NewBufferString(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["status"] != "completed" {
		t.Errorf("expected status=completed, got %v", resp["status"])
	}
	if resp["evidence_id"] == "" || resp["evidence_id"] == nil {
		t.Error("expected non-empty evidence_id token")
	}
}

// ─── 7. Evidence Vault Cryptographic Hash Tests ───────────────────

func TestResilienceEvidenceCryptographicHash(t *testing.T) {
	r := setupResilienceTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/evidence/evi_drill_001", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var ev ResilienceEvidence
	_ = json.Unmarshal(w.Body.Bytes(), &ev)
	if len(ev.VerificationHash) != 64 {
		t.Errorf("expected 64-char SHA256 verification hash, got %d chars", len(ev.VerificationHash))
	}
	if ev.AuditReference == "" {
		t.Error("expected non-empty audit reference")
	}
}

// ─── 8. Failure Injection Simulation Tests ────────────────────────

func TestFailureInjectionSimulation(t *testing.T) {
	// 1. Simulate Redis broker drop
	testDrill := &RecoveryDrill{
		ID:           "test_drill_inj_001",
		Title:        "Simulated Redis Failure Injection",
		ServiceID:    "enterprise-core",
		Scenario:     "REDIS_FAILURE",
		Mode:         DrillModeSimulation,
		RTOTargetSec: 300,
		RPOTargetSec: 60,
		Steps: []RecoveryDrillStep{
			{ID: "s1", DrillID: "test_drill_inj_001", StepNumber: 1, Name: "Simulate Redis Outage", ActionType: "SIMULATE"},
			{ID: "s2", DrillID: "test_drill_inj_001", StepNumber: 2, Name: "Verify Ingress Persistence", ActionType: "VERIFY_EVENT_BUS"},
			{ID: "s3", DrillID: "test_drill_inj_001", StepNumber: 3, Name: "Validate Readiness Probe", ActionType: "PROBE_HEALTH"},
		},
	}

	err := executeRecoveryDrill(testDrill, "Test Operator")
	if err != nil {
		t.Fatalf("executeRecoveryDrill failed: %v", err)
	}

	if testDrill.DrillResult != DrillResultSuccess {
		t.Errorf("expected DrillResultSuccess, got %s", testDrill.DrillResult)
	}
	if testDrill.EvidenceID == "" {
		t.Error("expected evidence ID generated for simulated injection")
	}
}
