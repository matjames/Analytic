# StatCollect - StatGate Data Collection Adapter

StatCollect is the **data collection adapter** for the StatGate Enterprise Evidence Intelligence Platform. It accepts ODK-style multipart submissions and integrates them into the StatGate ecosystem through unified identity, event-driven communication, and the StatChat communication backbone.

## Quick Start

```bash
# from workspace root
cd StatCollect
go run .
```

Or use the startup script (sets all StatGate integration env vars):

```bash
run.bat
```

## StatGate Platform Integration

StatCollect is fully integrated into the StatGate ecosystem:

### 1. Unified Identity (Registry JWT)
- Validates JWT tokens signed with `STATGATE_REGISTRY_JWT_SECRET`
- Captures submitting user identity (`submitted_by`)
- Admin operations can be authorized via Registry JWT roles

### 2. Event Bus (Redis pub/sub)
- Publishes tenant-scoped events through the shared `statgate-lib/events` envelope on `statgate:events`
- Event types: `submission.received`, `submission.validated`, `submission.rejected`, `submission.linked`
- Shared schema includes `event_id`, `event_type`, `source`, `object_type`, `object_id`, `tenant_id`, `payload`, `timestamp`, and `version`

### 3. StatChat Communication Backbone
- Every submission can get a canonical `obj:statcollect:submission:<instance_id>` StatChat discussion conversation
- Validation/rejection notifications are posted to the discussion
- Service requests use the shared internal key, service identity, tenant, and optional workspace context; messages target the conversation ID returned by StatChat

### 4. Object Linkage
- Submissions can be linked to any StatGate platform object
- Shared `object_links` table for cross-module connectivity

### 5. Validation Workflow
- Submissions move through status: `received` → `approved`/`rejected`
- Validation records persisted in `submission_validations`
- Events published + StatChat notified on each transition

## Endpoints

### Core
- `GET /health` - Health check
- `GET /health/full` - Full health check (DB + S3 + integrations)
- `POST /submission` - Accept ODK multipart submissions (API key or Registry JWT)
- `GET /metrics` - Prometheus metrics

### Admin API
- `GET /admin/submissions` - List submissions (paginated)
- `GET /admin/submission?instance_id=...` - Get submission detail + XML
- `POST /admin/keys` - Rotate API keys
- `POST /admin/submission/validate?instance_id=...&status=approved|rejected` - Validation workflow
- `GET /admin/events?object_type=...&object_id=...` - Event audit log
- `GET /admin/objects/links?object_type=...&object_id=...` - Object connections
- `GET /admin/submission/discussion?instance_id=...&create=true` - StatChat discussion link

## Environment Variables

See `.env.example` for all configuration options.

### StatGate Integration
| Variable | Description | Default |
|----------|-------------|---------|
| `STATGATE_REGISTRY_JWT_SECRET` | Unified identity JWT secret | `statgate_field_secret_key_2026` |
| `STATGATE_INTERNAL_API_KEY` | Service-to-service key | `Kb7Qx3pV9mL2rT8wY4nC6dF1hJ5sA0eR` |
| `STATGATE_TENANT_ID` | Tenant for multi-tenant isolation | `default` |
| `STATGATE_WORKSPACE_ID` | Optional workspace context for StatChat object discussions | empty |
| `STATCOLLECT_ENABLE_EVENTS` | Enable Redis event bus | `false` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `STATCOLLECT_ENABLE_STATCHAT` | Enable StatChat integration | `false` |
| `STATCHAT_API_URL` | StatChat backend URL | `http://localhost:4000` |
| `STATGATE_REGISTRY_API_URL` | Registry API URL | `http://localhost:9090/api` |

## Database Schema

- `submissions` - Submission records with tenant, status, submitted_by
- `attachments` - Uploaded file metadata
- `object_links` - Cross-module object connections
- `event_log` - Audit trail of cross-module events
- `statchat_links` - StatChat discussion conversation links
- `submission_validations` - Validation workflow records

## Docker Compose

```bash
cd StatCollect
docker-compose up -d
```

The compose file joins the shared `statgate-network` and connects to the platform's Redis, StatChat, and Registry services.

## Next Steps
- Add form definition management
- Map parsed XML to StatGate DB model
- Add object store for attachments (S3/MinIO)
- Add idempotency handling and tests
