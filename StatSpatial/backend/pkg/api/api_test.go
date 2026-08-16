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

	"statspatial/pkg/api"

	"github.com/gorilla/mux"
)

func createTestToken(secret, userID, role, tenantID string) string {
	claims := map[string]interface{}{
		"sub":       userID,
		"userId":    userID,
		"role":      role,
		"tenant_id": tenantID,
		"email":     userID + "@statgate.gov",
		"iss":       "statgate-registry",
		"aud":       "statgate",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	return signClaims(secret, claims)
}

func signClaims(secret string, claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	data := header + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return data + "." + sig
}

func TestStatSpatialRoutesAndAuth(t *testing.T) {
	secret := "test-statspatial-secret-32-chars-long"
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", secret)
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	r := mux.NewRouter()
	api.RegisterRoutes(r)

	// Test 1: Health endpoint should be publicly accessible
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 200 or 503 from /health, got %d", rr.Code)
	}

	// Test 2: Unauthenticated request to /api/v1/summary should be 401
	req = httptest.NewRequest("GET", "/api/v1/summary", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", rr.Code)
	}

	// Test 3: Authenticated request with valid token
	token := createTestToken(secret, "admin-1", "admin", "uganda-national")
	req = httptest.NewRequest("GET", "/api/v1/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Since DB is not connected in mock test, it may return 500 from store or proceed past auth.
	// The important check is it must NOT return 401.
	if rr.Code == http.StatusUnauthorized {
		t.Errorf("Valid token must not be rejected with 401")
	}

	// Test 4: Cross-tenant injection attempt should be rejected with 403
	req = httptest.NewRequest("GET", "/api/v1/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", "malicious-tenant-injection")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for mismatched X-Tenant-ID, got %d", rr.Code)
	}
}

func TestStatSpatialRejectsInvalidTokens(t *testing.T) {
	secret := "test-statspatial-secret-32-chars-long"
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", secret)
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	r := mux.NewRouter()
	api.RegisterRoutes(r)

	assertAuthRejection := func(name, token string) {
		t.Helper()
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/summary", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s: expected 401, got %d", name, rr.Code)
		}
	}

	// Expired token must be rejected.
	expired := signClaims(secret, map[string]interface{}{
		"sub": "usr-1", "role": "admin", "tenant_id": "uganda-national",
		"iss": "statgate-registry", "aud": "statgate",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	assertAuthRejection("expired token", expired)

	// Non-canonical role must be rejected.
	invalidRole := signClaims(secret, map[string]interface{}{
		"sub": "usr-1", "role": "super_hacker_bypass", "tenant_id": "uganda-national",
		"iss": "statgate-registry", "aud": "statgate",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	assertAuthRejection("non-canonical role", invalidRole)

	// Missing tenant context must be rejected.
	missingTenant := signClaims(secret, map[string]interface{}{
		"sub": "usr-1", "role": "admin",
		"iss": "statgate-registry", "aud": "statgate",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	assertAuthRejection("missing tenant context", missingTenant)

	// Mock/demo tokens must be rejected by the shared validator.
	assertAuthRejection("mock token", "mock_user_token")
}

func TestStatSpatialFailClosedOnMissingSecret(t *testing.T) {
	// Ensure no secret is configured for this test.
	os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	r := mux.NewRouter()
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/summary", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Zero-default: without a signing secret the API must fail closed (503),
	// never silently authorising anonymous access.
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 ServiceUnavailable when secret missing, got %d", rr.Code)
	}
}
