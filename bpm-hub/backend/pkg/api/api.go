package api

import (
	"github.com/gorilla/mux"
	"github.com/matjames/statgate-lib/tenant"
)

// RegisterRoutes mounts Business Process Management (App 11) endpoints.
// Probes: /health /ready /live /metrics (public). All /api/v1/* require JWT.
func RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", HealthHandler).Methods("GET")
	r.HandleFunc("/ready", ReadyHandler).Methods("GET")
	r.HandleFunc("/live", ReadyHandler).Methods("GET")
	r.Handle("/metrics", MetricsHandler())

	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(CORSMiddleware)
	api.Use(LoggingMiddleware)
	api.Use(AuthMiddleware)

	// Stage 2: workspace context + Enterprise Core membership enforcement.
	api.Use(tenant.WorkspaceContext)
	api.Use(tenant.WorkspaceMembership("", nil))

	// ── BPM: process definitions & workflow engine (P48) ──
	api.HandleFunc("/processes", ListProcessesHandler).Methods("GET")
	api.HandleFunc("/processes", CreateProcessHandler).Methods("POST")
	api.HandleFunc("/processes/{id}", GetProcessHandler).Methods("GET")
	api.HandleFunc("/processes/{id}/publish", PublishProcessHandler).Methods("POST")
	api.HandleFunc("/processes/{id}/instances", StartInstanceHandler).Methods("POST")

	// ── Instances ──
	api.HandleFunc("/instances", ListInstancesHandler).Methods("GET")
	api.HandleFunc("/instances/{id}", GetInstanceHandler).Methods("GET")
	api.HandleFunc("/instances/{id}/advance", AdvanceInstanceHandler).Methods("POST")
	api.HandleFunc("/instances/{id}/cancel", CancelInstanceHandler).Methods("POST")
	api.HandleFunc("/instances/{id}/tasks", ListInstanceTasksHandler).Methods("GET")
	api.HandleFunc("/instances/{id}/log", ListInstanceLogHandler).Methods("GET")

	// ── Case management (P48) ──
	api.HandleFunc("/cases", ListCasesHandler).Methods("GET")
	api.HandleFunc("/cases", CreateCaseHandler).Methods("POST")
	api.HandleFunc("/cases/{id}", GetCaseHandler).Methods("GET")
	api.HandleFunc("/cases/{id}", UpdateCaseHandler).Methods("PUT")
	api.HandleFunc("/cases/{id}", DeleteCaseHandler).Methods("DELETE")
	api.HandleFunc("/cases/{id}/items", ListCaseItemsHandler).Methods("GET")
	api.HandleFunc("/cases/{id}/items", AddCaseItemHandler).Methods("POST")
	api.HandleFunc("/cases/{id}/link", LinkCaseHandler).Methods("POST")

	// ── Task management (P48) ──
	api.HandleFunc("/tasks", ListTasksHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}", GetTaskHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}", UpdateTaskHandler).Methods("PUT")

	// ── Automation service (P48) ──
	api.HandleFunc("/automation", ListAutomationHandler).Methods("GET")
	api.HandleFunc("/automation/rules", CreateAutomationHandler).Methods("POST")
	api.HandleFunc("/automation/rules/{id}/toggle", ToggleAutomationHandler).Methods("POST")
	api.HandleFunc("/automation/evaluate", EvaluateAutomationHandler).Methods("POST")

	// ── Process mining / operational analytics (P48) ──
	api.HandleFunc("/mining/summary", MiningSummaryHandler).Methods("GET")

	// ── Cross-app object linkage + overview ──
	api.HandleFunc("/links", CreateLinkHandler).Methods("POST")
	api.HandleFunc("/links", ListLinksHandler).Methods("GET")
	api.HandleFunc("/summary", SummaryHandler).Methods("GET")
}