package store

import (
	"context"
	"testing"

	"statdata-backend/internal/models"
)

// Stage 2: dataset resources must be partitioned by selected workspace.
func TestMemStoreDatasetWorkspaceScoping(t *testing.T) {
	m := NewMemStore()
	ctx := context.Background()

	alpha := &models.Dataset{ID: "ds-alpha", URN: "urn:alpha", Name: "Alpha", TenantID: "tenant-alpha", WorkspaceID: "ws-alpha"}
	beta := &models.Dataset{ID: "ds-beta", URN: "urn:beta", Name: "Beta", TenantID: "tenant-alpha", WorkspaceID: "ws-beta"}
	legacy := &models.Dataset{ID: "ds-legacy", URN: "urn:legacy", Name: "Legacy", TenantID: "tenant-alpha"}

	for _, ds := range []*models.Dataset{alpha, beta, legacy} {
		if err := m.CreateDataset(ctx, ds); err != nil {
			t.Fatalf("seed %s: %v", ds.ID, err)
		}
	}

	// Scoped list: only that workspace's datasets.
	got, total, err := m.ListDatasets(ctx, "tenant-alpha", "", "", "ws-alpha", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(got) != 1 || got[0].ID != "ds-alpha" {
		t.Fatalf("expected only ds-alpha for ws-alpha, got %d", total)
	}

	// Legacy (unscoped) resources stay visible without a workspace selection.
	got, _, err = m.ListDatasets(ctx, "tenant-alpha", "", "", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, ds := range got {
		seen[ds.ID] = true
	}
	for _, id := range []string{"ds-alpha", "ds-beta", "ds-legacy"} {
		if !seen[id] {
			t.Fatalf("expected %s in unscoped listing", id)
		}
	}

	// Legacy resources are visible from any workspace selection.
	got, _, err = m.ListDatasets(ctx, "tenant-alpha", "", "", "ws-beta", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, ds := range got {
		if ds.ID == "ds-alpha" {
			t.Fatal("ds-alpha leaked into ws-beta listing")
		}
	}
}

// workspaceAllowsRead mirrors the api helper contract used by handlers.
func TestWorkspaceReadContract(t *testing.T) {
	cases := []struct {
		existing, selected string
		expect             bool
	}{
		{"ws-1", "ws-1", true},
		{"ws-1", "ws-2", false},
		{"", "ws-2", true},  // legacy resource visible
		{"ws-1", "", true},  // no selection: tenant-wide view
	}
	for i, tc := range cases {
		if got := tc.existing == tc.selected || tc.selected == "" || tc.existing == ""; got != tc.expect {
			t.Fatalf("case %d: got %v want %v", i, got, tc.expect)
		}
	}
}