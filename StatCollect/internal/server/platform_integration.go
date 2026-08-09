package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Directive 22 — Unified Platform Integration
// StatCollect is the primary gateway through which enterprise data enters
// StatGate. Once data is collected and validated, it flows automatically
// through the platform — to collaboration, analysis, visualization, reporting,
// AI, and decision-making — without requiring users to manually transfer data.
// ─────────────────────────────────────────────────────────────────────────────

// PlatformPublishEvent represents a cross-module data publication event.
type PlatformPublishEvent struct {
	EventType    string                 `json:"event_type"`
	SourceModule string                 `json:"source_module"`
	SourceType   string                 `json:"source_type"`
	SourceID     string                 `json:"source_id"`
	TenantID     string                 `json:"tenant_id"`
	Payload      map[string]interface{} `json:"payload"`
	PublishedAt  time.Time              `json:"published_at"`
}

// PlatformLink represents a cross-module object linkage.
type PlatformLink struct {
	ID           int64                  `json:"id"`
	SourceModule string                 `json:"source_module"`
	SourceType   string                 `json:"source_type"`
	SourceID     string                 `json:"source_id"`
	TargetModule string                 `json:"target_module"`
	TargetType   string                 `json:"target_type"`
	TargetID     string                 `json:"target_id"`
	Relationship string                 `json:"relationship"`
	Metadata     map[string]interface{} `json:"metadata"`
	TenantID     string                 `json:"tenant_id"`
	CreatedAt    time.Time              `json:"created_at"`
}

