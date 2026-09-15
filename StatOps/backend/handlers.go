package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler returns platform probe status
func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":   "StatOps (Platform Engineering & RunOps)",
		"status":    "healthy",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadyHandler returns service readiness
func ReadyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"subsystem": "statgate-platform-ops",
	})
}

// SummaryHandler returns mission control high-level dashboard KPIs
func SummaryHandler(c *gin.Context) {
	summary := globalStore.GetOverviewSummary()
	c.JSON(http.StatusOK, summary)
}

// ============================================================================
// P49: INTEGRATED OPERATIONS CENTER (IOC) & OBSERVABILITY
// ============================================================================

func ListTelemetryHandler(c *gin.Context) {
	telemetry := globalStore.ListTelemetry()
	c.JSON(http.StatusOK, gin.H{
		"count":     len(telemetry),
		"telemetry": telemetry,
	})
}

func ScrapeTelemetryHandler(c *gin.Context) {
	if globalScraper != nil {
		results := globalScraper.TriggerImmediateScrape()
		c.JSON(http.StatusOK, gin.H{
			"message": "Immediate telemetry scrape executed across all nodes",
			"count":   len(results),
			"results": results,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"telemetry": globalStore.ListTelemetry()})
}

func ListLogsHandler(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	workspaceID, _ := c.Get("workspace_id")
	logs := globalStore.ListLogs(stringValue(tenantID), stringValue(workspaceID))
	c.JSON(http.StatusOK, gin.H{
		"count": len(logs),
		"logs":  logs,
	})
}

func stringValue(value interface{}) string {
	valueString, _ := value.(string)
	return valueString
}

func ListCMDBHandler(c *gin.Context) {
	cmdb := globalStore.ListCMDB()
	c.JSON(http.StatusOK, gin.H{
		"count": len(cmdb),
		"cmdb":  cmdb,
	})
}

func ListAlertsHandler(c *gin.Context) {
	alerts := globalStore.ListAlerts()
	c.JSON(http.StatusOK, gin.H{
		"count":  len(alerts),
		"alerts": alerts,
	})
}

func CreateAlertHandler(c *gin.Context) {
	var req AlertIncident
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert payload: " + err.Error()})
		return
	}

	created := globalStore.CreateAlert(req)

	publishEvent("ops.alert.triggered", "alert", created.ID, "sre-monitor", "tenant-alpha", map[string]interface{}{
		"title":       created.Title,
		"severity":    created.Severity,
		"source_app":  created.SourceApp,
		"description": created.Description,
	})

	c.JSON(http.StatusCreated, created)
}

func ListStatusPageHandler(c *gin.Context) {
	comps := globalStore.ListPublicStatus()
	c.JSON(http.StatusOK, gin.H{
		"system_status": "All Systems Operational",
		"components":    comps,
		"updated_at":    time.Now().UTC(),
	})
}

// ============================================================================
// P20: CI/CD & CONTINUOUS DELIVERY
// ============================================================================

func ListPipelinesHandler(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tid, _ := tenantID.(string)
	workspaceID, _ := c.Get("workspace_id")
	pipes := globalStore.ListPipelines(tid, workspaceID.(string))
	c.JSON(http.StatusOK, gin.H{
		"count":     len(pipes),
		"pipelines": pipes,
	})
}

func TriggerPipelineHandler(c *gin.Context) {
	var req PipelineRun
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pipeline payload: " + err.Error()})
		return
	}

	tid, _ := c.Get("tenant_id")
	if req.TenantID == "" {
		req.TenantID, _ = tid.(string)
	}
	workspace, _ := c.Get("workspace_id")
	req.WorkspaceID, _ = workspace.(string)

	created := globalStore.TriggerPipeline(req)

	publishEvent("ops.pipeline.triggered", "pipeline", created.ID, created.Author, req.TenantID, map[string]interface{}{
		"repo":         created.RepoName,
		"branch":       created.Branch,
		"commit":       created.CommitSHA,
		"workspace_id": created.WorkspaceID,
	})

	c.JSON(http.StatusAccepted, created)
}

func ListDeploymentsHandler(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tid, _ := tenantID.(string)
	workspaceID, _ := c.Get("workspace_id")
	deps := globalStore.ListDeployments(tid, workspaceID.(string))
	c.JSON(http.StatusOK, gin.H{
		"count":       len(deps),
		"deployments": deps,
	})
}

