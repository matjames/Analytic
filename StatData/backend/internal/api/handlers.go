package api

import (
	"github.com/gin-gonic/gin"
	"statdata-backend/internal/events"
	"statdata-backend/internal/pipeline"
	"statdata-backend/internal/science"
	"statdata-backend/internal/search"
	"statdata-backend/internal/store"
)

// Handlers aggregates all domain controllers for StatData
type Handlers struct {
	CatalogHandler      *CatalogHandler
	PipelineHandler     *PipelineHandler
	FeatureStoreHandler *FeatureStoreHandler
	ScienceHandler      *ScienceHandler
	SearchHandler       *SearchHandler
	LinkHandler         *LinkHandler
	EventWorker         *events.EventWorker
}

// NewHandlers instantiates and wires all domain handlers
func NewHandlers(
	s store.Store,
	pe *pipeline.PipelineEngine,
	qe *pipeline.QualityEngine,
	lt *pipeline.LineageTracker,
	nb *science.NotebookRunner,
	et *science.ExperimentTracker,
	mr *science.ModelRegistry,
	co *science.ClusterOrchestrator,
	se *search.SearchEngine,
	idx *search.UniversalIndexer,
	ew *events.EventWorker,
) *Handlers {
	return &Handlers{
		CatalogHandler:      NewCatalogHandler(s, qe, lt, idx),
		PipelineHandler:     NewPipelineHandler(s, pe),
		FeatureStoreHandler: NewFeatureStoreHandler(s),
		ScienceHandler:      NewScienceHandler(s, nb, et, mr, co),
		SearchHandler:       NewSearchHandler(s, se, idx),
		LinkHandler:         NewLinkHandler(s),
		EventWorker:         ew,
	}
}

// Helper methods to extract tenant and user from context or headers
func getTenantID(c *gin.Context) string {
	if t, exists := c.Get("tenant_id"); exists {
		if s, ok := t.(string); ok && s != "" {
			return s
		}
	}
	if h := c.GetHeader("X-Tenant-ID"); h != "" {
		return h
	}
	return "default"
}

func getUserID(c *gin.Context) string {
	if u, exists := c.Get("user_id"); exists {
		if s, ok := u.(string); ok && s != "" {
			return s
		}
	}
	if h := c.GetHeader("X-User-ID"); h != "" {
		return h
	}
	return "system-architect"
}
