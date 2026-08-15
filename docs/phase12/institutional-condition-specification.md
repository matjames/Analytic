# Institutional Condition Specification
## StatGate Phase XII — Sprint 0 Architecture Validation
**Document:** `docs/phase12/institutional-condition-specification.md`
**Status:** APPROVED

---

## 1. Purpose

The Institutional Condition is a governed, computed, multi-domain signal that answers:
> **Is the institution operating effectively, safely, efficiently, and according to its strategic objectives?**

It is **not** manually declared. It is computed from observable domain signals. An administrator can always drill from the condition level down to the evidence that produced it.

---

## 2. Condition Levels

| Level | Code | Meaning |
|---|---|---|
| Optimal | `OPTIMAL` | All domains operating within strategic targets |
| Nominal | `NOMINAL` | Minor deviations within acceptable tolerances |
| Attention | `ATTENTION` | One or more domains approaching risk thresholds |
| Elevated | `ELEVATED` | Active risk with measurable institutional impact |
| Critical | `CRITICAL` | Strategic objectives at material risk |
| Emergency | `EMERGENCY` | Institutional continuity threatened |

---

## 3. Domain Signals

The Condition Calculator aggregates the following domain signals. Each domain signal has its own level (`OPTIMAL` → `EMERGENCY`), weight, and staleness model.

| Domain | Weight | Source | Staleness Threshold |
|---|---|---|---|
| Strategic Performance | 25% | StatGovernance Objectives + KPIs | 24h |
| Operational Health | 20% | Phase XI Resilience Engine | 5min |
| Data Quality | 20% | StatCollect + Phase IX Data Fabric | 6h |
| Institutional Risk | 15% | StatGovernance Risk Register | 12h |
| Security Posture | 10% | Phase X/XI Security Events | 30min |
| Financial Health | 5% | Finance source (**CURRENTLY UNKNOWN**) | 24h |
| Research Activity | 5% | Research source (**CURRENTLY UNKNOWN**) | 48h |

> **Missing data rule:** If a domain signal is `UNAVAILABLE` or `UNKNOWN`, its weight is redistributed proportionally across available domains AND a domain-unavailable flag is surfaced. **A missing domain never silently contributes a healthy score.**

---

## 4. Data Freshness Model

Every domain signal carries a freshness status:

| Freshness | Meaning | Age |
|---|---|---|
| `FRESH` | Within normal refresh window | < staleness threshold |
| `AGING` | Approaching staleness | 80–100% of threshold |
| `STALE` | Past staleness threshold — signal weight reduced by 50% | > threshold |
| `EXPIRED` | Data is so old it must not contribute to condition | > 3× threshold |
| `UNAVAILABLE` | Domain is unreachable / no data source configured | — |
| `UNKNOWN` | Source exists but has not yet provided data | — |

Stale signals reduce the `confidence` of the overall condition. An `EXPIRED` signal is treated as `UNKNOWN`.

---

## 5. Condition Calculation Algorithm

```
FOR each domain_signal IN active_signals:
    IF signal.freshness == EXPIRED OR signal.freshness == UNAVAILABLE:
        SKIP signal; redistribute weight
    ELSE IF signal.freshness == STALE:
        effective_weight = signal.weight * 0.5
    ELSE:
        effective_weight = signal.weight

    domain_score = map(signal.level, OPTIMAL=100, NOMINAL=85, ATTENTION=65, ELEVATED=40, CRITICAL=15, EMERGENCY=0)
    weighted_sum += domain_score * effective_weight
    total_weight += effective_weight

composite_score = weighted_sum / total_weight
confidence = total_weight / sum(all weights)  -- 1.0 if all signals fresh

condition_level = map_score_to_level(composite_score, confidence)
```

### Score → Level Mapping

| Composite Score | Condition Level |
|---|---|
| 90 – 100 | `OPTIMAL` |
| 75 – 89 | `NOMINAL` |
| 55 – 74 | `ATTENTION` |
| 35 – 54 | `ELEVATED` |
| 15 – 34 | `CRITICAL` |
| 0 – 14 | `EMERGENCY` |

**Confidence adjustment:** If `confidence < 0.60`, the condition level is capped at `ATTENTION` regardless of composite score, and a `LOW_CONFIDENCE` flag is set. This prevents a system with mostly unknown signals from appearing `OPTIMAL`.

---

## 6. Escalation Behaviour

| Trigger | Action |
|---|---|
| Condition changes level | Emit `institutional.condition.changed` event |
| Condition reaches `CRITICAL` | Alert platform administrators via notification |
| Condition reaches `EMERGENCY` | Alert all registered institutional leads |
| Domain signal goes `STALE` | Log warning; reduce confidence; flag in UI |
| Domain signal goes `EXPIRED` | Log warning; exclude from calculation; flag in UI |

---

## 7. Condition Explainability Requirements

The Command Centre must never show only `CRITICAL`. Every condition display includes:

```
Current Level:       CRITICAL
Previous Level:      ELEVATED
Change:              Downgrade
Confidence:          0.72
Data Freshness:      AGING

Top Contributing Factors:
  1. Strategic Performance: CRITICAL (weight 25%) — 3 KPIs below target
  2. Operational Health:    ELEVATED  (weight 20%) — StatCollect RECOVERING incident
  3. Data Quality:          ATTENTION (weight 20%) — Reporting completeness 62% vs 95% target

Unavailable Domains:
  - Financial Health: UNKNOWN — no data in 36 hours
  - Research Activity: UNKNOWN — no data source configured

Affected Objectives:   3 objectives at risk
Active Risks:          2 elevated, 1 critical
Related Incidents:     1 active (StatCollect — RECOVERING)
```

---

## 8. Condition History

Every computed condition is persisted in `institutional_conditions` (time-series). The history is queryable by:
- Time range
- Level
- Tenant

The condition is never retroactively modified. If a recalculation finds a different result for a past moment, a new record is appended with a `RECALCULATED` flag.

---

*Status: APPROVED*
*Sprint 0 Gate: CONDITION SPECIFICATION — APPROVED*
