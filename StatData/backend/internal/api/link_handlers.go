package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"statdata-backend/internal/models"
	"statdata-backend/internal/store"
)

// LinkHandler handles cross-application Object Links in the StatGate ecosystem
type LinkHandler struct {
	store store.Store
}

// NewLinkHandler creates a new link handler
func NewLinkHandler(s store.Store) *LinkHandler {
	return &LinkHandler{store: s}
}

// CreateObjectLink handles POST /api/data/links
func (h *LinkHandler) CreateObjectLink(c *gin.Context) {
	var link models.ObjectLink
	if err := c.ShouldBindJSON(&link); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	link.TenantID = getTenantID(c)
	link.CreatedBy = getUserID(c)

	if err := h.store.CreateObjectLink(c.Request.Context(), &link); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, link)
}

// GetObjectLinks handles GET /api/data/links
func (h *LinkHandler) GetObjectLinks(c *gin.Context) {
	sourceType := c.Query("source_type")
	sourceID := c.Query("source_id")
	tenantID := getTenantID(c)

	if sourceType == "" || sourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_type and source_id query parameters are required"})
		return
	}

	links, err := h.store.GetObjectLinks(c.Request.Context(), sourceType, sourceID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"links": links})
}

// ListAuditLogs handles GET /api/data/audit/logs
func (h *LinkHandler) ListAuditLogs(c *gin.Context) {
	tenantID := getTenantID(c)
	resType := c.Query("resource_type")
	logs, err := h.store.ListAuditLogs(c.Request.Context(), tenantID, resType, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"audit_logs": logs})
}
