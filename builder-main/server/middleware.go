package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// Default: 100 requests/second globally, burst of 200
var (
	rateLimitRate  = 100.0
	rateLimitBurst = 200
	globalLimiter  *rate.Limiter
)

// Per-IP rate limiting: 20 requests/second per IP, burst of 40
var (
	perIPRate  = 20.0
	perIPBurst = 40
	ipLimiters sync.Map // map[string]*ipLimiterEntry
)

// Per-user AI rate limiting for /api/ask/* endpoints. Each chat call hits a
// paid LLM API (Claude/Gemini), so this guards against a single authenticated
// user draining the API key budget. Keyed by JWT username, falling back to
// "ip:<addr>" when AUTH_MODE=off.
// Defaults: 10 requests/minute, burst of 5.
var (
	chatAIRate     = 10.0 / 60.0
	chatAIBurst    = 5
	chatAILimiters sync.Map // map[string]*ipLimiterEntry — reuses ipLimiterEntry shape
)

// contentSecurityPolicy is built in init() from static app needs + optional KEYCLOAK_URL origin.
var contentSecurityPolicy string

type ipLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64 // UnixNano timestamp; atomic to avoid data races
}

func init() {
	// Global limiter config from env
	if v := os.Getenv("RATE_LIMIT_PER_SEC"); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil {
			rateLimitRate = r
		}
	}
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		if b, err := strconv.Atoi(v); err == nil {
			rateLimitBurst = b
		}
	}
	globalLimiter = rate.NewLimiter(rate.Limit(rateLimitRate), rateLimitBurst)

	// Per-IP limiter config from env
	if v := os.Getenv("RATE_LIMIT_PER_IP"); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil {
			perIPRate = r
		}
	}
	if v := os.Getenv("RATE_LIMIT_PER_IP_BURST"); v != "" {
		if b, err := strconv.Atoi(v); err == nil {
			perIPBurst = b
		}
	}

	// Chat AI limiter config. CHAT_AI_RATE_PER_MIN is requests/minute (converted
	// to per-second internally because rate.Limit is per-second).
	if v := os.Getenv("CHAT_AI_RATE_PER_MIN"); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil && r > 0 {
			chatAIRate = r / 60.0
		}
	}
	if v := os.Getenv("CHAT_AI_BURST"); v != "" {
		if b, err := strconv.Atoi(v); err == nil && b > 0 {
			chatAIBurst = b
		}
	}

	// Cleanup stale IP and chat-AI entries every 10 minutes
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-10 * time.Minute).UnixNano()
			sweep := func(m *sync.Map) {
				m.Range(func(key, value any) bool {
					entry, ok := value.(*ipLimiterEntry)
					if !ok {
						m.Delete(key)
						return true
					}
					if entry.lastSeen.Load() < cutoff {
						m.Delete(key)
					}
					return true
				})
			}
			sweep(&ipLimiters)
			sweep(&chatAILimiters)
		}
	}()

}

// InitCSP builds the Content-Security-Policy header. Must be called after .env
// is loaded so that KEYCLOAK_URL is available via os.Getenv.
func InitCSP() {
	contentSecurityPolicy = buildContentSecurityPolicy()
}

func buildContentSecurityPolicy() string {
	var b strings.Builder
	b.WriteString("default-src 'self'; ")
	b.WriteString("base-uri 'self'; ")
	b.WriteString("frame-ancestors 'none'; ")
	b.WriteString("object-src 'none'; ")
	b.WriteString("upgrade-insecure-requests; ")
	b.WriteString("script-src 'self' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com https://www.googletagmanager.com 'unsafe-inline'; ")
	b.WriteString("style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com; ")
	b.WriteString("font-src 'self' https://fonts.gstatic.com data:; ")
	b.WriteString("img-src 'self' data: blob: https://a.tile.openstreetmap.org https://b.tile.openstreetmap.org https://c.tile.openstreetmap.org https://www.google-analytics.com; ")
	b.WriteString("worker-src 'self'; ")
	b.WriteString("connect-src 'self'")
	if kc := strings.TrimSpace(os.Getenv("KEYCLOAK_URL")); kc != "" {
		kc = strings.TrimSuffix(kc, "/")
		if u, err := url.Parse(kc); err == nil && u.Scheme != "" && u.Host != "" {
			b.WriteString(" ")
			b.WriteString(u.Scheme)
			b.WriteString("://")
			b.WriteString(u.Host)
		}
	}
	b.WriteString(" https://cdn.jsdelivr.net https://www.googletagmanager.com https://www.google-analytics.com https://region1.google-analytics.com https://analytics.google.com")
	return b.String()
}

