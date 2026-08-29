package diplomacy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"statfederation-backend/internal/models"
	"statfederation-backend/internal/store"
)

// peerServer returns an httptest server that responds to /api/public/search
// with the supplied JSON body.
func peerServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/public/search" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(ts.Close)
	return ts
}

// registerFederatedNode registers a node plus (optionally) an active DSA where
// the peer provides search results to the source hub.
func registerFederatedNode(t *testing.T, s *store.MemStore, nodeID, name, endpointURL string, withDSA bool) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	node := &models.FederatedNode{
		ID:           nodeID,
		Name:         name,
		Code:         "C-" + nodeID,
		NodeType:     models.NodeTypeNSSAgency,
		Jurisdiction: "NATIONAL",
		EndpointURL:  endpointURL,
		HealthStatus: models.HealthStatusHealthy,
		TrustLevel:   models.TrustLevelTier2DomesticAgency,
		TenantID:     "default",
	}
	if err := s.RegisterNode(ctx, node); err != nil {
		t.Fatalf("register node %s: %v", nodeID, err)
	}
	if !withDSA {
		return
	}

	dsa := &models.DataSharingAgreement{
		ID:               "dsa-" + nodeID,
		DSANumber:        "DSA-" + nodeID + "-2026",
		Title:            "Search DSA " + nodeID,
		ProviderNodeID:   nodeID,
		ConsumerNodeID:   "node-nss-001", // the source hub consumes peer results
		Status:           models.DSAStatusActive,
		AccessTier:       "INTER_AGENCY",
		PermittedDomains: []string{"search", "metadata"},
		Purpose:          "Federated search broadcast",
		ValidFrom:        now.Add(-time.Hour),
		ValidUntil:       now.Add(24 * time.Hour),
		DailyQuota:       100000,
		TenantID:         "default",
	}
	if err := s.CreateDSA(ctx, dsa); err != nil {
		t.Fatalf("create dsa for %s: %v", nodeID, err)
	}
}

// registerSourceHub registers this hub's own node so the DSA gate knows its ID.
func registerSourceHub(t *testing.T, s *store.MemStore) {
	t.Helper()
	err := s.RegisterNode(context.Background(), &models.FederatedNode{
		ID:           "node-nss-001",
		Name:         "StatGate National Hub",
		Code:         "NSS-HQ",
		NodeType:     models.NodeTypeNSSAgency,
		Jurisdiction: "NATIONAL",
		EndpointURL:  "https://self-statgate.example",
		HealthStatus: models.HealthStatusHealthy,
		TrustLevel:   models.TrustLevelTier1Sovereign,
		TenantID:     "default",
	})
	if err != nil {
		t.Fatalf("register source hub: %v", err)
	}
}
// TestFederatedSearchBroadcast verifies that a query is dispatched to a
// DSA-authorised peer and its results are merged into the local response.
func TestFederatedSearchBroadcast(t *testing.T) {
	mem := store.NewMemStore()
	registerSourceHub(t, mem)

	peer := peerServer(t, `[{"remote_resource_id":"hmis-malaria-001","title":"Malaria Incidence by District","abstract":"From Ministry of Health HMIS","node_id":"node-moh-002","node_name":"Ministry of Health","score":0.94}]`)
	registerFederatedNode(t, mem, "node-moh-002", "Ministry of Health", peer.URL, true)

	fsh := NewFederatedSearchHub(mem)
	fsh.SetSourceNodeID("node-nss-001")

	results, err := fsh.ExecuteFederatedSearch(context.Background(), SearchQuery{Keyword: "malaria"}, "default")
	if err != nil {
		t.Fatalf("ExecuteFederatedSearch: %v", err)
	}

	found := false
	for _, r := range results {
		if r.RemoteResourceID == "hmis-malaria-001" {
			found = true
			if r.NodeID != "node-moh-002" {
				t.Errorf("expected peer node id preserved, got %q", r.NodeID)
			}
			if r.NodeName == "" {
				t.Error("expected peer node name filled from registry")
			}
		}
	}
	if !found {
		t.Fatalf("expected broadcast result from peer to be merged, got %+v", results)
	}
}

