# StatGate Unified API Catalog

**Document Version:** 1.0 (Phase y Convergence)  
**Status:** Authoritative REST API Specifications  
**Scope:** Complete enterprise API endpoints, methods, authentication, and responses.

---

## 1. Enterprise Core Backbone APIs (`:8080`)

### Authentication & Health
- `GET /health` — Liveness & database connection health.
- `GET /ready` — Readiness probe for load balancing.
- `GET /live` — Kubernetes / Docker liveness probe.
- `GET /metrics` — Prometheus metrics exposition.

### Event Bus Ingress
- `POST /api/v1/events` — Ingress for domain events. Persists to PostgreSQL and broadcasts to Redis `statgate:events`.
- `GET /api/v1/events` — Query historical events with tenant filtering, pagination, and correlation ID search.

### Universal Object Fabric & Search
- `GET /api/v1/fabric/resolve?urn={urn}` — Resolves universal object identity (e.g. `tenant:app:type:id`) into full entity state, relationships, and metadata.
- `POST /api/v1/fabric/relationships` — Creates directional link between two universal objects.
- `GET /api/v1/fabric/search?q={query}` — Cross-application federated search.

### Enterprise Workflows & Action Centre
- `POST /api/v1/workflows` — Register declarative workflow definition.
- `POST /api/v1/workflows/{id}/start` — Instantiate workflow instance for an object.
- `POST /api/v1/workflows/tasks/{task_id}/transition` — Transition task state with comments and human approval.
- `GET /api/v1/action-centre/items` — Aggregated action items and pending approvals.

### Decision Records & Evidence
- `POST /api/v1/decisions` — Create immutable institutional decision record.
- `GET /api/v1/decisions/{id}` — Retrieve decision with attached evidence records and audit history.
- `POST /api/v1/evidence` — Attach document or dataset evidence to decision.

### Enterprise Analytics & KPIs
- `GET /api/v1/analytics/kpis` — Real-data executive KPIs with definitions and provenance.
- `GET /api/v1/analytics/anomalies` — Real-time statistical anomalies and alerts.
- `POST /api/v1/analytics/investigate` — Trigger evidence-based AI analysis session.

---

## 2. Domain Application APIs

### 2.1 Registry (`:8081`)
- `POST /api/auth/login` — Authenticate credentials and receive high-entropy JWT.
- `GET /api/auth/me` — Return authenticated user profile, roles, and tenant context.
- `GET /api/users` — Paginated user directory.
- `GET /api/organizations` — Institutional organizational hierarchy.
- `GET /api/facilities` — Master facility registry.

### 2.2 PMS (`:8091`)
- `GET /api/v1/projects` — List capital projects with pagination and filtering.
- `POST /api/v1/projects` — Create project (publishes `project.created`).
- `GET /api/v1/projects/{id}` — Project details and milestone breakdowns.
- `PUT /api/v1/projects/{id}/milestones/{mid}` — Update milestone progress.

### 2.3 RMS (`:8092`)
- `GET /api/v1/research` — List research protocols and publication status.
- `POST /api/v1/research` — Register research protocol (publishes `research.created`).
- `POST /api/v1/research/{id}/ethics-review` — Submit IRB ethics review.

### 2.4 StatCollect (`:8082`)
- `GET /api/v1/surveys` — Active field survey forms.
- `POST /api/v1/surveys` — Create new survey instrument (publishes `survey.created`).
- `POST /api/v1/submissions` — Ingest enumerator submission (publishes `submission.received`).

### 2.5 HelpDesk (`:8083`)
- `GET /api/v1/tickets` — Paginated service desk tickets.
- `POST /api/v1/tickets` — Create support ticket (publishes `ticket.created`).
- `POST /api/v1/tickets/{id}/assign` — Assign agent to ticket.

### 2.6 StatChat (`:8084`)
- `GET /api/v1/channels` — Team channels.
- `GET /api/v1/discussions?context={urn}` — Retrieve discussion thread for universal object.
- `POST /api/v1/messages` — Post message (publishes `message.created`).

### 2.7 StatGovernance (`:8085`)
- `GET /api/v1/risks` — Institutional risk register.
- `POST /api/v1/findings` — Log audit finding and corrective action.

### 2.8 StatSpatial (`:4200`)
- `GET /api/v1/admin-units` — Boundary hierarchy (country -> region -> district -> sub-county).
- `GET /api/v1/admin-units/tree` — Complete administrative tree.
- `GET /api/v1/layers` — GIS map layers.
- `GET /api/v1/features` — GeoJSON features.
- `GET /api/v1/spatial-index` — Target entity geographic coordinates.
- `GET /api/v1/nodes` — Federated external data nodes.
- `GET /api/v1/summary` — Geospatial intelligence summary.
