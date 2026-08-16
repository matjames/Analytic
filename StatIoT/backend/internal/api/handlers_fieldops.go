package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"statiot-backend/internal/fieldops"
	"statiot-backend/internal/models"
	"statiot-backend/internal/store"
)

type FieldOpsHandlers struct {
	store   store.Store
	syncEng *fieldops.SyncEngine
	spatial *fieldops.SpatialTracker
}

func NewFieldOpsHandlers(st store.Store, syncEng *fieldops.SyncEngine, spatial *fieldops.SpatialTracker) *FieldOpsHandlers {
	return &FieldOpsHandlers{
		store:   st,
		syncEng: syncEng,
		spatial: spatial,
	}
}

// CreateWorker registers a field enumerator or inspector.
func (h *FieldOpsHandlers) CreateWorker(c *gin.Context) {
	var req models.FieldWorker
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worker payload", "details": err.Error()})
		return
	}

	if req.FullName == "" || req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "full_name and user_id are required"})
		return
	}
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.TenantID == "" {
		req.TenantID = "default"
	}
	if req.Role == "" {
		req.Role = "ENUMERATOR"
	}

	if err := h.store.CreateFieldWorker(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create field worker", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Field worker created", "worker": req})
}

// ListWorkers returns field workers in a tenant.
func (h *FieldOpsHandlers) ListWorkers(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	workers, err := h.store.ListFieldWorkers(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list workers", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"workers": workers, "count": len(workers)})
}

// GetWorker returns a single worker by ID.
func (h *FieldOpsHandlers) GetWorker(c *gin.Context) {
	id := c.Param("id")
	w, err := h.store.GetFieldWorker(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}
	c.JSON(http.StatusOK, w)
}

// RegisterMobileDevice registers a tablet or smartphone running ODK / StatCollect / FieldOps client.
func (h *FieldOpsHandlers) RegisterMobileDevice(c *gin.Context) {
	var req models.MobileDevice
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mobile device payload", "details": err.Error()})
		return
	}

	if req.DeviceUUID == "" || req.Platform == "" || req.AppVersion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_uuid, platform, and app_version are required"})
		return
	}
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.TenantID == "" {
		req.TenantID = "default"
	}

	if err := h.store.RegisterMobileDevice(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register mobile device", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Mobile device registered", "device": req})
}

// CreateVisit schedules an in-person field visit for an enumerator or technician.
func (h *FieldOpsHandlers) CreateVisit(c *gin.Context) {
	var req models.FieldVisit
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid visit payload", "details": err.Error()})
		return
	}

	if req.WorkerID == "" || req.Title == "" || req.TargetEntityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "worker_id, title, and target_entity_id are required"})
		return
	}
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.Status == "" {
		req.Status = "SCHEDULED"
	}
	if req.TenantID == "" {
		req.TenantID = "default"
	}
	if req.ScheduledStart.IsZero() {
		req.ScheduledStart = time.Now().UTC()
	}
	if req.ScheduledEnd.IsZero() {
		req.ScheduledEnd = req.ScheduledStart.Add(2 * time.Hour)
	}

	if err := h.store.CreateFieldVisit(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create visit", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Field visit scheduled", "visit": req})
}

// ListVisits returns visits assigned to a worker.
func (h *FieldOpsHandlers) ListVisits(c *gin.Context) {
	workerID := c.Query("worker_id")
	visits, err := h.store.ListVisitsByWorker(c.Request.Context(), workerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list visits", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"visits": visits, "count": len(visits)})
}

// CreateVisitTask attaches a task (e.g. Form Collection, Inspection) to a visit.
func (h *FieldOpsHandlers) CreateVisitTask(c *gin.Context) {
	visitID := c.Param("id")
	var req models.VisitTask
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task payload", "details": err.Error()})
		return
	}

	req.VisitID = visitID
	if req.Title == "" || req.TaskType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and task_type are required"})
		return
	}
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.Status == "" {
		req.Status = "PENDING"
	}

	if err := h.store.CreateVisitTask(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create visit task", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Visit task added", "task": req})
}

// ListVisitTasks returns tasks for a specific visit.
func (h *FieldOpsHandlers) ListVisitTasks(c *gin.Context) {
	visitID := c.Param("id")
	tasks, err := h.store.ListTasksByVisit(c.Request.Context(), visitID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks, "count": len(tasks)})
}

// CreateFormDefinition registers a JSON-schema dynamic mobile questionnaire.
func (h *FieldOpsHandlers) CreateFormDefinition(c *gin.Context) {
	var req models.MobileFormDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form definition", "details": err.Error()})
		return
	}

	if req.FormCode == "" || req.Title == "" || req.SchemaDefinition == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "form_code, title, and schema_definition are required"})
		return
	}
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.Version == 0 {
		req.Version = 1
	}
	if req.TenantID == "" {
		req.TenantID = "default"
	}

	if err := h.store.CreateFormDefinition(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create form definition", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Form definition published", "form": req})
}

