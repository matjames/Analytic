# PHASE 6 — OFFICIAL STATISTICS, SURVEYS & CENSUS MANAGEMENT

## Objective

Develop the complete Official Statistics Production Platform aligned with the Generic Statistical Business Process Model (GSBPM), enabling institutions to design, collect, process, analyse, disseminate, and preserve official statistics.

This phase establishes StatGate as a comprehensive statistical production system for governments, NGOs, and research organisations.

## Current Deployment Status (4 October 2026)

The Phase 6 StatCollect backend is live-certified as of 4 October 2026. The deployed image includes migrations `003_phase6_official_statistics.sql` through `010_dataset_handoffs.sql`, and covers durable questionnaire templates/version history, workspace-scoped field submissions, approved-submission-backed tabulation, durable scoped weighting, imputation, coding, structured-editing runs, formula-driven indicator calculation with bounds/unit validation, standards-based indicator quality checks and revalidation, versioned indicator observation publication, processed-dataset records/schema/lineage handoff, authenticated Analytics Core observation handoff with lineage, the question bank, survey projects and GSBPM workflow, XForm generation, sampling and enumeration areas, field workforce assignments, supervisor review and quality signals, CSV export, SDMX metadata/data responses, census/PES calculations, SDG indicators/dashboard data, and dissemination/open-data endpoints.

The release checks passed against the running service: StatCollect tests pass; representative Phase 6 endpoints return `200`; missing admin credentials fail with `401`; persisted administrative records and field submissions are forced into the authenticated tenant/workspace rather than trusting request-body ownership; records created in one workspace are invisible in a sibling workspace; scoped deletion is enforced; template updates create durable version history; submission listing, detail, export, QA, validation, approval, and rejection use the same boundary; approved submissions can produce durable weighting, imputation, coding, and structured-editing runs; failed edits remain auditable; a completed weighting run can feed scoped official tabulation; formula calculation produced a mean of `30` and a proportion of `50`, rejected an out-of-range result with `422`, and rejected a sibling-workspace processing run with `422`; quality publication records a `passed_with_warnings` small-source result, revalidation returns the same quality state, invalid percentage/unit values return `422`, and foreign-workspace revalidation returns `404`; a processed dataset handoff returned `201`, stored an `official_statistics_dataset` asset with two records, registered a daily Core schedule, and remained invisible in a sibling workspace; versioned indicator observations update the scoped SDG latest value and open-data feed without crossing workspaces; and the authenticated Analytics Core handoff creates a workspace-owned official-observation asset while recording handoff history. The implementation is in `StatCollect/internal/server/official_statistics.go`, `StatCollect/internal/server/official_statistics_handlers.go`, `StatCollect/internal/server/official_statistics_scope.go`, `StatCollect/internal/server/processing.go`, `StatCollect/internal/server/indicator_observations.go`, `StatCollect/internal/server/dataset_handoffs.go`, `StatCollect/internal/server/templates.go`, `StatCollect/internal/server/db.go`, `backend/cmd/server/main.go`, `backend/cmd/server/processing_schedules.go`, and migrations `003`-`010`.

The 4 October release certification covered the StatCollect service slice, dataset/indicator handoff slice, and Analytics Core schedule registration, not the entire Phase 6 milestone. The 5 October source checkpoint below adds a scheduled refresh runner but is not yet live-certified. Remaining gates are live scheduler certification, automated transformation reruns and broader lineage graph integration, browser-facing statistical production screens, collect-master branding/authentication/assignment integration, and public dissemination UI.

### Scheduled Dataset Refresh Implementation Checkpoint: 5 October 2026

Analytics Core now has a durable due-schedule worker, per-execution history, scoped schedule/run reads, manual trigger support, and short retry scheduling for upstream/server failures. StatCollect exposes an internal-key-only refresh endpoint that resolves the dataset's source form and republishes only when a newer completed processing run exists in the same tenant/workspace; otherwise the recorded outcome is `no_change`. Recurring schedules with no explicit first-run time default to one frequency interval from registration, and `on_demand` schedules remain unscheduled until manually triggered. Compose wires Core to StatCollect over the internal service network.