// TestFederatedSearchDSAGate verifies that peers without an active DSA are not
// queried (broadcast is skipped).
func TestFederatedSearchDSAGate(t *testing.T) {
	mem := store.NewMemStore()
	registerSourceHub(t, mem)

	// Peer exists but has NO active DSA with the source hub.
	peer := peerServer(t, `[{"remote_resource_id":"unknown-1","title":"Should Not Appear","score":0.5}]`)
	registerFederatedNode(t, mem, "node-rogue-999", "Rogue Peer", peer.URL, false)

	fsh := NewFederatedSearchHub(mem)
	fsh.SetSourceNodeID("node-nss-001")

	results, err := fsh.ExecuteFederatedSearch(context.Background(), SearchQuery{Keyword: "appear"}, "default")
	if err != nil {
		t.Fatalf("ExecuteFederatedSearch: %v", err)
	}

	for _, r := range results {
		if r.RemoteResourceID == "unknown-1" {
			t.Fatal("expected peer without a DSA to be skipped")
		}
	}
}

// TestFederatedSearchRankAndDedupe verifies score ordering and duplicate removal.
func TestFederatedSearchRankAndDedupe(t *testing.T) {
	fsh := NewFederatedSearchHub(store.NewMemStore())
	input := []*models.FederatedSearchResult{
		{RemoteResourceID: "a", NodeID: "n1", Title: "Low", Score: 0.2},
		{RemoteResourceID: "b", NodeID: "n1", Title: "High", Score: 0.95},
		{RemoteResourceID: "a", NodeID: "n1", Title: "Low Duplicate", Score: 0.99},
		{RemoteResourceID: "c", NodeID: "n1", Title: "Mid", Score: 0.5},
		{RemoteResourceID: "d", NodeID: "n1", Title: "Unscored", Score: 0},
	}

	out := fsh.rankAndDedupe(input)
	if len(out) != 4 {
		t.Fatalf("expected 4 unique results, got %d", len(out))
	}
	if out[0].Title != "High" || out[0].Score != 0.95 {
		t.Errorf("expected highest score first, got %+v", out[0])
	}
	if out[1].Title != "Mid" || out[1].Score != 0.5 {
		t.Errorf("expected second-highest, got %+v", out[1])
	}
	if out[2].Title != "Low" || out[2].Score != 0.2 {
		t.Errorf("expected low score, got %+v", out[2])
	}
	if out[3].Title != "Unscored" {
		t.Errorf("expected unscored last, got %+v", out[3])
	}
}

// TestFederatedSearchParsePeerResults verifies the response envelope parsing.
func TestFederatedSearchParsePeerResults(t *testing.T) {
	fsh := NewFederatedSearchHub(store.NewMemStore())

	bare := fsh.parsePeerResults([]byte(`[{"remote_resource_id":"x","title":"X","score":1}]`))
	if len(bare) != 1 {
		t.Fatalf("expected 1 bare-array result, got %d", len(bare))
	}
	wrapped := fsh.parsePeerResults([]byte(`{"results":[{"remote_resource_id":"y","title":"Y","score":1}]}`))
	if len(wrapped) != 1 {
		t.Fatalf("expected 1 wrapped result, got %d", len(wrapped))
	}
	databox := fsh.parsePeerResults([]byte(`{"data":[{"remote_resource_id":"z","title":"Z","score":1}]}`))
	if len(databox) != 1 {
		t.Fatalf("expected 1 data-envelope result, got %d", len(databox))
	}
	bad := fsh.parsePeerResults([]byte(`not json`))
	if bad != nil {
		t.Fatalf("expected nil for invalid json, got %+v", bad)
	}
}