# StatGate App 4 — Platform Engineering / RunOps

RunOps implements the control-plane foundation for P20, P25, P34 and P49.

`backend/` is the Go service. It keeps durable RunOps state in the `runops` PostgreSQL schema, emits infrastructure and alert domain events on `statgate:events`, and protects all API operations with `statgate-lib/auth`, tenant isolation, the shared permissions matrix, and cluster-scope ABAC checks.

## Runtime flow

1. The telemetry collector discovers the configured services (the default targets are Docker DNS services on `statgate-network`) and polls `/health`, `/ready`, and `/metrics`.
2. Probe records and Prometheus samples are persisted; failed probes upsert an alert and publish `runops.alert.raised` to Redis.
3. Agents may submit logs, metrics and traces to the authenticated ingestion API.
4. CI/CD, multi-cloud resource, and cluster provisioning requests are durable records and emit RunOps events for asynchronous executors.

`RUNOPS_SERVICE_TARGETS` optionally replaces the built-in target set with a JSON array of `{id,name,base_url,criticality,enabled}` objects.

## API surface

- `GET /health`, `/ready`, `/metrics` — service probes and Prometheus exporter.
- `GET /api/v1/summary` — IOC overview.
- `POST /api/v1/cicd/deployments` — queues a deployment.
- `POST /api/v1/cloud/resources`, `POST /api/v1/cloud/clusters` — multi-cloud inventory and provisioning requests.
- `GET /api/v1/ioc/targets`, `POST /api/v1/ioc/probes`, `GET /api/v1/ioc/alerts`, `POST /api/v1/ioc/alerts/:id/acknowledge` — operations centre.
- `POST /api/v1/telemetry/{logs,metrics,traces}` — telemetry ingestion.

All `/api/v1` routes require StatGate JWT authentication. Destructive or execution actions require the shared RBAC permissions plus the caller's cluster scope where applicable.
