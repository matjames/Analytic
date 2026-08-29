package lakehouse

import (
	"testing"
	"time"
)

// TestRecordEventAppendsToLedger verifies that domain events written via
// RecordEvent are queryable and do not spuriously create anomalies.
func TestRecordEventAppendsToLedger(t *testing.T) {
	se := NewStorageEngine()
	baselineAnomalies := len(se.GetAnomalies("tenant-alpha", ""))

	se.RecordEvent(EventRecord{
		TenantID:   "tenant-alpha",
		WorkspaceID: "workspace-demo",
		Source:     "institutional_condition",
		Payload: map[string]interface{}{
			"event_type": "institutional.condition.changed",
			"level":      "OPTIMAL",
		},
		Timestamp: time.Now().UTC(),
		Value:     95.0,
	})

	records := se.QueryTenantData("tenant-alpha", "workspace-demo", 5)
	if len(records) == 0 {
		t.Fatal("expected a recorded event in tenant data")
	}
	last := records[len(records)-1]
	if last.Source != "institutional_condition" {
		t.Errorf("expected institutional_condition source, got %q", last.Source)
	}
	if last.Payload["event_type"] != "institutional.condition.changed" {
		t.Errorf("expected event_type payload, got %v", last.Payload["event_type"])
	}

	after := len(se.GetAnomalies("tenant-alpha", ""))
	if after != baselineAnomalies {
		t.Errorf("RecordEvent must not create anomalies (before=%d after=%d)", baselineAnomalies, after)
	}
}