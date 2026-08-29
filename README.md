# StatGate — Enterprise Evidence Intelligence Platform

StatGate is a unified, multi-tenant enterprise platform for collecting, integrating, governing, analysing, visualizing, and communicating around trusted data. This repository is the StatGate workspace: it grew from the original **StatGate Analytical Hub** into a single source tree that also contains the Field Operations Registry, Operations Helpdesk, StatChat collaboration, Projects/Research management, data collection, an enterprise integration layer, and a full observability/analyst stack.

## What it provides

- **Analytics workspace** — dataset catalog and schema explorer, notebooks, semantic indicator registry, ABAC policy matrix, executive command centre, schema-health and asset healing, rule-based agentic reports, and NLQ/semantic query (Flask UI + Go core).
- **Unified identity** — every module validates the same Registry-signed JWT (`STATGATE_REGISTRY_JWT_SECRET`); users move across modules without separate login.
- **Field Operations Registry** — staff, facilities, organizations, and org units with a Go API + React UI.
- **Operations Helpdesk** — service requests, case triage, knowledge base, and admin oversight (Node/Express/Sequelize + React).
- **Projects Management System (PMS)** and **Research Management System (RMS)** — full-lifecycle project/programme and research management with workflow automation (Go APIs + React/Vite UIs).
- **StatChat** — real-time collaboration backbone: direct messages, channels, group chat over WebSockets, tasks, meetings, file/voice sharing, presence, notifications (Go + React/Vite).
- **Data collection** — StatCollect, an ODK-style multipart submission adapter (Go), plus the Android `collect-master` app for field capture.
- **Enterprise integration layer** — enterprise event bus, notifications, timelines, dashboard/widget engine, universal file service, permissions, calendar, reporting, approval and workflow engine, AI preparation, and cross-application search (`enterprise/core`, `enterprise/search`, `enterprise/design-system`).
- **Telemetry and monitoring** — Prometheus, Grafana, Alertmanager, Loki/Promtail, and optional analyst tooling: JupyterHub, Superset, Airflow, MLflow, and MinIO object storage.
- **Shared infrastructure** — one PostgreSQL 15 instance with per-module databases, Redis 7 cache/event bus, and cross-module object linkage.

## Architecture

```text
Browser / Android collector (collect-master)
  -> App Launcher (Next.js :3006) — single entry portal to all modules
       |-> Analytics UI (Flask :5000) -> Go Core (:8082) -> PostgreSQL + in-memory lakehouse
       |-> Registry UI (:3007)        -> Registry API (Go :9090)
       |-> Helpdesk UI (:3005)        -> Helpdesk API (Node :5006)
       |-> StatChat UI (:3009) <----> StatChat backend (Go :4000, WebSockets)
       |-> PMS UI (:3010)             -> PMS API (Go :8091)
       |-> RMS UI (:3011)             -> RMS API (Go :8092)
       |-> StatGovernance UI (:3012)  -> StatGovernance API (Go :8093)
       |-> Enterprise Search (:8095)  / Enterprise Core (:8096)
       `-> StatCollect (:8080) -> Redis event bus -> StatChat discussions / Registry identity
