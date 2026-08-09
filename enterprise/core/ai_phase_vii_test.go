package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func aiRequest(t *testing.T, method, path, body, user, tenant string) *httptest.ResponseRecorder {
	t.Helper()
	r := setupTestRouter().engine
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if user != "" {
		req.Header.Set("X-User-ID", user)
	}
	if tenant != "" {
		req.Header.Set("X-Tenant-ID", tenant)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAIQueryRequiresIdentity(t *testing.T) {
	w := aiRequest(t, "POST", "/api/ai/v1/query", `{}`, "", "tenant-ai-test")
	if w.Code != 401 {
		t.Fatalf("expected unauthenticated request to be rejected, got %d", w.Code)
	}
}

func TestAIQueryReturnsOnlyTenantEvidence(t *testing.T) {
	ingestEnterpriseRecord(EnterpriseRecord{SourceApp: "statcollect", SourceEntity: "submission", SourceID: "ai-visible", TenantID: "tenant-ai-test", EventType: "submission.received", Timestamp: nowUTC(), Metadata: map[string]interface{}{}})
	ingestEnterpriseRecord(EnterpriseRecord{SourceApp: "statcollect", SourceEntity: "submission", SourceID: "ai-hidden", TenantID: "other-tenant", EventType: "submission.received", Timestamp: nowUTC(), Metadata: map[string]interface{}{}})
	w := aiRequest(t, "POST", "/api/ai/v1/query", `{"question":"What happened?"}`, "phase-vii-viewer", "tenant-ai-test")
	if w.Code != 200 {
		t.Fatalf("expected query success, got %d: %s", w.Code, w.Body.String())
	}
	var response AIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "grounded" {
		t.Fatalf("expected grounded response, got %s", response.Status)
	}
	for _, source := range response.Sources {
		if source.TenantID != "tenant-ai-test" {
			t.Fatalf("cross-tenant source returned: %+v", source)
		}
	}
}

func TestAIQueryHonorsApplicationAndObjectContext(t *testing.T) {
	t.Setenv("PMS_UI_URL", "http://pms.test")
	ingestEnterpriseRecord(EnterpriseRecord{SourceApp: "pms", SourceEntity: "project", SourceID: "contextual-project", TenantID: "context-tenant", EventType: "project.updated", Timestamp: nowUTC(), Metadata: map[string]interface{}{}})
	ingestEnterpriseRecord(EnterpriseRecord{SourceApp: "pms", SourceEntity: "project", SourceID: "other-project", TenantID: "context-tenant", EventType: "project.updated", Timestamp: nowUTC(), Metadata: map[string]interface{}{}})
	w := aiRequest(t, "POST", "/api/ai/v1/query", `{"question":"What changed?","application":"pms","object_type":"project","object_id":"contextual-project"}`, "context-user", "context-tenant")
	if w.Code != 200 {
		t.Fatalf("expected contextual query success, got %d: %s", w.Code, w.Body.String())
	}
	var response AIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Sources) != 1 || response.Sources[0].RecordID != "contextual-project" {
		t.Fatalf("context did not restrict evidence: %+v", response.Sources)
	}
	if response.Sources[0].DeepLink == "" {
		t.Fatal("expected configured source deep link")
	}
}

func TestAIQueryReportsInsufficientEvidence(t *testing.T) {
	w := aiRequest(t, "POST", "/api/ai/v1/query", `{"question":"What is unknown?","organization_id":"no-records"}`, "phase-vii-viewer", "empty-ai-tenant")
	if w.Code != 200 {
		t.Fatalf("expected query success, got %d", w.Code)
	}
	var response AIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	if response.Status != "insufficient_evidence" {
		t.Fatalf("expected insufficient evidence, got %s", response.Status)
	}
}

func TestAcceptedHumanInvestigationCreatesAControlledTask(t *testing.T) {
	t.Setenv("STATGATE_ADMIN_ROLE_phase-vii-manager", "manager")
	ingestEnterpriseRecord(EnterpriseRecord{SourceApp: "statcollect", SourceEntity: "submission", SourceID: "ai-investigation-record", TenantID: "investigation-tenant", EventType: "submission.received", Timestamp: nowUTC(), Metadata: map[string]interface{}{}})
	query := aiRequest(t, "POST", "/api/ai/v1/query", `{"question":"Investigate a reporting change"}`, "phase-vii-manager", "investigation-tenant")
	if query.Code != 200 {
		t.Fatalf("expected query success, got %d: %s", query.Code, query.Body.String())
	}
	var response AIResponse
	if err := json.Unmarshal(query.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	create := aiRequest(t, "POST", "/api/ai/v1/investigations", `{"title":"Reporting review","question":"Investigate a reporting change","response_id":"`+response.ID+`"}`, "phase-vii-manager", "investigation-tenant")
	if create.Code != 201 {
		t.Fatalf("expected investigation creation, got %d: %s", create.Code, create.Body.String())
	}
	var investigation AIInvestigation
	if err := json.Unmarshal(create.Body.Bytes(), &investigation); err != nil {
		t.Fatal(err)
	}
	if investigation.TaskID == "" {
		t.Fatal("expected a human-controlled follow-up task")
	}
	if _, found := getEnterpriseTask(investigation.TaskID); !found {
		t.Fatal("investigation task was not stored by the existing task service")
	}
}
