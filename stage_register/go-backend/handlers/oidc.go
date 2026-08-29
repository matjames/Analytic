package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"

	"github.com/gin-gonic/gin"
)

const oidcStateCookie = "statgate_oidc_state"

type oidcConfig struct {
	Issuer        string
	ClientID      string
	ClientSecret  string
	RedirectURL   string
	Scopes        string
	DefaultRole   string
	DefaultTenant string
	AllowedRoles  map[string]bool
}

type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
}

type oidcStatePayload struct {
	State string `json:"state"`
	Nonce string `json:"nonce"`
	Iat   int64  `json:"iat"`
}

type oidcUserInfo struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	PreferredUsername string `json:"preferred_username"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	Name              string `json:"name"`
	Role              string `json:"role"`
	TenantID          string `json:"tenant_id"`
	OrgID             string `json:"org_id"`
	OrganizationID    string `json:"organization_id"`
	DistrictID        string `json:"district_id"`
}

var registryManageableRoles = map[string]bool{
	"viewer": true, "analyst": true, "editor": true, "operator": true,
	"manager": true, "agent": true, "district": true, "district_admin": true,
	"tenant_admin": true, "governance_officer": true, "property_officer": true,
	"admin": true, "superadmin": true, "platform_admin": true,
}

func loadOIDCConfig() (oidcConfig, bool) {
	cfg := oidcConfig{
		Issuer:        strings.TrimRight(strings.TrimSpace(os.Getenv("OIDC_ISSUER_URL")), "/"),
		ClientID:      strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID")),
		ClientSecret:  strings.TrimSpace(os.Getenv("OIDC_CLIENT_SECRET")),
		RedirectURL:   strings.TrimSpace(os.Getenv("OIDC_REDIRECT_URL")),
		Scopes:        strings.TrimSpace(os.Getenv("OIDC_SCOPES")),
		DefaultRole:   strings.ToLower(strings.TrimSpace(os.Getenv("OIDC_DEFAULT_ROLE"))),
		DefaultTenant: strings.TrimSpace(os.Getenv("OIDC_DEFAULT_TENANT")),
		AllowedRoles:  map[string]bool{},
	}
	if cfg.Scopes == "" {
		cfg.Scopes = "openid email profile"
	}
	if cfg.DefaultRole == "" {
		cfg.DefaultRole = "viewer"
	}
	if !registryManageableRoles[cfg.DefaultRole] {
		cfg.DefaultRole = "viewer"
	}
	for _, role := range strings.Split(os.Getenv("OIDC_ALLOWED_ROLES"), ",") {
		role = strings.ToLower(strings.TrimSpace(role))
		if registryManageableRoles[role] {
			cfg.AllowedRoles[role] = true
		}
	}
	if len(cfg.AllowedRoles) == 0 {
		cfg.AllowedRoles[cfg.DefaultRole] = true
	}
	return cfg, cfg.Issuer != "" && cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.RedirectURL != ""
}

func oidcHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func discoverOIDCProvider(cfg oidcConfig) (oidcDiscovery, error) {
	endpoint := cfg.Issuer + "/.well-known/openid-configuration"
	resp, err := oidcHTTPClient().Get(endpoint)
	if err != nil {
		return oidcDiscovery{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oidcDiscovery{}, fmt.Errorf("discovery returned %d", resp.StatusCode)
	}
	var discovery oidcDiscovery
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&discovery); err != nil {
		return oidcDiscovery{}, err
	}
	if discovery.AuthorizationEndpoint == "" || discovery.TokenEndpoint == "" || discovery.UserInfoEndpoint == "" {
		return oidcDiscovery{}, fmt.Errorf("discovery missing required endpoints")
	}
	return discovery, nil
}

func randomURLToken(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func oidcStateSecret() []byte {
	secret := strings.TrimSpace(os.Getenv("OIDC_STATE_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("STATGATE_REGISTRY_JWT_SECRET"))
	}
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	}
	return []byte(secret)
}

func encodeOIDCState(payload oidcStatePayload) (string, error) {
	secret := oidcStateSecret()
	if len(secret) == 0 {
		return "", fmt.Errorf("OIDC_STATE_SECRET or STATGATE_REGISTRY_JWT_SECRET is required")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(body))
	return body + "." + hex.EncodeToString(mac.Sum(nil)), nil
}

func decodeOIDCState(value string) (oidcStatePayload, error) {
	secret := oidcStateSecret()
	parts := strings.Split(value, ".")
	if len(secret) == 0 || len(parts) != 2 {
		return oidcStatePayload{}, fmt.Errorf("invalid state")
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(parts[0]))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return oidcStatePayload{}, fmt.Errorf("invalid state signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return oidcStatePayload{}, err
	}
	var payload oidcStatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return oidcStatePayload{}, err
	}
	issuedAt := time.Unix(payload.Iat, 0)
	if payload.State == "" || time.Since(issuedAt) > 10*time.Minute || issuedAt.After(time.Now().Add(1*time.Minute)) {
		return oidcStatePayload{}, fmt.Errorf("expired state")
	}
	return payload, nil
}

// OIDCLogin starts an optional OpenID Connect authorization-code login.
func OIDCLogin(c *gin.Context) {
	cfg, ok := loadOIDCConfig()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "oidc_not_configured"})
		return
	}
	discovery, err := discoverOIDCProvider(cfg)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "oidc_discovery_failed"})
		return
	}
	state, err := randomURLToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_start_oidc"})
		return
	}
	nonce, err := randomURLToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_start_oidc"})
		return
	}
	signedState, err := encodeOIDCState(oidcStatePayload{State: state, Nonce: nonce, Iat: time.Now().Unix()})
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "oidc_state_secret_missing"})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oidcStateCookie,
		Value:    signedState,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isProduction(),
	})
	authURL, err := url.Parse(discovery.AuthorizationEndpoint)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "oidc_authorization_endpoint_invalid"})
		return
	}
	q := authURL.Query()
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", cfg.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", cfg.Scopes)
	q.Set("state", state)
	q.Set("nonce", nonce)
	authURL.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, authURL.String())
}

// OIDCCallback completes OIDC login and issues normal Registry tokens.
func OIDCCallback(c *gin.Context) {
	cfg, ok := loadOIDCConfig()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "oidc_not_configured"})
		return
	}
	cookie, err := c.Request.Cookie(oidcStateCookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oidc_state_missing"})
		return
	}
	statePayload, err := decodeOIDCState(cookie.Value)
	if err != nil || statePayload.State != c.Query("state") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oidc_state_invalid"})
		return
	}
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oidc_code_required"})
		return
	}
	discovery, err := discoverOIDCProvider(cfg)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "oidc_discovery_failed"})
		return
	}
	accessToken, err := exchangeOIDCCode(cfg, discovery, code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "oidc_token_exchange_failed"})
		return
	}
	info, err := fetchOIDCUserInfo(discovery, accessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "oidc_userinfo_failed"})
		return
	}
	user, err := upsertOIDCUser(cfg, info)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "oidc_user_not_allowed"})
		return
	}
	token, err := utils.SignUserToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_issue_token"})
		return
	}
	refreshToken, err := utils.SignRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_issue_refresh_token"})
		return
	}
	if err := storeRefreshSession(c, refreshToken, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_persist_session"})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: oidcStateCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isProduction()})
	c.JSON(http.StatusOK, gin.H{"user": toSafeUser(user), "token": token, "refresh_token": refreshToken})
}

func exchangeOIDCCode(cfg oidcConfig, discovery oidcDiscovery, code string) (string, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("redirect_uri", cfg.RedirectURL)
	values.Set("client_id", cfg.ClientID)
	values.Set("client_secret", cfg.ClientSecret)
	req, err := http.NewRequest(http.MethodPost, discovery.TokenEndpoint, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := oidcHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}
	var body struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("missing access token")
	}
	return body.AccessToken, nil
}

func fetchOIDCUserInfo(discovery oidcDiscovery, accessToken string) (oidcUserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, discovery.UserInfoEndpoint, nil)
	if err != nil {
		return oidcUserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := oidcHTTPClient().Do(req)
	if err != nil {
		return oidcUserInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oidcUserInfo{}, fmt.Errorf("userinfo returned %d", resp.StatusCode)
	}
	var info oidcUserInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&info); err != nil {
		return oidcUserInfo{}, err
	}
	if strings.TrimSpace(info.Email) == "" {
		return oidcUserInfo{}, fmt.Errorf("userinfo missing email")
	}
	return info, nil
}

func upsertOIDCUser(cfg oidcConfig, info oidcUserInfo) (models.User, error) {
	email := strings.ToLower(strings.TrimSpace(info.Email))
	username := strings.TrimSpace(info.PreferredUsername)
	if username == "" {
		username = strings.Split(email, "@")[0]
	}
	firstName, lastName := oidcNames(info)
	role := mappedOIDCRole(cfg, info.Role)
	organisation := strings.TrimSpace(info.TenantID)
	if organisation == "" {
		organisation = strings.TrimSpace(info.OrgID)
	}
	if organisation == "" {
		organisation = strings.TrimSpace(info.OrganizationID)
	}
	if organisation == "" {
		organisation = cfg.DefaultTenant
	}
	if organisation == "" {
		return models.User{}, fmt.Errorf("oidc tenant missing")
	}
	districtID := strings.TrimSpace(info.DistrictID)
	passwordHash := utils.HashPassword(randomTemporaryPassword())

	var user models.User
	err := configs.DB.QueryRow(
		`INSERT INTO users (role, first_name, last_name, email, username, password, organisation, district_id, email_verified, must_change_password, "createdAt", "updatedAt")
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,true,false,NOW(),NOW())
		 ON CONFLICT (email) DO UPDATE SET
		   first_name = COALESCE(EXCLUDED.first_name, users.first_name),
		   last_name = COALESCE(EXCLUDED.last_name, users.last_name),
		   organisation = COALESCE(NULLIF(EXCLUDED.organisation, ''), users.organisation),
		   district_id = COALESCE(NULLIF(EXCLUDED.district_id, ''), users.district_id),
		   email_verified = true,
		   "updatedAt" = NOW()
		 RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, email_verified, "createdAt", "updatedAt"`,
		role, emptyToNil(firstName), emptyToNil(lastName), email, username, passwordHash, organisation, districtID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return models.User{}, fmt.Errorf("oidc user not found")
	}
	return user, err
}

func oidcNames(info oidcUserInfo) (string, string) {
	firstName := strings.TrimSpace(info.GivenName)
	lastName := strings.TrimSpace(info.FamilyName)
	if firstName == "" && lastName == "" && strings.TrimSpace(info.Name) != "" {
		parts := strings.Fields(info.Name)
		if len(parts) > 0 {
			firstName = parts[0]
		}
		if len(parts) > 1 {
			lastName = strings.Join(parts[1:], " ")
		}
	}
	return firstName, lastName
}

func mappedOIDCRole(cfg oidcConfig, requested string) string {
	role := strings.ToLower(strings.TrimSpace(requested))
	if role != "" && cfg.AllowedRoles[role] {
		return role
	}
	return cfg.DefaultRole
}

func emptyToNil(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}
