# StatGate — Business Process Management (App 11)

Go backend for **BPM / Workflow Automation & Digital Operations (P48)**.

| Sub-system | Implementation |
|---|---|
| **BPM Service** | `process_definitions` (BPMN-ish nodes + transitions), `process_instances`, deterministic workflow engine (`/instances/{id}/advance` resolves transitions, completes at end node) |
| **Case Management** | `case_instances`, `case_items` + object linkage to platform assets |
| **Task Management** | `work_items` (assignee/status/priority), completion emits `task.completed` |
| **Process Mining** | `activity_logs` (who/what/when/where) + `/mining/summary` ops analytics (counts, avg cycle time) |
| **Automation Service** | `automation_rules` keyed on `trigger_event`, `/automation/evaluate` fires `automation.triggered` |

A **first-class StatGate microservice**: Registry-JWT identity
(`statgate-lib/auth`), Redis event bus (`statgate:events`), immutable audit,
dedicated schema (`bpm` in `bpm_hub`), `/health` `/ready` `/metrics`.

## Mandatory hooks

1. **Emits** `process.started`, `process.completed`, `task.created`,
   `task.completed`, `case.created`, `automation.triggered`,
   `object.link.created` to `statgate:events`.
2. **`object_links`** — cases link to platform objects; generic `/links`.
3. `statgate-lib/auth` (fail-closed, `X-Tenant-ID` isolation), `/health`
   `/ready` `/metrics`.

## API surface (all `/api/v1`, Registry JWT required)

- **Processes**: `GET|POST /processes`, `GET /processes/{id}`,
  `POST /processes/{id}/publish`, `POST /processes/{id}/instances`
- **Instances**: `GET /instances`, `GET /instances/{id}`,
  `POST /instances/{id}/advance|cancel`, `GET /instances/{id}/tasks|log`
- **Cases**: `GET|POST /cases`, `GET|PUT|DELETE /cases/{id}`,
  `GET|POST /cases/{id}/items`, `POST /cases/{id}/link`
- **Tasks**: `GET /tasks?assignee=&status=`, `GET|PUT /tasks/{id}`
- **Automation**: `GET /automation`, `POST /automation/rules`,
  `POST /automation/rules/{id}/toggle`, `POST /automation/evaluate`
- **Mining & links**: `GET /mining/summary`, `POST /links`, `GET /links`,
  `GET /summary`

## Run

```powershell
cd backend
go run ./cmd/server      # :8104 (BPM_PORT)
curl http://localhost:8104/health
```

DB provisioned by `docker/postgres-init/20-create-bpm-db.sql`; registered in
`docker-compose.yml` as `bpm-hub` (host `8104`, `statgate-network`).

## Verify

```powershell
go build ./...
go test  ./...
go vet   ./...
```