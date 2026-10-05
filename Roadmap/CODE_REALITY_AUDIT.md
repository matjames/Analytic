# StatGate Code Reality Audit

**As of:** 18 September 2026
**Method:** Direct inspection of committed code (`git ls-files` plus source reads), not roadmap prose. Every claim below cites the exact file where it was verified.

**Purpose:** Record, with evidence, why the current codebase delivers the *image* of the expected full-lifecycle platform without delivering the *value* of it, and define the corrective plan.

## Verdict Summary

The platform spine is real and valuable. The data lifecycle core — the reason the platform exists — is simulated where it matters. The system collects real data into a hollow center: nothing moves real bytes between collection and publication.

## Verified Assets (Gold)

| Asset | Evidence |
|---|---|
| Registry identity/tenancy | `stage_register/go-backend/` — MFA/TOTP, OIDC, sessions, rate limiting, real handler suite |
| Durable event bus | `statgate-lib/events/` — consumer groups, retry-to-DLQ, reclaim (certified live) |
| Workspace tenancy contract | `statgate-lib/tenant/workspace.go`; attested by `tests/1-tenancy/` (36/36 + 47 authenticated live checks) |
| StatCollect (lifecycle stage 1) | `StatCollect/internal/server/db.go` — real XForms/XML submissions, attachments, Postgres persistence |
| StatData metadata plane | `StatData/backend/internal/store/pg_store.go` (2,556 lines) — datasets, pipelines, lineage with recursive-CTE graph, migrations |
| Business modules | PMS/RMS/StatChat/StatCitizen/Helpdesk — real persistence, routes, tests |
| Code honesty | Only 32 TODO/FIXME marks across ~2,080 Go files — the code is not stubbed; it is *shallow in the center* |

## Verified Hollow Points (Stones)

1. **statgate-core lakehouse is an in-memory demo.** `backend/internal/lakehouse/lakehouse.go` — `StorageEngine` holds records in Go slices and calls `seedDemoData()` on startup (fake tenants `tenant-alpha`/`tenant-beta`, synthetic temperature readings). All data is lost on restart. PostgreSQL is opened only for `Ping()` and asset-manager persistence (`backend/cmd/server/main.go` lines ~99–147).
2. **Analytics engines are stateless.** `backend/internal/tabulation/engine.go` and `backend/internal/sampling/engine.go` contain correct mathematics (crosstabs, chi-square, sample sizing, stratification), but all data arrives inside the HTTP request (`DataRecords []map[string]interface{}`) or lives in an in-memory frame map. They are not connected to any dataset, table, or file.
3. **The pipeline engine is a simulation.** `StatData/backend/internal/pipeline/engine.go`, `TriggerPipeline()` — `totalRead = 10000`, `totalWritten = 9985` are hardcoded; the loop comment states *"Simulate processing based on stage type"*; every stage returns `SUCCESS` plus an arbitrary `+45` ms duration. No stage ever reads from or writes to `storage_uri`. Quality scores are evaluated over data that never moved.
4. **No end-to-end proof exists.** `tests/` contains only `1-tenancy/`. There is no test, script, or certification anywhere in the repository that carries one real submission from StatCollect through StatData into a published statistic.

**Net lifecycle status:** collect (real) → ingest (events real; data plane absent) → process (simulated) → analyze (math on request payloads) → publish (dashboards over simulated/seeded results).

## Root Causes

1. **Breadth-first execution.** 50 roadmap phases and 42 Compose services were advanced in parallel while the core statistics service holds 19 Go files and a toy lakehouse. Breadth was treated as progress.
2. **Demo-data-driven development.** Where a hard problem appeared (real data movement), a simulation returning `SUCCESS` was inserted. Dashboards therefore read as complete, and per-service "certification" (endpoints answering) was mistaken for lifecycle delivery (data flowing).
3. **Missing definition of done.** Phases were verified in isolation. With no end-to-end acceptance test, a hollow center stayed invisible through eleven phase-completion reports.

## Corrective Plan

**Rule 1 — Breadth freeze.** No new services or roadmap domains until the lifecycle spine moves real bytes end to end.

**Rule 2 — One real vertical slice.** Definition of done for the slice:
1. A StatCollect submission is registered as a dataset in StatData with a real `storage_uri`.
2. A StatData pipeline run EXTRACTs the stored records, TRANSFORMs them, and LOADs the result, with read/written counters reflecting actual records and `FAILED` status on real errors.
3. statgate-core tabulates the transformed result **from persisted storage** (not request payloads).
4. The published output is reproducible from the persisted artifacts alone.

**Rule 3 — Desimulation.**
- Delete `seedDemoData()` from the lakehouse; persist event records to Postgres or Parquet.
- Replace the simulated stage loop in `TriggerPipeline` with real implementations of at least three stage types (EXTRACT from Postgres/MinIO, TRANSFORM via SQL, LOAD to Postgres/MinIO).
- Move sampling frames and any engine state out of memory maps into the database.

**Rule 4 — End-to-end acceptance gate.** Add `tests/2-lifecycle/`: one automated suite that executes Rule 2's slice against the live Compose stack and fails if any stage is simulated. This gate becomes mandatory for every future phase.

**Rule 5 — Sequencing.** Only after Rules 1–4 hold: Stage 5 advanced intelligence (LLM gateway, embeddings, model registry) and Stage 6 later domains, per `MATURITY_ASSESSMENT_AND_PLAN.md`.

## Remediation Progress

