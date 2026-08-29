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
	// Middleware injecting default tenant
	v1.Use(func(c *gin.Context) {
		c.Set("tenant_id", "tenant-alpha")
		c.Set("user_id", "secops-admin")
		c.Next()
	})
	{
		v1.GET("/summary", SummaryHandler)
		secops := v1.Group("/secops")
		{
			secops.GET("/incidents", ListIncidentsHandler)
			secops.POST("/incidents", CreateIncidentHandler)
			secops.PUT("/incidents/:id/status", UpdateIncidentStatusHandler)
			secops.POST("/dlp/scan", DLPScanHandler)
			secops.GET("/privacy/consents", ListConsentsHandler)
			secops.GET("/keys", ListEncryptionKeysHandler)
			secops.POST("/keys/rotate", RotateEncryptionKeyHandler)
			secops.GET("/keys/:id", GetEncryptionKeyHandler)
		}
		trust := v1.Group("/trust")
		{
			trust.GET("/ledger/records", ListLedgerHandler)
			trust.POST("/ledger/append", AppendLedgerHandler)
			trust.GET("/credentials", ListCredentialsHandler)
			trust.POST("/credentials/issue", IssueCredentialHandler)
			trust.POST("/credentials/verify", VerifyCredentialHandler)
			trust.DELETE("/credentials/:id/revoke", RevokeCredentialHandler)
			trust.GET("/provenance", ListProvenanceHandler)
			trust.POST("/provenance/register", RegisterProvenanceHandler)
			trust.POST("/provenance/:id/transfer", TransferProvenanceHandler)
			trust.POST("/signatures/sign", SignArtifactHandler)
			trust.POST("/signatures/verify", VerifySignatureHandler)
			trust.GET("/registry", ListTrustRegistryHandler)
			trust.POST("/registry", RegisterTrustEntryHandler)
			trust.GET("/registry/:id", GetTrustEntryHandler)
			trust.PUT("/registry/:id/status", UpdateTrustEntryHandler)
			trust.POST("/pki/csr", IssueCertificateHandler)
			trust.GET("/pki/certificates", ListCertificatesHandler)
			trust.POST("/pki/certificates/:id/revoke", RevokeCertificateHandler)
			trust.POST("/tsa/stamp", TimestampHandler)
			trust.GET("/tsa/verify/:id", VerifyTimestampHandler)
		}
		interop := v1.Group("/interop")
		{
			interop.GET("/apps", ListInterAppsHandler)
			interop.POST("/apps/probe", ProbeInterAppHandler)
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

func TestDLPScanDetection(t *testing.T) {
	r := setupTestRouter()

	payload := DLPScanRequest{
		SourceApp: "StatCollect",
		DataType:  "TEXT",
		Content:   "Participant CM89012345678A registered with api_key = 'secret_token_12345678'",
		Redact:    true,
	}

	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/secops/dlp/scan", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var res DLPScanResult
	_ = json.Unmarshal(w.Body.Bytes(), &res)

	if res.Safe {
		t.Fatalf("Expected unsafe scan result due to NIN and secret key")
	}
	if res.ViolationsFound < 2 {
		t.Fatalf("Expected at least 2 violations, got %d", res.ViolationsFound)
	}
	if res.ActionTaken != "BLOCKED" && res.ActionTaken != "REDACTED" {
		t.Fatalf("Expected BLOCKED or REDACTED action, got %s", res.ActionTaken)
	}
}

func TestAuditLedgerChaining(t *testing.T) {
	r := setupTestRouter()

	initialBlocks := globalStore.ListLedger()
	initialHeight := len(initialBlocks)

	appendReq := map[string]interface{}{
		"event_type": "rms.ethics_review.approved",
		"source_app": "RMS",
		"actor_id":   "ethics_board_chair",
		"payload": map[string]interface{}{
			"study_id": "RES-2026-999",
			"status":   "APPROVED_CONFIDENTIAL",
		},
	}

	b, _ := json.Marshal(appendReq)
	req, _ := http.NewRequest("POST", "/api/v1/trust/ledger/append", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d", w.Code)
	}

	var newBlock AuditLedgerBlock
	_ = json.Unmarshal(w.Body.Bytes(), &newBlock)

	if newBlock.Index != int64(initialHeight+1) {
		t.Fatalf("Expected block index %d, got %d", initialHeight+1, newBlock.Index)
	}
	if newBlock.PrevHash != initialBlocks[initialHeight-1].RecordHash {
		t.Fatalf("Expected prev_hash to match last block record hash")
	}
}

func TestVerifiableCredentialIssueAndVerify(t *testing.T) {
	r := setupTestRouter()

	issueReq := map[string]interface{}{
		"holder_did":      "did:statgate:user:researcher.alice",
		"holder_name":     "Alice Ssemwogerere",
		"credential_type": "FieldBiometricAuditorCredential",
		"credential_subject": map[string]interface{}{
			"authorization": "Uganda Census Survey 2026",
			"security_tier": "TIER_4_CONFIDENTIAL",
		},
	}

	b, _ := json.Marshal(issueReq)
	req, _ := http.NewRequest("POST", "/api/v1/trust/credentials/issue", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d", w.Code)
	}

	var vc VerifiableCredential
	_ = json.Unmarshal(w.Body.Bytes(), &vc)

	// Verify credential
	verifyReq := map[string]interface{}{
		"credential_id": vc.CredentialID,
	}
	vb, _ := json.Marshal(verifyReq)
	vreq, _ := http.NewRequest("POST", "/api/v1/trust/credentials/verify", bytes.NewBuffer(vb))
	vreq.Header.Set("Content-Type", "application/json")
	vw := httptest.NewRecorder()
	r.ServeHTTP(vw, vreq)

	if vw.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from credential verify, got %d", vw.Code)
	}
}

func TestEncryptionKeyRotation(t *testing.T) {
	r := setupTestRouter()

	// 1. List keys
	req, _ := http.NewRequest("GET", "/api/v1/secops/keys", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// 2. Rotate master key
	rotateReq := map[string]interface{}{
		"key_id": "key_sec_master_01",
	}
	b, _ := json.Marshal(rotateReq)
	rreq, _ := http.NewRequest("POST", "/api/v1/secops/keys/rotate", bytes.NewBuffer(b))
	rreq.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, rreq)
	if rw.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from key rotation, got %d", rw.Code)
	}

	var rotateResp map[string]interface{}
	_ = json.Unmarshal(rw.Body.Bytes(), &rotateResp)
	if rotateResp["action"] != "key_rotated" {
		t.Fatalf("Expected action 'key_rotated', got %v", rotateResp["action"])
	}
}

