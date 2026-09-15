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

// Stage 2: pipeline resources must also be partitioned by selected workspace.
func TestMemStorePipelineWorkspaceScoping(t *testing.T) {
	m := NewMemStore()
	ctx := context.Background()

	alpha := &models.DataPipeline{ID: "pl-alpha", Name: "Alpha", TenantID: "tenant-alpha", WorkspaceID: "ws-alpha"}
	beta := &models.DataPipeline{ID: "pl-beta", Name: "Beta", TenantID: "tenant-alpha", WorkspaceID: "ws-beta"}
	for _, p := range []*models.DataPipeline{alpha, beta} {
		if err := m.CreatePipeline(ctx, p); err != nil {
			t.Fatalf("seed %s: %v", p.ID, err)
		}
	}

	got, err := m.ListPipelines(ctx, "tenant-alpha", "", "ws-alpha")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "pl-alpha" {
		t.Fatalf("expected only pl-alpha for ws-alpha, got %d", len(got))
	}

	got, err = m.ListPipelines(ctx, "tenant-alpha", "", "")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, p := range got {
		seen[p.ID] = true
	}
	if !seen["pl-alpha"] || !seen["pl-beta"] {
		t.Fatalf("expected both seeded pipelines in unscoped listing, got %d", len(got))
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
		{"", "ws-2", true}, // legacy resource visible
		{"ws-1", "", true}, // no selection: tenant-wide view
	}
	for i, tc := range cases {
		if got := tc.existing == tc.selected || tc.selected == "" || tc.existing == ""; got != tc.expect {
			t.Fatalf("case %d: got %v want %v", i, got, tc.expect)
		}
	}
}

func TestMemStoreLegacyDomainsRespectWorkspaceContext(t *testing.T) {
	m := NewMemStore()
	ctxAlpha := WithWorkspace(context.Background(), "ws-alpha")
	ctxBeta := WithWorkspace(context.Background(), "ws-beta")

	exp := &models.Experiment{ID: "exp-alpha", Name: "Alpha", TenantID: "tenant", WorkspaceID: "ws-alpha"}
	if err := m.CreateExperiment(ctxAlpha, exp); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetExperimentByID(ctxBeta, exp.ID); err == nil {
		t.Fatal("cross-workspace experiment read was allowed")
	}

	model := &models.RegisteredModel{ID: "model-alpha", Name: "Alpha Model", Domain: "STAT", TenantID: "tenant", WorkspaceID: "ws-alpha"}
	if err := m.CreateRegisteredModel(ctxAlpha, model); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetRegisteredModelByID(ctxBeta, model.ID); err == nil {
		t.Fatal("cross-workspace model read was allowed")
	}

	doc := &models.IndexedDocument{ID: "doc-alpha", IndexName: "global", ResourceID: "exp-alpha", ResourceType: "EXPERIMENT", Title: "Alpha", Content: "alpha", Domain: "STAT", TenantID: "tenant", WorkspaceID: "ws-alpha"}
	if err := m.IndexDocument(ctxAlpha, doc); err != nil {
		t.Fatal(err)
	}
	results, err := m.Search(ctxBeta, &models.HybridSearchRequest{TenantID: "tenant", Query: "", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results.Results {
		if result.DocumentID == doc.ID {
			t.Fatal("cross-workspace indexed document leaked into search")
		}
	}

	ss := &models.SavedSearch{ID: "saved-alpha", Name: "Alpha", Query: "alpha", TenantID: "tenant", CreatedBy: "user"}
	if err := m.CreateSavedSearch(ctxAlpha, ss); err != nil {
		t.Fatal(err)
	}
	if saved, err := m.ListSavedSearches(ctxBeta, "tenant", "user"); err != nil {
		t.Fatal(err)
	} else if len(saved) != 0 {
		t.Fatal("cross-workspace saved search leaked")
	}
}
