package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Test key pair for signing test tokens
var testPrivateKey *rsa.PrivateKey
var testPublicKey *rsa.PublicKey
var testKeyID = "test-key-id"

func init() {
	// Generate a test RSA key pair
	var err error
	testPrivateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("failed to generate test key: " + err.Error())
	}
	testPublicKey = &testPrivateKey.PublicKey
}

// createTestToken creates a signed JWT for testing
func createTestToken(claims jwt.MapClaims, validSignature bool) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKeyID

	var key *rsa.PrivateKey
	if validSignature {
		key = testPrivateKey
	} else {
		// Generate a different key for invalid signature
		badKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		key = badKey
	}

	tokenString, err := token.SignedString(key)
	if err != nil {
		panic("failed to sign test token: " + err.Error())
	}
	return tokenString
}

// setupTestJWKS sets up a mock JWKS in the cache
func setupTestJWKS() {
	// Convert public key to JWK format
	nBytes := testPublicKey.N.Bytes()
	eBytes := big.NewInt(int64(testPublicKey.E)).Bytes()

	jwks := &JWKS{
		Keys: []JWK{
			{
				Kid: testKeyID,
				Kty: "RSA",
				Alg: "RS256",
				Use: "sig",
				N:   base64.RawURLEncoding.EncodeToString(nBytes),
				E:   base64.RawURLEncoding.EncodeToString(eBytes),
			},
		},
	}

	jwksCacheMux.Lock()
	jwksCache = jwks
	jwksCacheTime = time.Now()
	jwksCacheMux.Unlock()
}

// TestAuthMiddleware_ModeOff tests that auth is skipped when mode is "off"
func TestAuthMiddleware_ModeOff(t *testing.T) {
	// Save and restore original auth mode
	originalMode := authMode
	defer func() { authMode = originalMode }()

	authMode = "off"

	handler := AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Request without any auth header should pass
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

// TestAuthMiddleware_MissingHeader tests rejection when Authorization header is missing
func TestAuthMiddleware_MissingHeader(t *testing.T) {
	originalMode := authMode
	defer func() { authMode = originalMode }()

	authMode = "enforce"

	handler := AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}

	// Check error response
	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] == "" {
		t.Error("expected error message in response")
	}
}

// TestAuthMiddleware_InvalidFormat tests rejection for malformed Authorization header
func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	originalMode := authMode
	defer func() { authMode = originalMode }()

	authMode = "enforce"

	handler := AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test without "Bearer " prefix
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Basic abc123")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

// TestAuthMiddleware_ValidToken tests successful authentication
func TestAuthMiddleware_ValidToken(t *testing.T) {
	originalMode := authMode
	originalClientID := keycloakClientID
	defer func() {
		authMode = originalMode
		keycloakClientID = originalClientID
	}()

	authMode = "enforce"
	keycloakClientID = "test-client"
	setupTestJWKS()

	// Create valid token
	claims := jwt.MapClaims{
		"sub":                "user-123",
		"preferred_username": "testuser",
		"email":              "test@example.com",
		"azp":                "test-client",
		"exp":                time.Now().Add(1 * time.Hour).Unix(),
		"iat":                time.Now().Unix(),
		"realm_access": map[string]interface{}{
			"roles": []interface{}{"user"},
		},
		"resource_access": map[string]interface{}{
			"dwhlanding": map[string]interface{}{
				"roles": []interface{}{"admin", "viewer"},
			},
		},
	}
	token := createTestToken(claims, true)

	var capturedClaims *UserClaims
	handler := AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		capturedClaims = GetUserClaims(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify claims were extracted
	if capturedClaims == nil {
		t.Fatal("expected claims to be captured")
	}
	if capturedClaims.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", capturedClaims.Username)
	}
	if capturedClaims.Subject != "user-123" {
		t.Errorf("expected subject 'user-123', got %q", capturedClaims.Subject)
	}

	// Check client roles
	if !hasClientRole(capturedClaims, "dwhlanding", "admin") {
		t.Error("expected user to have 'admin' client role")
	}
	if !hasClientRole(capturedClaims, "dwhlanding", "viewer") {
		t.Error("expected user to have 'viewer' client role")
	}

	// Check realm roles
	if !hasRealmRole(capturedClaims, "user") {
		t.Error("expected user to have 'user' realm role")
	}
}

