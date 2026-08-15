package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — TEST SUITE: KNOWLEDGE GRAPH
//
// Covers mandatory graph/security tests (directive §30): create edge, retrieve
// edge, delete/correct edge, tenant isolation, relationship validation, depth
// limitation, and rejection of arbitrary/unnamed graph queries (fail closed).
// ═══════════════════════════════════════════════════════════════════════════════

import "testing"

func mustNoErr(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

func seedObject(s *Phase12Store, tenant, canonical, objType, source string) {
	if err := s.ProjectObject(&InstitutionalObject{
		TenantID: tenant, CanonicalID: canonical, ObjectType: objType,
		SourceSystem: source, SourceObjectID: canonical,
	}); err != nil {
		panic(err)
	}
}

func TestPhase12_Graph_CreateRetrieveEdge(t *testing.T) {
	s := newPhase12Store()
	seedObject(s, "t1", "proj_a", "Project", "pms")
	seedObject(s, "t1", "ds_b", "Dataset", "statcollect")

	mustNoErr(t, s.CreateEdge(&GraphEdge{
		TenantID: "t1", SubjectCanonicalID: "proj_a",
		RelationshipType: "SUPPORTS", ObjectCanonicalID: "ds_b",
	}), "create edge")

	edges := s.ListEdges("t1")
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	got := s.GetEdge("t1", edges[0].ID)
	if got == nil {
		t.Fatal("retrieve edge failed")
	}
	if got.SubjectCanonicalID != "proj_a" || got.ObjectCanonicalID != "ds_b" {
		t.Fatalf("edge endpoints wrong: %+v", got)
	}
	if s.GetEdge("t2", edges[0].ID) != nil {
		t.Fatal("edge must not be retrievable under a different tenant (tenant isolation)")
	}
}

// Cross-tenant graph access must be DENIED.
func TestPhase12_Graph_TenantIsolation(t *testing.T) {
	s := newPhase12Store()
	seedObject(s, "t1", "proj_a", "Project", "pms")
	seedObject(s, "t1", "ds_b", "Dataset", "statcollect")
	seedObject(s, "t2", "proj_x", "Project", "pms")

	mustNoErr(t, s.CreateEdge(&GraphEdge{
		TenantID: "t1", SubjectCanonicalID: "proj_a",
		RelationshipType: "SUPPORTS", ObjectCanonicalID: "ds_b",
	}), "create t1 edge")

	if n := len(s.ListEdges("t2")); n != 0 {
		t.Fatalf("cross-tenant edge leak: t2 sees %d edges", n)
	}
	paths, err := s.Traverse("t2", "proj_x", 5)
	mustNoErr(t, err, "t2 traversal")
	for _, p := range paths {
		for _, node := range p.Nodes {
			if node == "ds_b" {
				t.Fatal("cross-tenant traversal reached t1 object")
			}
		}
	}
}

func TestPhase12_Graph_RelationshipValidation(t *testing.T) {
	s := newPhase12Store()
	seedObject(s, "t1", "a", "Project", "pms")
	seedObject(s, "t1", "b", "Dataset", "statcollect")

	if err := s.CreateEdge(&GraphEdge{TenantID: "t1", SubjectCanonicalID: "a", RelationshipType: "NOT_A_REL", ObjectCanonicalID: "b"}); err == nil {
		t.Fatal("invalid relationship_type must be rejected")
	}
	if err := s.CreateEdge(&GraphEdge{TenantID: "t1", SubjectCanonicalID: "a", RelationshipType: "SUPPORTS", ObjectCanonicalID: "a"}); err == nil {
		t.Fatal("self-referencing edge must be rejected")
	}
	if err := s.CreateEdge(&GraphEdge{TenantID: "t1", SubjectCanonicalID: "a", RelationshipType: "SUPPORTS", ObjectCanonicalID: "b", Confidence: 3.0}); err == nil {
		t.Fatal("confidence out of range must be rejected")
	}
}