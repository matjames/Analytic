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

func rmsTestToken(secret, role, tenant string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := map[string]interface{}{
		"sub":       "usr-1",
		"userId":    "usr-1",
		"role":      role,
		"tenant_id": tenant,
		"iss":       "statgate-registry",
		"aud":       "statgate",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	payload, _ := json.Marshal(claims)
	data := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return data + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestRegistryAuthMiddlewareConverged(t *testing.T) {
	secret := "test-rms-secret-32-chars-long-!!"
	os.Setenv("STATGATE_REGISTRY_JWT_SECRET", secret)
	defer os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(registryAuthMiddleware())
	r.GET("/api/v1/research", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "user": c.GetString("user_id"), "tenant": c.GetString("tenant_id")})
	})

	// 1. Unauthenticated -> 401
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api/v1/research", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 unauthenticated, got %d", rr.Code)
	}

	// 2. Valid Registry token -> 200
	tok := rmsTestToken(secret, "admin", "uganda-national")
	req := httptest.NewRequest("GET", "/api/v1/research", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 valid token, got %d", rr.Code)
	}

	// 3. Cross-tenant header mismatch -> 403
	req = httptest.NewRequest("GET", "/api/v1/research", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("X-Tenant-ID", "other-tenant")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 cross-tenant, got %d", rr.Code)
	}
}

func TestRegistryAuthMiddlewareFailClosed(t *testing.T) {
	os.Unsetenv("STATGATE_REGISTRY_JWT_SECRET")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(registryAuthMiddleware())
	r.GET("/api/v1/research", func(c *gin.Context) { c.Status(http.StatusOK) })

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api/v1/research", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when secret missing, got %d", rr.Code)
	}
}