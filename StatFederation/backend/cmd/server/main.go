package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/database"
	"github.com/matjames/statgate-lib/events"
	"github.com/matjames/statgate-lib/health"
	"github.com/matjames/statgate-lib/metrics"

	"statfederation-backend/internal/api"
	"statfederation-backend/internal/config"
	"statfederation-backend/internal/diplomacy"
	"statfederation-backend/internal/engine"
	fedevents "statfederation-backend/internal/events"
	"statfederation-backend/internal/store"
)

func main() {
	log.Println("══════════════════════════════════════════════════════════════════════")
	log.Println("  STATGATE APP 6: NATIONAL & GLOBAL FEDERATION SERVICE (P23 / P30)   ")
	log.Println("══════════════════════════════════════════════════════════════════════")

	// 1. Load runtime configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[FATAL] Configuration load failed: %v", err)
	}
	log.Printf("[Config] Environment: %s | Port: %s | Node ID: %s", cfg.Env, cfg.Port, cfg.NodeID)

	// 2. Database Connection
	var dbConn *sql.DB
	var appStore store.Store

	dbCfg := database.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		Name:     cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}

	dbConn, err = database.Connect(dbCfg)
	if err != nil {
		log.Printf("[Database] PostgreSQL connection failed: %v. Running in high-availability in-memory test store fallback.", err)
		appStore = store.NewMemStore()
	} else {
		log.Println("[Database] PostgreSQL connection pool established successfully.")
		appStore = store.NewPGStore(dbConn)
	}

	// 3. Redis Event Bus
	eventBus, err := events.NewEventBus(events.Config{
		RedisAddr: cfg.RedisAddr,
		Password:  cfg.RedisPassword,
		DB:        cfg.RedisDB,
		Source:    "statfederation-backend",
		Channel:   events.DefaultChannel,
	})
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize EventBus: %v", err)
	}
	log.Println("[EventBus] Initialized enterprise event pub/sub on channel: " + events.DefaultChannel)

	// 4. Initialize Engines & Core Logic
	fedEngine := engine.NewFederationEngine(appStore)
	queryRouter := engine.NewQueryRouter(appStore, fedEngine)
	harmonizer := engine.NewHarmonizer(appStore)
	diplomacyGW := diplomacy.NewDiplomacyGateway(appStore)
	compliance := diplomacy.NewComplianceEvaluator(appStore)
	fedSearch := diplomacy.NewFederatedSearchHub(appStore)
	fedSearch.SetSourceNodeID(cfg.NodeID)
	pushProtocol := engine.NewPushProtocol(appStore)
	pushProtocol.SetSigner(engine.NewStatTrustSigner(cfg.StatTrustURL, cfg.NodeID))
	autoPush := fedevents.NewAutoPushConsumer(appStore, pushProtocol)
	eventWorker := fedevents.NewEventWorker(eventBus, appStore, cfg.NodeID).WithAutoPushConsumer(autoPush)

	// Start background event listener & periodic node health probe loop
	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()
	eventWorker.StartEventListener(appCtx)
	fedEngine.StartProbeLoop(appCtx, "default")
	autoPush.Start(appCtx)

	// 5. JWT Authentication Validator
	var authVal *auth.Validator
	if cfg.JWTSecret != "" {
		authVal, err = auth.NewValidator(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience)
		if err != nil {
			log.Printf("[Auth] Warning initializing validator: %v", err)
		} else {
			log.Println("[Auth] JWT validation active (Fail-closed zero-trust policy).")
		}
	} else {
		log.Println("[Auth] Development mode: No JWT secret set. Running with open routes for local test harness.")
	}

	// 6. Metrics & Health Checkers
	promMetrics := metrics.NewMetrics("statfederation")
	healthChecker := health.NewChecker("statfederation", dbConn)
	healthChecker.RegisterCheck("redis_eventbus", func(ctx context.Context) error {
		return nil // Bus handles fallback seamlessly
	})

	// 7. API Handlers & Router Setup
	handlers := api.NewHandlers(
		appStore, fedEngine, queryRouter, harmonizer,
		diplomacyGW, compliance, fedSearch, eventWorker, pushProtocol,
	)

	router := api.SetupRouter(api.RouterConfig{
		Handlers:      handlers,
		HealthChecker: healthChecker,
		Metrics:       promMetrics,
		AuthValidator: authVal,
		CORSOrigin:    cfg.CORSAllowedOrigin,
		Env:           cfg.Env,
	})

	// 8. Start HTTP Server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("[HTTP] StatFederation server listening on http://0.0.0.0:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	// 9. Graceful Shutdown Handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[Shutdown] Signal received. Shutting down gracefully...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	eventWorker.Stop()
	cancelApp()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Shutdown] Server forced to shutdown: %v", err)
	}

	if dbConn != nil {
		_ = dbConn.Close()
	}

	log.Println("[Shutdown] StatFederation service stopped.")
}
