package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"statdata-backend/internal/models"
	"statdata-backend/internal/store"
)

// FeatureStoreHandler handles Feature View and Online/Offline Feature Store endpoints
type FeatureStoreHandler struct {
	store store.Store
}

// NewFeatureStoreHandler creates a feature store handler
func NewFeatureStoreHandler(s store.Store) *FeatureStoreHandler {
	return &FeatureStoreHandler{store: s}
}

// ListFeatureViews handles GET /api/data/features/views
func (h *FeatureStoreHandler) ListFeatureViews(c *gin.Context) {
	tenantID := getTenantID(c)
	entity := c.Query("entity_name")
	views, err := h.store.ListFeatureViews(c.Request.Context(), tenantID, entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"feature_views": views})
}

// CreateFeatureView handles POST /api/data/features/views
func (h *FeatureStoreHandler) CreateFeatureView(c *gin.Context) {
	var fv models.FeatureView
	if err := c.ShouldBindJSON(&fv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fv.TenantID = getTenantID(c)
	fv.CreatedBy = getUserID(c)

	if err := h.store.CreateFeatureView(c.Request.Context(), &fv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, fv)
}

// IngestFeatures handles POST /api/data/features/ingest
func (h *FeatureStoreHandler) IngestFeatures(c *gin.Context) {
	var req struct {
		Records []models.FeatureRecord `json:"records" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := getTenantID(c)
	for i := range req.Records {
		req.Records[i].TenantID = tenantID
	}

	if err := h.store.SaveFeatureRecords(c.Request.Context(), req.Records); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ingested_count": len(req.Records)})
}

// GetOnlineFeatures handles GET /api/data/features/online
func (h *FeatureStoreHandler) GetOnlineFeatures(c *gin.Context) {
	viewID := c.Query("view_id")
	entityKey := c.Query("entity_key")
	tenantID := getTenantID(c)

	if viewID == "" || entityKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "view_id and entity_key are required"})
		return
	}

	features, err := h.store.GetOnlineFeatures(c.Request.Context(), viewID, entityKey, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "features not found for entity"})
		return
	}
	c.JSON(http.StatusOK, features)
}
