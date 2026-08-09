package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// registerRoutes wires all enterprise API routes.
func registerRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// Event Bus
		api.POST("/events", handlePublishEvent)
		api.GET("/events", handleListEvents)
		api.GET("/events/types", handleEventTypes)
		api.GET("/events/:id", handleGetEvent)
		api.GET("/events/subscriptions", handleListSubscriptions)
		api.POST("/events/subscriptions", handleCreateSubscription)

		// Notifications
		api.GET("/notifications", handleListNotifications)
		api.GET("/notifications/:id", handleGetNotification)
		api.PUT("/notifications/:id/read", handleMarkNotificationRead)
		api.PUT("/notifications/read-all", handleMarkAllRead)
		api.PUT("/notifications/:id/archive", handleArchiveNotification)
		api.DELETE("/notifications/:id", handleDeleteNotification)
		api.POST("/notifications", handleCreateNotification)

		// Timeline
		api.GET("/timeline", handleTimeline)
		api.GET("/timeline/entity/:entity/:id", handleTimelineByEntity)
		api.GET("/timeline/user/:user", handleTimelineByUser)
		api.POST("/timeline/entries", handleCreateTimelineEntry)

		// Dashboards
		api.GET("/dashboards", handleListDashboards)
		api.GET("/dashboards/:id", handleGetDashboard)
		api.POST("/dashboards", handleCreateDashboard)
		api.PUT("/dashboards/:id", handleUpdateDashboard)
		api.DELETE("/dashboards/:id", handleDeleteDashboard)
		api.POST("/dashboards/:id/share", handleShareDashboard)
		api.GET("/dashboards/:id/widgets/:widgetId/data", handleWidgetData)

		// Widgets
		api.GET("/widgets", handleListWidgets)
		api.GET("/widgets/:type/preview", handleWidgetPreview)

		// Files
		api.GET("/files", handleListFiles)
		api.GET("/files/:id", handleGetFile)
		api.POST("/files", handleUploadFile)
		api.GET("/files/:id/download", handleDownloadFile)
		api.PUT("/files/:id", handleUpdateFileMetadata)
		api.DELETE("/files/:id", handleDeleteFile)
		api.GET("/files/:id/versions", handleFileVersions)

		// Permissions
		api.GET("/permissions/policies", handleListPolicies)
		api.POST("/permissions/policies", handleCreatePolicy)
		api.GET("/permissions/check", handleCheckPermission)
		api.GET("/permissions/roles", handleListRoles)
		api.POST("/permissions/roles", handleCreateRole)
		api.GET("/permissions/grants", handleListGrants)
		api.POST("/permissions/grants", handleCreateGrant)
		api.POST("/permissions/grants/:id/revoke", handleRevokeGrant)
		api.POST("/permissions/grants/:id/delegate", handleDelegateGrant)

		// Calendar
		api.GET("/calendar", handleListCalendarEvents)
		api.GET("/calendar/:id", handleGetCalendarEvent)
		api.POST("/calendar", handleCreateCalendarEvent)
		api.PUT("/calendar/:id", handleUpdateCalendarEvent)
		api.DELETE("/calendar/:id", handleDeleteCalendarEvent)

		// Reports
		api.GET("/reports", handleListReports)
		api.GET("/reports/:id", handleGetReport)
		api.POST("/reports", handleCreateReport)
		api.GET("/reports/:id/download", handleDownloadReport)
		api.DELETE("/reports/:id", handleDeleteReport)

		// AI Preparation
		api.GET("/ai/catalog", handleAICatalog)
		api.GET("/ai/catalog/:app", handleAICatalogByApp)
		api.GET("/ai/schema", handleAISchema)

		// API Governance
		api.GET("/apis", handleListAPIs)
		api.GET("/apis/health", handleAPIHealth)
		api.GET("/apis/audit", handleAuditLog)
		api.GET("/openapi.json", handleOpenAPISpec)

		// Monitoring
		api.GET("/monitoring/services", handleMonitoringServices)
		api.GET("/monitoring/metrics", handleMonitoringMetrics)
		api.GET("/monitoring/health", handleMonitoringHealth)

		// Enterprise Workspace (Phase III)
		api.GET("/workspace", handleWorkspace)
		api.GET("/workspace/notifications", handleListNotifications)
		api.GET("/workspace/timeline", handleTimeline)
		api.GET("/workspace/events", handleListEvents)
		api.GET("/workspace/dead-letter", handleDeadLetterQueue)
		api.GET("/workspace/project/:id", handleProjectProvisioningStatus)

		// Project File Provisioning (Phase III)
		api.GET("/files/project/:id", handleProjectFiles)

		// ── Phase IV: Enterprise Data Intelligence & Real-Time Analytics ──
		// Enterprise Data Layer
		api.GET("/analytics/records", handleListEnterpriseRecords)
		api.GET("/analytics/records/:source/:entity/:id", handleGetEnterpriseRecord)
		api.GET("/analytics/datasets", handleListDatasets)
		api.GET("/analytics/datasets/:id", handleDatasetDetail)
		api.GET("/analytics/freshness", handleDataFreshness)
		api.GET("/analytics/events", handleListAnalyticsEvents)

		// KPI Engine
		api.GET("/analytics/kpis", handleListKPIs)
		api.POST("/analytics/kpis", handleKPIDefinitionsCRUD)
		api.GET("/analytics/kpis/:id", handleGetKPI)
		api.GET("/analytics/kpis/:id/drilldown", handleKPIDrillDown)
		api.GET("/analytics/kpis/:id/refresh", handleRefreshKPI)
		api.POST("/analytics/kpis/:id/refresh", handleRefreshKPI)

		// Data Quality Intelligence
		api.GET("/analytics/quality", handleDataQualityReport)
		api.GET("/analytics/quality/reports", handleListDataQualityReports)
		api.GET("/analytics/quality/issues", handleDataQualityIssues)

		// Anomaly Detection
		api.GET("/analytics/anomalies/rules", handleListAnomalyRules)
		api.GET("/analytics/anomalies", handleListAnomalies)
		api.POST("/analytics/anomalies/run", handleRunAnomalyDetection)
		api.PUT("/analytics/anomalies/:id/resolve", handleResolveAnomaly)

		// Alert Engine
		api.GET("/analytics/alerts/rules", handleListAlertRules)
		api.POST("/analytics/alerts/rules", handleCreateAlertRule)
		api.GET("/analytics/alerts", handleListAlerts)
		api.GET("/analytics/alerts/:id", handleGetAlert)
		api.PUT("/analytics/alerts/:id/acknowledge", handleAcknowledgeAlert)
		api.PUT("/analytics/alerts/:id/resolve", handleResolveAlert)
		api.POST("/analytics/alerts/:id/actions", handleAlertActions)

		// Analytics Dashboards
		api.GET("/analytics/dashboards", handleListAnalyticsDashboards)
		api.GET("/analytics/dashboards/:id", handleGetAnalyticsDashboard)
		api.GET("/analytics/dashboards/:id/data", handleDashboardData)
		api.GET("/analytics/dashboard", handleDashboardData)

		// Cross-Application Intelligence
		api.GET("/analytics/intelligence", handleCrossAppIntelligence)

		// Enterprise Report Builder
		api.GET("/analytics/reports", handleListReportDefinitions)
		api.POST("/analytics/reports", handleCreateReportFromBuilder)
		api.GET("/analytics/reports/:id", handleGetReportDefinition)
		api.PUT("/analytics/reports/:id", handleUpdateReportDefinition)
		api.POST("/analytics/reports/:id/generate", handleGenerateReportFromDef)
		api.GET("/analytics/reports/runs", handleListReportRuns)
		api.GET("/analytics/reports/runs/:runId", handleGetReportRun)
		api.GET("/analytics/reports/runs/:runId/data", handleGetReportRunData)

		// Data Export
		api.POST("/analytics/exports", handleCreateExport)
		api.GET("/analytics/exports", handleListExports)
		api.GET("/analytics/exports/:id", handleGetExport)
		api.GET("/analytics/exports/:id/download", handleDownloadExport)

		// Analytics Search
		api.GET("/analytics/search", handleAnalyticsSearch)

		// Data Lineage
		api.GET("/analytics/lineage", handleLineageOverview)
		api.GET("/analytics/lineage/kpi/:id", handleKPILineage)
		api.GET("/analytics/lineage/dashboard/:id", handleDashboardLineage)
		api.GET("/analytics/lineage/record/:source/:entity/:id", handleRecordLineage)

		// AI-Ready Analytics
		api.GET("/analytics/ai/catalog", handleAIAnalyticsCatalog)
		api.GET("/analytics/ai/kpis", handleAIAnalyticsKPIs)
		api.GET("/analytics/ai/trends", handleAIAnalyticsTrends)
		api.GET("/analytics/ai/quality", handleAIAnalyticsQuality)
		api.GET("/analytics/ai/events", handleAIAnalyticsEvents)
		api.GET("/analytics/ai/reports", handleAIAnalyticsReports)
		api.GET("/analytics/ai/anomalies", handleAIAnalyticsAnomalies)
		api.GET("/analytics/ai/alerts", handleAIAnalyticsAlerts)
		api.GET("/analytics/ai/project-activity", handleAIAnalyticsProjectActivity)

		// Real-Time Analytics Stream (SSE)
		api.GET("/analytics/stream", handleAnalyticsStream)

		// ── Phase V: Enterprise Decision & Workflow Automation ──
		// Workflow Engine
		api.GET("/workflows", handleListWorkflowDefinitions)
		api.POST("/workflows", handleCreateWorkflowDefinition)
		api.GET("/workflows/:id", handleGetWorkflowDefinition)
		api.PUT("/workflows/:id", handleUpdateWorkflowDefinition)
		api.POST("/workflows/:id/activate", handleActivateWorkflow)
		api.POST("/workflows/:id/deactivate", handleDeactivateWorkflow)
		api.POST("/workflows/:id/run", handleRunWorkflowManual)

		// Workflow Instances & Execution
		api.GET("/workflows/instances", handleListWorkflowInstances)
		api.GET("/workflows/instances/:id", handleGetWorkflowInstance)
		api.POST("/workflows/instances/:id/cancel", handleCancelWorkflowInstance)
		api.POST("/workflows/instances/:id/retry", handleRetryWorkflowInstance)
		api.GET("/workflows/executions", handleWorkflowExecutions)

		// Workflow Dashboard & Process Analytics
		api.GET("/workflows/dashboard", handleWorkflowDashboard)
		api.GET("/workflows/process-analytics", handleProcessAnalytics)

		// Approval Engine
		api.GET("/approvals", handleListApprovals)
		api.GET("/approvals/:id", handleGetApproval)
		api.POST("/approvals", handleCreateApproval)
		api.POST("/approvals/:id/:action", handleApprovalAction)

		// Enterprise Tasks
		api.GET("/tasks", handleListTasks)
		api.GET("/tasks/:id", handleGetTask)
		api.POST("/tasks", handleCreateTask)
		api.PUT("/tasks/:id", handleUpdateTask)
		api.POST("/tasks/:id/:action", handleTaskAction)
		api.POST("/tasks/:id/comments", handleTaskComment)
		api.PUT("/tasks/:id/dependencies", handleTaskDependencies)
		api.DELETE("/tasks/:id", handleDeleteTask)

		// Escalation Engine
		api.GET("/escalations/policies", handleListEscalationPolicies)
		api.POST("/escalations/policies", handleCreateEscalationPolicy)

		// SLA Management
		api.GET("/slas", handleListSLAs)
		api.POST("/slas", handleCreateSLA)
		api.GET("/slas/breaches", handleListSLABreaches)
		api.POST("/slas/breaches/:id/resolve", handleResolveSLABreach)

		// Decision Records
		api.GET("/decisions", handleListDecisions)
		api.GET("/decisions/:id", handleGetDecision)
		api.POST("/decisions", handleCreateDecision)
		api.PUT("/decisions/:id", handleUpdateDecision)

		// AI Recommendations
		api.GET("/ai-recommendations", handleListAIRecommendations)
		api.POST("/ai-recommendations/:id/:action", handleAIRecommendationAction)

		// Action Centre & My Work
		api.GET("/action-centre", handleActionCentre)
		api.GET("/my-work", handleMyWork)

		// ─── Phase VI: Enterprise Knowledge & Collaboration ───
		// The knowledge layer indexes relationships to authoritative records; it
		// does not duplicate source application data or authorization services.
		api.GET("/knowledge", handleKnowledgeHome)
		api.GET("/knowledge/search", handleKnowledgeSearch)
		api.GET("/knowledge/context/:type/:id", handleEntityContext)
		api.POST("/knowledge/relationships", handleCreateKnowledgeRelationship)
		api.GET("/knowledge/articles", handleListKnowledgeArticles)
		api.POST("/knowledge/articles", handleCreateKnowledgeArticle)
		api.PUT("/knowledge/articles/:id", handleUpdateKnowledgeArticle)
		api.GET("/knowledge/data-dictionary", handleListDictionaryEntries)
		api.POST("/knowledge/data-dictionary", handleCreateDictionaryEntry)
		api.GET("/knowledge/ai-context", handleKnowledgeAIContext)
		api.GET("/knowledge/quality", handleKnowledgeQuality)

		// Phase VII: Enterprise AI & Decision Intelligence. These endpoints are
		// versioned and only expose permission-filtered Enterprise Core context.
		ai := api.Group("/ai/v1")
		{
			ai.POST("/context", handleAIContextV1)
			ai.POST("/query", handleAIQueryV1)
			ai.POST("/analyze", handleAIAnalyzeV1)
			ai.POST("/investigate", handleAIInvestigateV1)
			ai.GET("/investigations", handleAIInvestigationsV1)
			ai.POST("/investigations", handleAICreateInvestigationV1)
			ai.GET("/investigations/:id", handleAIInvestigationV1)
			ai.PUT("/investigations/:id", handleAIUpdateInvestigationV1)
			ai.POST("/investigations/:id/decisions", handleAIInvestigationDecisionV1)
			ai.POST("/recommend", handleAIRecommendV1)
			ai.GET("/sources/:id", handleAISourcesV1)
			ai.POST("/feedback", handleAIFeedbackV1)
			ai.GET("/evaluations", handleAIEvaluationsV1)
			ai.GET("/models", handleAIModelsV1)
			ai.GET("/usage", handleAIUsageV1)
			ai.GET("/briefings", handleAIBriefingsV1)
		}
	}
}

