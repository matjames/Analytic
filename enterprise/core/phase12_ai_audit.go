package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — AI AUDIT & PERSISTENCE
//
// Dedicated ai_audit_log linked to the platform audit system via
// correlation_id / request_id / actor (directive §20). Sensitive prompts are
// never stored — only classifications and structured outputs.
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// recordAIAudit writes a dedicated AI audit entry.
func (s *Phase12Store) recordAIAudit(rec *IntelligenceRecommendation, authorizationStatus, actor, decision string) {
	entry := &AIAuditEntry{
		Provider:             rec.Provider,
		Model:                rec.Model,
		RequestID:            rec.RequestID,
		CorrelationID:        rec.CorrelationID,
		InputClassification:  rec.InputClassification,
		OutputClassification: rec.OutputClassification,
		RecommendationID:     rec.ID,
		Recommendation:       rec.Recommendation,
		Confidence:           rec.Confidence,
		Timestamp:            utcNow(),
		Actor:                actor,
		TenantID:             rec.TenantID,
		AuthorizationStatus:  authorizationStatus,
		HumanDecision:        decision,
	}
	s.mu.Lock()
	s.aiAudit = append(s.aiAudit, entry)
	s.mu.Unlock()
	persistPhase12AIAudit(entry)
}

// ListAIAudit returns the tenant-scoped AI audit trail.
func (s *Phase12Store) ListAIAudit(tenantID string) []*AIAuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*AIAuditEntry
	for _, e := range s.aiAudit {
		if e.TenantID == tenantID {
			out = append(out, e)
		}
	}
	return out
}

func persistPhase12Recommendation(rec *IntelligenceRecommendation) {
	if dbPool == nil {
		return
	}
	sevJSON, _ := json.Marshal(rec.SupportingEvidence)
	affJSON, _ := json.Marshal(rec.AffectedObjects)
	actJSON, _ := json.Marshal(rec.RecommendedActions)
	limJSON, _ := json.Marshal(rec.Limitations)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx, `INSERT INTO ai_recommendations
		(id, tenant_id, provider, model, request_id, correlation_id, status,
		 recommendation, confidence, reasoning_summary, supporting_evidence,
		 affected_objects, risk_level, recommended_actions, limitations,
		 input_classification, output_classification, ai_label, generated_by,
		 reviewed_by, reviewed_at, human_decision, executed_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12::jsonb,$13,$14::jsonb,
		        $15::jsonb,$16,$17,$18,$19,$20,$21,$22,$23,$24)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status, reviewed_by = EXCLUDED.reviewed_by,
			reviewed_at = EXCLUDED.reviewed_at, human_decision = EXCLUDED.human_decision,
			executed_at = EXCLUDED.executed_at`,
		rec.ID, rec.TenantID, rec.Provider, rec.Model, rec.RequestID, rec.CorrelationID, rec.Status,
		rec.Recommendation, rec.Confidence, rec.ReasoningSummary, string(sevJSON), string(affJSON),
		rec.RiskLevel, string(actJSON), string(limJSON),
		rec.InputClassification, rec.OutputClassification, rec.AILabel, rec.GeneratedBy,
		rec.ReviewedBy, rec.ReviewedAt, rec.HumanDecision, rec.ExecutedAt, rec.CreatedAt)
	if err != nil {
		log.Printf("phase12: persist recommendation failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()
}

func persistPhase12AIAudit(e *AIAuditEntry) {
	if dbPool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx, `INSERT INTO ai_audit_log
		(provider, model, request_id, correlation_id, input_classification,
		 output_classification, recommendation_id, recommendation, confidence,
		 timestamp, actor, tenant_id, authorization_status, human_decision)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		e.Provider, e.Model, e.RequestID, e.CorrelationID, e.InputClassification,
		e.OutputClassification, e.RecommendationID, e.Recommendation, e.Confidence,
		e.Timestamp, e.Actor, e.TenantID, e.AuthorizationStatus, e.HumanDecision)
	if err != nil {
		log.Printf("phase12: persist ai audit failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()
}
