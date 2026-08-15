# StatGate Tenant Isolation (SG-SEC-2026-08)

## Principle

The tenant is always derived from the verified JWT -> authenticated principal
-> authorization policy -> tenant-scoped query. Client-supplied headers are
never authoritative.

    Verified JWT -> Authenticated principal -> Authorization policy
                                                         |
                                                         v
                                     Tenant-scoped database query

## Enforcement points

- Enterprise core: jwtAuthMiddleware sets tenant_id from claims;
  tenantIsolationMiddleware rejects an X-Tenant-ID that does not match the
  JWT claim with 403 "tenant_mismatch".
- PMS / RMS / StatGovernance: strict JWT middleware rejects tokens missing
  tenant_id; getContextTenantID never reads the X-Tenant-ID header.
- Core engine (backend): getUserContext derives tenant from the verified
  bearer JWT. The X-Tenant-ID header is only accepted behind the internal
  service key as an authenticated intermediary, never from clients.
- StatChat: tenant scoping always originates from the verified token.

## Required test matrix

    Tenant A -> Tenant A data   PASS
    Tenant A -> Tenant B data   DENY
    Tenant B -> Tenant A data   DENY

These cases are automated in the security regression suite.
---
## Phase XII — Institutional Intelligence tenant isolation

Every Phase XII table (institutional_objects/institutional_graph_edges/institutional_conditions/intelligence_signals/institutional_objectives/kpis/kpi_measurements/isk_events/i_recommendations/i_audit_log) is tenant-scoped with 	enant_id mandatory and never derived from client headers.

- Graph traversal is scoped by tenant_id at every hop; cross-tenant traversal returns 0 results (DENIED).
- Graph edges are uniqueness-constrained per tenant; UNIQUE (tenant_id, canonical_id) prevents cross-tenant collisions.
- AI recommendation lifecycle review/execute is tenant-scoped; cross-tenant review is denied.

These cases are covered by the Phase XII test suite (e.g. TestPhase12_Graph_TenantIsolation, TestPhase12_AI_TenantScopedLifecycle).
