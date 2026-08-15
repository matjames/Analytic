package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — NAMED GRAPH QUERIES
//
// The API exposes only named, parameterized queries (directive §27). Arbitrary
// graph expressions from clients are NEVER accepted. Each named query maps to
// a deterministic traversal rule and is separately auditable.
// ═══════════════════════════════════════════════════════════════════════════════

import "fmt"

// NamedGraphQuery is the set of approved named queries.
type NamedGraphQuery string

const (
	// QueryProjectsAffectedByIncident: projects reached from an incident within 5 hops.
	QueryProjectsAffectedByIncident NamedGraphQuery = "projects-affected-by-incident"
	// QueryDatasetsSupportingKPI: datasets that support a KPI/objective within 5 hops.
	QueryDatasetsSupportingKPI NamedGraphQuery = "datasets-supporting-kpi"
	// QueryObjectivesAtRisk: objectives connected to signals of elevated severity.
	QueryObjectivesAtRisk NamedGraphQuery = "objectives-at-risk"
	// QueryObjectNeighborhood: the 5-hop neighbourhood of a canonical object.
	QueryObjectNeighborhood NamedGraphQuery = "object-neighborhood"
)

// validNamedGraphQuery reports whether a query name is approved.
func validNamedGraphQuery(name string) bool {
	switch NamedGraphQuery(name) {
	case QueryProjectsAffectedByIncident, QueryDatasetsSupportingKPI, QueryObjectivesAtRisk, QueryObjectNeighborhood:
		return true
	}
	return false
}

// namedQueryResult is the standardized result envelope for named queries.
type namedQueryResult struct {
	Query       NamedGraphQuery       `json:"query"`
	CanonicalID string                `json:"canonical_id,omitempty"`
	Count       int                   `json:"count"`
	Paths       []*GraphTraversalPath `json:"paths"`
	Notes       []string              `json:"notes,omitempty"`
}

// RunNamedGraphQuery dispatches to an approved named query only. It returns an
// error for any query name not in the approved set (fail closed).
func (s *Phase12Store) RunNamedGraphQuery(tenantID, queryName, param string) (*namedQueryResult, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("named graph query requires tenant_id")
	}
	if !validNamedGraphQuery(queryName) {
		return nil, fmt.Errorf("query %q is not an approved named graph query", queryName)
	}
	switch NamedGraphQuery(queryName) {
	case QueryProjectsAffectedByIncident:
		return s.queryProjectsAffectedByIncident(tenantID, param)
	case QueryDatasetsSupportingKPI:
		return s.queryDatasetsSupportingKPI(tenantID, param)
	case QueryObjectivesAtRisk:
		return s.queryObjectivesAtRisk(tenantID)
	case QueryObjectNeighborhood:
		return s.queryObjectNeighborhood(tenantID, param)
	}
	return nil, fmt.Errorf("query %q is not implemented", queryName)
}

// queryProjectsAffectedByIncident returns all paths from an incident to any
// Project within DefaultMaxTraversalDepth hops.
func (s *Phase12Store) queryProjectsAffectedByIncident(tenantID, incidentID string) (*namedQueryResult, error) {
	if incidentID == "" {
		return nil, fmt.Errorf("incident canonical_id is required")
	}
	paths, err := s.Traverse(tenantID, incidentID, DefaultMaxTraversalDepth)
	if err != nil {
		return nil, err
	}
	result := &namedQueryResult{Query: QueryProjectsAffectedByIncident, CanonicalID: incidentID}
	seen := map[string]bool{}
	for _, p := range paths {
		last := p.Nodes[len(p.Nodes)-1]
		obj := s.GetObject(tenantID, last)
		if obj != nil && obj.ObjectType == "Project" && !seen[last] {
			seen[last] = true
			result.Paths = append(result.Paths, p)
		}
	}
	result.Count = len(result.Paths)
	if result.Count == 0 {
		result.Notes = []string{"NO DATA: no projects correlated to this incident"}
	}
	return result, nil
}

// queryDatasetsSupportingKPI returns paths from a KPI/objective canonical id
// to any Dataset that supports it within DefaultMaxTraversalDepth hops.
func (s *Phase12Store) queryDatasetsSupportingKPI(tenantID, kpiID string) (*namedQueryResult, error) {
	if kpiID == "" {
		return nil, fmt.Errorf("kpi canonical_id is required")
	}
	paths, err := s.Traverse(tenantID, kpiID, DefaultMaxTraversalDepth)
	if err != nil {
		return nil, err
	}
	result := &namedQueryResult{Query: QueryDatasetsSupportingKPI, CanonicalID: kpiID}
	seen := map[string]bool{}
	for _, p := range paths {
		last := p.Nodes[len(p.Nodes)-1]
		obj := s.GetObject(tenantID, last)
		if obj != nil && obj.ObjectType == "Dataset" && !seen[last] {
			seen[last] = true
			result.Paths = append(result.Paths, p)
		}
	}
	result.Count = len(result.Paths)
	if result.Count == 0 {
		result.Notes = []string{"NO DATA: no datasets correlated to this KPI"}
	}
	return result, nil
}
// queryObjectivesAtRisk returns paths to Objectives connected (within 5 hops)
// to elevated/critical signal-bearing objects.
func (s *Phase12Store) queryObjectivesAtRisk(tenantID string) (*namedQueryResult, error) {
	result := &namedQueryResult{Query: QueryObjectivesAtRisk}
	highSeverityObjects := map[string]bool{}
	for _, sig := range s.ListSignals(tenantID) {
		if sig.Severity >= 60 {
			if oid, ok := sig.Payload["object_canonical_id"].(string); ok && oid != "" {
				highSeverityObjects[oid] = true
			}
		}
	}
	seen := map[string]bool{}
	for objID := range highSeverityObjects {
		paths, err := s.Traverse(tenantID, objID, DefaultMaxTraversalDepth)
		if err != nil {
			continue
		}
		for _, p := range paths {
			last := p.Nodes[len(p.Nodes)-1]
			o := s.GetObject(tenantID, last)
			if o != nil && o.ObjectType == "Objective" && !seen[last] {
				seen[last] = true
				result.Paths = append(result.Paths, p)
			}
		}
	}
	result.Count = len(result.Paths)
	if result.Count == 0 {
		result.Notes = []string{"NO DATA: no objectives currently correlated to elevated signals"}
	}
	return result, nil
}

// queryObjectNeighborhood returns the full depth-limited neighbourhood of a
// canonical object (used by KnowledgeGraphView and DataLineageView).
func (s *Phase12Store) queryObjectNeighborhood(tenantID, canonicalID string) (*namedQueryResult, error) {
	if canonicalID == "" {
		return nil, fmt.Errorf("canonical_id is required")
	}
	paths, err := s.Traverse(tenantID, canonicalID, DefaultMaxTraversalDepth)
	if err != nil {
		return nil, err
	}
	result := &namedQueryResult{Query: QueryObjectNeighborhood, CanonicalID: canonicalID, Paths: paths}
	result.Count = len(paths)
	if result.Count == 0 {
		result.Notes = []string{"NO DATA: no relationships found for this object"}
	}
	return result, nil
}