// WorkflowTask represents an auto-generated workflow task from a survey event.
type WorkflowTask struct {
	ID           int64      `json:"id"`
	TaskType     string     `json:"task_type"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	AssignedTo   string     `json:"assigned_to"`
	AssignedRole string     `json:"assigned_role"`
	SourceModule string     `json:"source_module"`
	SourceType   string     `json:"source_type"`
	SourceID     string     `json:"source_id"`
	Priority     string     `json:"priority"`
	Status       string     `json:"status"`
	DueAt        *time.Time `json:"due_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	TenantID     string     `json:"tenant_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// GISSyncRecord represents a record in the GIS synchronization queue.
type GISSyncRecord struct {
	ID                  int64                  `json:"id"`
	SubmissionInstanceID string                 `json:"submission_instance_id"`
	FormID              string                 `json:"form_id"`
	GISLayerID          string                 `json:"gis_layer_id,omitempty"`
	Latitude            float64                `json:"latitude"`
	Longitude           float64                `json:"longitude"`
	Accuracy            float32                `json:"accuracy"`
	FeatureProperties   map[string]interface{} `json:"feature_properties"`
	SyncStatus          string                 `json:"sync_status"`
	SyncAttempt         int                    `json:"sync_attempt"`
	SyncedAt            *time.Time             `json:"synced_at,omitempty"`
	ErrorMessage        string                 `json:"error_message,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
}

// StatsExport represents a submission's export status to the Statistics module.
type StatsExport struct {
	ID                  int64      `json:"id"`
	SubmissionInstanceID string    `json:"submission_instance_id"`
	FormID              string     `json:"form_id"`
	DatasetID           string     `json:"dataset_id,omitempty"`
	ExportStatus        string     `json:"export_status"`
	ExportAttempt       int        `json:"export_attempt"`
	ExportedAt          *time.Time `json:"exported_at,omitempty"`
	ErrorMessage        string     `json:"error_message,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// PublishSubmissionEvent fires all cross-module integrations for a submission
// event. It is called whenever a submission is received, approved, or rejected.
// ─────────────────────────────────────────────────────────────────────────────

// PublishSubmissionEvent fires cross-module integration for a submission lifecycle event.
// eventType: "submission.received", "submission.approved", "submission.rejected"
func PublishSubmissionEvent(eventType, instanceID, formID, tenantID string, meta map[string]interface{}) {
	// Load header config to know which modules to publish to
	// (use defaults if no specific config is found)
	sh := DefaultSurveyHeader(formID)
	if dbPool != nil {
		if loaded, err := GetSurveyHeader(formID); err == nil {
			sh = *loaded
		}
	}

	evt := PlatformPublishEvent{
		EventType:    eventType,
		SourceModule: "statcollect",
		SourceType:   "submission",
		SourceID:     instanceID,
		TenantID:     tenantID,
		Payload: map[string]interface{}{
			"instance_id": instanceID,
			"form_id":     formID,
			"meta":        meta,
		},
		PublishedAt: time.Now(),
	}

	// Fire integrations in background to not block the submission handler
	go func(ev PlatformPublishEvent, header SurveyHeader) {
		if header.PublishToResearch {
			publishToModule(ev, "research", "dataset", cfg.ResearchURL)
		}
		if header.PublishToProjects {
			publishToModule(ev, "projects", "project_data", cfg.ProjectsURL)
		}
		if header.PublishToStatistics && eventType == "submission.approved" {
			publishToModule(ev, "statistics", "dataset", cfg.StatisticsURL)
			_ = queueStatsExport(instanceID, formID, tenantID)
		}
		if header.PublishToGIS && eventType == "submission.approved" {
			_ = queueGISSync(instanceID, formID, tenantID, meta)
		}
		if header.PublishToReporting && eventType == "submission.approved" {
			publishToModule(ev, "reporting", "report_data", cfg.ReportingURL)
		}
		if header.AutoCreateTasks {
			_ = autoCreateWorkflowTasks(eventType, instanceID, formID, tenantID)
		}
		// Always log the event
		_ = LogEvent(eventType, "statcollect", "submission", instanceID, ev.Payload)
		// Broadcast to SSE live dashboard stream
		BroadcastAnalyticsEvent(eventType, ev.Payload)
		// Always notify via StatChat if enabled
		if statChat != nil && statChat.Enabled {
			msg := formatStatChatNotification(eventType, instanceID, formID, meta)
			_ = statChat.PostMessageToObject("submission", instanceID, "system", msg)
		}
		// Queue notification for supervisors
		_ = SaveNotification(Notification{
			UserID:  "supervisor",
			Channel: "app",
			Message: fmt.Sprintf("📊 Survey event [%s]: %s (Form: %s)", eventType, instanceID, formID),
			Status:  "Pending",
		})
	}(evt, sh)
}

// publishToModule sends a cross-module event to a StatGate module's webhook endpoint.
// If the module URL is not configured, the event is logged but not sent externally.
func publishToModule(evt PlatformPublishEvent, targetModule, targetType, moduleURL string) {
	status := 0
	success := false
	errMsg := ""

	if moduleURL != "" && moduleURL != "-" {
		payload, _ := json.Marshal(evt)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, moduleURL+"/api/statcollect/ingest", strings.NewReader(string(payload)))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Source-Module", "statcollect")
			req.Header.Set("X-Event-Type", evt.EventType)
			if cfg != nil && cfg.InternalKey != "" {
				req.Header.Set("X-Internal-Key", cfg.InternalKey)
			}
			if resp, err := http.DefaultClient.Do(req); err == nil {
				status = resp.StatusCode
				success = resp.StatusCode < 300
				resp.Body.Close()
			} else {
				errMsg = err.Error()
			}
		} else {
			errMsg = err.Error()
		}
	} else {
		// Module URL not configured — record as skipped (not an error)
		success = true
		errMsg = "module_url_not_configured"
	}

	// Log the publish attempt
	_ = logPlatformPublish(evt.EventType, evt.SourceID, targetModule, moduleURL, status, success, errMsg, evt.Payload)
	if !success && moduleURL != "" && moduleURL != "-" {
		log.Printf("platform_integration: failed to publish %s to %s: %s", evt.EventType, targetModule, errMsg)
	}
}

// logPlatformPublish records a cross-module publish attempt to the audit log.
func logPlatformPublish(eventType, sourceID, targetModule, endpoint string, httpStatus int, success bool, errMsg string, payload map[string]interface{}) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ps, _ := json.Marshal(payload)
	_, err := dbPool.Exec(ctx, `INSERT INTO platform_publish_log
		(event_type, source_id, target_module, target_endpoint, http_status, success, error_message, payload_summary)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		eventType, sourceID, targetModule, endpoint, httpStatus, success, errMsg, ps)
	return err
}

// queueStatsExport adds a submission to the statistics export queue.
func queueStatsExport(instanceID, formID, tenantID string) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO stats_exports
		(submission_instance_id, form_id, export_status, tenant_id)
		VALUES ($1,$2,'pending',$3)
		ON CONFLICT (submission_instance_id) DO UPDATE SET export_status='pending', export_attempt=stats_exports.export_attempt+1`,
		instanceID, formID, tenantID)
	return err
}

