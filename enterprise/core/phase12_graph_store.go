package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — KNOWLEDGE GRAPH STORE OPERATIONS
//
// Graph edges are append/correction oriented (directive §8, §32): deletion
// retires an edge by time-boxing ValidUntil — historical evidence is never
// silently destroyed. Tenant scoping is enforced on every operation.
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"fmt"
	"time"
)

// CreateEdge adds a graph edge. Relationship type, provenance, confidence and
// endpoint validity are enforced here. The API layer enforces role authorization.
func (s *Phase12Store) CreateEdge(e *GraphEdge) error {
	if e == nil {
		return fmt.Errorf("graph edge is nil")
	}
	if e.TenantID == "" {
		return fmt.Errorf("graph edge requires tenant_id")
	}
	if e.SubjectCanonicalID == "" || e.ObjectCanonicalID == "" {
		return fmt.Errorf("graph edge requires subject_canonical_id and object_canonical_id")
	}
	if e.SubjectCanonicalID == e.ObjectCanonicalID {
		return fmt.Errorf("self-referencing graph edges are not permitted")
	}
	if !validRelationshipType(e.RelationshipType) {
		return fmt.Errorf("invalid relationship_type %q", e.RelationshipType)
	}
	if e.ProvenanceType == "" {
		e.ProvenanceType = ProvenanceAuthoritative
	}
	if !validProvenance(e.ProvenanceType) {
		return fmt.Errorf("invalid provenance_type %q", e.ProvenanceType)
	}
	if e.Confidence < 0 || e.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	if e.ValidFrom.IsZero() {
		e.ValidFrom = utcNow()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = utcNow()
	}
	if e.ID == "" {
		e.ID = fmt.Sprintf("gedge_%d", time.Now().UnixNano())
	}

	s.mu.Lock()
	s.edges[e.TenantID] = append(s.edges[e.TenantID], e)
	s.mu.Unlock()
	persistPhase12Edge(e)
	recordAuditEvent(e.CreatedBy, "graph.edge.create", "institutional_graph_edges", e.ID, e.TenantID, "", "", map[string]interface{}{
		"subject_canonical_id": e.SubjectCanonicalID,
		"relationship_type":    e.RelationshipType,
		"object_canonical_id":  e.ObjectCanonicalID,
		"provenance_type":      e.ProvenanceType,
	})
	return nil
}

// ListEdges returns all currently valid edges for a tenant.
func (s *Phase12Store) ListEdges(tenantID string) []*GraphEdge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*GraphEdge
	now := utcNow()
	for _, e := range s.edges[tenantID] {
		if e.ValidUntil != nil && !e.ValidUntil.After(now) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// DeleteEdge retires an edge (append/correction oriented). Historical evidence
// is never silently destroyed; the edge is time-boxed.
func (s *Phase12Store) DeleteEdge(tenantID, edgeID, actor string) error {
	s.mu.Lock()
	edges := s.edges[tenantID]
	now := utcNow()
	for _, e := range edges {
		if e.ID == edgeID && (e.ValidUntil == nil || e.ValidUntil.After(now)) {
			e.ValidUntil = &now
			s.mu.Unlock()
			retirePhase12Edge(e)
			recordAuditEvent(actor, "graph.edge.delete", "institutional_graph_edges", edgeID, tenantID, "", "", map[string]interface{}{
				"subject_canonical_id": e.SubjectCanonicalID,
				"relationship_type":    e.RelationshipType,
				"object_canonical_id":  e.ObjectCanonicalID,
			})
			return nil
		}
	}
	s.mu.Unlock()
	return fmt.Errorf("active edge %q not found for tenant", edgeID)
}

// GetEdge returns a specific active edge or nil.
func (s *Phase12Store) GetEdge(tenantID, edgeID string) *GraphEdge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := utcNow()
	for _, e := range s.edges[tenantID] {
		if e.ID == edgeID && (e.ValidUntil == nil || e.ValidUntil.After(now)) {
			return e
		}
	}
	return nil
}

// DirectRelations returns active edges from a canonical object.
func (s *Phase12Store) DirectRelations(tenantID, canonicalID string) []*GraphEdge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*GraphEdge
	now := utcNow()
	for _, e := range s.edges[tenantID] {
		if e.SubjectCanonicalID == canonicalID && (e.ValidUntil == nil || e.ValidUntil.After(now)) {
			out = append(out, e)
		}
	}
	return out
}