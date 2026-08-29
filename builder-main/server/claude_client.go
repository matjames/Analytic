package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var claudeAPIKey string
var claudeModel string

const claudeTimeout = 60 * time.Second

func initClaude(apiKey string) {
	claudeAPIKey = apiKey
	claudeModel = os.Getenv("CLAUDE_MODEL")
	if claudeModel == "" {
		claudeModel = "claude-sonnet-4-6"
	}
	log.Printf("Claude API key configured (model: %s)", claudeModel)
}

// Anthropic Messages API types

type claudeRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	System    string           `json:"system,omitempty"`
	Messages  []claudeMessage  `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []claudeContentBlock `json:"content"`
	Error   *claudeError         `json:"error,omitempty"`
	Type    string               `json:"type"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// callClaude sends a prompt to the Anthropic Messages API.
// When wantJSON is true, the system prompt is amended to request pure JSON output.
func callClaude(ctx context.Context, systemPrompt, userPrompt string, wantJSON bool) (string, error) {
	if claudeAPIKey == "" {
		return "", fmt.Errorf("Claude API key not configured")
	}

	system := systemPrompt
	if wantJSON && system != "" {
		system += "\n\nIMPORTANT: Respond with ONLY valid JSON. No markdown fences, no commentary, no text before or after the JSON object."
	} else if wantJSON {
		system = "Respond with ONLY valid JSON. No markdown fences, no commentary, no text before or after the JSON object."
	}

	reqBody := claudeRequest{
		Model:     claudeModel,
		MaxTokens: 4096,
		System:    system,
		Messages: []claudeMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpCtx, cancel := context.WithTimeout(ctx, claudeTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", claudeAPIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("Claude API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if claudeResp.Error != nil {
		return "", fmt.Errorf("Claude error (%s): %s", claudeResp.Error.Type, claudeResp.Error.Message)
	}

	// Extract text from content blocks
	for _, block := range claudeResp.Content {
		if block.Type == "text" && block.Text != "" {
			text := block.Text
			if wantJSON {
				text = stripMarkdownCodeFence(text)
			}
			return text, nil
		}
	}

	return "", fmt.Errorf("Claude returned no text content")
}
