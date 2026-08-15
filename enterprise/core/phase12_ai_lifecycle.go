package main

// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•
// PHASE XII â€” AI RECOMMENDATION LIFECYCLE
//
// GENERATED -> PENDING_REVIEW -> AUTHORIZED|REJECTED -> EXECUTED|CANCELLED
// No AI recommendation may skip the human review stage (directive Â§19).
// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•

import (
	"fmt"
	"time"
)

// GenerateRecommendation runs the reasoning provider, stores the output as a
// GENERAL recommendation, then immediately submits it for human review
// (PENDING_REVIEW). Returns the recommendation record.
func (s *Phase12Store) GenerateRecommendation(tenantID, actor, correlationID string, input AIInput) (*IntelligenceRecommendation, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("recommendation requires tenant_id")
	}
	input.TenantID = tenantID
	if input.CorrelationID == "" {
		input.CorrelationID = correlationID
	}
	if input.DataClassification == "" {
		input.DataClassification = "INTERNAL"
	}

	provider := GetReasoningProvider(tenantID)
	if !dataClassificationAllowedForProvider(input.DataClassification) {
		return nil, fmt.Errorf("data classification %q is not permitted for AI reasoning (fail closed)", input.DataClassification)
	}

	out, err := provider.Generate(input)
	if err != nil {
		return nil, fmt.Errorf("reasoning provider failed: %w", err)
	}
	if out == nil {
		return nil, fmt.Errorf("reasoning provider returned no output")
	}
	rec := &IntelligenceRecommendation{
		ID:                   fmt.Sprintf("airec_%d", time.Now().UnixNano()),
		TenantID:             tenantID,
		Provider:             provider.Name(),
		Model:                "structured-output-v1",
		RequestID:            fmt.Sprintf("req_%d", time.Now().UnixNano()),
		CorrelationID:        input.CorrelationID,
		Status:               AIStatusGenerated,
		Recommendation:       out.Recommendation,
		Confidence:           out.Confidence,
		ReasoningSummary:     out.ReasoningSummary,
		SupportingEvidence:   out.SupportingEvidence,
		AffectedObjects:      out.AffectedObjects,
		RiskLevel:            out.RiskLevel,
		RecommendedActions:   out.RecommendedActions,
		Limitations:          out.Limitations,
		InputClassification:  input.DataClassification,
		OutputClassification: input.DataClassification,
		AILabel:              "AI_GENERATED",
		GeneratedBy:          actor,
		CreatedAt:            utcNow(),
	}

	s.mu.Lock()
	s.recommendations[objectKey(tenantID, rec.ID)] = rec
	s.mu.Unlock()
	statgateAIRecommendations.Add(1)

	if err := s.transitionRecommendation(tenantID, rec.ID, AIStatusPendingReview, actor, ""); err != nil {
		return nil, err
	}
	s.recordAIAudit(rec, AIStatusPendingReview, actor, "")
	return rec, nil
}

// transitionRecommendation applies a validated state transition.
func (s *Phase12Store) transitionRecommendation(tenantID, recID, to, actor, decision string) error {
	key := objectKey(tenantID, recID)
	s.mu.Lock()
	rec, ok := s.recommendations[key]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("recommendation %q not found for tenant", recID)
	}
	if !validAIStatusTransition(rec.Status, to) {
		s.mu.Unlock()
		return fmt.Errorf("invalid AI recommendation transition %s -> %s", rec.Status, to)
	}
	rec.Status = to
	now := utcNow()
	if to == AIStatusAuthorized || to == AIStatusRejected {
		rec.ReviewedBy = actor
		rec.ReviewedAt = &now
		rec.HumanDecision = decision
	}
	if to == AIStatusExecuted {
		rec.ExecutedAt = &now
	}
	s.mu.Unlock()

	persistPhase12Recommendation(rec)
	recordAuditEvent(actor, "ai.recommendation."+to, "ai_recommendations", recID, tenantID, rec.CorrelationID, "", map[string]interface{}{
		"recommendation": rec.Recommendation,
		"human_decision": decision,
	})
	statgateAIRecommendationsPending.Add(recommendationPendingDelta(rec))
	return nil
}

func recommendationPendingDelta(rec *IntelligenceRecommendation) int64 {
	if rec.Status == AIStatusPendingReview {
		return 1
	}
	switch rec.Status {
	case AIStatusAuthorized, AIStatusRejected, AIStatusExecuted, AIStatusCancelled, AIStatusFailed:
		return -1
	}
	return 0
}

// ReviewRecommendation authorizes or rejects a pending recommendation. Only a
// human with institutional_lead / admin role may do this (enforced at the API
// layer; the engine enforces state validity).
func (s *Phase12Store) ReviewRecommendation(tenantID, recID, actor, decision string, approve bool) error {
	to := AIStatusRejected
	if approve {
		to = AIStatusAuthorized
	}
	return s.transitionRecommendation(tenantID, recID, to, actor, decision)
}

// ExecuteRecommendation marks an AUTHORIZED recommendation as EXECUTED. AI
// never executes itself â€” a human performs the consequential action and
// records it here (directive Â§4).
func (s *Phase12Store) ExecuteRecommendation(tenantID, recID, actor string) error {
	return s.transitionRecommendation(tenantID, recID, AIStatusExecuted, actor, "executed by human operator")
}

func (s *Phase12Store) ListRecommendations(tenantID string) []*IntelligenceRecommendation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*IntelligenceRecommendation
	for _, rec := range s.recommendations {
		if rec.TenantID == tenantID {
			out = append(out, rec)
		}
	}
	return out
}
