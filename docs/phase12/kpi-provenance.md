# KPI Provenance Model
## StatGate Phase XII — Sprint 0 Architecture Validation
**Document:** `docs/phase12/kpi-provenance.md`
**Status:** APPROVED

---

## Principle

> The Command Centre must be able to answer: **Where did this number come from?**
> If the answer cannot be traced to an authoritative source, the KPI must be marked accordingly.

---

## KPI Provenance Record

Every KPI measurement in Phase XII carries a full provenance record:

| Field | Required | Description |
|---|---|---|
| `kpi_canonical_id` | Yes | UOI canonical ID of the KPI definition |
| `kpi_name` | Yes | Human-readable name |
| `source_system` | Yes | Originating application or system |
| `source_dataset_canonical_id` | Recommended | UOI canonical ID of the source dataset |
| `source_indicator_canonical_id` | Recommended | UOI canonical ID of the source indicator |
| `calculation_method` | Yes | Formula or description of how value was computed |
| `measurement_timestamp` | Yes | When the measurement was taken at source |
| `ingested_timestamp` | Yes | When Phase XII received the measurement |
| `target` | Yes | KPI target value |
| `unit` | Yes | Unit of measure (ratio, count, days, %, etc.) |
| `measurement_period` | Yes | Period this measurement covers |
| `owner_canonical_id` | Yes | UOI canonical ID of owning organization/person |
| `freshness` | Yes | `FRESH` / `AGING` / `STALE` / `EXPIRED` |
| `quality_status` | Yes | `VERIFIED` / `UNVERIFIED` / `SUSPECTED_ERROR` / `MANUAL` |
| `entry_method` | Yes | `AUTOMATED` / `MANUAL` |
| `entered_by` | If manual | Actor canonical ID |
| `audit_reference` | Yes | Audit log entry for this measurement receipt |

---

## Provenance Drill Path

A user must be able to drill from displayed value back to source:

```
KPI Value Displayed in Command Centre
        │
        ▼
KPI Measurement Record (Phase XII)
  - value, unit, period, quality_status
        │
        ▼
Source Dataset (via canonical_id → UOI → StatCollect)
  - Dataset name, collection method, fieldwork period
        │
        ▼
Source Indicator (via canonical_id → Knowledge Layer)
  - Calculation formula, base population
        │
        ▼
Source System (StatCollect / StatGovernance / etc.)
  - Authoritative system of record
```

---

## Manual KPI Entry Rules

Manual entry is only permitted when ALL of the following are true:

1. No authoritative machine-readable source exists
2. KPI owner is identified and recorded
3. Manual measurement is auditable (creates audit log entry)
4. Provenance is recorded (entry method = MANUAL, entered_by is set)
5. Quality status = `MANUAL` (never `VERIFIED` for manual entries without review)
6. Freshness degrades at the same rate as automated KPIs

Manual KPIs are visually distinguished in the Command Centre with a `MANUAL ENTRY` badge.

---

## Quality Status Definitions

| Status | Meaning |
|---|---|
| `VERIFIED` | Source system confirmed measurement through validation rules |
| `UNVERIFIED` | Received from source but validation not yet run |
| `SUSPECTED_ERROR` | Measurement outside historical range — flagged for review |
| `MANUAL` | Entered by human actor — not from machine-readable source |
| `INTERPOLATED` | Gap in series filled by interpolation — not measured |

---

## KPI Data Entry Rule

The Command Centre is NOT the default KPI data entry application. The preferred data flow is:

```
Authoritative Source
       ↓
Event Bus / Data Fabric (Phase IX)
       ↓
`kpi.measurement.updated` event
       ↓
Phase XII KPI Projection
       ↓
Intelligence Layer → Condition Calculator
```

Manual entry through the Command Centre is the **fallback of last resort**.

---

*Status: APPROVED*
*Sprint 0 Gate: KPI PROVENANCE — APPROVED*
