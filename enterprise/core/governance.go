package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var auditLog = struct {
	sync.RWMutex
	entries []AuditRecord
}{entries: []AuditRecord{}}

type AuditRecord struct {
	ID        string                 `json:"id"`
	Action    string                 `json:"action"`
	App       string                 `json:"app"`
	User      string                 `json:"user"`
	Resource  string                 `json:"resource,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IP        string                 `json:"ip,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// recordAudit and handleAuditLog are defined in audit_log.go (Phase X).
// The in-memory auditLog store below is kept for supplementary in-process
// access (e.g. the monitoring endpoints) but is no longer the authoritative
// audit trail — PostgreSQL is.

func appendAuditRecord(action, app, user string, details map[string]interface{}) {
	rec := AuditRecord{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		Action:    action,
		App:       app,
		User:      user,
		Details:   details,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	auditLog.Lock()
	auditLog.entries = append(auditLog.entries, rec)
	if len(auditLog.entries) > 10000 {
		auditLog.entries = auditLog.entries[len(auditLog.entries)-10000:]
	}
	auditLog.Unlock()
}

func handleListAPIs(c *gin.Context) {
	apis := []map[string]interface{}{
		{"name": "Enterprise Events", "version": "v1", "base_path": "/api/events", "methods": []string{"GET", "POST"}},
		{"name": "Notifications", "version": "v1", "base_path": "/api/notifications", "methods": []string{"GET", "PUT", "DELETE", "POST"}},
		{"name": "Timeline", "version": "v1", "base_path": "/api/timeline", "methods": []string{"GET", "POST"}},
		{"name": "Dashboards", "version": "v1", "base_path": "/api/dashboards", "methods": []string{"GET", "POST", "PUT", "DELETE"}},
		{"name": "Widgets", "version": "v1", "base_path": "/api/widgets", "methods": []string{"GET"}},
		{"name": "Files", "version": "v1", "base_path": "/api/files", "methods": []string{"GET", "POST", "PUT", "DELETE"}},
		{"name": "Permissions", "version": "v1", "base_path": "/api/permissions", "methods": []string{"GET", "POST"}},
		{"name": "Calendar", "version": "v1", "base_path": "/api/calendar", "methods": []string{"GET", "POST", "PUT", "DELETE"}},
		{"name": "Reports", "version": "v1", "base_path": "/api/reports", "methods": []string{"GET", "POST", "DELETE"}},
		{"name": "AI Catalog", "version": "v1", "base_path": "/api/ai", "methods": []string{"GET"}},
		{"name": "Monitoring", "version": "v1", "base_path": "/api/monitoring", "methods": []string{"GET"}},
	}
	c.JSON(200, gin.H{"apis": apis, "count": len(apis)})
}

func handleAPIHealth(c *gin.Context) {
	health := checkAllServices()
	c.JSON(200, gin.H{"services": health, "timestamp": time.Now().UTC().Format(time.RFC3339)})
}

func handleMonitoringServices(c *gin.Context) {
	c.JSON(200, gin.H{"services": checkAllServices()})
}

func handleMonitoringMetrics(c *gin.Context) {
	uptime := time.Since(startTime).Seconds()
	c.JSON(200, gin.H{
		"uptime_seconds": int(uptime),
		"version":        version,
		"services":       checkAllServices(),
	})
}

func handleMonitoringHealth(c *gin.Context) {
	services := checkAllServices()
	allHealthy := true
	for _, s := range services {
		if s.Status != "healthy" {
			allHealthy = false
			break
		}
	}
	status := "healthy"
	if !allHealthy {
		status = "degraded"
	}
	c.JSON(200, gin.H{
		"status":   status,
		"service":  "statgate-enterprise-core",
		"version":  version,
		"uptime_s": int(time.Since(startTime).Seconds()),
		"services": services,
	})
}

func checkAllServices() []ServiceHealth {
	serviceURLs := []ServiceHealth{
		{Name: "pms", URL: getEnv("PMS_API_URL", "http://localhost:8091")},
		{Name: "rms", URL: getEnv("RMS_API_URL", "http://localhost:8092")},
		{Name: "statgovernance", URL: getEnv("STATGOVERNANCE_API_URL", "http://localhost:8093")},
		{Name: "registry", URL: getEnv("REGISTRY_API_URL", "http://localhost:9090")},
		{Name: "statchat", URL: getEnv("STATCHAT_API_URL", "http://localhost:4000")},
		{Name: "helpdesk", URL: getEnv("HELPDESK_API_URL", "http://localhost:5006")},
		{Name: "analytics", URL: getEnv("ANALYTICS_API_URL", "http://localhost:5000")},
	}
	results := make([]ServiceHealth, 0, len(serviceURLs))
	for _, svc := range serviceURLs {
		results = append(results, probeService(svc))
	}
	return results
}

func probeService(svc ServiceHealth) ServiceHealth {
	client := &http.Client{Timeout: 3 * time.Second}
	start := time.Now()
	resp, err := client.Get(svc.URL + "/health")
	latency := time.Since(start).Milliseconds()
	svc.LatencyMs = latency
	svc.LastCheck = time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		svc.Status = "unreachable"
		svc.Metrics = map[string]interface{}{"error": err.Error()}
		return svc
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		svc.Status = "healthy"
		var info map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&info); err == nil {
			if v, ok := info["version"].(string); ok {
				svc.Version = v
			}
			svc.Metrics = info
		}
	} else {
		svc.Status = "degraded"
		svc.Metrics = map[string]interface{}{"http_status": resp.StatusCode}
	}
	return svc
}

