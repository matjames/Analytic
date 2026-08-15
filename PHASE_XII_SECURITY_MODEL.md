# PHASE_XII_SECURITY_MODEL.md
# StatGate Phase XII — Security Model

Base: security baseline `SG-SEC-2026-08` and Phases I–XI hardening. Phase XII
**adds** to the platform without weakening it. It introduces none of:
anonymous fallback, demo authentication, default credentials, default JWT
secrets, header-based identity trust, tenant fallback, admin fallback, or
unauthenticated mutation.

## Identity & authorization
- All Phase XII routes (`/api/intelligence`, `/api/graph`, `/api/objectives`,
  `/api/kpis`, `/api/risks`, `/api/ai`) are registered **inside** the existing
  JWT-authentication + tenant-isolation middleware group.
- Identity is derived exclusively from the verified JWT context
  (`getContextUserID` / `getContextTenantID` / `getContextRole`). Client-supplied
  `X-User-ID` / `X-Tenant-ID` are never trusted.
- Privileged actions (graph mutation, KPI/objective/risk/AI authorization and
  execution, signal ingest) require `admin` or `institutional_lead`.

## Tenant isolation
Every Phase XII table is tenant-scoped with `tenant_id` mandatory. Graph
traversal is scoped per hop; cross-tenant traversal returns 0 results. AI
recommendation review/execute is tenant-scoped; cross-tenant review is denied.

## Fail-closed behaviour
- Missing tenant context → `403 tenant_context_required`.
- Unapproved/arbitrary graph query → rejected.
- Disallowed AI data classification (RESTRICTED/SENSITIVE) → rejected.
- External AI provider configured without a key → disabled, not anonymous.
- Invalid/expired/wrong-issuer/audience/algorithm JWT and missing secret →
  denied by the existing middleware (covered by the Phase X security tests).

## Graph provenance & evidence
Edges carry `provenance_type` (AUTHORITATIVE | DERIVED | INFERRED | AI_GENERATED)
so AI-derived relationships are never presented as authoritative. Deletion is
append/correction oriented (time-boxed `valid_until`) — historical evidence is
never destroyed.

## Audit
Graph mutations produce `graph.*` audit events; AI produces a dedicated
`ai_audit_log` linked to the platform audit system. No sensitive prompt content
is stored.
