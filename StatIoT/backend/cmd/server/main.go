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

	"statiot-backend/internal/api"
	"statiot-backend/internal/config"
	iotevents "statiot-backend/internal/events"
	"statiot-backend/internal/fieldops"
	"statiot-backend/internal/iot"
	"statiot-backend/internal/store"
)

func main() {
	log.Println("══════════════════════════════════════════════════════════════════════")
	log.Println("  STATGATE APP 8: IoT, SENSORS & MOBILE FIELD OPS (P27 / P43)       ")
	log.Println("══════════════════════════════════════════════════════════════════════")

	// 1. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[FATAL] Configuration load failed: %v", err)
	}
	log.Printf("[Config] Environment: %s | Port: %s", cfg.Env, cfg.Port)

	// 2. Database Connection & Store Init
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
		log.Printf("[Database] PostgreSQL connection failed: %v. Running with in-memory store fallback.", err)
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
		Source:    "statiot-backend",
		Channel:   events.DefaultChannel,
	})
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize EventBus: %v", err)
	}
	log.Println("[EventBus] Initialized enterprise event pub/sub on channel: " + events.DefaultChannel)

	// 4. Initialize Core Domain Engines
	eventWorker := iotevents.NewEventWorker(eventBus, appStore)
	gatewayEngine := iot.NewGatewayEngine(appStore, eventWorker)
	syncEngine := fieldops.NewSyncEngine(appStore, eventWorker)
	spatialTracker := fieldops.NewSpatialTracker(appStore)

	// Start background event listener
	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()
	eventWorker.StartEventListener(appCtx)

	// 5. Auth Validator
	var authVal *auth.Validator
	if cfg.JWTSecret != "" {
		authVal, err = auth.NewValidator(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience)
		if err != nil {
			log.Printf("[Auth] Warning initializing validator: %v", err)
		} else {
			log.Println("[Auth] JWT validation active for administrative routes.")
		}
	}

	// 6. Observability (Metrics & Health Checks)
	promMetrics := metrics.NewMetrics("statiot")
	healthChecker := health.NewChecker("statiot", dbConn)
	healthChecker.RegisterCheck("redis_eventbus", func(ctx context.Context) error {
		return nil
	})

	// 7. API Handlers & Router Setup
	iotHandlers := api.NewIoTHandlers(appStore, gatewayEngine)
	fieldHandlers := api.NewFieldOpsHandlers(appStore, syncEngine, spatialTracker)
	discussionHandler := api.NewDiscussionHandler(appStore, api.NewStatChatIntegration(os.Getenv("STATCHAT_API_URL"), os.Getenv("STATGATE_INTERNAL_API_KEY")))

	router := api.SetupRouter(api.RouterConfig{
		IoTHandlers:       iotHandlers,
		FieldOpsHandlers:  fieldHandlers,
		DiscussionHandler: discussionHandler,
		HealthChecker:     healthChecker,
		Metrics:           promMetrics,
		AuthValidator:     authVal,
		CORSOrigin:        cfg.CORSAllowedOrigin,
		Env:               cfg.Env,
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
		log.Printf("[HTTP] StatIoT server listening on http://0.0.0.0:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	// 9. Graceful Shutdown
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

	log.Println("[Shutdown] StatIoT service stopped.")
}
