# Security Threat Model
## StatGate Phase XII — Sprint 0 Architecture Validation
**Document:** `docs/phase12/threat-model.md`
**Status:** REVIEWED

---

## Threat Surface Summary

Phase XII introduces: a knowledge graph, an intelligence condition engine, a KPI layer, a risk intelligence projection, an AI recommendation layer, and an AI audit trail. Each introduces new attack surface that must be addressed.

---

## Threat Catalog

### T01 — Graph Traversal Abuse
**Description:** Attacker attempts deep or unbounded graph traversal to enumerate all institutional objects and relationships.
**Mitigation:**
- Maximum traversal depth enforced server-side (configurable, default: 5 hops)
- All graph queries require authentication
- Paginated results (max 200 per page)
- Query timeout enforced (configurable, default: 10 seconds)
- Slow/expensive queries logged and rate-limited per user

---

### T02 — Tenant Escape via Graph Edge
**Description:** Tenant A creates a relationship edge pointing to Tenant B's canonical object, gaining read access to cross-tenant data.
**Mitigation:**
- All graph edge creation validates that `subject_canonical_id` and `object_canonical_id` belong to the same `tenant_id` as the actor
- Direct object access always filters by `tenant_id` — even if a cross-tenant edge somehow exists, the object fetch returns 404 for wrong tenant
- Cross-tenant graph traversal is blocked at the query layer using RLS or mandatory WHERE clause

---

### T03 — Prompt Injection
**Description:** Attacker embeds malicious instructions in institutional data that is subsequently sent to an AI provider, causing the AI to output misleading or harmful content.
**Mitigation:**
- AI prompts are structured templates with parameter substitution — institutional data is injected as data fields, not free-text instructions
- Prompt templates are version-controlled and reviewed
- AI outputs are classified and reviewed before storage; `UNKNOWN` classification triggers human review
- Input sanitization strips control characters before inclusion in prompts

---

### T04 — Data Exfiltration Through AI
**Description:** Attacker crafts AI analysis requests designed to extract RESTRICTED or SENSITIVE institutional data via the AI provider's response.
**Mitigation:**
- Data classification gate blocks RESTRICTED/SENSITIVE data from reaching external AI providers
- Canonical IDs (not raw values) are sent to AI providers where possible
- AI outputs are inspected for PII patterns before storage
- External provider calls are fully audited (correlation ID, tenant, classification, actor)

---

### T05 — AI-Generated Misinformation Presented as Fact
**Description:** AI output containing incorrect inferences or hallucinations is presented to decision-makers as verified institutional fact.
**Mitigation:**
- All AI outputs carry `ai_generated = true` and a classification (`FACT` / `INFERENCE` / `RECOMMENDATION` / `PREDICTION`)
- `UNKNOWN` outputs are never shown directly to decision-makers
- Recommendations with zero supporting evidence are rejected before storage
- UI clearly distinguishes AI-generated content from verified institutional data

---

### T06 — Unauthorized Recommendation Approval
**Description:** Attacker with `viewer` or `user` role approves an AI recommendation, triggering an unintended action.
**Mitigation:**
- Recommendation authorization requires `admin` or `institutional_lead` role
- Authorization creates a mandatory audit record
- The authorization API validates role before state transition

---

### T07 — Privilege Escalation via Graph Mutation
**Description:** Attacker creates graph edges linking themselves to privileged roles or objects to gain elevated access.
**Mitigation:**
- Knowledge graph edges are informational only — they do not confer permissions
- Access control is always enforced at the platform identity layer (Phase X JWT/RBAC), not via graph
- Graph mutation requires `admin` role

---

### T08 — Event Spoofing
**Description:** Attacker crafts fraudulent events (e.g., `kpi.measurement.updated` with inflated values) and publishes them to the Event Bus.
**Mitigation:**
- Events are published by source applications authenticated via service identity
- The Intelligence Processor validates `source_application` against the Registry
- Events from unknown sources are logged as `UNRECOGNIZED_SOURCE` and excluded from condition calculation
- Anomalous KPI values (outside historical range) are flagged as `SUSPECTED_ERROR`

---

### T09 — Event Replay Abuse
**Description:** Attacker replays legitimate past events to manipulate the intelligence projection into an older (or false) state.
**Mitigation:**
- Events carry `event_id` (deduplication) and `timestamp`
- The Intelligence Processor tracks processed `event_id` values to prevent re-processing
- Replay mode (used only for projection rebuild) requires `admin` authorization and is fully audited
- Replay does not overwrite current state; it reconstructs into a separate validation projection

---

### T10 — Graph Poisoning
**Description:** Attacker creates false authoritative relationships (e.g., linking a malicious dataset as supporting a critical KPI).
**Mitigation:**
- Authoritative edges can only be created via governed API by `admin` role
- INFERRED and AI-generated edges are visually distinguished from AUTHORITATIVE edges
- Edge provenance is immutable — once set, `provenance_type` cannot be changed
- Audit record created for every edge mutation

---

### T11 — KPI Manipulation
**Description:** Attacker submits false KPI measurements (via manual entry or compromised source system) to manipulate the institutional condition.
**Mitigation:**
- Manual entries are always flagged `quality_status = MANUAL` and visually distinguished
- Automated measurements that fall outside expected range are flagged `SUSPECTED_ERROR`
- KPI measurements carry full provenance — source system, dataset canonical ID, measurement timestamp
- Anomaly detection compares new measurements against historical rolling average

---

### T12 — Evidence Tampering
**Description:** Attacker modifies evidence records to conceal a recovery failure or security incident.
**Mitigation:**
- Evidence records are immutable — no UPDATE or DELETE API is exposed
- Evidence records carry SHA-256 verification hash computed at creation time
- The hash is re-verified on read; mismatch triggers an integrity alert
- Evidence records are replicated in the audit log for cross-reference

---

## Residual Risks

| Risk | Status |
|---|---|
| AI provider data breach (cloud provider) | ACCEPTED — mitigated by data classification gate; RESTRICTED/SENSITIVE data not sent to external providers |
| Sophisticated prompt injection in structured prompts | ACCEPTED — mitigated by template-based prompts; monitoring in AI audit log |
| Insider threat — admin creates false authoritative graph edges | ACCEPTED — mitigated by mandatory audit records and periodic graph integrity assertion |

---

*Status: REVIEWED*
*Sprint 0 Gate: THREAT MODEL — REVIEWED*
