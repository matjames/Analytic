package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"

	"go-backend/configs"

	"github.com/gin-gonic/gin"
)

const refreshSessionLifetime = 30 * 24 * time.Hour

type loginSessionMetadata struct {
	UserAgent  string
	IPHash     string
	GeoCountry string
	GeoRegion  string
}

func refreshTokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func sessionMetadata(c *gin.Context) loginSessionMetadata {
	userAgent := c.GetHeader("User-Agent")
	ip := c.ClientIP()
	secret := os.Getenv("SESSION_METADATA_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	digest := sha256.Sum256([]byte(secret + ":" + ip))
	return loginSessionMetadata{
		UserAgent:  userAgent,
		IPHash:     hex.EncodeToString(digest[:]),
		GeoCountry: geoHeader(c, "CF-IPCountry", "X-Geo-Country", "X-Country-Code", "X-Client-Country"),
		GeoRegion:  geoHeader(c, "X-Geo-Region", "X-Region-Code", "X-Client-Region"),
	}
}

func geoHeader(c *gin.Context, names ...string) string {
	for _, name := range names {
		value := strings.ToUpper(strings.TrimSpace(c.GetHeader(name)))
		if value == "" || value == "XX" || value == "ZZ" || value == "UNKNOWN" {
			continue
		}
		if len(value) > 64 {
			value = value[:64]
		}
		return value
	}
	return ""
}

func detectGeoAnomaly(userID int64, geoCountry string) bool {
	if strings.TrimSpace(geoCountry) == "" || configs.DB == nil {
		return false
	}
	var exists bool
	err := configs.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM refresh_sessions
			WHERE user_id=$1
			  AND revoked_at IS NULL
			  AND expires_at > NOW()
			  AND geo_country <> ''
			  AND geo_country <> $2
		)`, userID, geoCountry).Scan(&exists)
	return err == nil && exists
}

func storeRefreshSession(c *gin.Context, token string, userID int64) error {
	metadata := sessionMetadata(c)
	geoAnomaly := detectGeoAnomaly(userID, metadata.GeoCountry)
	_, err := configs.DB.Exec(
		`INSERT INTO refresh_sessions (user_id, token_hash, expires_at, user_agent, ip_hash, geo_country, geo_region, geo_anomaly, last_seen_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
		userID,
		refreshTokenHash(token),
		time.Now().UTC().Add(refreshSessionLifetime),
		metadata.UserAgent,
		metadata.IPHash,
		metadata.GeoCountry,
		metadata.GeoRegion,
		geoAnomaly,
	)
	return err
}

// consumeRefreshSession makes rotation server-enforced: a refresh token can
// only be exchanged once, even while its JWT is otherwise still valid.
func consumeRefreshSession(token string, userID int64) bool {
	result, err := configs.DB.Exec(`UPDATE refresh_sessions SET revoked_at=NOW() WHERE user_id=$1 AND token_hash=$2 AND revoked_at IS NULL AND expires_at > NOW()`, userID, refreshTokenHash(token))
	if err != nil {
		return false
	}
	count, err := result.RowsAffected()
	return err == nil && count == 1
}

func revokeUserSessions(userID int64) error {
	_, err := configs.DB.Exec(`UPDATE refresh_sessions SET revoked_at=NOW() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return err
}

func recoveryCodeHash(code string) string {
	digest := sha256.Sum256([]byte(code))
	return hex.EncodeToString(digest[:])
}

func generateRecoveryCodes() ([]string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codes := make([]string, 10)
	for i := range codes {
		raw := make([]byte, 10)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		for j := range raw {
			raw[j] = alphabet[int(raw[j])%len(alphabet)]
		}
		codes[i] = string(raw)
	}
	return codes, nil
}

func replaceRecoveryCodes(userID int64) ([]string, error) {
	codes, err := generateRecoveryCodes()
	if err != nil {
		return nil, err
	}
	tx, err := configs.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM mfa_recovery_codes WHERE user_id=$1`, userID); err != nil {
		return nil, err
	}
	for _, code := range codes {
		if _, err = tx.Exec(`INSERT INTO mfa_recovery_codes (user_id,code_hash) VALUES ($1,$2)`, userID, recoveryCodeHash(code)); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return codes, nil
}

func consumeRecoveryCode(userID int64, code string) bool {
	result, err := configs.DB.Exec(`UPDATE mfa_recovery_codes SET used_at=NOW() WHERE user_id=$1 AND code_hash=$2 AND used_at IS NULL`, userID, recoveryCodeHash(code))
	if err != nil {
		return false
	}
	count, err := result.RowsAffected()
	return err == nil && count == 1
}

func LogoutUser(c *gin.Context) {
	if err := revokeUserSessions(c.GetInt64("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke sessions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logged_out": true})
}

func ListSessions(c *gin.Context) {
	rows, err := configs.DB.Query(`SELECT id,created_at,expires_at,revoked_at,user_agent,last_seen_at,geo_country,geo_region,geo_anomaly FROM refresh_sessions WHERE user_id=$1 ORDER BY created_at DESC`, c.GetInt64("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load sessions"})
		return
	}
	defer rows.Close()
	type session struct {
		ID         int64      `json:"id"`
		CreatedAt  time.Time  `json:"created_at"`
		ExpiresAt  time.Time  `json:"expires_at"`
		RevokedAt  *time.Time `json:"revoked_at,omitempty"`
		UserAgent  string     `json:"user_agent,omitempty"`
		LastSeenAt time.Time  `json:"last_seen_at"`
		GeoCountry string     `json:"geo_country,omitempty"`
		GeoRegion  string     `json:"geo_region,omitempty"`
		GeoAnomaly bool       `json:"geo_anomaly"`
	}
	result := make([]session, 0)
	for rows.Next() {
		var item session
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.ExpiresAt, &item.RevokedAt, &item.UserAgent, &item.LastSeenAt, &item.GeoCountry, &item.GeoRegion, &item.GeoAnomaly); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read sessions"})
			return
		}
		result = append(result, item)
	}
	c.JSON(http.StatusOK, gin.H{"sessions": result})
}

func RevokeSession(c *gin.Context) {
	result, err := configs.DB.Exec(`UPDATE refresh_sessions SET revoked_at=NOW() WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL`, c.Param("id"), c.GetInt64("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke session"})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "active session not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}
