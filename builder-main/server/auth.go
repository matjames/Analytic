// Package main: Keycloak JWT Authentication Middleware
// Validates Bearer tokens against Keycloak JWKS endpoint
// Supports two modes: off (development), on (production)
package main

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Auth configuration from environment variables
var (
	keycloakURL         string // KEYCLOAK_URL: e.g., https://auth.statgate.ug
	keycloakRealm       string // KEYCLOAK_REALM: e.g., StatGate
	keycloakClientID    string // KEYCLOAK_CLIENT_ID: e.g., statgate-report-browser
	keycloakRolesClient string // KEYCLOAK_ROLES_CLIENT_ID: app client holding roles/groups (e.g., dwhlanding)
	keycloakAdminID     string // KEYCLOAK_ADMIN_ID: service account client for Admin API
	keycloakAdminSecret string // KEYCLOAK_ADMIN_SECRET: client secret (server-side only)
	KeycloakAdminRole   string // KEYCLOAK_ADMIN_ROLE: role required to access user management
	authMode            string // AUTH_MODE: "off" or "on"
)

// JWKS cache - stores Keycloak public keys for JWT verification
var (
	jwksCache     *JWKS
	jwksCacheMux  sync.RWMutex
	jwksCacheTime time.Time
	jwksCacheTTL  = 1 * time.Hour // Refresh keys hourly
)

// JWKS represents a JSON Web Key Set from Keycloak
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a single JSON Web Key (RSA public key)
type JWK struct {
	Kid string `json:"kid"` // Key ID - matches "kid" header in JWT
	Kty string `json:"kty"` // Key Type - should be "RSA"
	Alg string `json:"alg"` // Algorithm - e.g., "RS256"
	Use string `json:"use"` // Usage - "sig" for signature verification
	N   string `json:"n"`   // RSA modulus (base64url encoded)
	E   string `json:"e"`   // RSA exponent (base64url encoded)
}

// UserClaims holds validated user information extracted from JWT
type UserClaims struct {
	Subject       string              // sub: unique user ID
	Username      string              // preferred_username: display name
	Email         string              // email address
	EmailVerified bool                // email_verified flag
	Name          string              // full name
	RealmRoles    []string            // roles from realm_access
	ClientRoles   map[string][]string // roles from resource_access (per client)
	ExpiresAt     time.Time           // token expiration time
}

// Context key for storing user claims in request context
type authContextKey string

const userClaimsContextKey authContextKey = "userClaims"

// InitAuth initializes authentication configuration from environment variables.
// Called once at server startup.
//
// Environment variables:
//   - KEYCLOAK_URL: Base URL of Keycloak server (e.g., https://auth.statgate.ug)
//   - KEYCLOAK_REALM: Keycloak realm name (e.g., StatGate)
//   - KEYCLOAK_CLIENT_ID: Client ID for audience validation (e.g., statgate-report-browser)
//   - AUTH_MODE: Authentication mode - "off" or "on" (default: "off")
func InitAuth() {
	keycloakURL = strings.TrimSuffix(os.Getenv("KEYCLOAK_URL"), "/")
	keycloakRealm = os.Getenv("KEYCLOAK_REALM")
	keycloakClientID = os.Getenv("KEYCLOAK_CLIENT_ID")
	keycloakRolesClient = os.Getenv("KEYCLOAK_ROLES_CLIENT_ID")
	keycloakAdminID = strings.TrimSpace(os.Getenv("KEYCLOAK_ADMIN_ID"))
	keycloakAdminSecret = os.Getenv("KEYCLOAK_ADMIN_SECRET")
	KeycloakAdminRole = os.Getenv("KEYCLOAK_ADMIN_ROLE")
	if KeycloakAdminRole == "" {
		KeycloakAdminRole = "admin_manage"
	}
	authMode = strings.ToLower(os.Getenv("AUTH_MODE"))

	// Default to "off" if not specified (no auth required for local development)
	if authMode == "" {
		authMode = "off"
	}

	// Validate auth mode
	if authMode != "off" && authMode != "on" {
		logWarn("Invalid AUTH_MODE=%q, must be 'off' or 'on'. Defaulting to 'off'", authMode)
		authMode = "off"
	}

	// Check if Keycloak is configured when auth is enabled
	if authMode != "off" && (keycloakURL == "" || keycloakRealm == "") {
		logWarn("AUTH_MODE=%s but KEYCLOAK_URL or KEYCLOAK_REALM not set. Disabling auth.", authMode)
		authMode = "off"
	}

	// Log configuration
	if authMode == "off" {
		logInfo("Auth: DISABLED (AUTH_MODE=off)")
	} else {
		logInfo("Auth: mode=%s, keycloak=%s, realm=%s, client=%s",
			authMode, keycloakURL, keycloakRealm, keycloakClientID)

		// Pre-fetch JWKS to catch configuration errors early
		if _, err := refreshJWKS(); err != nil {
			logError("Auth: Failed to fetch JWKS from Keycloak: %v", err)
			logWarn("Auth: Will retry on first request. Verify KEYCLOAK_URL and KEYCLOAK_REALM are correct.")
		} else {
			logInfo("Auth: Successfully loaded JWKS from Keycloak")
		}
	}
}

