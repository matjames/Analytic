package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"statgate/internal/abac"
	"statgate/internal/lakehouse"
	"statgate/internal/semantic"
)

// asInternalCall marks the request exactly as requireInternalKey does after a
// valid service-to-service key is presented. Unit tests for the ABAC layer
// drive this authenticated-intermediary path so identity resolution follows
// the same rules as production (SG-SEC-2026-08 fail-closed model).
func asInternalCall(req *http.Request) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), internalKeyContextKey{}, true))
}

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

	req := asInternalCall(httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil))
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

// TestRequireUserAccessFailsClosedWithoutIdentity encodes the SG-SEC-2026-08
// policy: a request that has neither a verified bearer JWT nor the internal
// service key gate produces an empty identity and MUST be denied access to
// protected data. This guards against future "helpful default" regressions.
func TestRequireUserAccessFailsClosedWithoutIdentity(t *testing.T) {
	server := &Server{abacEngine: abac.NewEngine()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req.Header.Set("X-Tenant-ID", "tenant-alpha")
	req.Header.Set("X-User-Role", "analyst")
	req.Header.Set("X-User-Clearance", "5")
	res := httptest.NewRecorder()

	_, ok := server.requireUserAccess(res, req, "telemetry", "read")
	if ok {
		t.Fatalf("expected request without verified identity to be denied (fail closed)")
	}
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unauthenticated access, got %d", res.Code)
	}
}

func TestHandleIndicatorsUsesTenantScopedRegistryData(t *testing.T) {
	server := &Server{
		abacEngine:    abac.NewEngine(),
		semRegistry:   semantic.NewRegistry(),
		storageEngine: lakehouse.NewStorageEngine(),
	}

	req := asInternalCall(httptest.NewRequest(http.MethodGet, "/api/v1/indicators?tenant_id=tenant-alpha", nil))
	req.Header.Set("X-User-Role", "analyst")
	req.Header.Set("X-User-Clearance", "2")
	res := httptest.NewRecorder()

	server.handleIndicators(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected indicators request to succeed with valid tenant access, got %d", res.Code)
	}
}
