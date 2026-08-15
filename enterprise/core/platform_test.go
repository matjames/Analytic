package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ─── Test helpers ─────────────────────────────────────────────────

// buildTestSig generates an HS256 signature for testing, matching middleware_security.go logic.
func buildTestSig(secret, input string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// buildTestJWT creates a canonical HS256 JWT for testing purposes only.
// It follows the StatGate JWT standard (iss/aud/tenant/role/exp) so the
// strict verifier accepts it.
func buildTestJWT(secret, userID, tenantID, role string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	now := time.Now().Unix()
	claims, _ := json.Marshal(map[string]interface{}{
		"sub":       userID,
		"user_id":   userID,
		"tenant_id": tenantID,
		"org_id":    "org-national",
		"role":      role,
		"email":     userID + "@statgate.local",
		"iat":       now,
		"nbf":       now - 30,
		"exp":       9999999999,
		"iss":       "statgate-registry",
		"aud":       "statgate",
	})
	claimsEnc := base64.RawURLEncoding.EncodeToString(claims)
	sig := buildTestSig(secret, header+"."+claimsEnc)
	return header + "." + claimsEnc + "." + sig
}

// ─── Sprint 1: Security Middleware Tests ─────────────────────────

func TestSecurityHeadersPresent(t *testing.T) {
	r := gin.New()
	r.Use(securityHeadersMiddleware())
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	expected := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Strict-Transport-Security",
		"X-XSS-Protection",
	}
	for _, h := range expected {
		if w.Header().Get(h) == "" {
			t.Errorf("expected security header %q to be set", h)
		}
	}
}

func TestCorrelationIDGenerated(t *testing.T) {
	r := gin.New()
	r.Use(correlationMiddleware())
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Correlation-ID") == "" {
		t.Error("expected X-Correlation-ID to be generated")
	}
}

func TestCorrelationIDPropagated(t *testing.T) {
	r := gin.New()
	r.Use(correlationMiddleware())
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", "test-corr-123")
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Correlation-ID") != "test-corr-123" {
		t.Errorf("expected propagated correlation ID, got %q", w.Header().Get("X-Correlation-ID"))
	}
}

func TestJWTMiddlewareRejectsNoAuth(t *testing.T) {
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", "test-secret-for-unit-tests-only-32chars!!")
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	r := gin.New()
	r.Use(jwtAuthMiddleware())
	r.GET("/api/protected", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for missing auth, got %d", w.Code)
	}
}

func TestJWTMiddlewareRejectsInvalidToken(t *testing.T) {
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", "test-secret-for-unit-tests-only-32chars!!")
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	r := gin.New()
	r.Use(jwtAuthMiddleware())
	r.GET("/api/protected", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer this.is.invalid")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for invalid token, got %d", w.Code)
	}
}

func TestJWTMiddlewareAllowsProbeEndpoints(t *testing.T) {
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", "test-secret-for-unit-tests-only-32chars!!")
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	r := gin.New()
	r.Use(jwtAuthMiddleware())
	for _, path := range []string{"/health", "/ready", "/live", "/metrics"} {
		p := path
		r.GET(p, func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	}

	for _, path := range []string{"/health", "/ready", "/live", "/metrics"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("probe %s should not require auth, got %d", path, w.Code)
		}
	}
}

func TestTenantIsolationRejectsMismatch(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", "tenant-A")
		c.Next()
	})
	r.Use(tenantIsolationMiddleware())
	r.GET("/api/data", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-Tenant-ID", "tenant-B")
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 for tenant mismatch, got %d", w.Code)
	}
}

func TestTenantIsolationAllowsMatch(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", "tenant-A")
		c.Next()
	})
	r.Use(tenantIsolationMiddleware())
	r.GET("/api/data", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-Tenant-ID", "tenant-A")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 when tenant matches, got %d", w.Code)
	}
}

func TestRateLimiterAllowsFirstRequest(t *testing.T) {
	os.Setenv("STATGATE_RATE_LIMIT_RPM", "600")
	defer os.Unsetenv("STATGATE_RATE_LIMIT_RPM")

	// Clear any existing bucket for test IP
	rateMu.Lock()
	delete(rateBuckets, "192.0.2.99")
	rateMu.Unlock()

	r := gin.New()
	r.Use(rateLimitMiddleware())
	r.GET("/api/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.0.2.99:1234"
	r.ServeHTTP(w, req)

	if w.Code == 429 {
		t.Error("first request should not be rate limited")
	}
}

func TestRequestSizeRejection(t *testing.T) {
	os.Setenv("STATGATE_MAX_REQUEST_BODY_KB", "1")
	defer os.Unsetenv("STATGATE_MAX_REQUEST_BODY_KB")

	r := gin.New()
	r.Use(requestSizeMiddleware())
	r.POST("/api/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	body := strings.Repeat("a", 2048)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/test", bytes.NewBufferString(body))
	req.ContentLength = int64(len(body))
	r.ServeHTTP(w, req)

	if w.Code != 413 {
		t.Errorf("expected 413 for oversized body, got %d", w.Code)
	}
}

// ─── Sprint 2: Production Secret Validation ───────────────────────

func TestProductionSecretValidation_DevPassesWithoutSecrets(t *testing.T) {
	os.Setenv("STATGATE_ENV", "development")
	os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")
	os.Unsetenv("STATGATE_INTERNAL_API_KEY")
	defer os.Unsetenv("STATGATE_ENV")

	// Should not panic in development mode
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("validateProductionSecrets should not panic in dev: %v", r)
		}
	}()
	validateProductionSecrets()
}