// ─── Event Handlers ───────────────────────────────────────────────

func handlePublishEvent(c *gin.Context) {
	var ev DomainEvent
	if err := c.ShouldBindJSON(&ev); err != nil {
		c.JSON(400, gin.H{"error": "invalid event", "detail": err.Error()})
		return
	}
	publishEvent(ev)
	recordAudit("event.publish", ev.Source, ev.Actor, map[string]interface{}{
		"event_type": ev.EventType, "object_id": ev.ObjectID,
	})
	c.JSON(201, gin.H{"id": ev.ID, "status": "published", "timestamp": ev.Timestamp})
}

func handleListEvents(c *gin.Context) {
	if redisClient == nil {
		c.JSON(200, gin.H{"events": []interface{}{}, "count": 0})
		return
	}
	limit := parseIntDefault(c.Query("limit"), 100)
	ctx := context.Background()
	raw, err := redisClient.LRange(ctx, "statgate:event:history", 0, int64(limit-1)).Result()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to list events"})
		return
	}
	events := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		var ev map[string]interface{}
		if err := json.Unmarshal([]byte(item), &ev); err == nil {
			events = append(events, ev)
		}
	}
	c.JSON(200, gin.H{"count": len(events), "events": events})
}

func handleEventTypes(c *gin.Context) {
	c.JSON(200, gin.H{"event_types": []string{
		"user.created", "user.updated",
		"project.created", "project.closed", "project.updated", "project.milestone",
		"survey.created", "survey.published", "survey.submitted",
		"dataset.created", "dataset.updated",
		"submission.received", "submission.approved",
		"field_data.submitted",
		"research.approved", "research.created", "research.stage_changed",
		"meeting.scheduled", "meeting.cancelled",
		"ticket.raised", "ticket.assigned", "ticket.closed", "ticket.updated",
		"report.generated", "dashboard.refreshed",
		"file.uploaded", "file.updated",
		"approval.requested", "approval.approved", "approval.rejected",
	}})
}