// AuthMiddleware validates JWT tokens and extracts user claims.
// Wraps HTTP handlers to require authentication.
//
// Behavior depends on AUTH_MODE:
//   - "off": Skip validation entirely, pass through all requests
//   - "on": Reject requests with invalid/missing tokens (401)
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if disabled
		if authMode == "off" {
			next(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			handleAuthFailure(w, r, next, "missing Authorization header", nil)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			handleAuthFailure(w, r, next, "invalid Authorization header format (expected 'Bearer <token>')", nil)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			handleAuthFailure(w, r, next, "empty token", nil)
			return
		}

		// Validate token and extract claims
		claims, err := validateJWT(tokenString)
		if err != nil {
			handleAuthFailure(w, r, next, "token validation failed", err)
			return
		}

		// Token is valid - add claims to request context
		ctx := context.WithValue(r.Context(), userClaimsContextKey, claims)

		logDebug("Auth OK: user=%s path=%s", claims.Username, r.URL.Path)

		next(w, r.WithContext(ctx))
	}
}

// RequireRole wraps a handler to require a specific role.
// The role is checked in both client roles (resource_access.dwhlanding.roles)
// and realm roles (realm_access.roles).
//
// Usage:
//
//	http.HandleFunc("/api/admin", RequireRole("admin", AdminHandler))
func RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Skip role check if auth is off
		if authMode == "off" {
			next(w, r)
			return
		}

		claims := GetUserClaims(r)
		if claims == nil {
			// This shouldn't happen if AuthMiddleware is working
			writeJSONError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		clientID := keycloakClientID
		if clientID != "" && hasClientRole(claims, clientID, role) {
			next(w, r)
			return
		}
		if clientID != "dwhlanding" && hasClientRole(claims, "dwhlanding", role) {
			next(w, r)
			return
		}
		for _, roles := range claims.ClientRoles {
			for _, clientRole := range roles {
				if clientRole == role {
					next(w, r)
					return
				}
			}
		}

		// Also check realm roles (realm_access.roles)
		if hasRealmRole(claims, role) {
			next(w, r)
			return
		}

		// User doesn't have the required role
		logWarn("Auth: access denied user=%s role=%s (has client=%v realm=%v) path=%s",
			claims.Username, role, claims.ClientRoles[clientID], claims.RealmRoles, r.URL.Path)

		writeJSONError(w, http.StatusForbidden, "insufficient permissions: requires '"+role+"' role")
	})
}

