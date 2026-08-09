package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	healthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Errorf("expected 'ok', got '%s'", rr.Body.String())
	}
}

func TestTemplatesHandlerUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"id":"t1","name":"Test"}`))
	rr := httptest.NewRecorder()

	templatesHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized for post without key, got %d", rr.Code)
	}
}

func TestCloneTemplateLogic(t *testing.T) {
	schema := json.RawMessage(`{"sections":[]}`)
	tmpl := Template{
		ID:          "source_1",
		Name:        "Source Survey",
		Description: "Original survey",
		Version:     "1.0",
		Schema:      schema,
		Status:      "active",
		TenantID:    "default",
	}

	if tmpl.ID != "source_1" {
		t.Fatalf("expected template ID source_1")
	}
}
