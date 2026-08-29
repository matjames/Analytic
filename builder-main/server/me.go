package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type MeEnvelope struct {
	Success bool                   `json:"success"`
	Data    MeData                 `json:"data"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

type MeData struct {
	User AuthUserProfile `json:"user"`
}

type AuthUserProfile struct {
	ID                string              `json:"id"`
	Username          string              `json:"username"`
	Email             string              `json:"email,omitempty"`
	FirstName         string              `json:"firstName,omitempty"`
	LastName          string              `json:"lastName,omitempty"`
	FullName          string              `json:"fullName,omitempty"`
	IsAdmin           bool                `json:"isAdmin"`
	IsUser            bool                `json:"isUser"`
	RealmRoles        []string            `json:"realmRoles"`
	ClientRoles       map[string][]string `json:"clientRoles"`
	Permissions       []string            `json:"permissions"`
	Systems           []string            `json:"systems"`
	AccessibleSystems []SystemAccess      `json:"accessibleSystems"`
	Enabled           bool                `json:"enabled"`
	EmailVerified     bool                `json:"emailVerified"`
	RequirePwdChange  bool                `json:"requirePwdChange"`
	LastLoginAt       *string             `json:"lastLoginAt,omitempty"`
	CreatedAt         *string             `json:"createdAt,omitempty"`
	UpdatedAt         *string             `json:"updatedAt,omitempty"`
}

type SystemAccess struct {
	ClientID          string   `json:"clientId"`
	DisplayName       string   `json:"displayName"`
	LaunchURL         string   `json:"launchUrl,omitempty"`
	Icon              string   `json:"icon,omitempty"`
	Category          string   `json:"category,omitempty"`
	Navigation        string   `json:"navigation,omitempty"`
	SystemType        string   `json:"systemType,omitempty"`
	DisplayInLauncher *bool    `json:"displayInLauncher,omitempty"`
	DisplayInSideNav  *bool    `json:"displayInSideNav,omitempty"`
	LaunchMode        string   `json:"launchMode,omitempty"`
	SortOrder         *int     `json:"sortOrder,omitempty"`
	Roles             []string `json:"roles"`
}

// HandleMe exposes an StatGate Portal compatible current-user profile.
// It prefers the portal /api/v1/auth/me contract and falls back to local JWT claims
// so DWH can still operate when the portal profile service is unavailable.
func HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := extractBearerToken(r)
	if token != "" {
		if envelope, err := fetchPortalMe(r.Context(), token); err == nil {
			writeMeEnvelope(w, envelope)
			return
		} else {
			logWarn("/api/me: portal profile fetch failed, using local claims fallback: %v", err)
		}
	}

	claims := GetUserClaims(r)
	if authMode == "on" && claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	writeMeEnvelope(w, MeEnvelope{
		Success: true,
		Data:    MeData{User: fallbackMeFromClaims(claims)},
		Meta:    map[string]interface{}{},
	})
}

func writeMeEnvelope(w http.ResponseWriter, envelope MeEnvelope) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(envelope)
}

func fetchPortalMe(ctx context.Context, token string) (MeEnvelope, error) {
	url := portalAuthMeURL()
	if url == "" {
		return MeEnvelope{}, fmt.Errorf("PORTAL_AUTH_ME_URL or PORTAL_API_BASE_URL is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return MeEnvelope{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return MeEnvelope{}, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return MeEnvelope{}, fmt.Errorf("portal profile returned status %d", res.StatusCode)
	}

	var envelope MeEnvelope
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return MeEnvelope{}, err
	}
	if !envelope.Success || envelope.Data.User.ID == "" {
		return MeEnvelope{}, fmt.Errorf("portal profile response did not include a user")
	}
	normalizeMeEnvelope(&envelope)
	return envelope, nil
}

func portalAuthMeURL() string {
	if value := strings.TrimSpace(os.Getenv("PORTAL_AUTH_ME_URL")); value != "" {
		return value
	}
	if base := strings.TrimRight(strings.TrimSpace(os.Getenv("PORTAL_API_BASE_URL")), "/"); base != "" {
		return base + "/api/v1/auth/me"
	}
	return ""
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
}

func fallbackMeFromClaims(claims *UserClaims) AuthUserProfile {
	if claims == nil {
		return AuthUserProfile{
			ID:                "dev-user",
			Username:          "developer",
			FullName:          "Developer",
			IsAdmin:           true,
			IsUser:            true,
			RealmRoles:        []string{"admin", "user"},
			ClientRoles:       map[string][]string{},
			Permissions:       []string{},
			Systems:           []string{},
			AccessibleSystems: []SystemAccess{},
			Enabled:           true,
			EmailVerified:     true,
		}
	}

	username := claims.Username
	if username == "" {
		username = claims.Email
	}
	profile := AuthUserProfile{
		ID:                claims.Subject,
		Username:          username,
		Email:             claims.Email,
		FullName:          claims.Name,
		RealmRoles:        copyStrings(claims.RealmRoles),
		ClientRoles:       copyClientRoles(claims.ClientRoles),
		Permissions:       []string{},
		Systems:           []string{},
		AccessibleSystems: []SystemAccess{},
		Enabled:           true,
		EmailVerified:     claims.EmailVerified,
	}
	profile.IsAdmin = hasRealmRole(claims, "admin") || HasAnyClientRole(claims, "admin", "portal_admin")
	profile.IsUser = hasRealmRole(claims, "user") || HasAnyClientRole(claims, "user", "portal_user")
	return profile
}

func normalizeMeEnvelope(envelope *MeEnvelope) {
	if envelope.Meta == nil {
		envelope.Meta = map[string]interface{}{}
	}
	user := &envelope.Data.User
	if user.RealmRoles == nil {
		user.RealmRoles = []string{}
	}
	if user.ClientRoles == nil {
		user.ClientRoles = map[string][]string{}
	}
	if user.Permissions == nil {
		user.Permissions = []string{}
	}
	if user.Systems == nil {
		user.Systems = []string{}
	}
	if user.AccessibleSystems == nil {
		user.AccessibleSystems = []SystemAccess{}
	}
	for idx := range user.AccessibleSystems {
		if user.AccessibleSystems[idx].Roles == nil {
			user.AccessibleSystems[idx].Roles = []string{}
		}
	}
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func copyClientRoles(values map[string][]string) map[string][]string {
	out := map[string][]string{}
	for key, roles := range values {
		out[key] = copyStrings(roles)
	}
	return out
}

func HasAnyClientRole(claims *UserClaims, roles ...string) bool {
	if claims == nil {
		return false
	}
	for _, clientRoles := range claims.ClientRoles {
		for _, clientRole := range clientRoles {
			for _, role := range roles {
				if clientRole == role {
					return true
				}
			}
		}
	}
	return false
}
