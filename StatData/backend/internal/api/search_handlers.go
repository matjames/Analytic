package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"statdata-backend/internal/models"
	"statdata-backend/internal/search"
	"statdata-backend/internal/store"
)

// SearchHandler handles Universal & Semantic Search API routes
type SearchHandler struct {
	store   store.Store
	engine  *search.SearchEngine
	indexer *search.UniversalIndexer
}

// NewSearchHandler creates a search handler
func NewSearchHandler(s store.Store, se *search.SearchEngine, idx *search.UniversalIndexer) *SearchHandler {
	return &SearchHandler{
		store:   s,
		engine:  se,
		indexer: idx,
	}
}

// Search handles POST /api/search/query and POST /api/search/hybrid
func (h *SearchHandler) Search(c *gin.Context) {
	var req models.HybridSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TenantID == "" {
		req.TenantID = getTenantID(c)
	}

	res, err := h.engine.ExecuteHybridSearch(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// AutoComplete handles GET /api/search/suggest
func (h *SearchHandler) AutoComplete(c *gin.Context) {
	prefix := c.Query("q")
	tenantID := getTenantID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	suggestions, err := h.engine.AutoComplete(c.Request.Context(), prefix, tenantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// IndexDocument handles POST /api/search/index
func (h *SearchHandler) IndexDocument(c *gin.Context) {
	var doc models.IndexedDocument
	if err := c.ShouldBindJSON(&doc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc.TenantID = getTenantID(c)
	doc.WorkspaceID = getWorkspaceID(c)
	if doc.IndexName == "" {
		doc.IndexName = "statgate_global"
	}

	if err := h.indexer.IndexGeneric(c.Request.Context(), doc.ResourceID, doc.ResourceType, doc.Title, doc.Content, doc.Domain, doc.Classification, doc.TenantID, doc.Tags); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "document indexed", "resource_id": doc.ResourceID})
}

// ListSavedSearches handles GET /api/search/saved
func (h *SearchHandler) ListSavedSearches(c *gin.Context) {
	tenantID := getTenantID(c)
	user := getUserID(c)
	saved, err := h.store.ListSavedSearches(c.Request.Context(), tenantID, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"saved_searches": saved})
}

// CreateSavedSearch handles POST /api/search/saved
func (h *SearchHandler) CreateSavedSearch(c *gin.Context) {
	var ss models.SavedSearch
	if err := c.ShouldBindJSON(&ss); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ss.TenantID = getTenantID(c)
	ss.WorkspaceID = getWorkspaceID(c)
	ss.CreatedBy = getUserID(c)

	if err := h.store.CreateSavedSearch(c.Request.Context(), &ss); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ss)
}
