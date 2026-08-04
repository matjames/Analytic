package store

import "testing"

func TestConversationRouteKeyUsesTenantAndConversation(t *testing.T) {
	got := conversationRouteKey("uganda", "ops")
	want := "uganda:ops"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
