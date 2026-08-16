package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"statdata-backend/internal/models"
	"statdata-backend/internal/science"
	"statdata-backend/internal/store"
)

// ScienceHandler handles Interactive Notebooks, Experiments, Models, and Compute Cluster endpoints
type ScienceHandler struct {
	store        store.Store
	notebooks    *science.NotebookRunner
	experiments  *science.ExperimentTracker
	models       *science.ModelRegistry
	orchestrator *science.ClusterOrchestrator
}

// NewScienceHandler creates a science handler
func NewScienceHandler(
	s store.Store,
	nb *science.NotebookRunner,
	et *science.ExperimentTracker,
	mr *science.ModelRegistry,
	co *science.ClusterOrchestrator,
) *ScienceHandler {
	return &ScienceHandler{
		store:        s,
		notebooks:    nb,
		experiments:  et,
		models:       mr,
		orchestrator: co,
	}
}

// ─── Notebook Handlers ──────────────────────────────────────────────────────

// ListNotebooks handles GET /api/science/notebooks
func (h *ScienceHandler) ListNotebooks(c *gin.Context) {
	tenantID := getTenantID(c)
	language := c.Query("language")
	notebooks, err := h.store.ListNotebookSessions(c.Request.Context(), tenantID, language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"notebooks": notebooks})
}

// CreateNotebook handles POST /api/science/notebooks
func (h *ScienceHandler) CreateNotebook(c *gin.Context) {
	var nb models.NotebookSession
	if err := c.ShouldBindJSON(&nb); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	nb.TenantID = getTenantID(c)
	nb.CreatedBy = getUserID(c)
	nb.KernelState = "IDLE"

	if err := h.store.CreateNotebookSession(c.Request.Context(), &nb); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, nb)
}

// ExecuteNotebookCell handles POST /api/science/notebooks/:id/execute
func (h *ScienceHandler) ExecuteNotebookCell(c *gin.Context) {
	sessionID := c.Param("id")
	tenantID := getTenantID(c)

	var req models.NotebookCellExecution
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cell, err := h.notebooks.ExecuteCell(c.Request.Context(), sessionID, req.CellID, req.Source, req.Language, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cell)
}

// ─── Experiment Tracking Handlers ───────────────────────────────────────────

// ListExperiments handles GET /api/science/experiments
func (h *ScienceHandler) ListExperiments(c *gin.Context) {
	tenantID := getTenantID(c)
	domain := c.Query("domain")
	exps, err := h.store.ListExperiments(c.Request.Context(), tenantID, domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"experiments": exps})
}

// CreateExperiment handles POST /api/science/experiments
func (h *ScienceHandler) CreateExperiment(c *gin.Context) {
	var exp models.Experiment
	if err := c.ShouldBindJSON(&exp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	exp.TenantID = getTenantID(c)
	exp.CreatedBy = getUserID(c)

	if err := h.store.CreateExperiment(c.Request.Context(), &exp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, exp)
}

// StartExperimentRun handles POST /api/science/experiments/:id/runs
func (h *ScienceHandler) StartExperimentRun(c *gin.Context) {
	expID := c.Param("id")
	tenantID := getTenantID(c)
	user := getUserID(c)

	var req struct {
		RunName    string                 `json:"run_name" binding:"required"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	run, err := h.experiments.StartRun(c.Request.Context(), expID, req.RunName, user, tenantID, req.Parameters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, run)
}

// LogMetrics handles POST /api/science/runs/:id/metrics
func (h *ScienceHandler) LogMetrics(c *gin.Context) {
	runID := c.Param("id")
	var req struct {
		Metrics map[string]float64 `json:"metrics" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	run, err := h.experiments.LogMetrics(c.Request.Context(), runID, req.Metrics)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, run)
}

// ─── ML Model Registry Handlers ─────────────────────────────────────────────

// ListModels handles GET /api/science/models
func (h *ScienceHandler) ListModels(c *gin.Context) {
	tenantID := getTenantID(c)
	domain := c.Query("domain")
	mods, err := h.store.ListRegisteredModels(c.Request.Context(), tenantID, domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"models": mods})
}

// CreateModel handles POST /api/science/models
func (h *ScienceHandler) CreateModel(c *gin.Context) {
	var mod models.RegisteredModel
	if err := c.ShouldBindJSON(&mod); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mod.TenantID = getTenantID(c)
	mod.CreatedBy = getUserID(c)
	mod.LatestStage = models.ModelStageNone

	if err := h.store.CreateRegisteredModel(c.Request.Context(), &mod); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, mod)
}

// RegisterModelVersion handles POST /api/science/models/:id/versions
func (h *ScienceHandler) RegisterModelVersion(c *gin.Context) {
	modelID := c.Param("id")
	tenantID := getTenantID(c)
	user := getUserID(c)

	var req struct {
		ArtifactURI string             `json:"artifact_uri" binding:"required"`
		Metrics     map[string]float64 `json:"metrics"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mv, err := h.models.RegisterNewVersion(c.Request.Context(), modelID, req.ArtifactURI, user, tenantID, req.Metrics)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, mv)
}

// TransitionModelStage handles POST /api/science/versions/:id/stage
func (h *ScienceHandler) TransitionModelStage(c *gin.Context) {
	versionID := c.Param("id")
	tenantID := getTenantID(c)

	var req struct {
		Stage models.ModelStage `json:"stage" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.models.TransitionStage(c.Request.Context(), versionID, req.Stage, tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stage transition successful", "stage": req.Stage})
}

// ─── Compute Cluster Handlers ───────────────────────────────────────────────

// ListComputeNodes handles GET /api/science/compute/nodes
func (h *ScienceHandler) ListComputeNodes(c *gin.Context) {
	tenantID := getTenantID(c)
	nodes, err := h.store.ListComputeNodes(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"compute_nodes": nodes})
}

// SubmitComputeJob handles POST /api/science/compute/jobs
func (h *ScienceHandler) SubmitComputeJob(c *gin.Context) {
	var job models.ComputeJob
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	job.TenantID = getTenantID(c)
	job.CreatedBy = getUserID(c)

	scheduled, err := h.orchestrator.SubmitJob(c.Request.Context(), &job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, scheduled)
}
