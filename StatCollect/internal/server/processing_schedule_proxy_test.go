package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProcessingScheduleProxyForwardsAuthenticatedWorkspaceScope(t *testing.T) {
	previousConfig := cfg
	defer func() { cfg = previousConfig }()
	var forwarded bool
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/processing/schedules" || r.Method != http.MethodPost {
			t.Errorf("unexpected Core request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-StatGate-Internal-Key") != "core-secret" ||
			r.Header.Get("X-Tenant-ID") != "tenant-a" ||
			r.Header.Get("X-Workspace-ID") != "workspace-a" {
			t.Errorf("missing scoped service credentials")
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode Core request: %v", err)
		}
		if payload["dataset_id"] != "dataset-a" || payload["asset_id"] != "official-dataset-dataset-a" || payload["frequency"] != "daily" {
			t.Errorf("unexpected schedule payload: %+v", payload)
		}
		forwarded = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"schedule-a"}`))
	}))
	defer core.Close()
	cfg = &Config{AdminKeys: []string{"admin-key"}, StatisticsURL: core.URL, InternalKey: "core-secret"}

	req := httptest.NewRequest(http.MethodPost, "/admin/statistics/processing/schedules", strings.NewReader(`{"dataset_id":"dataset-a","frequency":"daily"}`))
	req.Header.Set("X-API-Key", "admin-key")
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Workspace-ID", "workspace-a")
	res := httptest.NewRecorder()
	adminProcessingSchedulesHandler(res, req)
	if res.Code != http.StatusCreated || !forwarded || !strings.Contains(res.Body.String(), "schedule-a") {
		t.Fatalf("proxy response = %d %q; forwarded=%v", res.Code, res.Body.String(), forwarded)
	}
}

func TestProcessingScheduleProxyRejectsUnauthenticatedMutation(t *testing.T) {
	previousConfig := cfg
	defer func() { cfg = previousConfig }()
	cfg = &Config{AdminKeys: []string{"admin-key"}, StatisticsURL: "http://127.0.0.1:1", InternalKey: "core-secret"}
	req := httptest.NewRequest(http.MethodPost, "/admin/statistics/processing/schedules", strings.NewReader(`{"dataset_id":"dataset-a","frequency":"daily"}`))
	res := httptest.NewRecorder()
	adminProcessingSchedulesHandler(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated mutation returned %d, want %d", res.Code, http.StatusUnauthorized)
	}
}
