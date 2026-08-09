package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// StatChatIntegration enables StatGate's communication backbone:
// every submission can be discussed through a dedicated StatChat
// object-linked conversation.
type StatChatIntegration struct {
	BaseURL string
	Enabled bool
	Client  *http.Client
}

var statChat *StatChatIntegration

// InitStatChat sets up the StatChat object-discussion integration.
func InitStatChat(baseURL string, enabled bool) *StatChatIntegration {
	statChat = &StatChatIntegration{
		BaseURL: baseURL,
		Enabled: enabled,
		Client:  &http.Client{Timeout: 10 * time.Second},
	}
	if !enabled {
		log.Printf("StatChat integration disabled")
	} else {
		log.Printf("StatChat integration enabled (base URL: %s)", baseURL)
	}
	return statChat
}

// ObjectConversationID returns the deterministic StatChat conversation ID
// for a platform object. StatChat derives the ID as "obj:{type}:{id}".
func ObjectConversationID(objectType, objectID string) string {
	return fmt.Sprintf("obj:%s:%s", objectType, objectID)
}

// EnsureObjectDiscussion creates (or fetches) the StatChat object-linked
// conversation for a submission. Returns the conversation ID on success.
func (s *StatChatIntegration) EnsureObjectDiscussion(objectType, objectID, name string) (string, error) {
	if s == nil || !s.Enabled {
		return "", fmt.Errorf("statchat integration disabled")
	}
	url := fmt.Sprintf("%s/v1/objects/%s/%s/discussion", s.BaseURL, objectType, objectID)
	if name != "" {
		url += "?name=" + urlQueryEscape(name)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	// Use the internal key if configured (StatChat validates Registry JWT,
	// but for service-to-service we send X-API-Key which passes through).
	req.Header.Set("X-API-Key", cfg.InternalAPIKey)
	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("statchat request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("statchat returned status %d", resp.StatusCode)
	}
	var conv struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&conv); err != nil {
		return "", err
	}
	if conv.ID == "" {
		return "", fmt.Errorf("statchat response missing conversation id")
	}
	return conv.ID, nil
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

// PostMessageToObject sends a message to the object-linked conversation.
// Useful for automated notifications when submissions arrive.
func (s *StatChatIntegration) PostMessageToObject(objectType, objectID, sender, text string) error {
	if s == nil || !s.Enabled {
		return fmt.Errorf("statchat integration disabled")
	}
	convID := ObjectConversationID(objectType, objectID)
	body, err := json.Marshal(map[string]interface{}{
		"conversationId": convID,
		"sender":         sender,
		"text":           text,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, s.BaseURL+"/v1/chat/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", cfg.InternalAPIKey)
	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("statchat message failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("statchat returned status %d", resp.StatusCode)
	}
	return nil
}

// IsEnabled returns whether StatChat integration is active.
func (s *StatChatIntegration) IsEnabled() bool {
	return s != nil && s.Enabled
}

// String returns a human-readable description of the StatChat integration.
func (s *StatChatIntegration) String() string {
	if s == nil || !s.Enabled {
		return "disabled"
	}
	return fmt.Sprintf("enabled (%s)", s.BaseURL)
}

// urlQueryEscape is a small helper to keep imports minimal.
func urlQueryEscape(s string) string {
	var sb bytes.Buffer
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			sb.WriteByte(c)
		} else {
			fmt.Fprintf(&sb, "%%%02X", c)
		}
	}
	return sb.String()
}
