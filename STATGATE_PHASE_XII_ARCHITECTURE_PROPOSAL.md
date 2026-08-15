# STATGATE PHASE XII — ARCHITECTURE PROPOSAL
## Institutional Intelligence & Autonomous Governance

**Document Type:** Architecture Proposal — For Review Before Implementation
**Status:** PROPOSED — Awaiting Institutional Approval
**Predecessor:** Phase XI — Institutional Resilience, Continuity & Autonomous Assurance
**Prepared By:** StatGate Engineering — Antigravity Development Assistant
**Date:** 2026-08-15

---

## Review Instruction

> This document must be reviewed and approved before any substantial Phase XII implementation code is written. The proposal raises open architectural questions that require institutional direction. Sections marked **[DECISION REQUIRED]** need explicit sign-off before the corresponding work can begin.

---

## 1. Problem Statement

StatGate has evolved through eleven architectural phases. It now provides:

- Enterprise applications (PMS, RMS, StatCollect, StatChat, HelpDesk)
- Enterprise infrastructure (Core, Registry, Event Bus, Search, Analytics)
- Workflow and decision automation
- Institutional knowledge and AI-assisted investigations
- Institutional Command Centre
- Universal Object Identity across all applications
- Institutional Data & Knowledge Fabric
- Cross-application interoperability and data lineage
- Platform governance, auditability, and observability
- Production-grade security, identity, and tenant isolation
- Sovereign resilience assurance and autonomous incident recovery

Phase XI answers: **"Is the platform healthy?"**

Phase XII must answer a more demanding institutional question:

> **"Is the institution operating effectively, safely, efficiently, and according to its strategic objectives?"**

StatGate currently records *what happened* inside its own platform. Phase XII must enable it to understand *what is happening* across the institution — and what it *means*.

The gap is not more applications. The gap is an **institutional intelligence layer**: a governed, correlated, auditable model that connects operational data, research data, financial data, risk data, and strategic objectives into a unified institutional understanding.

---

## 2. Architectural Objectives

Phase XII shall:

1. Establish a **unified institutional intelligence model** — not another silo.
2. Connect institutional objects across all StatGate applications through a **Knowledge Graph** built on the existing Universal Object Identity (UOI) framework.
3. Enable **cross-domain correlation** — operations, data, finance, research, risk, and strategic performance.
4. Introduce a **governed AI reasoning layer** that can detect patterns, explain anomalies, correlate events, and recommend actions — without autonomously modifying institutional records.
5. Provide an **Institutional Intelligence Hub** in the Command Centre for decision-makers.
6. Preserve and extend all governance, audit, tenant isolation, and security foundations established in Phases I–XI.
7. Never duplicate existing infrastructure (Registry, Identity, Event Bus, PostgreSQL, Audit, Search, Metrics, UOI).

---

## 3. Institutional Intelligence Model

Phase XII introduces the concept of an **Institutional Condition** — a unified, cross-domain view of whether the institution is operating within expected parameters.

An Institutional Condition is computed from:

```
Strategic Objectives + KPI Performance
        +
Operational Health (Phase XI) + Incident History
        +
Data Quality + Reporting Completeness + Timeliness
        +
Financial Health + Budget Adherence + Procurement Status
        +
Research Activity + Dataset Availability + Publication Timeliness
        +
Risk Register + Compliance Status + Security Posture
```

This is NOT a new data store. It is a **correlation and reasoning layer** that draws from existing application data via the Event Bus and Data Fabric already established in Phase IX.

### Institutional Condition Levels

| Level | Meaning |
|---|---|
| `OPTIMAL` | All domains operating within strategic targets |
| `NOMINAL` | Minor deviations — within acceptable tolerances |
| `ATTENTION` | One or more domains approaching risk thresholds |
| `ELEVATED` | Active risk with measurable institutional impact |
| `CRITICAL` | Strategic objectives at material risk |
| `EMERGENCY` | Institutional continuity threatened |

These levels are automatically derived — not manually declared.

---

## 4. Canonical Object Relationships — Knowledge Graph Strategy

