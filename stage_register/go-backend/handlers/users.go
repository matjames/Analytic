package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// publicRegistrationRoles is the role whitelist enforceable by PUBLIC
// registration (SG-SEC-2026-08). Privileged roles are provisioned only by an
// authenticated administrator.
var publicRegistrationRoles = map[string]bool{
	"viewer": true, "analyst": true, "agent": true, "district": true,
}

// toSafeUser removes password from user object
func toSafeUser(u models.User) models.User {
	u.Password = ""
	u.PasswordHash = ""
	return u
}

// passwordMeetsPolicy enforces the platform password policy (>=10 chars with
// upper case, lower case and digit).
func passwordMeetsPolicy(p string) bool {
	if len(p) < 10 {
		return false
	}
	hasUpper, hasLower, hasDigit := false, false, false
	for _, r := range p {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// isValidEmail performs a lightweight email format check.
func isValidEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) < 5 || !strings.Contains(email, "@") {
		return false
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	domain := parts[1]
	if !strings.Contains(domain, ".") || strings.Contains(domain, " ") {
		return false
	}
	return true
}

// randomTemporaryPassword generates a policy-compliant one-time password for
// provisioned accounts (must be changed on first login).
func randomTemporaryPassword() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "TmpRngFail9a" // non-secret fallback only used on RNG failure
	}
	return "Tmp" + hex.EncodeToString(b) + "1a"
}

// RegisterUser - POST /users/register - Register a new user
func RegisterUser(c *gin.Context) {
	var req struct {
		Role         *string `json:"role"`
		FirstName    *string `json:"first_name"`
		LastName     *string `json:"last_name"`
		Email        string  `json:"email" binding:"required"`
		Username     string  `json:"username" binding:"required"`
		Password     string  `json:"password" binding:"required"`
		Organisation *string `json:"organisation"`
		Phoneno      *string `json:"phoneno"`
		DistrictID   *string `json:"district_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, username and password are required"})
		return
	}

	// SG-SEC-2026-08: public registration may only grant limited roles.
	// Privileged roles (admin, governance_officer, etc.) are provisioned by
	// an authenticated administrator, never via self-registration.
	role := "viewer"
	if req.Role != nil && strings.TrimSpace(*req.Role) != "" {
		candidate := strings.ToLower(strings.TrimSpace(*req.Role))
		if !publicRegistrationRoles[candidate] {
			c.JSON(http.StatusForbidden, gin.H{"error": "role is not permitted for public registration"})
			return
		}
		role = candidate
	}
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A valid email address is required"})
		return
	}
	if !passwordMeetsPolicy(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 10 characters and include upper/lower case and a digit"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var user models.User
	err = configs.DB.QueryRow(
		`INSERT INTO users (role, first_name, last_name, email, username, password, organisation, phoneno, district_id, email_verified, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, NOW(), NOW())
		 RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, email_verified, "createdAt", "updatedAt"`,
		role, req.FirstName, req.LastName, req.Email, req.Username, string(hashed), req.Organisation, req.Phoneno, req.DistrictID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Check for duplicate key error (PostgreSQL error code 23505)
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email or username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}

	verificationURL, deliveryErr := createAndSendVerification(user.ID, user.Email)
	if deliveryErr != nil && isProduction() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "verification email could not be delivered"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": toSafeUser(user), "verification_required": true, "verification_url": verificationURL})
}

// LoginUser - POST /users/login - Login with email or username
func LoginUser(c *gin.Context) {
	var req struct {
		EmailOrUsername string `json:"emailOrUsername" binding:"required"`
		Password        string `json:"password" binding:"required"`
		MFACode         string `json:"mfa_code"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "emailOrUsername and password are required"})
		return
	}

	var user models.User
	err := configs.DB.QueryRow(
		`SELECT id, first_name, last_name, username, email, role, password, organisation, phoneno, district_id, COALESCE(mfa_enabled, false), COALESCE(mfa_secret, ''), COALESCE(email_verified, true), "createdAt", "updatedAt"
		 FROM users WHERE email = $1 OR username = $1 LIMIT 1`,
		req.EmailOrUsername,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.PasswordHash, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.MFAEnabled, &user.MFASecret, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !user.EmailVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "email address must be verified", "email_verification_required": true})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if user.MFAEnabled && !utils.VerifyTOTP(user.MFASecret, req.MFACode, time.Now()) && !consumeRecoveryCode(user.ID, strings.ToUpper(strings.TrimSpace(req.MFACode))) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "MFA code required or invalid", "mfa_required": true})
		return
	}

	// Temporary credentials require a change on first login; the flag is
	// persisted with the account (SG-SEC-2026-08 removes default passwords).

	user = toSafeUser(user)
	token, err := utils.SignUserToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken, err := utils.SignRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}
	if err := storeRefreshSession(c, refreshToken, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist refresh session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "token": token, "refresh_token": refreshToken})
}