func handleGetEvent(c *gin.Context) {
	id := c.Param("id")
	if redisClient != nil {
		raw, _ := redisClient.LRange(context.Background(), "statgate:event:history", 0, -1).Result()
		for _, item := range raw {
			var ev map[string]interface{}
			if err := json.Unmarshal([]byte(item), &ev); err == nil {
				if ev["id"] == id {
					c.JSON(200, ev)
					return
				}
			}
		}
	}
	c.JSON(404, gin.H{"error": "event not found"})
}

func handleListSubscriptions(c *gin.Context) {
	if redisClient == nil {
		c.JSON(200, gin.H{"subscriptions": []interface{}{}})
		return
	}
	raw, _ := redisClient.LRange(context.Background(), "statgate:subscriptions", 0, -1).Result()
	subs := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		var s map[string]interface{}
		if err := json.Unmarshal([]byte(item), &s); err == nil {
			subs = append(subs, s)
		}
	}
	c.JSON(200, gin.H{"subscriptions": subs})
}

func handleCreateSubscription(c *gin.Context) {
	var sub map[string]interface{}
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(400, gin.H{"error": "invalid subscription"})
		return
	}
	if _, ok := sub["id"]; !ok {
		sub["id"] = fmt.Sprintf("sub_%d", time.Now().UnixNano())
	}
	if redisClient != nil {
		data, _ := json.Marshal(sub)
		redisClient.RPush(context.Background(), "statgate:subscriptions", string(data))
	}
	recordAudit("subscription.create", "enterprise", fmt.Sprintf("%v", sub["created_by"]), sub)
	c.JSON(201, sub)
}
