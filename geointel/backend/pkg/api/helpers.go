package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"geointel/pkg/store"

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

func varsOf(r *http.Request) map[string]string { return mux.Vars(r) }

func queryFloat(r *http.Request, key string, def float64) float64 {
	if v := r.URL.Query().Get(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func queryInt(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

// ─── Probes & Metrics ────────────────────────────────────────────────────────

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": "unhealthy", "service": "geointel", "db": "disconnected"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "healthy", "service": "geointel", "db": "connected"})
}

func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func MetricsHandler() http.Handler { return metrics.Handler() }

// f2s formats a float for reverse-geocode labels.
func f2s(v float64) string { return strconv.FormatFloat(v, 'f', 6, 64) }

// deterministicGeocode maps a query string to a stable coordinate using its
// hash — a dependency-free placeholder for a real geocoder (P44).
func deterministicGeocode(query string) (lat, lng float64, confidence float64) {
	h := 0
	for _, c := range query {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	lat = -11.0 + float64(h%1000)/1000
	lng = 27.0 + float64((h/1000)%1000)/1000
	return lat, lng, 0.6
}