package api

import (
	"net/http/httptest"
	"testing"
)

func TestTrustedObjectConversationRequestFailsClosed(t *testing.T) {
	t.Setenv("STATGATE_INTERNAL_API_KEY", "internal-key")
	req := httptest.NewRequest("POST", "/v1/chat/conversations/object", nil)
	if trustedObjectConversationRequest(req) {
		t.Fatal("expected missing internal key to be denied")
	}
	req.Header.Set("X-Internal-API-Key", "wrong-key")
	if trustedObjectConversationRequest(req) {
		t.Fatal("expected incorrect internal key to be denied")
	}
	req.Header.Set("X-Internal-API-Key", "internal-key")
	if !trustedObjectConversationRequest(req) {
		t.Fatal("expected matching internal key to be trusted")
	}
}