This is an implementation checkpoint, not a live deployment certification. The scheduled action refreshes the latest completed output; it does not yet rerun processing transformations against newly arrived submissions. Automated transformation reruns, full Analytics Core lineage graph integration, and live schedule execution certification remain open, alongside browser workflow and user-facing production gaps below.

---

## Vision

Provide a unified environment for the production of high-quality official statistics, supporting household surveys, facility assessments, censuses, administrative data systems, indicator management, and statistical dissemination.

---

## Services Involved

This phase spans two services and the analytics core:

| Service | Technology | Port |
|---|---|---|
| StatCollect (data collection) | Go + Gin | :8080 |
| Analytics Core (analytics backbone) | Go + Gin | :8082 |
| Analytics UI (statistical workspace) | Python / Flask | :5000 |
| collect-master (Android app) | Kotlin / ODK Collect fork | N/A |

---

## Service: StatCollect

**Repository:** `StatCollect/`  
**Backend:** Go + Gin (:8080)  
**Database:** PostgreSQL (submissions) + Redis event bus  
**Role:** Receives ODK-format multipart survey submissions from collect-master (Android) and web forms; validates, stores, and publishes events to `statgate:events` Redis channel.

### What Exists ✅

**Submission endpoints**
```
POST   /submissions                    ← ODK multipart form submission (collect-master)
GET    /submissions                    ← list submissions (admin)
GET    /submissions/:id                ← submission detail
GET    /submissions/:id/attachments    ← media attachments (photos, audio)
PUT    /submissions/:id/status         ← update submission status (validate, reject)
DELETE /submissions/:id                ← delete submission
```

**OpenRosa/ODK protocol**
```
GET    /formList                       ← ODK formList (XML)
GET    /manifest                       ← ODK manifest
POST   /submission                     ← ODK submission endpoint
```

**Admin UI** — HTML admin interface for viewing and managing submissions

**Redis events published**
```
submission.received     ← on successful receipt
submission.validated    ← on status → validated
submission.rejected     ← on status → rejected
```

**Object linkage** — `object_links` table entry created linking submission to research project or survey project

**Durable processing endpoints**
```
POST /admin/statistics/processing/weight    -> approved-submission weighting run
POST /admin/statistics/processing/transform -> imputation, coding, or structured-editing run
GET  /admin/statistics/processing/runs      -> scoped processing-run history
POST /admin/statistics/datasets/handoff    -> dataset records, schema, and lineage handoff
GET  /admin/statistics/datasets/handoffs   -> scoped dataset handoff history
POST /admin/statistics/tabulate             -> raw, form-backed, or completed-run-backed table
POST /admin/sdg/observations/calculate       -> formula-driven scoped calculation and publication
POST /admin/sdg/observations/validate        -> scoped standards and quality revalidation
POST /admin/sdg/observations/publish        -> versioned indicator observation publication
GET  /admin/sdg/observations                -> current or historical observations
GET  /admin/sdg/observations/handoffs       -> scoped Analytics Core handoff history
GET  /api/v1/open-data/observations         -> public current observations
POST /api/v1/processing/schedules           -> create/update scoped recurring or on-demand schedule
GET  /api/v1/processing/schedules           -> list schedules in the caller's workspace
POST /api/v1/processing/schedules/run       -> manually trigger a scoped schedule
GET  /api/v1/processing/schedules/runs      -> scoped execution history
POST /internal/statistics/datasets/refresh  -> internal-key-only latest-output refresh
```

### Remaining Completion Work

- **Production processing depth** — durable approved-submission-backed weighting, imputation, coding, structured editing, formula-driven indicator calculation, and automated quality checks are live.
- **Scheduled refresh** — durable schedule execution/history, workspace-scoped manual triggering, and latest-output refresh are implemented; live deployment certification and automatic transformation reruns are still open.
- **Indicator publication depth** — versioned, scoped observation publication, bounds/unit validation, quality warnings/rejection, revalidation, open-data readback, and Analytics Core observation handoff are live.
- **Statistical calendar** — not implemented.
- **Browser workflow depth** — the API is live-certified, but all design, sampling, workforce, review, tabulation, SDMX, census, SDG, and dissemination workflows still need role-aware browser acceptance coverage.