### 4.1 Canonical Object Types

Phase XII extends the existing UOI framework with the following institutional object types, each assigned a canonical StatGate Object Identity:

| Object Type | Source Domain | Examples |
|---|---|---|
| `Person` | HR / Identity | Staff, Researchers, Officers |
| `Organisation` | Registry | Divisions, Units, Departments |
| `Facility` | PMS | Buildings, Stations, Offices |
| `Project` | StatGovernance | Active projects, programmes |
| `Programme` | StatGovernance | Multi-project programmes |
| `Dataset` | StatCollect / Data Catalogue | Survey data, admin data |
| `Indicator` | Knowledge Layer | KPIs, statistical indicators |
| `ResearchStudy` | StatCollect | Surveys, censuses, studies |
| `Budget` | Finance domain | Budget allocations |
| `Transaction` | Finance domain | Expenditure records |
| `Task` | Workflow Engine | Operational tasks |
| `Incident` | Phase XI | Platform incidents |
| `Service` | Registry | StatGate microservices |
| `Event` | Event Bus | Cross-domain events |
| `Document` | RMS | Reports, policies, records |
| `Policy` | StatGovernance | Institutional policies |
| `Risk` | Risk Register | Identified institutional risks |
| `Objective` | Strategic Layer | Institutional objectives |

### 4.2 Relationship Types

The Knowledge Graph captures typed relationships between canonical objects:

| Relationship | Example |
|---|---|
| `SUPPORTS` | Dataset → Indicator |
| `CONTRIBUTES_TO` | Project → Objective |
| `AFFECTS` | Incident → Service |
| `DEPENDS_ON` | Service → Service |
| `PRODUCED_BY` | Dataset → ResearchStudy |
| `OWNED_BY` | Project → Organisation |
| `ASSIGNED_TO` | Task → Person |
| `FUNDED_BY` | Project → Budget |
| `GOVERNED_BY` | Dataset → Policy |
| `ASSOCIATED_WITH` | Risk → Objective |
| `OBSERVED_IN` | Anomaly → Dataset |
| `REFERENCED_BY` | Document → Policy |
| `ALERTS_ON` | Incident → Risk |

### 4.3 Example Institutional Queries Enabled

```
Q: Which projects are affected by this incident?
   MATCH (:Incident {id: X})-[:AFFECTS]->(:Service)<-[:DEPENDS_ON]-(:Project)

Q: Which facilities have persistent reporting problems?
   MATCH (:Facility)-[:PRODUCED_BY]->(:Dataset {quality_status: "DEGRADED"})

Q: Which datasets support this KPI?
   MATCH (:Dataset)-[:SUPPORTS]->(:Indicator {id: X})

Q: Which strategic objectives are currently at risk?
   MATCH (:Risk {status: "ELEVATED"})-[:ASSOCIATED_WITH]->(:Objective)

Q: Which operational events are associated with a performance decline?
   MATCH (:Event)-[:OBSERVED_IN]->(:Dataset)<-[:SUPPORTS]-(:Indicator {trend: "DECLINING"})

Q: Which risks are appearing across multiple applications?
   MATCH (:Risk)-[:ALERTS_ON]->(:Incident) WHERE count > threshold
```

### 4.4 Knowledge Graph Storage Strategy — [DECISION REQUIRED]

Three storage approaches are viable. A decision is required:

**Option A — PostgreSQL Adjacency List (Recommended for Phase XII)**
- Store relationships as `(subject_canonical_id, relationship_type, object_canonical_id, metadata, tenant_id)` in a new `institutional_graph_edges` table.
- Query using recursive CTEs for graph traversal.
- Pro: Reuses existing PostgreSQL infrastructure, no new technology.
- Con: Deep traversal queries may become expensive at scale.

**Option B — PostgreSQL with `pg_graphql` or `ltree`**
- Adds graph-query acceleration within PostgreSQL.
- Pro: Better hierarchical traversal, still single database.
- Con: Requires pg extension availability.