// TestAuthMiddleware_ExpiredToken tests rejection of expired tokens
func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	originalMode := authMode
	defer func() { authMode = originalMode }()

	authMode = "enforce"
	setupTestJWKS()

	// Create expired token
	claims := jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := createTestToken(claims, true)

	handler := AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for expired token, got %d", rec.Code)
	}
}

// TestAuthMiddleware_InvalidSignature tests rejection of tokens with invalid signature
func TestAuthMiddleware_InvalidSignature(t *testing.T) {
	originalMode := authMode
	defer func() { authMode = originalMode }()

	authMode = "enforce"
	setupTestJWKS()

	// Create token with invalid signature
	claims := jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := createTestToken(claims, false) // false = invalid signature

	handler := AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for invalid signature, got %d", rec.Code)
	}
}


// TestRequireRole_HasRole tests RequireRole when user has the required role
func TestRequireRole_HasRole(t *testing.T) {
	originalMode := authMode
	originalClientID := keycloakClientID
	defer func() {
		authMode = originalMode
		keycloakClientID = originalClientID
	}()

	authMode = "enforce"
	keycloakClientID = "test-client"
	setupTestJWKS()

	// Create token with admin role
	claims := jwt.MapClaims{
		"sub": "user-123",
		"azp": "test-client",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"resource_access": map[string]interface{}{
			"dwhlanding": map[string]interface{}{
				"roles": []interface{}{"admin"},
			},
		},
	}
	token := createTestToken(claims, true)

	handler := RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Admin access granted"))
	})

	req := httptest.NewRequest("GET", "/api/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

// TestRequireRole_MissingRole tests RequireRole when user lacks the required role
func TestRequireRole_MissingRole(t *testing.T) {
	originalMode := authMode
	originalClientID := keycloakClientID
	defer func() {
		authMode = originalMode
		keycloakClientID = originalClientID
	}()

	authMode = "enforce"
	keycloakClientID = "test-client"
	setupTestJWKS()

	// Create token with only viewer role (not admin)
	claims := jwt.MapClaims{
		"sub": "user-123",
		"azp": "test-client",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"resource_access": map[string]interface{}{
			"dwhlanding": map[string]interface{}{
				"roles": []interface{}{"viewer"},
			},
		},
	}
	token := createTestToken(claims, true)

	handler := RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for missing role, got %d", rec.Code)
	}
}

// TestRequireRole_RealmRole tests that realm roles are also checked
func TestRequireRole_RealmRole(t *testing.T) {
	originalMode := authMode
	originalClientID := keycloakClientID
	defer func() {
		authMode = originalMode
		keycloakClientID = originalClientID
	}()

	authMode = "enforce"
	keycloakClientID = "test-client"
	setupTestJWKS()

	// Create token with admin as realm role (not client role)
	claims := jwt.MapClaims{
		"sub": "user-123",
		"azp": "test-client",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"realm_access": map[string]interface{}{
			"roles": []interface{}{"admin"},
		},
	}
	token := createTestToken(claims, true)

	handler := RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for realm role, got %d", rec.Code)
	}
}

// TestJWKToRSAPublicKey tests the JWK to RSA conversion
func TestJWKToRSAPublicKey(t *testing.T) {
	// Create JWK from test public key
	nBytes := testPublicKey.N.Bytes()
	eBytes := big.NewInt(int64(testPublicKey.E)).Bytes()

	jwk := JWK{
		Kid: "test",
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(nBytes),
		E:   base64.RawURLEncoding.EncodeToString(eBytes),
	}

	// Convert back to RSA
	pubKey, err := jwkToRSAPublicKey(jwk)
	if err != nil {
		t.Fatalf("failed to convert JWK: %v", err)
	}

	// Verify it matches original
	if pubKey.N.Cmp(testPublicKey.N) != 0 {
		t.Error("modulus mismatch")
	}
	if pubKey.E != testPublicKey.E {
		t.Errorf("exponent mismatch: got %d, expected %d", pubKey.E, testPublicKey.E)
	}
}

