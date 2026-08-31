package statchat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsureObjectConversationForwardsIdentityAndServiceContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/conversations/object" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer registry-token" || r.Header.Get("X-Internal-API-Key") != "internal-key" || r.Header.Get("X-StatGate-User-ID") != "service-user" || r.Header.Get("X-Tenant-ID") != "tenant-1" || r.Header.Get("X-Workspace-ID") != "workspace-1" {
			t.Fatalf("required integration headers were not forwarded")
		}
		var input ObjectConversationRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ObjectRef != "obj:pms:project:p-1" {
			t.Fatalf("unexpected payload: %+v, err=%v", input, err)
		}
		_ = json.NewEncoder(w).Encode(Conversation{ID: "conversation-1", ObjectRef: input.ObjectRef})
	}))
	defer server.Close()

	client := NewClient(server.URL, "internal-key")
	client.ServiceUserID = "service-user"
	client.TenantID = "tenant-1"
	conversation, err := client.EnsureObjectConversation(context.Background(), "Bearer registry-token", "workspace-1", ObjectConversationRequest{ObjectRef: "obj:pms:project:p-1", Name: "Project"})
	if err != nil || conversation.ID != "conversation-1" {
		t.Fatalf("unexpected result: %+v, err=%v", conversation, err)
	}
}

func TestClientReturnsUpstreamStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"denied"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "internal-key")
	_, err := client.Messages(context.Background(), "Bearer token", "", "conversation-1")
	statusErr, ok := err.(*StatusError)
	if !ok || statusErr.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status error, got %v", err)
	}
}
