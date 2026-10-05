package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateResearchAssistant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("authorization header = %q, want bearer token", r.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request: %v", err)
		}
		var payload researchAssistantRequestPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Model != "research-test" || len(payload.Messages) != 2 {
			t.Fatalf("payload model/messages = %q/%d", payload.Model, len(payload.Messages))
		}
		if !strings.Contains(payload.Messages[1].Content, "grounded evidence") {
			t.Errorf("grounded context missing from provider request: %s", payload.Messages[1].Content)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"Use the recorded evidence to refine the next study step."}}]}`)
	}))
	defer server.Close()

	t.Setenv("RMS_LLM_API_URL", server.URL)
	t.Setenv("RMS_LLM_API_KEY", "test-key")
	t.Setenv("RMS_LLM_MODEL", "research-test")

	output, model, err := generateResearchAssistant(context.Background(), "proposal planning", "Help plan the proposal.", "grounded evidence")
	if err != nil {
		t.Fatalf("generate assistant output: %v", err)
	}
	if output != "Use the recorded evidence to refine the next study step." || model != "research-test" {
		t.Fatalf("output/model = %q/%q", output, model)
	}
}

func TestGenerateResearchAssistantRequiresProvider(t *testing.T) {
	t.Setenv("RMS_LLM_API_URL", "")
	_, _, err := generateResearchAssistant(context.Background(), "proposal planning", "Help.", "evidence")
	if err == nil {
		t.Fatal("expected missing provider error")
	}
	if got := researchAssistantProviderErrorCode(err); got != http.StatusServiceUnavailable {
		t.Fatalf("error code = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestGenerateResearchLanguageEdit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request: %v", err)
		}
		var payload researchAssistantRequestPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(payload.Messages[0].Content, "without changing scientific meaning") || !strings.Contains(payload.Messages[1].Content, "Source text:") {
			t.Fatalf("language editing instructions missing: %+v", payload.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"Edited research text."}}]}`)
	}))
	defer server.Close()

	t.Setenv("RMS_LLM_API_URL", server.URL)
	t.Setenv("RMS_LLM_API_KEY", "")
	t.Setenv("RMS_LLM_MODEL", "editing-test")

	output, model, err := generateResearchLanguageEdit(context.Background(), "Original research text.")
	if err != nil {
		t.Fatalf("generate language edit: %v", err)
	}
	if output != "Edited research text." || model != "editing-test" {
		t.Fatalf("output/model = %q/%q", output, model)
	}
}
