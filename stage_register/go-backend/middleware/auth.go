package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
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

		tokenStr := parts[1]

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		
		// Extract userId, role, and districtId from token (matching Node.js)
		if userId, ok := claims["userId"].(float64); ok {
			c.Set("user_id", int64(userId))
		} else if userId, ok := claims["user_id"].(float64); ok {
			// Fallback for backward compatibility
			c.Set("user_id", int64(userId))
		}

		if role, ok := claims["role"].(string); ok {
			c.Set("user_role", role)
			// Also set "role" for backward compatibility
			c.Set("role", role)
		}

		if districtIdVal, ok := claims["districtId"]; ok {
			if districtId, ok := districtIdVal.(string); ok {
				// districtId is now stored as mfl_uid string
				c.Set("user_district_id", districtId)
				c.Set("district_id", districtId)
			} else if districtId, ok := districtIdVal.(float64); ok {
				// Backward compatibility: convert old int64 tokens to string
				c.Set("user_district_id", strconv.FormatInt(int64(districtId), 10))
				c.Set("district_id", strconv.FormatInt(int64(districtId), 10))
			} else if districtIdVal == nil {
				c.Set("user_district_id", nil)
				c.Set("district_id", nil)
			}
		} else if districtIdVal, ok := claims["district_id"]; ok {
			// Fallback for backward compatibility
			if districtId, ok := districtIdVal.(string); ok {
				c.Set("user_district_id", districtId)
				c.Set("district_id", districtId)
			} else if districtId, ok := districtIdVal.(float64); ok {
				c.Set("user_district_id", strconv.FormatInt(int64(districtId), 10))
				c.Set("district_id", strconv.FormatInt(int64(districtId), 10))
			} else if districtIdVal == nil {
				c.Set("user_district_id", nil)
				c.Set("district_id", nil)
			}
		}

		c.Next()
	}
}
