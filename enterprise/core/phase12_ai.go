package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — GOVERNED AI REASONING LAYER (SPRINT 6)
//
// AI is advisory only (directive §4, §19). The provider is abstracted behind
// ReasoningProvider so the platform is not coupled to a single vendor.
// AI never directly mutates records; every recommendation flows through:
//   GENERATED -> PENDING_REVIEW -> AUTHORIZED|REJECTED -> EXECUTED|CANCELLED
// Every request is written to ai_audit_log (linked to the platform audit).
// ═══════════════════════════════════════════════════════════════════════════════

import "fmt"

// AI lifecycle states.
const (
	AIStatusGenerated     = "GENERATED"
	AIStatusPendingReview = "PENDING_REVIEW"
	AIStatusAuthorized    = "AUTHORIZED"
	AIStatusRejected      = "REJECTED"
	AIStatusExecuted      = "EXECUTED"
	AIStatusCancelled     = "CANCELLED"
	AIStatusFailed        = "FAILED"
)

// validAIStatusTransition reports allowed lifecycle transitions.
func validAIStatusTransition(from, to string) bool {
	switch from {
	case AIStatusGenerated:
		return to == AIStatusPendingReview || to == AIStatusFailed || to == AIStatusCancelled
	case AIStatusPendingReview:
		return to == AIStatusAuthorized || to == AIStatusRejected || to == AIStatusCancelled
	case AIStatusAuthorized:
		return to == AIStatusExecuted || to == AIStatusCancelled
	}
	return false
}

// defaultReasoningProvider is the deterministic local provider used when no
// external provider is configured. It is NOT a mock-data generator: it derives
// its recommendation from the real signals / condition in the store.
type defaultReasoningProvider struct{ store *Phase12Store }

func (p *defaultReasoningProvider) Name() string { return "deterministic-local" }

func (p *defaultReasoningProvider) Generate(input AIInput) (*AIOutput, error) {
	cond := p.store.GetCurrentCondition(input.TenantID)
	out := &AIOutput{
		Confidence:         0.5,
		Recommendation:     "NO DATA: no institutional signals are available; no recommendation can be derived.",
		ReasoningSummary:   "The deterministic local provider found no active intelligence signals for this tenant.",
		SupportingEvidence: []string{},
		AffectedObjects:    []string{},
		RiskLevel:          string(ConditionUnknown),
		RecommendedActions: []string{},
		Limitations:        []string{"Deterministic local provider; replace with a configured ReasoningProvider for richer analysis."},
	}
	if cond == nil {
		return out, nil
	}
	out.RiskLevel = string(cond.ConditionLevel)
	if cond.CompositeScore >= 55 {
		out.Recommendation = "Institutional condition is " + string(cond.ConditionLevel) +
			" (severity " + fmt.Sprintf("%d", cond.CompositeScore) + "). Management review is recommended."
		out.ReasoningSummary = cond.Explanation
		out.Confidence = 0.8
		out.RecommendedActions = []string{
			"Review supporting signals in " + joinDomainNames(cond.AffectedDomains) + ".",
			"Confirm data freshness before taking consequential action.",
		}
		for _, ref := range cond.SupportingSignals {
			out.SupportingEvidence = append(out.SupportingEvidence, ref.SignalType+" ("+ref.Domain+", severity "+fmt.Sprintf("%d", ref.Severity)+")")
		}
	} else {
		out.Recommendation = "Institutional condition is " + string(cond.ConditionLevel) +
			" (severity " + fmt.Sprintf("%d", cond.CompositeScore) + "). Continue monitoring."
		out.ReasoningSummary = cond.Explanation
		out.Confidence = 0.7
	}
	return out, nil
}

func joinDomainNames(domains []AffectedDomain) string {
	names := make([]string, 0, len(domains))
	for _, d := range domains {
		names = append(names, d.Domain)
	}
	return joinStrings(names, ", ")
}

// configuredProvider is the optional external reasoning provider, set at
// startup from configuration; nil means the deterministic local provider.
var configuredProvider ReasoningProvider

// GetReasoningProvider returns the configured provider or the deterministic
// local default. Configuration is read from env (provider-agnostic).
func GetReasoningProvider(tenantID string) ReasoningProvider {
	if p := configuredProvider; p != nil {
		return p
	}
	return &defaultReasoningProvider{store: phase12}
}

// dataClassificationAllowedForProvider applies the AI data classification
// gate (directive §17): RESTRICTED and SENSITIVE data are never sent to
// external providers.
func dataClassificationAllowedForProvider(classification string) bool {
	switch classification {
	case "PUBLIC", "INTERNAL", "OFFICIAL":
		return true
	case "CONFIDENTIAL", "RESTRICTED", "SENSITIVE":
		return false
	}
	return false
}
