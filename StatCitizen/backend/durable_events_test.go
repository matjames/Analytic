package main

import (
	"testing"
)

func TestCitizenEventUsesCanonicalDurableEnvelope(t *testing.T) {
	event := newCitizenEnterpriseEvent(EventPublicationPublished, "consultation", "consultation-1", "tenant-alpha", "citizen-admin", "corr-1", map[string]interface{}{"title": "Public consultation"})
	if event.EventID == "" || event.Source != "statcitizen" || event.TenantID != "tenant-alpha" || event.Version != "1.0" {
		t.Fatalf("unexpected canonical event envelope: %+v", event)
	}
}
