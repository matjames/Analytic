package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — RISK INTELLIGENCE & GOVERNED AI API HANDLERS
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ─── Risk Intelligence ────────────────────────────────────────────────────────

func handleListRiskEvents(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	events := phase12.ListRiskEvents(tenantID)
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "count": len(events), "events": events})
}

// handleRecordRiskEvent records a risk-intelligence correlation event.
// Privileged. This is NOT a risk register — it is correlation intelligence.
func handleRecordRiskEvent(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	var ev RiskEvent
	if err := c.ShouldBindJSON(&ev); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_risk_event", "message": err.Error()})
		return
	}
	ev.TenantID = tenantID
	if err := phase12.RecordRiskEvent(&ev); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "risk_event_rejected", "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": ev.ID, "status": "risk_event_recorded"})
}

// handleDetectEmergingRisk runs the emerging-risk detector over the tenant's
// risk events. It creates an intelligence SIGNAL, never an authoritative risk.
func handleDetectEmergingRisk(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	events := phase12.ListRiskEvents(tenantID)
	sig, err := phase12.DetectEmergingRisk(tenantID, events)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "detection_failed", "message": err.Error()})
		return
	}
	if sig == nil {
		c.JSON(http.StatusOK, gin.H{"emerging_risk": false, "message": "No emerging risk threshold reached."})
		return
	}
	recordAuditFromContext(c, "risk.emerging.detected", "intelligence_signals", sig.ID, map[string]interface{}{
		"severity": sig.Severity,
	})
	c.JSON(http.StatusOK, gin.H{"emerging_risk": true, "signal_id": sig.ID, "severity": sig.Severity})
}

// ─── Governed AI ──────────────────────────────────────────────────────────────

// handleGenerateRecommendation runs the reasoning provider and creates a
// PENDING_REVIEW recommendation. AI can never execute anything itself.
func handleGenerateRecommendation(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	var input AIInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_ai_input", "message": err.Error()})
		return
	}
	corrID, _ := c.Get("correlation_id")
	corrStr, _ := corrID.(string)
	rec, err := phase12.GenerateRecommendation(tenantID, getContextUserID(c), corrStr, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recommendation_failed", "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rec)
}

func handleListRecommendations(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	recs := phase12.ListRecommendations(tenantID)
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "count": len(recs), "recommendations": recs})
}

// handleReviewRecommendation is the human review gate. Only admin /
// institutional_lead may authorize or reject (directive §19).
func handleReviewRecommendation(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	recID := c.Param("id")
	var body struct {
		Decision string `json:"decision"`
		Approve  bool   `json:"approve"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_review", "message": err.Error()})
		return
	}
	if err := phase12.ReviewRecommendation(tenantID, recID, getContextUserID(c), body.Decision, body.Approve); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "review_failed", "message": err.Error()})
		return
	}
	status := "rejected"
	if body.Approve {
		status = "authorized"
	}
	c.JSON(http.StatusOK, gin.H{"id": recID, "status": status})
}

// handleExecuteRecommendation marks an AUTHORIZED recommendation EXECUTED by
// a human operator. AI never executes itself.
func handleExecuteRecommendation(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	recID := c.Param("id")
	if err := phase12.ExecuteRecommendation(tenantID, recID, getContextUserID(c)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "execute_failed", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": recID, "status": "executed"})
}

func handleListAIAudit(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	audit := phase12.ListAIAudit(tenantID)
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "count": len(audit), "audit": audit})
}