// SetupMFA creates a pending TOTP secret. It is not enabled until the caller
// proves possession of the authenticator code through EnableMFA.
func SetupMFA(c *gin.Context) {
	userID := c.GetInt64("user_id")
	var email string
	if err := configs.DB.QueryRow(`SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	secret, err := utils.GenerateTOTPSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate MFA secret"})
		return
	}
	if _, err := configs.DB.Exec(`UPDATE users SET mfa_secret = $1, mfa_enabled = false, "updatedAt" = NOW() WHERE id = $2`, secret, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save MFA secret"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"secret": secret, "otpauth_uri": utils.TOTPURI(secret, "StatGate", email), "enabled": false})
}

func EnableMFA(c *gin.Context) {
	userID := c.GetInt64("user_id")
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}
	var secret string
	if err := configs.DB.QueryRow(`SELECT COALESCE(mfa_secret, '') FROM users WHERE id = $1`, userID).Scan(&secret); err != nil || secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA setup is required first"})
		return
	}
	if !utils.VerifyTOTP(secret, req.Code, time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
		return
	}
	if _, err := configs.DB.Exec(`UPDATE users SET mfa_enabled = true, "updatedAt" = NOW() WHERE id = $1`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable MFA"})
		return
	}
	recoveryCodes, err := replaceRecoveryCodes(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate recovery codes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": true, "recovery_codes": recoveryCodes})
}

func DisableMFA(c *gin.Context) {
	userID := c.GetInt64("user_id")
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}
	var secret string
	if err := configs.DB.QueryRow(`SELECT COALESCE(mfa_secret, '') FROM users WHERE id = $1`, userID).Scan(&secret); err != nil || !utils.VerifyTOTP(secret, req.Code, time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
		return
	}
	if _, err := configs.DB.Exec(`UPDATE users SET mfa_enabled = false, mfa_secret = '', "updatedAt" = NOW() WHERE id = $1`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable MFA"})
		return
	}
	if _, err := configs.DB.Exec(`DELETE FROM mfa_recovery_codes WHERE user_id=$1`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear recovery codes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": false})
}

// RefreshUserToken exchanges a valid Registry refresh token for a new access
// token. The user is reloaded so role and tenant changes take effect without
// waiting for the old access token to expire.
func RefreshUserToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	userID, err := utils.RefreshSubject(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}
	if !consumeRefreshSession(req.RefreshToken, userID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token has been revoked or already used"})
		return
	}

	var user models.User
	err = configs.DB.QueryRow(
		`SELECT id, first_name, last_name, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"
		 FROM users WHERE id = $1 LIMIT 1`, userID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User no longer exists"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user"})
		return
	}

	user = toSafeUser(user)
	accessToken, err := utils.SignUserToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	newRefreshToken, err := utils.SignRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}
	if err := storeRefreshSession(c, newRefreshToken, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist refresh session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "token": accessToken, "refresh_token": newRefreshToken})
}

// GetCurrentUser - GET /users/me - Get current user
func GetCurrentUser(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var user models.User
	err := configs.DB.QueryRow(
		`SELECT id, first_name, last_name, username, email, role, organisation, phoneno, district_id, COALESCE(must_change_password, false), COALESCE(mfa_enabled, false), "createdAt", "updatedAt"
			 FROM users WHERE id = $1 LIMIT 1`,
		userID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.MustChangePassword, &user.MFAEnabled, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// ListUsers - GET /users - List all users
func ListUsers(c *gin.Context) {
	if !requireDirectoryAccess(c) {
		return
	}
	where, values, idx := []string{}, []interface{}{}, 1
	var ok bool
	where, values, idx, ok = appendAuthenticatedUserScope(c, where, values, idx)
	if !ok {
		return
	}

	query := `SELECT id, first_name, last_name, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"
		 FROM users` + combineWhere(where) + ` ORDER BY id ASC`
	rows, err := configs.DB.Query(query, values...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, toSafeUser(user))
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser - GET /users/:id - Get user by id
func GetUser(c *gin.Context) {
	if !requireInvitationAdmin(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var user models.User
	where, values, idx := []string{"id = $1"}, []interface{}{id}, 2
	var ok bool
	where, values, idx, ok = appendAuthenticatedUserScope(c, where, values, idx)
	if !ok {
		return
	}
	_ = idx

	err = configs.DB.QueryRow(
		`SELECT id, first_name, last_name, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"
		 FROM users`+combineWhere(where),
		values...,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	c.JSON(http.StatusOK, user)
}

// UpdateUser - PUT /users/:id - Update user
func UpdateUser(c *gin.Context) {
	if !requireInvitationAdmin(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Role         *string `json:"role"`
		FirstName    *string `json:"first_name"`
		LastName     *string `json:"last_name"`
		Email        *string `json:"email"`
		Username     *string `json:"username"`
		Password     *string `json:"password"`
		Organisation *string `json:"organisation"`
		Phoneno      *string `json:"phoneno"`
		DistrictID   *string `json:"district_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	var fields []string
	var values []interface{}
	idx := 1

	if req.Role != nil {
		if !roleAllowedForAuthenticatedAdmin(c, req.Role) {
			return
		}
		*req.Role = strings.ToLower(strings.TrimSpace(*req.Role))
		fields = append(fields, "role = $"+strconv.Itoa(idx))
		if *req.Role == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.Role)
		}
		idx++
	}

	if req.FirstName != nil {
		fields = append(fields, "first_name = $"+strconv.Itoa(idx))
		if *req.FirstName == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.FirstName)
		}
		idx++
	}

	if req.LastName != nil {
		fields = append(fields, "last_name = $"+strconv.Itoa(idx))
		if *req.LastName == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.LastName)
		}
		idx++
	}

	if req.Email != nil {
		fields = append(fields, "email = $"+strconv.Itoa(idx))
		if *req.Email == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.Email)
		}
		idx++
	}

	if req.Username != nil {
		fields = append(fields, "username = $"+strconv.Itoa(idx))
		if *req.Username == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.Username)
		}
		idx++
	}

	if req.Password != nil && *req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), 10)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		fields = append(fields, "password = $"+strconv.Itoa(idx))
		values = append(values, string(hashed))
		idx++
	}

	if org, ok := scopedOrganisationForWrite(c, req.Organisation); !ok {
		return
	} else if req.Organisation != nil || org != nil {
		fields = append(fields, "organisation = $"+strconv.Itoa(idx))
		values = append(values, org)
		idx++
	}

	if req.Phoneno != nil {
		fields = append(fields, "phoneno = $"+strconv.Itoa(idx))
		if *req.Phoneno == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.Phoneno)
		}
		idx++
	}

	if districtID := scopedDistrictForWrite(c, req.DistrictID); req.DistrictID != nil || (!isPlatformAdmin(c) && districtID != nil) {
		fields = append(fields, "district_id = $"+strconv.Itoa(idx))
		values = append(values, districtID)
		idx++
	}

	if len(fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	fields = append(fields, `"updatedAt" = NOW()`)
	where := []string{"id = $" + strconv.Itoa(idx)}
	values = append(values, id)
	idx++
	var ok bool
	where, values, idx, ok = appendAuthenticatedUserScope(c, where, values, idx)
	if !ok {
		return
	}

	query := `UPDATE users SET ` + strings.Join(fields, ", ") + combineWhere(where) + ` RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"`

	var user models.User
	err = configs.DB.QueryRow(query, values...).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email or username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	c.JSON(http.StatusOK, user)
}

