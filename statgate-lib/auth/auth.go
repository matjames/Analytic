package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ContextKeyUser   contextKey = "statgate_user"
	ContextKeyTenant contextKey = "statgate_tenant"
	ContextKeyOrg    contextKey = "statgate_org"
	ContextKeyRole   contextKey = "statgate_role"
	ContextKeyClaims contextKey = "statgate_claims"
)

// UserContext holds standard identity attributes across StatGate applications.
type UserContext struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	TenantID  string    `json:"tenant_id"`
	OrgID     string    `json:"org_id,omitempty"`
	Role      string    `json:"role"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CanonicalRoles lists authorized StatGate roles.
var CanonicalRoles = map[string]bool{
	"viewer":             true,
	"analyst":            true,
	"editor":             true,
	"operator":           true,
	"manager":            true,
	"agent":              true,
	"district":           true,
	"district_admin":     true,
	"tenant_admin":       true,
	"governance_officer": true,
	"property_officer":   true,
	"admin":              true,
	"superadmin":         true,
	"platform_admin":     true,
}

// Validator manages JWT verification with zero hardcoded defaults.
type Validator struct {
	Secret   []byte
	Issuer   string
	Audience string
}

// NewValidator creates a validator requiring an explicit secret.
func NewValidator(secret, issuer, audience string) (*Validator, error) {
	s := strings.TrimSpace(secret)
	if s == "" {
		s = strings.TrimSpace(os.Getenv("STATGATE_REGISTRY_JWT_SECRET"))
	}
	if s == "" {
		return nil, errors.New("STATGATE_REGISTRY_JWT_SECRET is required; zero-default enforcement active")
	}

	if issuer == "" {
		issuer = os.Getenv("STATGATE_JWT_ISSUER")
		if issuer == "" {
			issuer = "statgate-registry"
		}
	}
	if audience == "" {
		audience = os.Getenv("STATGATE_JWT_AUDIENCE")
		if audience == "" {
			audience = "statgate"
		}
	}

	return &Validator{
		Secret:   []byte(s),
		Issuer:   issuer,
		Audience: audience,
	}, nil
}

// ValidateToken parses and validates a raw JWT string.
func (v *Validator) ValidateToken(tokenStr string) (*UserContext, error) {
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		return nil, errors.New("token is empty")
	}
	if strings.HasPrefix(tokenStr, "demo_") || strings.HasPrefix(tokenStr, "mock_") {
		return nil, errors.New("mock and demo tokens are rejected in enterprise mode")
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unsupported signing method: %v", t.Header["alg"])
		}
		return v.Secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(v.Issuer),
		jwt.WithAudience(v.Audience),
	)

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	userID := extractClaim(claims, "userId", "user_id", "sub")
	if userID == "" {
		return nil, errors.New("token missing user identity")
	}

	role := strings.ToLower(strings.TrimSpace(extractClaim(claims, "role")))
	if !CanonicalRoles[role] {
		return nil, fmt.Errorf("unauthorized role: %s", role)
	}

	tenantID := extractClaim(claims, "tenant_id", "tenantId")
	if tenantID == "" {
		return nil, errors.New("token missing tenant context")
	}

	uCtx := &UserContext{
		UserID:   userID,
		Email:    extractClaim(claims, "email"),
		TenantID: tenantID,
		OrgID:    extractClaim(claims, "org_id", "orgId"),
		Role:     role,
	}

	if exp, err := claims.GetExpirationTime(); err == nil && exp != nil {
		uCtx.ExpiresAt = exp.Time
	}
	if iat, err := claims.GetIssuedAt(); err == nil && iat != nil {
		uCtx.IssuedAt = iat.Time
	}

	return uCtx, nil
}

// GinMiddleware creates a Gin authentication filter.
func (v *Validator) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health" || path == "/ready" || path == "/live" || path == "/metrics" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		uCtx, err := v.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set("user_id", uCtx.UserID)
		c.Set("user_role", uCtx.Role)
		c.Set("tenant_id", uCtx.TenantID)
		c.Set("org_id", uCtx.OrgID)
		c.Set("email", uCtx.Email)
		c.Set("user_context", uCtx)

		c.Next()
	}
}

// HTTPMiddleware creates a standard net/http middleware.
func (v *Validator) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/health" || path == "/ready" || path == "/live" || path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		uCtx, err := v.ValidateToken(parts[1])
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ContextKeyUser, uCtx)
		ctx = context.WithValue(ctx, ContextKeyTenant, uCtx.TenantID)
		ctx = context.WithValue(ctx, ContextKeyOrg, uCtx.OrgID)
		ctx = context.WithValue(ctx, ContextKeyRole, uCtx.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext retrieves UserContext from request context.
func GetUserFromContext(ctx context.Context) (*UserContext, bool) {
	u, ok := ctx.Value(ContextKeyUser).(*UserContext)
	return u, ok
}

func extractClaim(claims jwt.MapClaims, names ...string) string {
	for _, n := range names {
		if val, ok := claims[n]; ok {
			if s, ok := val.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}
