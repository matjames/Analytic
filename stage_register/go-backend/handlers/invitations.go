package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var invitationRoles = map[string]bool{
	"viewer": true, "analyst": true, "editor": true, "operator": true,
	"manager": true, "agent": true, "district": true, "district_admin": true,
	"tenant_admin": true, "governance_officer": true,
}

func requireInvitationAdmin(c *gin.Context) bool {
	roleValue, _ := c.Get("user_role")
	role, _ := roleValue.(string)
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin", "superadmin", "platform_admin", "tenant_admin":
		return true
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "administrator access required"})
		return false
	}
}

func invitationToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token, base64.RawURLEncoding.EncodeToString(hash[:]), nil
}

type invitationRecord struct {
	ID           int64      `json:"id"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	Organisation string     `json:"organisation"`
	ExpiresAt    time.Time  `json:"expires_at"`
	AcceptedAt   *time.Time `json:"accepted_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// CreateInvitation creates a single-use invitation and sends it through the
// configured Registry email channel. The raw token is returned only outside
// production so local operators still have a safe manual fallback.
func CreateInvitation(c *gin.Context) {
	if !requireInvitationAdmin(c) {
		return
	}
	var req struct {
		Email         string `json:"email" binding:"required"`
		Role          string `json:"role"`
		Organisation  string `json:"organisation"`
		ExpiresInDays int    `json:"expires_in_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid email is required"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if req.Role == "" {
		req.Role = "viewer"
	}
	if !invitationRoles[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role cannot be invited"})
		return
	}
	if !roleAllowedForAuthenticatedAdmin(c, &req.Role) {
		return
	}
	organisation, ok := scopedOrganisationForWrite(c, &req.Organisation)
	if !ok {
		return
	}
	if req.ExpiresInDays <= 0 || req.ExpiresInDays > 30 {
		req.ExpiresInDays = 7
	}

	var existingID int64
	if err := configs.DB.QueryRow(`SELECT id FROM users WHERE LOWER(email) = $1`, req.Email).Scan(&existingID); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "a user with this email already exists"})
		return
	}
	token, tokenHash, err := invitationToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invitation"})
		return
	}
	invitedBy := c.GetInt64("user_id")
	expiresAt := time.Now().UTC().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
	_, err = configs.DB.Exec(`INSERT INTO user_invitations (email, role, organisation, token_hash, invited_by, expires_at) VALUES ($1,$2,$3,$4,$5,$6)`, req.Email, req.Role, organisation, tokenHash, invitedBy, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save invitation"})
		return
	}
	baseURL := strings.TrimRight(os.Getenv("STATGATE_REGISTRY_PUBLIC_URL"), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(os.Getenv("APP_PUBLIC_URL"), "/")
	}
	acceptURL := "/accept-invitation?token=" + token
	if baseURL != "" {
		acceptURL = baseURL + acceptURL
	}
	body := fmt.Sprintf(
		"You have been invited to StatGate as %s. Accept this invitation before %s:\n\n%s\n\nIf you did not expect this invitation, you can ignore this message.",
		req.Role,
		expiresAt.Format(time.RFC1123),
		acceptURL,
	)
	deliveryErr := sendRegistryEmail(req.Email, "You're invited to StatGate", body)
	if deliveryErr != nil && isProduction() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "invitation email could not be delivered"})
		return
	}

	response := gin.H{"email": req.Email, "role": req.Role, "expires_at": expiresAt, "accept_url": acceptURL}
	if deliveryErr != nil {
		response["delivery_status"] = "not_sent"
		response["delivery_error"] = deliveryErr.Error()
	} else {
		response["delivery_status"] = "sent"
	}
	if !isProduction() {
		response["token"] = token
	}
	c.JSON(http.StatusCreated, response)
}

func ListInvitations(c *gin.Context) {
	if !requireInvitationAdmin(c) {
		return
	}
	where, values, idx := []string{}, []interface{}{}, 1
	var ok bool
	where, values, idx, ok = appendAuthenticatedOrganisationScope(c, "organisation", where, values, idx)
	if !ok {
		return
	}
	_ = idx
	rows, err := configs.DB.Query(`SELECT id,email,role,organisation,expires_at,accepted_at,created_at FROM user_invitations`+combineWhere(where)+` ORDER BY created_at DESC LIMIT 200`, values...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load invitations"})
		return
	}
	defer rows.Close()
	result := make([]invitationRecord, 0)
	for rows.Next() {
		var item invitationRecord
		if err := rows.Scan(&item.ID, &item.Email, &item.Role, &item.Organisation, &item.ExpiresAt, &item.AcceptedAt, &item.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read invitations"})
			return
		}
		result = append(result, item)
	}
	c.JSON(http.StatusOK, gin.H{"invitations": result})
}

func RevokeInvitation(c *gin.Context) {
	if !requireInvitationAdmin(c) {
		return
	}
	where, values, idx := []string{"id = $1", "accepted_at IS NULL"}, []interface{}{c.Param("id")}, 2
	var ok bool
	where, values, idx, ok = appendAuthenticatedOrganisationScope(c, "organisation", where, values, idx)
	if !ok {
		return
	}
	_ = idx
	if _, err := configs.DB.Exec(`UPDATE user_invitations SET expires_at = NOW()`+combineWhere(where), values...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke invitation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}

// AcceptInvitation consumes the raw token exactly once and creates the user.
func AcceptInvitation(c *gin.Context) {
	var req struct {
		Token     string `json:"token" binding:"required"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username" binding:"required"`
		Password  string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token, username and password are required"})
		return
	}
	if !passwordMeetsPolicy(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 10 characters and include upper/lower case and a digit"})
		return
	}
	hash := sha256.Sum256([]byte(strings.TrimSpace(req.Token)))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])
	tx, err := configs.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to accept invitation"})
		return
	}
	defer tx.Rollback()
	var email, role, organisation string
	var invitationID int64
	err = tx.QueryRow(`SELECT id,email,role,organisation FROM user_invitations WHERE token_hash=$1 AND accepted_at IS NULL AND expires_at > NOW() FOR UPDATE`, tokenHash).Scan(&invitationID, &email, &role, &organisation)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invitation is invalid, expired, or already accepted"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate invitation"})
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure password"})
		return
	}
	var user models.User
	err = tx.QueryRow(`INSERT INTO users (role,first_name,last_name,email,username,password,organisation,must_change_password,"createdAt","updatedAt") VALUES ($1,$2,$3,$4,$5,$6,$7,false,NOW(),NOW()) RETURNING id,first_name,last_name,username,email,role,organisation,"createdAt","updatedAt"`, role, strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName), email, strings.TrimSpace(req.Username), string(hashed), organisation).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email or username already exists"})
		return
	}
	if _, err = tx.Exec(`UPDATE user_invitations SET accepted_at=NOW() WHERE id=$1`, invitationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete invitation"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete invitation"})
		return
	}
	token, err := utils.SignUserToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign account"})
		return
	}
	refresh, err := utils.SignRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign session"})
		return
	}
	if err := storeRefreshSession(c, refresh, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist session"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": toSafeUser(user), "token": token, "refresh_token": refresh})
}
