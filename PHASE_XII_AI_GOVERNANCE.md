# PHASE_XII_AI_GOVERNANCE.md
# StatGate Phase XII — Governed AI & Reasoning Layer

## Provider abstraction (directive §16)
AI integration is provider-agnostic via the `ReasoningProvider` interface:
`Name() string` and `Generate(AIInput) (*AIOutput, error)`. The platform is not
coupled to a single provider. Implementations: a deterministic local default,
and `openai` / `gemini` / custom HTTP adapters. Configuration:
`STATGATE_AI_PROVIDER` + provider key + `STATGATE_AI_MODEL` + `STATGATE_AI_BASE_URL`
(optional). A provider configured without a key **fails closed** — AI reasoning
is disabled rather than falling back anonymously.

## AI is advisory only (directive §4, §19)
AI may observe, analyse, correlate, forecast, explain, recommend, summarise. It
never directly modifies/ deletes records, changes permissions/security policies,
executes recovery, alters financial records, changes governance/resilience
configuration. The engine exposes **no** path for a provider to mutate institutional
records — only the human-gated recommendation lifecycle exists.

## Input contract (directive §17)
Minimal necessary data only: tenant context, authorised institutional snapshot,
supporting evidence, relevant graph relationships, relevant metrics, data
classification, `correlation_id`. **Classification gate:** `RESTRICTED` and
`SENSITIVE` data are never sent to a provider (fail closed).

## Output contract (directive §18)
Structured JSON: `recommendation, confidence, reasoning_summary,
supporting_evidence[], affected_objects[], risk_level, recommended_actions[],
limitations[]`. Output is labelled `AI_GENERATED`.

## Lifecycle (directive §19)
```
GENERATED → PENDING_REVIEW → AUTHORIZED | REJECTED → EXECUTED | CANCELLED
```
No AI recommendation skips the human review stage. Only a human with
`admin` / `institutional_lead` may authorize or execute. A `REJECTED`
recommendation can never execute. Execution is recorded as performed by a human
operator — AI never self-executes.

## Audit (directive §20)
Dedicated `ai_audit_log` (linked to the platform audit system via
`correlation_id` / `request_id` / `actor`) records: provider, model, request_id,
correlation_id, input/output classification, recommendation, confidence,
timestamp, actor, tenant, authorization status, human decision. Sensitive prompts
are never stored.
