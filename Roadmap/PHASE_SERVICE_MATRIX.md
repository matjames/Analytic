# StatGate Phase → Service → Status Matrix

**As of:** 29 August 2026
**Purpose:** Map every roadmap phase to the implementing service(s), its deployment status in `docker-compose.yml`, and its reality-based status (per `CURRENT_STATE_AND_WAY_FORWARD.md`).

**Status model:** **OP** = Operational · **IMP** = Implemented (integration/production proof incomplete) · **IP** = In progress · **SPEC** = Specified only · **DEF** = Deferred

## Deployment Stack Inventory (42 services)

| Service | Build source | Role |
|---|---|---|
| postgres, redis | images | Core datastores |
| statgate-launcher | ./appluancher | App Launcher (Next.js entry) |
| statgate-core | ./backend | Analytics Core backend |
| statgate-analytics | ./frontend | Analytics UI (Next.js) |
| statgate-registry-api | ./stage_register/go-backend | Registry (identity/tenancy) |
| statgate-registry-ui | ./stage_register/frontend | Registry UI |
| statgate-helpdesk-api/ui | ./helpdesk-master | Helpdesk |
| statchat-backend/frontend/turn | ./StatChat | Collaboration (Phase 3) |
| statgate-pms-api/ui | ./PMS | Project management (Phase 4) |
| statgate-rms-api/ui | ./RMS | Research management (Phase 5) |
| statgate-governance-api/ui | ./StatGovernance | Governance (Phase 13) |
| statgate-trust-api/ui | ./StatTrust | Security/trust (Phase 19) |
| prometheus, grafana, alertmanager | images | Monitoring (Phase 20) |
| minio | image | Object storage |
| jupyterhub, superset, airflow, mlflow | images | Data/ML platforms (Phases 7–9, 37–38) |
| statgate-enterprise | ./enterprise/search | Enterprise search (Phase 47) |
| statgate-enterprise-core | ./enterprise/core | Enterprise Core (workspaces, files) |
| statgate-runops-api | ./PlatformEngineering/backend | RunOps / platform engineering (Phase 20/25) |
| statgate-integration-fabric | ./enterprise/integration | Integration platform (Phase 16) |
| statgate-ops-api/ui | ./StatOps | Ops / command centre (Phase 49) |
| knowledge-portal | ./knowledge-portal | Open data / knowledge (Phase 17/41) |
| ai-autonomy | ./ai-autonomy | Digital twins / agents (Phase 22/31) |
| learning-crm | ./learning-crm | Learning / capacity (Phase 24) |
| geointel | ./geointel | Geospatial intelligence (Phase 44) |
| bpm-hub | ./bpm-hub | BPM / workflow (Phase 48) |
| statfederation(-ui) | ./StatFederation | Federation / diplomacy (Phase 23/30) |
| statiot | ./StatIoT | IoT / field devices (Phase 27) |
| statdata | ./StatData | Data engineering (Phase 37) |

## Phase Mapping (all 50 phases)

