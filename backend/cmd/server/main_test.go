package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"statgate/internal/abac"
	"statgate/internal/lakehouse"
	"statgate/internal/semantic"
)

func TestIsSafeIdentifier(t *testing.T) {
	valid := []string{"covid_19_data", "dataset1", "A"}
	invalid := []string{"", "1dataset", "dataset-name", "dataset/name", "dataset\"name"}

	for _, value := range valid {
		if !isSafeIdentifier(value) {
			t.Errorf("expected %q to be valid", value)
		}
	}
	for _, value := range invalid {
		if isSafeIdentifier(value) {
			t.Errorf("expected %q to be invalid", value)
		}
	}
}

func TestRequireInternalKey(t *testing.T) {
	server := &Server{internalAPIKey: "test-secret"}
	handler := server.requireInternalKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	unauthorized := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	unauthorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedResponse, unauthorized)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized request to return %d, got %d", http.StatusUnauthorized, unauthorizedResponse.Code)
	}

	authorized := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	authorized.Header.Set("X-StatGate-Internal-Key", "test-secret")
	authorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(authorizedResponse, authorized)
	if authorizedResponse.Code != http.StatusNoContent {
		t.Fatalf("expected authorized request to return %d, got %d", http.StatusNoContent, authorizedResponse.Code)
	}
}

func TestRequireUserAccessHonorsABACForReadOperations(t *testing.T) {
	server := &Server{abacEngine: abac.NewEngine()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req.Header.Set("X-Tenant-ID", "tenant-alpha")
	req.Header.Set("X-User-Role", "viewer")
	req.Header.Set("X-User-Clearance", "1")
	res := httptest.NewRecorder()

	_, ok := server.requireUserAccess(res, req, "telemetry", "read")
	if !ok {
		t.Fatalf("expected viewer with clearance 1 to be allowed to read telemetry")
	}

	_, ok = server.requireUserAccess(res, req, "telemetry", "write")
	if ok {
		t.Fatalf("expected viewer with clearance 1 to be denied from writing telemetry")
	}
}

func TestHandleIndicatorsUsesTenantScopedRegistryData(t *testing.T) {
	server := &Server{
		abacEngine:    abac.NewEngine(),
		semRegistry:   semantic.NewRegistry(),
		storageEngine: lakehouse.NewStorageEngine(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/indicators?tenant_id=tenant-alpha", nil)
	req.Header.Set("X-User-Role", "analyst")
	req.Header.Set("X-User-Clearance", "2")
	res := httptest.NewRecorder()

	server.handleIndicators(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected indicators request to succeed with valid tenant access, got %d", res.Code)
	}
}
