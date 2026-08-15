# PHASE_XII_OPERATIONS_RUNBOOK.md
# StatGate Phase XII — Operations Runbook

## Startup / schema
- The additive schema `docker/postgres-init/12-create-phase12-intelligence.sql` is
  applied automatically on PostgreSQL container init
  (`./docker/postgres-init:/docker-entrypoint-initdb.d:ro`).
- Enterprise Core verifies `12-create-phase12-intelligence.sql` only at DB init;
  for existing deployments run the migration against `statgate_enterprise` once.

## AI provider configuration
- Default: deterministic-local provider (no external calls). Advisory only.
- To enable an external provider:
  1. Set `STATGATE_AI_PROVIDER` (`openai` / `gemini` / custom).
  2. Set the matching key (`OPENAI_API_KEY` / `GEMINI_API_KEY` /
     `STATGATE_AI_API_KEY`) and optional `STATGATE_AI_MODEL`,
     `STATGATE_AI_BASE_URL`.
  A provider set without a key is disabled (fail closed) — no anonymous fallback.

## Privileged roles
Graph mutation, KPI/objective/risk/AI authorization require `admin` or
`institutional_lead`. Assign `institutional_lead` in the identity/registry system
as appropriate.

## Health & observability
- Phase XII metrics are exposed via `/metrics` (`phase12.*`) and
  `/api/intelligence/overview`, including `statgate_intelligence_events_processed_total`,
  `statgate_intelligence_condition_calculations_total`,
  `statgate_intelligence_condition_calculation_duration_ms`,
  `statgate_graph_queries_total`, `statgate_graph_query_duration_ms`,
  `statgate_ai_recommendations_total`, `statgate_ai_recommendations_pending`.

## Backup / restore
Phase XII tables live in `statgate_enterprise` and are covered by existing
PostgreSQL backup (continuous WAL + snapshot) and the Phase XI backup and
integrity engines.

## Operator actions required before certification (BLOCKERS)
1. **BLOCKER-01:** rotate all previously exposed credentials
   (`docs/security/SECRET_MANAGEMENT.md`). Do not hardcode replacements.
2. **BLOCKER-02:** approved Git-history rewrite to remove historical secrets
   (operator-approved; coordinate with rotation).
3. Re-run the full regression suite after those actions, then issue certification.

## Incident notes
- A signal/condition recalculation is synchronous on event; if Redis is down the
  event bus degrades to persisted replay without data loss (existing Phase X
  behaviour). Phase XII conditions remain durable in PostgreSQL.
