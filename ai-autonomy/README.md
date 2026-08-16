# StatGate — AI & Autonomy (App 5)

Go backend for the **AI & Autonomy** group of phases:

| Phase | Domain |
|---|---|
| **P22** | Advanced AI, Digital Twins & Decision Intelligence |
| **P31** | Autonomous AI Organs / Multi-Agent Systems |
| **P39** | Knowledge Graph & Semantic Intelligence |

A **first-class StatGate microservice**: Registry-JWT identity
(`statgate-lib/auth`), Redis event bus (`statgate:events`), immutable audit
(`enterprise_audit_log`), dedicated schema (`ai` in the `ai_intelligence`
database), and `/health` `/ready` `/metrics` probes.

## Architecture

```
Platform event bus (statgate:events)
   │  dataset.published, survey.published, submission.received,
   │  project.created, research.published, content.published, ...
   ▼
AI & Autonomy (:8101)
   ├─ P22 Digital Twin Simulation Engine   -> digital_twins, simulations
   ├─ P22 Decision Intelligence           -> ai_models, predictions, pipelines
   ├─ P31 Multi-Agent System              -> agents, agent_tasks, agent_memory,
   │                                        agent_messages, governance_actions
   ├─ P39 Knowledge Graph Engine          -> graph_nodes, graph_edges, graph_statements
   ├─ P39 Contextual Intelligence Indexer -> context_index
   └─ object_links  (cross-app entity registration)
   │  Events out: prediction.generated, simulation.completed,
   │              agent.task.created|completed, agent.action.taken,
   │              graph.entity.registered, object.link.created
   ▼
Enterprise Core (:8096) + other apps react
```

## Mandatory system hooks (implemented)

1. **Listens** to `statgate:events` — `StartEventConsumer()` maps domain events to
   agent tasks (analyst/auditor/researcher roles) and auto-registers digital
   twins for project events (`api/events.go`, verified by `TestEventTriggerMapping`).
2. **Emits** `prediction.generated`, `simulation.completed`,
   `agent.task.created|completed`, `agent.action.taken`,
   `graph.entity.registered`, `object.link.created`.
3. **Registers in `object_links`** — graph nodes referencing source platform
   objects and explicit `/api/v1/links` both persist the shared contract.
4. `statgate-lib/auth` JWT enforcement (fail-closed, `X-Tenant-ID` isolation),
   `/health` `/ready` `/metrics`.

## API surface (all under `/api/v1`, Registry JWT required)

- **Twins & simulations**: `GET|POST /twins`, `GET|PUT|DELETE /twins/{id}`,
  `POST /twins/{id}/simulate`, `GET /twins/{id}/simulations`
- **Models & predictions**: `GET|POST /models`, `GET|DELETE /models/{id}`,
  `POST /models/{id}/predict`, `GET /predictions`, `GET /predictions/{id}`
- **Pipelines**: `GET|POST /pipelines`
- **Agents**: `GET|POST /agents`, `GET|PUT /agents/{id}`,
  `POST|GET /agents/{id}/tasks`, `POST|GET /agents/{id}/memory`,
  `POST|GET /agents/messages` (Agent Communications Bus),
  `GET /agents/runtime`
- **Governance**: `GET /governance?verdict=`, `POST /governance/{id}/review`
- **Knowledge graph**: `GET|POST /graph/nodes`, `GET|DELETE /graph/nodes/{id}`,
  `GET /graph/neighbors/{id}`, `GET|POST /graph/edges`, `DELETE /graph/edges/{id}`,
  `GET|POST /graph/statements` (SPARQL-style pattern query)
- **Context indexer**: `POST /index/enrich`, `GET /index/search?q=`
- **Cross-app links**: `POST /links`, `GET /links?type=&id=`

## Integration contracts

| Contract | Mechanism |
|---|---|
| Events in → agent tasks | `statgate-lib/events` subscribe on `statgate:events` |
| Events out | `emitEvent` → same canonical channel, standard schema |
| Identity | `statgate-lib/auth` (Registry JWT, fail-closed, tenant isolation) |
| Audit | `statgate-lib/audit` → `enterprise_audit_log` |
| Entity linkage | shared `object_links` table |
| Observability | `/health` `/ready` `/metrics` (Prometheus via statgate-lib) |

## Run

```powershell
# 1. DB provisioned by docker/postgres-init/17-create-ai-db.sql (compose mount)
# 2. Configure .env (see backend/.env.example), then:
cd backend
go run ./cmd/server      # :8101 (AIENG_PORT)
curl http://localhost:8101/health
```

Registered in `docker-compose.yml` as `ai-autonomy` (host `8101`,
`statgate-network`).

## Verify

```powershell
cd backend
go build ./...
go test  ./...
go vet   ./...
```