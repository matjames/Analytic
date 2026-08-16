package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Correlation ID Middleware ────────────────────────────────────
// Injects/propagates X-Correlation-ID and X-Request-ID on every request.

func correlationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		corrID := c.GetHeader("X-Correlation-ID")
		if corrID == "" {
			corrID = fmt.Sprintf("corr_%d", time.Now().UnixNano())
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

// ─── Security Headers Middleware ─────────────────────────────────
// Applies a baseline set of HTTP security headers to every response.

func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// HSTS: 1 year, include subdomains. Only effective over HTTPS.
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}

// ─── Request Size Middleware ──────────────────────────────────────
// Rejects payloads exceeding STATGATE_MAX_REQUEST_BODY_KB (default 1024 KB).

func requestSizeMiddleware() gin.HandlerFunc {
	maxKB := int64(parseIntDefault(getEnvValue("STATGATE_MAX_REQUEST_BODY_KB"), 1024))
	maxBytes := maxKB * 1024
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "request_too_large",
				"message": fmt.Sprintf("Request body exceeds %d KB limit", maxKB),
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// ─── Rate Limiting Middleware ─────────────────────────────────────
// Token-bucket rate limiter keyed by remote IP.
// STATGATE_RATE_LIMIT_RPM controls max requests per minute (default 300).

type rateBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
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
	rateMu      sync.RWMutex
	rateBuckets = make(map[string]*rateBucket)
)

func getRateBucket(key string, rpm float64) *rateBucket {
	rateMu.RLock()
	b, ok := rateBuckets[key]
	rateMu.RUnlock()
	if ok {
		return b
	}
	rateMu.Lock()
	defer rateMu.Unlock()
	b = &rateBucket{
		tokens:     rpm / 60,
		maxTokens:  rpm / 60 * 5, // burst up to 5 seconds worth
		refillRate: rpm / 60,
		lastRefill: time.Now(),
	}
	rateBuckets[key] = b
	return b
}

func rateLimitMiddleware() gin.HandlerFunc {
	rpm := float64(parseIntDefault(getEnvValue("STATGATE_RATE_LIMIT_RPM"), 300))
	// Background cleanup: evict stale buckets every 5 minutes
	go func() {
		for range time.Tick(5 * time.Minute) {
			rateMu.Lock()
			for k, b := range rateBuckets {
				b.mu.Lock()
				idle := time.Since(b.lastRefill) > 10*time.Minute
				b.mu.Unlock()
				if idle {
					delete(rateBuckets, k)
				}
			}
			rateMu.Unlock()
		}
	}()
	return func(c *gin.Context) {
		ip := c.ClientIP()
		b := getRateBucket(ip, rpm)
		if !b.allow() {
			c.Header("Retry-After", "5")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please slow down.",
			})
			return
		}
		c.Next()
	}
}

// ─── JWT Validation Middleware ────────────────────────────────────
// Validates Registry-issued JWTs using STATGATE_REGISTRY_JWT_SECRET.
// Extracts claims into Gin context: user_id, tenant_id, role.
//
// This is a slim HS256 verifier that avoids external JWT libraries,
// keeping the dependency footprint minimal and auditable.

