# StatGate Integration Fabric (EIP)

The platform-wide application boundary for Phase 16, Phase 18, and Phase 42.
It owns external API registration, connector configuration, webhook ingress,
schema metadata, marketplace/plugin submissions, and low-code app definitions.
Domain data and business APIs remain in their owning StatGate applications.

| Phase | Capability in this application |
| --- | --- |
| P16 | API control plane, webhooks, connectors, API analytics event log, Developer Portal, and ESB integration boundary |
| P18 | Plugin marketplace and publisher submissions |
| P42 | Component Registry and low-code app definitions with controlled deployment requests |

## Run locally

```powershell
cd enterprise/integration
$env:STATGATE_INTERNAL_API_KEY = "local-development-key"
go run .
```

The service listens on `8097` by default. In Docker it uses the shared
PostgreSQL database and the `integration` schema. Configure it with
`INTEGRATION_DB_*` (or the matching `ENTERPRISE_DB_*` fallback variables).

## API surface

- `GET /health`, `GET /ready`
- `GET|POST /api/v1/apis`, `PUT|DELETE /api/v1/apis/:id`
- `GET|POST /api/v1/connectors`, `PATCH /api/v1/connectors/:id/status`
- `GET|POST /api/v1/schemas`
- `GET|POST /api/v1/webhooks`, `DELETE /api/v1/webhooks/:id`
- `POST /hooks/:id` (public webhook ingress; HMAC SHA-256 signed)
- `GET|POST /api/v1/api-keys`, `POST /api/v1/api-keys/:id/revoke`
- `GET /api/v1/events`
- `GET|POST /api/v1/marketplace/plugins`
- `GET|POST /api/v1/components`
- `GET|POST /api/v1/apps`, `POST /api/v1/apps/:id/deploy`
- `GET /api/v1/systems` — live health/discovery for connected StatGate applications

## Existing StatGate applications

At startup, the Fabric registers Enterprise Core, PMS, RMS, Registry, StatChat,
Helpdesk, StatGovernance, and Analytics as managed systems. It checks their
health through the private platform network. Fabric-originated webhook and
deployment events are also forwarded to Enterprise Core's authenticated event
bus, preserving a shared audit/timeline trail.

Control-plane routes require either the shared `X-Internal-API-Key` or a
Registry-issued bearer token. Secrets are never returned after creation.
