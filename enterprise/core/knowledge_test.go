package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func knowledgeRequest(t *testing.T, router *ginTestRouter, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "knowledge-test-user")
	w := httptest.NewRecorder()
	router.engine.ServeHTTP(w, req)
	return w
}

func TestKnowledgeArticleLifecycleAndSearch(t *testing.T) {
	ts := setupTestRouter()
	w := knowledgeRequest(t, ts, "POST", "/api/knowledge/articles", `{"title":"Offline field reporting lesson","summary":"Use offline-first collection","body":"Connectivity limitations delayed submissions.","category":"lessons_learned","tags":["offline","survey"],"source":"statcollect"}`)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var article KnowledgeArticle
	if err := json.Unmarshal(w.Body.Bytes(), &article); err != nil {
		t.Fatal(err)
	}
	if article.Status != "draft" || article.Version != 1 {
		t.Fatalf("unexpected article lifecycle: %#v", article)
	}
	w = knowledgeRequest(t, ts, "PUT", "/api/knowledge/articles/"+article.ID, `{"status":"published"}`)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	w = knowledgeRequest(t, ts, "GET", "/api/knowledge/search?q=offline", "")
	if w.Code != 200 {
		t.Fatalf("expected searchable article, got %d", w.Code)
	}
	var result struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	if result.Count == 0 {
		t.Fatal("expected knowledge search result")
	}
}

func TestKnowledgeGraphContext(t *testing.T) {
	ts := setupTestRouter()
	w := knowledgeRequest(t, ts, "POST", "/api/knowledge/relationships", `{"from_type":"project","from_id":"PRJ-KNOW-1","to_type":"dataset","to_id":"DS-KNOW-1","relation":"uses","source":"statcollect"}`)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	w = knowledgeRequest(t, ts, "GET", "/api/knowledge/context/project/PRJ-KNOW-1", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var context struct {
		Relationships []KnowledgeRelationship `json:"relationships"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &context)
	if len(context.Relationships) == 0 || context.Relationships[0].ToID != "DS-KNOW-1" {
		t.Fatalf("expected dataset relationship, got %#v", context.Relationships)
	}
}

func TestKnowledgeRejectsAnonymousRead(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/knowledge/search?q=anything", "")
	if w.Code != 403 {
		t.Fatalf("expected 403 for anonymous knowledge access, got %d", w.Code)
	}
}

func TestKnowledgeEventCreatesRealRelationship(t *testing.T) {
	ts := setupTestRouter()
	w := knowledgeRequest(t, ts, "POST", "/api/events", `{"event_type":"survey.created","source":"statcollect","object_type":"survey","object_id":"SURVEY-KNOW-1","project_id":"PRJ-KNOW-EVENT","actor":"knowledge-test-user","payload":{"project_id":"PRJ-KNOW-EVENT","dataset_id":"DATASET-KNOW-1"}}`)
	if w.Code != 201 {
		t.Fatalf("expected event accepted, got %d: %s", w.Code, w.Body.String())
	}
	w = knowledgeRequest(t, ts, "GET", "/api/knowledge/context/survey/SURVEY-KNOW-1", "")
	if w.Code != 200 {
		t.Fatalf("expected context, got %d", w.Code)
	}
	var context struct {
		Relationships []KnowledgeRelationship `json:"relationships"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &context)
	if len(context.Relationships) < 1 {
		t.Fatal("expected event-driven knowledge relationship")
	}
}

func TestKnowledgeQualityDetectsUnapprovedContent(t *testing.T) {
	ts := setupTestRouter()
	w := knowledgeRequest(t, ts, "POST", "/api/knowledge/articles", `{"title":"Needs review","source":"pms","category":"sops"}`)
	if w.Code != 201 {
		t.Fatalf("expected article, got %d", w.Code)
	}
	w = knowledgeRequest(t, ts, "GET", "/api/knowledge/quality", "")
	if w.Code != 200 {
		t.Fatalf("expected quality response, got %d", w.Code)
	}
	var response struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	if response.Count == 0 {
		t.Fatal("expected quality issue for draft article")
	}
}