// RequireAdminPermission wraps a handler to require admin rights from role-permissions.
func RequireAdminPermission(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if authMode != "on" {
			next(w, r)
			return
		}
		perms := GetEffectivePermissionsFromRequest(r)
		if perms != nil && perms.Admin {
			next(w, r)
			return
		}
		writeJSONError(w, http.StatusForbidden, "insufficient permissions: requires admin rights")
	})
}

// HasKeycloakAdminRole returns true when the user has the configured Keycloak admin role.
func HasKeycloakAdminRole(r *http.Request) bool {
	claims := GetUserClaims(r)
	if claims == nil || KeycloakAdminRole == "" {
		return false
	}
	clientID := keycloakClientID
	if clientID == "" {
		clientID = "dwhlanding"
	}
	if hasClientRole(claims, clientID, KeycloakAdminRole) {
		return true
	}
	if hasRealmRole(claims, KeycloakAdminRole) {
		return true
	}
	for _, roles := range claims.ClientRoles {
		for _, role := range roles {
			if strings.EqualFold(role, KeycloakAdminRole) {
				return true
			}
		}
	}
	return false
}

// RequireKeycloakAdminRole wraps a handler to require the configured Keycloak admin role.
func RequireKeycloakAdminRole(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if authMode != "on" {
			next(w, r)
			return
		}
		if HasKeycloakAdminRole(r) {
			next(w, r)
			return
		}
		writeJSONError(w, http.StatusForbidden, "insufficient permissions: requires Keycloak role "+KeycloakAdminRole)
	})
}

// GetUserClaims retrieves the validated user claims from request context.
// Returns nil if no claims are present (unauthenticated request or auth disabled).
func GetUserClaims(r *http.Request) *UserClaims {
	claims, ok := r.Context().Value(userClaimsContextKey).(*UserClaims)
	if !ok {
		return nil
	}
	return claims
}

