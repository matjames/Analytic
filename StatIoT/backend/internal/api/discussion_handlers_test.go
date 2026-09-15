package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matjames/statgate-lib/statchat"
)

func TestFieldObjectReference(t *testing.T) {
	if got := fieldObjectReference("device", "device-1"); got != "obj:statiot:device:device-1" {
		t.Fatalf("unexpected object reference: %q", got)
	}
}

func TestStatChatIntegrationForwardsFieldIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/conversations/object" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Internal-API-Key") != "internal" || r.Header.Get("X-StatGate-User-ID") != "user-1" || r.Header.Get("X-Tenant-ID") != "tenant-1" || r.Header.Get("X-Workspace-ID") != "workspace-1" {
			t.Fatalf("field identity context was not forwarded")
		}
		var input statchat.ObjectConversationRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if input.ObjectRef != "obj:statiot:device:device-1" {
			t.Fatalf("unexpected object ref: %q", input.ObjectRef)
		}
		_ = json.NewEncoder(w).Encode(statchat.Conversation{ID: "conversation-1", ObjectRef: input.ObjectRef, Name: input.Name})
	}))
	defer server.Close()

	integration := NewStatChatIntegration(server.URL, "internal")
	conversation, err := integration.ensureDiscussion(context.Background(), "user-1", "tenant-1", "workspace-1", "device", "device-1", "Device 1")
	if err != nil || conversation.ID != "conversation-1" {
		t.Fatalf("unexpected conversation result: %+v, err=%v", conversation, err)
	}
}
