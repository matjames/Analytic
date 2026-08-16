package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type InterAppClient struct {
	httpClient *http.Client
}

var interAppClient = &InterAppClient{
	httpClient: &http.Client{Timeout: 4 * time.Second},
}

// SendSecurityAlertToStatChat posts critical security notifications to StatChat channel
func (c *InterAppClient) SendSecurityAlertToStatChat(title, severity, details string) error {
	statchatURL := getEnv("STATCHAT_API_URL", "http://localhost:4000")
	reqBody := map[string]interface{}{
		"channel":   "security-operations",
		"sender":    "StatTrust SecOps Guard",
		"message":   fmt.Sprintf("🚨 [%s ALERT] %s\n%s", severity, title, details),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"severity":  severity,
	}

	b, _ := json.Marshal(reqBody)
	resp, err := c.httpClient.Post(statchatURL+"/api/v1/alerts", "application/json", bytes.NewBuffer(b))
	if err != nil {
		log.Printf("StatChat alert delivery skipped (offline): %v", err)
		return nil
	}
	defer resp.Body.Close()
	return nil
}

// SyncAuditToEnterpriseCore transmits notarized ledger blocks to Enterprise Core
func (c *InterAppClient) SyncAuditToEnterpriseCore(block AuditLedgerBlock) error {
	coreURL := getEnv("ENTERPRISE_CORE_URL", "http://localhost:8096")
	b, _ := json.Marshal(block)

	req, err := http.NewRequest("POST", coreURL+"/api/enterprise/audit", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-API-Key", getEnv("STATGATE_INTERNAL_API_KEY", "statgate-dev-key"))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Log gracefully when offline
		return nil
	}
	defer resp.Body.Close()
	return nil
}

// CheckPeerAppHealth checks connectivity and responds with latency/status
func (c *InterAppClient) CheckPeerAppHealth(targetURL string) (string, time.Duration) {
	start := time.Now()
	resp, err := c.httpClient.Get(targetURL + "/health")
	latency := time.Since(start)
	if err != nil || resp.StatusCode >= 400 {
		return "UNREACHABLE", latency
	}
	defer resp.Body.Close()
	return "CONNECTED", latency
}

// FetchArtifactFromApp attempts to retrieve artifact raw hash/metadata from PMS, RMS, or Spatial
func (c *InterAppClient) FetchArtifactFromApp(sourceApp, artifactID string) (map[string]interface{}, error) {
	var baseURL string
	switch sourceApp {
	case "RMS":
		baseURL = getEnv("RMS_API_URL", "http://localhost:8092") + "/api/research/" + artifactID
	case "PMS":
		baseURL = getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects/" + artifactID
	case "StatGovernance":
		baseURL = getEnv("GOVERNANCE_API_URL", "http://localhost:8093") + "/api/v1/compliance/frameworks"
	default:
		return nil, fmt.Errorf("unsupported source application: %s", sourceApp)
	}

	resp, err := c.httpClient.Get(baseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}