// ─── Sprint 3: Persistence Helpers ───────────────────────────────

func TestMarshalJSON_ValidInput(t *testing.T) {
	result, err := marshalJSON(map[string]interface{}{"key": "value", "n": 42})
	if err != nil {
		t.Fatalf("marshalJSON failed: %v", err)
	}
	if !strings.Contains(result, "key") || !strings.Contains(result, "42") {
		t.Errorf("unexpected marshalled JSON: %s", result)
	}
}

func TestBuildDSN_EmptyConfig(t *testing.T) {
	os.Unsetenv("STATGATE_DB_URL")
	os.Unsetenv("STATGATE_DB_HOST")
	if dsn := buildDSN(); dsn != "" {
		t.Errorf("expected empty DSN when unconfigured, got %q", dsn)
	}
}

func TestBuildDSN_WithURL(t *testing.T) {
	os.Setenv("STATGATE_DB_URL", "postgres://user:pass@localhost/db")
	defer os.Unsetenv("STATGATE_DB_URL")
	dsn := buildDSN()
	if dsn != "postgres://user:pass@localhost/db" {
		t.Errorf("expected URL DSN, got %q", dsn)
	}
}

// ─── Sprint 5: Observability ──────────────────────────────────────

func TestMetricsCollectorCounts(t *testing.T) {
	mc := &MetricsCollector{startTime: startTime}
	mc.incRequests()
	mc.incRequests()
	mc.incErrors()
	mc.incEvents()
	mc.incDBQuery()

	snap := mc.snapshot()
	req := snap["requests"].(map[string]interface{})
	if req["total"].(int64) != 2 {
		t.Errorf("expected 2 total requests, got %v", req["total"])
	}
	if req["errors"].(int64) != 1 {
		t.Errorf("expected 1 error, got %v", req["errors"])
	}
	evts := snap["events"].(map[string]interface{})
	if evts["total_published"].(int64) != 1 {
		t.Errorf("expected 1 published event, got %v", evts["total_published"])
	}
}

func TestIsProbeEndpoint(t *testing.T) {
	for _, p := range []string{"/health", "/ready", "/live", "/metrics", "/api/info"} {
		if !isProbeEndpoint(p) {
			t.Errorf("expected %s to be a probe endpoint", p)
		}
	}
	if isProbeEndpoint("/api/events") {
		t.Error("/api/events should not be a probe endpoint")
	}
}

func TestLivenessEndpointReturnsAlive(t *testing.T) {
	r := gin.New()
	r.GET("/live", handleLiveness)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/live", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if body["status"] != "alive" {
		t.Errorf("expected status=alive, got %v", body["status"])
	}
}

func TestReadinessWithNoDependencies(t *testing.T) {
	// With no DB or Redis — should not panic, must return sensible output
	ready, details := isReady()
	if details == nil {
		t.Error("expected non-nil readiness details")
	}
	_ = ready
}

// ─── Sprint 1: JWT parse unit tests ──────────────────────────────

func TestJWTParse_ExpiredToken(t *testing.T) {
	secret := "test-secret-for-unit-tests"
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]interface{}{"user_id": "u1", "exp": int64(1)})
	claimsEnc := base64.RawURLEncoding.EncodeToString(claims)
	sig := buildTestSig(secret, header+"."+claimsEnc)
	token := header + "." + claimsEnc + "." + sig

	_, err := parseJWT(token, secret)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestJWTParse_WrongSecret(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]interface{}{"user_id": "u1", "exp": int64(9999999999)})
	claimsEnc := base64.RawURLEncoding.EncodeToString(claims)
	sig := buildTestSig("wrong-secret", header+"."+claimsEnc)
	token := header + "." + claimsEnc + "." + sig

	_, err := parseJWT(token, "correct-secret")
	if err == nil {
		t.Error("expected error for wrong signing secret")
	}
}

func TestJWTParse_ValidToken(t *testing.T) {
	secret := "test-secret-for-unit-tests"
	token := buildTestJWT(secret, "user-123", "tenant-A", "admin")

	claims, err := parseJWT(token, secret)
	if err != nil {
		t.Fatalf("expected valid token to parse: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("expected user_id=user-123, got %q", claims.UserID)
	}
	if claims.TenantID != "tenant-A" {
		t.Errorf("expected tenant_id=tenant-A, got %q", claims.TenantID)
	}
	if claims.Role != "admin" {
		t.Errorf("expected role=admin, got %q", claims.Role)
	}
}

func TestJWTParse_MalformedToken(t *testing.T) {
	_, err := parseJWT("not.a.jwt.at.all", "secret")
	if err == nil {
		t.Error("expected error for malformed token")
	}
}
