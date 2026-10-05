# End-to-End Data Lifecycle Acceptance Gate

One executable suite that proves StatGate's data lifecycle works as a single
product, against a running stack:

**collect / ingest → real pipeline execution → durable storage → analysis**

It exists because the audit in `Roadmap/CODE_REALITY_AUDIT.md` found that the
platform could report lifecycle success while moving no data (hardcoded counters,
guaranteed `SUCCESS`, analytics fed by request payloads). This gate is the
guardrail against that class of regression.

| File | Purpose |
|---|---|
| `manifest.json` | The stages, the invariants asserted, and the failure conditions |
| `lifecycle.ps1` | The executable gate (58 checks; exits non-zero on any violation) |
| `TEST_RESULT` | Recorded evidence from the last run |

## Running

```powershell
# against the Compose stack (defaults: statdata :8107, statgate-core :8082)
powershell -NoProfile -ExecutionPolicy Bypass -File tests/2-lifecycle/lifecycle.ps1

# against services started from source on alternate ports
powershell ... -StatDataBase http://localhost:8307 -CoreBase http://localhost:8302
```

Reads `STATGATE_INTERNAL_API_KEY` and `STATGATE_REGISTRY_JWT_SECRET` from the
environment, falling back to the gitignored `.env`. Exits `1` on any failed
check and `2` when the stack or the required secrets are missing.

## What is asserted

**Preflight** — both services healthy; the catalog rejects requests without a
token (`401`); core tabulation rejects requests without the internal key (`401`).

**Stage 1 · collect/ingest** — datasets are created and hold *real* rows: an
empty batch is refused (`400`), four rows store and read back, `bytes_written`
is positive, and `total_rows` reflects storage.

**Stage 2 · define** — a `NOT_NULL` quality rule and a four-stage ETL pipeline
(`EXTRACT → VALIDATE → TRANSFORM → LOAD`) are created. Validation deliberately
runs **before** the transform: the rule targets `district`, which the transform
renames to `region`. (Running it the other way is a genuine pipeline-design bug
— and the engine reports it honestly.)

**Stage 3 · process** — the run reports `SUCCESS` with `records_read=4`,
`records_written=3`, `records_rejected=1`, four stage runs, and per-stage
summaries that must match reality: `records_extracted=4`, `records_in=4`,
`records_out=3`, `quality_status=PASSED` with a real rule evaluated,
`records_loaded=3` with `bytes_written>0`, `total_rows=3`.

**Stage 4 · store** — the target physically contains three transformed rows
(`region` present, `district` renamed away, `note` removed); catalog
`row_count`/`size_bytes` match storage; a **second run appends** (6 rows);
a record reset deletes all 6 and the target reads back empty.

**Stage 5 · fail honestly** — a pipeline whose source dataset does not exist
reports **`FAILED`** with a real error message and writes nothing; a dataset
that violates `NOT_NULL` fails at the `VALIDATE` stage, names the violated rule,
never runs `LOAD`, and leaves the target empty.

**Stage 6 · analyze** — tabulation bound to `dataset_id` returns results
computed from **persisted rows** (`total_record_count=4`, `North` row total `2`),
and a request that supplies bogus payload rows *alongside* `dataset_id` is
ignored — the result still reflects storage, proving analytics does not read the
request body.

## Results (2026-09-18)

- **58 checks, 0 failures** against services built from current source
  (`statdata` :8307, `statgate-core` :8302 over real PostgreSQL + Redis).
  Repeatable: consecutive runs both returned `58 passed, 0 failed` and exit `0`.
- **The gate discriminates real from simulated.** Run against the *pre-change*
  Compose containers (which still run the old build), it fails and exposes the
  simulation directly:

  | Check | Old stack | New code |
  |---|---|---|
  | `POST .../datasets/:id/records` | `404` (no data plane) | `201`, real rows |
  | `records_read` | **`10000`** (hardcoded) | `4` (rows actually read) |
  | `records_written` | **`9985`** (hardcoded) | `3` |
  | `records_rejected` | **`15`** (hardcoded) | `1` |
  | `run status` | `SUCCESS` while moving nothing | `SUCCESS`, verified |
  | Exit code | `1` | `0` |

  Those three numbers are the literal constants removed from
  `StatData/backend/internal/pipeline/engine.go`, and the old build reported
  `SUCCESS` while producing them — exactly the failure mode this gate exists to
  prevent.
- The gate also proved its worth on its first run against the new code: it
  **failed** because the pipeline validated *after* renaming `district` →
  `region`, so the `NOT_NULL` rule correctly fired and the run honestly reported
  `FAILED`. That was a pipeline design error surfaced by the engine, not a
  simulated pass.

## Notes

- The gate cleans up the datasets it creates. Pipeline *definitions* remain
  (there is no `DELETE /api/v1/data/pipelines/:id` route yet); they use
  timestamped ids so runs do not collide.
- The nightly `full-stack-certification` job runs this gate after health
  certification, so any regression to simulated behaviour fails CI.

## Reproducing the 58/58 evidence without rebuilding containers

The recorded run drove services built straight from source over the developer
PostgreSQL/Redis (faster than a full image rebuild):

```powershell
# 1. dev databases (same server the services use)
#    CREATE DATABASE statdata_live;  CREATE DATABASE statgate_live;

# 2. build both services
go build -C StatData/backend -o ../.runtime-bin/statdata-live.exe ./cmd/server
go build -C backend        -o ../.runtime-bin/statgate-core-live.exe ./cmd/server

# 3. run statdata on :8307 (DB_* -> statdata_live, REDIS_ADDR -> localhost:6379,
#    JWT_ISSUER=statgate-registry, JWT_AUDIENCE=statgate)
#    and statgate-core on :8302 (KAGGLE_DB_* -> statgate_live,
#    STATDATA_DB_* -> statdata_live, STATGATE_INTERNAL_API_KEY set)

# 4. run the gate against those ports
powershell -NoProfile -ExecutionPolicy Bypass -File tests/2-lifecycle/lifecycle.ps1 `
    -StatDataBase http://localhost:8307 -CoreBase http://localhost:8302
```

The core's dataset tabulation binding additionally has Go-level coverage in
`backend/internal/tabulation/tabulation_statdata_test.go`, which executes
against real PostgreSQL when `STATDATA_TEST_DSN` is provided (it skips cleanly
otherwise).