| Phase | Title | Service(s) | In Compose | Status |
|---|---|---|---|---|
| 1 | Foundation & Dev Environment | postgres, redis, monitoring, compose | ✅ | IMP (startup/health certification outstanding) |
| 2 | Identity, Organization & Platform | statgate-registry-api/ui, statgate-enterprise-core | ✅ | IMP (advanced admin done; deployment verification pending) |
| 3 | Communication & Collaboration | statchat-backend/frontend/turn, helpdesk | ✅ | IMP (114/126 route audit; browser E2E + production TURN outstanding) |
| 4 | Project & Portfolio Management | statgate-pms-api/ui | ✅ | IMP (workspace scoping certified; LogFrame/ToC/donor workflows remaining) |
| 5 | Research Ecosystem | statgate-rms-api/ui | ✅ | IMP (workspace scoping certified; DOI/citations/IRB remaining) |
| 6 | Official Statistics & Census | statgate-core (Analytics) | ✅ | IP (questionnaire/sampling/enumeration/tabulation remaining) |
| 7 | Data Management & Governance | statdata, superset | ✅ | IP (classification/retention/OCR/metadata automation remaining) |
| 8 | BI & Analytics | statgate-core, statgate-analytics | ✅ | IP (forecasting, ad-hoc query, NLQ, scheduled reports remaining) |
| 9 | AI & Intelligent Automation | mlflow, ai-autonomy | ✅ | IP (LLM gateway, embeddings, governed RAG = Stage 5) |
| 10 | GIS & Spatial Analytics | statgate-spatial-api/ui (StatSpatial) | ✅ (added 2026-08-29) | IP (PostGIS/spatial analysis/vector tiles remaining) |
| 11 | Field Ops & Mobile Collection | **StatCollect — NOT in Compose**, statiot | ❌ | IP (supervisor ops, GPS streaming remaining) |
| 12 | M&E & Results Management | statgate-pms-api | ✅ | IP (evaluation workflows remaining) |
| 13 | Governance, Risk & Compliance | statgate-governance-api/ui | ✅ | IP (whistleblower/COI/board packs remaining) |
| 14 | Finance, Grants, Procurement | — | ❌ | SPEC (Phase 45 target) |
| 15 | Document & Records Management | statgate-enterprise-core (files) | ✅ | IP (records/archive workflows remaining) |
| 16 | Enterprise Integration Platform | statgate-integration-fabric | ✅ | IP |
| 17 | Public Portals & Open Data | knowledge-portal | ✅ | IP |
| 18 | Marketplace & Plugin Ecosystem | — | ❌ | SPEC |
| 19 | Security Operations & Trust | statgate-trust-api/ui | ✅ | IP |
| 20 | DevOps, CI/CD & Observability | statgate-runops-api, prometheus/grafana | ✅ | IP (CI pipeline is Critical Gap #2) |
| 21 | Tenant Isolation & RBAC | registry + all services | ✅ | IP (test suite for every service = Critical Gap #3) |
| 22 | Digital Twins & Decision Intel | ai-autonomy | ✅ | IP |
| 23 | National Statistics Federation | statfederation | ✅ | IP |
| 24 | Enterprise Learning | learning-crm | ✅ | IP |
| 25 | Sovereign Cloud & Infrastructure | statgate-runops-api | ✅ | IP |
| 26 | Commercial Ecosystem | learning-crm | ✅ | IP |
| 27 | IoT & Sensor Intelligence | statiot | ✅ | IP |
| 28 | Blockchain & Digital Identity | statgate-trust-api | ✅ | IP |
| 29 | Sustainability & Smart Government | — | ❌ | SPEC |
| 30 | Global Federation & Diplomacy | statfederation(-ui) | ✅ | IP |
| 31 | Autonomous AI Organizations | ai-autonomy | ✅ | IP |
| 32 | Innovation Lab | — | ❌ | SPEC |
| 33 | Enterprise Excellence / QMS | — | ❌ | SPEC |
| 34 | Production Readiness & Launch | statgate-ops-api | ✅ | IP |
| 35 | Platform Sustainability | — | ❌ | SPEC |
| 36 | Vision 2050 | — | ❌ | SPEC (vision document) |
| 37 | Data Engineering & DataOps | statdata | ✅ | IP |
| 38 | Scientific & HPC Analytics | jupyterhub, superset | ✅ | IP |
| 39 | Knowledge Graph & Semantics | ai-autonomy (KG) | ✅ | IP |
| 40 | National Digital Library | knowledge-portal | ✅ | IP |
| 41 | National Open Data Platform | knowledge-portal | ✅ | IP |
| 42 | Low-Code Platform | — | ❌ | SPEC |
| 43 | Mobile & Offline Ecosystem | StatCollect | ❌ | SPEC |
| 44 | Geospatial Intelligence & NSDI | geointel | ✅ | IP |
| 45 | Enterprise Finance / ERP | — | ❌ | SPEC |
| 46 | Laboratory / LIMS | — | ❌ | SPEC |
| 47 | Enterprise Search | statgate-enterprise | ✅ | IP |
| 48 | BPM & Workflow | bpm-hub | ✅ | IP |
| 49 | Ops Centers & Command | statgate-ops-api/ui | ✅ | IP |
| 50 | Next Generation / Futures | — | ❌ | SPEC (framework) |

## Confirmed Deployment Gaps (repo → stack)

1. ~~**StatSpatial** absent from docker-compose.yml~~ — **DONE 2026-08-29**: `statgate-spatial-api` (port 8108→4200) and `statgate-spatial-ui` (port 3017→80) added; `StatSpatial/frontend/Dockerfile` created (node build + nginx, `VITE_SPATIAL_API_URL` build-arg). DB bootstrap `13-create-statspatial-db.sql` and `.env` secret already existed.
2. **StatCollect** (`StatCollect/`) — still absent. Integration checklist: (a) add `STATCOLLECT_DB_PASSWORD`, `STATCOLLECT_API_KEY`, `STATCOLLECT_ADMIN_KEYS` to `.env`; (b) add a `22-create-statcollect-db.sql` bootstrap creating DB `statcollect` + role with the password injected from env; (c) confirm StatCollect runs its own `migrations/` on start or wire them into bootstrap; (d) add compose service reusing the shared postgres/redis (its standalone compose uses its own postgres — do NOT duplicate).
3. **CI pipeline exists** (`.github/workflows/integration.yml`: secret scan, all Go modules build+test, compose validation, analytics pytest, 11 frontend builds, StatCollect integration) but has not been proven green; it does not yet include the full-stack `up` + per-service health certification.
4. **No tenant/authorization test suite** spanning every service (Critical Gap #3).
5. Health/readiness endpoints not yet certified for all 42 services from a clean environment (Critical Gap #1).

## Execution Order (per Way Forward)

1. **Stage 1 — Release baseline:** ✅ **CERTIFIED 2026-08-29** — full `docker compose build` completed (37 images), stack brought up: **45/45 containers running, all healthchecks passing**. Fixes required to get there: (a) `statgate-runops-api` build context/Dockerfile; (b) Registry identity bootstrap (`docker/postgres-init/00-create-registry-schema.sql`) — the `users` table never existed on a clean volume; (c) removed impossible `wget` healthchecks on the five distroless Go services (ai-autonomy, bpm-hub, geointel, knowledge-portal, learning-crm — all verified 200 on /health from host); (d) governance-ui healthcheck `localhost`→`127.0.0.1` (busybox wget resolves ::1, nginx listens IPv4 only); (e) StatCollect Dockerfile now ships `migrations/` into the runtime image. Clean-environment bootstrap **VERIFIED 2026-08-29** (non-destructive throwaway postgres test): 19 databases, 14 service roles, and `kaggle.users` created with zero aborts — after fixing 13 init scripts that had unconditional `CREATE DATABASE` statements (converted to guarded `SELECT 'CREATE DATABASE …' WHERE NOT EXISTS … \gexec`). Note: init requires the ~14 `*_DB_PASSWORD` vars on the postgres service (compose already provides them). Remaining: ~~written env/secrets/ports runbook~~ — **DONE 2026-08-29**: see `docs/BASELINE_RUNBOOK.md` (env/secrets inventory, 45-service port map, verification commands, troubleshooting).
2. **Stage 2 — Identity/security closure:** ✅ **ROLLED OUT 2026-08-29 to all 10 gap services** — shared middleware in `statgate-lib/tenant` (gin variants `GinWorkspaceContext`/`GinWorkspaceMembership`; net/http variants `WorkspaceContext`/`WorkspaceMembership` for gorilla/mux services; `VerifyWorkspaceMembership` helper; 12 unit tests). Adopted by: **statdata, statiot, geointel, ai-autonomy, learning-crm, bpm-hub, knowledge-portal, statgate-enterprise (search), statgate-integration-fabric, statgate-runops-api** — all rebuilt, redeployed, healthy (45/45 containers), live-verified (malformed X-Workspace-ID → 400; pass-through without credentials; membership 403/503 mapping unit-tested). Dockerfile/compose notes: enterprise search + integration Dockerfiles rewritten to repo-root context; STATGATE_ENTERPRISE_API_URL wired into all 10. **Remaining Stage 2 follow-ups:** ~~statiot AuthValidator~~ **DONE 2026-08-29** — router restructured into admin (Registry JWT + tenant isolation + workspace membership), device (fail-closed X-Device-UID/X-Device-Token via GatewayEngine.AuthenticateDevice, authenticated identity overrides body device_uid to prevent spoofing) and gateway (X-Gateway-Secret constant-time compare) route groups; telemetry ingest no longer fail-open; live-verified 401s. Remaining: tenant-branding adoption by later-phase UIs; deeper per-domain scoping (statdata contracts/quality child resources, geointel drones/rasters/scenes).

**Resource-level scoping progress:** ✅ **DONE for the later-phase services (2026-08-29)** — statdata (datasets, pipelines), statiot (gateways, devices, alerts, field-workers), geointel (geo_layers), bpm-hub (process_definitions, process_instances), learning-crm (leads), ai-autonomy (agents) — `WorkspaceID` on models; list filters + create stamps + ownership guards (404 on cross-workspace; legacy unscoped rows stay visible and are adopted on first edit); schema columns self-provisioned (statdata `ensureSchema`, statiot `ensureMigrations` runner — fixed latent gaps where statdata/statiot DBs had no tables; geointel/bpm-hub/learning-crm/ai-autonomy ALTERs via ensureSchema). Unit tests: `workspace_scope_test.go` (statdata). knowledge-portal public datasets intentionally left tenant-wide (open-data products are public by design). Note: ~~geointel REDIS_ADDR not set in compose~~ **FIXED 2026-08-29** — 9 compose services used `REDIS_HOST=${REDIS_HOST:-redis}` passthroughs which resolved to the `.env` value `localhost` inside containers; all hardcoded to `REDIS_HOST=redis` (containers recreated, Redis warning gone). Remaining statdata domains (data sources, streaming jobs, feature views) also scoped; **full statdata schema (15 tables) now self-provisioned via ensureSchema** — includes quoting fix for the reserved `values` column in feature_records; contracts/quality remain dataset-child-scoped.
3. **Stage 3 — Cross-module collaboration:** **PARTIAL** — PMS/RMS, Enterprise workflows, and StatCollect surveys now use canonical StatChat object conversations; the shared event envelope is adopted by StatCollect. Field/governance/later-module adoption plus durable event retries, idempotency, and DLQ processing remain. **Event-bus standardization audit (2026-08-29):** docs/EVENT_CATALOG.md extended with sections 2.10–2.16 for all later-phase domains; fixed a taxonomy violation where statiot's `PublishTelemetryIngested` emitted `dataset.updated` (statdata indexed every telemetry batch as a dataset) — now emits `telemetry.ingested` with the device's real tenant.
4. **Stage 4 — Finish product modules:** PMS LogFrame/ToC/donor, RMS DOI/IRB, Statistics questionnaire→dissemination, Analytics forecasting→conversational, Governance whistleblower→board packs, GIS PostGIS certification.
5. **Stage 5 — Advanced intelligence:** LLM gateway, embeddings, governed RAG, model registry (only after 1–4 stabilize).
6. **Stage 6 — Later domains** by dependency/value: data platform → integration/federation → finance/procurement/docs → low-code/learning/mobile → IoT/identity/twins/AI.

