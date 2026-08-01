package handlers

import (
	"net/http"
	"os"
	"time"

	"go-backend/configs"
	"go-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var user models.User

	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(user.Password), 10)

	_, err := configs.DB.Exec(
		"INSERT INTO users(email,username,password,role) VALUES($1,$2,$3,$4)",
		user.Email, user.Username, string(hash), user.Role,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "registered"})
}

func Login(c *gin.Context) {
	var user models.User

	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	var dbUser models.User

	err := configs.DB.QueryRow(
		"SELECT id,password FROM users WHERE email=$1",
		user.Email,
	).Scan(&dbUser.ID, &dbUser.PasswordHash)

	if err != nil ||
		bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(user.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	claims := jwt.MapClaims{
		"user_id": dbUser.ID,
		"exp":     time.Now().Add(2 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": t})
}