**Option C — Dedicated Graph Database (e.g., Apache AGE on top of PostgreSQL)**
- Full Cypher query support over PostgreSQL storage.
- Pro: Native graph queries, powerful traversal.
- Con: New infrastructure dependency, operational overhead.

> **Recommendation:** Begin with Option A (adjacency list). It is sufficient for Phase XII discovery queries and avoids introducing new infrastructure. Migrate to Option C only when traversal performance becomes a measurable constraint.

---

## 5. Data Lineage Approach

Phase XII extends the data lineage model established in Phase IX.

Every institutional object that flows through StatGate will have a lineage record describing:

```
Source system → Collection method → Processing steps → Transformations → Derived outputs → Consumers
```

Lineage is event-driven: whenever a dataset is processed, a derivation is computed, or a report is published, the Event Bus emits a `data.lineage.event` that the Lineage Recorder captures.

This approach requires no polling and adds no coupling between source systems.

---

## 6. Event-Driven Intelligence Architecture

Phase XII intelligence is **reactive, not polling-based**.

```
Application Events (Event Bus — Phase III/IX)
           │
           ▼
Intelligence Event Processor
  [classifies event type → updates knowledge graph → evaluates institutional condition]
           │
           ▼
Institutional Condition Calculator
  [aggregates domain signals → computes condition level → emits condition.changed event]
           │
           ▼
Command Centre Intelligence Hub
  [displays current condition → exposes drill-through → surfaces AI recommendations]
           │
           ▼
AI Reasoning Layer (governed — read-only)
  [detects patterns → correlates events → recommends actions → generates reports]
```

The AI layer receives read-only snapshots. It **never writes to institutional records directly**.

---

## 7. AI Governance Model

Phase XII introduces an explicit **AI Governance Layer** formalising the boundary between AI observation and AI action.

### Permitted AI Operations (No Authorization Required)

| Operation | Description |
|---|---|
| Anomaly detection | Identify statistical outliers in institutional metrics |
| Event correlation | Link related events across domains |
| Pattern recognition | Identify recurring risk patterns |
| Incident explanation | Summarize incident cause, impact, and timeline |
| Report generation | Generate institutional summary reports |
| Risk forecasting | Predict emerging risks based on historical patterns |
| Runbook recommendation | Suggest appropriate runbooks for an incident type |
| KPI trend analysis | Describe performance direction against targets |

### Prohibited AI Autonomous Actions

| Action | Reason |
|---|---|
| Modify institutional records | Requires human authorization |
| Execute recovery drills | Requires explicit operator authorization |
| Change security policies | Requires governance approval |
| Alter financial records | Requires financial authority |
| Change permissions or roles | Requires identity governance approval |
| Delete evidence records | Evidence is immutable |
| Override governance controls | Governance controls are inviolable |
| Modify resilience profiles | Requires platform engineering sign-off |

### AI Authorization Model

For future AI operations that *do* carry institutional consequence (e.g., automated remediation), a formal **AI Authorization Request** must be raised:

```
AI_RECOMMENDATION
       │
       ▼
HUMAN REVIEW (Command Centre)
       │
       ▼
AUTHORIZED / REJECTED (with reason)
       │
       ▼
EXECUTED (audited) / CANCELLED (audited)
       │
       ▼
EVIDENCE RECORDED
```

Every AI recommendation must be distinguishable in the audit trail from an executed human action.

---

## 8. Security Model

Phase XII inherits all Phase X security controls:

- JWT-based authentication with cryptographic signing
- Role-based authorization (`admin`, `analyst`, `viewer`, `institutional_lead`)
- Tenant isolation — all institutional objects scoped by `tenant_id`
- Rate limiting and request validation
- Audit logging for all write operations

**New Phase XII security considerations:**

- **Knowledge graph access** — Graph edge queries must respect tenant boundaries. Cross-tenant graph traversal is prohibited.
- **AI output labelling** — All AI-generated content must be marked as `AI_GENERATED` in the audit trail, distinguishing it from human-authored institutional records.
- **Intelligence data classification** — Institutional condition signals and AI recommendations should be classified by sensitivity level.
- **Graph mutation authorization** — Creating or modifying knowledge graph edges is a privileged operation requiring `admin` or `institutional_lead` role.

