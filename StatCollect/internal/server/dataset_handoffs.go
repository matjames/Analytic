package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DatasetHandoffRequest struct {
	DatasetID         string     `json:"dataset_id"`
	Name              string     `json:"name,omitempty"`
	Description       string     `json:"description,omitempty"`
	FormID            string     `json:"form_id,omitempty"`
	ProcessingRunID   string     `json:"processing_run_id"`
	ScheduleFrequency string     `json:"schedule_frequency,omitempty"`
	NextRunAt         *time.Time `json:"next_run_at,omitempty"`
}

type DatasetHandoff struct {
	ID              int64                  `json:"id"`
	DatasetID       string                 `json:"dataset_id"`
	FormID          string                 `json:"form_id"`
	ProcessingRunID string                 `json:"processing_run_id"`
	TargetAssetID   string                 `json:"target_asset_id"`
	Status          string                 `json:"status"`
	HTTPStatus      int                    `json:"http_status"`
	Attempts        int                    `json:"attempts"`
	RecordCount     int                    `json:"record_count"`
	ScheduleID      string                 `json:"schedule_id,omitempty"`
	ScheduleStatus  string                 `json:"schedule_status,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	Lineage         map[string]interface{} `json:"lineage"`
	TenantID        string                 `json:"tenant_id"`
	WorkspaceID     string                 `json:"workspace_id"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

func validDatasetScheduleFrequency(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "hourly", "daily", "weekly", "monthly", "on_demand":
		return true
	default:
		return false
	}
}

