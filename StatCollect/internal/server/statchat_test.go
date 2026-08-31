package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matjames/statgate-lib/statchat"
)

func TestStatChatIntegrationUsesCanonicalObjectContract(t *testing.T) {
	previousConfig := cfg
	previousIntegration := statChat
	defer func() {
		cfg = previousConfig
		statChat = previousIntegration
	}()

	cfg = &Config{
		InternalAPIKey: "internal-key",
		TenantID:       "tenant-7",
		WorkspaceID:    "workspace-7",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-API-Key") != "internal-key" ||
			r.Header.Get("X-StatGate-User-ID") != "statcollect-service" ||
			r.Header.Get("X-Tenant-ID") != "tenant-7" ||
			r.Header.Get("X-Workspace-ID") != "workspace-7" {
			t.Fatalf("missing trusted StatChat headers")
		}
		if r.URL.Path == "/v1/chat/conversations/object" {
			var request statchat.ObjectConversationRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode conversation request: %v", err)
			}
			if request.ObjectRef != "obj:statcollect:submission:instance-7" {
				t.Fatalf("unexpected canonical object reference %q", request.ObjectRef)
			}
			_ = json.NewEncoder(w).Encode(statchat.Conversation{ID: "conversation-7", ObjectRef: request.ObjectRef})
			return
		}
		if r.URL.Path == "/v1/chat/messages" {
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode message request: %v", err)
			}
			if request["conversationId"] != "conversation-7" || request["text"] != "submission received" {
				t.Fatalf("unexpected object message request: %+v", request)
			}
			_ = json.NewEncoder(w).Encode(statchat.Message{ID: "message-7"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	integration := InitStatChat(server.URL, true)
	conversationID, err := integration.EnsureObjectDiscussion("submission", "instance-7", "Submission instance-7")
	if err != nil || conversationID != "conversation-7" {
		t.Fatalf("unexpected conversation result %q: %v", conversationID, err)
	}
	if err := integration.PostMessageToObject("submission", "instance-7", "ignored-body-sender", "submission received"); err != nil {
		t.Fatalf("post object message: %v", err)
	}
}
