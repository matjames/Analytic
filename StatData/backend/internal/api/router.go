package api

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/health"
	"github.com/matjames/statgate-lib/metrics"
	"github.com/matjames/statgate-lib/tenant"
)

// RouterConfig holds initialization requirements for the Gin router
type RouterConfig struct {
	Handlers      *Handlers
	HealthChecker *health.Checker
	Metrics       *metrics.Metrics
	AuthValidator *auth.Validator
	CORSOrigin    string
	Env           string
}

// SetupRouter initializes the Gin engine with all routes, middleware, and metrics
func SetupRouter(cfg RouterConfig) *gin.Engine {
	if strings.EqualFold(cfg.Env, "production") {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// 1. Prometheus Metrics Middleware
	if cfg.Metrics != nil {
		r.Use(cfg.Metrics.GinMiddleware())
	}

	// 2. CORS Policy
	corsCfg := cors.DefaultConfig()
	if cfg.CORSOrigin == "*" || cfg.CORSOrigin == "" {
		corsCfg.AllowAllOrigins = true
	} else {
		corsCfg.AllowOrigins = strings.Split(cfg.CORSOrigin, ",")
	}
	corsCfg.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID", "X-Workspace-ID", "X-Request-ID", "X-User-ID"}
	corsCfg.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsCfg.AllowCredentials = true
	r.Use(cors.New(corsCfg))

	// 3. Health, Readiness & Metrics Endpoints (Public)
	if cfg.HealthChecker != nil {
		cfg.HealthChecker.RegisterGinRoutes(r)
	}
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	// Root Welcome Route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":     "StatData (App 12)",
			"description": "Enterprise Data Engineering, Scientific Computing & Universal Semantic Search Platform",
			"phases": []string{
				"Phase 37: Enterprise Data Engineering, DataOps & Modern Data Platform",
				"Phase 38: Scientific Computing, Statistical Computing & High-Performance Analytics",
				"Phase 47: Enterprise Search, Discovery, Universal Indexing & Semantic Retrieval",
			},
			"status": "OPERATIONAL",
			"docs":   "/api/v1/data/catalog/datasets",
		})
	})

	// 4. Authenticated API Group
	api := r.Group("/api/v1")
	if cfg.AuthValidator != nil {
		api.Use(cfg.AuthValidator.GinMiddleware())
		api.Use(tenant.GinTenantIsolation())
	}
	// Stage 2: workspace context propagation + membership enforcement via
	// Enterprise Core (no-op when no workspace is selected or unauthenticated).
	api.Use(tenant.GinWorkspaceContext())
	api.Use(tenant.GinWorkspaceMembership("", nil))

	h := cfg.Handlers

	// ─── Phase 37: Data Engineering & Catalog Routes ──────────────────────────
	data := api.Group("/data")
	{
		// Central Data Catalog & Datasets
		data.GET("/catalog/datasets", h.CatalogHandler.ListDatasets)
		data.POST("/catalog/datasets", h.CatalogHandler.CreateDataset)
		data.GET("/catalog/datasets/:id", h.CatalogHandler.GetDataset)
		data.PUT("/catalog/datasets/:id", h.CatalogHandler.UpdateDataset)
		data.DELETE("/catalog/datasets/:id", h.CatalogHandler.DeleteDataset)

		// Schema Registry & Data Contracts
		data.GET("/schemas", h.CatalogHandler.ListSchemas)
		data.POST("/schemas", h.CatalogHandler.RegisterSchema)
		data.POST("/schemas/validate", h.CatalogHandler.ValidatePayloadAgainstSchema)

		// Automated Data Quality & Anomaly Detection
		data.POST("/quality/evaluate", h.CatalogHandler.EvaluateQuality)
		data.GET("/quality/rules", h.CatalogHandler.ListQualityRules)
		data.POST("/quality/rules", h.CatalogHandler.CreateQualityRule)

		// Data Lineage & Dependency Graph
		data.GET("/lineage/:id", h.CatalogHandler.GetLineageGraph)

		// Data Pipelines & DAG Orchestration
		data.GET("/pipelines", h.PipelineHandler.ListPipelines)
		data.POST("/pipelines", h.PipelineHandler.CreatePipeline)
		data.GET("/pipelines/:id", h.PipelineHandler.GetPipeline)
		data.POST("/pipelines/:id/run", h.PipelineHandler.TriggerPipelineRun)
		data.GET("/pipelines/:id/runs", h.PipelineHandler.ListPipelineRuns)

		// Real-time Streaming & CDC
		data.GET("/streaming", h.PipelineHandler.ListStreamingJobs)
		data.POST("/streaming", h.PipelineHandler.CreateStreamingJob)
		data.POST("/cdc/events", h.PipelineHandler.IngestCDCEvent)

		// Feature Store
		data.GET("/features/views", h.FeatureStoreHandler.ListFeatureViews)
		data.POST("/features/views", h.FeatureStoreHandler.CreateFeatureView)
		data.POST("/features/ingest", h.FeatureStoreHandler.IngestFeatures)
		data.GET("/features/online", h.FeatureStoreHandler.GetOnlineFeatures)

		// Object Links & Compliance Logs
		data.GET("/links", h.LinkHandler.GetObjectLinks)
		data.POST("/links", h.LinkHandler.CreateObjectLink)
		data.GET("/audit/logs", h.LinkHandler.ListAuditLogs)
	}

	// ─── Phase 38: Scientific Computing & ML Routes ───────────────────────────
	scienceGroup := api.Group("/science")
	{
		// Interactive Notebook Workspaces
		scienceGroup.GET("/notebooks", h.ScienceHandler.ListNotebooks)
		scienceGroup.POST("/notebooks", h.ScienceHandler.CreateNotebook)
		scienceGroup.POST("/notebooks/:id/execute", h.ScienceHandler.ExecuteNotebookCell)

		// Scientific Experiment Tracking
		scienceGroup.GET("/experiments", h.ScienceHandler.ListExperiments)
		scienceGroup.POST("/experiments", h.ScienceHandler.CreateExperiment)
		scienceGroup.POST("/experiments/:id/runs", h.ScienceHandler.StartExperimentRun)
		scienceGroup.POST("/runs/:id/metrics", h.ScienceHandler.LogMetrics)

		// Model Registry & Lifecycle Governance
		scienceGroup.GET("/models", h.ScienceHandler.ListModels)
		scienceGroup.POST("/models", h.ScienceHandler.CreateModel)
		scienceGroup.POST("/models/:id/versions", h.ScienceHandler.RegisterModelVersion)
		scienceGroup.POST("/versions/:id/stage", h.ScienceHandler.TransitionModelStage)

		// Compute Cluster & Resource Allocation
		scienceGroup.GET("/compute/nodes", h.ScienceHandler.ListComputeNodes)
		scienceGroup.POST("/compute/jobs", h.ScienceHandler.SubmitComputeJob)
	}

	// ─── Phase 47: Enterprise Search & Semantic Retrieval Routes ──────────────
	searchGroup := api.Group("/search")
	{
		// Universal & Hybrid Search
		searchGroup.POST("/query", h.SearchHandler.Search)
		searchGroup.POST("/hybrid", h.SearchHandler.Search)
		searchGroup.GET("/suggest", h.SearchHandler.AutoComplete)

		// Indexing & Document Management
		searchGroup.POST("/index", h.SearchHandler.IndexDocument)

		// Saved Searches & Alerts
		searchGroup.GET("/saved", h.SearchHandler.ListSavedSearches)
		searchGroup.POST("/saved", h.SearchHandler.CreateSavedSearch)
	}

	return r
}
