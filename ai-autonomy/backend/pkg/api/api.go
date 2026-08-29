package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matjames/statgate-lib/tenant"
)

// RegisterRoutes mounts every AI & Autonomy endpoint.
//   - Probes: /health /ready /live /metrics (public)
//   - All /api/v1/* require a Registry-issued JWT (statgate-lib/auth)
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

	// ── P22: Digital Twins & Simulations ──
	api.HandleFunc("/twins", ListTwinsHandler).Methods("GET")
	api.HandleFunc("/twins", CreateTwinHandler).Methods("POST")
	api.HandleFunc("/twins/{id}", GetTwinHandler).Methods("GET")
	api.HandleFunc("/twins/{id}", UpdateTwinHandler).Methods("PUT")
	api.HandleFunc("/twins/{id}", DeleteTwinHandler).Methods("DELETE")
	api.HandleFunc("/twins/{id}/simulate", SimulateTwinHandler).Methods("POST")
	api.HandleFunc("/twins/{id}/simulations", ListTwinSimulationsHandler).Methods("GET")

	// ── P22: Models, Predictions & Decision Pipelines ──
	api.HandleFunc("/models", ListModelsHandler).Methods("GET")
	api.HandleFunc("/models", CreateModelHandler).Methods("POST")
	api.HandleFunc("/models/{id}", GetModelHandler).Methods("GET")
	api.HandleFunc("/models/{id}", DeleteModelHandler).Methods("DELETE")
	api.HandleFunc("/models/{id}/predict", PredictHandler).Methods("POST")
	api.HandleFunc("/predictions", ListPredictionsHandler).Methods("GET")
	api.HandleFunc("/predictions/{id}", GetPredictionHandler).Methods("GET")
	api.HandleFunc("/pipelines", ListPipelinesHandler).Methods("GET")
	api.HandleFunc("/pipelines", CreatePipelineHandler).Methods("POST")

	// ── P31: Agents, Tasks, Memory, Messages, Governance ──
	api.HandleFunc("/agents", ListAgentsHandler).Methods("GET")
	api.HandleFunc("/agents", CreateAgentHandler).Methods("POST")
	api.HandleFunc("/agents/{id}", GetAgentHandler).Methods("GET")
	api.HandleFunc("/agents/{id}", UpdateAgentHandler).Methods("PUT")
	api.HandleFunc("/agents/{id}/tasks", ListAgentTasksHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/tasks", CreateAgentTaskHandler).Methods("POST")
	api.HandleFunc("/agents/{id}/memory", ListAgentMemoryHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/memory", CreateAgentMemoryHandler).Methods("POST")
	api.HandleFunc("/agents/messages", SendAgentMessageHandler).Methods("POST")
	api.HandleFunc("/agents/messages", ListAgentMessagesHandler).Methods("GET")
	api.HandleFunc("/governance", ListGovernanceHandler).Methods("GET")
	api.HandleFunc("/governance/{id}/review", ReviewGovernanceHandler).Methods("POST")

	// ── P39: Knowledge Graph, Triplestore & Context Indexer ──
	api.HandleFunc("/graph/nodes", ListGraphNodesHandler).Methods("GET")
	api.HandleFunc("/graph/nodes", CreateGraphNodeHandler).Methods("POST")
	api.HandleFunc("/graph/nodes/{id}", GetGraphNodeHandler).Methods("GET")
	api.HandleFunc("/graph/nodes/{id}", DeleteGraphNodeHandler).Methods("DELETE")
	api.HandleFunc("/graph/neighbors/{id}", GraphNeighborsHandler).Methods("GET")
	api.HandleFunc("/graph/edges", ListGraphEdgesHandler).Methods("GET")
	api.HandleFunc("/graph/edges", CreateGraphEdgeHandler).Methods("POST")
	api.HandleFunc("/graph/edges/{id}", DeleteGraphEdgeHandler).Methods("DELETE")
	api.HandleFunc("/graph/statements", ListTriplesHandler).Methods("GET")
	api.HandleFunc("/graph/statements", CreateTripleHandler).Methods("POST")
	api.HandleFunc("/index/enrich", EnrichIndexHandler).Methods("POST")
	api.HandleFunc("/index/search", SearchIndexHandler).Methods("GET")

	// ── Cross-app object linkage ──
	api.HandleFunc("/links", CreateLinkHandler).Methods("POST")
	api.HandleFunc("/links", ListLinksHandler).Methods("GET")

	// ── Overview ──
	api.HandleFunc("/summary", SummaryHandler).Methods("GET")

	// Agent bus internal health (used by orchestration tooling).
	api.HandleFunc("/agents/runtime", AgentRuntimeStatsHandler).Methods("GET")
}

// muxVars is an alias used by handlers for readability.
func varsOf(r *http.Request) map[string]string {
	return mux.Vars(r)
}