# API Guidelines

## Contract

- Use versioned paths such as `/api/<service>/v1` for public module APIs.
- Return JSON with stable field names, explicit errors, and appropriate HTTP status codes.
- Validate request bodies, query limits, identifiers, and enum values at the boundary.
- Use pagination for collections and return a deterministic ordering.
- Require Registry JWT authentication for protected routes; derive identity from claims rather than request fields.
- Enforce tenant and workspace access in handlers and storage queries.
- Make retries safe with idempotency keys for create or side-effecting operations where clients may retry.

## Errors and observability

Errors should be actionable to clients but must not disclose secrets, SQL, tokens, or internal stack traces. Include a correlation identifier in responses and logs. Expose `/health`, `/ready`, and `/metrics` according to the service standard.

Document route changes in the service contract or OpenAPI output and update the route audit when adding protected operations.
