package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — INTELLIGENCE EVENT PROCESSOR
//
// Consumes the canonical STATGATE_EVENT_CHANNEL through the existing event
// pipeline (directive §11). No polling loops. Flow:
//   Event Bus -> Intelligence Event Processor -> Signal Evaluation
//   -> Condition Calculation -> condition.changed -> Command Centre
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"fmt"
	"strconv"
	"time"
)

// processIntelligenceForEvent is called from the existing event pipeline
// (processEvent) for every published domain event. It is additive and never
// alters the behaviour of Phases I-XI processors.
func processIntelligenceForEvent(ev DomainEvent) {
	if ev.EventType == "condition.changed" {
		// Feedback loop guard: condition.changed is an OUTPUT of this engine.
		return
	}
	statgateIntelligenceEvents.Add(1)

	tenantID := ev.TenantID
	if tenantID == "" {
		tenantID = "statgate"
	}

	// 1. Project the event object into the institutional object registry.
	projectEventObject(ev)

	// 2. Derive signals from the event (evidence-based severity).
	sig := deriveSignalFromEvent(ev, tenantID)
	if sig != nil {
		if err := phase12.IngestSignal(sig); err != nil {
			recordDeadLetter(ev, "intelligence_signal_derivation_failed: "+err.Error())
		}
	}

	// 3. Risk intelligence correlation for risk-relevant events.
	if isRiskRelevantEvent(ev) {
		correlateRiskEvent(ev, tenantID)
	}
}

// projectEventObject creates/refreshes the object projection for the event's
// subject when the object type is in the approved registry set.
func projectEventObject(ev DomainEvent) {
	if ev.ObjectType == "" || ev.ObjectID == "" {
		return
	}
	if !validObjectType(ev.ObjectType) {
		return
	}
	canonicalID := fmt.Sprintf("uoi_%s_%s_%s", ev.Source, ev.ObjectType, ev.ObjectID)
	if ev.TenantID == "" {
		ev.TenantID = "statgate"
	}
	name := ev.ObjectID
	if n, ok := ev.Payload["name"].(string); ok && n != "" {
		name = n
	}
	obj := phase12.GetObject(ev.TenantID, canonicalID)
	if obj == nil {
		_ = phase12.ProjectObject(&InstitutionalObject{
			TenantID:         ev.TenantID,
			CanonicalID:      canonicalID,
			ObjectType:       ev.ObjectType,
			SourceSystem:     ev.Source,
			SourceObjectID:   ev.ObjectID,
			DisplayName:      name,
			ProjectionStatus: "ACTIVE",
			Metadata: map[string]interface{}{
				"source_event_id": ev.ID,
				"event_type":      ev.EventType,
			},
		})
	}
}