// TestGetUserClaims_NoClaims tests GetUserClaims when no claims in context
func TestGetUserClaims_NoClaims(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/test", nil)

	claims := GetUserClaims(req)
	if claims != nil {
		t.Error("expected nil claims for request without context")
	}
}

// TestHasClientRole tests the hasClientRole helper
func TestHasClientRole(t *testing.T) {
	claims := &UserClaims{
		ClientRoles: map[string][]string{
			"client1": {"admin", "user"},
			"client2": {"viewer"},
		},
	}

	tests := []struct {
		client   string
		role     string
		expected bool
	}{
		{"client1", "admin", true},
		{"client1", "user", true},
		{"client1", "superadmin", false},
		{"client2", "viewer", true},
		{"client2", "admin", false},
		{"client3", "admin", false},
	}

	for _, tc := range tests {
		result := hasClientRole(claims, tc.client, tc.role)
		if result != tc.expected {
			t.Errorf("hasClientRole(%s, %s) = %v, expected %v",
				tc.client, tc.role, result, tc.expected)
		}
	}
}

// TestHasRealmRole tests the hasRealmRole helper
func TestHasRealmRole(t *testing.T) {
	claims := &UserClaims{
		RealmRoles: []string{"admin", "user"},
	}

	tests := []struct {
		role     string
		expected bool
	}{
		{"admin", true},
		{"user", true},
		{"superadmin", false},
	}

	for _, tc := range tests {
		result := hasRealmRole(claims, tc.role)
		if result != tc.expected {
			t.Errorf("hasRealmRole(%s) = %v, expected %v",
				tc.role, result, tc.expected)
		}
	}
}

// TestPublishBlocked_ProductionMode tests that publishing is blocked when AUTH_MODE is "on"
func TestPublishBlocked_ProductionMode(t *testing.T) {
	originalMode := authMode
	defer func() { authMode = originalMode }()

	authMode = "on"

	req := httptest.NewRequest("POST", "/api/report/publish", nil)
	rec := httptest.NewRecorder()

	PublishReportHandler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for publish in production mode, got %d", rec.Code)
	}

	// Verify error message
	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "publishing disabled in production mode" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

// TestPublishAllowed_DevMode tests that publishing works when AUTH_MODE is "off"
func TestPublishAllowed_DevMode(t *testing.T) {
	originalMode := authMode
	defer func() { authMode = originalMode }()

	authMode = "off"

	// Send a minimal publish request (will fail validation but not get 403)
	req := httptest.NewRequest("POST", "/api/report/publish", nil)
	rec := httptest.NewRecorder()

	PublishReportHandler(rec, req)

	// Should not be 403 - may be 400 (bad request) due to missing body
	if rec.Code == http.StatusForbidden {
		t.Errorf("expected request to not be forbidden in dev mode, got 403")
	}
}

// TestAuthConfig_CanPublish tests that canPublish is set correctly in auth config
func TestAuthConfig_CanPublish(t *testing.T) {
	tests := []struct {
		mode       string
		canPublish bool
	}{
		{"off", true},
		{"on", false},
	}

	for _, tc := range tests {
		t.Run("mode="+tc.mode, func(t *testing.T) {
			originalMode := authMode
			defer func() { authMode = originalMode }()

			authMode = tc.mode

			req := httptest.NewRequest("GET", "/api/auth/config", nil)
			rec := httptest.NewRecorder()

			GetAuthConfigHandler(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rec.Code)
			}

			var response AuthConfigResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.CanPublish != tc.canPublish {
				t.Errorf("expected canPublish=%v for mode=%s, got %v",
					tc.canPublish, tc.mode, response.CanPublish)
			}
		})
	}
}
