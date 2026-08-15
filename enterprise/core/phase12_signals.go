package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — INTELLIGENCE SIGNALS
//
// Signals are computed facts consumed from existing systems via the event
// bus (directive §9, §11). Signals are the ONLY input to the institutional
// condition. The condition itself is always COMPUTED, never manually entered.
//
// Signal ingestion is fail-closed:
//   - tenant_id is mandatory (never derived from headers)
//   - domain must be one of the eight approved intelligence domains
//   - severity must be 0-100
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// IngestSignal validates and stores an intelligence signal, then triggers a
// deterministic condition recalculation for the tenant (directive §11:
// signal evaluation -> condition calculation -> condition.changed).
func (s *Phase12Store) IngestSignal(sig *IntelligenceSignal) error {
	if sig == nil {
		return fmt.Errorf("intelligence signal is nil")
	}
	if sig.TenantID == "" {
		return fmt.Errorf("intelligence signal requires tenant_id")
	}
	if sig.SignalType == "" {
		return fmt.Errorf("intelligence signal requires signal_type")
	}
	if sig.SourceSystem == "" {
		return fmt.Errorf("intelligence signal requires source_system")
	}
	if !validSignalDomain(sig.Domain) {
		return fmt.Errorf("invalid signal domain %q", sig.Domain)
	}
	if sig.Severity < 0 || sig.Severity > 100 {
		return fmt.Errorf("signal severity must be between 0 and 100")
	}
	if sig.Status == "" {
		sig.Status = "ACTIVE"
	}
	if sig.CreatedAt.IsZero() {
		sig.CreatedAt = utcNow()
	}
	if sig.ID == "" {
		sig.ID = fmt.Sprintf("sig_%d", time.Now().UnixNano())
	}

	s.mu.Lock()
	s.signals[sig.TenantID] = append(s.signals[sig.TenantID], sig)
	s.mu.Unlock()
	persistPhase12Signal(sig)

	// Deterministic recalculation. Recalculated on the event path; on direct
	// ingestion this guarantees signal -> condition in <= 60s.
	return s.RecalculateCondition(sig.TenantID, sig.CorrelationID)
}

// ListSignals returns the active signals for a tenant.
func (s *Phase12Store) ListSignals(tenantID string) []*IntelligenceSignal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*IntelligenceSignal
	for _, sig := range s.signals[tenantID] {
		if sig.Status == "ACTIVE" {
			out = append(out, sig)
		}
	}
	return out
}

// ListAllSignals returns active + superseded signals (for audit views).
func (s *Phase12Store) ListAllSignals(tenantID string) []*IntelligenceSignal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*IntelligenceSignal
	for _, sig := range s.signals[tenantID] {
		out = append(out, sig)
	}
	return out
}

// SupersedeSignalsForEvent marks signals derived from a source event as
// superseded before re-ingesting newer state (event replay support).
func (s *Phase12Store) SupersedeSignalsForEvent(tenantID, sourceEventID string) {
	s.mu.Lock()
	for _, sig := range s.signals[tenantID] {
		if sig.SourceEventID == sourceEventID && sig.Status == "ACTIVE" {
			sig.Status = "SUPERSEDED"
		}
	}
	s.mu.Unlock()
}

func persistPhase12Signal(sig *IntelligenceSignal) {
	if dbPool == nil {
		return
	}
	payload, _ := json.Marshal(sig.Payload)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx, `
		INSERT INTO intelligence_signals
			(id, tenant_id, signal_type, domain, severity, source_system, source_event_id,
			 status, payload, correlation_id, evidence, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12)
		ON CONFLICT (id) DO NOTHING`,
		sig.ID, sig.TenantID, sig.SignalType, sig.Domain, sig.Severity, sig.SourceSystem,
		sig.SourceEventID, sig.Status, string(payload), sig.CorrelationID, sig.Evidence, sig.CreatedAt)
	if err != nil {
		log.Printf("phase12: persist signal failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()
}
