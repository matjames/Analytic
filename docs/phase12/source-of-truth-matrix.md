# Source-of-Truth Ownership Matrix
## StatGate Phase XII — Architecture Validation
**Document:** `docs/phase12/source-of-truth-matrix.md`
**Sprint:** 0 — Architecture Validation
**Status:** APPROVED

---

## Core Principle

> **Source systems own facts. The Institutional Intelligence Layer owns correlation, interpretation, derived intelligence, and institutional condition — not the underlying source records.**

Phase XII's `institutional_objects` table is an **intelligence projection/index** of UOI-canonical objects. It does not replace or compete with the authoritative source. If the source record changes, the projection is updated via Event Bus — the intelligence layer never writes back to the source.

---

## Ownership Matrix

| Domain | Object | Authoritative Source | Phase XII Role | Notes |
|---|---|---|---|---|
| **Identity** | Users, Staff | Identity / Registry | Consume via Event Bus | Never replicate auth records |
| **Identity** | Roles & Permissions | Identity / Registry | Consume | Phase XII may not elevate permissions |
| **Platform** | Microservices | Registry | Consume | Service health owned by Phase XI |
| **Platform** | Incidents | Phase XI Engine | Consume + correlate to objectives/risks | Phase XI is the incident state machine |
| **Platform** | Resilience Profiles | Phase XI Engine | Consume | Phase XII observes; never modifies |
| **Platform** | Evidence Records | Phase XI Evidence Vault | Consume | Evidence is immutable; Phase XII links, never modifies |
| **Property** | Facilities | PMS (Property Management System) | Consume via Event Bus | PMS owns building/station lifecycle |
| **Property** | Leases / Assets | PMS | Consume | |
| **Field Data** | Field Submissions | StatCollect | Consume | StatCollect is the ingestion authority |
| **Field Data** | Survey Datasets | StatCollect | Consume + derive KPI inputs | Dataset quality assessed by Phase XII |
| **Communications** | Messages | StatChat | Consume (metadata only) | Message content must not flow to intelligence layer without policy |
| **HelpDesk** | Tickets, Issues | HelpDesk | Consume | Resolution metrics derived |
| **Governance** | Policies | StatGovernance | Consume | Phase XII observes compliance |
| **Governance** | Workflows | StatGovernance | Consume | Workflow completion rates derived |
| **Governance** | Risks | StatGovernance | Consume + project (see risk-intelligence-boundary.md) | StatGovernance is source of record |
| **Governance** | Objectives | StatGovernance | Consume + project | StatGovernance owns strategic objectives |
| **Records** | Documents | RMS | Consume (metadata/lineage only) | RMS owns document lifecycle |
| **Finance** | Transactions | Finance source system | Consume | **No finance source currently exists — see gap note** |
| **Finance** | Budgets | Finance source system | Consume | **No finance source currently exists — see gap note** |
| **Research** | Studies, Publications | Research source system | Consume | **No research source currently exists — see gap note** |
| **Intelligence** | Intelligence Signals | Phase XII | **OWN** | Not derived from a source system |
| **Intelligence** | Institutional Condition | Phase XII | **OWN** | Computed aggregate — never from a single source |
| **Intelligence** | AI Recommendations | Phase XII | **OWN** | AI output, always labelled AI_GENERATED |
| **Intelligence** | Knowledge Graph Edges (Derived) | Phase XII | **OWN** | Relationship projections derived from event data |
| **Intelligence** | Knowledge Graph Edges (Authoritative) | Source system via Event Bus | Consume + project | Sourced from application events |
| **Intelligence** | KPI Measurements (Manual) | Command Centre (fallback only) | **OWN with caveat** | Only permitted when no machine-readable source exists — must be auditable |
| **Intelligence** | KPI Measurements (Automated) | Source system (StatCollect / StatGovernance) | Derive | Preferred path |
| **Intelligence** | Data Lineage Events | Phase IX Data Fabric + Phase XII processor | Consume + extend | Phase XII adds intelligence lineage on top of data lineage |

---

## Documented Gaps

The following authoritative sources do not currently exist in StatGate. Phase XII must NOT silently create these systems. If a source does not exist, the domain condition must be reported as `UNKNOWN` rather than assumed healthy.

| Gap | Missing Source | Phase XII Response | Recommended Action |
|---|---|---|---|
| Financial data | No Finance application exists | Condition = `UNKNOWN — data unavailable` | Flag for future phase |
| Research management | No Research application exists | Condition = `UNKNOWN — data unavailable` | Flag for future phase |
| HR/People data | No HR application (beyond Registry identity) | Limited to identity metadata | Flag for future phase |

---

## Intelligence Projection Schema (Not a competing identity system)

The `institutional_objects` table is a projection index, not a master record:

```sql
CREATE TABLE institutional_objects (
    id              TEXT PRIMARY KEY,           -- intelligence layer internal ID
    tenant_id       TEXT NOT NULL,
    canonical_id    TEXT NOT NULL,              -- UOI canonical ID (authoritative)
    source_app      TEXT NOT NULL,              -- originating application
    object_type     TEXT NOT NULL,
    display_name    TEXT,
    display_metadata JSONB,
    projection_status TEXT DEFAULT 'ACTIVE',   -- ACTIVE, STALE, DELETED
    last_synced_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, canonical_id)            -- one projection per canonical object
);
```

The `canonical_id` is always the UOI canonical ID. The intelligence layer never creates a new canonical ID — it only indexes existing ones.

---

## Authoritative Resolution Rule

When Phase XII needs to resolve an object:

1. Look up `institutional_objects.canonical_id`
2. The canonical identity is owned by UOI — Phase XII queries UOI to resolve current state
3. If UOI is unavailable, Phase XII serves the cached projection with a `STALE` indicator
4. Phase XII never fabricates institutional identity

---

*Reviewed by: Sprint 0 Architecture Validation*
*Status: APPROVED*