type jwtClaims struct {
	Sub      string `json:"sub"`
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	OrgID    string `json:"org_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Iss      string `json:"iss"`
	Aud      string `json:"aud"`
	Iat      int64  `json:"iat"`
	Nbf      int64  `json:"nbf"`
	Exp      int64  `json:"exp"`
}

// validRoles is the canonical StatGate role whitelist (see
// docs/security/JWT_STANDARD.md). Tokens carrying any other role are
// rejected by jwtAuthMiddleware.
var validRoles = map[string]bool{
	"viewer": true, "analyst": true, "editor": true, "operator": true,
	"manager": true, "agent": true, "district": true, "district_admin": true,
	"tenant_admin": true, "governance_officer": true, "property_officer": true,
	"admin": true, "superadmin": true, "platform_admin": true,
}

// jwtConfig returns the canonical issuer/audience used by every StatGate
// service. Both can be overridden for non-standard deployments but MUST be
// identical across all services of a deployment.
func jwtConfig() (issuer, audience string) {
	issuer = getEnvValue("STATGATE_JWT_ISSUER")
	if issuer == "" {
		issuer = "statgate-registry"
	}
	audience = getEnvValue("STATGATE_JWT_AUDIENCE")
	if audience == "" {
		audience = "statgate"
	}
	return issuer, audience
}

func parseJWT(tokenStr, secret string) (*jwtClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token")
	}

	// Decode and verify the JOSE header. Only HS256 is accepted.
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("malformed header")
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("unparseable header")
	}
	if header.Alg != "HS256" {
		return nil, fmt.Errorf("unsupported algorithm")
	}

	// Verify signature with the shared secret (unsigned tokens always fail).
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode and unmarshal claims
	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("malformed claims")
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimBytes, &claims); err != nil {
		return nil, fmt.Errorf("unparseable claims")
	}

	now := time.Now().Unix()

	// exp is mandatory and must be in the future.
	if claims.Exp == 0 || now > claims.Exp {
		return nil, fmt.Errorf("token expired or missing exp")
	}
	// nbf: reject tokens used before they are valid.
	if claims.Nbf != 0 && now < claims.Nbf {
		return nil, fmt.Errorf("token not yet valid")
	}

	issuer, audience := jwtConfig()
	if claims.Iss != issuer {
		return nil, fmt.Errorf("invalid issuer")
	}
	if claims.Aud != audience {
		return nil, fmt.Errorf("invalid audience")
	}
	if strings.TrimSpace(claims.Sub) == "" && strings.TrimSpace(claims.UserID) == "" {
		return nil, fmt.Errorf("token missing subject")
	}

	// Canonical contract: a token MUST carry a tenant and a known role.
	if strings.TrimSpace(claims.TenantID) == "" {
		return nil, fmt.Errorf("token missing tenant")
	}
	if !validRoles[strings.ToLower(claims.Role)] {
		return nil, fmt.Errorf("token contains an unknown role")
	}

	return &claims, nil
}

func jwtAuthMiddleware() gin.HandlerFunc {
	jwtSecret := strings.TrimSpace(getEnvValue("STATGATE_REGISTRY_JWT_SECRET"))
	internalAPIKey := strings.TrimSpace(getEnvValue("STATGATE_INTERNAL_API_KEY"))
	if jwtSecret == "" {
		log.Println("security: STATGATE_REGISTRY_JWT_SECRET not set — ALL JWT-authenticated requests will be rejected (fail-closed)")
	}
	return func(c *gin.Context) {
		// Allow internal health/readiness probes without auth
		if isProbeEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}
		// Trusted StatGate services use the rotating internal key over the private
		// compose network. This lets Integration Fabric publish audited platform
		// events without forging an end-user JWT.
		if internalAPIKey != "" && hmac.Equal([]byte(c.GetHeader("X-Internal-API-Key")), []byte(internalAPIKey)) {
			tenantID := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
			if tenantID == "" { tenantID = "default" }
			c.Set("user_id", "integration-fabric")
			c.Set("tenant_id", tenantID)
			c.Set("org_id", "")
			c.Set("role", "service")
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authorization header with Bearer token is required",
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if jwtSecret == "" {
			// Fail closed: never trust client-supplied X-User-ID / X-Tenant-ID.
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":   "service_misconfigured",
				"message": "Authentication service is not configured",
			})
			return
		}

		claims, err := parseJWT(tokenStr, jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_token",
				"message": err.Error(),
			})
			return
		}

		// Resolve user ID: prefer dedicated claim, fall back to sub.
		userID := claims.UserID
		if userID == "" {
			userID = claims.Sub
		}
		if strings.TrimSpace(userID) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "message": "token missing user identity"})
			return
		}
		if strings.TrimSpace(claims.TenantID) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "message": "token missing tenant"})
			return
		}
		if !validRoles[strings.ToLower(claims.Role)] {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "message": "token contains an unknown role"})
			return
		}

		c.Set("user_id", userID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("org_id", claims.OrgID)
		c.Set("role", claims.Role)
		c.Set("email", claims.Email)
		c.Next()
	}
}
// ─── Tenant Isolation Middleware ──────────────────────────────────
// Validates that the X-Tenant-ID header (if provided) matches the JWT's
// tenant_id claim, preventing cross-tenant data access via header injection.

func tenantIsolationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtTenant, _ := c.Get("tenant_id")
		headerTenant := c.GetHeader("X-Tenant-ID")
		if headerTenant != "" && jwtTenant != nil && jwtTenant.(string) != "" {
			if headerTenant != jwtTenant.(string) {
				corrID, _ := c.Get("correlation_id")
				log.Printf("security: tenant isolation violation — header=%s jwt=%s corr=%v ip=%s",
					headerTenant, jwtTenant, corrID, c.ClientIP())
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "tenant_mismatch",
					"message": "X-Tenant-ID does not match authenticated tenant",
				})
				return
			}
		}

		// SG-SEC-2026-08: overwrite client-supplied identity headers with the
		// values derived from the VERIFIED JWT so downstream handlers can never
		// act on caller-claimed identities.
		c.Request.Header.Set("X-User-ID", getContextUserID(c))
		c.Request.Header.Set("X-Tenant-ID", getContextTenantID(c))
		c.Request.Header.Set("X-User-Role", getContextRole(c))
		if role, exists := c.Get("role"); exists {
			if s, ok := role.(string); ok {
				c.Request.Header.Set("X-User-Role", s)
			}
		}
		c.Next()
	}
}

// ─── Metrics Middleware ───────────────────────────────────────────
// Tracks per-request latency and increments platform metrics counters.

func requestMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := c.Writer.Status()
		latency := time.Since(start)
		metricsCollector.incRequests()
		if status >= 500 {
			metricsCollector.incErrors()
		}
		// Log slow requests (> 2s)
		if latency > 2*time.Second {
			log.Printf("observability: slow request %s %s status=%d latency=%v",
				c.Request.Method, c.Request.URL.Path, status, latency)
		}
	}
}

// ─── Helpers ─────────────────────────────────────────────────────

// isProbeEndpoint returns true for health/readiness/liveness endpoints
// that must be accessible without authentication.
func isProbeEndpoint(path string) bool {
	switch path {
	case "/health", "/ready", "/live", "/metrics", "/api/info":
		return true
	}
	return false
}

// getContextUserID extracts the authenticated user ID from Gin context.
// It NEVER falls back to client-supplied identity headers (SG-SEC-2026-08).
func getContextUserID(c *gin.Context) string {
	if uid, exists := c.Get("user_id"); exists {
		if s, ok := uid.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// getContextTenantID extracts the authenticated tenant ID from Gin context.
// It NEVER falls back to the X-Tenant-ID header (tenant boundaries are
// derived exclusively from the verified JWT).
func getContextTenantID(c *gin.Context) string {
	if tid, exists := c.Get("tenant_id"); exists {
		if s, ok := tid.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// getContextRole extracts the authenticated role from Gin context.
func getContextRole(c *gin.Context) string {
	if role, exists := c.Get("role"); exists {
		if s, ok := role.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// requireRole aborts if the authenticated role does not match any of the allowed roles.
func requireRole(c *gin.Context, allowedRoles ...string) bool {
	role := getContextRole(c)
	for _, r := range allowedRoles {
		if role == r {
			return true
		}
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error":   "forbidden",
		"message": fmt.Sprintf("Role '%s' is not permitted to perform this action", role),
	})
	return false
}
