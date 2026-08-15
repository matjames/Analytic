package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — TEST SUITE: GRAPH MUTATION POLICY, DEPTH, NAMED QUERIES & REGISTRY
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"testing"
)

// Delete/correct edge is append-oriented: history is time-boxed, never destroyed.
func TestPhase12_Graph_DeleteEdgePreservesHistory(t *testing.T) {
	s := newPhase12Store()
	seedObject(s, "t1", "a", "Project", "pms")
	seedObject(s, "t1", "b", "Dataset", "statcollect")

	e := &GraphEdge{TenantID: "t1", SubjectCanonicalID: "a", RelationshipType: "SUPPORTS", ObjectCanonicalID: "b"}
	mustNoErr(t, s.CreateEdge(e), "create edge")
	id := e.ID

	mustNoErr(t, s.DeleteEdge("t1", id, "institutional_lead"), "delete edge")

	if n := len(s.ListEdges("t1")); n != 0 {
		t.Fatalf("retired edge still listed (%d)", n)
	}
	if s.GetEdge("t1", id) != nil {
		t.Fatal("retired edge still retrievable")
	}
	s.mu.RLock()
	exists := false
	for _, stored := range s.edges["t1"] {
		if stored.ID == id && stored.ValidUntil != nil {
			exists = true
		}
	}
	s.mu.RUnlock()
	if !exists {
		t.Fatal("historical edge evidence was destroyed instead of time-boxed")
	}
}

// Graph traversal must enforce maximum depth (default 5 hops).
func TestPhase12_Graph_DepthLimitation(t *testing.T) {
	s := newPhase12Store()
	nodes := []string{"a", "b", "c", "d", "e", "f", "g"}
	for _, n := range nodes {
		seedObject(s, "t1", n, "Project", "pms")
	}
	for i := 0; i < len(nodes)-1; i++ {
		mustNoErr(t, s.CreateEdge(&GraphEdge{
			TenantID: "t1", SubjectCanonicalID: nodes[i],
			RelationshipType: "DEPENDS_ON", ObjectCanonicalID: nodes[i+1],
		}), "chain edge")
	}

	paths, err := s.Traverse("t1", "a", DefaultMaxTraversalDepth)
	mustNoErr(t, err, "traversal")
	maxHops := 0
	for _, p := range paths {
		if p.Hops > maxHops {
			maxHops = p.Hops
		}
	}
	if maxHops > DefaultMaxTraversalDepth {
		t.Fatalf("traversal exceeded max depth %d (got %d)", DefaultMaxTraversalDepth, maxHops)
	}
	if _, err := s.Traverse("t1", "a", 0); err == nil {
		t.Fatal("depth 0 must be rejected")
	}
	if _, err := s.Traverse("t1", "a", 8); err == nil {
		t.Fatal("depth > benchmark max (7) must be rejected")
	}
}

// Only NAMED, parameterized graph queries are permitted (directive §27).
func TestPhase12_Graph_NamedQueriesOnly(t *testing.T) {
	s := newPhase12Store()
	if _, err := s.RunNamedGraphQuery("t1", "DROP TABLE users", ""); err == nil {
		t.Fatal("arbitrary graph query must be rejected (fail closed)")
	}
	// An approved named query runs (returning NO DATA) without error.
	res, err := s.RunNamedGraphQuery("t1", string(QueryObjectNeighborhood), "a")
	if err != nil {
		t.Fatalf("approved named query failed: %v", err)
	}
	if res.Count != 0 {
		t.Fatalf("expected no data, got %d", res.Count)
	}
	if len(res.Notes) == 0 {
		t.Fatal("NO DATA state must be explicit")
	}
}

// Registry: idempotent projection, tenant isolation, object-type validation.
func TestPhase12_Registry_ProjectionAndTenantIsolation(t *testing.T) {
	s := newPhase12Store()
	mustNoErr(t, s.ProjectObject(&InstitutionalObject{TenantID: "t1", CanonicalID: "c1", ObjectType: "Project", SourceSystem: "pms", SourceObjectID: "a"}), "p1")
	mustNoErr(t, s.ProjectObject(&InstitutionalObject{TenantID: "t1", CanonicalID: "c1", ObjectType: "Project", SourceSystem: "pms", SourceObjectID: "a"}), "p2")
	if len(s.ListObjects("t1", "")) != 1 {
		t.Fatal("projection idempotency violated: duplicate canonical object")
	}

	seedObject(s, "t2", "cX", "Dataset", "statcollect")
	for _, o := range s.ListObjects("t1", "") {
		if o.TenantID != "t1" {
			t.Fatalf("tenant isolation leak: object tenant=%s", o.TenantID)
		}
	}
	if err := s.ProjectObject(&InstitutionalObject{TenantID: "t1", CanonicalID: "bad", ObjectType: "Alien", SourceSystem: "x", SourceObjectID: "y"}); err == nil {
		t.Fatal("invalid object_type must be rejected")
	}
}
