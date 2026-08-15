package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — EXTERNAL REASONING PROVIDER WRAPPER
//
// Provider-agnostic wrapper that enforces the AI input contract (directive
// §17): minimum necessary data, classification gate, full traceability.
//
// FAIL CLOSED: if an external provider is configured but the API key is
// missing, Generate() returns an error — there is no anonymous fallback.
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// externalProvider adapts any OpenAI/Gemini-compatible chat-completions-style
// HTTP API to the ReasoningProvider interface. It never falls back to
// anonymous or unauthenticated access.
type externalProvider struct {
	name    string
	apiKey  string
	model   string
	baseURL string
}

func newExternalProvider(name, apiKey, model string) ReasoningProvider {
	if model == "" {
		model = "structured-output-v1"
	}
	baseURL := getEnvValue("STATGATE_AI_BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434/v1" // local inference (e.g. Ollama) default
	}
	return &externalProvider{name: name, apiKey: apiKey, model: model, baseURL: baseURL}
}

func (p *externalProvider) Name() string { return p.name }

// Generate sends the minimum-necessary AIInput to the configured provider.
// It fails closed when the provider is not authenticated.
func (p *externalProvider) Generate(input AIInput) (*AIOutput, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("external provider %q is configured but its API key is missing — AI reasoning is disabled (fail closed)", p.name)
	}
	if !dataClassificationAllowedForProvider(input.DataClassification) {
		return nil, fmt.Errorf("data classification %q is not permitted for external provider %q (fail closed)", input.DataClassification, p.name)
	}

	payload := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{
				"role": "system",
				"content": "You are StatGate's institutional reasoning assistant. " +
					"Return ONLY strict JSON matching: {\"recommendation\":\"...\",\"confidence\":0.0,\"reasoning_summary\":\"...\",\"supporting_evidence\":[],\"affected_objects\":[],\"risk_level\":\"...\",\"recommended_actions\":[],\"limitations\":[]}. " +
					"AI is advisory only; never recommend unauthorized mutation of institutional records.",
			},
			{
				"role":    "user",
				"content": p.buildPrompt(input),
			},
		},
		"response_format": map[string]string{"type": "json_object"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("provider request marshal failed: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("provider request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("provider call failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("provider response decode failed: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("provider returned no choices")
	}

	var out AIOutput
	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &out); err != nil {
		return nil, fmt.Errorf("provider output is not structured JSON: %w", err)
	}
	return &out, nil
}

// buildPrompt renders the AI input contract. Only the minimum necessary data
// is sent; the correlation id is always included for traceability.
func (p *externalProvider) buildPrompt(input AIInput) string {
	snapshot, _ := json.Marshal(input.InstitutionalSnapshot)
	rel, _ := json.Marshal(input.GraphRelationships)
	return fmt.Sprintf(
		"Tenant: %s\nClassification: %s\nCorrelationID: %s\nContext: %s\nInstitutional snapshot: %s\nSupporting evidence: %v\nGraph relationships: %s\nRelevant metrics: %v\n",
		input.TenantID, input.DataClassification, input.CorrelationID, input.Context,
		string(snapshot), input.SupportingEvidence, string(rel), input.RelevantMetrics)
}
