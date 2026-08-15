package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — TEST SUITE: INTELLIGENCE SIGNALS & INSTITUTIONAL CONDITION ENGINE
//
// Covers mandatory intelligence tests (directive §30): event -> signal,
// signal -> condition, multiple signals -> aggregated condition, condition
// history, stale data, missing data. Also KPI data status (ACTUAL/STALE/MISSING).
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"testing"
	"time"
)

func TestPhase12_Intelligence_EventToSignal(t *testing.T) {
	ev := DomainEvent{
		EventType: "incident.escalated", Source: "phase11",
		ObjectType: "Incident", ObjectID: "inc_1", TenantID: "test_t",
		Payload: map[string]interface{}{"severity": 85},
	}
	before := len(phase12.ListAllSignals("test_t"))
	processIntelligenceForEvent(ev)
	after := len(phase12.ListAllSignals("test_t"))
	if after <= before {
		t.Fatal("event must produce an intelligence signal (event -> signal)")
	}
	// Condition must be recalculated (computed, never manual).
	if phase12.GetCurrentCondition("test_t") == nil {
		t.Fatal("condition must be computed after signal ingestion")
	}
}

func TestPhase12_Condition_SignalDrivesCondition(t *testing.T) {
	s := newPhase12Store()
	mustNoErr(t, s.RecalculateCondition("t1", "c0"), "recalc with no signals")
	if phase12_condLevel(s, "t1") != string(ConditionUnknown) {
		t.Fatal("no signals -> UNKNOWN (missing data not healthy)")
	}
	// Low severity in one domain: unavailable domains must cap below OPTIMAL.
	mustNoErr(t, s.IngestSignal(&IntelligenceSignal{
		TenantID: "t1", SignalType: "ok", Domain: DomainOperationalHealth,
		Severity: 10, SourceSystem: "test", Status: "ACTIVE",
	}), "ingest low signal")
	if phase12_condLevel(s, "t1") == string(ConditionUnknown) {
		t.Fatal("signal must drive condition away from UNKNOWN")
	}
}

func TestPhase12_Condition_MissingDataNotHealthy(t *testing.T) {
	s := newPhase12Store()
	// Only one domain has data (low severity). The rest are missing.
	mustNoErr(t, s.IngestSignal(&IntelligenceSignal{
		TenantID: "t1", SignalType: "ok", Domain: DomainOperationalHealth,
		Severity: 5, SourceSystem: "test",
	}), "ingest")
	if level := phase12_condLevel(s, "t1"); level == string(ConditionOptimal) {
		t.Fatal("missing data must never present as OPTIMAL")
	}
	cond := s.GetCurrentCondition("t1")
	if cond == nil {
		t.Fatal("condition missing")
	}
}

func TestPhase12_Condition_MultipleSignalsAggregate(t *testing.T) {
	s := newPhase12Store()
	// Low severity across all dominated domains -> low composite.
	for _, d := range allSignalDomains {
		mustNoErr(t, s.IngestSignal(&IntelligenceSignal{
			TenantID: "t1", SignalType: "low." + d, Domain: d,
			Severity: 5, SourceSystem: "test",
		}), "ingest "+d)
	}
	cond := s.GetCurrentCondition("t1")
	if cond == nil {
		t.Fatal("condition missing")
	}
	if cond.CompositeScore >= 20 {
		t.Fatalf("low-severity signals should aggregate to low composite, got %d", cond.CompositeScore)
	}

	// Now push heavy severity into several weighted domains -> aggregate rises.
	mustNoErr(t, s.IngestSignal(&IntelligenceSignal{
		TenantID: "t1", SignalType: "sev.security", Domain: DomainSecurity,
		Severity: 90, SourceSystem: "test",
	}), "ingest high security")
	mustNoErr(t, s.IngestSignal(&IntelligenceSignal{
		TenantID: "t1", SignalType: "sev.ops", Domain: DomainOperationalHealth,
		Severity: 90, SourceSystem: "test",
	}), "ingest high ops")
	mustNoErr(t, s.IngestSignal(&IntelligenceSignal{
		TenantID: "t1", SignalType: "sev.strat", Domain: DomainStrategicPerformance,
		Severity: 90, SourceSystem: "test",
	}), "ingest high strategic")

	cond = s.GetCurrentCondition("t1")
	if cond.CompositeScore <= 5 {
		t.Fatalf("high-severity signals must lift aggregate above baseline, got %d", cond.CompositeScore)
	}
	if cond.ConditionLevel == ConditionOptimal {
		t.Fatal("elevated signals must not present as OPTIMAL")
	}
	if cond.SupportingSignals == nil || len(cond.SupportingSignals) == 0 {
		t.Fatal("condition must carry supporting signal evidence (explainability)")
	}
	if cond.Explanation == "" {
		t.Fatal("condition must produce a human-readable explanation")
	}
}

