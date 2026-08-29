package main

import "testing"

func TestNewGovernanceEventUsesCanonicalEnvelope(t *testing.T) {
	evt := newGovernanceEvent(
		"risk.created",
		"risk",
		"risk-123",
		"user-456",
		"tenant-789",
		map[string]interface{}{"title": "Supply risk"},
	)

	if evt.Source != "statgovernance" || evt.Version != "1.0" {
		t.Fatalf("unexpected source/version: %q %q", evt.Source, evt.Version)
	}
	if evt.TenantID != "tenant-789" || evt.UserID != "user-456" {
		t.Fatalf("identity context was not preserved: tenant=%q user=%q", evt.TenantID, evt.UserID)
	}
	if evt.EventType != "risk.created" || evt.ObjectType != "risk" || evt.ObjectID != "risk-123" {
		t.Fatalf("object context was not preserved: %#v", evt)
	}
	if evt.CorrelationID == "" || evt.Timestamp.IsZero() {
		t.Fatal("canonical correlation and timestamp fields must be populated")
	}
}
