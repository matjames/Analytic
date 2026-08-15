package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — INSTITUTIONAL CONDITION ENGINE
//
// Deterministic and explainable (directive §10). Every calculation stores
// condition_level, calculation_timestamp, supporting_signals,
// affected_domains, calculation_version, correlation_id, tenant_id, so a
// decision-maker can answer "Why is the institution currently ELEVATED?".
//
// Missing data ≠ healthy: domains with no active signal are surfaced as
// UNAVAILABLE and never treated as a healthy zero. If no domain has data the
// condition is UNKNOWN — never OPTIMAL.
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"fmt"
	"time"
)

// Domain weights — deterministic, versioned (sum = 1.0).
var domainWeights = map[string]float64{
	DomainStrategicPerformance:  0.22,
	DomainOperationalHealth:     0.18,
	DomainDataQuality:           0.18,
	DomainReportingCompleteness: 0.12,
	DomainFinancial:             0.08,
	DomainResearch:              0.08,
	DomainRisk:                  0.10,
	DomainSecurity:              0.04,
}

// conditionCalculationVersion identifies this deterministic calculation.
const conditionCalculationVersion = "PHASE_XII-1"

// levelFromSeverity maps a composite severity (0-100, higher = worse) to a
// condition level.
func levelFromSeverity(severity int) ConditionLevel {
	switch {
	case severity < 20:
		return ConditionOptimal
	case severity < 40:
		return ConditionNominal
	case severity < 55:
		return ConditionAttention
	case severity < 70:
		return ConditionElevated
	case severity < 85:
		return ConditionCritical
	default:
		return ConditionEmergency
	}
}

// RecalculateCondition recomputes the institutional condition for a tenant
// from the current active signals. Deterministic for a given signal set.
func (s *Phase12Store) RecalculateCondition(tenantID, correlationID string) error {
	if tenantID == "" {
		return fmt.Errorf("condition calculation requires tenant_id")
	}
	startT := time.Now()
	defer func() {
		statgateConditionCalcs.Add(1)
		statgateConditionCalcDurNs.Add(time.Since(startT).Nanoseconds())
	}()

	signals := s.ListSignals(tenantID)
	if len(signals) == 0 {
		return s.storeCondition(&InstitutionalCondition{
			ID:                   fmt.Sprintf("cond_%d", time.Now().UnixNano()),
			TenantID:             tenantID,
			ConditionLevel:       ConditionUnknown,
			CompositeScore:       0,
			CalculationTimestamp: utcNow(),
			SupportingSignals:    []SupportingSignalRef{},
			AffectedDomains:      []AffectedDomain{},
			Explanation:          "NO DATA: no intelligence signals are available for this tenant. Condition is UNKNOWN, not healthy.",
			CalculationVersion:   conditionCalculationVersion,
			CorrelationID:        correlationID,
		})
	}

	domainMax := map[string]int{}
	domainSignals := map[string][]*IntelligenceSignal{}
	for _, sig := range signals {
		if sig.Severity > domainMax[sig.Domain] {
			domainMax[sig.Domain] = sig.Severity
		}
		domainSignals[sig.Domain] = append(domainSignals[sig.Domain], sig)
	}

	var (
		weightedSum        float64
		totalWeight        float64
		supportingRefs     []SupportingSignalRef
		affectedDomains    []AffectedDomain
		unavailableDomains []string
	)

	for _, dom := range allSignalDomains {
		weight := domainWeights[dom]
		sevs, ok := domainSignals[dom]
		if !ok {
			unavailableDomains = append(unavailableDomains, dom)
			continue
		}
		maxSev := domainMax[dom]
		affectedDomains = append(affectedDomains, AffectedDomain{Domain: dom, Severity: maxSev, SignalCount: len(sevs)})

		staleCount := 0
		for _, sig := range sevs {
			if sig.Payload != nil {
				if fs, ok := sig.Payload["freshness"].(string); ok && fs == "STALE" {
					staleCount++
				}
			}
		}
		effectiveWeight := weight
		if staleCount == len(sevs) {
			effectiveWeight *= 0.5
		}
		weightedSum += float64(maxSev) * effectiveWeight
		totalWeight += effectiveWeight

		for _, sig := range sevs {
			supportingRefs = append(supportingRefs, SupportingSignalRef{
				SignalID:   sig.ID,
				SignalType: sig.SignalType,
				Domain:     sig.Domain,
				Severity:   sig.Severity,
				Evidence:   sig.Evidence,
			})
		}
	}

	if totalWeight == 0 {
		return s.storeCondition(&InstitutionalCondition{
			ID:                   fmt.Sprintf("cond_%d", time.Now().UnixNano()),
			TenantID:             tenantID,
			ConditionLevel:       ConditionUnknown,
			CompositeScore:       0,
			CalculationTimestamp: utcNow(),
			SupportingSignals:    supportingRefs,
			AffectedDomains:      affectedDomains,
			Explanation:          "NO DATA: every intelligence domain is unavailable. Condition is UNKNOWN, not healthy.",
			CalculationVersion:   conditionCalculationVersion,
			CorrelationID:        correlationID,
		})
	}

	composite := int(weightedSum / totalWeight)
	level := levelFromSeverity(composite)

	// Missing data ≠ healthy: with any unavailable domain the condition
	// cannot be reported as OPTIMAL.
	if len(unavailableDomains) > 0 && level == ConditionOptimal {
		level = ConditionNominal
	}
	// Low coverage caps the condition (Sprint 0 validation).
	confidence := totalWeight / 1.0
	if confidence < 0.60 && level == ConditionOptimal {
		level = ConditionAttention
	}

	return s.storeCondition(&InstitutionalCondition{
		ID:                   fmt.Sprintf("cond_%d", time.Now().UnixNano()),
		TenantID:             tenantID,
		ConditionLevel:       level,
		CompositeScore:       composite,
		CalculationTimestamp: utcNow(),
		SupportingSignals:    supportingRefs,
		AffectedDomains:      affectedDomains,
		Explanation:          buildConditionExplanation(level, composite, supportingRefs, unavailableDomains),
		CalculationVersion:   conditionCalculationVersion,
		CorrelationID:        correlationID,
	})
}