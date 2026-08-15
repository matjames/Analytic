package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — RISK INTELLIGENCE (SPRINT 5)
//
// Phase XII does NOT create a competing risk register. StatGovernance is the
// source of record for authoritative risks. This layer CONSUMES risk status /
// transitions / severity / ownership / relationships and correlates them with
// incidents, datasets, KPIs, projects, services and events.
//
// Emerging-risk detection (directive §15) produces an AI / intelligence
// SIGNAL — human governance remains responsible for formally registering an
// authoritative institutional risk.
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// RecordRiskEvent records a risk-intelligence correlation event. The risk
// reference points to the authoritative register (e.g. StatGovernance).
func (s *Phase12Store) RecordRiskEvent(ev *RiskEvent) error {
	if ev == nil {
		return fmt.Errorf("risk event is nil")
	}
	if ev.TenantID == "" || ev.EventType == "" {
		return fmt.Errorf("risk event requires tenant_id and event_type")
	}
	if ev.SeverityEstimate < 0 || ev.SeverityEstimate > 100 {
		return fmt.Errorf("severity estimate must be between 0 and 100")
	}
	if ev.Status == "" {
		ev.Status = "DETECTED"
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = utcNow()
	}
	if ev.ID == "" {
		ev.ID = fmt.Sprintf("rsk_%d", time.Now().UnixNano())
	}

	s.mu.Lock()
	s.riskEvents[ev.TenantID] = append(s.riskEvents[ev.TenantID], ev)
	s.mu.Unlock()
	persistPhase12RiskEvent(ev)
	recordAuditEvent("", "risk.event.recorded", "risk_events", ev.ID, ev.TenantID, ev.CorrelationID, "", map[string]interface{}{
		"event_type":        ev.EventType,
		"severity_estimate": ev.SeverityEstimate,
		"risk_reference":    ev.RiskReference,
	})
	return nil
}

func (s *Phase12Store) ListRiskEvents(tenantID string) []*RiskEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*RiskEvent
	for _, ev := range s.riskEvents[tenantID] {
		out = append(out, ev)
	}
	return out
}

// DetectEmergingRisk analyses repeated incidents / declining KPIs affecting
// the same canonical object and, if a threshold is reached, creates an
// intelligence SIGNAL (never an authoritative risk record). This is the
// "emerging risk detection" layer of directive §15.
func (s *Phase12Store) DetectEmergingRisk(tenantID string, events []*RiskEvent) (*IntelligenceSignal, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("emerging risk detection requires tenant_id")
	}
	countByObject := map[string]int{}
	var totalSeverity int
	for _, ev := range events {
		totalSeverity += ev.SeverityEstimate
		for _, oid := range ev.CorrelatedObjects {
			countByObject[oid]++
		}
	}
	// Threshold: at least 3 repeated events on a corpus, or any event with
	// severity >= 75, triggers an emerging-risk signal.
	triggered := ""
	triggerSeverity := 0
	for oid, cnt := range countByObject {
		if cnt >= 3 {
			triggered = oid
			if cnt*20 > triggerSeverity {
				triggerSeverity = cnt * 20
			}
		}
	}
	if triggered == "" && totalSeverity >= 75 {
		triggered = "institution"
		triggerSeverity = 75
	}
	if triggered == "" {
		return nil, nil // no emerging risk
	}
	if triggerSeverity > 90 {
		triggerSeverity = 90
	}

	sig := &IntelligenceSignal{
		TenantID:     tenantID,
		SignalType:   "emerging_risk",
		Domain:       DomainRisk,
		Severity:     triggerSeverity,
		SourceSystem: "intelligence_risk",
		Status:       "ACTIVE",
		CorrelationID: fmt.Sprintf("corr_emerging_risk_%d", time.Now().UnixNano()),
		Evidence:     fmt.Sprintf("Emerging risk detected across %d correlated events targeting canonical object %s.", len(events), triggered),
		CreatedAt:    utcNow(),
		Payload: map[string]interface{}{
			"object_canonical_id": triggered,
			"event_count":         len(events),
		},
	}
	if err := s.IngestSignal(sig); err != nil {
		return nil, err
	}
	return sig, nil
}

func persistPhase12RiskEvent(ev *RiskEvent) {
	if dbPool == nil {
		return
	}
	cobJSON, _ := json.Marshal(ev.CorrelatedObjects)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx, `
		INSERT INTO risk_events
			(id, tenant_id, event_type, source_system, source_event_id, risk_reference,
			 severity_estimate, correlated_objects, signal_id, evidence, correlation_id, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO NOTHING`,
		ev.ID, ev.TenantID, ev.EventType, ev.SourceSystem, ev.SourceEventID, ev.RiskReference,
		ev.SeverityEstimate, string(cobJSON), ev.SignalID, ev.Evidence, ev.CorrelationID, ev.Status, ev.CreatedAt)
	if err != nil {
		log.Printf("phase12: persist risk event failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()
}