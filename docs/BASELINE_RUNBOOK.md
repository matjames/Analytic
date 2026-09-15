# StatGate Baseline Runbook — Compose Deployment

**Status:** Stage 1 release baseline, certified 2026-08-29 (see `Roadmap/PHASE_SERVICE_MATRIX.md`).

## 1. Prerequisites

- Docker Desktop with Compose v2
- `.env` in the repository root (copy from the template, populate **every** fail-closed secret — Compose refuses to start otherwise)
- First `up` is slow (Go module + npm downloads per service); BuildKit caches afterwards

## 2. Required environment variables (fail-closed)

`docker-compose.yml` and the postgres bootstrap scripts abort if any of these are empty:

- **Core DB:** `KAGGLE_DB_PASSWORD`
- **Per-service DB passwords** (also consumed by `docker/postgres-init` via `\getenv`): `HELPDESK_DB_PASSWORD`, `REGISTRY_DB_PASSWORD`, `STATCHAT_DB_PASSWORD`, `PMS_DB_PASSWORD`, `RMS_DB_PASSWORD`, `GOVERNANCE_DB_PASSWORD`, `STATSPATIAL_DB_PASSWORD`, `KNOWLEDGE_DB_PASSWORD`, `AIENG_DB_PASSWORD`, `LMS_DB_PASSWORD`, `GIS_DB_PASSWORD`, `BPM_DB_PASSWORD`, `STATFEDERATION_DB_PASSWORD`, `STATIOT_DB_PASSWORD`, `STATDATA_DB_PASSWORD`, `STATCOLLECT_DB_PASSWORD`, `INTEGRATION_DB_PASSWORD`, `RUNOPS_DB_PASSWORD`
- **Secrets/keys:** `STATGATE_REGISTRY_JWT_SECRET`, `STATGATE_INTERNAL_API_KEY`, `STATCITIZEN_CITIZEN_SESSION_SECRET`, `FLASK_SECRET_KEY`, `STATIOT_GATEWAY_SECRET`, `STATCOLLECT_API_KEY`, `STATCOLLECT_ADMIN_KEYS`, `REDIS_PASSWORD`, `MINIO_ROOT_PASSWORD`, `BASIC_AUTH_PASSWORD`, `SUPERSET_SECRET_KEY`
- **Admin users:** `MINIO_ROOT_USER`, `GRAFANA_ADMIN_USER`, `GRAFANA_ADMIN_PASSWORD`, `AIRFLOW_USER`, `AIRFLOW_PASSWORD`, `SUPERSET_ADMIN_USER`, `SUPERSET_ADMIN_PASSWORD`, `BASIC_AUTH_USERNAME`
- **WebRTC (Phase 3 release gate):** `TURN_PUBLIC_URL`, `TURN_REALM`, `TURN_USERNAME`, `TURN_CREDENTIAL`

## 3. Commands

```bash
docker compose build          # build all images (or: docker compose build <svc>)
docker compose up -d          # start the stack
docker compose ps             # status + health
docker compose logs -f <svc>  # follow one service
docker compose down           # stop (keeps volumes)
```

Startup order is dependency-managed: `postgres`/`redis` become healthy first, then backends, then UIs.


## 4. Service port map (host → container)

| Service | Host port | Notes |
|---|---|---|
| postgres | 5432 | shared cluster, all module DBs |
| redis | 6379 | shared event bus/cache |
| statgate-launcher | 3006 | App Launcher entry |
| statgate-analytics (UI) | 5000 | |
| statgate-core (Analytics API) | 8082 | |
| statgate-registry-api | 9090 | `/ready`, `/health` |
| statgate-registry-ui | 3007 | |
| statgate-helpdesk-api / ui | 5006 / 3005 | |
| statchat-backend / frontend | 4000 / 3009 | TURN relay: 3478 + 49160-49200 |
| statgate-pms-api / ui | 8091 / 3010 | |
| statgate-rms-api / ui | 8092 / 3011 | |
| statgate-governance-api / ui | 8093 / 3012 | |
| statgate-trust-api / ui | 8094 / 3013 | |
| statgate-ops-api / ui | 8098 / 3015 | |
| statgate-runops-api | 8100 | platform engineering |
| statgate-integration-fabric | 8097 | Phase 16 |
| statgate-enterprise (search) | 8095 | Phase 47 |
| statgate-enterprise-core | 8096 | workspaces/files |
| knowledge-portal | 8099 | Phases 17/40/41 |
| ai-autonomy | 8101 | Phases 22/31 |
| learning-crm | 8102 | Phases 24/26 |
| geointel | 8103 | Phase 44 |
| bpm-hub | 8104 | Phase 48 |
| statfederation / ui | 8105 / 3016 | Phases 23/30 |
| statiot | 8106 | Phase 27 |
| statdata | 8107 | Phase 37 |
| statgate-spatial-api / ui | 8108 / 3017 | Phases 10/44 (added 2026-08-29) |
| statcollect | 8081 | Phases 11/43 (added 2026-08-29) |
| prometheus / grafana / alertmanager | 9095 / 3003 / 9093 | monitoring |
| minio | 9000 (API) / 9001 (console) | object storage |
| jupyterhub / superset / airflow / mlflow | — / 8088 / 8085 / 5002 | data platforms |

## 5. Post-deployment verification

```bash
docker compose ps                                    # expect: 45 running, 0 unhealthy
docker compose config --quiet                        # validate compose + fail-closed vars
```

Spot-check endpoints (all should return HTTP 200):

```
http://localhost:9090/ready    (registry)      http://localhost:8091/health  (pms)
http://localhost:8108/health   (spatial)       http://localhost:8099/health  (knowledge)
http://localhost:8101/health   (ai-autonomy)   http://localhost:3006/        (launcher UI)
```

## 6. Known behaviors & troubleshooting

- **Port 4000 conflict:** a previously-launched local `StatChat\backend\server.call-moderation.exe` blocks statchat-backend; stop it before `up`.
- **Distroless Go services** (ai-autonomy, bpm-hub, geointel, knowledge-portal, learning-crm) have no in-container healthcheck — verify `/health` from the host (they return 200; the images contain no shell).
- **nginx UI healthchecks** must target `127.0.0.1`, not `localhost` (busybox wget resolves to IPv6 `::1`).
- **Bootstrap is idempotent:** `docker/postgres-init/*.sql` uses guarded `CREATE DATABASE`/`CREATE ROLE`; init runs only when the `pgdata` volume is empty and requires the `*_DB_PASSWORD` vars above on the postgres service.
- **First registration:** the registry DB starts empty (see `00-create-registry-schema.sql`); create the first account via the registry UI/API.
