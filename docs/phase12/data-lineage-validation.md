# Data Lineage Validation
## StatGate Phase XII — Sprint 0 Architecture Validation
**Document:** `docs/phase12/data-lineage-validation.md`
**Status:** APPROVED

---

## Required Drill Path

A decision-maker must be able to drill from an AI recommendation all the way back to the source system:

```
AI Recommendation
        │  "Investigate Facility X reporting degradation"
        ▼
Institutional Condition Factor
        │  "Data Quality: ATTENTION — reporting completeness below threshold"
        ▼
KPI Measurement
        │  "Reporting Completeness: 62% (target 95%, period 2026-Q3)"
        ▼
Source Dataset
        │  "StatCollect Dataset: Monthly Field Submission Report — July 2026"
        │  canonical_id: uoi_statcollect_dataset_042
        ▼
Source System
        │  "StatCollect — authoritative ingestion system"
        │  Data collected via: field submission forms, district coordinators
        ▼
Raw Collection Event
        │  "data.lineage.event emitted by Phase IX Data Fabric on 2026-07-31"
```

At every step, provenance must be resolvable via canonical ID. No step may be `UNKNOWN` without explicit labelling.

---

## Lineage Record Structure

Phase XII extends Phase IX data lineage with intelligence-layer provenance:

```
data.lineage.event (Phase IX)
        │
        ▼
intelligence.lineage.step (Phase XII — new)
  - step_id
  - tenant_id
  - input_canonical_id      (source dataset/indicator)
  - transformation_type     (AGGREGATE / DERIVE / INFER / AI_GENERATE)
  - output_canonical_id     (resulting KPI / signal / recommendation)
  - method_description
  - actor                   (system / AI provider name)
  - ai_generated            (boolean)
  - created_at
```

---

## Validation Proof (Sprint 0)

The following lineage chain is demonstrated in the knowledge graph prototype:

| Step | Input | Transformation | Output | Provenance Type |
|---|---|---|---|---|
| 1 | StatCollect Dataset `uoi_ds_042` | INGEST | Raw field submission records | AUTHORITATIVE |
| 2 | Raw submissions | AGGREGATE (submitted/expected) | Reporting Completeness ratio | DERIVED |
| 3 | Reporting Completeness | MEASURE | KPI `uoi_kpi_reporting_completeness` | DERIVED |
| 4 | KPI measurement | EVALUATE | Data Quality signal: ATTENTION | DERIVED |
| 5 | Data Quality signal | CONDITION_CALC | Institutional Condition: ATTENTION | DERIVED |
| 6 | Condition + KPI evidence | AI_ANALYZE | Recommendation: investigate Facility X | INFERRED (AI_GENERATED) |

---

## Provenance Type Definitions (Graph Edges)

| Provenance Type | Meaning |
|---|---|
| `AUTHORITATIVE` | Created by source system via Event Bus — institutional fact |
| `DERIVED` | Computed by Phase XII intelligence layer from authoritative inputs |
| `INFERRED` | Produced by analytics/rules — probabilistic, not verified |
| `AI_GENERATED` | Produced by an AI model — always requires supporting evidence refs |

AI-generated lineage nodes are never indistinguishable from authoritative facts. The `ai_generated = true` flag and `provenance_type = AI_GENERATED` are always set.

---

*Status: APPROVED*
*Sprint 0 Gate: LINEAGE PROOF — PASS*