// trustProxyHeaders is true when TRUSTED_PROXIES env var is set (non-empty),
// indicating the server is behind a reverse proxy that sets X-Real-IP/X-Forwarded-For.
var trustProxyHeaders = os.Getenv("TRUSTED_PROXIES") != ""

// getClientIP extracts the real client IP from the request.
// Only trusts X-Real-IP and X-Forwarded-For when TRUSTED_PROXIES is configured.
func getClientIP(r *http.Request) string {
	if trustProxyHeaders {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if i := strings.IndexByte(xff, ','); i > 0 {
				return strings.TrimSpace(xff[:i])
			}
			return strings.TrimSpace(xff)
		}
	}
	// Strip port from RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// getIPLimiter returns the per-IP rate limiter, creating one if needed.
func getIPLimiter(ip string) *rate.Limiter {
	now := time.Now().UnixNano()
	if v, ok := ipLimiters.Load(ip); ok {
		if entry, ok := v.(*ipLimiterEntry); ok {
			entry.lastSeen.Store(now)
			return entry.limiter
		}
	}
	entry := &ipLimiterEntry{
		limiter: rate.NewLimiter(rate.Limit(perIPRate), perIPBurst),
	}
	entry.lastSeen.Store(now)
	actual, _ := ipLimiters.LoadOrStore(ip, entry)
	if limiterEntry, ok := actual.(*ipLimiterEntry); ok {
		return limiterEntry.limiter
	}
	return rate.NewLimiter(rate.Limit(perIPRate), perIPBurst)
}

// getChatAILimiter returns the per-user AI rate limiter, creating one if needed.
// key should be the JWT username; callers fall back to "ip:<addr>" when AUTH_MODE=off.
func getChatAILimiter(key string) *rate.Limiter {
	now := time.Now().UnixNano()
	if v, ok := chatAILimiters.Load(key); ok {
		if entry, ok := v.(*ipLimiterEntry); ok {
			entry.lastSeen.Store(now)
			return entry.limiter
		}
	}
	entry := &ipLimiterEntry{
		limiter: rate.NewLimiter(rate.Limit(chatAIRate), chatAIBurst),
	}
	entry.lastSeen.Store(now)
	actual, _ := chatAILimiters.LoadOrStore(key, entry)
	if limiterEntry, ok := actual.(*ipLimiterEntry); ok {
		return limiterEntry.limiter
	}
	return rate.NewLimiter(rate.Limit(chatAIRate), chatAIBurst)
}

// SecurityHeadersMiddleware sets baseline browser security headers on every response.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if contentSecurityPolicy != "" {
			w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		}
		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware catches panics and returns 500 instead of crashing
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				reqID := GetRequestID(r.Context())
				log.Printf("[%s] [PANIC] %s %s: %v", reqID, r.Method, r.URL.Path, err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware rejects requests when rate exceeded.
// Dual-layer: per-IP check first, then global check.
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		// Per-IP check first
		if !getIPLimiter(ip).Allow() {
			reqID := GetRequestID(r.Context())
			log.Printf("[%s] [RATE_LIMIT_IP] %s %s %s rejected", reqID, ip, r.Method, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "rate limit exceeded, please slow down"))
			return
		}

		// Global check
		if !globalLimiter.Allow() {
			reqID := GetRequestID(r.Context())
			log.Printf("[%s] [RATE_LIMIT] %s %s rejected", reqID, r.Method, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "rate limit exceeded, please slow down"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