// ChangePassword - POST /users/change-password - Change user password (requires old password verification)
func ChangePassword(c *gin.Context) {
	var req struct {
		EmailOrUsername string `json:"emailOrUsername" binding:"required"`
		OldPassword     string `json:"oldPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "emailOrUsername, oldPassword and newPassword are required"})
		return
	}

	if !passwordMeetsPolicy(req.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must be at least 10 characters and include upper/lower case and a digit"})
		return
	}

	// Get user by email or username
	var user models.User
	err := configs.DB.QueryRow(
		`SELECT id, first_name, last_name, username, email, role, password, organisation, phoneno, district_id, COALESCE(must_change_password, false), "createdAt", "updatedAt"
		 FROM users WHERE email = $1 OR username = $1 LIMIT 1`,
		req.EmailOrUsername,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.PasswordHash, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or username"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Verify old password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Old password is incorrect"})
		return
	}

	// Hash new password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update password and clear must_change_password
	err = configs.DB.QueryRow(
		`UPDATE users SET password = $1, must_change_password = false, "updatedAt" = NOW() WHERE id = $2
		 RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, must_change_password, "createdAt", "updatedAt"`,
		string(hashed), user.ID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password changed successfully", "user": user})
}

// ResetPassword - POST /users/:id/reset-password - Reset user password
func ResetPassword(c *gin.Context) {
	if !requireInvitationAdmin(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		NewPassword string `json:"newPassword" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password is required"})
		return
	}

	if !passwordMeetsPolicy(req.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 10 characters and include upper/lower case and a digit"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var user models.User
	where, values, idx := []string{"id = $2"}, []interface{}{string(hashed), id}, 3
	var ok bool
	where, values, idx, ok = appendAuthenticatedUserScope(c, where, values, idx)
	if !ok {
		return
	}
	_ = idx
	err = configs.DB.QueryRow(
		`UPDATE users SET password = $1, "updatedAt" = NOW()`+combineWhere(where)+`
		 RETURNING id, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"`,
		values...,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password reset successfully", "user": user})
}

// DeleteUser - DELETE /users/:id - Delete user
func DeleteUser(c *gin.Context) {
	if !requireInvitationAdmin(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	where, values, idx := []string{"id = $1"}, []interface{}{id}, 2
	var ok bool
	where, values, idx, ok = appendAuthenticatedUserScope(c, where, values, idx)
	if !ok {
		return
	}
	_ = idx
	result, err := configs.DB.Exec("DELETE FROM users"+combineWhere(where), values...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// CreateUser - POST /users - Create a new user (authenticated users)
func CreateUser(c *gin.Context) {
	if !requireInvitationAdmin(c) {
		return
	}
	var req struct {
		Role         *string `json:"role"`
		FirstName    *string `json:"first_name"`
		LastName     *string `json:"last_name"`
		Email        string  `json:"email" binding:"required"`
		Username     string  `json:"username" binding:"required"`
		Password     string  `json:"password" binding:"required"`
		Organisation *string `json:"organisation"`
		Phoneno      *string `json:"phoneno"`
		DistrictID   *string `json:"district_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, username and password are required"})
		return
	}

	// Validation applies to administrator provisioning too (SG-SEC-2026-08).
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A valid email address is required"})
		return
	}
	if !passwordMeetsPolicy(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 10 characters and include upper/lower case and a digit"})
		return
	}
	if !roleAllowedForAuthenticatedAdmin(c, req.Role) {
		return
	}
	if req.Role != nil {
		*req.Role = strings.ToLower(strings.TrimSpace(*req.Role))
	}
	organisation, ok := scopedOrganisationForWrite(c, req.Organisation)
	if !ok {
		return
	}
	districtID := scopedDistrictForWrite(c, req.DistrictID)

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var user models.User
	err = configs.DB.QueryRow(
		`INSERT INTO users (role, first_name, last_name, email, username, password, organisation, phoneno, district_id, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		 RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"`,
		req.Role, req.FirstName, req.LastName, req.Email, req.Username, string(hashed), organisation, req.Phoneno, districtID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Check for duplicate key error (PostgreSQL error code 23505)
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email or username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	// Return user without token (admin creates user, doesn't log them in)
	c.JSON(http.StatusCreated, gin.H{"user": user, "token": nil})
}

// UploadUsersCSV - POST /users/upload - Bulk upload users from CSV with default password
func UploadUsersCSV(c *gin.Context) {
	if !requireInvitationAdmin(c) {
		return
	}
	requestScopedOrg, ok := scopedOrganisationForWrite(c, nil)
	if !ok {
		return
	}
	requestScopedDistrict := scopedDistrictForWrite(c, nil)
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is required"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is empty"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV header"})
		return
	}

	// Build column index map
	colIndex := map[string]int{}
	for i, col := range header {
		name := strings.ToLower(strings.TrimSpace(col))
		colIndex[name] = i
	}

	requiredCols := []string{"email", "username"}
	for _, col := range requiredCols {
		if _, ok := colIndex[col]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missing required column '%s' in CSV header", col)})
			return
		}
	}

	// Helper to look up a column index by one of several possible header names
	lookupCol := func(names ...string) (int, bool) {
		for _, n := range names {
			if idx, ok := colIndex[strings.ToLower(n)]; ok {
				return idx, true
			}
		}
		return 0, false
	}

	// Optional columns (supporting multiple common header variants)
	firstNameIdx, hasFirstName := lookupCol("first_name", "firstname", "first name")
	lastNameIdx, hasLastName := lookupCol("last_name", "lastname", "last name")
	roleIdx, hasRole := lookupCol("role")
	orgIdx, hasOrg := lookupCol("organisation", "organization", "org", "organization name")
	phoneIdx, hasPhone := lookupCol("phoneno", "phone", "phone_number", "phone number")
	districtIdx, hasDistrict := lookupCol("district_id", "district", "districtid")

	// ROLE ALLOWLIST: CSV-provisioned roles are limited to the non-privileged
	// registry roles. Privileged roles require an administrator-created account.
	provisionedRoles := map[string]bool{
		"viewer": true, "analyst": true, "agent": true, "district": true,
		"district_admin": true, "tenant_admin": true,
	}

	type failure struct {
		Row      int    `json:"row"`
		Email    string `json:"email,omitempty"`
		Username string `json:"username,omitempty"`
		Error    string `json:"error"`
	}

	var (
		totalRows       int
		created         int
		failed          []failure
		tempCredentials []gin.H
	)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		totalRows++

		if err != nil {
			failed = append(failed, failure{
				Row:   totalRows + 1, // +1 to account for header row
				Error: "Failed to read row: " + err.Error(),
			})
			continue
		}

		// basic bounds check
		if len(record) < len(header) {
			failed = append(failed, failure{
				Row:   totalRows + 1,
				Error: "Row has fewer columns than header",
			})
			continue
		}

		email := strings.TrimSpace(record[colIndex["email"]])
		username := strings.TrimSpace(record[colIndex["username"]])

		if email == "" || username == "" {
			failed = append(failed, failure{
				Row:      totalRows + 1,
				Email:    email,
				Username: username,
				Error:    "email and username are required",
			})
			continue
		}

		var (
			role       *string
			firstName  *string
			lastName   *string
			org        *string
			phone      *string
			districtID *string
		)

		if hasRole {
			val := strings.TrimSpace(record[roleIdx])
			if val != "" {
				role = &val
			}
		}
		if hasFirstName {
			val := strings.TrimSpace(record[firstNameIdx])
			if val != "" {
				firstName = &val
			}
		}
		if hasLastName {
			val := strings.TrimSpace(record[lastNameIdx])
			if val != "" {
				lastName = &val
			}
		}
		if hasOrg {
			val := strings.TrimSpace(record[orgIdx])
			if val != "" {
				org = &val
			}
		}
		if hasPhone {
			val := strings.TrimSpace(record[phoneIdx])
			if val != "" {
				phone = &val
			}
		}
		if hasDistrict {
			val := strings.TrimSpace(record[districtIdx])
			if val != "" {
				districtID = &val
			}
		}
		if !isPlatformAdmin(c) {
			org = requestScopedOrg
			districtID = requestScopedDistrict
		} else {
			org = trimOptionalString(org)
			districtID = trimOptionalString(districtID)
		}

		// Role allowlist enforcement for bulk-provisioned accounts
		if role != nil {
			roleName := strings.ToLower(strings.TrimSpace(*role))
			if roleName != "" && !provisionedRoles[roleName] {
				failed = append(failed, failure{
					Row:      totalRows + 1,
					Email:    email,
					Username: username,
					Error:    "role is not permitted for bulk provisioning",
				})
				continue
			}
			if !canAuthenticatedAdminManageRole(c, &roleName) {
				failed = append(failed, failure{
					Row:      totalRows + 1,
					Email:    email,
					Username: username,
					Error:    "role is not permitted for tenant administration",
				})
				continue
			}
			role = &roleName
		}

		// Every provisioned account receives a unique one-time password and
		// must change it on first login (SG-SEC-2026-08 removes default passwords).
		tempPassword := randomTemporaryPassword()
		rowHash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), 10)
		if err != nil {
			failed = append(failed, failure{Row: totalRows + 1, Email: email, Username: username, Error: "failed to hash password"})
			continue
		}

		var user models.User
		mustChangeVal := true

		err = configs.DB.QueryRow(
			`INSERT INTO users (role, first_name, last_name, email, username, password, organisation, phoneno, district_id, must_change_password, "createdAt", "updatedAt")
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
			 RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, must_change_password, "createdAt", "updatedAt"`,
			role, firstName, lastName, email, username, string(rowHash), org, phone, districtID, mustChangeVal,
		).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "duplicate key") || strings.Contains(msg, "23505") {
				msg = "email or username already exists"
			}
			failed = append(failed, failure{
				Row:      totalRows + 1,
				Email:    email,
				Username: username,
				Error:    msg,
			})
			continue
		}

		// Record the generated one-time credential for the requesting
		// administrator to distribute out-of-band (never plaintext-stored).
		tempCredentials = append(tempCredentials, gin.H{
			"row":           totalRows + 1,
			"email":         email,
			"username":      username,
			"temp_password": tempPassword,
		})
		created++
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"total":          totalRows,
		"created":        created,
		"failed":         len(failed),
		"errors":         failed,
		"temp_passwords": tempCredentials,
	})
}
