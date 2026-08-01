# StatGate Analytical Hub — System Summary

## Overview

StatGate is a local, multi-tenant analytical-orchestration prototype. It combines a Go telemetry/orchestration API with a Flask application that provides data exploration, dashboards, notebooks, executive decision support, and local analytical-asset persistence.

## Services

| Service | Address | Responsibility |
| --- | --- | --- |
| Go core | `http://localhost:8080` | In-memory telemetry ingestion, tenant-scoped queries, statistical anomaly detection, ABAC, semantic indicators, webhook dispatch |
| Flask application | `http://localhost:5000` | User interface, Kaggle/PostgreSQL access, profiling, notebooks, dashboard metadata, agentic reports, schema-health workflows |
| PostgreSQL | configured by `.env` | Source datasets in the `ml_staging` schema |

## Architecture

```
Browser
  └─ Flask UI (:5000)
       ├─ PostgreSQL / Kaggle staging datasets
       ├─ Pandas + DuckDB analytical engine
       ├─ JSON persistence under data/
       └─ Go core proxy calls (:8080)
            ├─ ABAC policy engine
            ├─ semantic indicator registry
            ├─ in-memory telemetry lakehouse
            ├─ 3-sigma anomaly / alert workers
            └─ ABAC-gated webhook dispatcher
```

## Implemented capabilities

- Telemetry ingestion and query APIs, partitioned by tenant in the Go in-memory store.
- Statistical anomaly monitoring using z-scores; the proactive alert feed seeds a demo alert when empty.
- ABAC evaluation based on supplied tenant, role, and clearance request headers.
- Tenant-aware semantic indicator registry with seeded latency, throughput, and SDG-health metrics.
- Dataset catalog, schema profiling, previews, and DuckDB-based functional metric evaluation.
- Persistent in-process notebook sessions with Pandas and StatGate helper objects.
- Dashboard/widget metadata and agent feedback persisted as local JSON files.
- Executive dashboard with rule-based intervention scenarios and federated aggregate summaries.
- Schema snapshots, drift detection, synonym-based column renaming, and JSON-asset healing.
- Webhook actions run in safe mode (logging only) unless matching endpoint environment variables are configured.
- Internal Go calls have bounded timeouts; production requires a shared internal API key.
- The dataset catalog reads PostgreSQL directly from Flask, avoiding a circular Flask-to-Go-to-Flask request path.

## Important implementation characteristics

- Go telemetry, indicators, policies, alerts, and agent actions are process-local and reset when the Go service restarts.
- The Python notebook endpoint executes submitted code with `exec`, but it is disabled unless `ENABLE_NOTEBOOK_EXECUTION=true` is explicitly set.
- User authentication is not implemented: `X-Tenant-ID`, `X-User-Role`, and `X-User-Clearance` remain trusted input headers. An authenticated reverse proxy or application identity layer is required in production.
- Dashboard and feedback persistence is filesystem-based; mount `data/` durably or migrate it before horizontal scaling.
- Go's database-backed asset manager is not initialised by the current bootstrap. Its save endpoint now reports `503` rather than claiming a successful save.
- Go defaults to loopback binding and a single explicit CORS origin. Set `BIND_ADDRESS` and `CORS_ALLOWED_ORIGIN` deliberately for a deployed topology.

## Current operational status

- **Go core:** running and verified on port 8080. `GET /health` returned HTTP 200 with `status: healthy` on 2026-07-31 after the hardening rebuild.
- **Flask UI:** not started in this environment. The detected `C:\Python314\python.exe` lacks the standard-library `encodings` module, so it cannot initialise. Install or select a valid Python runtime, install `frontend/requirements.txt`, then start `frontend/app.py` from the `frontend` directory.
- **Go validation:** `go test ./...` succeeds; packages currently contain no automated test files.

## Key source entry points

| Area | File |
| --- | --- |
| Go server and HTTP handlers | `backend/cmd/server/main.go` |
| Lakehouse and anomaly detection | `backend/internal/lakehouse/` |
| ABAC and semantic registry | `backend/internal/abac/abac.go`, `backend/internal/semantic/semantic.go` |
| Flask routes | `frontend/app.py` |
| Data analysis engine | `frontend/engine.py` |
| Agentic and schema-health workflows | `frontend/agentic_engine.py`, `frontend/schema_healer.py` |
| Database connector | `frontend/kaggle_connector.py` |

## Local startup

```powershell
# Go core
cd backend
go run ./cmd/server

# Flask UI — after repairing/selecting Python
cd frontend
python -m pip install -r requirements.txt
python app.py
```
