package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ─── Citizen Session Security ─────────────────────────────────────────────────

var (
	citizenSessionSecret []byte
	enterpriseJWTSecret  []byte
)

func initSecurity(cfg *Config) {
	if cfg.CitizenSessionSecret != "" {
		citizenSessionSecret = []byte(cfg.CitizenSessionSecret)
	} else {
		// Generate ephemeral secret for development (not for production)
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		citizenSessionSecret = b
		log.Println("[WARNING] STATCITIZEN_CITIZEN_SESSION_SECRET not set — using ephemeral key (development only)")
	}
	if cfg.EnterpriseJWTSecret != "" {
		enterpriseJWTSecret = []byte(cfg.EnterpriseJWTSecret)
	}
}

// generateSessionToken creates a cryptographically secure session token.
func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// hashIP creates a one-way hash of an IP address for logging (never store raw IP).
func hashIP(ip string) string {
	if ip == "" {
		return ""
	}
	mac := hmac.New(sha256.New, citizenSessionSecret)
	mac.Write([]byte(ip))
	return hex.EncodeToString(mac.Sum(nil))[:16] // first 16 chars only
}

// hashPII creates a one-way hash of PII (email/phone) for identity matching without storage.
func hashPII(value string) string {
	if value == "" {
		return ""
	}
	value = strings.ToLower(strings.TrimSpace(value))
	mac := hmac.New(sha256.New, citizenSessionSecret)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

// ─── Rate Limiting ────────────────────────────────────────────────────────────

type rateBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func (b *rateBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

var (
	rateLimitMu      sync.Mutex
	rateLimitBuckets = make(map[string]*rateBucket)
)

func getRateBucket(key string, maxRPM int) *rateBucket {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()
	if b, ok := rateLimitBuckets[key]; ok {
		return b
	}
	b := &rateBucket{
		tokens:     float64(maxRPM),
		maxTokens:  float64(maxRPM),
		refillRate: float64(maxRPM) / 60.0,
		lastRefill: time.Now(),
	}
	rateLimitBuckets[key] = b
	return b
}

// ─── Correlation ID Middleware ────────────────────────────────────────────────

func correlationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		corrID := c.GetHeader("X-Correlation-ID")
		if corrID == "" {
			corrID = fmt.Sprintf("sc_%d", time.Now().UnixNano())
		}
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = fmt.Sprintf("req_%d", time.Now().UnixNano())
		}
		c.Set("correlation_id", corrID)
		c.Set("request_id", reqID)
		c.Header("X-Correlation-ID", corrID)
		c.Header("X-Request-ID", reqID)
		c.Next()
	}
}

// ─── Security Headers Middleware ──────────────────────────────────────────────

func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(self), camera=(), microphone=()")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; script-src 'self'")
		c.Next()
	}
}

// ─── Request Size Middleware ──────────────────────────────────────────────────

func requestSizeMiddleware(cfg *Config) gin.HandlerFunc {
	maxBytes := cfg.MaxRequestKB * 1024
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, ErrorResponse{
				Error:   "request_too_large",
				Message: fmt.Sprintf("Request body exceeds %d KB limit", cfg.MaxRequestKB),
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// ─── Public Rate Limit Middleware ─────────────────────────────────────────────

func publicRateLimitMiddleware(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		bucket := getRateBucket("pub:"+ip, cfg.RateLimitRPM)
		if !bucket.allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: "Too many requests. Please try again later.",
				Code:    "RATE_LIMIT",
			})
			return
		}
		c.Next()
	}
}

// ─── Citizen Session Middleware ───────────────────────────────────────────────

// citizenSessionMiddleware validates the X-Citizen-Session header or cookie.
// It does NOT require enterprise JWT — it validates a citizen-specific session token.
func citizenSessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Citizen-Session")
		if token == "" {
			if cookie, err := c.Cookie("citizen_session"); err == nil {
				token = cookie
			}
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "unauthorized",
				Message: "Citizen session required. Please start a session.",
				Code:    "SESSION_REQUIRED",
			})
			return
		}

		session, err := lookupSession(token)
		if err != nil || session == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_session",
				Message: "Session not found or expired.",
				Code:    "SESSION_INVALID",
			})
			return
		}

		c.Set("citizen_session", session)
		c.Set("tenant_id", session.TenantID)
		c.Next()
	}
}

// optionalCitizenSessionMiddleware attaches a citizen session if present but does not require it.
func optionalCitizenSessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Citizen-Session")
		if token == "" {
			if cookie, err := c.Cookie("citizen_session"); err == nil {
				token = cookie
			}
		}
		if token != "" {
			if session, err := lookupSession(token); err == nil && session != nil {
				c.Set("citizen_session", session)
				c.Set("tenant_id", session.TenantID)
			}
		}
		c.Next()
	}
}