func TestPKICertificateIssuanceAndRevocation(t *testing.T) {
	r := setupTestRouter()

	// 1. Issue certificate
	csr := map[string]interface{}{
		"common_name":         "PMS Subservice Agent",
		"organization":        "StatGate Consortium",
		"organizational_unit": "PMS Production",
		"country":             "UG",
		"subject_did":         "did:statgate:service:pms",
		"key_usage":           []string{"DIGITAL_SIGNATURE", "KEY_ENCIPHERMENT"},
		"validity_days":       365,
	}
	b, _ := json.Marshal(csr)
	req, _ := http.NewRequest("POST", "/api/v1/trust/pki/csr", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for PKI CSR, got %d", w.Code)
	}

	var cert PKICertificate
	_ = json.Unmarshal(w.Body.Bytes(), &cert)

	// 2. Revoke certificate
	revReq := map[string]interface{}{
		"reason": "Routine Key Lifecycle Expiration",
	}
	rb, _ := json.Marshal(revReq)
	rreq, _ := http.NewRequest("POST", "/api/v1/trust/pki/certificates/"+cert.ID+"/revoke", bytes.NewBuffer(rb))
	rreq.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, rreq)
	if rw.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for cert revoke, got %d", rw.Code)
	}
}

func TestTimestampAuthority(t *testing.T) {
	r := setupTestRouter()

	// 1. Stamp artifact
	stampReq := map[string]interface{}{
		"artifact_id":   "dataset_census_2026_final.parquet",
		"artifact_type": "OFFICIAL_DATASET",
		"content":       "sample dataset contents to be hashed and stamped",
	}
	b, _ := json.Marshal(stampReq)
	req, _ := http.NewRequest("POST", "/api/v1/trust/tsa/stamp", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for TSA timestamp, got %d", w.Code)
	}

	var ts TimestampRecord
	_ = json.Unmarshal(w.Body.Bytes(), &ts)

	// 2. Verify timestamp
	vreq, _ := http.NewRequest("GET", "/api/v1/trust/tsa/verify/"+ts.ID, nil)
	vw := httptest.NewRecorder()
	r.ServeHTTP(vw, vreq)
	if vw.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for TSA verify, got %d", vw.Code)
	}
}

