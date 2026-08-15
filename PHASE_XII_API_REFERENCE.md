# PHASE_XII_API_REFERENCE.md
# StatGate Phase XII — API Reference

All endpoints live inside the existing JWT + tenant-isolation middleware group.
Identity is always derived from the verified JWT context — never from
`X-User-ID` / `X-Tenant-ID` headers. Every response is tenant-scoped. Privileged
actions require `admin` or `institutional_lead`.

Base URL: `https://<enterprise-core>/api`

## Intelligence

| Method | Path | Purpose | Auth |
|---|---|---|---|
| GET | `/intelligence/overview` | decision snapshot (condition, signals, risk, AI, metrics) | any authenticated |
| GET | `/intelligence/condition` | current explainable condition | any authenticated |
| GET | `/intelligence/condition/history` | condition calculation history | any authenticated |
| GET | `/intelligence/signals` | active + superseded signals | any authenticated |
| POST | `/intelligence/signals` | ingest a signal (triggers deterministic recalculation) | admin / institutional_lead |

## Graph (named queries only — §27)

| Method | Path | Purpose | Auth |
|---|---|---|---|
| GET | `/graph/objects` | list registry objects (`?object_type=`) | any authenticated |
| POST | `/graph/objects` | project a UOI-canonical object | admin / institutional_lead |
| GET | `/graph/edges` | list active edges | any authenticated |
| POST | `/graph/edges` | create an edge | admin / institutional_lead |
| DELETE | `/graph/edges/:id` | retire an edge (append/correction) | admin / institutional_lead |
| GET | `/graph/queries/objectives-at-risk` | named query | any authenticated |
| GET | `/graph/queries/:name/:id` | named, parameterized query | any authenticated |

Approved named queries: `projects-affected-by-incident`,
`datasets-supporting-kpi`, `objectives-at-risk`, `object-neighborhood`. Any other
query name is rejected (fail closed).

## Objectives & KPI

| Method | Path | Purpose | Auth |
|---|---|---|---|
| GET | `/objectives` | list objective projection | any authenticated |
| POST | `/objectives` | create objective | admin / institutional_lead |
| GET | `/kpis` | list KPIs (with data_status) | any authenticated |
| POST | `/kpis` | create KPI | admin / institutional_lead |
| GET | `/kpis/:id/measurements` | measurement history | any authenticated |
| POST | `/kpis/:id/measurements` | record a measurement | admin / institutional_lead |

`kpi.data_status` distinguishes `ACTUAL` / `ESTIMATED` / `STALE` / `MISSING` /
`INVALID`; stale is never presented as current.

## Risk

| Method | Path | Purpose | Auth |
|---|---|---|---|
| GET | `/risks/events` | list risk correlation events | any authenticated |
| POST | `/risks/events` | record a correlation event (not a register) | admin / institutional_lead |
| POST | `/risks/detect` | run emerging-risk detection (produces a signal) | admin / institutional_lead |

## AI (governed)

| Method | Path | Purpose | Auth |
|---|---|---|---|
| POST | `/ai/recommendations` | generate a recommendation (enters PENDING_REVIEW) | any authenticated |
| GET | `/ai/recommendations` | list recommendations | any authenticated |
| POST | `/ai/recommendations/:id/review` | authorize or reject | admin / institutional_lead |
| POST | `/ai/recommendations/:id/execute` | execute an AUTHORIZED recommendation (human) | admin / institutional_lead |
| GET | `/ai/audit` | dedicated AI audit trail | any authenticated |

AI lifecycle: `GENERATED → PENDING_REVIEW → AUTHORIZED|REJECTED → EXECUTED|CANCELLED`.
No AI recommendation skips human review. Output is always labelled `AI_GENERATED`.

## Error handling
Structured errors use `{"error": "<code>", "message": "..."}`. Common codes:
`tenant_context_required`, `query_not_approved`, `edge_rejected`,
`recommendation_failed`, `forbidden`. Unauthenticated/invalid JWT is denied by
the middleware before any handler runs.
