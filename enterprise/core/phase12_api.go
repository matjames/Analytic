package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — API NAMESPACES (directive §24)
//
//   /api/intelligence/   condition, signals, overview
//   /api/graph/          registry objects, edges, NAMED queries only
//   /api/objectives/     strategic objective projection
//   /api/kpis/           KPI framework + measurements
//   /api/risks/          risk intelligence events (never an authoritative register)
//   /api/ai/             governed AI recommendations + audit
//
// These routes are registered INSIDE the existing JWT + tenant-isolation
// middleware group, so every endpoint has authentication, authorization,
// tenant enforcement and rate limiting. Identity ALWAYS comes from the
// verified JWT context (getContextUserID / getContextTenantID) — never from
// client-supplied X-User-ID / X-Tenant-ID headers.
//
// Graph mutation and AI authorization are privileged: only `admin` and
// `institutional_lead` roles may perform them (directive §8, §19).
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"log"

	"github.com/gin-gonic/gin"
)

// requirePhase12Privilege enforces the privileged Phase XII roles.
func requirePhase12Privilege(c *gin.Context) bool {
	return requireRole(c, "admin", "institutional_lead")
}

// initPhase12 initialises the Phase XII intelligence layer. Missing security
// configuration fails closed: if an external provider is requested but not
// configured, the deterministic local provider remains active and no
// RESTRICTED/SENSITIVE data can leave the platform.
func initPhase12() {
	providerName := getEnvValue("STATGATE_AI_PROVIDER")
	switch providerName {
	case "":
		configuredProvider = nil
	case "openai":
		configuredProvider = newExternalProvider("openai", getEnvValue("OPENAI_API_KEY"), getEnvValue("STATGATE_AI_MODEL"))
	case "gemini":
		configuredProvider = newExternalProvider("gemini", getEnvValue("GEMINI_API_KEY"), getEnvValue("STATGATE_AI_MODEL"))
	default:
		configuredProvider = newExternalProvider(providerName, getEnvValue("STATGATE_AI_API_KEY"), getEnvValue("STATGATE_AI_MODEL"))
	}
	log.Printf("phase12: intelligence layer initialised (provider=%q, calculation_version=%s)", Phase12ProviderName(), conditionCalculationVersion)
}

// Phase12ProviderName returns the active reasoning provider name.
func Phase12ProviderName() string {
	if configuredProvider != nil {
		return configuredProvider.Name()
	}
	return "deterministic-local"
}

// registerPhase12Routes wires all Phase XII API namespaces onto the existing
// authenticated `/api` group.
func registerPhase12Routes(api *gin.RouterGroup) {
	intelligence := api.Group("/intelligence")
	{
		intelligence.GET("/overview", handleIntelligenceOverview)
		intelligence.GET("/condition", handleGetCondition)
		intelligence.GET("/condition/history", handleGetConditionHistory)
		intelligence.GET("/signals", handleListSignals)
		intelligence.POST("/signals", handleIngestSignal)
	}

	graph := api.Group("/graph")
	{
		graph.GET("/objects", handleListGraphObjects)
		graph.POST("/objects", handleProjectGraphObject)
		graph.GET("/edges", handleListGraphEdges)
		graph.POST("/edges", handleCreateGraphEdge)
		graph.DELETE("/edges/:id", handleDeleteGraphEdge)
		// Named queries ONLY (directive §27). Static route first.
		graph.GET("/queries/objectives-at-risk", handleNamedQueryObjectivesAtRisk)
		graph.GET("/queries/:name/:id", handleNamedQueryWithParam)
	}

	objectives := api.Group("/objectives")
	{
		objectives.GET("", handleListObjectives)
		objectives.POST("", handleCreateObjective)
	}

	kpis := api.Group("/kpis")
	{
		kpis.GET("", handlePhase12ListKPIs)
		kpis.POST("", handleCreateKPI)
		kpis.GET("/:id/measurements", handleListKPIMeasurements)
		kpis.POST("/:id/measurements", handleRecordKPIMeasurement)
	}

	risks := api.Group("/risks")
	{
		risks.GET("/events", handleListRiskEvents)
		risks.POST("/events", handleRecordRiskEvent)
		risks.POST("/detect", handleDetectEmergingRisk)
	}

	ai := api.Group("/ai")
	{
		ai.POST("/recommendations", handleGenerateRecommendation)
		ai.GET("/recommendations", handleListRecommendations)
		ai.POST("/recommendations/:id/review", handleReviewRecommendation)
		ai.POST("/recommendations/:id/execute", handleExecuteRecommendation)
		ai.GET("/audit", handleListAIAudit)
	}
}