// ListFormDefinitions returns active mobile form templates.
func (h *FieldOpsHandlers) ListFormDefinitions(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	forms, err := h.store.ListFormDefinitions(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list forms", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"forms": forms, "count": len(forms)})
}

// SyncPush processes offline batch submissions, GPS breadcrumbs, and task updates.
func (h *FieldOpsHandlers) SyncPush(c *gin.Context) {
	var req models.SyncPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sync push payload", "details": err.Error()})
		return
	}

	if req.WorkerID == "" || req.DeviceUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "worker_id and device_uuid are required"})
		return
	}

	resp, err := h.syncEng.HandlePushSync(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sync push failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SyncPull sends newly published forms, updated tasks, and schedule assignments.
func (h *FieldOpsHandlers) SyncPull(c *gin.Context) {
	var req models.SyncPullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sync pull payload", "details": err.Error()})
		return
	}

	if req.WorkerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "worker_id is required"})
		return
	}

	resp, err := h.syncEng.HandlePullSync(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sync pull failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// IngestGPSTrack streams GPS breadcrumb points and checks active geofence zones.
func (h *FieldOpsHandlers) IngestGPSTrack(c *gin.Context) {
	var req struct {
		WorkerID    string                 `json:"worker_id"`
		DeviceUUID  string                 `json:"device_uuid"`
		Breadcrumbs []models.GPSBreadcrumb `json:"breadcrumbs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GPS track payload", "details": err.Error()})
		return
	}

	if req.WorkerID == "" || len(req.Breadcrumbs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "worker_id and at least 1 breadcrumb are required"})
		return
	}

	for i := range req.Breadcrumbs {
		req.Breadcrumbs[i].WorkerID = req.WorkerID
		req.Breadcrumbs[i].DeviceUUID = req.DeviceUUID
	}

	if err := h.store.InsertGPSBreadcrumbs(c.Request.Context(), req.Breadcrumbs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store GPS breadcrumbs", "details": err.Error()})
		return
	}

	// Geofence check on last breadcrumb
	lastCrumb := req.Breadcrumbs[len(req.Breadcrumbs)-1]
	zones, _ := h.spatial.EvaluateGeofences(c.Request.Context(), "default", lastCrumb.Latitude, lastCrumb.Longitude)

	c.JSON(http.StatusAccepted, gin.H{
		"message":               "GPS breadcrumbs ingested",
		"points_stored":         len(req.Breadcrumbs),
		"active_geofence_zones": zones,
	})
}

// GetRecentBreadcrumbs returns worker travel path.
func (h *FieldOpsHandlers) GetRecentBreadcrumbs(c *gin.Context) {
	workerID := c.Param("worker_id")
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	crumbs, err := h.store.GetRecentBreadcrumbs(c.Request.Context(), workerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve breadcrumbs", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"worker_id": workerID, "breadcrumbs": crumbs, "count": len(crumbs)})
}

// CreateGeofence defines a spatial zone for mobile field operations.
func (h *FieldOpsHandlers) CreateGeofence(c *gin.Context) {
	var req models.GeofenceZone
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid geofence payload", "details": err.Error()})
		return
	}

	if req.Name == "" || req.RadiusMeters == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and radius_meters are required"})
		return
	}
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.TenantID == "" {
		req.TenantID = "default"
	}

	if err := h.store.CreateGeofence(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create geofence", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Geofence created", "geofence": req})
}

// ListGeofences lists active geofences.
func (h *FieldOpsHandlers) ListGeofences(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	zones, err := h.store.ListGeofences(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list geofences", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"geofences": zones, "count": len(zones)})
}

// ListConflicts lists sync conflict records.
func (h *FieldOpsHandlers) ListConflicts(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	status := c.Query("status")

	conflicts, err := h.store.ListConflicts(c.Request.Context(), tenantID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list conflicts", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"conflicts": conflicts, "count": len(conflicts)})
}

// ResolveConflict manually or programmatically resolves a data conflict.
func (h *FieldOpsHandlers) ResolveConflict(c *gin.Context) {
	conflictID := c.Param("id")
	var req struct {
		Strategy   models.ConflictResolutionStrategy `json:"strategy"`
		ResolvedBy string                             `json:"resolved_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resolution payload", "details": err.Error()})
		return
	}

	if req.Strategy == "" {
		req.Strategy = models.StrategyLastWriteWins
	}
	if req.ResolvedBy == "" {
		req.ResolvedBy = "supervisor"
	}

	resolved, err := h.syncEng.ResolveConflict(c.Request.Context(), conflictID, req.Strategy, req.ResolvedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve conflict", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conflict resolved", "conflict": resolved})
}
