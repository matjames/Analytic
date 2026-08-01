# StatGate Build Plan

This plan turns the current analytics prototype into a production-ready system by working service-by-service and phase-by-phase. The goal is to stabilize each service independently before integrating them into the full stack.

## Operating principle

- Build each service to a clear production definition.
- Freeze interfaces before integrating new services.
- Treat demo behavior as temporary and remove it before release.
- Do not move to the next phase until the prior phase passes health + test checks.

---

## Phase 0 — Stabilize the foundation

### Objective
Establish a reliable local runtime and a single source of truth for configuration.

### Deliverables
- Clean Python environment for the analytics frontend
- Validated environment variables and secrets loading
- A reproducible local start flow for analytics services
- A clear distinction between dev mode and production mode

### Tasks
- [ ] Replace the fragile vendored Python environment with a clean venv workflow
- [ ] Verify Python 3.12/3.13 compatibility for frontend dependencies
- [ ] Standardize `.env` and secret loading across local + Docker runtime
- [ ] Document required environment variables by service
- [ ] Add fail-fast startup checks for DB and backend connectivity
- [ ] Remove fallback behavior that hides bad config in production mode

### Definition of done
- The analytics app starts from a clean environment without relying on repo-local Python stubs
- Missing config causes clear startup errors
- Local dev run and Docker run both produce the same service contract

---

## Phase 1 — Stabilize the core Go service

### Service
`statgate-core`

### Objective
Make the Go orchestration engine production-stable and contract-based.

### Deliverables
- Healthy API surface for telemetry ingest, stats, alerts, indicators, and assets
- ABAC enforcement with tenant isolation
- Metrics and health endpoints
- Service-level tests for core API behavior

### Tasks
- [ ] Review and split `backend/cmd/server/main.go` into clearer route groups and handlers
- [ ] Add contract tests for `/health`, `/api/v1/query`, `/api/v1/ingest`, `/api/v1/stats`
- [ ] Harden ABAC enforcement with explicit policy validation and tenant checks
- [ ] Review internal key auth and ensure production-only behavior is enforced
- [ ] Improve logging and audit records for security and platform events
- [ ] Keep in-memory lakehouse as a dev/demo layer, but define a migration path to durable storage
- [ ] Add graceful failure handling for malformed payloads

### Definition of done
- The backend is the trusted source for telemetry, metrics, and alert orchestration
- All endpoints return predictable and documented responses
- Core service passes health checks without relying on demo data only

---

## Phase 2 — Stabilize the analytics service

### Service
`statgate-analytics`

### Objective
Make the Flask app a clean analytics frontend and API gateway without monolithic drift.

### Deliverables
- Clean Python app structure
- Stable route organization
- Reliable dashboard, dataset, and semantic APIs
- Frontend runtime without env masking

### Tasks
- [ ] Split [frontend/app.py](frontend/app.py) into modular components: auth, routes, proxy, analytics, assets, notebook
- [ ] Move dataset and schema logic behind service boundaries
- [ ] Centralize request headers and identity context for all analytics calls
- [ ] Standardize API responses for all routes
- [ ] Remove silent fake-data fallback from production code paths
- [ ] Make all runtime connections explicit and testable
- [ ] Verify the UI routes are working with the real backend contract

### Definition of done
- The Flask app is not a monolith
- Route behavior is deterministic
- The analytics UI can run with real backend responses and real config

---

## Phase 3 — Stabilize data access and schema engine

### Services
`kaggle_connector`, schema monitoring, DB-backed data discovery

### Objective
Make the analytics engine depend on real, validated data sources rather than soft fallbacks.

### Deliverables
- Safe DB connection lifecycle
- Table discovery and schema validation
- Drift detection with clear rules
- Production-safe defaults for missing data

### Tasks
- [x] Harden [frontend/kaggle_connector.py](frontend/kaggle_connector.py) connection and retry logic
- [x] Remove or gate silent static dataset fallback in production mode
- [x] Define a real table naming and schema contract for analytics datasets
- [x] Add schema validation tests for known dataset tables
- [x] Add dataset refresh and catalog discovery tests for newly stored tables
- [x] Review [frontend/schema_healer.py](frontend/schema_healer.py) and convert heuristics into explicit migration rules
- [x] Add snapshot baseline generation and drift comparison contracts
- [x] Add route tests for schema snapshot and engine contract behavior
- [x] Standardize how data source failures are surfaced to the UI

> Phase 3 complete: dataset access contract enforcement is implemented and validated.

> Phase 4 ready: analytics decision-support and agentic scenario logic can now be started.

### Definition of done
- Analytics data access works against real DB metadata without hidden fallback assumptions
- Schema drift is auditable and recoverable
- Dataset errors are visible and actionable

