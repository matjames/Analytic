package middleware

import (
	"database/sql"
	"encoding/base64"
	"net/http"
	"strings"

	"go-backend/configs"
	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
)

// BasicAuthRequired validates username and password from HTTP Basic Auth header
func BasicAuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			return
		}

		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Basic" {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
			return
		}

		credentials := strings.SplitN(string(decoded), ":", 2)
		if len(credentials) != 2 {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials format"})
			return
		}

		username := credentials[0]
		password := credentials[1]

		// Query user by username or email
		var dbUserID int64
		var dbPasswordHash string
		err = configs.DB.QueryRow(
			`SELECT id, password FROM users WHERE username = $1 OR email = $1 LIMIT 1`,
			username,
		).Scan(&dbUserID, &dbPasswordHash)

		if err == sql.ErrNoRows {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Verify password
		err = bcrypt.CompareHashAndPassword([]byte(dbPasswordHash), []byte(password))
		if err != nil {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Set user context
		c.Set("user_id", dbUserID)
		c.Set("username", username)

		c.Next()
	}
}

