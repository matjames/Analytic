package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matjames/statgate-lib/statchat"
)

func TestStatChatIntegrationForwardsSpatialIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/conversations/object" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		for key, want := range map[string]string{"X-Internal-API-Key": "internal-key", "X-StatGate-User-ID": "user-1", "X-Tenant-ID": "tenant-1", "X-Workspace-ID": "workspace-1"} {
			if got := r.Header.Get(key); got != want {
				t.Errorf("%s = %q", key, got)
			}
		}
		var input statchat.ObjectConversationRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if input.ObjectRef != "obj:statspatial:geo_layer:layer-1" {
			t.Errorf("object ref = %q", input.ObjectRef)
		}
		_ = json.NewEncoder(w).Encode(statchat.Conversation{ID: "conversation-1", ObjectRef: input.ObjectRef})
	}))
	defer server.Close()
	conversation, err := NewStatChatIntegration(server.URL, "internal-key").ensureDiscussion(context.Background(), "Bearer token", "user-1", "tenant-1", "workspace-1", "geo_layer", "layer-1", "Layer")
	if err != nil {
		t.Fatal(err)
	}
	if conversation.ID != "conversation-1" {
		t.Fatalf("conversation id = %q", conversation.ID)
	}
}
