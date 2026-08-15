package main

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

	"github.com/gin-gonic/gin"
)

// ── Helpers ─────────────────────────────────────────────────────────

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func signTestJWT(secret, alg string, claims map[string]interface{}) string {
	header, _ := json.Marshal(map[string]string{"alg": alg, "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	if alg == "none" {
		return b64url(header) + "." + b64url(payload) + "."
	}
	input := b64url(header) + "." + b64url(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return input + "." + b64url(mac.Sum(nil))
}

func canonicalClaims(exp time.Time) map[string]interface{} {
	now := time.Now()
	return map[string]interface{}{
		"sub":       "user-42",
		"tenant_id": "tenant-alpha",
		"org_id":    "org-national",
		"role":      "analyst",
		"email":     "user@example.org",
		"iat":       now.Unix(),
		"nbf":       now.Add(-30 * time.Second).Unix(),
		"exp":       exp.Unix(),
		"iss":       "statgate-registry",
		"aud":       "statgate",
	}
}

func testSecret(t *testing.T) {
	t.Helper()
	prev, had := os.LookupEnv("STATGATE_REGISTRY_JWT_SECRET")
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", "s3cret-key-0123456789abcdef")
	t.Cleanup(func() {
		if had {
			os.Setenv("STATGATE_REGISTRY_JWT_SECRET", prev)
		} else {
			os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")
		}
	})
}

func buildSecureRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(jwtAuthMiddleware(), tenantIsolationMiddleware())
	r.GET("/api/secure", func(c *gin.Context) {
		c.JSON(200, gin.H{"user": c.GetString("user_id"), "tenant": c.GetString("tenant_id"), "role": c.GetString("role")})
	})
	return r
}

func doSecure(r *gin.Engine, token, tenantHdr string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("GET", "/api/secure", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if tenantHdr != "" {
		req.Header.Set("X-Tenant-ID", tenantHdr)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── parseJWT unit tests (pure signing/claims enforcement) ──────────

func TestParseJWT_ValidToken(t *testing.T) {
	tok := signTestJWT("s3cret!", "HS256", canonicalClaims(time.Now().Add(1*time.Hour)))
	claims, err := parseJWT(tok, "s3cret!")
	if err != nil {
		t.Fatalf("expected valid token, got: %v", err)
	}
	if claims.TenantID != "tenant-alpha" || claims.Sub != "user-42" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestParseJWT_ExpiredToken(t *testing.T) {
	tok := signTestJWT("s3cret!", "HS256", canonicalClaims(time.Now().Add(-1*time.Hour)))
	if _, err := parseJWT(tok, "s3cret!"); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestParseJWT_MissingExp(t *testing.T) {
	claims := canonicalClaims(time.Now().Add(1 * time.Hour))
	delete(claims, "exp")
	tok := signTestJWT("s3cret!", "HS256", claims)
	if _, err := parseJWT(tok, "s3cret!"); err == nil {
		t.Fatal("token without exp was accepted")
	}
}

func TestParseJWT_WrongIssuer(t *testing.T) {
	claims := canonicalClaims(time.Now().Add(1 * time.Hour))
	claims["iss"] = "evil-issuer"
	tok := signTestJWT("s3cret!", "HS256", claims)
	if _, err := parseJWT(tok, "s3cret!"); err == nil {
		t.Fatal("token with wrong issuer was accepted")
	}
}

func TestParseJWT_WrongAudience(t *testing.T) {
	claims := canonicalClaims(time.Now().Add(1 * time.Hour))
	claims["aud"] = "attacker-app"
	tok := signTestJWT("s3cret!", "HS256", claims)
	if _, err := parseJWT(tok, "s3cret!"); err == nil {
		t.Fatal("token with wrong audience was accepted")
	}
}

func TestParseJWT_UnsupportedAlgorithm(t *testing.T) {
	tok := signTestJWT("s3cret!", "RS256", canonicalClaims(time.Now().Add(1*time.Hour)))
	if _, err := parseJWT(tok, "s3cret!"); err == nil {
		t.Fatal("token with RS256 header was accepted")
	}
}

func TestParseJWT_NoneAlgorithm(t *testing.T) {
	tok := signTestJWT("", "none", canonicalClaims(time.Now().Add(1*time.Hour)))
	if _, err := parseJWT(tok, "s3cret!"); err == nil {
		t.Fatal("unsigned token was accepted")
	}
}

func TestParseJWT_MalformedToken(t *testing.T) {
	if _, err := parseJWT("not-a-jwt", "s3cret!"); err == nil {
		t.Fatal("malformed token was accepted")
	}
	if _, err := parseJWT("a.b.c.d", "s3cret!"); err == nil {
		t.Fatal("4-part token was accepted")
	}
}

func TestParseJWT_WrongSignature(t *testing.T) {
	tok := signTestJWT("other-secret", "HS256", canonicalClaims(time.Now().Add(1*time.Hour)))
	if _, err := parseJWT(tok, "s3cret!"); err == nil {
		t.Fatal("token signed by another key was accepted")
	}
}

func TestParseJWT_MissingTenant(t *testing.T) {
	claims := canonicalClaims(time.Now().Add(1 * time.Hour))
	delete(claims, "tenant_id")
	tok := signTestJWT("s3cret!", "HS256", claims)
	if _, err := parseJWT(tok, "s3cret!"); err == nil {
		t.Fatal("token without tenant_id was accepted")
	}
}

func TestParseJWT_UnknownRole(t *testing.T) {
	claims := canonicalClaims(time.Now().Add(1 * time.Hour))
	claims["role"] = "root"
	tok := signTestJWT("s3cret!", "HS256", claims)
	if _, err := parseJWT(tok, "s3cret!"); err == nil {
		t.Fatal("token with unknown role was accepted")
	}
}

func TestParseJWT_NotYetValid(t *testing.T) {
	claims := canonicalClaims(time.Now().Add(1 * time.Hour))
	claims["nbf"] = time.Now().Add(1 * time.Hour).Unix()
	tok := signTestJWT("s3cret!", "HS256", claims)
	if _, err := parseJWT(tok, "s3cret!"); err == nil {
		t.Fatal("token with future nbf was accepted")
	}
}
// ── middleware-level tests (HTTP-aware: 401 / 503 / 403 semantics) ──

func TestJWTAuthMiddleware_MissingSecretFailsClosed(t *testing.T) {
	prev, had := os.LookupEnv("STATGATE_REGISTRY_JWT_SECRET")
	os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")
	t.Cleanup(func() {
		if had {
			os.Setenv("STATGATE_REGISTRY_JWT_SECRET", prev)
		} else {
			os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")
		}
	})
	// Router built while the secret is absent -> middleware is fail-closed.
	r := buildSecureRouter()
	w := doSecure(r, "anything", "")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when secret unset, got %d", w.Code)
	}
}

func TestJWTAuthMiddleware_ValidTokenAccepted(t *testing.T) {
	testSecret(t)
	r := buildSecureRouter()
	tok := signTestJWT("s3cret-key-0123456789abcdef", "HS256", canonicalClaims(time.Now().Add(1*time.Hour)))
	w := doSecure(r, tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid token, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestJWTAuthMiddleware_ExpiredRejected(t *testing.T) {
	testSecret(t)
	r := buildSecureRouter()
	tok := signTestJWT("s3cret-key-0123456789abcdef", "HS256", canonicalClaims(time.Now().Add(-1*time.Hour)))
	if w := doSecure(r, tok, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestJWTAuthMiddleware_WrongAudienceRejected(t *testing.T) {
	testSecret(t)
	r := buildSecureRouter()
	claims := canonicalClaims(time.Now().Add(1 * time.Hour))
	claims["aud"] = "attacker-app"
	tok := signTestJWT("s3cret-key-0123456789abcdef", "HS256", claims)
	if w := doSecure(r, tok, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong audience, got %d", w.Code)
	}
}

func TestJWTAuthMiddleware_WrongAlgorithmRejected(t *testing.T) {
	testSecret(t)
	r := buildSecureRouter()
	tok := signTestJWT("s3cret-key-0123456789abcdef", "HS512", canonicalClaims(time.Now().Add(1*time.Hour)))
	if w := doSecure(r, tok, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for non-HS256 alg, got %d", w.Code)
	}
}

func TestJWTAuthMiddleware_DemoTokenRejected(t *testing.T) {
	testSecret(t)
	r := buildSecureRouter()
	if w := doSecure(r, "demo_token", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for demo token, got %d", w.Code)
	}
}

func TestJWTAuthMiddleware_NoIdentityHeaderTrust(t *testing.T) {
	testSecret(t)
	r := buildSecureRouter()
	req, _ := http.NewRequest("GET", "/api/secure", nil)
	req.Header.Set("X-User-ID", "user-999")
	req.Header.Set("X-Tenant-ID", "tenant-other")
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when identity headers are spoofed without a token, got %d", w.Code)
	}
}

func TestTenantIsolation_MismatchRejected(t *testing.T) {
	testSecret(t)
	r := buildSecureRouter()
	tok := signTestJWT("s3cret-key-0123456789abcdef", "HS256", canonicalClaims(time.Now().Add(1*time.Hour)))
	// Header tenant differs from the JWT tenant -> 403.
	if w := doSecure(r, tok, "tenant-other"); w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for mismatched X-Tenant-ID, got %d", w.Code)
	}
}

func TestTenantIsolation_MatchAccepted(t *testing.T) {
	testSecret(t)
	r := buildSecureRouter()
	tok := signTestJWT("s3cret-key-0123456789abcdef", "HS256", canonicalClaims(time.Now().Add(1*time.Hour)))
	if w := doSecure(r, tok, "tenant-alpha"); w.Code != http.StatusOK {
		t.Fatalf("expected 200 for matching X-Tenant-ID, got %d", w.Code)
	}
}