---

## Service: Analytics Core (Statistical Backbone)

**Repository:** `backend/`  
**Backend:** Go + Gin (:8082)  
**Database:** PostgreSQL 15 (`statgate` database, `ml_staging` schema)

### What Exists ✅

**Semantic Indicator Registry** (`backend/internal/semantic/`)
- Indicators registered with name, description, formula, data source, and clearance level
- API: `GET /api/semantic/indicators`, `POST /api/semantic/indicators`

**Durable Lakehouse** (`backend/internal/lakehouse/`)
- Persists analytical assets and workspace-scoped catalog metadata in PostgreSQL
- Supports analytical queries for dashboard widgets

**ABAC Engine** (`backend/internal/abac/`)
- Attribute-Based Access Control policy evaluation for row-level data access

**Anomaly Detection Worker** (`backend/internal/lakehouse/anomaly_worker.go`)
- 3-sigma statistical anomaly detection on time-series lakehouse data
- Publishes alerts on detected anomalies

**Asset Manager** (`backend/internal/assets/`)
- Dataset catalog: register, list, search datasets
- Schema discovery and schema health checks

### Remaining Completion Work

- **Production handoff** — workspace-scoped dataset records, schema/lineage assets, indicator assets, and schedule registration are live. A durable scheduled refresh runner/history and manual trigger are implemented but await live certification; full lineage graph integration and automated transformation reruns remain.
- **Standards depth** — extend the current SDMX-shaped responses with persistent DDI/NQAF metadata and validation where required.
- **Data editing and cleaning** — the durable post-collection workflow now covers weighting, imputation, coding, structured editing, formula-driven indicator calculation, indicator observation publication, and approval history.
- **Forecasting and projections** — population projections and forecasting remain later analytical work.

---

## Service: Analytics UI (Flask)

**Repository:** `frontend/`  
**Technology:** Python / Flask + Jinja2 templates (:5000)  
**Key files:** `app.py`, `engine.py`, `agentic_engine.py`, `schema_healer.py`, `kaggle_connector.py`

### What Exists ✅

| Route | Description |
|---|---|
| `/` | Analytical dashboard |
| `/datasets` | Dataset catalog and schema explorer |
| `/notebook` | Interactive notebook UI |
| `/executive` | Executive command centre |
| `/semantic` | Semantic indicator registry |
| `/abac` | ABAC policy matrix |
| `/health`, `/ready`, `/metrics` | Health and Prometheus metrics |
| `/api/datasets/*` | Dataset CRUD and schema APIs |
| `/api/agent/*` | Agentic report engine API |
| `/api/schema-health/*` | Schema health check API |
| `/api/kaggle/*` | Kaggle connector API |

**Agentic Engine** (`agentic_engine.py`) — rule-based automated report generation with agent feedback loops

**Schema Healer** (`schema_healer.py`) — AI-assisted detection and suggestion for schema anomalies

**Analysis Engine** (`engine.py`) — Pandas + DuckDB for SQL-on-dataframe analysis

### Remaining Completion Work

- **Survey designer interface** — no complete browser questionnaire builder consuming the certified StatCollect API.
- **Statistical production workflow UI** — GSBPM, sampling, workforce, review, and approval screens are not yet complete in the Flask UI.
- **Tabulation and cross-tab UI** — backend tabulation is live, but the browser workflow is not complete.
- **National dashboard / SDG monitoring screen** — backend indicators are live, but the browser dashboard and role-aware publication workflow remain.
- **Dissemination portal** — open-data APIs are live; the public portal, metadata browsing, downloads, and publication governance UI remain.

---

## collect-master (Android App)

**Repository:** `collect-master/`  
**Type:** Android application (ODK Collect fork, v8.6 upstream, Gradle/Kotlin)  
**Role:** Field data capture — offline form filling, GPS stamping, photo/audio/barcode capture, background sync to StatCollect