---

## 9. Tenant Isolation Model

All Phase XII objects — knowledge graph nodes, edges, intelligence signals, AI recommendations, institutional conditions — are scoped by `tenant_id`.

Cross-tenant graph traversal is blocked at the query layer (same pattern as Phases IX–XI). A multi-tenant institution may have logically separate knowledge graphs per division or organizational unit, federated at the institutional root level with appropriate access controls.

---

## 10. APIs

### Institutional Intelligence

```
GET  /api/intelligence/condition              — Current institutional condition
GET  /api/intelligence/condition/history      — Historical condition timeline
GET  /api/intelligence/domains                — Per-domain health breakdown
GET  /api/intelligence/signals                — Active intelligence signals
```

### Knowledge Graph

```
GET    /api/graph/objects/:id                  — Get object with relationships
GET    /api/graph/objects/:id/relations        — List direct relationships
POST   /api/graph/edges                        — Create relationship (admin)
DELETE /api/graph/edges/:id                    — Remove relationship (admin)
GET    /api/graph/query                        — Execute named graph query
```

### Institutional Objectives & KPIs

```
GET  /api/objectives                           — List institutional objectives
POST /api/objectives                           — Create objective (admin)
GET  /api/objectives/:id                       — Get objective + linked KPIs
GET  /api/objectives/:id/performance           — Current performance vs target
GET  /api/kpis                                 — List KPIs
GET  /api/kpis/:id/trend                       — Historical KPI trend
```

### Risk Intelligence

```
GET  /api/risks                                — Active risk register
POST /api/risks                                — Register risk (admin)
GET  /api/risks/:id                            — Risk + linked incidents/objectives
GET  /api/risks/emerging                       — AI-flagged emerging risks
```

### AI Governance

```
GET  /api/ai/recommendations                   — Pending AI recommendations
GET  /api/ai/recommendations/:id               — Detail + supporting evidence
POST /api/ai/recommendations/:id/authorize     — Authorize for execution (admin)
POST /api/ai/recommendations/:id/reject        — Reject with reason (admin)
GET  /api/ai/audit                             — AI action audit log
```

---

## 11. Command Centre Integration

Phase XII extends the Command Centre with an **Institutional Intelligence Hub** — a new navigation group alongside the existing Resilience & Assurance, Security, and Analytics groups:

| View | Purpose |
|---|---|
| `InstitutionalConditionView.tsx` | Overall institutional condition level, domain breakdown, trend |
| `KnowledgeGraphView.tsx` | Interactive relationship explorer for canonical objects |
| `ObjectivesKPIView.tsx` | Strategic objectives, KPI performance, target vs actual |
| `RiskIntelligenceView.tsx` | Active risks, emerging risk signals, correlation with incidents |
| `AIRecommendationsView.tsx` | Governed AI recommendation queue, authorize/reject workflow |
| `DataLineageView.tsx` | Dataset-to-report lineage explorer |

All views connect to live APIs. No mock data is presented as institutional information.

---

## 12. Storage Requirements

### New PostgreSQL Tables (Proposed)

| Table | Purpose |
|---|---|
| `institutional_objects` | Registry of all canonical institutional objects |
| `institutional_graph_edges` | Typed relationships between canonical objects |
| `institutional_conditions` | Historical institutional condition log |
| `intelligence_signals` | Domain-level intelligence signals feeding condition |
| `institutional_objectives` | Strategic objectives |
| `kpis` | Key performance indicators |
| `kpi_measurements` | Time-series KPI values |
| `risk_register` | Active institutional risks |
| `risk_events` | Risk state transitions |
| `ai_recommendations` | AI-generated recommendations + authorization status |
| `ai_audit_log` | Separate audit trail for AI actions and outputs |

All tables follow existing StatGate conventions: `tenant_id` scoping, `created_at`/`updated_at` timestamps, UOI-compatible canonical IDs, appropriate indexes.

