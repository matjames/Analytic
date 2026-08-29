package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-backend/configs"
)

type userPreferences struct {
	Locale             string `json:"locale"`
	Timezone           string `json:"timezone"`
	Theme              string `json:"theme"`
	Density            string `json:"density"`
	EmailNotifications bool   `json:"email_notifications"`
}

func defaultPreferences() userPreferences {
	return userPreferences{Locale: "en", Timezone: "UTC", Theme: "system", Density: "comfortable", EmailNotifications: true}
}

func GetPreferences(c *gin.Context) {
	preferences := defaultPreferences()
	err := configs.DB.QueryRow(`SELECT locale, timezone, theme, density, email_notifications FROM user_preferences WHERE user_id=$1`, c.GetInt64("user_id")).Scan(&preferences.Locale, &preferences.Timezone, &preferences.Theme, &preferences.Density, &preferences.EmailNotifications)
	if err != nil {
		if _, insertErr := configs.DB.Exec(`INSERT INTO user_preferences (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, c.GetInt64("user_id")); insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to initialize preferences"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"preferences": preferences})
}

func UpdatePreferences(c *gin.Context) {
	var request userPreferences
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid preferences"})
		return
	}
	request.Locale = strings.TrimSpace(request.Locale)
	request.Timezone = strings.TrimSpace(request.Timezone)
	request.Theme = strings.TrimSpace(request.Theme)
	request.Density = strings.TrimSpace(request.Density)
	if request.Locale != "en" || request.Timezone == "" || len(request.Timezone) > 64 || (request.Theme != "system" && request.Theme != "light" && request.Theme != "dark") || (request.Density != "comfortable" && request.Density != "compact") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported preference value"})
		return
	}
	_, err := configs.DB.Exec(`INSERT INTO user_preferences (user_id, locale, timezone, theme, density, email_notifications, updated_at) VALUES ($1,$2,$3,$4,$5,$6,NOW()) ON CONFLICT (user_id) DO UPDATE SET locale=EXCLUDED.locale, timezone=EXCLUDED.timezone, theme=EXCLUDED.theme, density=EXCLUDED.density, email_notifications=EXCLUDED.email_notifications, updated_at=NOW()`, c.GetInt64("user_id"), request.Locale, request.Timezone, request.Theme, request.Density, request.EmailNotifications)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to save preferences"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"preferences": request})
}
