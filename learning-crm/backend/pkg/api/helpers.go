package api

import (
	"encoding/json"
	"net/http"

	"learningcrm/pkg/store"

	"github.com/gorilla/mux"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/metrics"
	"github.com/matjames/statgate-lib/tenant"
)

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func actorTenant(r *http.Request) string {
	if t, ok := r.Context().Value(auth.ContextKeyTenant).(string); ok && t != "" {
		return t
	}
	if u, ok := auth.GetUserFromContext(r.Context()); ok && u != nil {
		return u.TenantID
	}
	return "default"
}

// actorWorkspace returns the selected workspace (Stage 2: identity/security
// closure) captured by tenant.WorkspaceContext middleware. Empty when none.
func actorWorkspace(r *http.Request) string {
	return tenant.WorkspaceIDFromRequest(r)
}

// workspaceAllowsRead reports whether an existing resource may be seen from
// the selected workspace. Legacy resources without a workspace (empty) remain
// tenant-wide visible.
func workspaceAllowsRead(existingWorkspace, selectedWorkspace string) bool {
	if selectedWorkspace == "" || existingWorkspace == "" {
		return true
	}
	return existingWorkspace == selectedWorkspace
}

func actorID(r *http.Request) string {
	if u, ok := auth.GetUserFromContext(r.Context()); ok && u != nil {
		return u.UserID
	}
	return ""
}

// ─── Probes & Metrics ────────────────────────────────────────────────────────

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": "unhealthy", "service": "learning-crm", "db": "disconnected"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "healthy", "service": "learning-crm", "db": "connected"})
}

func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// MetricsHandler exposes Prometheus metrics (`/metrics`) from statgate-lib.
func MetricsHandler() http.Handler {
	return metrics.Handler()
}

// varsOf returns route variables.
func varsOf(r *http.Request) map[string]string {
	return mux.Vars(r)
}