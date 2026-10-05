package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AppendDatasetRecords handles POST /api/v1/data/catalog/datasets/:id/records
// and stores real rows into the dataset's data-plane storage.
func (h *CatalogHandler) AppendDatasetRecords(c *gin.Context) {
	id := c.Param("id")
	ds, err := h.store.GetDatasetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dataset not found"})
		return
	}
	if !workspaceAllowsRead(ds.WorkspaceID, getWorkspaceID(c)) {
		c.JSON(http.StatusNotFound, gin.H{"error": "dataset not found"})
		return
	}

	var body struct {
		Records []map[string]interface{} `json:"records"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(body.Records) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "records array cannot be empty"})
		return
	}

	bytesWritten, err := h.store.AppendDatasetRecords(c.Request.Context(), id, getTenantID(c), body.Records)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	total, _ := h.store.CountDatasetRecords(c.Request.Context(), id)
	c.JSON(http.StatusCreated, gin.H{
		"dataset_id":     id,
		"records_stored": len(body.Records),
		"bytes_written":  bytesWritten,
		"total_rows":     total,
	})
}

// ListDatasetRecords handles GET /api/v1/data/catalog/datasets/:id/records
// and reads the dataset's real stored rows.
func (h *CatalogHandler) ListDatasetRecords(c *gin.Context) {
	id := c.Param("id")
	ds, err := h.store.GetDatasetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dataset not found"})
		return
	}
	if !workspaceAllowsRead(ds.WorkspaceID, getWorkspaceID(c)) {
		c.JSON(http.StatusNotFound, gin.H{"error": "dataset not found"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	records, err := h.store.ListDatasetRecords(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"dataset_id": id,
		"count":      len(records),
		"records":    records,
	})
}

// DeleteDatasetRecords handles DELETE /api/v1/data/catalog/datasets/:id/records
// and removes all stored rows for the dataset.
func (h *CatalogHandler) DeleteDatasetRecords(c *gin.Context) {
	id := c.Param("id")
	ds, err := h.store.GetDatasetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dataset not found"})
		return
	}
	if !workspaceAllowsRead(ds.WorkspaceID, getWorkspaceID(c)) {
		c.JSON(http.StatusNotFound, gin.H{"error": "dataset not found"})
		return
	}

	deleted, err := h.store.DeleteDatasetRecords(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dataset_id": id, "deleted_rows": deleted})
}