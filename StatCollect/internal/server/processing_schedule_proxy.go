package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type scheduleProxyRequest struct {
	DatasetID string     `json:"dataset_id"`
	AssetID   string     `json:"asset_id,omitempty"`
	Frequency string     `json:"frequency"`
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
	ScheduleID string    `json:"schedule_id,omitempty"`
}

func adminProcessingSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		scope, ok := requireOfficialStatisticsReadScope(w, r)
		if !ok {
			return
		}
		proxyAnalyticsCore(w, r, scope, "/api/v1/processing/schedules", nil)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	var req scheduleProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	req.DatasetID = strings.TrimSpace(req.DatasetID)
	req.AssetID = strings.TrimSpace(req.AssetID)
	req.Frequency = strings.ToLower(strings.TrimSpace(req.Frequency))
	if !processingFieldNameValid(req.DatasetID) || !validDatasetScheduleFrequency(req.Frequency) {
		http.Error(w, "dataset_id and a valid frequency are required", http.StatusBadRequest)
		return
	}
	if req.AssetID == "" {
		req.AssetID = "official-dataset-" + req.DatasetID
	}
	if !processingFieldNameValid(req.AssetID) {
		http.Error(w, "asset_id is invalid", http.StatusBadRequest)
		return
	}
	body, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "could not encode schedule", http.StatusInternalServerError)
		return
	}
	proxyAnalyticsCore(w, r, scope, "/api/v1/processing/schedules", body)
}

func adminProcessingScheduleRunsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	scope, ok := requireOfficialStatisticsReadScope(w, r)
	if !ok {
		return
	}
	query := url.Values{}
	if scheduleID := strings.TrimSpace(r.URL.Query().Get("schedule_id")); scheduleID != "" {
		if len(scheduleID) > 128 {
			http.Error(w, "schedule_id is invalid", http.StatusBadRequest)
			return
		}
		query.Set("schedule_id", scheduleID)
	}
	if limit := strings.TrimSpace(r.URL.Query().Get("limit")); limit != "" {
		query.Set("limit", limit)
	}
	path := "/api/v1/processing/schedules/runs"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	proxyAnalyticsCore(w, r, scope, path, nil)
}

func adminRunProcessingScheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	var req scheduleProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	req.ScheduleID = strings.TrimSpace(req.ScheduleID)
	if req.ScheduleID == "" || len(req.ScheduleID) > 128 {
		http.Error(w, "schedule_id is required", http.StatusBadRequest)
		return
	}
	body, _ := json.Marshal(map[string]string{"schedule_id": req.ScheduleID})
	proxyAnalyticsCore(w, r, scope, "/api/v1/processing/schedules/run", body)
}

func proxyAnalyticsCore(w http.ResponseWriter, r *http.Request, scope OfficialStatisticsScope, path string, body []byte) {
	if cfg == nil || strings.TrimSpace(cfg.StatisticsURL) == "" || cfg.StatisticsURL == "-" || strings.TrimSpace(cfg.InternalKey) == "" {
		http.Error(w, "Analytics Core integration is not configured", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	target := strings.TrimRight(strings.TrimSpace(cfg.StatisticsURL), "/") + path
	req, err := http.NewRequestWithContext(ctx, r.Method, target, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "could not create Analytics Core request", http.StatusBadGateway)
		return
	}
	req.Header.Set("X-StatGate-Internal-Key", cfg.InternalKey)
	req.Header.Set("X-Tenant-ID", scope.TenantID)
	req.Header.Set("X-Workspace-ID", scope.WorkspaceID)
	req.Header.Set("X-User-ID", "statcollect-official-statistics")
	req.Header.Set("X-User-Role", "operator")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analytics Core request failed: %s", err.Error()), http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.WriteHeader(response.StatusCode)
	if _, err := io.Copy(w, io.LimitReader(response.Body, 2<<20)); err != nil {
		return
	}
}
