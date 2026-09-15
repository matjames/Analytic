package main

import (
	"context"
	"testing"

	"github.com/matjames/statgate-lib/events"
)

func TestCanonicalEventLogIsTenantAndWorkspaceScoped(t *testing.T) {
	globalStore = NewMemStore()

	if err := handleCanonicalEvent(context.Background(), events.EnterpriseEvent{
		EventID:    "event-alpha",
		EventType:  "ops.pipeline.triggered",
		Source:     "statdata",
		ObjectType: "pipeline",
		ObjectID:   "pipeline-alpha",
		TenantID:   "tenant-alpha",
		Payload:    map[string]interface{}{"workspace_id": "workspace-alpha"},
	}); err != nil {
		t.Fatalf("handle alpha event: %v", err)
	}
	if err := handleCanonicalEvent(context.Background(), events.EnterpriseEvent{
		EventID:    "event-beta",
		EventType:  "ops.pipeline.triggered",
		Source:     "statdata",
		ObjectType: "pipeline",
		ObjectID:   "pipeline-beta",
		TenantID:   "tenant-alpha",
		Payload:    map[string]interface{}{"workspace_id": "workspace-beta"},
	}); err != nil {
		t.Fatalf("handle beta event: %v", err)
	}

	alpha := globalStore.ListLogs("tenant-alpha", "workspace-alpha")
	if len(alpha) != 1 || alpha[0].WorkspaceID != "workspace-alpha" {
		t.Fatalf("expected one alpha workspace log, got %+v", alpha)
	}
	otherTenant := globalStore.ListLogs("tenant-beta", "workspace-alpha")
	if len(otherTenant) != 0 {
		t.Fatalf("expected no cross-tenant logs, got %+v", otherTenant)
	}
	tenantWide := globalStore.ListLogs("tenant-alpha", "")
	if len(tenantWide) != 2 {
		t.Fatalf("expected both tenant logs in tenant-wide view, got %+v", tenantWide)
	}
}
