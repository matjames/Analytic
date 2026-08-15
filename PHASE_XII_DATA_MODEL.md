# PHASE_XII_DATA_MODEL.md
# StatGate Phase XII — Data Model

Additive migration: `docker/postgres-init/12-create-phase12-intelligence.sql`
(database `statgate_enterprise`). No Phase I–XI table is modified. Every Phase XII
table is tenant-scoped; `UNIQUE (tenant_id, canonical_id)` prevents duplicates.

## Tables

| Table | Purpose |
|---|---|
| `institutional_objects` | Projection registry of UOI-canonical objects (identity/relationship layer; never a second copy). Fields: `tenant_id, canonical_id, object_type, source_system, source_object_id, metadata, created_at, updated_at` (+ projection_status). |
| `institutional_graph_edges` | Knowledge graph adjacency list. `tenant_id, subject_canonical_id, relationship_type, object_canonical_id, metadata, provenance_type, confidence, valid_from, valid_until, created_by, created_at`. Deletion is time-boxed (`valid_until`) — history is never destroyed. |
| `intelligence_signals` | Computed signals consumed from existing systems. `signal_type, domain, severity(0-100), source_system, source_event_id, status, payload, correlation_id, evidence`. |
| `institutional_conditions` | Deterministic condition snapshots. `condition_level, composite_score, calculation_timestamp, supporting_signals(jsonb), affected_domains(jsonb), explanation, calculation_version, correlation_id`. |
| `institutional_objectives` | Strategic objective projection (StatGovernance remains source of truth). |
| `kpis` | `name, definition, owner, target, measurement_period, actual_value, unit, source, status, data_status` (ACTUAL/ESTIMATED/STALE/MISSING/INVALID), `freshness_window_h, last_measured_at`. |
| `kpi_measurements` | Append-only measurement history; indexed `(kpi_id, measured_at DESC)`; partition-ready. |
| `risk_events` | Risk correlation events; `risk_reference` points to the authoritative register. Not a competing register. |
| `ai_recommendations` | AI lifecycle records. `provider, model, request_id, correlation_id, status, recommendation, confidence, ... ai_label='AI_GENERATED', reviewed_by, human_decision, executed_at`. |
| `ai_audit_log` | Dedicated AI audit trail: `provider, model, request_id, correlation_id, input/output classification, recommendation, confidence, timestamp, actor, tenant_id, authorization_status, human_decision`. Sensitive prompts are never stored. |

## Query pattern: depth-limited traversal (adjacency list + recursive CTE)

Every traversal is scoped by `tenant_id` and capped at `DefaultMaxTraversalDepth`
(5 hops; 7 benchmark-only). Indexes on `(tenant_id, subject_canonical_id)` and
`(tenant_id, object_canonical_id)` support efficient recursive CTE traversal.

## Relationship types (initial — directive §7)

`SUPPORTS, CONTRIBUTES_TO, AFFECTS, DEPENDS_ON, PRODUCED_BY, OWNED_BY,
ASSIGNED_TO, FUNDED_BY, GOVERNED_BY, ASSOCIATED_WITH, OBSERVED_IN,
REFERENCED_BY, ALERTS_ON`

## Object types (directive §6)

`Person, Organisation, Facility, Project, Programme, Dataset, Indicator,
ResearchStudy, Budget, Transaction, Task, Incident, Service, Event, Document,
Policy, Risk, Objective`