### No New Infrastructure Required for Phase XII (Option A)

Under the recommended adjacency list approach:
- No new database technology
- No new message broker
- No new identity system
- No new audit system
- No new notification system
- No new search system

All existing infrastructure (PostgreSQL, Redis, Registry, Event Bus, Audit, Search, UOI) is reused as-is.

---

## 13. Scalability Considerations

| Concern | Mitigation |
|---|---|
| Graph traversal depth | Limit query depth to configurable maximum (default: 5 hops) |
| Large object registries | Index `tenant_id + object_type + created_at`; paginate all list APIs |
| AI recommendation volume | Queue recommendations; batch processing with configurable rate limits |
| Institutional condition frequency | Compute on event trigger, not polling; debounce rapid signal changes |
| KPI time series growth | Partition `kpi_measurements` by `(tenant_id, year_month)` |
| Intelligence signal fan-out | Use existing Event Bus; consumers apply backpressure naturally |

---

## 14. Failure Scenarios

| Scenario | Impact | Mitigation |
|---|---|---|
| Intelligence processor crash | Condition becomes stale | Event Bus re-queues; processor recovers on restart with replay |
| Knowledge graph corruption | Incorrect relationship traversal | Graph edges are immutable-append; correct by adding new edge, not modifying |
| AI recommendation service unavailable | No new AI output | Existing recommendations remain accessible; system degrades gracefully |
| KPI data source unavailable | KPI becomes stale | Last known value displayed with staleness indicator |
| Condition calculator overload | Delayed institutional signal | Queue backpressure protects; SLA: condition update within 60s of signal |

---

## 15. Audit Requirements

Every Phase XII write operation must produce an audit record including:

- `actor` — user or system identifier
- `action` — operation performed (e.g., `graph.edge.create`, `objective.update`, `ai.recommendation.authorize`)
- `resource` — canonical ID of affected object
- `tenant_id` — tenant scope
- `correlation_id` — request correlation chain
- `timestamp` — nanosecond precision
- `ai_generated` — boolean flag, distinguishing AI outputs from human actions

AI-generated content (recommendations, reports, risk signals) is always flagged separately. An AI recommendation that is authorized and executed creates *two* audit records: the AI recommendation record and the human authorization record.

---

## 16. Testing Strategy

| Layer | Approach |
|---|---|
| Knowledge graph CRUD | Unit tests for edge creation, traversal, tenant isolation |
| Condition calculator | Property-based tests against known domain signal combinations |
| AI governance authorization | Negative tests: AI cannot execute without `AUTHORIZED` status |
| Cross-tenant isolation | Explicit cross-tenant access tests (must return 0 results) |
| API authorization | Role-based tests for all new endpoints |
| Event-driven condition update | Integration tests triggering signals and asserting condition recalculation |
| Performance | Graph traversal benchmarks at depth 3, 5, 7 hops with realistic data volume |

Target: **100% pass rate across Phase XII test suite** before implementation is declared complete.

---

## 17. Migration Strategy

Phase XII introduces new tables only. No existing Phase I–XI tables are modified.

Migration file: `12-create-phase12-intelligence.sql`

Migration is additive and fully backward-compatible. Phase XI continues to operate without any schema changes.

---

## 18. Backward Compatibility Strategy

All existing Phase I–XI APIs remain unchanged. Phase XII adds new API namespaces (`/api/intelligence/`, `/api/graph/`, `/api/objectives/`, `/api/risks/`, `/api/ai/`) without modifying existing endpoints.

The existing Command Centre navigation is extended with a new group. Existing views are unchanged.

---

## 19. Proposed Implementation Phases

Phase XII is substantial and should be delivered in controlled sprints:

### Sprint 1 — Institutional Object Registry & Knowledge Graph Foundation
- Database migration (`12-create-phase12-intelligence.sql`)
- `institutional_objects` and `institutional_graph_edges` tables
- Object and edge CRUD APIs
- Tenant isolation validation

