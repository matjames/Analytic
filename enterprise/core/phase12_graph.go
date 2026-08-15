package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — KNOWLEDGE GRAPH ENGINE
//
// PostgreSQL adjacency-list architecture (directive §7). Traversal is
// performed over the tenant-scoped edge list with a hard maximum depth
// (default 5 hops). All queries are NAMED, parameterized queries — the API
// NEVER accepts arbitrary graph/SQL expressions from clients (directive §27).
//
// Every traversal:
//   - enforces tenant_id (no cross-tenant traversal is possible)
//   - enforces maximum traversal depth
//   - excludes retired (valid_until in past) edges
//   - is measurable via statgate_graph_queries_total / _duration metrics
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"
)

const (
	// DefaultMaxTraversalDepth is the maximum number of hops a traversal may take.
	DefaultMaxTraversalDepth = 5
	// BenchmarkMaxTraversalDepth is benchmark-only (directive §28).
	BenchmarkMaxTraversalDepth = 7
)

// Traverse performs a depth-limited breadth-first traversal from a start
// canonical object within a single tenant. maxDepth > 5 is permitted only
// when explicitly enabled at the call site (never from ordinary clients).
func (s *Phase12Store) Traverse(tenantID, startCanonicalID string, maxDepth int) ([]*GraphTraversalPath, error) {
	if tenantID == "" || startCanonicalID == "" {
		return nil, fmt.Errorf("traversal requires tenant_id and start canonical_id")
	}
	if maxDepth < 1 || maxDepth > BenchmarkMaxTraversalDepth {
		return nil, fmt.Errorf("traversal depth must be between 1 and %d hops", BenchmarkMaxTraversalDepth)
	}

	startT := time.Now()
	paths := s.traverseBFS(tenantID, startCanonicalID, maxDepth)
	statgateGraphQueries.Add(1)
	statgateGraphQueryDuration.Add(time.Since(startT).Nanoseconds())
	return paths, nil
}

// traverseBFS implements the adjacency-list traversal without recursion.
// Paths are returned as ordered node chains. Cycles are prevented by
// never revisiting a node within a single path.
func (s *Phase12Store) traverseBFS(tenantID, start string, maxDepth int) []*GraphTraversalPath {
	s.mu.RLock()
	edges := s.edges[tenantID]
	s.mu.RUnlock()

	now := time.Now().UTC()
	active := make([]*GraphEdge, 0, len(edges))
	for _, e := range edges {
		if e.ValidUntil != nil && !e.ValidUntil.After(now) {
			continue
		}
		active = append(active, e)
	}

	// adjacency: subject -> list of active edges
	adj := map[string][]*GraphEdge{}
	for _, e := range active {
		adj[e.SubjectCanonicalID] = append(adj[e.SubjectCanonicalID], e)
	}

	type frontierItem struct {
		node  string
		hops  int
		nodes []string
		edges []GraphEdgeRef
	}

	paths := []*GraphTraversalPath{}
	frontier := []frontierItem{{node: start, hops: 0, nodes: []string{start}, edges: nil}}

	for len(frontier) > 0 {
		item := frontier[0]
		frontier = frontier[1:]

		// Record the path when we have moved at least one hop.
		if item.hops > 0 {
			paths = append(paths, &GraphTraversalPath{
				Hops:  item.hops,
				Nodes: item.nodes,
				Edges: item.edges,
			})
		}
		if item.hops >= maxDepth {
			continue
		}

		for _, e := range adj[item.node] {
			next := e.ObjectCanonicalID
			// No revisits within a path (cycle prevention).
			revisit := false
			for _, n := range item.nodes {
				if n == next {
					revisit = true
					break
				}
			}
			if revisit {
				continue
			}
			newNodes := make([]string, len(item.nodes)+1)
			copy(newNodes, item.nodes)
			newNodes[len(item.nodes)] = next
			newEdges := append(append([]GraphEdgeRef{}, item.edges...), GraphEdgeRef{
				ID:               e.ID,
				RelationshipType: e.RelationshipType,
				From:             item.node,
				To:               next,
			})
			frontier = append(frontier, frontierItem{
				node:  next,
				hops:  item.hops + 1,
				nodes: newNodes,
				edges: newEdges,
			})
		}
	}

	// Deterministic ordering for stable responses.
	sort.Slice(paths, func(i, j int) bool {
		if paths[i].Hops != paths[j].Hops {
			return paths[i].Hops < paths[j].Hops
		}
		return paths[i].Nodes[len(paths[i].Nodes)-1] < paths[j].Nodes[len(paths[j].Nodes)-1]
	})
	return paths
}

// ─── Metrics ──────────────────────────────────────────────────────────────────

var (
	statgateGraphQueries        atomic.Int64
	statgateGraphQueryDuration  atomic.Int64 // nanoseconds
	statgateIntelligenceEvents  atomic.Int64
	statgateConditionCalcs      atomic.Int64
	statgateConditionCalcDurNs  atomic.Int64
	statgateAIRecommendations   atomic.Int64
	statgateAIRecommendationsPending atomic.Int64
)

// PhaseXIIMetrics returns the Phase XII observability snapshot (directive §29).
func PhaseXIIMetrics() map[string]interface{} {
	return map[string]interface{}{
		"statgate_intelligence_events_processed_total":           statgateIntelligenceEvents.Load(),
		"statgate_intelligence_condition_calculations_total":     statgateConditionCalcs.Load(),
		"statgate_intelligence_condition_calculation_duration_ms": float64(statgateConditionCalcDurNs.Load()) / 1e6,
		"statgate_graph_queries_total":                           statgateGraphQueries.Load(),
		"statgate_graph_query_duration_ms":                       float64(statgateGraphQueryDuration.Load()) / 1e6,
		"statgate_ai_recommendations_total":                      statgateAIRecommendations.Load(),
		"statgate_ai_recommendations_pending":                    statgateAIRecommendationsPending.Load(),
	}
}