// queueGISSync adds a submission to the GIS sync queue, extracting GPS data from meta.
func queueGISSync(instanceID, formID, tenantID string, meta map[string]interface{}) error {
	if dbPool == nil {
		return nil
	}
	lat, lng, acc := extractGPSFromMeta(meta)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fp, _ := json.Marshal(meta)
	_, err := dbPool.Exec(ctx, `INSERT INTO gis_sync_queue
		(submission_instance_id, form_id, latitude, longitude, accuracy, feature_properties, sync_status, tenant_id)
		VALUES ($1,$2,$3,$4,$5,$6,'pending',$7)
		ON CONFLICT (submission_instance_id) DO UPDATE SET sync_status='pending', sync_attempt=gis_sync_queue.sync_attempt+1`,
		instanceID, formID, lat, lng, acc, fp, tenantID)
	return err
}

// extractGPSFromMeta extracts GPS coordinates from a submission metadata map.
func extractGPSFromMeta(meta map[string]interface{}) (lat, lng float64, acc float32) {
	if meta == nil {
		return
	}
	// Try _msh_gps field first (MSH standard field)
	if gps, ok := meta["_msh_gps"]; ok {
		if gpsStr, ok := gps.(string); ok {
			fmt.Sscanf(gpsStr, "%f %f", &lat, &lng)
		}
	}
	// Try generic gps_location field
	if lat == 0 {
		if gps, ok := meta["gps_location"]; ok {
			if gpsStr, ok := gps.(string); ok {
				fmt.Sscanf(gpsStr, "%f %f %f", &lat, &lng, &acc)
			}
		}
	}
	return
}

