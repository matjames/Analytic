package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — CONDITION EXPLAINABILITY & PERSISTENCE
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

// buildConditionExplanation renders a human-readable explanation so a
// decision-maker can answer "Why is the institution X?".
func buildConditionExplanation(level ConditionLevel, composite int, signals []SupportingSignalRef, unavailable []string) string {
	expl := fmt.Sprintf("Institution is %s (composite severity %d/100, calculation %s).", level, composite, conditionCalculationVersion)
	if len(signals) == 0 {
		return expl + " No supporting signals."
	}
	byDomain := map[string][]SupportingSignalRef{}
	for _, sig := range signals {
		byDomain[sig.Domain] = append(byDomain[sig.Domain], sig)
	}
	var keys []string
	for d := range byDomain {
		keys = append(keys, d)
	}
	sort.Strings(keys)
	expl += " Supporting signals:"
	for _, d := range keys {
		maxSev := 0
		for _, sig := range byDomain[d] {
			if sig.Severity > maxSev {
				maxSev = sig.Severity
			}
		}
		expl += fmt.Sprintf(" %d in %s (max severity %d)", len(byDomain[d]), d, maxSev)
	}
	if len(unavailable) > 0 {
		sort.Strings(unavailable)
		expl += " Domains with no data: " + joinStrings(unavailable, ", ") + "."
	}
	return expl
}

// joinStrings is a small deterministic join helper used across Phase XII.
func joinStrings(items []string, sep string) string {
	out := ""
	for i, it := range items {
		if i > 0 {
			out += sep
		}
		out += it
	}
	return out
}

// storeCondition persists a condition snapshot and emits condition.changed.
func (s *Phase12Store) storeCondition(c *InstitutionalCondition) error {
	s.mu.Lock()
	s.conditions[c.TenantID] = c
	s.mu.Unlock()
	persistPhase12Condition(c)
	return nil
}

// GetCurrentCondition returns the latest condition snapshot for a tenant.
func (s *Phase12Store) GetCurrentCondition(tenantID string) *InstitutionalCondition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conditions[tenantID]
}

// GetConditionHistory returns stored condition snapshots for a tenant. When a
// database is configured, the full history is read from PostgreSQL so every
// calculation is preserved for audit.
func (s *Phase12Store) GetConditionHistory(tenantID string) []*InstitutionalCondition {
	if dbPool != nil {
		rows, err := dbPool.QueryContext(context.Background(), `
			SELECT id, tenant_id, condition_level, composite_score, calculation_timestamp,
			       supporting_signals, affected_domains, explanation, calculation_version, correlation_id
			FROM institutional_conditions
			WHERE tenant_id = $1
			ORDER BY calculation_timestamp DESC LIMIT 200`, tenantID)
		if err == nil {
			defer rows.Close()
			var out []*InstitutionalCondition
			for rows.Next() {
				var c InstitutionalCondition
				var sigJSON, domJSON []byte
				var level string
				if err := rows.Scan(&c.ID, &c.TenantID, &level, &c.CompositeScore,
					&c.CalculationTimestamp, &sigJSON, &domJSON, &c.Explanation,
					&c.CalculationVersion, &c.CorrelationID); err == nil {
					c.ConditionLevel = ConditionLevel(level)
					_ = json.Unmarshal(sigJSON, &c.SupportingSignals)
					_ = json.Unmarshal(domJSON, &c.AffectedDomains)
					out = append(out, &c)
				}
			}
			if out != nil {
				return out
			}
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if c, ok := s.conditions[tenantID]; ok {
		return []*InstitutionalCondition{c}
	}
	return []*InstitutionalCondition{}
}