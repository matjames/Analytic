package main

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Enterprise integration middleware
	r.Use(registryAuthMiddleware())
	r.Use(workspaceResearchMiddleware())
	r.Use(workspaceResearchBodyMiddleware())
	r.Use(workspaceResearchChildIDMiddleware())

	api := r.Group("/api")
	{
		// ─── Research Projects ────────────────────────────────────
		api.GET("/research", dbGetResearchProjects)
		api.POST("/research", dbCreateResearch)
		api.GET("/research/:id", dbGetResearchWorkspace)
		api.PUT("/research/:id", dbUpdateResearch)
		api.DELETE("/research/:id", dbDeleteResearch)
		api.PUT("/research/:id/stage", dbUpdateResearchStage)
		api.GET("/research/:id/dashboard", dbGetResearchDashboard)
		api.GET("/research/:id/audit", dbGetAuditLogs)
		api.GET("/research/:id/assistant", dbGetResearchAssistant)
		api.POST("/research/:id/assistant", dbPostResearchAssistant)
		api.GET("/research/:id/publication-readiness", dbGetPublicationReadiness)
		api.GET("/research/:id/quality-review", dbGetResearchQualityReview)
		api.POST("/research/:id/language-review", dbPostResearchLanguageReview)
		api.GET("/research/:id/statistical-recommendations", dbGetStatisticalRecommendations)
		api.POST("/research/:id/statistical-recommendations", dbPostStatisticalRecommendations)
		api.POST("/research/:id/statistical-recommendations/run", dbRunStatisticalTabulation)
		api.GET("/research/:id/statcollect-links", dbGetStatCollectLinks)
		api.POST("/research/:id/statcollect-links", dbCreateStatCollectLink)
		api.DELETE("/research/:id/statcollect-links/:linkId", dbDeleteStatCollectLink)
		api.POST("/statcollect/ingest", dbIngestStatCollectEvent)

		// ─── Members ──────────────────────────────────────────────
		api.POST("/members", dbAddMember)
		api.PUT("/members/:id", dbUpdateMember)
		api.DELETE("/members/:id", dbRemoveMember)

		// ─── Proposals ────────────────────────────────────────────
		api.GET("/research/:id/proposals", dbGetProposals)
		api.POST("/proposals", dbCreateProposal)
		api.PUT("/proposals/:id", dbUpdateProposal)
		api.DELETE("/proposals/:id", dbDeleteProposal)

		// ─── Ethics & IRB ─────────────────────────────────────────
		api.GET("/research/:id/ethics", dbGetEthics)
		api.POST("/ethics", dbCreateEthics)
		api.PUT("/ethics/:id", dbUpdateEthics)
		api.DELETE("/ethics/:id", dbDeleteEthics)
		api.GET("/ethics-committees", dbGetEthicsCommittees)
		api.POST("/ethics-committees", dbCreateEthicsCommittee)
		api.PUT("/ethics-committees/:id", dbUpdateEthicsCommittee)
		api.DELETE("/ethics-committees/:id", dbDeleteEthicsCommittee)
		api.GET("/ethics-committees/:id/members", dbGetEthicsCommitteeMembers)
		api.POST("/ethics-committees/:id/members", dbInviteEthicsCommitteeMember)
		api.PUT("/ethics-committee-members/:id", dbUpdateEthicsCommitteeMember)
		api.DELETE("/ethics-committee-members/:id", dbDeleteEthicsCommitteeMember)

		// ─── Grants / Funding ─────────────────────────────────────
		api.GET("/research/:id/grants", dbGetGrants)
		api.POST("/grants", dbCreateGrant)
		api.PUT("/grants/:id", dbUpdateGrant)
		api.DELETE("/grants/:id", dbDeleteGrant)

		// ─── Literature & Citations ───────────────────────────────
		api.GET("/research/:id/literature", dbGetLiterature)
		api.POST("/literature", dbCreateLiterature)
		api.PUT("/literature/:id", dbUpdateLiterature)
		api.DELETE("/literature/:id", dbDeleteLiterature)
		api.POST("/citation/format", dbFormatCitation)
		api.POST("/citations/format", dbFormatCitation)

		// ─── Datasets ─────────────────────────────────────────────
		api.GET("/research/:id/datasets", dbGetDatasets)
		api.POST("/datasets", dbCreateDataset)
		api.PUT("/datasets/:id", dbUpdateDataset)
		api.DELETE("/datasets/:id", dbDeleteDataset)

		// ─── Publications & DOI ───────────────────────────────────
		api.GET("/research/:id/publications", dbGetPublications)
		api.POST("/publications", dbCreatePublication)
		api.PUT("/publications/:id", dbUpdatePublication)
		api.DELETE("/publications/:id", dbDeletePublication)
		api.GET("/research/:id/submissions", dbGetJournalSubmissions)
		api.POST("/submissions", dbCreateJournalSubmission)
		api.PUT("/submissions/:id", dbUpdateJournalSubmission)
		api.DELETE("/submissions/:id", dbDeleteJournalSubmission)
		api.GET("/research/:id/dois", dbGetDOIRecords)
		api.POST("/dois", dbCreateDOIRecord)
		api.PUT("/dois/:id", dbUpdateDOIRecord)
		api.DELETE("/dois/:id", dbDeleteDOIRecord)
		api.POST("/dois/:id/register", dbRegisterDOIRecord)

		// ─── Open Science Repository ──────────────────────────────
		api.GET("/open-access", dbGetOpenAccessRepo)
		api.GET("/research/:id/open-access", dbGetOpenAccessRepo)
		api.POST("/open-access", dbCreateOpenAccessRepo)
		api.PUT("/open-access/:id", dbUpdateOpenAccessRepo)
		api.DELETE("/open-access/:id", dbDeleteOpenAccessRepo)
		api.GET("/research/:id/archives", dbGetResearchArchives)
		api.POST("/archives", dbCreateResearchArchive)
		api.PUT("/archives/:id", dbUpdateResearchArchive)
		api.DELETE("/archives/:id", dbDeleteResearchArchive)
		api.GET("/research/:id/conferences", dbGetConferenceSubmissions)
		api.POST("/conferences", dbCreateConferenceSubmission)
		api.PUT("/conferences/:id", dbUpdateConferenceSubmission)
		api.DELETE("/conferences/:id", dbDeleteConferenceSubmission)
		api.GET("/research/:id/conference-events", dbGetConferenceEvents)
		api.POST("/conference-events", dbCreateConferenceEvent)
		api.PUT("/conference-events/:id", dbUpdateConferenceEvent)
		api.DELETE("/conference-events/:id", dbDeleteConferenceEvent)
		api.GET("/conference-events/:id/attendance", dbGetConferenceAttendance)
		api.POST("/conference-events/:id/attendance", dbRegisterConferenceAttendance)
		api.PUT("/conference-attendance/:id", dbUpdateConferenceAttendance)
		api.DELETE("/conference-attendance/:id", dbDeleteConferenceAttendance)
		api.GET("/research/:id/knowledge-transfers", dbGetKnowledgeTransfers)
		api.POST("/knowledge-transfers", dbCreateKnowledgeTransfer)
		api.PUT("/knowledge-transfers/:id", dbUpdateKnowledgeTransfer)
		api.DELETE("/knowledge-transfers/:id", dbDeleteKnowledgeTransfer)
		api.POST("/knowledge-transfers/:id/publish", dbPublishKnowledgeTransfer)

		// ─── Tasks ────────────────────────────────────────────────
		api.POST("/tasks", dbCreateTask)
		api.PUT("/tasks/:id", dbUpdateTask)
		api.DELETE("/tasks/:id", dbDeleteTask)

		// ─── Meetings ─────────────────────────────────────────────
		api.POST("/meetings", dbCreateMeeting)
		api.PUT("/meetings/:id", dbUpdateMeeting)

		// ─── Risks ────────────────────────────────────────────────
		api.POST("/risks", dbCreateRisk)
		api.PUT("/risks/:id", dbUpdateRisk)
		api.DELETE("/risks/:id", dbDeleteRisk)

		// ─── Issues ──────────────────────────────────────────────
		api.GET("/research/:id/issues", dbGetIssues)
		api.POST("/issues", dbCreateIssue)
		api.PUT("/issues/:id", dbUpdateIssue)
		api.DELETE("/issues/:id", dbDeleteIssue)

		// ─── Documents ───────────────────────────────────────────
		api.GET("/research/:id/documents", dbGetDocuments)
		api.POST("/documents", dbCreateDocument)
		api.DELETE("/documents/:id", dbDeleteDocument)

		// ─── Surveys ─────────────────────────────────────────────
		api.GET("/research/:id/surveys", dbGetSurveys)
		api.POST("/surveys", dbCreateSurvey)
		api.PUT("/surveys/:id", dbUpdateSurvey)
		api.DELETE("/surveys/:id", dbDeleteSurvey)

		// ─── Reports ─────────────────────────────────────────────
		api.GET("/research/:id/reports", dbGetReports)
		api.POST("/reports", dbCreateReport)
		api.DELETE("/reports/:id", dbDeleteReport)

		// ─── Calendar Events ─────────────────────────────────────
		api.GET("/research/:id/calendar", dbGetCalendarEvents)
		api.POST("/calendar", dbCreateCalendarEvent)
		api.PUT("/calendar/:id", dbUpdateCalendarEvent)
		api.DELETE("/calendar/:id", dbDeleteCalendarEvent)

		// ─── Chat ─────────────────────────────────────────────────
		api.POST("/chats", dbPostChatMessage)

		// ─── Global Dashboard ─────────────────────────────────────
		api.GET("/dashboard", dbGetDashboard)

		// ─── Search ───────────────────────────────────────────────
		api.GET("/search", dbSearch)

		// ─── Enterprise Activity Timeline ─────────────────────────
		api.GET("/activity", dbGetActivityTimeline)
	}
}
