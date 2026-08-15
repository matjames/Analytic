package requestid

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	HeaderRequestID     = "X-Request-ID"
	HeaderCorrelationID = "X-Correlation-ID"
)

type ctxKey string

const (
	ContextKeyRequestID     ctxKey = "request_id"
	ContextKeyCorrelationID ctxKey = "correlation_id"
)

// GinMiddleware creates a Gin middleware that extracts or generates correlation IDs.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := strings.TrimSpace(c.GetHeader(HeaderRequestID))
		if reqID == "" {
			reqID = uuid.New().String()
		}

		corrID := strings.TrimSpace(c.GetHeader(HeaderCorrelationID))
		if corrID == "" {
			corrID = reqID
		}

		c.Header(HeaderRequestID, reqID)
		c.Header(HeaderCorrelationID, corrID)

		c.Set(string(ContextKeyRequestID), reqID)
		c.Set(string(ContextKeyCorrelationID), corrID)

		c.Next()
	}
}

// HTTPMiddleware creates a standard net/http middleware.
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := strings.TrimSpace(r.Header.Get(HeaderRequestID))
		if reqID == "" {
			reqID = uuid.New().String()
		}

		corrID := strings.TrimSpace(r.Header.Get(HeaderCorrelationID))
		if corrID == "" {
			corrID = reqID
		}

		w.Header().Set(HeaderRequestID, reqID)
		w.Header().Set(HeaderCorrelationID, corrID)

		ctx := context.WithValue(r.Context(), ContextKeyRequestID, reqID)
		ctx = context.WithValue(ctx, ContextKeyCorrelationID, corrID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID returns the request ID from context.
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return val
	}
	return ""
}

// GetCorrelationID returns the correlation ID from context.
func GetCorrelationID(ctx context.Context) string {
	if val, ok := ctx.Value(ContextKeyCorrelationID).(string); ok {
		return val
	}
	return ""
}
