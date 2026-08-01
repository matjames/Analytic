package handlers

import (
	"net/http"

	"go-backend/services"

	"github.com/gin-gonic/gin"
)

// GetDashboardStats - GET /dashboard/stats - Get dashboard statistics
func GetDashboardStats(c *gin.Context) {
	stats, err := services.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

