# StatGate Security, Identity & Access Model

## Identity & Single Sign-On (SSO)
The Registry service is the authoritative identity provider.
- All services authenticate incoming requests by verifying HS256-signed JSON Web Tokens using `STATGATE_REGISTRY_JWT_SECRET`.
- Unsigned or fake tokens are rejected with HTTP 401.
- Fallback hardcoded credentials (`statgate_field_secret_key_2026`, etc.) have been completely eliminated from production Compose configurations.

## Role-Based & Tenant Isolation Model
- Roles: `admin`, `superadmin`, `manager`, `analyst`, `user`.
- Context claims (`user_id`, `tenant_id`, `role`) are injected into the handler context.
- `tenantIsolationMiddleware` verifies that any explicit `X-Tenant-ID` header matches the cryptographically verified JWT claim.

## Rate Limiting & Protection
- Token-bucket rate limiter (`STATGATE_RATE_LIMIT_RPM`, default 300 requests/minute per IP) protects against denial of service and API abuse.
- Request payload size protection (`STATGATE_MAX_REQUEST_BODY_KB`, default 1MB) rejects oversized payloads with HTTP 413.
- Standard security headers (`X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`, `Permissions-Policy`) are injected on every response.