```

All services share the `statgate-network`, PostgreSQL (databases created by `docker/postgres-init`), Redis, and Registry-issued JWT identity.

## Repository layout

| Path | Purpose |
| --- | --- |
| `backend/` | Go core engine: telemetry lakehouse, 3-sigma anomaly detection, ABAC, semantic registry, PostgreSQL-backed asset manager, Prometheus metrics. |
| `frontend/` | Flask analytical workspace: `app.py` (routes/auth/proxies), `engine.py` (Pandas/DuckDB), `agentic_engine.py` (reports), `schema_healer.py`, `kaggle_connector.py`. |
| `appluancher/` | Next.js 14 central application launcher and home portal. |
| `stage_register/` | Field Operations Registry: `go-backend` (:9090) + React `frontend` (:3007). |
| `helpdesk-master/` | Operations Helpdesk: Node/Express `backend` (:5006) + React `frontend` (:3005). |
| `StatChat/` | StatChat collaboration platform: Go `backend` (:4000) + React/Vite `frontend` (:3009). |
| `PMS/`, `RMS/` | Projects and Research management: Go backends + React/Vite UIs. |
| `StatGovernance/` | Phase VIII Institutional Governance, Compliance, Risk & Internal Control Platform: Go backend (:8093) + React/Vite frontend (:3012). |
| `StatCollect/` | Data-collection adapter for ODK multipart submissions (Go, :8080). |
| `StatSpatial/` | Early-stage spatial/GIS Go backend. |
| `collect-master/` | Android field-capture app (ODK Collect fork, Gradle). |
| `enterprise/` | `core` event/workflow/AI engine (:8096), `search` cross-app search (:8095), `design-system` shared UI library. |
| `monitoring/` | Prometheus, Grafana, Alertmanager, Loki/Promtail, Trino config, Airflow DAGs, JupyterHub config. |
| `docker/` | `postgres-init` scripts creating all platform databases and shared tables. |
| `jupyterhub/` | JupyterHub Dockerfile + config. |
| `data/` | Locally persisted assets: dashboards, agent feedback, schema snapshots. |
| `secrets/`, `uploads/` | Docker secret files and upload storage. |
| `docker-compose.yml`, `.env.example` | One compose file for the full platform; environment template as single source of truth. |
| `start-analytics.ps1`, `start-registry.bat` | Launchers for the analytics platform and the Registry API. |

## Services and ports

| Service | Source | Host access |
| --- | --- | --- |
| PostgreSQL 15 / Redis 7 | `docker/` | `localhost:5432` / `localhost:6379` |
| App Launcher | `appluancher/` | <http://localhost:3006> |
| Analytics UI (Flask) | `frontend/` | <http://localhost:5000> |
| Analytics Core (Go) | `backend/` | <http://localhost:8082> (`/health`) |
| Registry API / UI | `stage_register/` | <http://localhost:9090> / <http://localhost:3007> |
| Helpdesk API / UI | `helpdesk-master/` | <http://localhost:5006> / <http://localhost:3005> |
| StatChat backend / UI | `StatChat/` | <http://localhost:4000> / <http://localhost:3009> |
| PMS API / UI | `PMS/` | <http://localhost:8091> / <http://localhost:3010> |
| RMS API / UI | `RMS/` | <http://localhost:8092> / <http://localhost:3011> |
| Enterprise search / core | `enterprise/` | <http://localhost:8095> / <http://localhost:8096> |
| Prometheus / Grafana / Alertmanager | `monitoring/` | <http://localhost:9095> / <http://localhost:3003> / <http://localhost:9093> |
| MinIO (API / console) | — | `localhost:9000` / <http://localhost:9001> |
| JupyterHub / Superset / Airflow / MLflow | — | <http://localhost:8000> / <http://localhost:8088> / <http://localhost:8085> / <http://localhost:5002> |

The root `docker-compose.yml` starts the full platform. `StatCollect` ships its own compose file that joins the shared `statgate-network`.

## Integration model

- **Unified identity**: `STATGATE_REGISTRY_JWT_SECRET` is shared by every backend. The Registry API signs tokens and each module validates them (Flask registers and enforces the identity, StatChat, StatCollect, and Helpdesk verify the same token).
- **Event bus**: services publish cross-module events to the Redis channel `statgate:events` (for example StatCollect `submission.received` / `submission.validated` / `submission.rejected`).
- **Object linkage**: a shared `object_links` table connects submissions, tasks, documents, and messages across modules.
- **Collaboration backbone**: StatChat conversations are auto-created for platform objects (`obj:submission:{instance_id}`, ...) so every module gets threaded, searchable discussion and notifications.

## Quick start (full platform, Docker)

```powershell
# 1. Configuration — never commit .env
Copy-Item .env.example .env
# 2. Set at least: STATGATE_INTERNAL_API_KEY, FLASK_SECRET_KEY, STATGATE_REGISTRY_JWT_SECRET
# 3. Start everything (or run .\start-analytics.ps1 which validates .env and checks health)
docker compose up --build
```

PostgreSQL is initialized from `docker/postgres-init` (databases: `statgate_ml_staging`, `statgate`, `kaggle`, `statchat`, `pms`, `rms`). After boot, open the App Launcher at <http://localhost:3006> and use the platform's service directory to reach each module.

## Local development (analytics only)

Prerequisites: Go 1.25+, Python 3.12+ with `venv`/`pip`, PostgreSQL with the configured `ml_staging` schema.

1. Create your configuration from `.env.example`; keep `.env` private.

2. Start the Go core:

   ```powershell
   cd backend
   go run ./cmd/server
   ```

3. In a second terminal, create the frontend environment and start Flask:

   ```powershell
   cd frontend
   python -m venv .venv
   .\.venv\Scripts\Activate.ps1
   python -m pip install -r requirements.txt
   python app.py
   ```

4. Open <http://localhost:5000>. Check the Go core at <http://localhost:8080/health> when running it directly, or <http://localhost:8082/health> when running through the root Docker Compose stack.

> The checked-in `frontend/Lib` and `frontend/Scripts` directories are not a portable Python environment. Create a fresh virtual environment for each machine or deployment.

## Analytics main routes

| Route | Description |
| --- | --- |
| `/` | Analytical dashboard |
| `/datasets` | Dataset catalog and schema explorer |
| `/notebook` | Interactive notebook UI |
| `/executive` | Executive command centre |
| `/semantic` | Semantic indicator registry |
| `/abac` | ABAC policy matrix |
| `/health`, `/ready`, `/metrics/prometheus` | Health, readiness, and Prometheus metrics |
| `/api/*` | Dataset, proxy, dashboard, agent, schema-health, project/report/research, and registry APIs |

## Configuration and security

Use `.env.example` as the reference. For production:

- Set `STATGATE_ENV=production` and strong values for `STATGATE_INTERNAL_API_KEY`, `FLASK_SECRET_KEY`, and `STATGATE_REGISTRY_JWT_SECRET`. The Go core refuses to start in production without the internal API key.
- Run Go privately (loopback or private network); it must not be Internet-facing.
- Set `CORS_ALLOWED_ORIGIN` to one exact browser origin.
- Put the web UIs behind TLS and an authenticated reverse proxy. `ANALYTICS_REQUIRE_AUTH=true` enforces login on the analytics workspace.
- Leave `ENABLE_NOTEBOOK_EXECUTION=false` unless users are fully trusted; enabling it permits submitted Python to execute in the Flask process.
- Use durable storage for `data/`, or migrate local JSON persistence into a managed database before scaling horizontally.

Tenant, role, and clearance headers are application input today, not a complete user-authentication system. The Registry JWT identity layer is the mechanism for production authentication.

## Verification

```powershell
cd backend
go test ./...
```

Other modules carry their own test suites (for example `StatChat`, `StatCollect/tests`, and `enterprise/core`). The GitHub Action `.github/workflows/integration.yml` builds the StatCollect stack and runs its integration suite. Verify a running service with:

```powershell
curl http://localhost:8082/health
```

## Documentation

- [DEPLOYMENT.md](DEPLOYMENT.md) — service topology, production configuration, and persistence limitations.
- [STATGATE_ENGINEERING_DIRECTIVE.md](STATGATE_ENGINEERING_DIRECTIVE.md) — the platform vision and engineering expectations.
- [STATGATE_ENGINEERING_REVIEW.md](STATGATE_ENGINEERING_REVIEW.md) and [STATGATE_PHASE_VII_IMPLEMENTATION_REVIEW.md](STATGATE_PHASE_VII_IMPLEMENTATION_REVIEW.md) — implementation reviews.
- [STATGATE_PHASE_VII_OPERATIONALIZATION.md](STATGATE_PHASE_VII_OPERATIONALIZATION.md) — operationalization notes.
- [SECURITY.md](SECURITY.md) — security policy.

## Optional analyst tooling

The Compose file includes experimental analyst services — JupyterHub, Superset, Airflow, MLflow, and MinIO object storage — for notebooks, dashboards, orchestration, model tracking, and artifacts, plus Trino configuration under `monitoring/` for federated SQL. These services need additional initialization to be fully functional (for example Superset DB migrations, Airflow DB init, JupyterHub authenticator setup, and MLflow artifact bucket creation).
