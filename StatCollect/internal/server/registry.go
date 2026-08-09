package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RegistryIdentity integrates StatCollect with the StatGate unified identity
// model. The Registry issues JWT tokens signed with
// STATGATE_REGISTRY_JWT_SECRET that all services validate.
type RegistryIdentity struct {
	JWTSecret   string
	RegistryURL string
	Enabled     bool
	Client      *http.Client
}

var registry *RegistryIdentity

// InitRegistry sets up unified identity validation against Registry JWT.
func InitRegistry(jwtSecret, registryURL string, enabled bool) *RegistryIdentity {
	registry = &RegistryIdentity{
		JWTSecret:   jwtSecret,
		RegistryURL: registryURL,
		Enabled:     enabled,
		Client:      &http.Client{Timeout: 10 * time.Second},
	}
	if !enabled {
		log.Printf("Registry identity integration disabled")
	} else if jwtSecret == "" {
		log.Printf("warning: Registry identity enabled but no JWT secret configured")
	} else {
		log.Printf("Registry identity integration enabled")
	}
	return registry
}

// ParseToken validates a JWT token issued by the StatGate Registry.
// Returns the user identity claims on success.
func (r *RegistryIdentity) ParseToken(tokenString string) (jwt.MapClaims, error) {
	if r == nil || !r.Enabled {
		return nil, fmt.Errorf("registry identity disabled")
	}
	if r.JWTSecret == "" {
		return nil, fmt.Errorf("registry JWT secret not configured")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(r.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// Identity represents an authenticated StatGate user.
type Identity struct {
	UserID   string   `json:"user_id"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	TenantID string   `json:"tenant_id"`
}

// IdentityFromToken extracts an Identity from a Registry JWT.
func IdentityFromToken(claims jwt.MapClaims) Identity {
	ident := Identity{}
	if v, ok := claims["sub"].(string); ok && v != "" {
		ident.UserID = v
	}
	if ident.UserID == "" {
		ident.UserID, _ = claims["userId"].(string)
	}
	if ident.UserID == "" {
		ident.UserID, _ = claims["user_id"].(string)
	}
	if v, ok := claims["name"].(string); ok {
		ident.Name = v
	}
	if v, ok := claims["email"].(string); ok {
		ident.Email = v
	}
	if v, ok := claims["tenantId"].(string); ok {
		ident.TenantID = v
	}
	if v, ok := claims["role"].(string); ok && v != "" {
		ident.Roles = []string{v}
	}
	if arr, ok := claims["roles"].([]interface{}); ok {
		for _, r := range arr {
			if s, ok := r.(string); ok && s != "" {
				ident.Roles = append(ident.Roles, s)
			}
		}
	}
	return ident
}

// checkRegistryToken validates the Authorization header as a Registry JWT.
// Returns the identity if valid, nil otherwise.
func checkRegistryToken(r *http.Request) *Identity {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil
	}
	tokenString := strings.TrimPrefix(auth, "Bearer ")
	if registry == nil || !registry.Enabled {
		return nil
	}
	claims, err := registry.ParseToken(tokenString)
	if err != nil {
		return nil
	}
	ident := IdentityFromToken(claims)
	if ident.UserID == "" {
		return nil
	}
	return &ident
}

// authorizerHeader returns the authorized identity header value for
// service-to-service calls to the Go Core (X-User-ID).
func authorizerHeader(ident *Identity) string {
	if ident == nil {
		return ""
	}
	b, _ := json.Marshal(map[string]string{
		"user_id":   ident.UserID,
		"name":      ident.Name,
		"tenant_id": ident.TenantID,
	})
	return string(b)
}

// verifyWithRegistry calls the Registry API to verify a token.
// This is a secondary verification path for stronger validation.
func (r *RegistryIdentity) verifyWithRegistry(tokenString string) (bool, error) {
	if r == nil || !r.Enabled || r.RegistryURL == "" {
		return false, fmt.Errorf("registry verification disabled")
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		r.RegistryURL+"/v1/auth/verify", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+tokenString)
	resp, err := r.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}

// IsEnabled returns whether Registry identity integration is active.
func (r *RegistryIdentity) IsEnabled() bool {
	return r != nil && r.Enabled
}

// String returns a human-readable description of the Registry integration.
func (r *RegistryIdentity) String() string {
	if r == nil || !r.Enabled {
		return "disabled"
	}
	return fmt.Sprintf("enabled (Registry at %s)", r.RegistryURL)
}
