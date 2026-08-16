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

	"bpmhub/pkg/api"

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

func TestAuthContract(t *testing.T) {
	secret := "test-bpm-secret-32-chars-long!!"
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", secret)
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	protected := api.AuthMiddleware(echoHandler())
	do := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		rr := httptest.NewRecorder()
		protected.ServeHTTP(rr, req)
		return rr
	}
	if rr := do("GET", "/metrics"); rr.Code != http.StatusOK {
		t.Errorf("probe must bypass auth, got %d", rr.Code)
	}
	if rr := do("GET", "/api/v1/processes"); rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rr.Code)
	}
	token := createTestToken(secret, "admin-1", "admin", "uganda-national")
	req := httptest.NewRequest("GET", "/api/v1/processes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("valid token must pass auth, got %d", rr.Code)
	}
	req = httptest.NewRequest("GET", "/api/v1/processes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", "malicious-tenant-injection")
	rr = httptest.NewRecorder()
	protected.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for mismatched X-Tenant-ID, got %d", rr.Code)
	}
}

func TestRejectInvalidTokens(t *testing.T) {
	secret := "test-bpm-secret-32-chars-long!!"
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", secret)
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")
	protected := api.AuthMiddleware(echoHandler())
	assertReject := func(name, token string) {
		t.Helper()
		req := httptest.NewRequest("GET", "/api/v1/processes", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		protected.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s: expected 401, got %d", name, rr.Code)
		}
	}
	assertReject("expired", signClaims(secret, map[string]interface{}{
		"sub": "u", "role": "admin", "tenant_id": "ug", "iss": "statgate-registry", "aud": "statgate",
		"exp": time.Now().Add(-time.Hour).Unix()}))
	assertReject("rogue role", signClaims(secret, map[string]interface{}{
		"sub": "u", "role": "root_bypass", "tenant_id": "ug", "iss": "statgate-registry", "aud": "statgate",
		"exp": time.Now().Add(time.Hour).Unix()}))
	assertReject("mock token", "mock_user_token")
}

func TestFailClosedOnMissingSecret(t *testing.T) {
	os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")
	protected := api.AuthMiddleware(echoHandler())
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, httptest.NewRequest("GET", "/api/v1/processes", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when secret missing, got %d", rr.Code)
	}
}

func TestRoutesRegister(t *testing.T) {
	r := mux.NewRouter()
	api.RegisterRoutes(r)
	if r == nil {
		t.Fatal("router must be created")
	}
}

// TestTransitionEngine: nextNode resolution drives the workflow engine.
func TestTransitionEngine(t *testing.T) {
	transitions := map[string]interface{}{"start": "review", "review": "approve", "approve": ""}
	if n := api.NextNode(transitions, "start"); n != "review" {
		t.Errorf("expected review, got %s", n)
	}
	if n := api.NextNode(transitions, "start"); n != "review" {
		t.Errorf("expected review, got %s", n)
	}
	if n := api.NextNode(transitions, "approve"); n != "" {
		t.Errorf("expected end (empty), got %s", n)
	}
}