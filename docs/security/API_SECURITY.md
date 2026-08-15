# StatGate API Security (SG-SEC-2026-08)

## Error responses

Production APIs return structured errors, never raw DB errors, stack traces,
filesystem paths, SQL text or internal addresses.

    { "error": { "code": "RESOURCE_NOT_FOUND",
                 "message": "The requested resource was not found.",
                 "correlation_id": "..." } }

Detailed error detail belongs in server-side logs only.

## Rate limiting

Enterprise core applies a token bucket rate limiter per client IP
(STATGATE_RATE_LIMIT_RPM, default 300). Client IP is derived from the
transport connection; X-Forwarded-For is only trusted behind a trusted proxy.
Authentication-gated endpoints protect login, registration, ticket creation
and WebSocket connections.

## Request constraints

- Maximum request body size: STATGATE_MAX_REQUEST_BODY_KB (default 1024 KB).
- Uploads capped per-endpoint (helpdesk avatars 5 MB, video 500 MB with
  type checks).
- CORS restricted to configured origins (helpdesk) and localhost dev sets
  (Go services).

## Sensitive data

- Password hashes are never returned or logged (helpdesk toSafeUser).
- Tokens/secrets are never logged.
- Sensitive fields excluded from responses and model associations.