### Sprint 2 — Institutional Condition Engine
- `intelligence_signals` and `institutional_conditions` tables
- Intelligence Event Processor (Event Bus subscriber)
- Institutional Condition Calculator
- `/api/intelligence/condition` and `/api/intelligence/signals` APIs

### Sprint 3 — Strategic Objectives & KPI Framework
- `institutional_objectives`, `kpis`, `kpi_measurements` tables
- Objectives and KPI CRUD APIs
- KPI trend calculation
- Performance vs target comparison

### Sprint 4 — Risk Intelligence Layer
- `risk_register` and `risk_events` tables
- Risk registration, state management, correlation to incidents/objectives
- Emerging risk signal detection
- `/api/risks/` APIs

### Sprint 5 — Governed AI Recommendation Engine
- `ai_recommendations` and `ai_audit_log` tables
- AI output ingestion (read-only reasoning)
- Authorization workflow (human review required before execution)
- AI audit trail (separate from platform audit log, but linked)
- `/api/ai/` APIs

### Sprint 6 — Command Centre Intelligence Hub
- `InstitutionalConditionView.tsx`
- `KnowledgeGraphView.tsx`
- `ObjectivesKPIView.tsx`
- `RiskIntelligenceView.tsx`
- `AIRecommendationsView.tsx`
- `DataLineageView.tsx`

### Sprint 7 — Testing, Documentation & Phase XII Certification
- Full test suite (unit + integration + cross-tenant)
- Performance benchmarks
- Documentation
- Phase XII Operational Certification
- `STATGATE_PHASE_XII_COMPLETION.md`

---

## 20. Open Questions — [DECISION REQUIRED]

The following questions must be answered before implementation begins:

### Q1 — Knowledge Graph Storage
**Question:** Proceed with PostgreSQL adjacency list (Option A), or evaluate Apache AGE / ltree extension?
**Recommendation:** Option A for Phase XII; revisit at Phase XIII if traversal performance becomes measurable.

### Q2 — AI Provider
**Question:** Which AI model/provider will supply the institutional reasoning layer?
**Options:** Embedded local inference, Google Gemini API, OpenAI API, private hosted model.
**Constraint:** All AI inputs/outputs must flow through the audit log. The provider must support structured output for governance compliance.

### Q3 — KPI Data Source Integration
**Question:** Where does KPI measurement data originate? Is it entered manually through the Command Centre, or pulled automatically from StatCollect or other source systems?
**Impact:** Affects Event Bus integration design and data quality assurance model.

### Q4 — Risk Register Ownership
**Question:** Is the risk register owned by a specific application (e.g., StatGovernance), or is it a new standalone module in the institutional intelligence layer?
**Impact:** Determines whether Phase XII creates a new risk engine or extends an existing one.

### Q5 — Institutional Condition Sensitivity
**Question:** Should the institutional condition level be visible to all authenticated users, or restricted to `institutional_lead` / `admin` roles?
**Impact:** Authorization design for the Intelligence Hub.

### Q6 — Phasing with StatGovernance
**Question:** StatGovernance already has governance workflows. Should Phase XII defer risk and objective management to StatGovernance (via API), or implement a parallel institutional layer?
**Recommendation:** Phase XII should consume StatGovernance data via Event Bus/API rather than duplicate functionality. StatGovernance remains the source of record.

---

## Approval Required

This proposal is submitted for institutional review. Phase XII implementation will not begin until:

1. This proposal has been reviewed and approved.
2. All `[DECISION REQUIRED]` questions above have received explicit answers.
3. The Knowledge Graph storage strategy has been confirmed.
4. The AI provider and governance model have been confirmed.

**Phase XI Engineering Status:** ✅ `COMPLETE`
**Phase XI Operational Certification:** ✅ `CERTIFIED`
**Phase XII Architecture Status:** 🔵 `PROPOSED — PENDING REVIEW`

---

*Prepared by: StatGate Engineering — Antigravity Development Assistant*
*Date: 2026-08-15*
*For Review By: Platform Architecture & Institutional Governance*