---

## Phase 4 — Stabilize decision support and agentic analytics

### Services
`agentic_engine`, semantic registry, report generation

### Objective
Turn the rule-based analytics engine into a traceable decision-support layer with consistent outputs.

### Deliverables
- Scenario generation with validated formulas
- Executive summary output with hidden assumptions removed
- Feedback capture and persisted reports
- Clear action-priority model

### Tasks
- [ ] Review `agentic_engine.py` formulas and replace placeholder logic with validated patterns
- [ ] Define the data contract for report generation and scenario output
- [ ] Add tests for domain extraction, scenario generation, and executive summaries
- [ ] Document which scenarios are advisory only versus operationally enforced
- [ ] Review RLHF feedback storage and ensure it is safe and tenant-aware
- [ ] Add limit caps for dataset analysis to avoid performance drift

### Definition of done
- The engine produces consistent, explainable recommendations
- All generated scenarios are auditable and reproducible
- Reports are persisted with structured metadata

---

## Phase 5 — Stabilize registry integration

### Service
`statgate-registry` integration path

### Objective
Connect the analytics layer to the registry in a secure, tenant-aware way.

### Deliverables
- Registry JWT validation contract
- Identity-to-tenant mapping
- Authenticated analytics access flows
- Error handling for registry outages

### Tasks
- [ ] Validate registry JWT claims against the real platform identity model
- [ ] Standardize tenant and district mapping between registry and analytics
- [ ] Add failure modes for registry unavailability
- [ ] Confirm that auth-required routes work with real tokens
- [ ] Separate local dev login from production auth flow
- [ ] Document supported roles and clearance levels across the platform

### Definition of done
- Analytics and registry are aligned on identity, roles, and tenant scope
- Unauthenticated access is blocked by design
- Registry outages are visible and safe

---

## Phase 6 — Stabilize asset and notebook workflows

### Services
`assets`, notebook execution sandbox, dashboard persistence

### Objective
Make saved assets and notebook usage reliable and safe enough for trusted users.

### Deliverables
- Safe asset save/load APIs
- Versioned dashboard and notebook storage
- Notebook execution controls that are off by default
- Clear ownership and tenant scoping

### Tasks
- [ ] Review asset schema and persistence flow in the Go asset manager
- [ ] Ensure saved assets include ownership and version metadata consistently
- [ ] Validate dashboard save/load endpoints against tenant ownership rules
- [ ] Keep notebook execution disabled unless explicitly allowed in trusted environments
- [ ] Add tests around asset validation and malformed payload handling
- [ ] Document asset lifecycle and versioning rules

### Definition of done
- Saved assets are consistent and tenant-aware
- Notebook execution is safe by default
- Asset history is auditable

---

## Phase 7 — Production hardening and monitoring

### Objective
Lock the system down for operational use.

### Deliverables
- Metrics and logs for all services
- Deployment checks and health tests
- Alerting and observability baseline
- Deployment documentation for runtime operations

### Tasks
- [ ] Review Prometheus and Grafana integration for actual service level coverage
- [ ] Ensure all services emit structured health and failure logs
- [ ] Add end-to-end smoke tests for startup, health, and core endpoints
- [ ] Document deployment requirements for Docker, DB, secrets, and registry config
- [ ] Validate backup and recovery strategy for local data and asset states
- [ ] Set a clear release checklist before production promotion

### Definition of done
- Each service has an operational health definition
- Monitoring is meaningful and tied to business-critical services
- The stack can be deployed and verified repeatedly

---

## Phase 8 — Release readiness

### Objective
Prepare the system for controlled handoff and live use.

### Deliverables
- Release checklist
- Final security review
- Known risks and mitigations
- Production rollout plan

### Tasks
- [ ] Review all service readiness checklists
- [ ] Confirm auth, tenant, and data-access rules are enforced
- [ ] Validate hidden fallback paths are removed or explicitly gated
- [ ] Freeze the supported runtime matrix
- [ ] Prepare a deployment runbook and rollback procedure
- [ ] Document scope limits and known technical debt

### Definition of done
- The stack is ready for one controlled production-like deployment
- Risk areas are documented with owners and mitigation plans

---

## Service-by-service execution order

1. Core Go service
2. Analytics Flask frontend
3. Data access and DB contracts
4. Agentic analytics and semantic registry
5. Registry integration and auth
6. Asset and notebook workflows
7. Monitoring and observability
8. Release readiness

---

## Immediate next build block

The first actionable milestone is:

- Phase 0: fix the Python environment and config contract
- Phase 1: harden the Go core service
- Phase 2: clean up the Flask app structure

This is the smallest sequence that removes the highest-risk setup problems before we add more features.
