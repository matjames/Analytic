# Intelligence Event Contracts
## StatGate Phase XII — Sprint 0 Architecture Validation
**Document:** `docs/phase12/intelligence-event-contracts.md`
**Status:** APPROVED

---

## 1. Principles

- Every event is versioned. The Intelligence Processor must safely ignore events from unknown future schema versions.
- Every event includes a `schema_version`. The processor checks this field first.
- Every event includes `tenant_id`. The processor enforces tenant isolation at the event level.
- Events are the source of truth for intelligence state reconstruction. The Intelligence Projection must be fully reconstructable from a replay of all past events.
- The Event Bus (Redis) infrastructure from Phase III/IX is reused. No new broker is introduced.

---

## 2. Common Event Envelope

All Phase XII intelligence events share this envelope:

```json
{
  "event_id": "evt_<nanoid>",
  "event_type": "institutional.object.created",
  "schema_version": "1.0",
  "tenant_id": "statgate",
  "canonical_object_id": "uoi_<canonical_id>",
  "source_application": "pms",
  "timestamp": "2026-08-15T13:00:00.000Z",
  "correlation_id": "corr_<id>",
  "request_id": "req_<id>",
  "actor": "system",
  "payload": { },
  "ai_generated": false
}
```

**Required fields:** `event_id`, `event_type`, `schema_version`, `tenant_id`, `timestamp`
**Recommended:** `canonical_object_id`, `source_application`, `correlation_id`, `actor`

---

## 3. Event Catalog

### 3.1 Institutional Object Events

#### `institutional.object.created`
Emitted when a source application creates an object that should be projected into the intelligence layer.

```json
{
  "event_type": "institutional.object.created",
  "payload": {
    "canonical_id": "uoi_pms_facility_001",
    "object_type": "Facility",
    "display_name": "National Statistics Office — Nairobi",
    "source_app": "pms",
    "source_id": "facility_001"
  }
}
```

#### `institutional.object.updated`
Emitted when a projected object's display metadata changes in the source.

```json
{
  "event_type": "institutional.object.updated",
  "payload": {
    "canonical_id": "uoi_pms_facility_001",
    "changed_fields": ["display_name"],
    "display_name": "NSO Nairobi Branch — Updated"
  }
}
```

#### `institutional.object.deleted`
Emitted when a source object is retired. Intelligence projection is marked `DELETED` — not physically removed.

---

### 3.2 Relationship Events

#### `institutional.relationship.created`

```json
{
  "event_type": "institutional.relationship.created",
  "payload": {
    "subject_canonical_id": "uoi_statcollect_dataset_042",
    "relationship_type": "SUPPORTS",
    "object_canonical_id": "uoi_gov_indicator_gdp_001",
    "provenance_type": "AUTHORITATIVE",
    "source_event_id": "evt_abc123",
    "confidence": 1.0
  }
}
```

#### `institutional.relationship.removed`

```json
{
  "event_type": "institutional.relationship.removed",
  "payload": {
    "edge_id": "edge_<id>",
    "reason": "Dataset retired"
  }
}
```

---

### 3.3 Data & Knowledge Events

#### `data.lineage.event`
Emitted by the Phase IX Data Fabric whenever a dataset transformation occurs.

```json
{
  "event_type": "data.lineage.event",
  "payload": {
    "source_canonical_id": "uoi_statcollect_dataset_042",
    "transformation": "AGGREGATE",
    "output_canonical_id": "uoi_kpi_reporting_completeness_q3",
    "method": "COUNT(submitted) / COUNT(expected)",
    "lineage_step": 2,
    "total_steps": 3
  }
}
```

---

### 3.4 KPI & Objective Events

#### `kpi.measurement.updated`

```json
{
  "event_type": "kpi.measurement.updated",
  "payload": {
    "kpi_canonical_id": "uoi_kpi_reporting_completeness",
    "value": 0.847,
    "unit": "ratio",
    "target": 0.95,
    "measurement_period": "2026-Q3",
    "source_system": "statcollect",
    "source_dataset_id": "uoi_statcollect_dataset_042",
    "quality_status": "VERIFIED",
    "freshness": "FRESH"
  }
}
```

#### `objective.changed`

