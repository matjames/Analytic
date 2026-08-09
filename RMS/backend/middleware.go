package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func registryAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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

		tokenStr := parts[1]
		secret := os.Getenv("STATGATE_REGISTRY_JWT_SECRET")
		if secret == "" {
			secret = "REDACTED_PLACEHOLDER"
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		if userID, ok := claims["userId"].(string); ok && userID != "" {
			c.Set("user_id", userID)
		} else if userID, ok := claims["user_id"].(string); ok && userID != "" {
			c.Set("user_id", userID)
		} else if sub, ok := claims["sub"].(string); ok && sub != "" {
			c.Set("user_id", sub)
		}

		if role, ok := claims["role"].(string); ok {
			c.Set("user_role", role)
		}

		c.Next()
	}
}
