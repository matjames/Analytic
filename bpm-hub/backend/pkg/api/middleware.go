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

func validator() (*auth.Validator, error) {
	return auth.NewValidator("", "", "")
}

func isPublicPath(path string) bool {
	switch path {
	case "/health", "/ready", "/live", "/metrics":
		return true
	}
	return false
}

func CORSMiddleware(next http.Handler) http.Handler {
	allowed := "*"
	if v := strings.TrimSpace(os.Getenv("BPM_CORS_ORIGINS")); v != "" {
		allowed = v
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowed)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Tenant-ID, X-Workspace-ID, X-Request-ID, X-Correlation-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("[bpm-hub] method=%s path=%s status=%d duration=%s",
			r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
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