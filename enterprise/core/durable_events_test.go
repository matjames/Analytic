package main

import (
	"testing"
	"time"

	"github.com/matjames/statgate-lib/events"
)

func TestEnterpriseEventRoundTripPreservesIdentity(t *testing.T) {
	original := DomainEvent{
		ID:          "event-round-trip",
		EventType:   "project.created",
		Source:      "pms",
		ObjectType:  "project",
		ObjectID:    "project-1",
		Actor:       "user-1",
		TenantID:    "tenant-alpha",
		Payload:     map[string]interface{}{"name": "Certification project"},
		Timestamp:   "2026-09-02T08:00:00Z",
		Correlation: "corr-1",
	}

	canonical := enterpriseEventFromDomain(original)
	if canonical.EventID != original.ID || canonical.UserID != original.Actor || canonical.TenantID != original.TenantID {
		t.Fatalf("domain to enterprise conversion lost identity: %+v", canonical)
	}
	converted := domainEventFromEnterprise(events.EnterpriseEvent{
		EventID:       canonical.EventID,
		EventType:     canonical.EventType,
		Source:        canonical.Source,
		ObjectType:    canonical.ObjectType,
		ObjectID:      canonical.ObjectID,
		TenantID:      canonical.TenantID,
		UserID:        canonical.UserID,
		CorrelationID: canonical.CorrelationID,
		Payload:       canonical.Payload,
		Timestamp:     time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC),
	})
	if converted.ID != original.ID || converted.EventType != original.EventType || converted.Actor != original.Actor || converted.Correlation != original.Correlation {
		t.Fatalf("enterprise to domain conversion lost identity: %+v", converted)
	}
}
