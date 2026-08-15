package main

// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•
// PHASE XII â€” INTELLIGENCE API HANDLERS
// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleIntelligenceOverview returns the decision-oriented institutional
// intelligence snapshot (what is happening / why / how serious / evidence).
func handleIntelligenceOverview(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required", "message": "Authenticated tenant context is missing (fail closed)."})
		return
	}
	cond := phase12.GetCurrentCondition(tenantID)
	signals := phase12.ListSignals(tenantID)
	riskEvents := phase12.ListRiskEvents(tenantID)
	recs := phase12.ListRecommendations(tenantID)

	resp := gin.H{
		"tenant_id": tenantID,
		"condition": cond,
		"signals": gin.H{
			"count":   len(signals),
			"signals": signals,
		},
		"risk_events": gin.H{
			"count": len(riskEvents),
		},
		"ai_recommendations": gin.H{
			"count":  len(recs),
			"pending": countPendingRecommendations(recs),
		},
		"metrics": PhaseXIIMetrics(),
	}
	c.JSON(http.StatusOK, resp)
}

func countPendingRecommendations(recs []*IntelligenceRecommendation) int {
	n := 0
	for _, r := range recs {
		if r.Status == AIStatusPendingReview {
			n++
		}
	}
	return n
}

// handleGetCondition returns the current institutional condition with full
// explainability ("Why is the institution currently ELEVATED?").
func handleGetCondition(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	cond := phase12.GetCurrentCondition(tenantID)
	if cond == nil {
		c.JSON(http.StatusOK, gin.H{
			"condition_level":   "UNKNOWN",
			"composite_score":   0,
			"explanation":       "NO DATA: no institutional condition has been calculated for this tenant.",
			"calculation_version": conditionCalculationVersion,
		})
		return
	}
	c.JSON(http.StatusOK, cond)
}

func handleGetConditionHistory(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tenant_id":  tenantID,
		"count":      len(phase12.GetConditionHistory(tenantID)),
		"conditions": phase12.GetConditionHistory(tenantID),
	})
}

func handleListSignals(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	signals := phase12.ListAllSignals(tenantID)
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "count": len(signals), "signals": signals})
}

// handleIngestSignal ingests an intelligence signal. Privileged: admin /
// institutional_lead. The condition is recomputed deterministically.
func handleIngestSignal(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	var sig IntelligenceSignal
	if err := c.ShouldBindJSON(&sig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_signal", "message": err.Error()})
		return
	}
	sig.TenantID = tenantID
	if err := phase12.IngestSignal(&sig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "signal_rejected", "message": err.Error()})
		return
	}
	recordAuditFromContext(c, "intelligence.signal.ingested", "intelligence_signals", sig.ID, map[string]interface{}{
		"signal_type": sig.SignalType,
		"domain":      sig.Domain,
		"severity":    sig.Severity,
	})
	c.JSON(http.StatusCreated, gin.H{"id": sig.ID, "status": "signal_ingested", "tenant_id": tenantID})
}
