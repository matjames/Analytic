# StatGate JWT Standard (SG-SEC-2026-08)

One authoritative JWT contract shared by every StatGate service:
Registry, Enterprise, PMS, RMS, StatGovernance, StatChat, Analytics,
StatCollect and other authenticated services.

## Canonical claims

Every StatGate JWT MUST conform to:

    {
      "sub": "user-id",
      "tenant_id": "tenant-id",
      "org_id": "organization-id",
      "role": "role",
      "email": "user@example.org",
      "iat": 0,
      "nbf": 0,
      "exp": 0,
      "iss": "statgate-registry",
      "aud": "statgate"
    }

| Claim      | Required | Description                                           |
|------------|----------|-------------------------------------------------------|
| sub        | yes      | Unique user identifier (subject)                       |
| tenant_id  | yes      | Tenant boundary; derived from the verified JWT only    |
| org_id     | conditional | Organization id when organization scoping applies   |
| role       | yes      | One of the canonical role whitelist (below)            |
| email      | no       | User email address                                     |
| iat        | yes      | Issued-at (epoch seconds)                              |
| nbf        | yes      | Not-before (epoch seconds)                             |
| exp        | yes      | Expiration (epoch seconds)                             |
| iss        | yes      | MUST be "statgate-registry"                            |
| aud        | yes      | MUST be "statgate"                                     |

## Canonical role whitelist

Configurable deployment-level issuer/audience overrides exist via
STATGATE_JWT_ISSUER (default "statgate-registry") and STATGATE_JWT_AUDIENCE
(default "statgate"), but they MUST be identical across all services.

Accepted roles (case-insensitive):

    viewer, analyst, editor, operator, manager, agent,
    district, district_admin, tenant_admin,
    governance_officer, property_officer, admin, superadmin, platform_admin

## Verification rules (enforced by every service)

1. alg header MUST be HS256 (reject "none", RS*, etc.)
2. signing secret STATGATE_REGISTRY_JWT_SECRET MUST be non-empty; if empty
   the caller fails closed (HTTP 503 or startup abort).
3. iss MUST equal the configured issuer.
4. aud MUST equal the configured audience.
5. exp MUST be present and >= now.
6. nbf, when present, MUST be <= now.
7. sub / user_id MUST be present and non-empty.
8. tenant_id MUST be present and non-empty.
9. role MUST be in the canonical whitelist.
10. Signature MUST verify; malformed / unsigned / wrong-issuer /
    wrong-audience / expired tokens are rejected with HTTP 401.

## Legacy compatibility

The Registry still emits legacy aliases (userId, tenantId, districtId) for
the React front-ends, but they are never used as the enforcement source.

## Service status

| Service      | HS256 | Issuer/Aud | Role whitelist | tenant from JWT | Fail-closed |
|--------------|-------|------------|----------------|-----------------|-------------|
| Registry     | yes   | yes        | yes            | yes             | yes (prod)  |
| Enterprise   | yes   | yes        | yes            | yes             | yes         |
| PMS          | yes   | yes        | yes            | yes             | yes         |
| RMS          | yes   | yes        | yes            | yes             | yes         |
| Governance   | yes   | yes        | yes            | yes             | yes (prod)  |
| Core (Go)    | yes   | yes        | via ABAC       | yes             | yes         |
| StatChat     | yes   | yes        | via shared JWT | yes             | yes         |
| Helpdesk     | yes   | n/a        | registration   | n/a             | yes (prod)  |
---
## Phase XII — Privileged role note

Phase XII introduces the institutional_lead role for privileged knowledge-graph mutation and AI recommendation authorization. This role is validated from the verified JWT role claim via equireRole. No Phase XII endpoint trusts header-supplied identity; JWT mechanism and validation rules are unchanged by Phase XII.
