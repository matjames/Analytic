package main

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.Use(registryAuthMiddleware())
	// API routes
	api := r.Group("/api")
	{
		// ─── Projects ─────────────────────────────────────────────
		api.GET("/projects", dbGetProjects)
		api.POST("/projects", dbCreateProject)
		api.GET("/projects/:id", dbGetProjectWorkspace)
		api.PUT("/projects/:id", dbUpdateProject)
		api.DELETE("/projects/:id", dbDeleteProject)
		api.PUT("/projects/:id/stage", dbUpdateProjectStage)
		api.GET("/projects/:id/audit", dbGetProjectAuditLogs)
		api.GET("/projects/:id/calendar", dbGetProjectCalendar)
		api.GET("/projects/:id/reports", dbGetProjectReports)
		api.POST("/projects/:id/reports", dbGenerateReport)
		api.GET("/projects/:id/relationships", dbGetProjectRelationships)

		// ─── Enterprise Hierarchy ────────────────────────────────
		api.GET("/organizations", dbGetOrganizations)
		api.POST("/organizations", dbCreateOrganization)
		api.GET("/portfolios", dbGetPortfolios)
		api.POST("/portfolios", dbCreatePortfolio)
		api.GET("/programmes", dbGetProgrammes)
		api.POST("/programmes", dbCreateProgramme)
		api.GET("/projects/:id/components", dbGetComponents)
		api.POST("/components", dbCreateComponent)
		api.GET("/projects/:id/activities", dbGetActivities)
		api.POST("/activities", dbCreateActivity)
		api.GET("/projects/:id/deliverables", dbGetDeliverables)
		api.POST("/deliverables", dbCreateDeliverable)
		api.GET("/projects/:id/milestones", dbGetMilestones)
		api.POST("/milestones", dbCreateMilestone)

		// ─── Tasks ───────────────────────────────────────────────
		api.POST("/tasks", dbCreateTask)
		api.PUT("/tasks/:id", dbUpdateTask)
		api.DELETE("/tasks/:id", dbDeleteTask)
		api.PUT("/tasks/:id/status", dbUpdateTaskStatus)

		// ─── Team Members ────────────────────────────────────────
		api.POST("/members", dbAddMember)
		api.DELETE("/members/:id", dbRemoveMember)

		// ─── Budget & Finance ────────────────────────────────────
		api.POST("/budgets", dbCreateBudgetLine)
		api.PUT("/budgets/:id", dbUpdateBudgetLine)
		api.DELETE("/budgets/:id", dbDeleteBudgetLine)
		api.GET("/projects/:id/funding", dbGetFundingSources)
		api.POST("/funding", dbCreateFundingSource)
		api.GET("/projects/:id/cost-centres", dbGetCostCentres)
		api.POST("/cost-centres", dbCreateCostCentre)
		api.GET("/projects/:id/budget-revisions", dbGetBudgetRevisions)
		api.POST("/budget-revisions", dbCreateBudgetRevision)
		api.GET("/projects/:id/procurement", dbGetProcurementRefs)
		api.POST("/procurement", dbCreateProcurementRef)

		// ─── Risks & Issues ──────────────────────────────────────
		api.POST("/risks", dbCreateRisk)
		api.PUT("/risks/:id", dbUpdateRisk)
		api.DELETE("/risks/:id", dbDeleteRisk)
		api.GET("/projects/:id/issues", dbGetIssues)
		api.POST("/issues", dbCreateIssue)
		api.PUT("/issues/:id", dbUpdateIssue)
		api.GET("/projects/:id/assumptions", dbGetAssumptions)
		api.POST("/assumptions", dbCreateAssumption)
		api.GET("/projects/:id/lessons", dbGetLessonsLearned)
		api.POST("/lessons", dbCreateLessonLearned)
		api.GET("/projects/:id/corrective-actions", dbGetCorrectiveActions)
		api.POST("/corrective-actions", dbCreateCorrectiveAction)

		// ─── Documents ───────────────────────────────────────────
		api.POST("/documents", dbCreateDocument)
		api.DELETE("/documents/:id", dbDeleteDocument)

		// ─── Meetings ────────────────────────────────────────────
		api.POST("/meetings", dbCreateMeeting)
		api.PUT("/meetings/:id", dbUpdateMeeting)

		// ─── Surveys ─────────────────────────────────────────────
		api.POST("/surveys", dbCreateSurvey)
		api.PUT("/surveys/:id", dbUpdateSurvey)

		// ─── Chat ────────────────────────────────────────────────
		api.POST("/chats", dbPostChatMessage)

		// ─── HelpDesk ────────────────────────────────────────────
		api.POST("/helpdesk", dbCreateHelpDeskTicket)
		api.PUT("/helpdesk/:id", dbUpdateHelpDeskTicket)

		// ─── Workflow & Permissions ──────────────────────────────
		api.GET("/workflow-rules", dbGetWorkflowRules)
		api.POST("/workflow-rules", dbCreateWorkflowRule)
		api.GET("/permissions", dbGetPermissions)
		api.POST("/permissions", dbCreatePermission)

		// ─── Search ──────────────────────────────────────────────
		api.GET("/search", dbSearch)

		// ─── Dashboard / Analytics ───────────────────────────────
		api.GET("/dashboard", dbGetDashboard)
		api.GET("/projects/:id/dashboard", dbGetProjectDashboard)

		// ─── Enterprise Activity Timeline ────────────────────────
		api.GET("/activity", dbGetActivityTimeline)
	}
}
