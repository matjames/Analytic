# AI Governance Model
## StatGate Phase XII — Sprint 0 Architecture Validation
**Document:** `docs/phase12/ai-governance.md`
**Status:** APPROVED

---

## Foundational Principle

> AI may observe, correlate, reason, explain, forecast and recommend. It must not silently become the owner of institutional facts, and it must never transition from RECOMMENDATION to EXECUTION without explicit human authorization.

---

## AI Recommendation Lifecycle

```
AI ANALYSIS (read-only inputs)
        │
        ▼
RECOMMENDATION CREATED
  - recommendation_id
  - category
  - summary
  - supporting_evidence (canonical IDs)
  - confidence (HIGH/MEDIUM/LOW)
  - classification (INTERNAL/CONFIDENTIAL)
  - ai_generated = true
        │
        ▼
HUMAN REVIEW (Command Centre — AIRecommendationsView)
        │
        ├─ AUTHORIZED → EXECUTION → VERIFICATION → AUDIT + EVIDENCE
        │                  (human executes or delegates to governed automation)
        └─ REJECTED  → CLOSED (with reason) → AUDIT
```

**The AI must NEVER transition directly:**
```
RECOMMENDATION → EXECUTION
```
without an `AUTHORIZED` status set by a human actor with appropriate role.

---

## Output Classification

All AI outputs must be classified by type before storage or display:

| Type | Meaning | Example |
|---|---|---|
| `FACT` | Directly derived from a verifiable institutional record | "Reporting completeness was 62% in July 2026 (source: StatCollect dataset uoi_ds_042)" |
| `INFERENCE` | Conclusion drawn from multiple facts | "The three concurrent outages are likely related" |
| `RECOMMENDATION` | Suggested action — not a decision | "Investigate Facility X reporting degradation" |
| `PREDICTION` | Probabilistic future state | "Risk escalation likelihood 73% over next 30 days" |
| `UNKNOWN` | AI output for which classification could not be determined | Flagged for human review before use |

AI outputs labelled `UNKNOWN` must not be displayed directly to decision-makers without review.

---

## Hallucination Controls

1. All AI recommendations must reference `supporting_evidence` (list of canonical IDs).
2. Recommendations with zero supporting evidence are rejected before storage.
3. AI outputs that contradict a current verifiable institutional fact are flagged as `SUSPECTED_HALLUCINATION` and surfaced for review.
4. The AI layer must use the `AIResultBase.Classification` field — never present output as `FACT` if it is an `INFERENCE` or `PREDICTION`.

---

## Prohibited AI Autonomous Actions

The following actions require explicit human authorization — AI may recommend but never execute:

| Action | Reason |
|---|---|
| Modify institutional records | Requires human authorization |
| Execute recovery drills | Requires operator authorization (Phase XI) |
| Change security policies | Requires governance approval |
| Alter financial records | Requires financial authority |
| Change permissions or roles | Requires identity governance |
| Delete or modify evidence | Evidence is immutable |
| Override governance controls | Governance controls are inviolable |
| Modify resilience profiles | Requires platform engineering sign-off |
| Register or close risks in StatGovernance | StatGovernance owns the risk register |

---

## AI Audit Trail

Every AI operation produces a record in `ai_audit_log`:
- Separate table from the platform audit log
- Linked via `correlation_id`
- Includes: provider, model, operation, input classification, tenant, actor, AI output classification, latency, token counts, success/error
- All records are immutable — no UPDATE or DELETE permitted
- Records are retained for the same period as the platform audit log

---

*Status: APPROVED*
*Sprint 0 Gate: AI GOVERNANCE — APPROVED*
