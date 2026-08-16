package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/matjames/statgate-lib/auth"
)

type userContextKey string

const (
	UserCtxKey   userContextKey = "statgate_user"
	TenantCtxKey userContextKey = "statgate_tenant"
)

// validator returns the shared statgate-lib Auth Validator enforcing
// Registry-issued JWTs with zero hardcoded defaults (fail-closed).
func validator() (*auth.Validator, error) {
	return auth.NewValidator("", "", "")
}

// isPublicPath reports whether a probe endpoint is exempt from authentication.
// All /api/v1 AI endpoints are authenticated; /metrics is exempt so the
// platform monitoring stack can scrape it.
func isPublicPath(path string) bool {
	switch path {
	case "/health", "/ready", "/live", "/metrics",
		"/api/v1/health", "/api/v1/ready", "/api/v1/live", "/api/v1/metrics":
		return true
	}
	return false
}

// CORSMiddleware handles cross-origin requests. Set AIENG_CORS_ORIGINS to an
// explicit allow-list for production.
func CORSMiddleware(next http.Handler) http.Handler {
	allowed := "*"
	if v := strings.TrimSpace(os.Getenv("AIENG_CORS_ORIGINS")); v != "" {
		allowed = v
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowed)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Tenant-ID, X-Request-ID, X-Correlation-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware emits structured request logs.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("[ai-autonomy] method=%s path=%s status=%d duration=%s req=%s",
			r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond), r.Header.Get("X-Request-ID"))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// AuthMiddleware validates Registry JWTs using statgate-lib/auth and enforces
// tenant isolation.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		val, err := validator()
		if err != nil {
			http.Error(w, `{"error":"service_misconfigured","message":"STATGATE_REGISTRY_JWT_SECRET is not configured"}`, http.StatusServiceUnavailable)
			return
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

		uCtx, err := val.ValidateToken(strings.TrimSpace(parts[1]))
		if err != nil {
			http.Error(w, `{"error":"unauthorized","message":"Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Tenant header isolation: an explicit X-Tenant-ID must match the token.
		headerTenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		if headerTenant != "" && uCtx.TenantID != "" && !strings.EqualFold(headerTenant, uCtx.TenantID) {
			http.Error(w, `{"error":"cross_tenant_violation","message":"Header tenant mismatch with token"}`, http.StatusForbidden)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, auth.ContextKeyUser, uCtx)
		ctx = context.WithValue(ctx, auth.ContextKeyTenant, uCtx.TenantID)
		ctx = context.WithValue(ctx, auth.ContextKeyOrg, uCtx.OrgID)
		ctx = context.WithValue(ctx, auth.ContextKeyRole, uCtx.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}