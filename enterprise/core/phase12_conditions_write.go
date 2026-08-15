package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — CONDITION PERSISTENCE (WRITE) & EVENT EMISSION
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

func persistPhase12Condition(c *InstitutionalCondition) {
	if dbPool == nil {
		return
	}
	sigJSON, _ := json.Marshal(c.SupportingSignals)
	domJSON, _ := json.Marshal(c.AffectedDomains)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx, `INSERT INTO institutional_conditions
		(id, tenant_id, condition_level, composite_score, calculation_timestamp,
		 supporting_signals, affected_domains, explanation, calculation_version, correlation_id)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10)
		ON CONFLICT (id) DO NOTHING`,
		c.ID, c.TenantID, string(c.ConditionLevel), c.CompositeScore, c.CalculationTimestamp,
		string(sigJSON), string(domJSON), c.Explanation, c.CalculationVersion, c.CorrelationID)
	if err != nil {
		log.Printf("phase12: persist condition failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()

	// Emit condition.changed on the canonical event channel (directive §11).
	publishEvent(DomainEvent{
		EventType:   "condition.changed",
		Source:      "intelligence_engine",
		ObjectType:  "institutional_condition",
		ObjectID:    c.ID,
		TenantID:    c.TenantID,
		Correlation: c.CorrelationID,
		Payload: map[string]interface{}{
			"condition_level": string(c.ConditionLevel),
			"composite_score": c.CompositeScore,
			"tenant_id":       c.TenantID,
		},
	})
}