func TestPhase12_Condition_StaleDataWeighting(t *testing.T) {
	s := newPhase12Store()
	// Two-domain scenario: a moderate fresh signal and a very high, STALE
	// signal. Stale data must carry half weight (Sprint 0 validation), so the
	// severe stale signal must NOT spike the composite into ELEVATED range.
	mustNoErr(t, s.IngestSignal(&IntelligenceSignal{
		TenantID: "t1", SignalType: "ops", Domain: DomainOperationalHealth,
		Severity: 20, SourceSystem: "test",
	}), "ingest fresh moderate")
	mustNoErr(t, s.IngestSignal(&IntelligenceSignal{
		TenantID: "t1", SignalType: "sec", Domain: DomainSecurity,
		Severity: 100, SourceSystem: "test",
		Payload: map[string]interface{}{"freshness": "STALE"},
	}), "ingest stale severe")

	cond := s.GetCurrentCondition("t1")
	if cond == nil {
		t.Fatal("condition missing")
	}
	// With half weighting the stale severe signal is discounted: composite must
	// remain well below the ELEVATED threshold, yet above "healthy".
	if cond.CompositeScore >= 55 {
		t.Fatalf("stale high-severity signal must be discounted (half weight), got composite %d", cond.CompositeScore)
	}
	if cond.CompositeScore == 0 {
		t.Fatal("condition must not be treated as healthy zero")
	}
}

// KPI data status: ACTUAL / STALE / MISSING distinction (directive §13).
func TestPhase12_KPI_DataStatus(t *testing.T) {
	s := newPhase12Store()
	mustNoErr(t, s.UpsertKPI(&KPI{TenantID: "t1", Name: "Coverage", MeasurementPeriod: "MONTHLY", CanonicalID: "k1"}), "create kpi")
	kpi := s.ListKPIs("t1")[0]
	if kpi.DataStatus != KpiDataMissing {
		t.Fatalf("new KPI must be MISSING, got %s", kpi.DataStatus)
	}
	mustNoErr(t, s.RecordMeasurement("t1", kpi.ID, 95, "%", "statcollect", false, time.Now()), "measure")
	kpi = s.ListKPIs("t1")[0]
	if kpi.DataStatus != KpiDataActual {
		t.Fatalf("fresh measurement must be ACTUAL, got %s", kpi.DataStatus)
	}
	mustNoErr(t, s.RecordMeasurement("t1", kpi.ID, 80, "%", "statcollect", false, time.Now().Add(-300*time.Hour)), "measure stale")
	kpi = s.ListKPIs("t1")[0]
	if kpi.DataStatus != KpiDataStale {
		t.Fatalf("old measurement must be STALE, got %s", kpi.DataStatus)
	}
}

// Condition history is preserved for auditability.
func TestPhase12_Condition_History(t *testing.T) {
	s := newPhase12Store()
	mustNoErr(t, s.RecalculateCondition("t1", "h1"), "recalc1")
	mustNoErr(t, s.IngestSignal(&IntelligenceSignal{TenantID: "t1", SignalType: "s", Domain: DomainRisk, Severity: 60, SourceSystem: "test"}), "ingest")
	hist := s.GetConditionHistory("t1")
	if len(hist) == 0 {
		t.Fatal("condition history must be available")
	}
}

func phase12_condLevel(s *Phase12Store, tenantID string) string {
	if c := s.GetCurrentCondition(tenantID); c != nil {
		return string(c.ConditionLevel)
	}
	return ""
}
