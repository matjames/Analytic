package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

// SignUserToken creates a JWT token for a user (matching Node.js implementation)
func SignUserToken(userID int64, role *string, districtID *string) (string, error) {
	claims := jwt.MapClaims{
		"userId": userID,
		"exp":    time.Now().Add(6 * time.Hour).Unix(),
	}

	if role != nil {
		claims["role"] = *role
	}

	if districtID != nil {
		claims["districtId"] = *districtID
	} else {
		claims["districtId"] = nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}