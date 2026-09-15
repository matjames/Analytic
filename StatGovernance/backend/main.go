package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version   = "8.0.0"
	startTime = time.Now()
)

func main() {
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	// Fail fast in production when mandatory secrets are missing (SG-SEC-2026-08).
	env := getEnv("STATGOVERNANCE_ENV", "development")
	if strings.EqualFold(env, "production") {
		if strings.TrimSpace(os.Getenv("STATGATE_REGISTRY_JWT_SECRET")) == "" {
			log.Fatal("FATAL: STATGATE_REGISTRY_JWT_SECRET is required in production. Startup aborted.")
		}
		if strings.TrimSpace(os.Getenv("GOVERNANCE_DB_PASSWORD")) == "" {
			log.Fatal("FATAL: GOVERNANCE_DB_PASSWORD is required in production. Startup aborted.")
		}
	}

	// Database Configuration (no hardcoded credential fallbacks)
	dbHost := getEnv("GOVERNANCE_DB_HOST", getEnv("ENTERPRISE_DB_HOST", "postgres"))
	dbPort := getEnv("GOVERNANCE_DB_PORT", "5432")
	dbUser := getEnv("GOVERNANCE_DB_USER", "StatGovernance")
	dbPassword := getEnv("GOVERNANCE_DB_PASSWORD", "")
	dbName := getEnv("GOVERNANCE_DB_NAME", "statgovernance")
	dbSSLMode := getEnv("GOVERNANCE_DB_SSLMODE", "disable")

	if dbPassword == "" {
		log.Println("WARNING: GOVERNANCE_DB_PASSWORD is not configured; database features will fail.")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	if err := InitDB(dsn); err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
	}
	if DB != nil {
		defer DB.Close()
	}

	if err := initRedis(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}
	initStatChat()

	r := gin.Default()

	// Prometheus Metrics Instrumentation
	reqCounter := promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "statgate",
		Name:      "governance_http_requests_total",
		Help:      "StatGovernance HTTP requests processed",
	}, []string{"method", "endpoint", "status"})

	reqLatency := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "statgate",
		Name:      "governance_http_request_duration_seconds",
		Help:      "StatGovernance HTTP request latencies in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "endpoint"})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		dur := time.Since(start).Seconds()
		reqLatency.WithLabelValues(c.Request.Method, path).Observe(dur)
		reqCounter.WithLabelValues(c.Request.Method, path, fmt.Sprintf("%d", c.Writer.Status())).Inc()
	})

	// Health check endpoints
	r.GET("/health", func(c *gin.Context) {
		dbStatus := "connected"
		if DB == nil || DB.Ping() != nil {
			dbStatus = "disconnected"
		}
		redisAvailable := false
		if eventBus != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			redisAvailable = eventBus.RedisAvailable(ctx)
			cancel()
		}
		c.JSON(200, gin.H{
			"status":   "healthy",
			"service":  "statgate-governance",
			"version":  version,
			"database": dbStatus,
			"uptime_s": int(time.Since(startTime).Seconds()),
			"redis":    redisAvailable,
		})
	})

	r.GET("/ready", func(c *gin.Context) {
		if DB != nil && DB.Ping() != nil {
			c.JSON(503, gin.H{"status": "not-ready", "database": "unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ready", "service": "statgate-governance", "version": version})
	})

	// CORS Setup
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3012",
			"http://localhost:3006",
			"http://localhost:3010",
			"http://localhost:3011",
			"http://localhost:5000",
			"http://localhost:3009",
			"http://localhost:3005",
			"http://host.docker.internal:3012",
			"http://host.docker.internal:3006",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Request-ID", "X-Correlation-ID", "X-User-ID", "X-Tenant-ID", "X-Workspace-ID"},
		ExposeHeaders:    []string{"Content-Disposition", "Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	RegisterRoutes(r)

	port := getEnv("PORT", getEnv("STATGOVERNANCE_PORT", "8093"))
	log.Printf("Starting StatGate StatGovernance v%s Backend Server on :%s...", version, port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
