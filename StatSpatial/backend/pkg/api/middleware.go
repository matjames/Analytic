package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type userContextKey string

const (
	UserCtxKey   userContextKey = "statgate_user"
	TenantCtxKey userContextKey = "statgate_tenant"
)

type UserContext struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
}

// CORSMiddleware handles cross-origin requests.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Tenant-ID, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs request metadata.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		_ = duration
	})
}

// AuthMiddleware validates Registry JWT token with zero-default secret enforcement.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Allow health/readiness endpoints
		if path == "/health" || path == "/ready" || path == "/live" || path == "/api/v1/health" || path == "/api/v1/ready" {
			next.ServeHTTP(w, r)
			return
		}

		secret := strings.TrimSpace(os.Getenv("STATGATE_REGISTRY_JWT_SECRET"))
		if secret == "" {
			// Fail-closed in strict/production mode
			if strings.EqualFold(os.Getenv("STATGATE_ENV"), "production") {
				http.Error(w, `{"error":"service_misconfigured","message":"STATGATE_REGISTRY_JWT_SECRET is not configured"}`, http.StatusServiceUnavailable)
				return
			}
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"unauthorized","message":"Missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"unauthorized","message":"Invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		uCtx, err := validateJWT(tokenStr, secret)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":%q}`, err.Error()), http.StatusUnauthorized)
			return
		}

		// Tenant header isolation check
		headerTenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		if headerTenant != "" && uCtx.TenantID != "" && !strings.EqualFold(headerTenant, uCtx.TenantID) {
			http.Error(w, `{"error":"cross_tenant_violation","message":"Header tenant mismatch with token"}`, http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), UserCtxKey, uCtx)
		ctx = context.WithValue(ctx, TenantCtxKey, uCtx.TenantID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validateJWT(tokenStr, secret string) (*UserContext, error) {
	if strings.HasPrefix(tokenStr, "mock_") || strings.HasPrefix(tokenStr, "demo_") {
		return nil, errors.New("mock tokens are rejected")
	}

	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed JWT")
	}

	if secret != "" {
		sig, err := base64.RawURLEncoding.DecodeString(parts[2])
		if err != nil {
			return nil, errors.New("invalid signature encoding")
		}

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(parts[0] + "." + parts[1]))
		expectedSig := mac.Sum(nil)

		if !hmac.Equal(sig, expectedSig) {
			return nil, errors.New("invalid signature")
		}
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, errors.New("invalid payload json")
	}

	// Check expiration
	if expVal, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(expVal) {
			return nil, errors.New("token has expired")
		}
	}

	userID := ""
	if u, ok := claims["userId"].(string); ok {
		userID = u
	} else if u, ok := claims["user_id"].(string); ok {
		userID = u
	} else if u, ok := claims["sub"].(string); ok {
		userID = u
	}

	if userID == "" {
		return nil, errors.New("missing user identity in token")
	}

	tenantID := "default"
	if t, ok := claims["tenant_id"].(string); ok && t != "" {
		tenantID = t
	} else if t, ok := claims["tenantId"].(string); ok && t != "" {
		tenantID = t
	}

	role := "viewer"
	if r, ok := claims["role"].(string); ok && r != "" {
		role = r
	}

	email := ""
	if e, ok := claims["email"].(string); ok {
		email = e
	}

	return &UserContext{
		UserID:   userID,
		Email:    email,
		TenantID: tenantID,
		Role:     role,
	}, nil
}
