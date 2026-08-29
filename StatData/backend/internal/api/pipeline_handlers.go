package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"statdata-backend/internal/models"
	"statdata-backend/internal/pipeline"
	"statdata-backend/internal/store"
)

// PipelineHandler handles Pipeline execution, streaming, and CDC API routes
type PipelineHandler struct {
	store    store.Store
	pipeline *pipeline.PipelineEngine
}

// NewPipelineHandler creates a pipeline handler
func NewPipelineHandler(s store.Store, pe *pipeline.PipelineEngine) *PipelineHandler {
	return &PipelineHandler{
		store:    s,
		pipeline: pe,
	}
}

// ListPipelines handles GET /api/data/pipelines
func (h *PipelineHandler) ListPipelines(c *gin.Context) {
	tenantID := getTenantID(c)
	workspaceID := getWorkspaceID(c)
	status := c.Query("status")
	pipes, err := h.store.ListPipelines(c.Request.Context(), tenantID, status, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pipelines": pipes})
}

// GetPipeline handles GET /api/data/pipelines/:id
func (h *PipelineHandler) GetPipeline(c *gin.Context) {
	id := c.Param("id")
	p, err := h.store.GetPipelineByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}
	if !workspaceAllowsRead(p.WorkspaceID, getWorkspaceID(c)) {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// CreatePipeline handles POST /api/data/pipelines
func (h *PipelineHandler) CreatePipeline(c *gin.Context) {
	var p models.DataPipeline
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.TenantID = getTenantID(c)
	p.WorkspaceID = getWorkspaceID(c)
	p.CreatedBy = getUserID(c)
	p.Status = models.PipelineStatusActive

	if err := h.store.CreatePipeline(c.Request.Context(), &p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// TriggerPipelineRun handles POST /api/data/pipelines/:id/run
func (h *PipelineHandler) TriggerPipelineRun(c *gin.Context) {
	id := c.Param("id")
	pipe, err := h.store.GetPipelineByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}
	if !workspaceAllowsRead(pipe.WorkspaceID, getWorkspaceID(c)) {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}
	tenantID := getTenantID(c)
	user := getUserID(c)

	run, err := h.pipeline.TriggerPipeline(c.Request.Context(), id, "MANUAL", user, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, run)
}

// ListPipelineRuns handles GET /api/data/pipelines/:id/runs
func (h *PipelineHandler) ListPipelineRuns(c *gin.Context) {
	id := c.Param("id")
	tenantID := getTenantID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	runs, err := h.store.ListPipelineRuns(c.Request.Context(), id, tenantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"runs": runs})
}

// ─── Streaming Handlers ─────────────────────────────────────────────────────

// ListStreamingJobs handles GET /api/data/streaming
func (h *PipelineHandler) ListStreamingJobs(c *gin.Context) {
	tenantID := getTenantID(c)
	workspaceID := getWorkspaceID(c)
	status := c.Query("status")
	jobs, err := h.store.ListStreamingJobs(c.Request.Context(), tenantID, status, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"streaming_jobs": jobs})
}

// CreateStreamingJob handles POST /api/data/streaming
func (h *PipelineHandler) CreateStreamingJob(c *gin.Context) {
	var job models.StreamingJob
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	job.TenantID = getTenantID(c)
	job.WorkspaceID = getWorkspaceID(c)
	job.Status = "RUNNING"

	if err := h.store.CreateStreamingJob(c.Request.Context(), &job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, job)
}

// IngestCDCEvent handles POST /api/data/cdc/events
func (h *PipelineHandler) IngestCDCEvent(c *gin.Context) {
	var event models.CDCEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	event.TenantID = getTenantID(c)

	if err := h.pipeline.ProcessCDCEvent(c.Request.Context(), &event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "CDC event processed", "event_id": event.EventID})
}
