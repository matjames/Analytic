package api

import (
	"context"
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
	Handlers          *Handlers
	DiscussionHandler *DiscussionHandler
	HealthChecker     *health.Checker
	Metrics           *metrics.Metrics
	AuthValidator     *auth.Validator
	CORSOrigin        string
	Env               string
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
	corsCfg.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID", "X-Workspace-ID", "X-Request-ID"}
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
			"service":     "StatFederation (App 6)",
			"description": "National Statistical System (NSS) & Global Digital Diplomacy Platform",
			"phases":      []string{"Phase 23: NSS Federation", "Phase 30: Global Diplomacy & Cross-Border Evidence"},
			"status":      "OPERATIONAL",
			"docs":        "/api/v1/federation/nodes",
		})
	})

	// 4. Authenticated API Group
	api := r.Group("/api/v1")
	if cfg.AuthValidator != nil {
		api.Use(cfg.AuthValidator.GinMiddleware())
		api.Use(tenant.GinTenantIsolation())
	}
	api.Use(func(c *gin.Context) {
		workspaceID := c.GetHeader("X-Workspace-ID")
		if len(workspaceID) > 128 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
			return
		}
		for i, r := range workspaceID {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && i > 0) || r == '_') {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
				return
			}
		}
		c.Set("workspace_id", workspaceID)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "workspace_id", workspaceID))
		c.Next()
	})
	if cfg.DiscussionHandler != nil {
		api.POST("/discussions", cfg.DiscussionHandler.Create)
	}

	h := cfg.Handlers

	// ─── NSS Federation Engine Routes ─────────────────────────────────────────
	fed := api.Group("/federation")
	{
		// Node Registry
		fed.GET("/nodes", h.ListNodes)
		fed.POST("/nodes", h.RegisterNode)
		fed.GET("/nodes/:id", h.GetNode)
		fed.POST("/nodes/:id/heartbeat", h.NodeHeartbeat)
		fed.POST("/nodes/probe", h.ProbeNodes)

		// Distributed Query Router
		fed.POST("/queries/dispatch", h.DispatchQuery)
		fed.GET("/queries", h.ListQueries)
		fed.GET("/queries/:id", h.GetQuery)

		// Data Sharing Agreements (DSAs)
		fed.GET("/dsa", h.ListDSAs)
		fed.POST("/dsa", h.CreateDSA)
		fed.GET("/dsa/:id", h.GetDSA)
		fed.POST("/dsa/:id/approve", h.ApproveDSA)
		fed.POST("/dsa/:id/revoke", h.RevokeDSA)

		// National Indicator Repository & Release Calendar
		fed.GET("/indicators", h.ListIndicators)
		fed.POST("/indicators", h.CreateIndicator)
		fed.GET("/indicators/:id", h.GetIndicator)
		fed.PUT("/indicators/:id/values", h.UpdateIndicatorValue)
		fed.POST("/indicators/push", h.PushIndicators)
		fed.POST("/indicators/ingest", h.IngestIndicators)
		fed.GET("/calendar", h.OfficialStatisticsCalendar)

		// Metadata Harmonization
		fed.POST("/metadata/harmonize", h.HarmonizePayload)
		fed.GET("/metadata/vocabularies", h.ListVocabularies)
		fed.POST("/metadata/vocabularies", h.CreateVocabulary)

		// Cross-Agency Object Linkage
		fed.GET("/links", h.ListObjectLinks)
		fed.POST("/links", h.CreateObjectLink)
	}

	// ─── Global & Digital Diplomacy Routes ────────────────────────────────────
	dip := api.Group("/diplomacy")
	{
		// Treaties & Sovereign Accords
		dip.GET("/treaties", h.ListTreaties)
		dip.POST("/treaties", h.RegisterTreaty)
		dip.GET("/treaties/:id", h.GetTreaty)

		// Multilateral Reporting (SDG, AU, EAC, UN)
		dip.POST("/reports/sdg", h.GenerateSDGReport)
		dip.POST("/reports/au", h.GenerateAUReport)
		dip.GET("/reports", h.ListInternationalReports)
		dip.GET("/reports/:id", h.GetInternationalReport)

		// Federated Search Hub
		dip.POST("/search", h.FederatedSearch)

		// Transboundary Sovereignty & Compliance
		dip.POST("/compliance/evaluate", h.EvaluateCompliance)
		dip.GET("/compliance/audit-logs", h.ListComplianceLogs)
	}

	return r
}