```json
{
  "event_type": "objective.changed",
  "payload": {
    "objective_canonical_id": "uoi_gov_objective_001",
    "status": "AT_RISK",
    "previous_status": "ON_TRACK",
    "change_reason": "KPI reporting_completeness below target"
  }
}
```

---

### 3.5 Risk Events

#### `risk.changed`

```json
{
  "event_type": "risk.changed",
  "payload": {
    "risk_canonical_id": "uoi_gov_risk_001",
    "status": "ELEVATED",
    "previous_status": "MONITORED",
    "severity": "HIGH",
    "affected_objectives": ["uoi_gov_objective_001"],
    "change_reason": "Third consecutive quarterly miss"
  }
}
```

---

### 3.6 Incident Events (Consumed from Phase XI)

#### `incident.changed`
Phase XI emits these. Phase XII Intelligence Processor subscribes and correlates to objectives/risks.

```json
{
  "event_type": "incident.changed",
  "payload": {
    "incident_id": "inc_<id>",
    "service_id": "statcollect",
    "status": "RECOVERING",
    "severity": "CRITICAL",
    "affected_canonical_ids": ["uoi_statcollect_dataset_042"]
  }
}
```

---

### 3.7 Institutional Condition Events

#### `institutional.condition.changed`
Emitted by Phase XII condition calculator after recalculation.

```json
{
  "event_type": "institutional.condition.changed",
  "payload": {
    "condition_level": "ATTENTION",
    "previous_level": "NOMINAL",
    "top_factors": [
      { "domain": "data_quality", "signal": "reporting_completeness < 85%", "weight": 0.20 },
      { "domain": "operational", "signal": "StatCollect incident RECOVERING", "weight": 0.15 }
    ],
    "confidence": 0.87,
    "freshness": "FRESH",
    "calculated_at": "2026-08-15T13:05:00Z"
  }
}
```

---

### 3.8 AI Events

#### `ai.recommendation.created`

```json
{
  "event_type": "ai.recommendation.created",
  "ai_generated": true,
  "payload": {
    "recommendation_id": "rec_<id>",
    "category": "DATA_QUALITY",
    "summary": "Investigate Facility X reporting degradation",
    "evidence": [
      { "type": "KPI", "canonical_id": "uoi_kpi_reporting_completeness", "value": 0.62, "target": 0.95 }
    ],
    "confidence": "HIGH",
    "classification": "INTERNAL"
  }
}
```

#### `ai.recommendation.authorized`

```json
{
  "event_type": "ai.recommendation.authorized",
  "ai_generated": false,
  "actor": "admin_user_001",
  "payload": {
    "recommendation_id": "rec_<id>",
    "authorization_note": "Approved for StatCollect team investigation"
  }
}
```

#### `ai.recommendation.rejected`

```json
{
  "event_type": "ai.recommendation.rejected",
  "ai_generated": false,
  "actor": "admin_user_001",
  "payload": {
    "recommendation_id": "rec_<id>",
    "rejection_reason": "Already under investigation via HelpDesk ticket HLPx-1042"
  }
}
```

---

## 4. Schema Versioning Policy

- All events carry `schema_version` as a `"MAJOR.MINOR"` string.
- The processor handles events for which it knows the schema version.
- Events with an unknown MAJOR version are logged as `UNRECOGNIZED_SCHEMA` and skipped — never silently processed.
- Events with a newer MINOR version are processed using the known fields; unknown fields are ignored.
- Schema version history must be maintained in `docs/phase12/event-schema-changelog.md`.

---

## 5. Event Replay Requirement

The intelligence projection must be fully reconstructable from event replay:

```
REPLAY START
    │
    ▼
Process institutional.object.created events → Rebuild institutional_objects
    │
    ▼
Process institutional.relationship.created events → Rebuild institutional_graph_edges
    │
    ▼
Process kpi.measurement.updated events → Rebuild kpi_measurements
    │
    ▼
Process institutional.condition.changed events → Rebuild condition history
    │
    ▼
REPLAY COMPLETE → Projection state matches pre-deletion state
```

The intelligence layer must never create state that cannot be reconstructed from events.

---

*Status: APPROVED*
*Sprint 0 Gate: EVENT CONTRACTS — APPROVED*
