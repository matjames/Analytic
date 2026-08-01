package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go-backend/configs"
	"go-backend/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	godotenv.Load()
	// Read JWT secret from Docker secret file if not in env
	if os.Getenv("STATGATE_REGISTRY_JWT_SECRET") == "" {
		secretPath := "/run/secrets/STATGATE_REGISTRY_JWT_SECRET"
		if b, err := os.ReadFile(secretPath); err == nil {
			os.Setenv("STATGATE_REGISTRY_JWT_SECRET", strings.TrimSpace(string(b)))
		}
	}
	configs.ConnectDB()

	r := gin.Default()

	// Prometheus metrics
	reqCounter := promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "statgate",
		Name:      "http_requests_total",
		Help:      "HTTP requests processed, labeled by method, endpoint and status",
	}, []string{"method", "endpoint", "status"})

	reqLatency := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "statgate",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latencies in seconds",
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

	r.GET("/health", func(c *gin.Context) {
		if configs.DB == nil {
			c.JSON(503, gin.H{"status": "unavailable", "service": "statgate-registry-api", "database": "not-connected"})
			return
		}
		if err := configs.DB.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "unavailable", "service": "statgate-registry-api", "database": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "healthy", "service": "statgate-registry-api", "database": "connected"})
	})

	r.GET("/ready", func(c *gin.Context) {
		if configs.DB == nil {
			c.JSON(503, gin.H{"status": "not-ready", "service": "statgate-registry-api", "database": "not-connected"})
			return
		}
		if err := configs.DB.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "not-ready", "service": "statgate-registry-api", "database": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ready", "service": "statgate-registry-api", "database": "connected"})
	})

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5000",
			"http://localhost:5001",
			"http://host.docker.internal:3000",
			"http://host.docker.internal:5000",
			"http://host.docker.internal:5001",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "accept", "origin", "Cache-Control", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	r.MaxMultipartMemory = 60 << 20 // 60MB
	routes.Register(r)

	port := os.Getenv("PORT")
	fmt.Printf("DEBUG: PORT=%q runAddr=%q\n", port, ":"+port)
	if err := r.Run(":" + port); err != nil {
		fmt.Printf("ERROR starting server: %v\n", err)
		os.Exit(1)
	}
}
