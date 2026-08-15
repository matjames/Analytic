# Risk Intelligence Boundary
## StatGate Phase XII — Sprint 0 Architecture Validation
**Document:** `docs/phase12/risk-intelligence-boundary.md`
**Status:** APPROVED

---

## Ruling

Phase XII does NOT create a competing risk-management system. StatGovernance is the authoritative source of the institutional risk register. Phase XII is a consumer, projector, and correlator.

---

## Authority Boundary

```
StatGovernance (source of record)
        │  ← risk.changed events via Event Bus
        ▼
Phase XII Risk Intelligence Projection
  [project + correlate + detect emerging patterns]
        │
        ▼
Institutional Condition Calculator
        │
        ▼
Command Centre Risk Intelligence View
        │
        ▼
AI Recommendation (read-only analysis)
```

Phase XII must NEVER:
- Modify the risk status in StatGovernance
- Create a new authoritative risk record without routing through StatGovernance
- Delete or close a risk in the source system
- Override risk severity set by StatGovernance

---

## What Phase XII Owns in the Risk Domain

| Capability | Phase XII Role | Notes |
|---|---|---|
| Risk projection | Own | A copy of risk data for intelligence purposes — linked to canonical_id |
| Risk correlation to incidents | Own | Detect: this incident is associated with this risk |
| Risk correlation to objectives | Own | Detect: this risk threatens this objective |
| Emerging risk signal detection | Own | Pattern-detected signal — must be labelled INFERRED |
| Risk escalation forecast | Own | AI-generated — must be labelled AI_GENERATED |
| Risk status update | NOT OWNED | Route request to StatGovernance via governed API |
| Risk creation | NOT OWNED | Route to StatGovernance |
| Risk closure | NOT OWNED | Route to StatGovernance |

---

## Emerging Risk Signal

When Phase XII detects a pattern that may constitute a risk not yet registered in StatGovernance, it creates an `EMERGING_RISK_SIGNAL` (not a risk record):

```json
{
  "signal_type": "EMERGING_RISK",
  "description": "Three consecutive StatCollect incidents correlated with declining reporting completeness",
  "confidence": "MEDIUM",
  "evidence": ["inc_001", "inc_002", "inc_003", "kpi_reporting_completeness"],
  "provenance_type": "INFERRED",
  "ai_generated": true,
  "recommended_action": "Consider registering operational reliability risk in StatGovernance"
}
```

This signal is surfaced in the Command Centre as a recommendation — not as a registered risk. A human must decide whether to create a formal risk in StatGovernance.

---

*Status: APPROVED*
*Sprint 0 Gate: RISK INTELLIGENCE BOUNDARY — APPROVED*
