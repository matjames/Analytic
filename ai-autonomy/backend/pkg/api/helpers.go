package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"aiengines/pkg/store"

	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/metrics"
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

// ─── Probes & Metrics ────────────────────────────────────────────────────────

// HealthHandler reports service + database health.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": "unhealthy", "service": "ai-autonomy", "db": "disconnected"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "healthy", "service": "ai-autonomy", "db": "connected"})
}

// ReadyHandler returns the readiness probe.
func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// MetricsHandler exposes Prometheus metrics (`/metrics`) using the shared
// statgate-lib/metrics independent of a Gin engine.
func MetricsHandler() http.Handler {
	return metrics.Handler()
}

// helper for the event consumer: most inbound payloads carry an object id.
func payloadString(p map[string]interface{}, key string) string {
	if p == nil {
		return ""
	}
	if v, ok := p[key]; ok {
		switch t := v.(type) {
		case string:
			return strings.TrimSpace(t)
		default:
			return ""
		}
	}
	return ""
}