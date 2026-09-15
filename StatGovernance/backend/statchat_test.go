package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matjames/statgate-lib/statchat"
)

func TestGovernanceObjectReference(t *testing.T) {
	if got := governanceObjectReference("risk", "risk-1"); got != "obj:statgovernance:risk:risk-1" {
		t.Fatalf("unexpected object reference: %q", got)
	}
}

func TestGovernanceStatChatUsesRequestIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/conversations/object" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Internal-API-Key") != "internal" || r.Header.Get("X-StatGate-User-ID") != "user-7" || r.Header.Get("X-Tenant-ID") != "tenant-7" || r.Header.Get("X-Workspace-ID") != "workspace-7" {
			t.Fatalf("request identity was not forwarded")
		}
		var input statchat.ObjectConversationRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if input.ObjectRef != "obj:statgovernance:risk:risk-7" {
			t.Fatalf("unexpected object ref: %q", input.ObjectRef)
		}
		_ = json.NewEncoder(w).Encode(statchat.Conversation{ID: "conversation-7", ObjectRef: input.ObjectRef, Name: input.Name})
	}))
	defer server.Close()

	integration := &governanceStatChat{
		baseURL: server.URL,
		client:  statchat.NewClient(server.URL, "internal"),
	}
	conversation, err := integration.ensureDiscussion(context.Background(), "user-7", "tenant-7", "workspace-7", "risk", "risk-7", "Risk 7")
	if err != nil || conversation.ID != "conversation-7" {
		t.Fatalf("unexpected conversation result: %+v, err=%v", conversation, err)
	}
}