### 18 September 2026 — StatData pipeline engine desimulated (Rules 2–3, partial)

The simulated pipeline executor is gone and replaced with a real one:

- **Real data plane:** `statdata.dataset_records` (JSONB rows) added via `StatData/backend/migrations/000002_dataset_records.up.sql` (+ down migration) and auto-provisioned by `PGStore.ensureSchema`. Implemented in both `store/pg_store.go` (transactional batch JSONB inserts) and `store/mem_store.go`.
- **Real execution:** `pipeline/engine.go` `TriggerPipeline` now dispatches EXTRACT (reads stored source rows), TRANSFORM (per-record filter/rename/remove_fields/set_constant/cast_number via `pipeline/transform.go`), VALIDATE (rules evaluated against real in-flight rows, ERROR failures fail the stage), LOAD (append/replace into the target dataset with catalog counters synced to actual storage), ENRICH, ANONYMIZE. Unknown stage types and missing datasets fail honestly with messages; runs record `FAILED` with `ErrorMessage` instead of guaranteed `SUCCESS`.
- **HTTP data-plane API:** `POST/GET/DELETE /api/v1/data/catalog/datasets/:id/records` (`api/record_handlers.go`) so other services (StatCollect, statgate-core) can push and read dataset rows.
- **Evidence:** `go vet`, `go build`, and `go test ./...` all pass; `tests/pipeline_test.go` now proves a 4-in/3-out/1-rejected execution with physically stored transformed rows, plus two failure tests (missing source → FAILED with message; quality violation → FAILED with an untouched target).

### 18 September 2026 — statgate-core lakehouse desimulated + analytics bound to stored data (Rule 3, continued)

- **`backend` lakehouse is now durable PostgreSQL.** `internal/lakehouse/lakehouse.go` was rewritten: `NewStorageEngine(db)` returns an error instead of fabricating a demo store, `seedDemoData()` is deleted, and events/alerts persist to `core.event_ledger` and `core.anomaly_alerts` (self-provisioned, idempotent). `Ingest` computes the 3σ window from persisted rows; `QueryTenantData`/`GetAnomalies`/`GetStats` are SQL-scoped by (tenant, workspace). `cmd/server/main.go` now fails closed at startup if Postgres or the schema is unavailable.
- **Tabulation reads what pipelines stored.** `internal/tabulation/engine.go` gained `LoadRecordsFromStatData()`, and `TabulationRequest.DatasetID` binds a tabulation to rows in `statdata.dataset_records`. `handleTabulate` and `handleTabulationExport` load stored rows whenever `dataset_id` is present; request-embedded `data_records` remain only as a compatibility path when no dataset is given.
- **Deployment wiring.** `docker-compose.yml` now passes `STATDATA_DB_*` to `statgate-core`; `docker compose config` validates. Verified live in the running stack that `StatDataEngine` owns the `statdata` schema, can create `dataset_records`, and can insert/select rows.
- **Evidence:** `go vet ./...` and `go test ./...` pass for `backend` and `StatData/backend`. New tests execute against real PostgreSQL (not skipped) and pass: durable ledger readback, 30-steady-then-outlier 3σ detection with restart-safe re-open (31 rows, 1 persisted anomaly), tenant/workspace query isolation, and dataset-backed tabulation with correct crosstab marginals.

### 18 September 2026 — End-to-end lifecycle acceptance gate added (Rule 4)

`tests/2-lifecycle/` now enforces Rule 2's vertical slice as an executable gate
(`lifecycle.ps1`, 58 checks, `manifest.json`, `TEST_RESULT`, `README.md`). It
asserts ingest → real pipeline execution → durable storage → dataset-bound
analysis, and it fails if any stage is simulated: guaranteed `SUCCESS`, counters
that disagree with stored rows, a failed `VALIDATE` that still loads, or
analytics that reads the request payload instead of persisted rows.

Verified against services built from current source over real PostgreSQL/Redis:
**58 passed, 0 failed, exit 0**, repeatable. Verified as a *discriminating* gate
by running it against the pre-change Compose containers, where it failed and
printed the old hardcoded counters — `records_read=10000`,
`records_written=9985`, `records_rejected=15` — while the old build still
reported `SUCCESS`. The gate is wired into the nightly
`full-stack-certification` CI job.

**Running the durable tests.** The PostgreSQL-backed Go tests skip cleanly when
no database is reachable, so they cannot silently rot. To execute them, provide a
DSN (CI and local both use this mechanism):

```powershell
$env:STATGATE_TEST_DSN     = 'postgres://<user>:<pass>@<host>:5432/statgate_test?sslmode=disable'
$env:STATGATE_CORE_TEST_DSN = $env:STATGATE_TEST_DSN
$env:STATDATA_TEST_DSN     = 'postgres://<user>:<pass>@<host>:5432/statdata_test?sslmode=disable'
go test ./...               # backend; StatData/backend takes STATDATA_TEST_DSN only
```

**Still open (not claimed):** the running Compose containers still execute the
pre-change build (the gate deliberately fails against them); StatCollect
submissions are not yet auto-registered as StatData datasets; sampling frames and
the condition/semantic registries remain in-memory.

## Honest Restatement of Position

The repository contains an excellent platform chassis — identity, tenancy, durable events, and several genuine business applications — wrapped around a hollow data engine. The corrective work is not more services, more phases, or more documents. It is making one lifecycle slice real, proving it with an executable test, and refusing to report success on simulated stages.
