# PHASE_XII_KNOWLEDGE_GRAPH.md
# StatGate Phase XII — Knowledge Graph Design

## Architecture
- **Approved default:** PostgreSQL adjacency-list (`institutional_graph_edges`) with
  indexes on `(tenant_id, subject_canonical_id)` and `(tenant_id, object_canonical_id)`.
- **No graph database** (Apache AGE/ltree) is introduced during Phase XII.
- Traversal uses depth-limited BFS (not recursive SQL) in the engine; the same
  adjacency model is materialized in PostgreSQL for durability and enables future
  recursive-CTE queries.

## Mandatory protections (directive §7)
Every graph query enforces:
1. `tenant_id` — scoping is mandatory; cross-tenant traversal returns 0 results.
2. **authorization** — read: any authenticated user; mutation: `admin` /
   `institutional_lead`.
3. **maximum traversal depth** — default `5` hops; `7` is benchmark-only.

No cross-tenant graph traversal is permitted.

## Relationship types (initial)
`SUPPORTS, CONTRIBUTES_TO, AFFECTS, DEPENDS_ON, PRODUCED_BY, OWNED_BY,
ASSIGNED_TO, FUNDED_BY, GOVERNED_BY, ASSOCIATED_WITH, OBSERVED_IN,
REFERENCED_BY, ALERTS_ON`

## Edge model
Each edge carries: `tenant_id, subject_canonical_id, relationship_type,
object_canonical_id, metadata, provenance_type (AUTHORITATIVE|DERIVED|INFERRED|AI_GENERATED),
confidence (0..1), valid_from, valid_until, source_event_id, created_by, created_at`.

Provenance ensures an `AI_GENERATED` or `INFERRED` edge is never presented as an
`AUTHORITATIVE` one. Self-referencing edges are rejected.

## Mutation policy (directive §8)
Creation/modification is privileged. Every mutation produces an audit event
(`graph.edge.create`, `graph.edge.delete`, `graph.object.create`,
`graph.object.update`). Deletion is **append/correction oriented**: an edge is
time-boxed via `valid_until`; historical evidence is never silently destroyed.

## Named queries only (directive §27)
The API exposes only named, parameterized queries:
- `projects-affected-by-incident/:id`
- `datasets-supporting-kpi/:id`
- `objectives-at-risk`
- `object-neighborhood/:id`

Arbitrary Cypher/SQL-like graph expressions from clients are rejected (fail closed).
This improves security, authorization, performance, auditability, and API stability.