func TestTrustRegistryOperations(t *testing.T) {
	r := setupTestRouter()

	// 1. Register trust authority
	entry := map[string]interface{}{
		"did":               "did:statgate:health:infectious-disease-unit",
		"organization_name": "Infectious Disease Control Agency",
		"trust_level":       "ACCREDITED_PARTNER",
		"public_keys":       []string{"ed25519:abcdef1234567890abcdef1234567890"},
		"authorized_scopes": []string{"HEALTH_SURVEILLANCE"},
	}
	b, _ := json.Marshal(entry)
	req, _ := http.NewRequest("POST", "/api/v1/trust/registry", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for Trust Registry, got %d", w.Code)
	}

	// 2. Get trust entry
	greq, _ := http.NewRequest("GET", "/api/v1/trust/registry/did:statgate:health:infectious-disease-unit", nil)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, greq)
	if gw.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for GetTrustEntry, got %d", gw.Code)
	}
}

func TestInterAppConnectivityList(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("GET", "/api/v1/interop/apps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestWorkspaceScopedTrustRecords(t *testing.T) {
	store := NewMemStore()

	store.AddIncidentScoped(SecurityIncident{Title: "Workspace one"}, "tenant-alpha", "workspace_one")
	second := store.AddIncidentScoped(SecurityIncident{Title: "Workspace two"}, "tenant-alpha", "workspace_two")
	workspaceOneIncidents := store.ListIncidentsScoped("tenant-alpha", "workspace_one")
	if len(workspaceOneIncidents) != 1 {
		t.Fatal("workspace one should only see its incident")
	}
	if _, err := store.UpdateIncidentStatusScoped(second.ID, "CLOSED", "cross-workspace", "tenant-alpha", "workspace_one"); err == nil {
		t.Fatal("cross-workspace incident update should be rejected")
	}

	one := store.AppendLedgerScoped("workspace.one", "StatTrust", "actor", map[string]interface{}{}, "tenant-alpha", "workspace_one")
	two := store.AppendLedgerScoped("workspace.two", "StatTrust", "actor", map[string]interface{}{}, "tenant-alpha", "workspace_two")
	if one.Index != 1 || two.Index != 1 {
		t.Fatalf("workspace ledgers should have independent chain indexes, got %d and %d", one.Index, two.Index)
	}
	if len(store.ListLedgerScoped("tenant-alpha", "workspace_one")) != 1 {
		t.Fatal("workspace one should only see its ledger")
	}

	store.RegisterProvenanceScoped(ArtifactProvenance{ArtifactID: "shared-artifact", ArtifactName: "One", ArtifactType: "DATASET", OriginApp: "PMS"}, "tenant-alpha", "workspace_one")
	if _, found := store.GetProvenanceScoped("shared-artifact", "tenant-alpha", "workspace_two"); found {
		t.Fatal("cross-workspace provenance lookup should be rejected")
	}
}
