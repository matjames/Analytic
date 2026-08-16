package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"knowledgeportal/pkg/store"

	"github.com/matjames/statgate-lib/auth"
)

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// actorTenant returns the tenant id from an authenticated request context.
func actorTenant(r *http.Request) string {
	if t, ok := r.Context().Value(auth.ContextKeyTenant).(string); ok && t != "" {
		return t
	}
	if u, ok := auth.GetUserFromContext(r.Context()); ok && u != nil {
		return u.TenantID
	}
	return "default"
}

func actorID(r *http.Request) string {
	if u, ok := auth.GetUserFromContext(r.Context()); ok && u != nil {
		return u.UserID
	}
	return ""
}

// publicTenant resolves the tenant for public endpoints from a query param.
func publicTenant(r *http.Request) string {
	if t := strings.TrimSpace(r.URL.Query().Get("tenant_id")); t != "" {
		return t
	}
	return "default"
}

// ─── Probes ────────────────────────────────────────────────────────────────

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": "unhealthy", "service": "knowledge-portal", "db": "disconnected"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "healthy", "service": "knowledge-portal", "db": "connected"})
}

func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
