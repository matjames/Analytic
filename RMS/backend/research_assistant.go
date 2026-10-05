package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type researchAssistantProviderError struct {
	Code int
	Err  error
}

func (e *researchAssistantProviderError) Error() string { return e.Err.Error() }

type researchAssistantMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type researchAssistantRequestPayload struct {
	Model       string                     `json:"model"`
	Messages    []researchAssistantMessage `json:"messages"`
	Temperature float64                    `json:"temperature"`
	MaxTokens   int                        `json:"max_tokens,omitempty"`
}

func generateResearchAssistant(ctx context.Context, task, prompt, groundedContext string) (string, string, error) {
	return generateResearchProvider(ctx,
		"You are a careful research planning assistant. Use only the grounded study context supplied by the application. Do not invent sources, results, ethics approvals, methods, or claims. Clearly label suggestions as suggestions, identify missing evidence, and recommend human or ethics review when appropriate. Return concise Markdown.",
		task, prompt, groundedContext)
}

func generateResearchLanguageEdit(ctx context.Context, original string) (string, string, error) {
	return generateResearchProvider(ctx,
		"You are a careful research language editor. Improve clarity, grammar, structure, and readability without changing scientific meaning, study results, citations, ethics status, methods, or claims. Return only the edited text in plain text or Markdown. Do not invent or remove evidence.",
		"language editing",
		"Edit the following research text while preserving its meaning and all factual details:\n\n"+original,
		"Source text:\n"+original)
}

func generateResearchProvider(ctx context.Context, systemInstruction, task, prompt, groundedContext string) (string, string, error) {
	endpoint := strings.TrimSpace(getEnv("RMS_LLM_API_URL", ""))
	if endpoint == "" {
		return "", "", &researchAssistantProviderError{Code: http.StatusServiceUnavailable, Err: fmt.Errorf("research assistant provider is not configured")}
	}
	model := firstNonEmpty(getEnv("RMS_LLM_MODEL", ""), "research-assistant")
	maxTokens := 1200
	if configured := strings.TrimSpace(getEnv("RMS_LLM_MAX_TOKENS", "")); configured != "" {
		if parsed, err := strconv.Atoi(configured); err == nil && parsed >= 128 && parsed <= 8000 {
			maxTokens = parsed
		}
	}
	payload := researchAssistantRequestPayload{
		Model: model,
		Messages: []researchAssistantMessage{
			{Role: "system", Content: systemInstruction},
			{Role: "user", Content: fmt.Sprintf("Task: %s\n\nResearcher request:\n%s\n\nGrounded study context:\n%s", task, prompt, groundedContext)},
		},
		Temperature: 0.2,
		MaxTokens:   maxTokens,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", model, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", model, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(getEnv("RMS_LLM_API_KEY", "")); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := (&http.Client{Timeout: 48 * time.Second}).Do(req)
	if err != nil {
		return "", model, &researchAssistantProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("research assistant request failed: %w", err)}
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", model, &researchAssistantProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("research assistant provider returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))}
	}
	var response struct {
		Choices []struct {
			Message researchAssistantMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", model, &researchAssistantProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("research assistant provider returned invalid JSON: %w", err)}
	}
	if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
		return "", model, &researchAssistantProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("research assistant provider returned no assistant content")}
	}
	return strings.TrimSpace(response.Choices[0].Message.Content), model, nil
}

func researchAssistantProviderErrorCode(err error) int {
	if providerErr, ok := err.(*researchAssistantProviderError); ok {
		return providerErr.Code
	}
	return http.StatusBadGateway
}
