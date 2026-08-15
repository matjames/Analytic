package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// statGovValidRoles is the canonical StatGate role whitelist.
var statGovValidRoles = map[string]bool{
	"viewer": true, "analyst": true, "editor": true, "operator": true,
	"manager": true, "agent": true, "district": true, "district_admin": true,
	"tenant_admin": true, "governance_officer": true, "property_officer": true,
	"admin": true, "superadmin": true, "platform_admin": true,
}

// statGovEnv helper: reads env var with fallback.
func statGovEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func registryAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow public health, ready, metrics endpoints without auth
		path := c.Request.URL.Path
		if path == "/health" || path == "/ready" || path == "/metrics" {
			c.Next()
			return
		}

		// Fail closed: the shared StatGate JWT secret is mandatory. There is
		// NO development fallback and NO demo/identity bypass (SG-SEC-2026-08).
		secret := strings.TrimSpace(os.Getenv("STATGATE_REGISTRY_JWT_SECRET"))
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":   "service_misconfigured",
				"message": "STATGATE_REGISTRY_JWT_SECRET is not configured",
			})
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			return
		}

		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		if tokenStr == "" || strings.HasPrefix(tokenStr, "demo_") {
			// Demo tokens are rejected on all environments.
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		issuer := statGovEnv("STATGATE_JWT_ISSUER", "statgate-registry")
		audience := statGovEnv("STATGATE_JWT_AUDIENCE", "statgate")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unsupported signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		},
			jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithIssuer(issuer),
			jwt.WithAudience(audience),
		)

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		userID := ""
		if uid, ok := claims["userId"].(string); ok && uid != "" {
			userID = uid
		} else if uid, ok := claims["user_id"].(string); ok && uid != "" {
			userID = uid
		} else if sub, ok := claims["sub"].(string); ok && sub != "" {
			userID = sub
		}
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token missing user identity"})
			return
		}
		c.Set("user_id", userID)

		if name, ok := claims["name"].(string); ok && name != "" {
			c.Set("user_name", name)
		} else {
			c.Set("user_name", userID)
		}

		role := ""
		if r, ok := claims["role"].(string); ok {
			role = strings.TrimSpace(r)
		}
		if !statGovValidRoles[strings.ToLower(role)] {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token contains an unknown role"})
			return
		}
		c.Set("user_role", role)

		tenant := strings.TrimSpace(claimString(claims, "tenant_id", "tenantId"))
		if tenant == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token missing tenant"})
			return
		}
		c.Set("tenant_id", tenant)

		if org, ok := claims["org_id"].(string); ok {
			c.Set("org_id", org)
		}

		c.Next()
	}
}

// claimString safely reads a string claim under any of the given names.
func claimString(claims jwt.MapClaims, names ...string) string {
	for _, name := range names {
		if value, ok := claims[name]; ok {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

func getAuthContext(c *gin.Context) (userID, userName, userRole, tenantID string) {
	if uid, exists := c.Get("user_id"); exists {
		if s, ok := uid.(string); ok {
			userID = s
		}
	}
	if name, exists := c.Get("user_name"); exists {
		if s, ok := name.(string); ok {
			userName = s
		}
	}
	if role, exists := c.Get("user_role"); exists {
		if s, ok := role.(string); ok {
			userRole = s
		}
	}
	if tenant, exists := c.Get("tenant_id"); exists {
		if s, ok := tenant.(string); ok {
			tenantID = s
		}
	}
	if userID == "" {
		userID = "user-system"
	}
	return
}
