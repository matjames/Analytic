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
	"github.com/joho/godotenv"
)

var (
	version   = "1.0.0"
	startTime = time.Now()
)

func main() {
	// Load .env from repository root (two levels up from StatCitizen/backend)
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load(".env")

	cfg := loadConfig()

	// Startup validation
	if cfg.Env == "production" {
		validateProductionSecrets(cfg)
	}

	log.Printf("StatCitizen v%s starting (env=%s)", version, cfg.Env)

	// Initialize database
	initDB(cfg)

	// Run schema migrations
	if err := runMigrations(); err != nil {
		log.Printf("[WARNING] Migration warning: %v", err)
	}

	// Initialize Redis event bus
	initEventBus(cfg)

	// Initialize security secrets
	initSecurity(cfg)

	// Register with Enterprise Core fabric
	go registerWithEnterpriseFabric(cfg)

	// Build HTTP server
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(correlationMiddleware())
	r.Use(securityHeadersMiddleware())
	r.Use(requestSizeMiddleware(cfg))

	// CORS: allow citizen browser origins + enterprise apps
	corsOrigins := cfg.CORSOrigins
	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Request-ID", "X-Correlation-ID", "X-Citizen-Session"},
		ExposeHeaders:    []string{"X-Correlation-ID", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	registerRoutes(r, cfg)

	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start background workers
	go startEventRetryWorker()
	go startOfflineSyncWorker()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("StatCitizen API listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("StatCitizen server error: %v", err)
		}
	}()

	<-quit
	log.Println("StatCitizen shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("StatCitizen shutdown error: %v", err)
	}

	closeEventBus()
	closeDB()
	log.Println("StatCitizen stopped.")
}
