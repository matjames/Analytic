# StatGate Incident Response (SG-SEC-2026-08)

## Triage

1. Confirm scope (service, endpoint, tenant, timestamps) using correlation-id
   in structured logs.
2. Severity P0 = suspected credential exposure / auth bypass / cross-tenant
   access. P1 = configuration or middleware gaps. P2 = hardening backlog.

## Credential leak procedure

1. IMMEDIATELY rotate the affected secret(s) (see SECRET_MANAGEMENT.md).
2. JWT secret rotation invalidates every token; plan a maintenance window and
   restart all verifying services with the new secret.
3. Review logs for usage of the exposed value around the exposure window.
4. Re-run scripts/secret-scan.ps1 to confirm the value is gone.
5. If the value existed in git history, run an approved history rewrite.
6. Update the incident record and the completion report.

## Authentication anomaly procedure

1. Validate JWTs are HS256, issued by statgate-registry, audience statgate,
   and non-expired.
2. Confirm no "demo_" tokens or auth bypass flags are present.
3. Audit for X-Tenant-ID mismatches in the audit log.
4. If a service trusts identity headers, upgrade to JWT enforcement.

## Communication

- Coordinate with the platform operator before JWT/DB rotation.
- Use maintenance windows for coordinated rotations.