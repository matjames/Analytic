package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestEngine() (*gin.Engine, *Config) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(correlationMiddleware())

	cfg := &Config{
		Env:                  "test",
		Port:                 "8097",
		UIPort:               "3015",
		RateLimitRPM:         1000,
		MaxRequestKB:         10240,
		CitizenSessionSecret: "test-secret-key-for-sha256-hmac-test",
		EnterpriseJWTSecret:  "test-jwt-secret",
		DefaultTenantID:      "tenant_test",
	}

	initSecurity(cfg)
	registerRoutes(r, cfg)
	return r, cfg
}

func TestHealthAndReadiness(t *testing.T) {
	r, _ := setupTestEngine()

	// Test /health
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected /health 200, got %d", w.Code)
	}

	var healthRes map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &healthRes); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if healthRes["status"] != "healthy" || healthRes["app"] != "statcitizen" {
		t.Fatalf("unexpected health payload: %v", healthRes)
	}

	// Test /ready
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/ready", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected /ready 200, got %d", w.Code)
	}

	// Test /metrics
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected /metrics 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "statcitizen_uptime_seconds") {
		t.Fatalf("expected metrics output to contain statcitizen_uptime_seconds")
	}
}

func TestCitizenSessionCreation(t *testing.T) {
	r, _ := setupTestEngine()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/statcitizen/v1/session", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["session_id"] == nil || res["token"] == nil {
		t.Fatalf("expected session_id and token in response: %v", res)
	}

	tokenStr := res["token"].(string)
	if len(tokenStr) < 16 {
		t.Fatalf("session token too short: %s", tokenStr)
	}
}

func TestExplicitConsentRequirement(t *testing.T) {
	r, _ := setupTestEngine()

	// Register without consent -> Must be rejected
	payload := map[string]interface{}{
		"display_name":    "Citizen Test",
		"email":           "citizen@test.org",
		"consent_granted": false,
	}
	b, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/statcitizen/v1/register", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for unconsented registration, got %d", w.Code)
	}

	var errRes ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &errRes)
	if errRes.Error != "consent_required" {
		t.Fatalf("expected error code 'consent_required', got '%s'", errRes.Error)
	}

	// Register with explicit consent -> Accepted
	payload["consent_granted"] = true
	b, _ = json.Marshal(payload)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/statcitizen/v1/register", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for consented registration, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPIIHashingAndMinimization(t *testing.T) {
	email := "citizen.confidential@example.gov"
	hash1 := hashPII(email)
	hash2 := hashPII("  CITIZEN.CONFIDENTIAL@EXAMPLE.GOV  ")

	if hash1 == "" {
		t.Fatal("expected non-empty PII hash")
	}
	if hash1 != hash2 {
		t.Fatalf("expected normalized PII hashes to match: %s != %s", hash1, hash2)
	}
	if strings.Contains(hash1, "citizen") || strings.Contains(hash1, "example") {
		t.Fatal("PII hash leaked plaintext characters")
	}
}

func TestServiceRatingDimensions(t *testing.T) {
	r, _ := setupTestEngine()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/statcitizen/v1/ratings/dimensions", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res struct {
		Dimensions []RatingDimension `json:"dimensions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode dimensions: %v", err)
	}

	if len(res.Dimensions) == 0 {
		t.Fatal("expected at least default configurable rating dimensions")
	}

	foundSatisfaction := false
	for _, d := range res.Dimensions {
		if d.Slug == "satisfaction" {
			foundSatisfaction = true
			break
		}
	}
	if !foundSatisfaction {
		t.Fatal("expected 'satisfaction' dimension in default rating dimensions")
	}
}

func TestGovernedAIQueryInsufficientEvidenceFallback(t *testing.T) {
	r, _ := setupTestEngine()

	payload := AIQueryRequest{
		Query: "What is the confidential internal budget code for secret facility X99?",
	}
	b, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/statcitizen/v1/ai/query", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var aiRes AIQueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &aiRes); err != nil {
		t.Fatalf("failed to decode AI response: %v", err)
	}

	// Must trigger insufficient evidence fallback and not invent internal data
	if !aiRes.InsufficientEvidence {
		t.Fatalf("expected InsufficientEvidence=true for unauthorized/unindexed query")
	}
	if aiRes.Confidence != 0.0 {
		t.Fatalf("expected Confidence=0.0 on fallback")
	}
	if !strings.Contains(aiRes.Disclaimer, "sovereign intelligence constraints") && !strings.Contains(aiRes.Disclaimer, "StatCitizen AI") {
		t.Fatalf("expected sovereign intelligence disclaimer in AI response")
	}
}

func TestUniversalObjectContextStructure(t *testing.T) {
	r, _ := setupTestEngine()

	canonicalID := "tenant_test:statcitizen:report:rpt_12345"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/statcitizen/v1/universal/context/"+canonicalID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var uCtx UniversalObjectContext
	if err := json.Unmarshal(w.Body.Bytes(), &uCtx); err != nil {
		t.Fatalf("failed to decode universal context: %v", err)
	}

	if uCtx.CanonicalID != canonicalID {
		t.Fatalf("expected canonical_id %s, got %s", canonicalID, uCtx.CanonicalID)
	}
	if uCtx.Overview == nil || uCtx.Workflow == nil || uCtx.AIIntelligence == nil {
		t.Fatal("universal object context missing core facets (Overview, Workflow, AIIntelligence)")
	}
}

func TestOfflineDraftQueuing(t *testing.T) {
	r, _ := setupTestEngine()

	payload := map[string]interface{}{
		"session_id": "cs_test_session_123",
		"draft_type": "report",
		"payload": map[string]interface{}{
			"title":       "Draft Road Issue",
			"description": "Pothole on main highway",
			"district":    "Kampala",
		},
	}
	b, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/statcitizen/v1/offline/drafts", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted for offline draft queue, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["draft_id"] == nil || res["correlation_id"] == nil {
		t.Fatalf("expected draft_id and correlation_id in response: %v", res)
	}
}