// hasClientRole checks if user has a specific role for a client
func hasClientRole(claims *UserClaims, client, role string) bool {
	if claims == nil || claims.ClientRoles == nil {
		return false
	}
	roles, ok := claims.ClientRoles[client]
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// hasRealmRole checks if user has a specific realm role
func hasRealmRole(claims *UserClaims, role string) bool {
	if claims == nil {
		return false
	}
	for _, r := range claims.RealmRoles {
		if r == role {
			return true
		}
	}
	return false
}

// handleAuthFailure handles authentication failures when AUTH_MODE is "on"
func handleAuthFailure(w http.ResponseWriter, r *http.Request, _ http.HandlerFunc, message string, err error) {
	// Log the failure
	if err != nil {
		logWarn("Auth FAIL: %s - %v (path=%s)", message, err, r.URL.Path)
	} else {
		logWarn("Auth FAIL: %s (path=%s)", message, r.URL.Path)
	}

	// Reject the request
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s"`, keycloakRealm))
	writeJSONError(w, http.StatusUnauthorized, message)
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// validateJWT validates a JWT token and extracts claims
func validateJWT(tokenString string) (*UserClaims, error) {
	// Parse token without verification first to get the key ID
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("malformed token: %w", err)
	}

	// Get key ID from header
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		return nil, fmt.Errorf("missing 'kid' in token header")
	}

	// Get the public key for this key ID
	publicKey, err := getPublicKey(kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Parse and validate token with the public key
	token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method is RSA
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid signature or expired: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	// Extract claims
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	// Verify audience/authorized party if client ID is configured
	if keycloakClientID != "" {
		azp, _ := mapClaims["azp"].(string)
		if azp != "" && azp != keycloakClientID {
			return nil, fmt.Errorf("invalid authorized party: got %s, expected %s", azp, keycloakClientID)
		}
	}

	// Build UserClaims from token
	claims := &UserClaims{
		Subject:       getClaimString(mapClaims, "sub"),
		Username:      getClaimString(mapClaims, "preferred_username"),
		Email:         getClaimString(mapClaims, "email"),
		EmailVerified: getClaimBool(mapClaims, "email_verified"),
		Name:          getClaimString(mapClaims, "name"),
		ClientRoles:   make(map[string][]string),
	}

	// Extract expiration time
	if exp, ok := mapClaims["exp"].(float64); ok {
		claims.ExpiresAt = time.Unix(int64(exp), 0)
	}

	// Extract realm roles (realm_access.roles)
	if realmAccess, ok := mapClaims["realm_access"].(map[string]interface{}); ok {
		if roles, ok := realmAccess["roles"].([]interface{}); ok {
			for _, role := range roles {
				if roleStr, ok := role.(string); ok {
					claims.RealmRoles = append(claims.RealmRoles, roleStr)
				}
			}
		}
	}

	// Extract client roles (resource_access.<client>.roles)
	if resourceAccess, ok := mapClaims["resource_access"].(map[string]interface{}); ok {
		for clientName, clientAccess := range resourceAccess {
			if accessMap, ok := clientAccess.(map[string]interface{}); ok {
				if roles, ok := accessMap["roles"].([]interface{}); ok {
					for _, role := range roles {
						if roleStr, ok := role.(string); ok {
							claims.ClientRoles[clientName] = append(claims.ClientRoles[clientName], roleStr)
						}
					}
				}
			}
		}
	}

	return claims, nil
}

// getPublicKey retrieves the RSA public key for a given key ID from JWKS
func getPublicKey(kid string) (*rsa.PublicKey, error) {
	jwks, err := getJWKS()
	if err != nil {
		return nil, err
	}

	// Look for the key in cache
	for _, key := range jwks.Keys {
		if key.Kid == kid && key.Kty == "RSA" {
			return jwkToRSAPublicKey(key)
		}
	}

	// Key not found - maybe Keycloak rotated keys, refresh cache
	logDebug("Auth: key %s not in cache, refreshing JWKS", kid)
	jwks, err = refreshJWKS()
	if err != nil {
		return nil, fmt.Errorf("JWKS refresh failed: %w", err)
	}

	// Try again with fresh keys
	for _, key := range jwks.Keys {
		if key.Kid == kid && key.Kty == "RSA" {
			return jwkToRSAPublicKey(key)
		}
	}

	return nil, fmt.Errorf("key ID '%s' not found in Keycloak JWKS", kid)
}

// getJWKS returns cached JWKS or fetches fresh from Keycloak
func getJWKS() (*JWKS, error) {
	jwksCacheMux.RLock()
	if jwksCache != nil && time.Since(jwksCacheTime) < jwksCacheTTL {
		defer jwksCacheMux.RUnlock()
		return jwksCache, nil
	}
	jwksCacheMux.RUnlock()

	return refreshJWKS()
}

// refreshJWKS fetches fresh JWKS from Keycloak's certs endpoint
func refreshJWKS() (*JWKS, error) {
	jwksCacheMux.Lock()
	defer jwksCacheMux.Unlock()

	// Double-check after acquiring write lock
	if jwksCache != nil && time.Since(jwksCacheTime) < jwksCacheTTL {
		return jwksCache, nil
	}

	// Build JWKS URL
	// Format: https://auth.statgate.ug/realms/StatGate/protocol/openid-connect/certs
	url := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs",
		keycloakURL, keycloakRealm)

	logDebug("Auth: fetching JWKS from %s", url)

	// Fetch with timeout
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Keycloak: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Keycloak returned status %d", resp.StatusCode)
	}

	// Parse response
	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	if len(jwks.Keys) == 0 {
		return nil, fmt.Errorf("JWKS contains no keys")
	}

	// Update cache
	jwksCache = &jwks
	jwksCacheTime = time.Now()

	logDebug("Auth: JWKS refreshed, %d keys loaded", len(jwks.Keys))

	return &jwks, nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode modulus (n) from base64url
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode exponent (e) from base64url
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert exponent bytes to int (typically 65537 = 0x010001)
	var e int
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

// Helper functions for extracting typed values from JWT claims

