package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Service Registry ─────────────────────────────────────────────
// Authoritative list of all StatGate services, persisted to PostgreSQL.
// Services register via POST /api/registry/services (requires internal API key).
// The Command Centre Platform Health view reads this to display live service status.

type ServiceRegistration struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"display_name"`
	Description  string                 `json:"description"`
	APIURL       string                 `json:"api_url"`
	UIURL        string                 `json:"ui_url"`
	HealthURL    string                 `json:"health_url"`
	Version      string                 `json:"version"`
	Capabilities []string               `json:"capabilities,omitempty"`
	Status       string                 `json:"status"`
	RegisteredAt string                 `json:"registered_at"`
	LastHeartbeat string                `json:"last_heartbeat,omitempty"`
}

// bootstrapServiceRegistry seeds well-known StatGate services into the registry
// on first startup if the table is empty.
func bootstrapServiceRegistry() {
	if dbPool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	_ = dbPool.QueryRowContext(ctx, "SELECT COUNT(*) FROM platform_service_registry").Scan(&count)
	if count > 0 {
		return
	}

	services := []ServiceRegistration{
		{ID: "enterprise-core", Name: "enterprise-core", DisplayName: "Enterprise Core", Description: "StatGate institutional backbone — events, notifications, analytics, workflows", APIURL: getEnv("ENTERPRISE_CORE_URL", "http://localhost:8096"), HealthURL: getEnv("ENTERPRISE_CORE_URL", "http://localhost:8096") + "/health", Status: "active"},
		{ID: "registry", Name: "registry", DisplayName: "Registry", Description: "Identity, users, organisations, roles", APIURL: getEnv("REGISTRY_API_URL", "http://localhost:8080"), HealthURL: getEnv("REGISTRY_API_URL", "http://localhost:8080") + "/health", UIURL: getEnv("REGISTRY_UI_URL", "http://localhost:3007"), Status: "active"},
		{ID: "pms", Name: "pms", DisplayName: "PMS", Description: "Project Management System", APIURL: getEnv("PMS_API_URL", "http://localhost:8001"), HealthURL: getEnv("PMS_API_URL", "http://localhost:8001") + "/health", UIURL: getEnv("PMS_UI_URL", "http://localhost:3010"), Status: "active"},
		{ID: "rms", Name: "rms", DisplayName: "RMS", Description: "Research Management System", APIURL: getEnv("RMS_API_URL", "http://localhost:8002"), HealthURL: getEnv("RMS_API_URL", "http://localhost:8002") + "/health", UIURL: getEnv("RMS_UI_URL", "http://localhost:3011"), Status: "active"},
		{ID: "statchat", Name: "statchat", DisplayName: "StatChat", Description: "AI-assisted investigation and knowledge chat", APIURL: getEnv("STATCHAT_API_URL", "http://localhost:5001"), HealthURL: getEnv("STATCHAT_API_URL", "http://localhost:5001") + "/health", UIURL: getEnv("STATCHAT_UI_URL", "http://localhost:3009"), Status: "active"},
		{ID: "statcollect", Name: "statcollect", DisplayName: "StatCollect", Description: "Field data collection and survey engine", APIURL: getEnv("STATCOLLECT_API_URL", "http://localhost:5050"), HealthURL: getEnv("STATCOLLECT_API_URL", "http://localhost:5050") + "/health", Status: "active"},
		{ID: "helpdesk", Name: "helpdesk", DisplayName: "HelpDesk", Description: "Institutional support ticketing", APIURL: getEnv("HELPDESK_API_URL", "http://localhost:8003"), HealthURL: getEnv("HELPDESK_API_URL", "http://localhost:8003") + "/health", UIURL: getEnv("HELPDESK_UI_URL", "http://localhost:3005"), Status: "active"},
		{ID: "statgovernance", Name: "statgovernance", DisplayName: "StatGovernance", Description: "Risk, compliance, audit and governance", APIURL: getEnv("STATGOVERNANCE_API_URL", "http://localhost:8097"), HealthURL: getEnv("STATGOVERNANCE_API_URL", "http://localhost:8097") + "/health", Status: "active"},
		{ID: "statgate-report-builder", Name: "statgate-report-builder", DisplayName: "StatGate Report Builder", Description: "Visual YAML report builder & explorer — schema introspection, KPIs, charts, maps and tables over the StatGate analytics data warehouse", APIURL: getEnv("REPORT_BUILDER_API_URL", "http://localhost:8110"), HealthURL: getEnv("REPORT_BUILDER_API_URL", "http://localhost:8110") + "/health", UIURL: getEnv("REPORT_BUILDER_UI_URL", "http://localhost:8110/builder.html"), Status: "active"},
	}

	for _, svc := range services {
		capsJSON, _ := json.Marshal(svc.Capabilities)
		_, err := dbPool.ExecContext(ctx,
			`INSERT INTO platform_service_registry (id, name, display_name, description, api_url, ui_url, health_url, version, capabilities, status, registered_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,NOW()) ON CONFLICT (id) DO NOTHING`,
			svc.ID, svc.Name, svc.DisplayName, svc.Description, svc.APIURL, svc.UIURL, svc.HealthURL, "", string(capsJSON), svc.Status,
		)
		if err != nil {
			log.Printf("service_registry: failed to seed %s: %v", svc.ID, err)
		}
	}
	log.Println("service_registry: seeded well-known services")
}

// ─── API Handlers ─────────────────────────────────────────────────

func handleListRegistryServices(c *gin.Context) {
	if dbPool == nil {
		c.JSON(503, gin.H{"error": "service registry unavailable — database not configured"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := dbPool.QueryContext(ctx,
		`SELECT id, name, display_name, description, api_url, ui_url, health_url, version, status, registered_at, last_heartbeat
		FROM platform_service_registry ORDER BY display_name`)
	if err != nil {
		c.JSON(500, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	services := []ServiceRegistration{}
	for rows.Next() {
		var s ServiceRegistration
		var lastHeartbeat *time.Time
		var registeredAt time.Time
		if err := rows.Scan(&s.ID, &s.Name, &s.DisplayName, &s.Description,
			&s.APIURL, &s.UIURL, &s.HealthURL, &s.Version, &s.Status, &registeredAt, &lastHeartbeat); err != nil {
			continue
		}
		s.RegisteredAt = registeredAt.UTC().Format(time.RFC3339)
		if lastHeartbeat != nil {
			s.LastHeartbeat = lastHeartbeat.UTC().Format(time.RFC3339)
		}
		services = append(services, s)
	}
	metricsCollector.incDBQuery()
	c.JSON(200, gin.H{"count": len(services), "services": services})
}

func handleRegisterService(c *gin.Context) {
	// Require internal API key for service registration
	apiKey := c.GetHeader("X-Internal-API-Key")
	expectedKey := getEnvValue("STATGATE_INTERNAL_API_KEY")
	if expectedKey != "" && apiKey != expectedKey {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid internal API key"})
		return
	}

	var svc ServiceRegistration
	if err := c.ShouldBindJSON(&svc); err != nil {
		c.JSON(400, gin.H{"error": "invalid service registration", "detail": err.Error()})
		return
	}
	if svc.ID == "" {
		svc.ID = svc.Name
	}
	if dbPool == nil {
		c.JSON(503, gin.H{"error": "service registry unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	capsJSON, _ := json.Marshal(svc.Capabilities)
	_, err := dbPool.ExecContext(ctx,
		`INSERT INTO platform_service_registry (id, name, display_name, description, api_url, ui_url, health_url, version, capabilities, status, registered_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,NOW())
		ON CONFLICT (id) DO UPDATE SET display_name=$3, description=$4, api_url=$5, ui_url=$6, health_url=$7, version=$8, capabilities=$9::jsonb, status=$10`,
		svc.ID, svc.Name, svc.DisplayName, svc.Description, svc.APIURL, svc.UIURL,
		svc.HealthURL, svc.Version, string(capsJSON), "active",
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "registration failed", "detail": err.Error()})
		return
	}
	metricsCollector.incDBQuery()
	c.JSON(201, gin.H{"status": "registered", "id": svc.ID})
}

func handleServiceHeartbeat(c *gin.Context) {
	id := c.Param("id")
	apiKey := c.GetHeader("X-Internal-API-Key")
	expectedKey := getEnvValue("STATGATE_INTERNAL_API_KEY")
	if expectedKey != "" && apiKey != expectedKey {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid internal API key"})
		return
	}
	if dbPool == nil {
		c.JSON(503, gin.H{"error": "service registry unavailable"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx,
		`UPDATE platform_service_registry SET last_heartbeat=NOW(), status='active' WHERE id=$1`, id)
	if err != nil {
		c.JSON(500, gin.H{"error": "heartbeat update failed"})
		return
	}
	metricsCollector.incDBQuery()
	c.JSON(200, gin.H{"status": "ok", "id": id, "heartbeat": time.Now().UTC().Format(time.RFC3339)})
}

func handleServiceHealthProxy(c *gin.Context) {
	id := c.Param("id")
	if dbPool == nil {
		c.JSON(503, gin.H{"error": "service registry unavailable"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var healthURL string
	err := dbPool.QueryRowContext(ctx, `SELECT health_url FROM platform_service_registry WHERE id=$1`, id).Scan(&healthURL)
	if err != nil || healthURL == "" {
		c.JSON(404, gin.H{"error": "service not found", "id": id})
		return
	}

	// Proxy the health check with a short timeout
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer httpCancel()
	req, _ := http.NewRequestWithContext(httpCtx, http.MethodGet, healthURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(200, gin.H{"service": id, "status": "unreachable", "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		result = map[string]interface{}{"raw": string(body)}
	}
	result["_proxy_status"] = resp.StatusCode
	result["_service_id"] = id
	c.JSON(resp.StatusCode, result)
}

func handlePlatformMetrics(c *gin.Context) {
	snap := metricsCollector.snapshot()
	snap["phase12"] = PhaseXIIMetrics()
	c.JSON(200, snap)
}

func handleReadiness(c *gin.Context) {
	ready, details := isReady()
	status := 200
	statusText := "ready"
	if !ready {
		status = 503
		statusText = "not_ready"
	}
	c.JSON(status, gin.H{
		"status":  statusText,
		"service": "statgate-enterprise-core",
		"checks":  details,
	})
}

func handleLiveness(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "alive",
		"service":   "statgate-enterprise-core",
		"uptime_s":  int(time.Since(startTime).Seconds()),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handlePlatformSummary returns an admin overview of the platform:
// service count, event stats, DLQ depth, DB connectivity.
func handlePlatformSummary(c *gin.Context) {
	if !requireRole(c, "admin", "superadmin") {
		return
	}
	snap := metricsCollector.snapshot()
	_, readinessDetails := isReady()
	c.JSON(200, gin.H{
		"platform_metrics": snap,
		"readiness":        readinessDetails,
		"phase":            "PHASE_X",
		"version":          version,
	})
}

// ─── Service Registry init ────────────────────────────────────────

func initServiceRegistry() {
	if dbPool == nil {
		return
	}
	bootstrapServiceRegistry()
	// Start periodic heartbeat updater for self (Enterprise Core)
	go func() {
		for range time.Tick(30 * time.Second) {
			if dbPool == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, _ = dbPool.ExecContext(ctx,
				`UPDATE platform_service_registry SET last_heartbeat=NOW(), status='active', version=$1 WHERE id='enterprise-core'`,
				version,
			)
			cancel()
		}
	}()
	log.Println("service_registry: initialized")
}

// parseIntDefault already declared in workspace.go or analytics helpers.
// Redeclared here only if missing — Go build will catch duplication.
func parseServiceIntDefault(s string, d int) int {
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return d
	}
	return v
}
