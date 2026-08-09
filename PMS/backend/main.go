package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load PMS-specific environment (PMS/.env) with fallback to local .env
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")
	// PMS_ENV should be set from the loaded .env or environment
	if getEnv("PMS_ENV", "") == "" {
		_ = godotenv.Load("../../.env")
	}

	// Database configuration
	dbHost := getEnv("PMS_DB_HOST", "postgres")
	dbPort := getEnv("PMS_DB_PORT", "5432")
	dbUser := getEnv("PMS_DB_USER", "PMS")
	dbPassword := getEnv("PMS_DB_PASSWORD", "Statgate")
	dbName := getEnv("PMS_DB_NAME", "pms")
	dbSSLMode := getEnv("PMS_DB_SSLMODE", "disable")

	// Initialize database connection
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	if err := InitDB(dsn); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer DB.Close()

	if err := initRedis(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	r := gin.Default()

	// Prometheus metrics
	reqCounter := promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "statgate",
		Name:      "pms_http_requests_total",
		Help:      "PMS HTTP requests processed, labeled by method, endpoint and status",
	}, []string{"method", "endpoint", "status"})

	reqLatency := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "statgate",
		Name:      "pms_http_request_duration_seconds",
		Help:      "PMS HTTP request latencies in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "endpoint"})

	// Expose Prometheus metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Instrumentation middleware
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

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		if err := DB.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "unavailable", "service": "statgate-pms", "database": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "healthy", "service": "statgate-pms", "database": "connected"})
	})

	r.GET("/ready", func(c *gin.Context) {
		if err := DB.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "not-ready", "service": "statgate-pms", "database": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ready", "service": "statgate-pms", "database": "connected"})
	})

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3010",
			"http://localhost:3006",
			"http://localhost:5000",
			"http://localhost:5175",
			"http://host.docker.internal:3010",
			"http://host.docker.internal:3006",
			"http://host.docker.internal:5175",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "accept", "origin", "Cache-Control", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	r.MaxMultipartMemory = 60 << 20 // 60MB
	RegisterRoutes(r)

	port := getEnv("PORT", "8091")
	log.Printf("Starting StatGate PMS Backend Server on :%s...", port)
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