func getClaimString(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key].(string); ok {
		return v
	}
	return ""
}

func getClaimBool(claims jwt.MapClaims, key string) bool {
	if v, ok := claims[key].(bool); ok {
		return v
	}
	return false
}

// AuthConfigResponse is the response body for GET /api/auth/config
type AuthConfigResponse struct {
	Mode         string                `json:"mode"`
	RequireLogin bool                  `json:"requireLogin"`
	CanPublish   bool                  `json:"canPublish"`
	Keycloak     *KeycloakConfigPublic `json:"keycloak,omitempty"`
}

type FrontendConfigResponse struct {
	ClientAPIUrl string `json:"clientAPIUrl"`
	AskEnabled   bool   `json:"askEnabled"`
}

// KeycloakConfigPublic contains the public Keycloak config for frontend initialization
type KeycloakConfigPublic struct {
	URL           string `json:"url"`
	Realm         string `json:"realm"`
	ClientID      string `json:"clientId"`
	RolesClientID string `json:"rolesClientId,omitempty"`
}

// GetAuthConfigHandler returns the authentication configuration for frontend initialization.
// This endpoint is PUBLIC (no auth required) so the frontend can determine whether to
// initialize Keycloak before any authentication is set up.
//
// Response:
//
//	{
//	  "mode": "on",            // "off" or "on"
//	  "requireLogin": true,    // true only when mode="on"
//	  "canPublish": false,     // false when mode="on" (production disables publishing)
//	  "keycloak": {            // present only when mode is not "off"
//	    "url": "https://auth.statgate.ug",
//	    "realm": "StatGate",
//	    "clientId": "statgate-report-browser"
//	  }
//	}
func GetAuthConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	response := AuthConfigResponse{
		Mode:         authMode,
		RequireLogin: authMode == "on",
		CanPublish:   authMode == "off", // Publishing disabled in production
	}

	// Include Keycloak config when auth is enabled (mode != "off")
	if authMode != "off" && keycloakURL != "" && keycloakRealm != "" {
		response.Keycloak = &KeycloakConfigPublic{
			URL:           keycloakURL,
			Realm:         keycloakRealm,
			ClientID:      keycloakClientID,
			RolesClientID: getAppClientID(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetFrontendConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var clientAPIURL = os.Getenv("PORTAL_API_CLIENT_URL")

	if clientAPIURL == "" {
		logWarn("PORTAL_API_ClIENT_URL is not set in environment!")
	}

	askEnabled := strings.ToLower(os.Getenv("ASK_ENABLED")) == "on"

	response := FrontendConfigResponse{
		ClientAPIUrl: clientAPIURL,
		AskEnabled:   askEnabled,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEffectivePermissionsFromRequest returns the user's merged role-permissions.
func GetEffectivePermissionsFromRequest(r *http.Request) *UserPermissions {
	claims := GetUserClaims(r)
	if claims == nil || authMode != "on" {
		return &UserPermissions{Paths: []string{"*"}, Builder: true, Admin: true}
	}
	clientID := keycloakClientID
	if keycloakRolesClient != "" {
		clientID = keycloakRolesClient
	}
	if clientID == "" {
		clientID = "dwhlanding"
	}
	roles, _ := claims.ClientRoles[clientID]
	if len(roles) == 0 && clientID != "dwhlanding" {
		roles = claims.ClientRoles["dwhlanding"]
	}
	perms, err := GetEffectivePermissionsForUser(roles)
	if err != nil || perms == nil {
		return &UserPermissions{Paths: []string{"*"}, Builder: false, Admin: false}
	}
	if HasKeycloakAdminRole(r) {
		perms.Admin = true
	}
	return perms
}

// GetMePermissionsHandler exposes current menu permissions to the frontend.
func GetMePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	perms := GetEffectivePermissionsFromRequest(r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"paths":   perms.Paths,
		"builder": perms.Builder,
		"admin":   perms.Admin,
	})
}
