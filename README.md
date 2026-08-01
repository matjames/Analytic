# StatGate Analytical Hub

StatGate is a local multi-tenant analytical-orchestration prototype. It combines a Go telemetry service with a Flask analytical workspace for dataset exploration, dashboards, notebooks, semantic metrics, executive reporting, and schema-health workflows.

## What it provides

- Tenant-scoped telemetry ingestion and querying.
- In-memory 3-sigma anomaly detection and an alert feed.
- Attribute-based access-control (ABAC) evaluation for telemetry operations.
- A semantic indicator registry for reusable business metrics.
- PostgreSQL-backed dataset catalog, schema profiling, previews, and DuckDB analysis.
- Dashboard metadata, agent feedback, and schema snapshots persisted under `data/`.
- Rule-based agentic reports and an executive command-centre view.
- Schema-drift detection with synonym-based column mapping.

## Architecture

```text
Browser
  -> Flask UI and analytical API (:5000)
       -> PostgreSQL / Kaggle staging schema
       -> Pandas and DuckDB
       -> local data/ persistence
       -> Go core (:8080)
            -> ABAC and semantic registry
            -> in-memory telemetry lakehouse
            -> anomaly and webhook workers
```

## Repository layout

| Path | Purpose |
| --- | --- |
| `backend/cmd/server/main.go` | Go HTTP server and orchestration endpoints |
| `backend/internal/` | Lakehouse, anomaly, ABAC, semantic, and asset packages |
| `frontend/app.py` | Flask routes and Go-service proxy layer |
| `frontend/engine.py` | Pandas/DuckDB analysis and local dashboard persistence |
| `frontend/agentic_engine.py` | Rule-based decision-support reports |
| `frontend/schema_healer.py` | Schema snapshots, drift detection, and asset healing |
| `frontend/templates/`, `frontend/static/` | Browser UI |
| `data/` | Local dashboards, feedback, and schema snapshots |

## Prerequisites

- Go 1.22+
- A complete Python 3.13+ installation with `venv` and `pip`
- PostgreSQL access to the configured `ml_staging` schema

> The checked-in `frontend/Lib` and `frontend/Scripts` directories are not a portable Python environment. Create a fresh virtual environment for each machine or deployment.

## Local setup

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

4. Open `http://localhost:5000`. Check the Go core at `http://localhost:8080/health`.

### Container startup

With Docker Desktop running and a populated `.env`, start the Analytics services with:

```powershell
docker compose up --build
```

This compose file starts the Analytics UI and core only, on `http://localhost:5001` and `http://localhost:8081` by default to avoid conflicting with existing local services. Override these with `ANALYTICS_UI_PORT` and `ANALYTICS_CORE_PORT` when needed. The Field Operations Registry remains a separately deployed StatGate application.

## Main routes

| Route | Description |
| --- | --- |
| `/` | Analytical dashboard |
| `/datasets` | Dataset catalog and schema explorer |
| `/notebook` | Interactive notebook UI |
| `/executive` | Executive command centre |
| `/semantic` | Semantic indicator registry |
| `/abac` | ABAC policy matrix |
| `/health` on port 8080 | Go service health check |

## Configuration and security

Use `.env.example` as the configuration reference. For production:

- Set `STATGATE_ENV=production` and a long random `STATGATE_INTERNAL_API_KEY`.
- Run Go privately; it defaults to `127.0.0.1` and should not be Internet-facing.
- Set `CORS_ALLOWED_ORIGIN` to one exact browser origin.
- Put Flask behind TLS and authenticated reverse-proxy or application authentication.
- Leave `ENABLE_NOTEBOOK_EXECUTION=false` unless users are fully trusted. Enabling it permits submitted Python to execute in the Flask process.
- Use durable storage for `data/`, or move local JSON persistence into a managed database before scaling horizontally.

Tenant, role, and clearance headers are currently application input, not a complete user-authentication system. A production identity layer must establish and validate them.

## Verification

Run the Go test suite:

```powershell
cd backend
go test ./...
```

Then verify the service:

```powershell
curl http://localhost:8080/health
```

## Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) for the service topology, configuration requirements, and current persistence limitations. [STATGATE_SYSTEM_SUMMARY.md](STATGATE_SYSTEM_SUMMARY.md) contains an implementation-oriented system inventory.

## Optional analytics services

This repository includes Compose service stubs for additional analyst tooling: JupyterHub, Superset, Metabase, ClickHouse, Trino, Airflow, Redpanda (Kafka-compatible), and MLflow. These are provided as local, experimental services to enable analysts to run notebooks, dashboards, federated SQL queries, orchestration, streaming, and model tracking.

Quick start (after creating required secrets and volumes):

```powershell
docker compose build --no-cache
docker compose up -d
```

Endpoints (defaults):
- JupyterHub: http://localhost:8000
- Superset: http://localhost:8088
- Metabase: http://localhost:3001
- ClickHouse HTTP: http://localhost:8123
- Trino: http://localhost:8082
- Airflow webserver: http://localhost:8085
- Redpanda (Kafka): 9092
- MLflow: http://localhost:5002

Note: these services need additional initialization to be fully functional (e.g., Superset DB migrations, Airflow DB init, JupyterHub authenticator setup, MLflow artifact bucket creation). I can scaffold init scripts and secure production-ready configs next if you'd like.
