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

func HashPassword(p string) string {
	b, _ := bcrypt.GenerateFromPassword([]byte(p), 10)
	return string(b)
}

func CheckPassword(hash, p string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)) == nil
}

func CreateToken(userID int64) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(6 * time.Hour).Unix(),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, _ := t.SignedString([]byte(os.Getenv("JWT_SECRET")))
	return token
}

// SignUserToken creates the shared StatGate identity token. The legacy userId
// and districtId claims are retained for existing services while the standard
// claims let every app establish the same account without a second login.
func SignUserToken(user models.User) (string, error) {
	name := strings.TrimSpace(strings.Join([]string{valueOrEmpty(user.FirstName), valueOrEmpty(user.LastName)}, " "))
	if name == "" {
		name = user.Username
	}
	claims := jwt.MapClaims{
		"sub":    fmt.Sprint(user.ID),
		"userId": user.ID,
		"name":   name,
		"email":  user.Email,
		"exp":    time.Now().Add(6 * time.Hour).Unix(),
	}

	if user.Role != nil {
		claims["role"] = *user.Role
	}

	if user.DistrictID != nil {
		claims["districtId"] = *user.DistrictID
	} else {
		claims["districtId"] = nil
	}
	if user.Organisation != nil && strings.TrimSpace(*user.Organisation) != "" {
		claims["tenantId"] = *user.Organisation
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
