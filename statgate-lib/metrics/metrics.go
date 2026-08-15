package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Namespace     string
	RequestsTotal *prometheus.CounterVec
	LatencySec    *prometheus.HistogramVec
	ActiveConns   prometheus.Gauge
	ErrorsTotal   *prometheus.CounterVec
}

// NewMetrics initializes standardized Prometheus metrics for a StatGate service.
func NewMetrics(serviceName string) *Metrics {
	ns := "statgate"
	m := &Metrics{
		Namespace: ns,
		RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: serviceName,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests processed labeled by method, path, and status code",
		}, []string{"method", "path", "status"}),
		LatencySec: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: serviceName,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency distributions in seconds",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "path"}),
		ActiveConns: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: serviceName,
			Name:      "active_connections",
			Help:      "Current active connections in service",
		}),
		ErrorsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: serviceName,
			Name:      "errors_total",
			Help:      "Total errors categorized by error code and subsystem",
		}, []string{"code", "subsystem"}),
	}
	return m
}

// GinMiddleware tracks request counts and latency automatically.
func (m *Metrics) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		m.ActiveConns.Inc()
		defer m.ActiveConns.Dec()

		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		duration := time.Since(start).Seconds()
		statusStr := fmt.Sprintf("%d", c.Writer.Status())

		m.RequestsTotal.WithLabelValues(c.Request.Method, path, statusStr).Inc()
		m.LatencySec.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// HTTPHandler returns the Prometheus metrics exposition endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