// ─── Enterprise JWT Middleware ────────────────────────────────────────────────
// Used for admin endpoints that require an institutional user JWT
// (same JWT format as Enterprise Core uses).

func enterpriseJWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "unauthorized",
				Message: "Enterprise JWT required for admin endpoints.",
			})
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		if len(enterpriseJWTSecret) == 0 {
			// If no secret configured, allow in development
			log.Println("[WARNING] STATGATE_REGISTRY_JWT_SECRET not set — enterprise auth disabled")
			c.Set("enterprise_user", "dev_admin")
			c.Set("enterprise_role", "admin")
			c.Next()
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return enterpriseJWTSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_token",
				Message: "Enterprise JWT validation failed.",
			})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if sub, ok := claims["sub"].(string); ok {
				c.Set("enterprise_user", sub)
			}
			if role, ok := claims["role"].(string); ok {
				c.Set("enterprise_role", role)
			}
			if tenantID, ok := claims["tenant_id"].(string); ok {
				c.Set("tenant_id", tenantID)
			}
		}
		c.Next()
	}
}

// ─── Context Helpers ──────────────────────────────────────────────────────────

func getCorrelationID(c *gin.Context) string {
	if v, exists := c.Get("correlation_id"); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fmt.Sprintf("sc_%d", time.Now().UnixNano())
}

func getTenantID(c *gin.Context, cfg *Config) string {
	// Try from header first (set by middleware)
	if v, exists := c.Get("tenant_id"); exists {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	// Try query param (e.g., for public routes)
	if t := c.Query("tenant"); t != "" {
		return t
	}
	// Try X-Tenant-ID header
	if t := c.GetHeader("X-Tenant-ID"); t != "" {
		return t
	}
	return cfg.DefaultTenantID
}

func getCitizenSession(c *gin.Context) *CitizenSession {
	if v, exists := c.Get("citizen_session"); exists {
		if s, ok := v.(*CitizenSession); ok {
			return s
		}
	}
	return nil
}

func getEnterpriseUser(c *gin.Context) string {
	if v, exists := c.Get("enterprise_user"); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ─── Audit Recording ──────────────────────────────────────────────────────────

func recordCitizenAudit(c *gin.Context, cfg *Config, actorType, action, resourceType, resourceID string, meta map[string]interface{}) {
	if dbPool == nil {
		return
	}
	actor := "anonymous"
	session := getCitizenSession(c)
	if session != nil {
		actor = "session:" + session.ID
	}
	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)
	id := fmt.Sprintf("caud_%d", time.Now().UnixNano())
	metaJSON, _ := json.Marshal(meta)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx,
		`INSERT INTO citizen_audit_log 
		(id, tenant_id, actor, actor_type, action, resource_type, resource_id, correlation_id, outcome, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'success',$9::jsonb)`,
		id, tenantID, actor, actorType, action, resourceType, resourceID, corrID, string(metaJSON),
	)
	if err != nil {
		log.Printf("audit: write error: %v", err)
	}
}

// ─── Session Lookup ───────────────────────────────────────────────────────────

func lookupSession(token string) (*CitizenSession, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := dbPool.QueryRowContext(ctx,
		`SELECT id, token, tenant_id, created_at, expires_at, last_seen
		 FROM citizen_sessions WHERE token = $1 AND expires_at > NOW()`, token)
	var s CitizenSession
	if err := row.Scan(&s.ID, &s.Token, &s.TenantID, &s.CreatedAt, &s.ExpiresAt, &s.LastSeen); err != nil {
		return nil, err
	}
	// Update last_seen asynchronously
	go func() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel2()
		_, _ = dbPool.ExecContext(ctx2, `UPDATE citizen_sessions SET last_seen=NOW() WHERE id=$1`, s.ID)
	}()
	return &s, nil
}

// ─── Middleware Aliases & Helpers ─────────────────────────────────────────────

func rateLimitMiddleware(cfg *Config) gin.HandlerFunc {
	return publicRateLimitMiddleware(cfg)
}

func citizenAuthMiddleware(cfg *Config) gin.HandlerFunc {
	return citizenSessionMiddleware()
}

func enterpriseAuthMiddleware(cfg *Config) gin.HandlerFunc {
	return enterpriseJWTMiddleware()
}

func buildCanonicalID(tenantID, objectType, id string) string {
	if tenantID == "" {
		tenantID = "default"
	}
	return fmt.Sprintf("%s:statcitizen:%s:%s", tenantID, objectType, id)
}


