package api

import (
	"strings"
	"testing"

	"statchat/pkg/model"
)

func TestAssistantModes(t *testing.T) {
	for _, mode := range []string{"summary", "actions", "draft"} {
		if !isAssistantMode(mode) {
			t.Fatalf("expected mode %q to be accepted", mode)
		}
	}
	if isAssistantMode("invent") {
		t.Fatal("expected unknown assistant mode to be rejected")
	}
}

func TestBuildAssistantResponseUsesVisibleConversationOnly(t *testing.T) {
	messages := []model.Message{
		{Sender: "Amina", Text: "Please prepare the district report by Friday.", Status: "active"},
		{Sender: "Noah", Text: "secret deleted text", Status: "deleted"},
	}
	actions := buildAssistantResponse("actions", messages)
	if !strings.Contains(actions, "prepare the district report") || strings.Contains(actions, "secret deleted text") {
		t.Fatalf("unexpected action response: %s", actions)
	}
	draft := buildAssistantResponse("draft", messages)
	if !strings.Contains(draft, "Amina") || !strings.Contains(draft, "district report") {
		t.Fatalf("unexpected draft response: %s", draft)
	}
}

func TestBuildAssistantResponseHandlesEmptyConversation(t *testing.T) {
	if response := buildAssistantResponse("summary", nil); !strings.Contains(response, "no visible messages") {
		t.Fatalf("unexpected empty response: %s", response)
	}
}