// deriveSignalFromEvent maps an event to a deterministic intelligence signal.
// Severity is read from the event payload when present, otherwise a
// conservative default per event family is used. Missing data is never
// treated as healthy.
func deriveSignalFromEvent(ev DomainEvent, tenantID string) *IntelligenceSignal {
	signalType := ""
	domain := ""
	defaultSeverity := 0
	freshness := "FRESH"

	switch ev.EventType {
	case "incident.created":
		signalType, domain, defaultSeverity = "incident_active", DomainOperationalHealth, 55
	case "incident.escalated", "incident.critical":
		signalType, domain, defaultSeverity = "incident_escalated", DomainOperationalHealth, 80
	case "incident.resolved":
		signalType, domain, defaultSeverity = "incident_resolved", DomainOperationalHealth, 15
	case "dataset.quality_degraded", "data_quality.degraded":
		signalType, domain, defaultSeverity = "data_quality_degraded", DomainDataQuality, 60
	case "dataset.stale", "dataset.expired":
		signalType, domain, defaultSeverity = "data_stale", DomainDataQuality, 70
		freshness = "STALE"
	case "reporting.missed", "reporting.incomplete":
		signalType, domain, defaultSeverity = "reporting_incomplete", DomainReportingCompleteness, 65
	case "submission.received":
		signalType, domain, defaultSeverity = "submission_received", DomainReportingCompleteness, 10
	case "kpi.declining", "kpi.missed_target":
		signalType, domain, defaultSeverity = "kpi_declining", DomainStrategicPerformance, 60
	case "kpi.measured":
		signalType, domain, defaultSeverity = "kpi_measured", DomainStrategicPerformance, 20
	case "risk.created", "risk.escalated":
		signalType, domain, defaultSeverity = "risk_escalated", DomainRisk, 65
	case "security.incident", "security.breach":
		signalType, domain, defaultSeverity = "security_incident", DomainSecurity, 85
	case "research.delay", "research.milestone_missed":
		signalType, domain, defaultSeverity = "research_delay", DomainResearch, 45
	case "budget.overrun", "financial.variance":
		signalType, domain, defaultSeverity = "financial_variance", DomainFinancial, 55
	case "service.degraded", "service.outage":
		signalType, domain, defaultSeverity = "service_degraded", DomainOperationalHealth, 60
	default:
		return nil
	}

	severity := defaultSeverity
	if v, ok := ev.Payload["severity"].(float64); ok {
		sev := int(v)
		if sev >= 0 && sev <= 100 {
			severity = sev
		}
	} else if vs, ok := ev.Payload["severity"].(string); ok {
		if sev, err := strconv.Atoi(vs); err == nil && sev >= 0 && sev <= 100 {
			severity = sev
		}
	}
	if fs, ok := ev.Payload["freshness"].(string); ok && fs == "STALE" {
		freshness = "STALE"
	}

	evidence := fmt.Sprintf("Event %s (%s) from %s: severity %d/100.", ev.EventType, ev.ObjectID, ev.Source, severity)

	return &IntelligenceSignal{
		TenantID:      tenantID,
		SignalType:    signalType,
		Domain:        domain,
		Severity:      severity,
		SourceSystem:  ev.Source,
		SourceEventID: ev.ID,
		Status:        "ACTIVE",
		CorrelationID: ev.Correlation,
		Evidence:      evidence,
		CreatedAt:     utcNow(),
		Payload: map[string]interface{}{
			"object_canonical_id": fmt.Sprintf("uoi_%s_%s_%s", ev.Source, ev.ObjectType, ev.ObjectID),
			"event_type":          ev.EventType,
			"freshness":           freshness,
		},
	}
}

// isRiskRelevantEvent reports whether an event should be correlated as a risk
// intelligence event.
func isRiskRelevantEvent(ev DomainEvent) bool {
	switch ev.EventType {
	case "incident.created", "incident.escalated", "incident.resolved",
		"dataset.quality_degraded", "kpi.declining", "risk.created", "risk.escalated",
		"security.incident", "security.breach", "service.outage":
		return true
	}
	return false
}

// correlateRiskEvent records a risk-event correlation entry. It NEVER creates
// an authoritative risk record.
func correlateRiskEvent(ev DomainEvent, tenantID string) {
	severity := 40
	if v, ok := ev.Payload["severity"].(float64); ok {
		sev := int(v)
		if sev >= 0 && sev <= 100 {
			severity = sev
		}
	}
	objects := []string{}
	if ev.ObjectType != "" && ev.ObjectID != "" {
		objects = append(objects, fmt.Sprintf("uoi_%s_%s_%s", ev.Source, ev.ObjectType, ev.ObjectID))
	}
	_ = phase12.RecordRiskEvent(&RiskEvent{
		TenantID:          tenantID,
		EventType:         "correlation." + ev.EventType,
		SourceSystem:      ev.Source,
		SourceEventID:     ev.ID,
		SeverityEstimate:  severity,
		CorrelatedObjects: objects,
		Evidence:          fmt.Sprintf("Risk-relevant event %s (%s) correlated by intelligence processor.", ev.EventType, ev.ObjectID),
		CorrelationID:     ev.Correlation,
		Status:            "DETECTED",
		CreatedAt:         time.Now().UTC(),
	})
}

