# AI Data Classification
## StatGate Phase XII — Sprint 0 Architecture Validation
**Document:** `docs/phase12/ai-data-classification.md`
**Status:** APPROVED

---

## Classification Levels

| Level | Code | Description | External AI Permitted |
|---|---|---|---|
| Public | `PUBLIC` | Non-sensitive institutional information | Yes |
| Internal | `INTERNAL` | Normal operational data; limited distribution | Yes (with logging) |
| Confidential | `CONFIDENTIAL` | Sensitive operational/financial; restricted access | Yes — if provider approved |
| Restricted | `RESTRICTED` | Personally identifiable, legally sensitive | No — private provider only |
| Sensitive | `SENSITIVE` | Security, auth, evidence, investigation data | No — blocked |

---

## Classification Assignment

Every institutional object and event is assigned a data classification at creation time. The classification propagates to AI requests through the `AIRequestBase.DataClassification` field.

Default classifications:
- Platform metadata (service names, incident summaries): `INTERNAL`
- KPI values and condition signals: `INTERNAL`
- Individual person records: `RESTRICTED`
- Financial transaction details: `CONFIDENTIAL`
- Security audit records: `SENSITIVE`
- Evidence vault records: `SENSITIVE`
- Public statistical outputs (aggregated, non-PII): `PUBLIC`

---

## AI Request Gate

Before any AI provider call:

```
1. Determine data classification of all input signals
2. Take the highest (most restrictive) classification
3. Check provider max_classification config
4. If input classification > provider max_classification → BLOCK (log + audit)
5. If input classification ≤ provider max_classification → PERMIT (log + audit)
```

Blocked requests return a structured error and create an audit record. They do NOT silently fail.

---

## Prompt Safety Rules

- AI prompts must never include raw personal identifiers (names, ID numbers, phone numbers) unless the provider is approved for `RESTRICTED` data.
- AI prompts must use canonical IDs to reference objects — the provider receives identifiers, not raw PII values.
- AI outputs must be inspected for potential PII before being stored or displayed.
- Prompts and completions are always logged to the AI audit log (separate from the platform audit log, linked via correlation ID).

---

## Tenant Isolation in AI Calls

- AI requests are tenant-scoped. Data from Tenant A must never appear in a prompt for Tenant B.
- Multi-tenant AI batch calls are prohibited. Each request is strictly single-tenant.
- The `tenant_id` is included in the AI audit log for every call.

---

*Status: APPROVED*
*Sprint 0 Gate: AI DATA CLASSIFICATION — APPROVED*