// autoCreateWorkflowTasks generates workflow tasks based on survey lifecycle events.
func autoCreateWorkflowTasks(eventType, instanceID, formID, tenantID string) error {
	if dbPool == nil {
		return nil
	}
	var tasks []WorkflowTask
	now := time.Now()
	due := now.Add(24 * time.Hour)

	switch eventType {
	case "submission.received":
		tasks = []WorkflowTask{
			{
				TaskType:     "review_submission",
				Title:        fmt.Sprintf("Review submission %s", instanceID),
				Description:  fmt.Sprintf("New data collection submission received for form %s. Review for completeness and quality.", formID),
				AssignedRole: "supervisor",
				SourceType:   "submission",
				SourceID:     instanceID,
				Priority:     "normal",
				Status:       "open",
				DueAt:        &due,
				TenantID:     tenantID,
			},
		}
	case "submission.approved":
		tasks = []WorkflowTask{
			{
				TaskType:     "validate_gps",
				Title:        fmt.Sprintf("Validate GPS for %s", instanceID),
				Description:  "Verify GPS coordinates are within the expected geographic boundary.",
				AssignedRole: "gis_analyst",
				SourceType:   "submission",
				SourceID:     instanceID,
				Priority:     "low",
				Status:       "open",
				TenantID:     tenantID,
			},
		}
	case "submission.rejected":
		tasks = []WorkflowTask{
			{
				TaskType:     "reassign_visit",
				Title:        fmt.Sprintf("Reassign rejected submission %s", instanceID),
				Description:  "Submission was rejected. Review rejection reason and reassign field visit if necessary.",
				AssignedRole: "field_coordinator",
				SourceType:   "submission",
				SourceID:     instanceID,
				Priority:     "high",
				Status:       "open",
				DueAt:        &due,
				TenantID:     tenantID,
			},
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, t := range tasks {
		_, err := dbPool.Exec(ctx, `INSERT INTO workflow_tasks
			(task_type, title, description, assigned_role, source_module, source_type, source_id, priority, status, due_at, tenant_id)
			VALUES ($1,$2,$3,$4,'statcollect',$5,$6,$7,$8,$9,$10)`,
			t.TaskType, t.Title, t.Description, t.AssignedRole, t.SourceType, t.SourceID, t.Priority, t.Status, t.DueAt, t.TenantID)
		if err != nil {
			log.Printf("workflow_task create error: %v", err)
		}
	}
	return nil
}

// SavePlatformLink creates a cross-module object link in the database.
func SavePlatformLink(link PlatformLink) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	md, _ := json.Marshal(link.Metadata)
	_, err := dbPool.Exec(ctx, `INSERT INTO platform_links
		(source_module, source_type, source_id, target_module, target_type, target_id, relationship, metadata, tenant_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT DO NOTHING`,
		link.SourceModule, link.SourceType, link.SourceID,
		link.TargetModule, link.TargetType, link.TargetID,
		link.Relationship, md, link.TenantID)
	return err
}

// GetPlatformLinks returns all cross-module links for a given source object.
func GetPlatformLinks(sourceModule, sourceType, sourceID string) ([]PlatformLink, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT id, source_module, source_type, source_id,
		target_module, target_type, target_id, relationship, metadata, tenant_id, created_at
		FROM platform_links WHERE source_module=$1 AND source_type=$2 AND source_id=$3`,
		sourceModule, sourceType, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var links []PlatformLink
	for rows.Next() {
		var l PlatformLink
		var md []byte
		if err := rows.Scan(&l.ID, &l.SourceModule, &l.SourceType, &l.SourceID,
			&l.TargetModule, &l.TargetType, &l.TargetID, &l.Relationship, &md, &l.TenantID, &l.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(md, &l.Metadata)
		links = append(links, l)
	}
	return links, nil
}

// ListWorkflowTasks returns open workflow tasks, optionally filtered by source.
func ListWorkflowTasks(sourceID, tenantID string) ([]WorkflowTask, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT id, task_type, title, COALESCE(description,''), COALESCE(assigned_to,''), COALESCE(assigned_role,''),
		source_module, source_type, source_id, priority, status, due_at, completed_at, tenant_id, created_at, updated_at
		FROM workflow_tasks WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if sourceID != "" {
		query += " AND source_id=$2"
		args = append(args, sourceID)
	}
	query += " ORDER BY created_at DESC LIMIT 100"
	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []WorkflowTask
	for rows.Next() {
		var t WorkflowTask
		if err := rows.Scan(&t.ID, &t.TaskType, &t.Title, &t.Description,
			&t.AssignedTo, &t.AssignedRole, &t.SourceModule, &t.SourceType, &t.SourceID,
			&t.Priority, &t.Status, &t.DueAt, &t.CompletedAt, &t.TenantID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// ListGISSyncQueue returns records in the GIS sync queue.
func ListGISSyncQueue(tenantID string) ([]GISSyncRecord, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT id, submission_instance_id, form_id,
		COALESCE(gis_layer_id,''), COALESCE(latitude,0), COALESCE(longitude,0), COALESCE(accuracy,0),
		feature_properties, sync_status, sync_attempt, synced_at, COALESCE(error_message,''), created_at
		FROM gis_sync_queue WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT 200`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []GISSyncRecord
	for rows.Next() {
		var r GISSyncRecord
		var fp []byte
		if err := rows.Scan(&r.ID, &r.SubmissionInstanceID, &r.FormID,
			&r.GISLayerID, &r.Latitude, &r.Longitude, &r.Accuracy,
			&fp, &r.SyncStatus, &r.SyncAttempt, &r.SyncedAt, &r.ErrorMessage, &r.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(fp, &r.FeatureProperties)
		records = append(records, r)
	}
	return records, nil
}

// ListStatsExports returns submission statistics export records.
func ListStatsExports(tenantID string) ([]StatsExport, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT id, submission_instance_id, form_id,
		COALESCE(dataset_id,''), export_status, export_attempt, exported_at, COALESCE(error_message,''), created_at
		FROM stats_exports WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT 200`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var exports []StatsExport
	for rows.Next() {
		var e StatsExport
		if err := rows.Scan(&e.ID, &e.SubmissionInstanceID, &e.FormID,
			&e.DatasetID, &e.ExportStatus, &e.ExportAttempt, &e.ExportedAt, &e.ErrorMessage, &e.CreatedAt); err != nil {
			return nil, err
		}
		exports = append(exports, e)
	}
	return exports, nil
}

// UpdateWorkflowTaskStatus updates the status of a workflow task.
func UpdateWorkflowTaskStatus(taskID int64, status string) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var completedAt interface{}
	if status == "completed" || status == "cancelled" {
		now := time.Now()
		completedAt = now
	}
	_, err := dbPool.Exec(ctx,
		`UPDATE workflow_tasks SET status=$1, completed_at=$2, updated_at=now() WHERE id=$3`,
		status, completedAt, taskID)
	return err
}

// formatStatChatNotification builds a StatChat message for a survey lifecycle event.
func formatStatChatNotification(eventType, instanceID, formID string, meta map[string]interface{}) string {
	icons := map[string]string{
		"submission.received": "📥",
		"submission.approved": "✅",
		"submission.rejected": "❌",
	}
	icon := icons[eventType]
	if icon == "" {
		icon = "📊"
	}
	enumerator := ""
	if e, ok := meta["_msh_enumerator_name"]; ok {
		enumerator = fmt.Sprintf(" by %v", e)
	}
	return fmt.Sprintf("%s **%s** — Submission `%s`%s (Form: `%s`)", icon, strings.ToUpper(eventType), instanceID, enumerator, formID)
}
