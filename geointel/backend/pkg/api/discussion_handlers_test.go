package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matjames/statgate-lib/statchat"
)

func TestStatChatIntegrationForwardsGeoIntelIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/conversations/object" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-Internal-API-Key"); got != "internal-key" {
			t.Errorf("internal key = %q", got)
		}
		if got := r.Header.Get("X-StatGate-User-ID"); got != "user-1" {
			t.Errorf("user = %q", got)
		}
		if got := r.Header.Get("X-Tenant-ID"); got != "tenant-1" {
			t.Errorf("tenant = %q", got)
		}
		if got := r.Header.Get("X-Workspace-ID"); got != "workspace-1" {
			t.Errorf("workspace = %q", got)
		}
		var input statchat.ObjectConversationRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if input.ObjectRef != "obj:geointel:scene:scene-1" {
			t.Errorf("object ref = %q", input.ObjectRef)
		}
		_ = json.NewEncoder(w).Encode(statchat.Conversation{ID: "conversation-1", ObjectRef: input.ObjectRef})
	}))
	defer server.Close()

	integration := NewStatChatIntegration(server.URL, "internal-key")
	conversation, err := integration.ensureDiscussion(context.Background(), "Bearer token", "user-1", "tenant-1", "workspace-1", "scene", "scene-1", "Scene")
	if err != nil {
		t.Fatal(err)
	}
	if conversation.ID != "conversation-1" {
		t.Fatalf("conversation id = %q", conversation.ID)
	}
}
