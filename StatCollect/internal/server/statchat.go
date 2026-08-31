package server

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/matjames/statgate-lib/statchat"
)

// StatChatIntegration enables StatGate's communication backbone through the
// canonical object-conversation API and shared service authentication.
type StatChatIntegration struct {
	BaseURL string
	Enabled bool
	Client  *statchat.Client
}

var statChat *StatChatIntegration

// InitStatChat sets up the StatChat object-discussion integration.
func InitStatChat(baseURL string, enabled bool) *StatChatIntegration {
	internalKey := ""
	if cfg != nil {
		internalKey = cfg.InternalAPIKey
	}
	statChat = &StatChatIntegration{
		BaseURL: baseURL,
		Enabled: enabled,
		Client:  newStatChatClient(baseURL, internalKey),
	}
	if !enabled {
		log.Printf("StatChat integration disabled")
	} else {
		log.Printf("StatChat integration enabled (base URL: %s)", baseURL)
	}
	return statChat
}

func newStatChatClient(baseURL, internalKey string) *statchat.Client {
	client := statchat.NewClient(baseURL, internalKey)
	client.ServiceUserID = "statcollect-service"
	if cfg != nil && strings.TrimSpace(cfg.TenantID) != "" {
		client.TenantID = strings.TrimSpace(cfg.TenantID)
	} else {
		client.TenantID = "default"
	}
	return client
}

func statChatWorkspaceID() string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.WorkspaceID)
}

// ObjectReference returns the canonical StatChat object reference. StatChat
// generates the conversation ID when the object conversation is first created.
func ObjectReference(objectType, objectID string) string {
	return fmt.Sprintf("obj:statcollect:%s:%s", objectType, objectID)
}

// EnsureObjectDiscussion creates (or fetches) the StatChat object-linked
// conversation for a submission. Returns the conversation ID on success.
func (s *StatChatIntegration) EnsureObjectDiscussion(objectType, objectID, name string) (string, error) {
	if s == nil || !s.Enabled || s.Client == nil {
		return "", fmt.Errorf("statchat integration disabled")
	}
	if strings.TrimSpace(objectType) == "" || strings.TrimSpace(objectID) == "" {
		return "", fmt.Errorf("object type and object id are required")
	}
	conversation, err := s.Client.EnsureObjectConversation(
		context.Background(), "", statChatWorkspaceID(),
		statchat.ObjectConversationRequest{
			ObjectRef: ObjectReference(objectType, objectID),
			Name:      name,
			Metadata: map[string]any{
				"source":     "statcollect",
				"objectType": objectType,
				"objectId":   objectID,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("ensure statchat object conversation: %w", err)
	}
	if strings.TrimSpace(conversation.ID) == "" {
		return "", fmt.Errorf("statchat response missing conversation id")
	}
	return conversation.ID, nil
}

// LinkSubmission creates the StatChat discussion for a submission and
// publishes a submission.linked event. This is the core integration point
// that makes StatCollect submissions discussable within StatGate.
func LinkSubmission(instanceID, formID string) {
	if statChat == nil || !statChat.Enabled {
		return
	}
	name := fmt.Sprintf("Submission %s (%s)", instanceID, formID)
	convID, err := statChat.EnsureObjectDiscussion("submission", instanceID, name)
	if err != nil {
		log.Printf("warning: failed to link submission %s to StatChat: %v", instanceID, err)
		return
	}
	log.Printf("submission %s linked to StatChat conversation %s", instanceID, convID)
	PublishSubmissionLinked(instanceID, convID)
}

// PostMessageToObject sends a service-authored message to the object-linked
// conversation. The request identity is derived from trusted service headers,
// never from the message body.
func (s *StatChatIntegration) PostMessageToObject(objectType, objectID, sender, text string) error {
	if s == nil || !s.Enabled || s.Client == nil {
		return fmt.Errorf("statchat integration disabled")
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("message text is required")
	}
	conversation, err := s.Client.EnsureObjectConversation(
		context.Background(), "", statChatWorkspaceID(),
		statchat.ObjectConversationRequest{
			ObjectRef: ObjectReference(objectType, objectID),
			Name:      fmt.Sprintf("%s %s", objectType, objectID),
		},
	)
	if err != nil {
		return fmt.Errorf("resolve statchat object conversation: %w", err)
	}
	if strings.TrimSpace(conversation.ID) == "" {
		return fmt.Errorf("statchat response missing conversation id")
	}
	_, err = s.Client.SendMessage(context.Background(), "", statChatWorkspaceID(), conversation.ID, "", text)
	if err != nil {
		return fmt.Errorf("send statchat object message: %w", err)
	}
	return nil
}

// IsEnabled returns whether StatChat integration is active.
func (s *StatChatIntegration) IsEnabled() bool {
	return s != nil && s.Enabled
}

// String returns a human-readable description of the integration.
func (s *StatChatIntegration) String() string {
	if s == nil || !s.Enabled {
		return "disabled"
	}
	return fmt.Sprintf("enabled (%s)", s.BaseURL)
}
