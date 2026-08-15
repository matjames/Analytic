package main

import (
	"context"
	"log"
	"runtime"
	"sync/atomic"
	"time"
)

// ─── Metrics Collector ────────────────────────────────────────────
// A lightweight, lock-free atomic metrics accumulator.
// This avoids importing Prometheus or OpenTelemetry at this stage
// while providing machine-readable observability data via /metrics.

type MetricsCollector struct {
	requests      atomic.Int64
	errors        atomic.Int64
	events        atomic.Int64
	dbQueries     atomic.Int64
	startTime     time.Time
}

func (m *MetricsCollector) incRequests()  { m.requests.Add(1) }
func (m *MetricsCollector) incErrors()    { m.errors.Add(1) }
func (m *MetricsCollector) incEvents()    { m.events.Add(1) }
func (m *MetricsCollector) incDBQuery()   { m.dbQueries.Add(1) }

func (m *MetricsCollector) snapshot() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptimeSec := int64(time.Since(m.startTime).Seconds())
	req := m.requests.Load()
	errs := m.errors.Load()
	errorRate := float64(0)
	if req > 0 {
		errorRate = float64(errs) / float64(req) * 100
	}

	redisDepth := int64(0)
	if redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		depth, _ := redisClient.LLen(ctx, "statgate:event:history").Result()
		redisDepth = depth
	}

	dlqDepth := int64(0)
	if redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		depth, _ := redisClient.LLen(ctx, deadLetterKey).Result()
		dlqDepth = depth
	}

	dbPoolStats := map[string]interface{}{"configured": dbPool != nil}
	if dbPool != nil {
		stats := dbPool.Stats()
		dbPoolStats = map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":          stats.InUse,
			"idle":            stats.Idle,
			"max_open":        stats.MaxOpenConnections,
		}
	}

	return map[string]interface{}{
		"service":   "statgate-enterprise-core",
		"version":   version,
		"phase":     "PHASE_X",
		"uptime_s":  uptimeSec,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"requests": map[string]interface{}{
			"total":      req,
			"errors":     errs,
			"error_rate": errorRate,
		},
		"events": map[string]interface{}{
			"total_published": m.events.Load(),
			"redis_queue_depth": redisDepth,
			"dlq_depth":        dlqDepth,
		},
		"database": map[string]interface{}{
			"queries": m.dbQueries.Load(),
			"pool":    dbPoolStats,
		},
		"memory": map[string]interface{}{
			"alloc_mb":      memStats.Alloc / 1024 / 1024,
			"sys_mb":        memStats.Sys / 1024 / 1024,
			"heap_inuse_mb": memStats.HeapInuse / 1024 / 1024,
			"gc_runs":       memStats.NumGC,
		},
		"goroutines": runtime.NumGoroutine(),
	}
}

var metricsCollector = &MetricsCollector{
	startTime: startTime,
}

// ─── Metrics Aggregation Goroutine ───────────────────────────────
// Publishes a metrics snapshot to Redis every 30 seconds so that
// the Command Centre PlatformHealthView can read live platform stats.

func startMetricsAggregator() {
	go func() {
		for range time.Tick(30 * time.Second) {
			if redisClient == nil {
				continue
			}
			snap := metricsCollector.snapshot()
			// Use the observability event bus key for cross-service reading
			if data, err := marshalJSON(snap); err == nil {
				ctx := context.Background()
				redisClient.Set(ctx, "statgate:metrics:enterprise-core", string(data), 2*time.Minute)
			}
		}
	}()
	log.Println("observability: metrics aggregator started (30s interval)")
}

// ─── Readiness Probe ─────────────────────────────────────────────
// isReady returns true only if both Redis and the database are reachable.
// This is used by /ready which Kubernetes/Docker Compose uses before routing traffic.

func isReady() (bool, map[string]interface{}) {
	details := map[string]interface{}{}
	allGood := true

	// Redis
	if redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			details["redis"] = map[string]interface{}{"status": "unhealthy", "error": err.Error()}
			allGood = false
		} else {
			details["redis"] = map[string]interface{}{"status": "healthy"}
		}
	} else {
		details["redis"] = map[string]interface{}{"status": "not_configured"}
		// Redis degraded is not fatal for readiness — we have persistence fallback
	}

	// Database
	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if err := dbPool.PingContext(ctx); err != nil {
			details["database"] = map[string]interface{}{"status": "unhealthy", "error": err.Error()}
			allGood = false
		} else {
			details["database"] = map[string]interface{}{"status": "healthy"}
		}
	} else {
		details["database"] = map[string]interface{}{"status": "not_configured"}
		// DB not configured in development — not a readiness failure
	}

	return allGood, details
}
