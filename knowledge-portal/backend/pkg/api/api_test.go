package api_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"knowledgeportal/pkg/api"

	"github.com/gorilla/mux"
)

func signClaims(secret string, claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	data := header + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return data + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func createTestToken(secret, userID, role, tenantID string) string {
	return signClaims(secret, map[string]interface{}{
		"sub": userID, "userId": userID, "role": role, "tenant_id": tenantID,
		"email": userID + "@statgate.gov", "iss": "statgate-registry", "aud": "statgate",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
}

func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// TestKnowledgeAuthContract validates the Registry-JWT security contract
// (public bypass, 401, valid pass, 403 cross-tenant) without a live database.
func TestKnowledgeAuthContract(t *testing.T) {
	secret := "test-knowledge-secret-32-chars-long"
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", secret)
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	protected := api.AuthMiddleware(echoHandler())

	do := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		rr := httptest.NewRecorder()
		protected.ServeHTTP(rr, req)
		return rr
	}

	// Public portal path bypasses auth -> next handler runs (200).
	if rr := do("GET", "/api/public/search?q=malaria"); rr.Code != http.StatusOK {
		t.Errorf("Public path must bypass auth, got %d", rr.Code)
	}
	// Probe path bypasses auth.
	if rr := do("GET", "/health"); rr.Code != http.StatusOK {
		t.Errorf("Probe must bypass auth, got %d", rr.Code)
	}
	// No token on protected path -> 401.
	if rr := do("GET", "/api/v1/summary"); rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 without token, got %d", rr.Code)
	}

	// Valid token on protected path -> next handler runs (200).
	token := createTestToken(secret, "admin-1", "admin", "uganda-national")
	req := httptest.NewRequest("GET", "/api/v1/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Valid token must pass auth, got %d", rr.Code)
	}

	// Cross-tenant header mismatch -> 403.
	req = httptest.NewRequest("GET", "/api/v1/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", "malicious-tenant-injection")
	rr = httptest.NewRecorder()
	protected.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for mismatched X-Tenant-ID, got %d", rr.Code)
	}
}

// TestKnowledgeRejectsInvalidTokens covers expired / rogue-role / no-tenant / mock.
func TestKnowledgeRejectsInvalidTokens(t *testing.T) {
	secret := "test-knowledge-secret-32-chars-long"
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", secret)
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	protected := api.AuthMiddleware(echoHandler())
	assertReject := func(name, token string) {
		t.Helper()
		req := httptest.NewRequest("GET", "/api/v1/summary", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		protected.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s: expected 401, got %d", name, rr.Code)
		}
	}

	assertReject("expired token", signClaims(secret, map[string]interface{}{
		"sub": "u", "role": "admin", "tenant_id": "ug", "iss": "statgate-registry", "aud": "statgate",
		"exp": time.Now().Add(-time.Hour).Unix()}))
	assertReject("non-canonical role", signClaims(secret, map[string]interface{}{
		"sub": "u", "role": "root_bypass", "tenant_id": "ug", "iss": "statgate-registry", "aud": "statgate",
		"exp": time.Now().Add(time.Hour).Unix()}))
	assertReject("missing tenant", signClaims(secret, map[string]interface{}{
		"sub": "u", "role": "admin", "iss": "statgate-registry", "aud": "statgate",
		"exp": time.Now().Add(time.Hour).Unix()}))
	assertReject("mock token", "mock_user_token")
}

// TestKnowledgeFailClosedOnMissingSecret verifies zero-default enforcement.
func TestKnowledgeFailClosedOnMissingSecret(t *testing.T) {
	os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	protected := api.AuthMiddleware(echoHandler())
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, httptest.NewRequest("GET", "/api/v1/summary", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 when secret missing, got %d", rr.Code)
	}
}

// TestKnowledgeRoutesRegister verifies the route table mounts without panic.
func TestKnowledgeRoutesRegister(t *testing.T) {
	r := mux.NewRouter()
	api.RegisterRoutes(r)
	if r == nil {
		t.Fatal("router must be created")
	}
}