func CreateDeploymentHandler(c *gin.Context) {
	var req DeploymentRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deployment payload: " + err.Error()})
		return
	}

	tid, _ := c.Get("tenant_id")
	if req.TenantID == "" {
		req.TenantID, _ = tid.(string)
	}
	workspace, _ := c.Get("workspace_id")
	req.WorkspaceID, _ = workspace.(string)

	created := globalStore.CreateDeployment(req)

	publishEvent("ops.deployment.started", "deployment", created.ID, created.DeployedBy, req.TenantID, map[string]interface{}{
		"service":      created.ServiceName,
		"version":      created.Version,
		"environment":  created.Environment,
		"strategy":     created.Strategy,
		"workspace_id": created.WorkspaceID,
	})

	c.JSON(http.StatusCreated, created)
}

func RollbackDeploymentHandler(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		TargetVersion string `json:"target_version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_version is required for rollback"})
		return
	}

	actor, _ := c.Get("user_id")
	actorStr, _ := actor.(string)
	if actorStr == "" {
		actorStr = "sre-operator"
	}

	rolledBack, err := globalStore.RollbackDeployment(id, req.TargetVersion, actorStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	publishEvent("ops.deployment.rolled_back", "deployment", id, actorStr, rolledBack.TenantID, map[string]interface{}{
		"service":        rolledBack.ServiceName,
		"target_version": req.TargetVersion,
		"workspace_id":   rolledBack.WorkspaceID,
	})

	c.JSON(http.StatusOK, rolledBack)
}

// ============================================================================
// P25: SOVEREIGN & MULTI-CLOUD INFRASTRUCTURE
// ============================================================================

func ListClustersHandler(c *gin.Context) {
	clusters := globalStore.ListClusters()
	c.JSON(http.StatusOK, gin.H{
		"count":    len(clusters),
		"clusters": clusters,
	})
}

func ListCostsHandler(c *gin.Context) {
	costs := globalStore.ListCosts()
	c.JSON(http.StatusOK, gin.H{
		"count": len(costs),
		"costs": costs,
	})
}

// ============================================================================
// P34: PRODUCTION READINESS & SRE AUTOMATION
// ============================================================================

func ListReadinessHandler(c *gin.Context) {
	checks := globalStore.ListReadinessChecks()
	c.JSON(http.StatusOK, gin.H{
		"count":            len(checks),
		"readiness_checks": checks,
	})
}

func ListSLOsHandler(c *gin.Context) {
	slos := globalStore.ListSLOs()
	c.JSON(http.StatusOK, gin.H{
		"count": len(slos),
		"slos":  slos,
	})
}

func ExecuteRunbookHandler(c *gin.Context) {
	var req RunbookExecution
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runbook execution request: " + err.Error()})
		return
	}

	actor, _ := c.Get("user_id")
	actorStr, _ := actor.(string)
	if req.TriggeredBy == "" {
		req.TriggeredBy = actorStr
		if req.TriggeredBy == "" {
			req.TriggeredBy = "sre-automation-runner"
		}
	}

	exec := globalStore.ExecuteRunbook(req)

	publishEvent("ops.runbook.executed", "runbook", exec.ID, exec.TriggeredBy, "tenant-alpha", map[string]interface{}{
		"runbook_name":   exec.RunbookName,
		"target_service": exec.TargetService,
		"status":         exec.Status,
	})

	c.JSON(http.StatusOK, exec)
}

func SimulateChaosDrillHandler(c *gin.Context) {
	var req struct {
		Scenario      string `json:"scenario" binding:"required"` // POD_KILL, NETWORK_LATENCY, REGION_FAILOVER
		TargetService string `json:"target_service" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chaos scenario payload"})
		return
	}

	drill := DisasterRecoveryDrill{
		ID:             fmt.Sprintf("chaos-%d", time.Now().Unix()),
		DrillName:      fmt.Sprintf("Simulated %s on %s", req.Scenario, req.TargetService),
		PrimarySite:    "kampala-central",
		DRSite:         "entebbe-dr",
		RTOAchievedMin: 2.4,
		RPOAchievedSec: 0.0,
		Status:         "PASSED",
		ConductedAt:    time.Now().UTC(),
		ConductedBy:    "Chaos Engineering Bot",
		Notes:          "Self-healing controller spun up replica node in 144 seconds; zero data loss.",
	}

	globalStore.mu.Lock()
	globalStore.drDrills = append(globalStore.drDrills, drill)
	globalStore.mu.Unlock()

	publishEvent("ops.chaos.drill.completed", "chaos_drill", drill.ID, drill.ConductedBy, "tenant-alpha", map[string]interface{}{
		"scenario": req.Scenario,
		"service":  req.TargetService,
		"status":   drill.Status,
		"rto_min":  drill.RTOAchievedMin,
	})

	c.JSON(http.StatusOK, drill)
}
