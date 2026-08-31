package statchat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL       string
	InternalKey   string
	ServiceUserID string
	TenantID      string
	HTTPClient    *http.Client
}

type ObjectConversationRequest struct {
	ObjectRef string         `json:"objectRef"`
	Name      string         `json:"name"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Conversation struct {
	ID        string   `json:"id"`
	ObjectRef string   `json:"objectRef"`
	Name      string   `json:"name"`
	MemberIDs []string `json:"memberIds"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	ChannelID      string    `json:"channelId"`
	Sender         string    `json:"sender"`
	Text           string    `json:"text"`
	CreatedAt      time.Time `json:"createdAt"`
}

type StatusError struct {
	StatusCode int
	Message    string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("StatChat returned %d: %s", e.StatusCode, e.Message)
}

func NewClient(baseURL, internalKey string) *Client {
	return &Client{
		BaseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		InternalKey: strings.TrimSpace(internalKey),
		HTTPClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) EnsureObjectConversation(ctx context.Context, authorization, workspaceID string, input ObjectConversationRequest) (Conversation, error) {
	var conversation Conversation
	err := c.doJSON(ctx, http.MethodPost, "/v1/chat/conversations/object", authorization, workspaceID, input, &conversation)
	return conversation, err
}

func (c *Client) Messages(ctx context.Context, authorization, workspaceID, conversationID string) ([]Message, error) {
	messages := []Message{}
	err := c.doJSON(ctx, http.MethodGet, "/v1/chat/conversations/"+conversationID+"/messages", authorization, workspaceID, nil, &messages)
	return messages, err
}

func (c *Client) SendMessage(ctx context.Context, authorization, workspaceID, conversationID, channelID, text string) (Message, error) {
	var message Message
	payload := map[string]string{
		"conversationId": conversationID,
		"channelId":      channelID,
		"text":           text,
	}
	err := c.doJSON(ctx, http.MethodPost, "/v1/chat/messages", authorization, workspaceID, payload, &message)
	return message, err
}

func (c *Client) doJSON(ctx context.Context, method, requestPath, authorization, workspaceID string, input, output any) error {
	if c.BaseURL == "" || c.InternalKey == "" {
		return fmt.Errorf("StatChat service integration is not configured")
	}

	var body *bytes.Reader
	if input == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+requestPath, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-Internal-API-Key", c.InternalKey)
	if c.ServiceUserID != "" {
		req.Header.Set("X-StatGate-User-ID", c.ServiceUserID)
	}
	if c.TenantID != "" {
		req.Header.Set("X-Tenant-ID", c.TenantID)
	}
	if workspaceID != "" {
		req.Header.Set("X-Workspace-ID", workspaceID)
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(response.Body).Decode(&failure)
		return &StatusError{StatusCode: response.StatusCode, Message: failure.Error}
	}
	if output == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(output)
}
