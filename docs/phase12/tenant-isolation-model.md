# Tenant Isolation Model
## StatGate Phase XII — Sprint 0 Architecture Validation
**Document:** `docs/phase12/tenant-isolation-model.md`
**Status:** APPROVED

---

## Principle

All Phase XII objects are tenant-scoped. Cross-tenant intelligence leakage is prohibited by design, enforced at both the application layer and the database query layer.

---

## Scope of Isolation

The following Phase XII entities must be tenant-isolated:

| Entity | Isolation Mechanism |
|---|---|
| Knowledge graph nodes (`institutional_objects`) | `WHERE tenant_id = $tenant` on all queries |
| Knowledge graph edges (`institutional_graph_edges`) | `WHERE tenant_id = $tenant`; cross-tenant edges blocked on creation |
| Intelligence signals | `WHERE tenant_id = $tenant` |
| Institutional conditions | `WHERE tenant_id = $tenant` |
| KPI definitions and measurements | `WHERE tenant_id = $tenant` |
| Objectives | `WHERE tenant_id = $tenant` |
| Risks | `WHERE tenant_id = $tenant` |
| AI recommendations | `WHERE tenant_id = $tenant` |
| AI audit records | `WHERE tenant_id = $tenant` |

---

## Graph Edge Cross-Tenant Prevention

When creating a graph edge, the following validation is enforced:

```
1. Resolve subject_canonical_id → institutional_objects → verify tenant_id == actor.tenant_id
2. Resolve object_canonical_id  → institutional_objects → verify tenant_id == actor.tenant_id
3. If either fails → return 403 Forbidden + audit record
4. If both pass   → insert edge with tenant_id = actor.tenant_id
```

This prevents Tenant A from creating a relationship to a Tenant B object even if Tenant A knows the canonical ID of a Tenant B object.

---

## Graph Traversal Tenant Enforcement

During multi-hop traversal, every intermediate node is filtered by `tenant_id`. The traversal cannot "escape" to another tenant's subgraph mid-hop.

```sql
-- Example: two-hop traversal (adjacency list CTE)
WITH RECURSIVE graph_walk AS (
    -- Base: direct neighbours of start node
    SELECT e.object_canonical_id AS node_id, 1 AS depth
    FROM institutional_graph_edges e
    WHERE e.subject_canonical_id = $start
      AND e.tenant_id = $tenant          -- ← always enforced
      AND e.valid_until IS NULL
    UNION
    -- Recursive: next hop
    SELECT e.object_canonical_id, gw.depth + 1
    FROM institutional_graph_edges e
    JOIN graph_walk gw ON e.subject_canonical_id = gw.node_id
    WHERE e.tenant_id = $tenant          -- ← always enforced
      AND gw.depth < $max_depth          -- ← depth limit enforced
      AND e.valid_until IS NULL
)
SELECT o.* FROM institutional_objects o
JOIN graph_walk gw ON o.canonical_id = gw.node_id
WHERE o.tenant_id = $tenant;             -- ← always enforced on final fetch
```

---

## JWT Tenant Mismatch Handling

If an authenticated request carries a JWT with `tenant_id = A` but the request URL or body specifies `tenant_id = B`:
- The application uses the JWT `tenant_id` (not the supplied value)
- The mismatch is logged as a security event
- The response uses only Tenant A data

---

## Tenant Isolation Test Cases (Sprint 0)

Verified in the knowledge graph prototype (`knowledge_graph_test.go`):

| Test | Expected Result | Status |
|---|---|---|
| Tenant A requests Tenant A objects | Returns Tenant A data | PASS |
| Tenant A requests Tenant B object by direct ID | Returns 404 | PASS |
| Tenant A creates edge to Tenant B object | Returns 403 | PASS |
| Traversal from Tenant A node cannot reach Tenant B node | Returns 0 cross-tenant nodes | PASS |
| JWT tenant_id mismatch | JWT tenant_id used; mismatch logged | PASS |
| Unauthorized AI recommendation access (wrong tenant) | Returns 404 | PASS |

---

*Status: APPROVED*
*Sprint 0 Gate: TENANT ISOLATION — PASS*
