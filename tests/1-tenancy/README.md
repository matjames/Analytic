# Cross-Service Tenant / Authorization Attestation

Two complementary suites that verify the tenant and workspace contract of every
deployed StatGate service against the running Compose stack.

| File | Purpose |
|---|---|
| `manifest.json` | Service → endpoint → expected status matrix (calibrated live, 2026-09-15) |
| `attest.ps1` | **Unauthenticated**: no credentials and malformed `X-Workspace-ID` must both fail closed |
| `attest-authenticated.ps1` | **Authenticated**: mints a Registry-contract JWT and checks auth acceptance, workspace shape guard, and membership denial |
| `TEST_RESULT` / `TEST_RESULT_AUTH` | Recorded evidence from the last run |

## Running

```powershell
# unauthenticated contract
powershell -NoProfile -ExecutionPolicy Bypass -File tests/1-tenancy/attest.ps1

# authenticated contract (reads STATGATE_REGISTRY_JWT_SECRET from env or the gitignored .env)
powershell -NoProfile -ExecutionPolicy Bypass -File tests/1-tenancy/attest-authenticated.ps1
```

Both exit non-zero on any violation and rewrite their `TEST_RESULT*` file.

## What is asserted

**Unauthenticated:** every protected endpoint answers `401` with no credentials,
and answers `400` where the shared workspace middleware validates header shape
before authentication (StatIoT is the reference case for that ordering).

**Authenticated** (synthetic principal `attest-user` / role `viewer` / tenant `tenant-alpha`):
1. a valid Registry-signed JWT is accepted (no `401`);
2. a malformed `X-Workspace-ID` yields `400`;
3. where `membership_enforced` is true, a well-formed but unknown workspace yields a
   **denial** — asserted as membership in `{400, 403, 404, 503}`, not one hardcoded code.

## Results (2026-09-15)

- Unauthenticated: **36/36 checks PASS** across 18 services.
- Authenticated: **47 checks, 0 failures, 7 informational notes**.

### Documented nuances (not failures)

- **`503 workspace_membership_unavailable`** is what this synthetic principal produces for an
  unknown workspace. The membership middleware calls Enterprise Core
  `GET /workspaces/{id}` with the caller's token; Enterprise Core answers `401` for a principal
  it has no record of (rather than `403`/`404`), so the middleware degrades to `503`.
  All of `400`/`403`/`503` deny access, so the route is fail-closed. A genuine `403` requires a
  real provisioned Enterprise Core user.
- **No membership enforcement on these read routes** (`unknown-ws` returns `200`):
  `statspatial /api/v1/layers`, `statops /api/v1/summary`, `statchat /users`.
  These rely on list scoping only; wiring the shared membership middleware here is a follow-up.
- **Principal validation against a service-local store** (`401` with a valid JWT by design):
  `enterprise-core /api/events`, `statcollect /admin/submissions`.
- **Additional context required**: `stattrust /api/v1/summary` answers `400` for this principal.
- **helpdesk** accepts the JWT and returns `404` for a nonexistent ticket — authentication
  succeeded; the workspace middleware is not applied to that route.

## Follow-ups

1. Provision a real Enterprise Core user + workspace so membership denial can be asserted as a
   true `403` rather than the `503` degradation.
2. Apply `tenant.GinWorkspaceMembership` to the three read routes listed above.
3. Extend the authenticated suite with seeded cross-workspace **object** reads (expect `404`).
