package middleware

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// InternalServiceRequired protects server-to-server endpoints. It is separate
// from user JWT authentication so a StatChat directory refresh never depends
// on a browser session, while the endpoint remains inaccessible to clients.
func InternalServiceRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := strings.TrimSpace(os.Getenv("STATGATE_INTERNAL_API_KEY"))
		provided := strings.TrimSpace(c.GetHeader("X-StatGate-Internal-Key"))
		if expected == "" || provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid internal service credential"})
			return
		}
		c.Set("internal_service", true)
		c.Next()
	}
}

// authEnv reads an env var with a fallback.
func authEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// registryValidRoles is the canonical role whitelist (see JWT_STANDARD.md).
var registryValidRoles = map[string]bool{
	"viewer": true, "analyst": true, "editor": true, "operator": true,
	"manager": true, "agent": true, "district": true, "district_admin": true,
	"tenant_admin": true, "governance_officer": true, "property_officer": true,
	"admin": true, "superadmin": true, "platform_admin": true,
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Fail closed: no hardcoded default secret (SG-SEC-2026-08).
		secret := authEnv("STATGATE_REGISTRY_JWT_SECRET", "")
		if secret == "" {
			secret = authEnv("JWT_SECRET", "")
		}
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		if tokenStr == "" || strings.HasPrefix(tokenStr, "demo_") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		issuer := authEnv("STATGATE_JWT_ISSUER", "statgate-registry")
		audience := authEnv("STATGATE_JWT_AUDIENCE", "statgate")

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

		// Required subject
		if sub, ok := claims["sub"].(string); ok && sub != "" {
			if parsed, err := strconv.ParseInt(sub, 10, 64); err == nil {
				c.Set("user_id", parsed)
			} else {
				c.Set("user_id", sub)
			}
		} else if userId, ok := claims["userId"].(float64); ok {
			c.Set("user_id", int64(userId))
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token missing user identity"})
			return
		}

		// Validate role against the canonical whitelist
		role := ""
		if r, ok := claims["role"].(string); ok {
			role = strings.TrimSpace(r)
		}
		if !registryValidRoles[strings.ToLower(role)] {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token contains an unknown role"})
			return
		}
		c.Set("user_role", role)
		c.Set("role", role)

		if tenant, ok := claims["tenant_id"].(string); ok && tenant != "" {
			c.Set("tenant_id", tenant)
		} else if districtIdVal, ok := claims["districtId"]; ok {
			if districtId, ok := districtIdVal.(string); ok {
				c.Set("tenant_id", districtId)
			}
		}

		if districtIdVal, ok := claims["districtId"]; ok {
			if districtId, ok := districtIdVal.(string); ok {
				c.Set("user_district_id", districtId)
				c.Set("district_id", districtId)
			} else if districtId, ok := districtIdVal.(float64); ok {
				c.Set("user_district_id", strconv.FormatInt(int64(districtId), 10))
				c.Set("district_id", strconv.FormatInt(int64(districtId), 10))
			}
		} else if districtIdVal, ok := claims["district_id"]; ok {
			if districtId, ok := districtIdVal.(string); ok {
				c.Set("user_district_id", districtId)
				c.Set("district_id", districtId)
			}
		}

		c.Next()
	}
}
