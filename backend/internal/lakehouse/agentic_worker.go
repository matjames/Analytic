package lakehouse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// WebhookAction represents a policy-approved auto-action the engine can trigger.
type WebhookAction struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	TenantID    string `json:"tenant_id"`
	MinClearance int   `json:"min_clearance"`
}

// AgenticWorker dispatches ABAC-gated webhooks when anomalies are confirmed.
type AgenticWorker struct {
	policyActions []WebhookAction
}

var globalAgentWorker *AgenticWorker

func GetAgenticWorker() *AgenticWorker {
	if globalAgentWorker == nil {
		globalAgentWorker = &AgenticWorker{
			policyActions: defaultPolicyActions(),
		}
	}
	return globalAgentWorker
}

func defaultPolicyActions() []WebhookAction {
	return []WebhookAction{
		{
			ID:           "wh-001",
			Name:         "Alert Health Coordinator",
			Endpoint:     os.Getenv("WEBHOOK_HEALTH_URL"),
			TenantID:     "tenant-alpha",
			MinClearance: 2,
		},
		{
			ID:           "wh-002",
			Name:         "Alert Education Director",
			Endpoint:     os.Getenv("WEBHOOK_EDUCATION_URL"),
			TenantID:     "tenant-beta",
			MinClearance: 2,
		},
		{
			ID:           "wh-003",
			Name:         "Budget Reallocation Trigger",
			Endpoint:     os.Getenv("WEBHOOK_BUDGET_URL"),
			TenantID:     "*",
			MinClearance: 3,
		},
	}
}

// DispatchIfApproved fires a webhook if the alert crosses policy thresholds.
// Requires clearance >= action.MinClearance. If endpoint is blank, it logs
// the intent instead (safe mode for environments without webhook targets).
func (aw *AgenticWorker) DispatchIfApproved(alert AnomalyAlert, actionID string, clearance int) error {
	var target *WebhookAction
	for _, a := range aw.policyActions {
		if a.ID == actionID {
			target = &a
			break
		}
	}

	if target == nil {
		return fmt.Errorf("unknown action: %s", actionID)
	}
	if clearance < target.MinClearance {
		return fmt.Errorf("ABAC denied: clearance %d < required %d", clearance, target.MinClearance)
	}
	if target.TenantID != "*" && target.TenantID != alert.TenantID {
		return fmt.Errorf("ABAC denied: tenant mismatch")
	}

	payload := map[string]interface{}{
		"alert_id":   alert.ID,
		"metric":     alert.MetricName,
		"dataset":    alert.Dataset,
		"value":      alert.Value,
		"sigma":      alert.SigmaScore,
		"severity":   alert.Severity,
		"message":    alert.Message,
		"tenant_id":  alert.TenantID,
		"triggered_at": time.Now().UTC().Format(time.RFC3339),
	}

	body, _ := json.Marshal(payload)

	// If no real endpoint is configured, log the intent (safe mode)
	if target.Endpoint == "" {
		log.Printf("[AgenticWorker] SAFE-MODE webhook → %s: %s", target.Name, string(body))
		return nil
	}

	resp, err := http.Post(target.Endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook dispatch error: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[AgenticWorker] Webhook '%s' fired → HTTP %d", target.Name, resp.StatusCode)
	return nil
}

// ListActions returns all registered policy-approved webhook actions.
func (aw *AgenticWorker) ListActions() []WebhookAction {
	return aw.policyActions
}
