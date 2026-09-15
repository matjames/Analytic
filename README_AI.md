# AI Guidance

AI features must assist people without hiding uncertainty or bypassing authorization. Every AI request is tenant-scoped, attributable to an authenticated user or service identity, and subject to the source data permissions of that user.

## Required controls

- Do not send protected or personal data to an external model unless the tenant has explicitly enabled the integration.
- Record model, task type, source references, output status, and human approval where the feature creates an operational decision.
- Make generated content visibly distinguishable from verified evidence.
- Provide a correction, refusal, and escalation path for every user-facing assistant.
- Apply rate limits, spend limits, retention controls, and provider timeouts.
- Never let generated text execute code, change permissions, or publish a decision without an explicit guarded workflow.

AI services should use the shared identity, event, audit, and workspace conventions. See [docs/AI/README.md](docs/AI/README.md) and [Roadmap/phase9.md](Roadmap/phase9.md).
