package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-backend/configs"
)

const emailVerificationLifetime = 24 * time.Hour

func isProduction() bool {
	return strings.EqualFold(os.Getenv("APP_ENV"), "production") ||
		strings.EqualFold(os.Getenv("GO_ENV"), "production") ||
		strings.EqualFold(os.Getenv("STATGATE_ENV"), "production")
}

func verificationToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	return token, base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

func createAndSendVerification(userID int64, email string) (string, error) {
	token, hash, err := verificationToken()
	if err != nil {
		return "", err
	}
	_, err = configs.DB.Exec(`UPDATE email_verification_tokens SET used_at=NOW() WHERE user_id=$1 AND used_at IS NULL`, userID)
	if err != nil {
		return "", err
	}
	_, err = configs.DB.Exec(`INSERT INTO email_verification_tokens (user_id, token_hash, expires_at) VALUES ($1,$2,$3)`, userID, hash, time.Now().UTC().Add(emailVerificationLifetime))
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(os.Getenv("APP_PUBLIC_URL"), "/")
	if base == "" {
		base = "http://localhost:3007"
	}
	verificationURL := base + "/verify-email?token=" + url.QueryEscape(token)
	body := "Verify your email within 24 hours: " + verificationURL + "\n\nIf you did not create a StatGate account, you can ignore this message."
	return verificationURL, sendRegistryEmail(email, "Verify your StatGate email", body)
}

func VerifyEmail(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification token is required"})
		return
	}
	digest := sha256.Sum256([]byte(token))
	hash := base64.RawURLEncoding.EncodeToString(digest[:])
	result, err := configs.DB.Exec(`UPDATE email_verification_tokens SET used_at=NOW() WHERE token_hash=$1 AND used_at IS NULL AND expires_at > NOW()`, hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification token is invalid or expired"})
		return
	}
	var userID int64
	if err := configs.DB.QueryRow(`SELECT user_id FROM email_verification_tokens WHERE token_hash=$1`, hash).Scan(&userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}
	if _, err := configs.DB.Exec(`UPDATE users SET email_verified=true, "updatedAt"=NOW() WHERE id=$1`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"verified": true, "user_id": userID})
}

func ResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid email is required"})
		return
	}
	var userID int64
	var verified bool
	err := configs.DB.QueryRow(`SELECT id, COALESCE(email_verified, true) FROM users WHERE LOWER(email)=LOWER($1)`, req.Email).Scan(&userID, &verified)
	if err == nil && !verified {
		_, _ = createAndSendVerification(userID, strings.TrimSpace(req.Email))
	}
	// Do not disclose whether an address exists.
	c.JSON(http.StatusOK, gin.H{"message": "If the account requires verification, a new email has been sent."})
}
