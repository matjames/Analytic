package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/tenant"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log.Println("Starting StatGate App 4: Platform Engineering / RunOps (StatOps)...")

	// Initialize In-Memory Store & Scraper
	globalStore = NewMemStore()
	initScraper()

	// Initialize Optional PostgreSQL and Redis (non-blocking)
	go func() {
		_, _ = initDB()
	}()
	go func() {
		_ = initRedis()
	}()

	router := gin.Default()

	// ── CORS ──────────────────────────────────────────────────────────────────
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Internal-API-Key", "X-Request-ID", "X-Tenant-ID", "X-Workspace-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── Mandatory Platform Probes (no auth) ───────────────────────────────────
	router.GET("/health", HealthHandler)
	router.GET("/ready", ReadyHandler)
	router.GET("/readyz", ReadyHandler)
	router.GET("/live", HealthHandler)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// ── Zero-Trust Auth Middleware ─────────────────────────────────────────────
	jwtValidator, err := auth.NewValidator(
		getEnv("STATGATE_REGISTRY_JWT_SECRET", ""),
		getEnv("STATGATE_JWT_ISSUER", "statgate-registry"),
		getEnv("STATGATE_JWT_AUDIENCE", "statgate"),
	)
	if err != nil {
		log.Printf("[WARN] JWT validator not configured — running in OPEN-DEV mode: %v", err)
	}

	// ── API v1 ─────────────────────────────────────────────────────────────────
	v1 := router.Group("/api/v1")

	// Apply auth + tenant isolation when validator active
	if jwtValidator != nil {
		v1.Use(jwtValidator.GinMiddleware())
		v1.Use(tenant.GinTenantIsolation())
	} else {
		v1.Use(func(c *gin.Context) {
			hdr := c.GetHeader("X-Tenant-ID")
			if hdr == "" {
				hdr = "tenant-alpha"
			}
			c.Set("tenant_id", hdr)
			c.Next()
		})
	}
	v1.Use(func(c *gin.Context) {
		workspaceID := c.GetHeader("X-Workspace-ID")
		if !validWorkspaceID(workspaceID) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
			return
		}
		c.Set("workspace_id", workspaceID)
		c.Next()
	})

	{
		v1.GET("/summary", SummaryHandler)

		// ── P49: IOC & Observability ──────────────────────────────────────────
		obs := v1.Group("/observability")
		{
			obs.GET("/telemetry", ListTelemetryHandler)
			obs.POST("/telemetry/scrape", ScrapeTelemetryHandler)
			obs.GET("/logs", ListLogsHandler)
			obs.GET("/cmdb", ListCMDBHandler)
			obs.GET("/alerts", ListAlertsHandler)
			obs.POST("/alerts", CreateAlertHandler)
			obs.GET("/status-page", ListStatusPageHandler)
		}

		// ── P20: CI/CD & DevOps ───────────────────────────────────────────────
		devops := v1.Group("/devops")
		{
			devops.GET("/pipelines", ListPipelinesHandler)
			devops.POST("/pipelines/trigger", TriggerPipelineHandler)
			devops.GET("/deployments", ListDeploymentsHandler)
			devops.POST("/deployments", CreateDeploymentHandler)
			devops.POST("/deployments/:id/rollback", RollbackDeploymentHandler)
		}

		// ── P25: Sovereign Multi-Cloud ─────────────────────────────────────────
		cloud := v1.Group("/cloud")
		{
			cloud.GET("/clusters", ListClustersHandler)
			cloud.GET("/costs", ListCostsHandler)
		}

		// ── P34: SRE & Production Readiness ───────────────────────────────────
		sre := v1.Group("/sre")
		{
			sre.GET("/readiness", ListReadinessHandler)
			sre.GET("/slos", ListSLOsHandler)
			sre.POST("/runbooks/execute", ExecuteRunbookHandler)
			sre.POST("/chaos/simulate", SimulateChaosDrillHandler)
		}
	}

	// ── Start Server ──────────────────────────────────────────────────────────
	port := getEnv("PORT", "8098")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("StatOps Backend operational on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down StatOps service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("StatOps service exited cleanly.")
}

func validWorkspaceID(workspaceID string) bool {
	if len(workspaceID) > 128 {
		return false
	}
	for _, r := range workspaceID {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}