func handleOpenAPISpec(c *gin.Context) {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "StatGate Enterprise Core API",
			"version": version,
		},
		"paths": map[string]interface{}{
			"/api/events": map[string]interface{}{
				"get":  map[string]interface{}{"summary": "List recent enterprise events"},
				"post": map[string]interface{}{"summary": "Publish a new enterprise event"},
			},
			"/api/notifications": map[string]interface{}{
				"get":  map[string]interface{}{"summary": "List notifications for a user"},
				"post": map[string]interface{}{"summary": "Create a notification"},
			},
			"/api/timeline": map[string]interface{}{
				"get":  map[string]interface{}{"summary": "Get enterprise-wide activity timeline"},
				"post": map[string]interface{}{"summary": "Create a timeline entry"},
			},
			"/api/dashboards": map[string]interface{}{
				"get":  map[string]interface{}{"summary": "List configurable dashboards"},
				"post": map[string]interface{}{"summary": "Create a new dashboard"},
			},
			"/api/files": map[string]interface{}{
				"get":  map[string]interface{}{"summary": "List files in the universal file service"},
				"post": map[string]interface{}{"summary": "Upload a file"},
			},
			"/api/permissions/check": map[string]interface{}{
				"get": map[string]interface{}{"summary": "Check if a user has permission"},
			},
			"/api/calendar": map[string]interface{}{
				"get":  map[string]interface{}{"summary": "List calendar events"},
				"post": map[string]interface{}{"summary": "Create a calendar event"},
			},
			"/api/reports": map[string]interface{}{
				"get":  map[string]interface{}{"summary": "List reports"},
				"post": map[string]interface{}{"summary": "Create a report generation request"},
			},
			"/api/monitoring/services": map[string]interface{}{
				"get": map[string]interface{}{"summary": "Get health of all platform services"},
			},
		},
	}
	c.JSON(200, spec)
}

// ─── Helpers ──────────────────────────────────────────────────────

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return def
	}
	return v
}

func fetchJSON(url string) map[string]interface{} {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}
	return data
}

func fetchJSONArray(url string) []map[string]interface{} {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	var data []map[string]interface{}
	if err := json.Unmarshal(body, &data); err == nil {
		return data
	}
	var wrapper map[string]interface{}
	if err := json.Unmarshal(body, &wrapper); err == nil {
		if items, ok := wrapper["items"].([]interface{}); ok {
			result := make([]map[string]interface{}, 0, len(items))
			for _, it := range items {
				if m, ok := it.(map[string]interface{}); ok {
					result = append(result, m)
				}
			}
			return result
		}
		if items, ok := wrapper["results"].([]interface{}); ok {
			result := make([]map[string]interface{}, 0, len(items))
			for _, it := range items {
				if m, ok := it.(map[string]interface{}); ok {
					result = append(result, m)
				}
			}
			return result
		}
	}
	return nil
}

// ─── Scheduler ────────────────────────────────────────────────────

func startScheduler() {
	go func() {
		// Phase IV: Run analytics periodic tasks immediately
		time.Sleep(10 * time.Second)
		runScheduledAlertSweep()
		runAnomalyDetection()
		runScheduledReports()

		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			checkAllServices()
			expireStaleCalendarEvents()

			// ── Phase IV: Periodic analytics tasks ──
			runScheduledAlertSweep()
			runAnomalyDetection()
			runScheduledReports()

			// ── Phase V: Periodic workflow tasks ──
			runApprovalExpirySweep()
			runEscalationSweep()
			runSLACompliance()
		}
	}()
}

func expireStaleCalendarEvents() {
	if redisClient == nil {
		return
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, "statgate:calendar", 0, -1).Result()
	for _, item := range raw {
		var e CalendarEvent
		if err := json.Unmarshal([]byte(item), &e); err == nil {
			if e.End != "" {
				if end, err := time.Parse(time.RFC3339, e.End); err == nil {
					if time.Since(end) > 365*24*time.Hour {
						redisClient.LRem(ctx, "statgate:calendar", 1, item)
					}
				}
			}
		}
	}
}
