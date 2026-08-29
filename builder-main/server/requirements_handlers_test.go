package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func initRequirementsTestDB(t *testing.T) {
	t.Helper()

	CloseFeedbackDB()
	CloseRequirementsDB()

	tmpDir := t.TempDir()
	t.Setenv("FEEDBACK_DB_PATH", filepath.Join(tmpDir, "feedback-test.db"))
	if err := InitFeedbackDB(); err != nil {
		t.Fatalf("InitFeedbackDB failed: %v", err)
	}
	t.Setenv("REQUIREMENTS_DB_PATH", filepath.Join(tmpDir, "requirements-test.db"))
	if err := InitRequirementsDB(); err != nil {
		t.Fatalf("InitRequirementsDB failed: %v", err)
	}
	t.Cleanup(func() {
		CloseFeedbackDB()
		CloseRequirementsDB()
		_ = os.Unsetenv("FEEDBACK_DB_PATH")
		_ = os.Unsetenv("REQUIREMENTS_DB_PATH")
	})
}

func TestRequirementsSpecHandler_PostSuccess(t *testing.T) {
	initRequirementsTestDB(t)

	payload := map[string]interface{}{
		"documentControl": map[string]interface{}{
			"reportName": "District Performance Report",
			"reportId":   "DPR-01",
		},
		"reportOverview": map[string]interface{}{
			"businessObjective": "Track district-level outcomes.",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/requirements-specs", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	RequirementsSpecHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["id"] == nil {
		t.Fatalf("expected id in response, got %+v", resp)
	}

	var count int
	if err := requirementsDB.QueryRow(`SELECT COUNT(*) FROM requirements_specs WHERE report_name = ?`, "District Performance Report").Scan(&count); err != nil {
		t.Fatalf("query inserted row: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 inserted row, got %d", count)
	}
}

func TestRequirementsSpecHandler_Validation(t *testing.T) {
	initRequirementsTestDB(t)

	payload := map[string]interface{}{
		"documentControl": map[string]interface{}{},
		"reportOverview": map[string]interface{}{
			"businessObjective": "Some objective",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/requirements-specs", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	RequirementsSpecHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
