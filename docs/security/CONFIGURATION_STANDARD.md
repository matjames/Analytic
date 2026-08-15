# StatGate Configuration Standard (SG-SEC-2026-08)

## One canonical source of truth

.env / .env.example (root) is the authoritative configuration contract.
docker-compose.yml and all services reference the same variable names.
Per-service .env files are dev-only conveniences and must not diverge.

## Canonical variable names

| Area              | Variable                            |
|-------------------|-------------------------------------|
| Platform mode     | STATGATE_ENV |
| Identity          | STATGATE_REGISTRY_JWT_SECRET, STATGATE_JWT_ISSUER, STATGATE_JWT_AUDIENCE |
| Internal API      | STATGATE_INTERNAL_API_KEY |
| Event bus         | STATGATE_EVENT_CHANNEL (default statgate:events) |
| Core DB           | KAGGLE_DB_* |
| Helpdesk DB       | HELPDESK_DB_* / POSTGRES_* |
| Registry DB       | REGISTRY_DB_* |
| StatChat DB       | STATCHAT_DB_*, DATABASE_URL |
| PMS / RMS / Gov   | PMS_DB_*, RMS_DB_*, GOVERNANCE_DB_* |
| StatCollect       | STATCOLLECT_DB_*, STATCOLLECT_API_KEY, STATCOLLECT_ADMIN_KEYS |
| Rate limit        | STATGATE_RATE_LIMIT_RPM |
| Body limit        | STATGATE_MAX_REQUEST_BODY_KB |
| Upload root       | ENTERPRISE_UPLOAD_DIR, STATCHAT_UPLOAD_DIR |

## Standard ports

Registry 9090, Core 8080, Analytics 5000, StatChat 4000, PMS 8091,
RMS 8092, Governance 8093, Enterprise search 8095, Enterprise core 8096,
Helpdesk 5006; front-ends 3006/3007/3009/3010/3011/3012.

Drift findings D-01..D-10 are resolved in this standard. See
SERVICE_SECURITY_MATRIX.md.
---
## Phase XII — Institutional Intelligence configuration

- STATGATE_AI_PROVIDER` ('' = deterministic-local default; 'openai', 'gemini', or a custom name).
- STATGATE_AI_MODEL`, STATGATE_AI_BASE_URL` (default local inference http://127.0.0.1:11434/v1).
- Provider keys: OPENAI_API_KEY`, GEMINI_API_KEY`, or STATGATE_AI_API_KEY` for custom providers.
- STATGATE_AI_PROVIDER set without a key FAILS CLOSED — AI reasoning is disabled rather than falling back anonymously.
- Privileged Phase XII role: institutional_lead (alongside dmin) for graph mutation and AI authorization.
- Graph traversal depth: DefaultMaxTraversalDepth = 5 (7 benchmark-only).
