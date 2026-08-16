package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"statdata-backend/internal/models"
	"statdata-backend/internal/pipeline"
	"statdata-backend/internal/search"
	"statdata-backend/internal/store"
)

// CatalogHandler handles Dataset, Schema Registry, Data Quality, and Lineage endpoints
type CatalogHandler struct {
	store   store.Store
	quality *pipeline.QualityEngine
	lineage *pipeline.LineageTracker
	indexer *search.UniversalIndexer
}

// NewCatalogHandler creates a catalog handler
func NewCatalogHandler(s store.Store, qe *pipeline.QualityEngine, lt *pipeline.LineageTracker, idx *search.UniversalIndexer) *CatalogHandler {
	return &CatalogHandler{
		store:   s,
		quality: qe,
		lineage: lt,
		indexer: idx,
	}
}

// ListDatasets handles GET /api/data/catalog/datasets
func (h *CatalogHandler) ListDatasets(c *gin.Context) {
	tenantID := getTenantID(c)
	domain := c.Query("domain")
	classification := c.Query("classification")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	datasets, total, err := h.store.ListDatasets(c.Request.Context(), tenantID, domain, classification, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"datasets": datasets,
	})
}

// GetDataset handles GET /api/data/catalog/datasets/:id
func (h *CatalogHandler) GetDataset(c *gin.Context) {
	id := c.Param("id")
	ds, err := h.store.GetDatasetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dataset not found"})
		return
	}
	c.JSON(http.StatusOK, ds)
}

// CreateDataset handles POST /api/data/catalog/datasets
func (h *CatalogHandler) CreateDataset(c *gin.Context) {
	var ds models.Dataset
	if err := c.ShouldBindJSON(&ds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ds.TenantID = getTenantID(c)
	ds.CreatedBy = getUserID(c)

	if err := h.store.CreateDataset(c.Request.Context(), &ds); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Auto-index into search
	_ = h.indexer.IndexDataset(c.Request.Context(), &ds)

	c.JSON(http.StatusCreated, ds)
}

// UpdateDataset handles PUT /api/data/catalog/datasets/:id
func (h *CatalogHandler) UpdateDataset(c *gin.Context) {
	id := c.Param("id")
	var ds models.Dataset
	if err := c.ShouldBindJSON(&ds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ds.ID = id
	ds.TenantID = getTenantID(c)

	if err := h.store.UpdateDataset(c.Request.Context(), &ds); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = h.indexer.IndexDataset(c.Request.Context(), &ds)
	c.JSON(http.StatusOK, ds)
}

// DeleteDataset handles DELETE /api/data/catalog/datasets/:id
func (h *CatalogHandler) DeleteDataset(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.DeleteDataset(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "dataset deleted successfully"})
}

// ─── Schema Registry Handlers ───────────────────────────────────────────────

// RegisterSchema handles POST /api/data/schemas
func (h *CatalogHandler) RegisterSchema(c *gin.Context) {
	var schema models.SchemaDefinition
	if err := c.ShouldBindJSON(&schema); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schema.TenantID = getTenantID(c)
	schema.CreatedBy = getUserID(c)

	if err := h.store.RegisterSchema(c.Request.Context(), &schema); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, schema)
}

// ValidatePayloadAgainstSchema handles POST /api/data/schemas/validate
func (h *CatalogHandler) ValidatePayloadAgainstSchema(c *gin.Context) {
	var req struct {
		Subject string                 `json:"subject" binding:"required"`
		Version int                    `json:"version"`
		Payload map[string]interface{} `json:"payload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := getTenantID(c)
	var schema *models.SchemaDefinition
	var err error

	if req.Version > 0 {
		schemaID := req.Subject + "-v" + strconv.Itoa(req.Version)
		schema, err = h.store.GetSchemaByID(c.Request.Context(), schemaID)
	} else {
		schema, err = h.store.GetLatestSchemaBySubject(c.Request.Context(), req.Subject, tenantID)
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "registered schema not found for subject"})
		return
	}

	// Validate fields
	missingFields := make([]string, 0)
	for _, f := range schema.Fields {
		if !f.Nullable {
			if val, ok := req.Payload[f.Name]; !ok || val == nil || val == "" {
				missingFields = append(missingFields, f.Name)
			}
		}
	}

	valid := len(missingFields) == 0
	c.JSON(http.StatusOK, gin.H{
		"valid":          valid,
		"schema_id":      schema.ID,
		"subject":        schema.Subject,
		"version":        schema.Version,
		"missing_fields": missingFields,
		"validated_at":   time.Now().UTC(),
	})
}

// ListSchemas handles GET /api/data/schemas
func (h *CatalogHandler) ListSchemas(c *gin.Context) {
	tenantID := getTenantID(c)
	schemas, err := h.store.ListSchemas(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"schemas": schemas})
}

// ─── Data Quality Handlers ──────────────────────────────────────────────────

// EvaluateQuality handles POST /api/data/quality/evaluate
func (h *CatalogHandler) EvaluateQuality(c *gin.Context) {
	var req struct {
		DatasetID     string                   `json:"dataset_id" binding:"required"`
		PipelineRunID string                   `json:"pipeline_run_id"`
		SampleData    []map[string]interface{} `json:"sample_data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := getTenantID(c)
	rep, err := h.quality.EvaluateDataset(c.Request.Context(), req.DatasetID, req.PipelineRunID, tenantID, req.SampleData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rep)
}

// CreateQualityRule handles POST /api/data/quality/rules
func (h *CatalogHandler) CreateQualityRule(c *gin.Context) {
	var rule models.DataQualityRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rule.TenantID = getTenantID(c)
	rule.IsEnabled = true

	if err := h.store.CreateQualityRule(c.Request.Context(), &rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rule)
}

// ListQualityRules handles GET /api/data/quality/rules
func (h *CatalogHandler) ListQualityRules(c *gin.Context) {
	tenantID := getTenantID(c)
	datasetID := c.Query("dataset_id")
	rules, err := h.store.ListQualityRules(c.Request.Context(), tenantID, datasetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// ─── Data Lineage Handlers ──────────────────────────────────────────────────

// GetLineageGraph handles GET /api/data/lineage/:id
func (h *CatalogHandler) GetLineageGraph(c *gin.Context) {
	id := c.Param("id")
	tenantID := getTenantID(c)
	depth, _ := strconv.Atoi(c.DefaultQuery("depth", "3"))

	graph, err := h.lineage.GetGraph(c.Request.Context(), id, tenantID, depth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, graph)
}
