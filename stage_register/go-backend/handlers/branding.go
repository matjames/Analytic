package handlers

import (
	"database/sql"
	"net/http"
	"regexp"
	"strings"

	"go-backend/configs"

	"github.com/gin-gonic/gin"
)

var brandingColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type brandingResponse struct {
	TenantID       string `json:"tenant_id"`
	DisplayName    string `json:"display_name"`
	LogoURL        string `json:"logo_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
}

func authenticatedTenant(c *gin.Context) (string, bool) {
	tenantID := strings.TrimSpace(c.GetString("tenant_id"))
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authenticated tenant context is required"})
		return "", false
	}
	return tenantID, true
}

func GetBranding(c *gin.Context) {
	tenantID, ok := authenticatedTenant(c)
	if !ok {
		return
	}
	var branding brandingResponse
	err := configs.DB.QueryRow(`SELECT tenant_id,display_name,logo_url,primary_color,secondary_color FROM organisation_branding WHERE tenant_id=$1`, tenantID).Scan(&branding.TenantID, &branding.DisplayName, &branding.LogoURL, &branding.PrimaryColor, &branding.SecondaryColor)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, brandingResponse{TenantID: tenantID, PrimaryColor: "#0f766e", SecondaryColor: "#0f3d3e"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load organisation branding"})
		return
	}
	c.JSON(http.StatusOK, branding)
}

func UpdateBranding(c *gin.Context) {
	if !requireInvitationAdmin(c) {
		return
	}
	tenantID, ok := authenticatedTenant(c)
	if !ok {
		return
	}
	var req struct {
		DisplayName    string `json:"display_name"`
		LogoURL        string `json:"logo_url"`
		PrimaryColor   string `json:"primary_color"`
		SecondaryColor string `json:"secondary_color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branding payload"})
		return
	}
	req.DisplayName, req.LogoURL = strings.TrimSpace(req.DisplayName), strings.TrimSpace(req.LogoURL)
	req.PrimaryColor, req.SecondaryColor = strings.TrimSpace(req.PrimaryColor), strings.TrimSpace(req.SecondaryColor)
	if req.DisplayName == "" || len(req.DisplayName) > 120 || len(req.LogoURL) > 1000 || !brandingColorPattern.MatchString(req.PrimaryColor) || !brandingColorPattern.MatchString(req.SecondaryColor) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "display name, logo URL, and six-digit hex colours are required"})
		return
	}
	if req.LogoURL != "" && !(strings.HasPrefix(req.LogoURL, "https://") || strings.HasPrefix(req.LogoURL, "http://")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "logo URL must use http or https"})
		return
	}
	userID := c.GetInt64("user_id")
	_, err := configs.DB.Exec(`INSERT INTO organisation_branding (tenant_id,display_name,logo_url,primary_color,secondary_color,updated_by,updated_at) VALUES ($1,$2,$3,$4,$5,$6,NOW()) ON CONFLICT (tenant_id) DO UPDATE SET display_name=EXCLUDED.display_name,logo_url=EXCLUDED.logo_url,primary_color=EXCLUDED.primary_color,secondary_color=EXCLUDED.secondary_color,updated_by=EXCLUDED.updated_by,updated_at=NOW()`, tenantID, req.DisplayName, req.LogoURL, req.PrimaryColor, req.SecondaryColor, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save organisation branding"})
		return
	}
	c.JSON(http.StatusOK, brandingResponse{TenantID: tenantID, DisplayName: req.DisplayName, LogoURL: req.LogoURL, PrimaryColor: req.PrimaryColor, SecondaryColor: req.SecondaryColor})
}
