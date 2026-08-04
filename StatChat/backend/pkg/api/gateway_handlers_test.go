package api

import "testing"

func TestBuildMessageFromPayloadSupportsGatewayEnvelope(t *testing.T) {
	message, err := buildMessageFromPayload(map[string]interface{}{
		"event": "message.send",
		"payload": map[string]interface{}{
			"conversationId": "uganda-ops",
			"channelId":      "public-health",
			"sender":         "Ada",
			"text":           "Gateway message",
		},
	})
	if err != nil {
		t.Fatalf("expected gateway payload to build, got %v", err)
	}
	if message.ConversationID != "uganda-ops" {
		t.Fatalf("expected gateway conversation, got %s", message.ConversationID)
	}
	if message.ChannelID != "public-health" {
		t.Fatalf("expected gateway channel, got %s", message.ChannelID)
	}
	if message.Status != "active" {
		t.Fatalf("expected active status, got %s", message.Status)
	}
}
