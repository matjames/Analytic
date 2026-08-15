package utils

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go-backend/models"
	"golang.org/x/crypto/bcrypt"
)

// jwtEnv returns the registry signing secret. The canonical StatGate secret
// (STATGATE_REGISTRY_JWT_SECRET) takes precedence; the legacy JWT_SECRET is
// accepted only during migration and MUST be removed afterwards.
func jwtEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// jwtSecret returns the configured signing secret or an error if absent.
func jwtSecret() (string, error) {
	secret := jwtEnv("STATGATE_REGISTRY_JWT_SECRET", "")
	if secret == "" {
		secret = jwtEnv("JWT_SECRET", "")
	}
	if secret == "" {
		return "", fmt.Errorf("STATGATE_REGISTRY_JWT_SECRET is not configured")
	}
	return secret, nil
}

func HashPassword(p string) string {
	b, _ := bcrypt.GenerateFromPassword([]byte(p), 10)
	return string(b)
}

func CheckPassword(hash, p string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)) == nil
}

// CreateToken signs a minimal token used internally by legacy callers.
func CreateToken(userID int64) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":    fmt.Sprint(userID),
		"user_id": userID,
		"iat":    now.Unix(),
		"nbf":    now.Add(-30 * time.Second).Unix(),
		"exp":    now.Add(6 * time.Hour).Unix(),
		"iss":    jwtEnv("STATGATE_JWT_ISSUER", "statgate-registry"),
		"aud":    jwtEnv("STATGATE_JWT_AUDIENCE", "statgate"),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := jwtSecret()
	if err != nil {
		return ""
	}
	token, _ := t.SignedString([]byte(secret))
	return token
}

// SignUserToken creates the shared StatGate identity token. It follows the
// canonical JWT contract (docs/security/JWT_STANDARD.md): all services read
// sub, tenant_id, org_id, role, email, iat, nbf, exp, iss and aud from the
// same claim keys. Legacy aliases (userId, tenantId, districtId) are emitted
// for the React frontends but are never used as the enforcement source.
func SignUserToken(user models.User) (string, error) {
	name := strings.TrimSpace(strings.Join([]string{valueOrEmpty(user.FirstName), valueOrEmpty(user.LastName)}, " "))
	if name == "" {
		name = user.Username
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":    fmt.Sprint(user.ID),
		"userId": user.ID,
		"name":   name,
		"email":  user.Email,
		"iat":    now.Unix(),
		"nbf":    now.Add(-30 * time.Second).Unix(),
		"exp":    now.Add(6 * time.Hour).Unix(),
		"iss":    jwtEnv("STATGATE_JWT_ISSUER", "statgate-registry"),
		"aud":    jwtEnv("STATGATE_JWT_AUDIENCE", "statgate"),
	}

	if user.Role != nil {
		claims["role"] = *user.Role
	}

	// Canonical tenant binding: organisation when present, otherwise fall back
	// to the district-derived tenant. Both keys are written.
	if user.Organisation != nil && strings.TrimSpace(*user.Organisation) != "" {
		org := strings.TrimSpace(*user.Organisation)
		claims["tenant_id"] = org
		claims["org_id"] = org
		claims["tenantId"] = org
		claims["organization_id"] = org
	} else if user.DistrictID != nil && strings.TrimSpace(*user.DistrictID) != "" {
		district := strings.TrimSpace(*user.DistrictID)
		claims["tenant_id"] = district
		claims["org_id"] = "district:" + district
		claims["tenantId"] = district
	}

	if user.DistrictID != nil {
		claims["districtId"] = *user.DistrictID
		claims["district_id"] = *user.DistrictID
	}

	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
