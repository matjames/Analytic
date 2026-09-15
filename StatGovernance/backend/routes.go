package main

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {
	// Apply StatGate Registry JWT Authentication middleware
	r.Use(registryAuthMiddleware())

	api := r.Group("/api")
	api.POST("/discussions", dbCreateDiscussion)
	{
		// ─── Dashboard ────────────────────────────────────────────
		api.GET("/dashboard", dbGetDashboard)

		// ─── Policy Management ────────────────────────────────────
		api.GET("/policies", dbGetPolicies)
		api.POST("/policies", dbCreatePolicy)
		api.GET("/policies/:id", dbGetPolicyByID)
		api.PUT("/policies/:id", dbUpdatePolicy)
		api.POST("/policies/:id/approve", dbApprovePolicy)
		api.POST("/policies/:id/publish", dbPublishPolicy)
		api.GET("/policies/:id/versions", dbGetPolicyVersions)

		// ─── SOP Management ───────────────────────────────────────
		api.GET("/sops", dbGetSOPs)
		api.POST("/sops", dbCreateSOP)

		// ─── Regulatory & Compliance Register ─────────────────────
		api.GET("/regulations", dbGetRegulations)
		api.POST("/regulations", dbCreateRegulation)
		api.GET("/compliance/obligations", dbGetObligations)
		api.POST("/compliance/obligations", dbCreateObligation)
		api.GET("/compliance/assessments", dbGetAssessments)
		api.POST("/compliance/assessments", dbCreateAssessment)

		// ─── Risk Management ──────────────────────────────────────
		api.GET("/risks", dbGetRisks)
		api.POST("/risks", dbCreateRisk)
		api.POST("/risks/:id/escalate", dbEscalateRisk)

		// ─── Controls Library ─────────────────────────────────────
		api.GET("/controls", dbGetControls)
		api.POST("/controls", dbCreateControl)
		api.POST("/controls/:id/test", dbTestControl)

		// ─── Audits & Findings ────────────────────────────────────
		api.GET("/audits", dbGetAudits)
		api.POST("/audits", dbCreateAudit)
		api.GET("/findings", dbGetFindings)
		api.POST("/findings", dbCreateFinding)
		api.POST("/findings/:id/close", dbCloseFinding)

		// ─── Corrective & Preventive Actions (CAPA) ───────────────
		api.GET("/corrective-actions", dbGetCorrectiveActions)
		api.POST("/corrective-actions", dbCreateCorrectiveAction)

		// ─── Committees & Meetings ────────────────────────────────
		api.GET("/committees", dbGetCommittees)
		api.POST("/committees", dbCreateCommittee)
		api.GET("/meetings", dbGetMeetings)
		api.POST("/meetings", dbCreateMeeting)

		// ─── Decisions ────────────────────────────────────────────
		api.GET("/decisions", dbGetDecisions)
		api.POST("/decisions", dbCreateDecision)

		// ─── Evidence Vault ───────────────────────────────────────
		api.GET("/evidence", dbGetEvidence)
		api.POST("/evidence", dbCreateEvidence)

		// ─── Data Governance & Privacy ────────────────────────────
		api.GET("/data-governance", dbGetDataGovernance)
		api.POST("/data-governance", dbCreateDataGovernance)
		api.GET("/data-governance/privacy", dbGetPrivacyAssessments)

		// ─── Delegations ──────────────────────────────────────────
		api.GET("/delegations", dbGetDelegations)
		api.POST("/delegations", dbCreateDelegation)

		// ─── Whistleblower & Integrity ────────────────────────────
		api.GET("/whistleblower", dbGetWhistleblowerReports)
		api.POST("/whistleblower", dbSubmitWhistleblowerReport)

		// ─── Conflict of Interest (COI) ───────────────────────────
		api.GET("/conflict-declarations", dbGetConflictDeclarations)
		api.POST("/conflict-declarations", dbSubmitConflictDeclaration)

		// ─── Feature Flags & Parameters ───────────────────────────
		api.GET("/feature-flags", dbGetFeatureFlags)
		api.POST("/feature-flags", dbCreateFeatureFlag)
		api.GET("/system-parameters", dbGetSystemParameters)
		api.POST("/system-parameters", dbSaveSystemParameter)

		// ─── Cross-Application Search & Activity Timeline ─────────
		api.GET("/search", dbSearch)
		api.GET("/activity", dbGetActivity)

		// ─── Grounded AI Analysis ─────────────────────────────────
		api.POST("/ai/analyze", dbAIAnalyze)
	}
}
