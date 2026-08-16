# PHASE 6 — OFFICIAL STATISTICS, SURVEYS & CENSUS MANAGEMENT

## Objective

Develop the complete Official Statistics Production Platform aligned with the Generic Statistical Business Process Model (GSBPM), enabling institutions to design, collect, process, analyse, disseminate, and preserve official statistics.

This phase establishes StatGate as a comprehensive statistical production system for governments, NGOs, and research organisations.

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

### What is Missing ❌

- **Questionnaire Designer / Form Builder** — StatCollect only receives pre-built ODK XForms; no in-platform form designer exists
- **Question Bank** — no reusable question library
- **Survey project management** — surveys are entities in PMS/RMS but not managed end-to-end here
- **Sampling Designer** — no sample frame, sampling design, or enumeration area management
- **Supervisor Dashboard** — no server-side supervisor console for field monitoring
- **Assignment Management** — no enumerator assignment or workload management
- **Weighting, Imputation, Coding** — no statistical processing pipeline on received submissions
- **Tabulation Engine** — no cross-tabulation or indicator calculation pipeline
- **Statistical Calendar** — not implemented
- **Dissemination Portal / Open Data Portal** — not implemented

---

## Service: Analytics Core (Statistical Backbone)

**Repository:** `backend/`  
**Backend:** Go + Gin (:8082)  
**Database:** PostgreSQL 15 (`statgate` database, `ml_staging` schema)

### What Exists ✅

**Semantic Indicator Registry** (`backend/internal/semantic/`)
- Indicators registered with name, description, formula, data source, and clearance level
- API: `GET /api/semantic/indicators`, `POST /api/semantic/indicators`

**In-Memory Lakehouse** (`backend/internal/lakehouse/`)
- Ingests tabular data (CSV, database tables) into an in-memory columnar store
- Supports analytical queries for dashboard widgets

**ABAC Engine** (`backend/internal/abac/`)
- Attribute-Based Access Control policy evaluation for row-level data access

**Anomaly Detection Worker** (`backend/internal/lakehouse/anomaly_worker.go`)
- 3-sigma statistical anomaly detection on time-series lakehouse data
- Publishes alerts on detected anomalies

**Asset Manager** (`backend/internal/assets/`)
- Dataset catalog: register, list, search datasets
- Schema discovery and schema health checks

### What is Missing ❌

- **Survey/Questionnaire Designer** — not in analytics core
- **Sampling framework** — no sample frame management, stratification, or cluster sampling
- **Enumeration area management** — not implemented
- **Tabulation engine** — no crosstab or pivot calculation
- **Indicator metadata (SDMX/DDI)** — semantic registry is lightweight; no full SDMX or DDI compliance
- **Data editing & cleaning pipeline** — no structured post-collection editing workflow
- **Population projections / forecasting** — anomaly detection exists but not projections
- **Census management** — not implemented
- **Dissemination portal** — not implemented
- **SDG monitoring framework** — not implemented
- **Open data API** — not implemented
- **SDMX/DDI/NQAF standards** — not implemented

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

### What is Missing ❌

- **Survey designer interface** — no questionnaire builder in Flask UI
- **Statistical production workflow UI** — no GSBPM-aligned workflow screens
- **Tabulation and cross-tab UI** — not implemented
- **National dashboard / SDG monitoring screen** — not implemented
- **Dissemination portal** — not implemented

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

## Database Tables (to be implemented)

```sql
-- StatCollect
submissions, submission_attachments, submission_metadata

-- Analytics Core
ml_staging.indicators, ml_staging.indicator_metadata
ml_staging.datasets, ml_staging.dataset_versions
ml_staging.anomalies, ml_staging.schema_health

-- Future (to build)
survey_projects, questionnaires, question_bank, survey_templates
sample_frames, sampling_designs, enumeration_areas
enumerators, field_supervisors, assignment_plans
tabulation_outputs, statistical_publications
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
- [ ] In-platform questionnaire/survey designer operational
- [ ] Sampling designer and enumeration area management operational
- [ ] Enumerator assignment management operational
- [ ] Supervisor field monitoring dashboard operational
- [ ] Tabulation engine operational
- [ ] Indicator metadata (SDMX-compliant) operational
- [ ] Open data portal / dissemination portal operational
- [ ] Census management operational
- [ ] SDG monitoring framework operational

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