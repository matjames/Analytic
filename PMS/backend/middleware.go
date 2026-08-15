package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func registryAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow public health/ready/metrics probes without auth
		path := c.Request.URL.Path
		if path == "/health" || path == "/ready" || path == "/metrics" {
			c.Next()
			return
		}

		// Fail closed (SG-SEC-2026-08): no hardcoded default secret. If the
		// shared identity secret is missing, the service is misconfigured.
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		if tokenStr == "" || strings.HasPrefix(tokenStr, "demo_") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		issuer := pmsEnv("STATGATE_JWT_ISSUER", "statgate-registry")
		audience := pmsEnv("STATGATE_JWT_AUDIENCE", "statgate")

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

		role := ""
		if r, ok := claims["role"].(string); ok {
			role = strings.TrimSpace(r)
		}
		if !validPlatformRole(role) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token contains an unknown role"})
			return
		}
		c.Set("user_role", role)

		tenant := strings.TrimSpace(pmsClaimString(claims, "tenant_id", "tenantId"))
		if tenant == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token missing tenant"})
			return
		}
		c.Set("tenant_id", tenant)

		if org, ok := claims["org_id"].(string); ok {
			c.Set("org_id", org)
		}
		c.Set("email", pmsClaimString(claims, "email"))

		c.Next()
	}
}

// validPlatformRole is the canonical role whitelist shared across services.
func validPlatformRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "viewer", "analyst", "editor", "operator", "manager", "agent",
		"district", "district_admin", "tenant_admin", "governance_officer",
		"property_officer", "admin", "superadmin", "platform_admin":
		return true
	}
	return false
}

// pmsClaimString reads a string claim under any of the given names.
func pmsClaimString(claims jwt.MapClaims, names ...string) string {
	for _, name := range names {
		if value, ok := claims[name]; ok {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

// pmsEnv reads an env var with a fallback.
func pmsEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
