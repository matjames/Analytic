package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

type ComponentHealth struct {
	Status    Status `json:"status"`
	Message   string `json:"message,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
}

type Response struct {
	Service    string                     `json:"service"`
	Status     Status                     `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentHealth `json:"components,omitempty"`
}

type Checker struct {
	service string
	db      *sql.DB
	custom  map[string]func(ctx context.Context) error
}

func NewChecker(service string, db *sql.DB) *Checker {
	return &Checker{
		service: service,
		db:      db,
		custom:  make(map[string]func(ctx context.Context) error),
	}
}

func (c *Checker) RegisterCheck(name string, fn func(ctx context.Context) error) {
	c.custom[name] = fn
}

func (c *Checker) CheckHealth(ctx context.Context) Response {
	resp := Response{
		Service:    c.service,
		Status:     StatusHealthy,
		Timestamp:  time.Now().UTC(),
		Components: make(map[string]ComponentHealth),
	}

	if c.db != nil {
		start := time.Now()
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		err := c.db.PingContext(dbCtx)
		dur := time.Since(start).Milliseconds()

		if err != nil {
			resp.Status = StatusUnhealthy
			resp.Components["database"] = ComponentHealth{
				Status:    StatusUnhealthy,
				Message:   err.Error(),
				LatencyMs: dur,
			}
		} else {
			resp.Components["database"] = ComponentHealth{
				Status:    StatusHealthy,
				LatencyMs: dur,
			}
		}
	}

	for name, fn := range c.custom {
		start := time.Now()
		chkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		err := fn(chkCtx)
		dur := time.Since(start).Milliseconds()

		if err != nil {
			if resp.Status != StatusUnhealthy {
				resp.Status = StatusDegraded
			}
			resp.Components[name] = ComponentHealth{
				Status:    StatusUnhealthy,
				Message:   err.Error(),
				LatencyMs: dur,
			}
		} else {
			resp.Components[name] = ComponentHealth{
				Status:    StatusHealthy,
				LatencyMs: dur,
			}
		}
	}

	return resp
}

// GinHandlers registers `/health`, `/ready`, and `/live` endpoints onto a Gin engine.
func (c *Checker) RegisterGinRoutes(r *gin.Engine) {
	r.GET("/health", func(ctx *gin.Context) {
		h := c.CheckHealth(ctx.Request.Context())
		code := http.StatusOK
		if h.Status == StatusUnhealthy {
			code = http.StatusServiceUnavailable
		}
		ctx.JSON(code, h)
	})

	r.GET("/ready", func(ctx *gin.Context) {
		h := c.CheckHealth(ctx.Request.Context())
		if h.Status == StatusUnhealthy {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "health check failed"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ready", "service": c.service})
	})

	r.GET("/live", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "live", "service": c.service})
	})
}

// HTTPHandler returns a standard net/http health check handler.
func (c *Checker) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := c.CheckHealth(r.Context())
		w.Header().Set("Content-Type", "application/json")
		if h.Status == StatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(h)
	})
}
