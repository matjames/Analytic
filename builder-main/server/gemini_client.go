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

var geminiAPIKey string
var geminiModel string

const geminiTimeout = 30 * time.Second

func initGemini(apiKey string) {
	geminiAPIKey = apiKey
	geminiModel = os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-2.5-flash-lite"
	}
	log.Printf("Gemini API key configured (model: %s)", geminiModel)
}

// GeminiRequest is the request body for the Gemini generateContent API
type GeminiRequest struct {
	Contents          []GeminiContent    `json:"contents"`
	SystemInstruction *GeminiContent     `json:"systemInstruction,omitempty"`
	GenerationConfig  *GeminiGenConfig   `json:"generationConfig,omitempty"`
}

// GeminiContent represents a content block (user or model)
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

// GeminiPart represents a single part of content
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenConfig controls generation parameters
type GeminiGenConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	ResponseMimeType string `json:"responseMimeType,omitempty"`
}

// GeminiResponse is the response from the Gemini generateContent API
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
	Error      *GeminiError      `json:"error,omitempty"`
}

// GeminiCandidate represents a single response candidate
type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

// GeminiError represents an API error
type GeminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// callGemini sends a prompt to the Gemini API and returns the text response.
func callGemini(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if geminiAPIKey == "" {
		return "", fmt.Errorf("Gemini API key not configured")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		geminiModel, geminiAPIKey)

	req := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{{Text: userPrompt}},
				Role:  "user",
			},
		},
	}

	if systemPrompt != "" {
		req.SystemInstruction = &GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpCtx, cancel := context.WithTimeout(ctx, geminiTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("Gemini API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if geminiResp.Error != nil {
		return "", fmt.Errorf("Gemini error %d: %s", geminiResp.Error.Code, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini returned no content")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// callGeminiJSON sends a prompt expecting JSON response from Gemini.
func callGeminiJSON(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if geminiAPIKey == "" {
		return "", fmt.Errorf("Gemini API key not configured")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		geminiModel, geminiAPIKey)

	req := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{{Text: userPrompt}},
				Role:  "user",
			},
		},
		GenerationConfig: &GeminiGenConfig{
			Temperature:     0.1,
			ResponseMimeType: "application/json",
		},
	}

	if systemPrompt != "" {
		req.SystemInstruction = &GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpCtx, cancel := context.WithTimeout(ctx, geminiTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("Gemini API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if geminiResp.Error != nil {
		return "", fmt.Errorf("Gemini error %d: %s", geminiResp.Error.Code, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini returned no content")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}
