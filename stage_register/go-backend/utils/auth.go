package utils

import (
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
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
		"sub":     fmt.Sprint(userID),
		"user_id": userID,
		"iat":     now.Unix(),
		"nbf":     now.Add(-30 * time.Second).Unix(),
		"exp":     now.Add(6 * time.Hour).Unix(),
		"iss":     jwtEnv("STATGATE_JWT_ISSUER", "statgate-registry"),
		"aud":     jwtEnv("STATGATE_JWT_AUDIENCE", "statgate"),
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

	return signClaims(claims)
}

// SignRefreshToken issues a typed, long-lived token that can only be used at
// the Registry refresh endpoint. Downstream services accept access tokens
// only, so a stolen refresh token cannot be used as an API credential.
func SignRefreshToken(user models.User) (string, error) {
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("generate refresh token id: %w", err)
	}

	claims := jwt.MapClaims{
		"sub": fmt.Sprint(user.ID),
		"typ": "refresh",
		"jti": fmt.Sprintf("%x", nonce[:]),
		"iat": time.Now().Unix(),
		"nbf": time.Now().Add(-30 * time.Second).Unix(),
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iss": jwtEnv("STATGATE_JWT_ISSUER", "statgate-registry"),
		"aud": jwtEnv("STATGATE_JWT_AUDIENCE", "statgate"),
	}
	return signClaims(claims)
}

// RefreshSubject validates a refresh token and returns its Registry user id.
func RefreshSubject(tokenString string) (int64, error) {
	secret, err := jwtSecret()
	if err != nil {
		return 0, err
	}

	issuer := jwtEnv("STATGATE_JWT_ISSUER", "statgate-registry")
	audience := jwtEnv("STATGATE_JWT_AUDIENCE", "statgate")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unsupported signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(issuer), jwt.WithAudience(audience))
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != "refresh" {
		return 0, fmt.Errorf("token is not a refresh token")
	}
	sub, ok := claims["sub"].(string)
	if !ok {
		return 0, fmt.Errorf("refresh token missing subject")
	}
	userID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil || userID <= 0 {
		return 0, fmt.Errorf("refresh token has invalid subject")
	}
	return userID, nil
}

func signClaims(claims jwt.MapClaims) (string, error) {
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