### What Exists ✅
- Full ODK Collect Android app (fork of upstream v8.6)
- Offline form storage and sync to `POST /submission` on StatCollect
- GPS validation, photo/audio capture, barcode/QR scanning
- Background synchronisation with conflict handling (ODK default)

### What is Missing ❌
- Custom StatGate branding and configuration screen
- Direct Registry JWT authentication (currently uses ODK's own auth model)
- Supervisor console integration (no in-app supervisor view)
- Assignment management UI (enumerators cannot see their assigned surveys in-app)

---

## Database Tables

```sql
-- StatCollect
submissions, submission_attachments, submission_metadata

-- Analytics Core
ml_staging.indicators, ml_staging.indicator_metadata
ml_staging.datasets, ml_staging.dataset_versions, processing_schedules, processing_schedule_runs
ml_staging.anomalies, ml_staging.schema_health

-- Delivered by StatCollect migrations 003-010
survey_projects, question_bank, sample_frames, enumeration_areas
enumerators, field_supervisors, assignment_plans, supervisor_reviews
tabulation_outputs, statistical_publications, census_rounds, census_pes
sdg_indicators, indicator_observations, analytics_handoffs, analytics_dataset_handoffs
templates, template_versions, submissions

-- Remaining persistence to build
sampling_designs, statistical_calendar
census_operations, population_projections
```

---

## Integration Points

| Module | Integration |
|---|---|
| collect-master | Submits to StatCollect via ODK protocol |
| StatCollect → Redis | Publishes `submission.*` events to `statgate:events` |
| StatCollect → RMS | Submissions linked to research projects via `object_links` |
| StatCollect → PMS | Surveys linked to project activities via `object_links` |
| Analytics Core → Enterprise Core | Datasets registered in asset catalog |
| Analytics Core → StatChat | Anomaly alerts can trigger StatChat notifications |

---

## Acceptance Criteria

- [x] StatCollect receives ODK submissions from collect-master (Android)
- [x] Submission validation and status management operational
- [x] Redis events published on submission lifecycle events
- [x] Object links created connecting submissions to research/projects
- [x] Semantic indicator registry operational
- [x] Dataset catalog and schema health checks operational
- [x] Anomaly detection (3-sigma) operational
- [x] Flask analytics UI with dataset catalog and notebook operational
- [x] Agentic report engine operational
- [x] StatCollect question bank, survey workflow, XForm generation, sampling, enumeration, assignments, supervisor monitoring, approved-submission-backed tabulation, SDMX-shaped responses, census/PES, SDG, and dissemination API slice operational
- [x] Tenant/workspace isolation and admin authorization live-certified for administrative data, templates, version history, and field submissions
- [x] Durable questionnaire/template persistence, workspace isolation, version history, rollback, import/export, and scoped browser API context
- [x] Durable workspace-scoped weighting, imputation, coding, and structured-editing runs with approved-submission inputs and run-backed weighted tabulation
- [x] Versioned workspace-scoped indicator observation publication with processing-run provenance and open-data readback
- [x] Authenticated workspace-scoped Analytics Core observation handoff with durable handoff history and lineage asset
- [x] Workspace-scoped processed-dataset handoff with records, inferred schema, lineage history, and durable schedule registration
- [x] Formula-driven indicator calculation and bounds/unit validation
- [x] Standards validation and automated indicator quality checks
- [x] Durable scheduled dataset refresh runner, workspace-scoped execution history, manual trigger, and retry for transient server/upstream failures
- [ ] Automatically rerun weighting/imputation/coding/editing from new submissions and integrate complete dataset/indicator lineage graph
- [ ] Browser statistical production workflow and public dissemination portal
- [ ] collect-master StatGate branding, direct identity integration, and assignment-aware field UX
- [ ] Statistical calendar and population projections

---

## Ports & Services

| Component | Port |
|---|---|
| StatCollect API (Go) | :8080 |
| Analytics Core (Go) | :8082 |
| Analytics UI (Flask) | :5000 |

---

## Estimated Duration

12–14 weeks

## Milestone

Enterprise Statistical Production Platform complete. Surveys designed, deployed, collected, processed, and disseminated from a single system.