func recordDatasetHandoff(handoff DatasetHandoff) (*DatasetHandoff, error) {
	if dbPool == nil {
		return &handoff, nil
	}
	lineage, err := json.Marshal(handoff.Lineage)
	if err != nil {
		return &handoff, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var stored DatasetHandoff
	var scheduleID, scheduleStatus, errorMessage sql.NullString
	err = dbPool.QueryRow(ctx, `
INSERT INTO analytics_dataset_handoffs
  (dataset_id, form_id, processing_run_id, target_asset_id, status, http_status, attempts,
   record_count, schedule_id, schedule_status, error_message, lineage, tenant_id, workspace_id, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,now(),now())
ON CONFLICT (dataset_id, processing_run_id, tenant_id, workspace_id) DO UPDATE SET
  form_id=EXCLUDED.form_id, target_asset_id=EXCLUDED.target_asset_id, status=EXCLUDED.status,
  http_status=EXCLUDED.http_status, attempts=analytics_dataset_handoffs.attempts+1,
  record_count=EXCLUDED.record_count, schedule_id=EXCLUDED.schedule_id,
  schedule_status=EXCLUDED.schedule_status, error_message=EXCLUDED.error_message,
  lineage=EXCLUDED.lineage, updated_at=now()
RETURNING id, dataset_id, form_id, processing_run_id, target_asset_id, status, http_status,
  attempts, record_count, schedule_id, schedule_status, error_message, lineage,
  tenant_id, workspace_id, created_at, updated_at`,
		handoff.DatasetID, handoff.FormID, handoff.ProcessingRunID, handoff.TargetAssetID,
		handoff.Status, handoff.HTTPStatus, handoff.Attempts, handoff.RecordCount,
		nullableString(handoff.ScheduleID), nullableString(handoff.ScheduleStatus), nullableString(handoff.ErrorMessage),
		lineage, handoff.TenantID, handoff.WorkspaceID).Scan(
		&stored.ID, &stored.DatasetID, &stored.FormID, &stored.ProcessingRunID, &stored.TargetAssetID,
		&stored.Status, &stored.HTTPStatus, &stored.Attempts, &stored.RecordCount, &scheduleID,
		&scheduleStatus, &errorMessage, &lineage, &stored.TenantID, &stored.WorkspaceID,
		&stored.CreatedAt, &stored.UpdatedAt)
	if err != nil {
		return &handoff, err
	}
	stored.ScheduleID, stored.ScheduleStatus, stored.ErrorMessage = scheduleID.String, scheduleStatus.String, errorMessage.String
	_ = json.Unmarshal(lineage, &stored.Lineage)
	return &stored, nil
}

func ListDatasetHandoffs(limit int, scopes ...OfficialStatisticsScope) ([]DatasetHandoff, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	scope := officialScopeOrDefault(scopes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `
SELECT id, dataset_id, form_id, processing_run_id, target_asset_id, status, http_status,
  attempts, record_count, schedule_id, schedule_status, error_message, lineage,
  tenant_id, workspace_id, created_at, updated_at
FROM analytics_dataset_handoffs WHERE tenant_id=$1 AND workspace_id=$2
ORDER BY updated_at DESC LIMIT $3`, scope.TenantID, scope.WorkspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DatasetHandoff
	for rows.Next() {
		var handoff DatasetHandoff
		var scheduleID, scheduleStatus, errorMessage sql.NullString
		var lineage []byte
		if err := rows.Scan(&handoff.ID, &handoff.DatasetID, &handoff.FormID, &handoff.ProcessingRunID,
			&handoff.TargetAssetID, &handoff.Status, &handoff.HTTPStatus, &handoff.Attempts,
			&handoff.RecordCount, &scheduleID, &scheduleStatus, &errorMessage, &lineage,
			&handoff.TenantID, &handoff.WorkspaceID, &handoff.CreatedAt, &handoff.UpdatedAt); err != nil {
			return nil, err
		}
		handoff.ScheduleID, handoff.ScheduleStatus, handoff.ErrorMessage = scheduleID.String, scheduleStatus.String, errorMessage.String
		_ = json.Unmarshal(lineage, &handoff.Lineage)
		out = append(out, handoff)
	}
	return out, rows.Err()
}

func datasetSchema(records []ProcessingRunRecord) map[string]string {
	schema := map[string]string{}
	for _, record := range records {
		for key, value := range record.Values {
			if _, exists := schema[key]; exists {
				continue
			}
			switch value.(type) {
			case bool:
				schema[key] = "boolean"
			case float64, float32, int, int64, json.Number:
				schema[key] = "number"
			case nil:
				schema[key] = "null"
			default:
				schema[key] = "string"
			}
		}
	}
	return schema
}

func handoffDatasetToAnalyticsCore(req DatasetHandoffRequest, actor string, scopes ...OfficialStatisticsScope) (*DatasetHandoff, error) {
	scope := officialScopeOrDefault(scopes)
	req.DatasetID = strings.TrimSpace(req.DatasetID)
	req.FormID = strings.TrimSpace(req.FormID)
	req.ProcessingRunID = strings.TrimSpace(req.ProcessingRunID)
	req.ScheduleFrequency = strings.ToLower(strings.TrimSpace(req.ScheduleFrequency))
	if !processingFieldNameValid(req.DatasetID) || req.ProcessingRunID == "" {
		return nil, fmt.Errorf("dataset_id and processing_run_id are required and must be valid")
	}
	if req.ScheduleFrequency != "" && !validDatasetScheduleFrequency(req.ScheduleFrequency) {
		return nil, fmt.Errorf("schedule_frequency must be hourly, daily, weekly, monthly, or on_demand")
	}
	run, err := GetProcessingRun(req.ProcessingRunID, scope)
	if err != nil || run.Status != "completed" {
		return nil, fmt.Errorf("a completed processing_run_id in the same tenant/workspace is required")
	}
	if req.FormID != "" && req.FormID != run.FormID {
		return nil, fmt.Errorf("form_id does not match the processing run")
	}
	records, err := GetProcessingRunRecords(req.ProcessingRunID, scope)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("processing run has no output records")
	}
	req.FormID = run.FormID
	if req.Name == "" {
		req.Name = req.DatasetID
	}
	if actor == "" {
		actor = "admin"
	}
	targetAssetID := "official-dataset-" + req.DatasetID
	lineage := map[string]interface{}{
		"source_module":            "statcollect",
		"source_form_id":           run.FormID,
		"processing_run_id":        run.ID,
		"processing_method":        run.Method,
		"processing_configuration": run.Configuration,
		"record_count":             len(records),
		"published_by":             actor,
	}
	lineage["schema"] = datasetSchema(records)
	rowPayload := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		rowPayload = append(rowPayload, map[string]interface{}{
			"submission_instance_id": record.SubmissionInstanceID,
			"weight":                 record.Weight,
			"values":                 record.Values,
		})
	}
	handoff := DatasetHandoff{
		DatasetID: req.DatasetID, FormID: run.FormID, ProcessingRunID: run.ID,
		TargetAssetID: targetAssetID, Status: "skipped", RecordCount: len(records),
		Lineage: lineage, TenantID: scope.TenantID, WorkspaceID: scope.WorkspaceID,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	statisticsURL, internalKey := "", ""
	if cfg != nil {
		statisticsURL = strings.TrimRight(strings.TrimSpace(cfg.StatisticsURL), "/")
		internalKey = cfg.InternalKey
	}
	if statisticsURL == "" || statisticsURL == "-" {
		handoff.ErrorMessage = "statistics_url_not_configured"
		stored, recordErr := recordDatasetHandoff(handoff)
		if recordErr == nil {
			return stored, nil
		}
		return &handoff, recordErr
	}
	payload := map[string]interface{}{
		"id":          targetAssetID,
		"asset_type":  "official_statistics_dataset",
		"version_tag": fmt.Sprintf("%s-%d", run.ID, time.Now().Unix()),
		"content_definition": map[string]interface{}{
			"dataset_id":        req.DatasetID,
			"name":              req.Name,
			"description":       req.Description,
			"form_id":           run.FormID,
			"processing_run_id": run.ID,
			"record_count":      len(records),
			"schema":            datasetSchema(records),
			"records":           rowPayload,
			"lineage":           lineage,
		},
	}
	body, err := json.Marshal(payload)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		httpRequest, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, statisticsURL+"/api/v1/assets/save", strings.NewReader(string(body)))
		if requestErr == nil {
			httpRequest.Header.Set("Content-Type", "application/json")
			httpRequest.Header.Set("X-StatGate-Internal-Key", internalKey)
			httpRequest.Header.Set("X-Tenant-ID", scope.TenantID)
			httpRequest.Header.Set("X-Workspace-ID", scope.WorkspaceID)
			httpRequest.Header.Set("X-User-ID", actor)
			httpRequest.Header.Set("X-User-Role", "operator")
			if response, responseErr := http.DefaultClient.Do(httpRequest); responseErr == nil {
				handoff.HTTPStatus = response.StatusCode
				handoff.Attempts = 1
				if response.StatusCode >= 200 && response.StatusCode < 300 {
					handoff.Status = "published"
				} else {
					handoff.Status = "failed"
					handoff.ErrorMessage = "analytics_core_rejected_dataset_handoff"
				}
				response.Body.Close()
			} else {
				handoff.Status = "failed"
				handoff.Attempts = 1
				handoff.ErrorMessage = responseErr.Error()
			}
		} else {
			handoff.Status = "failed"
			handoff.ErrorMessage = requestErr.Error()
		}
	} else {
		handoff.Status = "failed"
		handoff.ErrorMessage = err.Error()
	}
	if handoff.Status == "published" && req.ScheduleFrequency != "" {
		schedulePayload := map[string]interface{}{
			"dataset_id": req.DatasetID, "asset_id": targetAssetID,
			"frequency": req.ScheduleFrequency, "next_run_at": req.NextRunAt,
		}
		scheduleBody, marshalErr := json.Marshal(schedulePayload)
		if marshalErr == nil {
			scheduleRequest, requestErr := http.NewRequestWithContext(context.Background(), http.MethodPost, statisticsURL+"/api/v1/processing/schedules", strings.NewReader(string(scheduleBody)))
			if requestErr == nil {
				scheduleRequest.Header.Set("Content-Type", "application/json")
				scheduleRequest.Header.Set("X-StatGate-Internal-Key", internalKey)
				scheduleRequest.Header.Set("X-Tenant-ID", scope.TenantID)
				scheduleRequest.Header.Set("X-Workspace-ID", scope.WorkspaceID)
				scheduleRequest.Header.Set("X-User-ID", actor)
				scheduleRequest.Header.Set("X-User-Role", "operator")
				if response, responseErr := http.DefaultClient.Do(scheduleRequest); responseErr == nil {
					if response.StatusCode >= 200 && response.StatusCode < 300 {
						var schedule struct {
							ID string `json:"id"`
						}
						_ = json.NewDecoder(response.Body).Decode(&schedule)
						handoff.ScheduleID, handoff.ScheduleStatus = schedule.ID, "registered"
					} else {
						handoff.ScheduleStatus = "failed"
						handoff.ErrorMessage = "analytics_core_rejected_processing_schedule"
					}
					response.Body.Close()
				} else {
					handoff.ScheduleStatus = "failed"
					handoff.ErrorMessage = responseErr.Error()
				}
			} else {
				handoff.ScheduleStatus = "failed"
			}
		}
	} else if handoff.Status == "published" {
		handoff.ScheduleStatus = "not_requested"
	}
	stored, recordErr := recordDatasetHandoff(handoff)
	if recordErr == nil {
		return stored, nil
	}
	return &handoff, recordErr
}

func adminDatasetHandoffHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req DatasetHandoffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	handoff, err := handoffDatasetToAnalyticsCore(req, strings.TrimSpace(r.Header.Get("X-Actor-ID")), scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if handoff.Status == "failed" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(handoff)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(handoff)
}

func internalDatasetRefreshHandler(w http.ResponseWriter, r *http.Request) {
	if !checkInternalKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	scope, err := officialStatisticsScope(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req struct {
		DatasetID string `json:"dataset_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	req.DatasetID = strings.TrimSpace(req.DatasetID)
	if !processingFieldNameValid(req.DatasetID) {
		http.Error(w, "dataset_id is required and must be valid", http.StatusBadRequest)
		return
	}
	if dbPool == nil {
		http.Error(w, "database not initialized", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var formID, lastPublishedRunID string
	err = dbPool.QueryRow(ctx, `SELECT form_id, processing_run_id FROM analytics_dataset_handoffs
WHERE dataset_id=$1 AND tenant_id=$2 AND workspace_id=$3 AND status='published'
	ORDER BY updated_at DESC LIMIT 1`, req.DatasetID, scope.TenantID, scope.WorkspaceID).Scan(&formID, &lastPublishedRunID)
	if err != nil {
		http.Error(w, "dataset handoff not found in this workspace", http.StatusNotFound)
		return
	}
	runs, err := ListProcessingRuns(formID, "completed", 1, scope)
	if err != nil {
		http.Error(w, "could not load latest completed processing run", http.StatusInternalServerError)
		return
	}
	if len(runs) == 0 {
		http.Error(w, "no completed processing run is available for the dataset form", http.StatusConflict)
		return
	}
	if runs[0].ID == lastPublishedRunID {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "no_change", "processing_run_id": runs[0].ID})
		return
	}
	handoff, err := handoffDatasetToAnalyticsCore(DatasetHandoffRequest{
		DatasetID: req.DatasetID, FormID: formID, ProcessingRunID: runs[0].ID,
	}, "statgate-core-scheduler", scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if handoff.Status != "published" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "failed", "handoff": handoff})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "refreshed", "handoff": handoff})
}

func adminDatasetHandoffsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	handoffs, err := ListDatasetHandoffs(limit, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"count": len(handoffs), "handoffs": handoffs})
}
