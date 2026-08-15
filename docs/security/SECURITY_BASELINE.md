# StatGate Security Baseline (SG-SEC-2026-08)

## Authentication

- Every production API endpoint requires authentication. Health, readiness
  and metrics probes are the only unauthenticated paths.
- No demo tokens exist. Tokens with the "demo_" prefix are rejected with 401.
- No anonymous/demo identities (user-001) exist anywhere in the platform.
- StatChat authentication, analytics authentication and governance bypasses
  are permanently on / removed.

## Fail-closed behavior

- STATGATE_REGISTRY_JWT_SECRET empty -> all authenticated requests rejected
  (503) or startup aborted in production.
- STATGATE_INTERNAL_API_KEY empty -> core engine rejects protected requests.
- Postgres build scripts use \getenv: missing passwords fail at bootstrap.
- docker-compose uses ${VAR:?...} mandatory substitution for every secret.

## Identity headers

Client-supplied identity headers (X-User-ID, X-User-Role, X-User-Clearance,
X-Tenant-ID) are NEVER authoritative. They are only honoured behind the
internal service key, and the JWT always wins when present.

## Secrets

- No production secret is committed to the repository.
- .env / .env.example / per-service .env contain placeholders only.
- Monitoring (Grafana, MinIO, Airflow, Superset) credentials are injected via
  mandatory-flag environment variables.
- Trino and Loki read credentials from the runtime environment only.

## Authorization

- Canonical role whitelist enforced at every Go middleware and at the
  Helpdesk/Registry registration paths.
- Privileged roles are provisioned by an authenticated administrator.
- ABAC engine in the core engine enforces clearance checks on every mutation.

## Tenant isolation

Tenant is derived from the verified JWT tenant_id claim and enforced by
tenantIsolationMiddleware (Enterprise) and per-service middleware. See
TENANT_ISOLATION.md.

## Default passwords removed

- biostat@2026 (Registry) removed; CSV provisioning generates a policy
  compliant per-user temporary password with must_change_password=true.
- Helpdesk and JupyterHub DummyAuthenticator "password" removed.
- Monitoring default admin credentials removed.