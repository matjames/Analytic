package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// RequestIDKey is the context key for request IDs
	RequestIDKey contextKey = "requestID"
)

// generateRequestID creates a short, unique request ID (8 hex characters)
func generateRequestID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return hex.EncodeToString([]byte(time.Now().Format("15040500")))[:8]
	}
	return hex.EncodeToString(b)
}

// GetRequestID extracts the request ID from context, returns empty string if not found
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithRequestID is middleware that adds a unique request ID to each request
// The ID is added to the request context and response header (X-Request-ID)
func WithRequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := generateRequestID()
		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)

		// Set response header so client can reference the ID
		w.Header().Set("X-Request-ID", reqID)

		// Log request start
		log.Printf("[%s] %s %s", reqID, r.Method, r.URL.Path)

		next(w, r.WithContext(ctx))
	}
}

// Context-aware logging functions
// These extract the request ID from context and include it in log output

// logInfoCtx logs an info message with request ID from context
func logInfoCtx(ctx context.Context, format string, v ...interface{}) {
	reqID := GetRequestID(ctx)
	if reqID != "" {
		log.Printf("[%s] [INFO] "+format, append([]interface{}{reqID}, v...)...)
	} else {
		log.Printf("[INFO] "+format, v...)
	}
}

// logWarnCtx logs a warning message with request ID from context
func logWarnCtx(ctx context.Context, format string, v ...interface{}) {
	reqID := GetRequestID(ctx)
	if reqID != "" {
		log.Printf("[%s] [WARN] "+format, append([]interface{}{reqID}, v...)...)
	} else {
		log.Printf("[WARN] "+format, v...)
	}
}

// logErrorCtx logs an error message with request ID from context
func logErrorCtx(ctx context.Context, format string, v ...interface{}) {
	reqID := GetRequestID(ctx)
	if reqID != "" {
		log.Printf("[%s] [ERROR] "+format, append([]interface{}{reqID}, v...)...)
	} else {
		log.Printf("[ERROR] "+format, v...)
	}
}

// logDebugCtx logs a debug message with request ID from context
func logDebugCtx(ctx context.Context, format string, v ...interface{}) {
	reqID := GetRequestID(ctx)
	if reqID != "" {
		log.Printf("[%s] [DEBUG] "+format, append([]interface{}{reqID}, v...)...)
	} else {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// ErrorResponse represents a JSON error response with request ID
type ErrorResponse struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// NewErrorResponse creates an error response, including request ID if available
func NewErrorResponse(ctx context.Context, message string) ErrorResponse {
	return ErrorResponse{
		Error:     message,
		RequestID: GetRequestID(ctx),
	}
}
