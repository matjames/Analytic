package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

// aiProvider tracks which AI backend is active: "gemini", "claude", or "" (none).
var aiProvider string

// InitAI reads AI provider configuration from environment.
// Checks CLAUDE_API_KEY first, then GEMINI_API_KEY.
// If both are set, CLAUDE_API_KEY takes precedence.
func InitAI() {
	claudeKey := os.Getenv("CLAUDE_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if claudeKey != "" {
		initClaude(claudeKey)
		aiProvider = "claude"
		if geminiKey != "" {
			log.Println("AI: Both CLAUDE_API_KEY and GEMINI_API_KEY set — using Claude")
		}
		return
	}

	if geminiKey != "" {
		initGemini(geminiKey)
		aiProvider = "gemini"
		return
	}

	log.Println("AI: No API key configured (set CLAUDE_API_KEY or GEMINI_API_KEY) — /api/ask/* endpoints will return 503")
}

// AIConfigured returns true if an AI provider is available.
func AIConfigured() bool {
	return aiProvider != ""
}

// AIProviderName returns a display name for the active provider.
func AIProviderName() string {
	switch aiProvider {
	case "claude":
		return "Claude"
	case "gemini":
		return "Gemini"
	default:
		return ""
	}
}

// callAI sends a prompt to the configured AI provider and returns the text response.
func callAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	switch aiProvider {
	case "claude":
		return callClaude(ctx, systemPrompt, userPrompt, false)
	case "gemini":
		return callGemini(ctx, systemPrompt, userPrompt)
	default:
		return "", fmt.Errorf("no AI provider configured")
	}
}

// callAIJSON sends a prompt expecting a JSON response from the configured AI provider.
func callAIJSON(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	switch aiProvider {
	case "claude":
		return callClaude(ctx, systemPrompt, userPrompt, true)
	case "gemini":
		return callGeminiJSON(ctx, systemPrompt, userPrompt)
	default:
		return "", fmt.Errorf("no AI provider configured")
	}
}

// stripMarkdownCodeFence removes ```json ... ``` wrapping that some models add.
func stripMarkdownCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}
