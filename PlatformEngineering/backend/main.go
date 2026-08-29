package main

import (
	"context"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/events"
	"github.com/matjames/statgate-lib/permissions"
	"github.com/matjames/statgate-lib/tenant"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := loadConfig()
	db, err := openDB(cfg)
	if err != nil {
		log.Fatalf("RunOps database is required: %v", err)
	}
	defer db.Close()
	bus, err := events.InitFromEnv("statgate-runops")
	if err != nil {
		log.Printf("RunOps event bus degraded: %v", err)
	}
	collector := NewTelemetryCollector(db, bus, cfg.ServiceTargets)
	app := &API{db: db, telemetry: collector, bus: bus}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector.Start(ctx, time.Duration(cfg.ProbeIntervalSeconds)*time.Second)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), cors.New(cors.Config{AllowOrigins: []string{"*"}, AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID", "X-Workspace-ID", "X-StatGate-Cluster", "X-StatGate-Allowed-Clusters"}, MaxAge: 12 * time.Hour}))
	r.GET("/health", app.health)
	r.GET("/live", app.health)
	r.GET("/ready", app.ready)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	v := r.Group("/api/v1")
	validator, authErr := auth.NewValidator(env("STATGATE_REGISTRY_JWT_SECRET", ""), env("STATGATE_JWT_ISSUER", "statgate-registry"), env("STATGATE_JWT_AUDIENCE", "statgate"))
	if authErr != nil {
		log.Fatalf("RunOps requires statgate-lib/auth configuration: %v", authErr)
	}
	v.Use(validator.GinMiddleware(), tenant.GinTenantIsolation())
	// Stage 2: workspace context + Enterprise Core membership enforcement.
	v.Use(tenant.GinWorkspaceContext(), tenant.GinWorkspaceMembership("", nil))
	v.GET("/summary", permissions.RequirePermission(permissions.PermRead), app.summary)
	// P20: CI/CD pipeline orchestration.
	cicd := v.Group("/cicd", requireClusterAccess(permissions.PermExecute))
	cicd.POST("/deployments", app.createDeployment)
	// P25: sovereign multi-cloud inventory and cluster provisioning.
	cloud := v.Group("/cloud", requireClusterAccess(permissions.PermWrite))
	cloud.POST("/resources", app.createResource)
	cloud.POST("/clusters", app.createCluster)
	// P34/P49: operations centre and ingestion pipelines.
	ioc := v.Group("/ioc", permissions.RequirePermission(permissions.PermRead))
	ioc.GET("/targets", app.targets)
	ioc.POST("/probes", requireClusterAccess(permissions.PermExecute), app.probe)
	ioc.GET("/alerts", app.listAlerts)
	ioc.POST("/alerts/:id/acknowledge", requireClusterAccess(permissions.PermWrite), app.acknowledgeAlert)
	telemetry := v.Group("/telemetry", permissions.RequirePermission(permissions.PermWrite))
	telemetry.POST("/logs", app.ingestLog)
	telemetry.POST("/metrics", app.ingestMetric)
	telemetry.POST("/traces", app.ingestTrace)
	log.Printf("StatGate App 4 RunOps listening on :%s", cfg.Port)
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}
	go func() {
		if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Fatal(e)
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	shutdown, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()
	_ = srv.Shutdown(shutdown)
}
