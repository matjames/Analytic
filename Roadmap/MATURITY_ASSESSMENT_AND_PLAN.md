# StatGate Maturity Assessment & 90-Day Plan

**As of:** 15 September 2026
**Purpose:** Evidence-based assessment of how mature the StatGate system actually is today, and a prioritized 30/60/90-day plan to move it from "broad active implementation" to a dependable, operable product.
**Method:** Every claim below was verified against the live repository, live Docker stack, and CI definition on this date — not copied from older status docs.

---

## 1. Verified Evidence Snapshot (15 September 2026)

| Evidence point | Finding |
|---|---|
| Live Compose stack | **46/46 `analytic-*` containers Up, zero stopped/unhealthy** (verified via `docker ps`) |
| StatCollect deployment | **Now IN Compose and running** (`analytic-statcollect-1` Up). `PHASE_SERVICE_MATRIX.md` (29 Aug) still says "NOT in Compose" — **doc drift confirmed** |
| CI workflow | `.github/workflows/integration.yml`: secret scan, all-module Go build+test, compose config validation, analytics pytest, 11 frontend builds, StatCollect-only integration test. **No full-stack `up` + per-service health certification job. No evidence of a green run.** |
| Backend tests | **114 Go test files tracked** (concentrated in statgate-lib, StatChat, StatCollect, statdata — added during Stages 2–3) |
| Frontend tests | **5 test files across the entire repo.** Frontends are build-only in CI; no component or E2E coverage |
| Operations docs | Strong and current: `docs/BASELINE_RUNBOOK.md` (3 Sep), `docs/EVENT_CATALOG.md` (31 Aug), `docs/PRODUCTION_READINESS.md`, `docs/SECURITY_BASELINE.md`, `docs/DATABASE_CATALOG.md`, plus DR/backup/incident runbook folders |
| Working tree | Substantial **uncommitted Stage 3 work**: StatGovernance discussion frontend (`DiscussionButton.jsx`, `statchat.go` + tests), StatData science/search/link handlers, StatFederation events, StatCitizen event/config changes, roadmap doc updates |
| Last commit | `0ac5153` (31 Aug): Stage 3 event-bus taxonomy audit |
| Roadmap status | Stage 1 ✅ certified · Stage 2 ✅ rolled out (2 follow-ups open) · Stage 3 🟡 partial (API boundaries certified, browser entry points open) · Stages 4–6 ⬜ not started · 12 phases spec-only with no code |

## 2. Maturity Level

Using a standard capability-maturity scale:

| Level | Name | Meaning | StatGate |
|---|---|---|---|
| L0 | Ad-hoc | Works on one machine, manual everything | |
| L1 | Repeatable | Scripts exist, results vary | |
| L2 | Defined | Documented process, codified patterns, some automation | **← mostly here** |
| L3 | Verified | Automated proof: CI green, tests enforce invariants, health certified from clean env | ← Stage 1/2 work got individual pieces here, not the whole |
| L4 | Measured | Metrics-driven ops, SLOs enforced, drift detected | |
| L5 | Optimizing | Self-healing, continuous improvement | |

**Placement: solid L2, with islands of L3.** The patterns are codified (shared service pattern, shared tenant middleware, durable event publisher, runbooks), but the system as a whole cannot yet *prove* itself automatically. The single defining gap: **a human, not a pipeline, is the source of truth for "is it healthy."**

**In one sentence:** StatGate is a well-engineered L2 platform one automated-proof layer away from L3 — the code is largely present; the *certification machinery* is what's missing.

## 3. Gap Analysis by Maturity Dimension

### 3.1 CI/CD proof (L2 → L3) — the keystone gap
- CI is well-designed but **never proven green** and does not exercise the real deployment path (only StatCollect's standalone compose).
- No full-stack job: `compose up` → wait → assert `/health` + `/ready` 200 for all services → teardown. Stage 1 proved this is possible by hand; it must be a pipeline job.
- **Risk:** any of the 15+ Go modules or 11 frontends can silently rot between manual runs.

### 3.2 Testing pyramid — thin above unit
- Backend: good unit/contract coverage where it matters most (tenant middleware, event bus, StatChat).
- Frontend: essentially zero tests (5 files); CI only proves it compiles.
- **Missing entirely:** cross-service tenant/authorization suite (Critical Gap #3) — the invariant "no cross-workspace read ever succeeds" is enforced per-service but not asserted *system-wide*.

### 3.3 Doc-vs-reality drift
- Confirmed example: matrix says StatCollect absent from Compose; reality shows it running. Drift erodes the value of the whole `Roadmap/` + `docs/` corpus.
- PROJECT_PROGRESS.md percentages (90–95%) contradict the phase matrix's honest IP/IMP statuses.

### 3.4 Stage 3 completion (in flight)
- All 10 later services: conversation APIs certified, fail-closed verified.
- Open: browser entry points (uncommitted work), broad durable-consumer rollout, StatChat browser E2E + production TURN.

### 3.5 Production posture
- `docs/PRODUCTION_READINESS.md` and `SECURITY_BASELINE.md` exist but no dated certification that the production settings (strong secrets, TLS, `ANALYTICS_REQUIRE_AUTH`, notebook execution off) are actually in force.
- Stage 2 leftovers: tenant-branding adoption by later-phase UIs; deeper scoping (statdata contracts/quality children, geointel drones/rasters/scenes).

### 3.6 Product completeness (Stage 4+)
- Business workflows above CRUD are the user-visible value gap: PMS LogFrame/ToC/donor, RMS DOI/IRB, Statistics questionnaire→dissemination, Governance whistleblower→board packs, Analytics forecasting/NLQ.
- 12 phases have no code (14, 18, 29, 32, 33, 35, 36, 42, 43, 45, 46, 50) — correctly parked; keep them parked until L3.

## 4. The 30/60/90-Day Plan

> Ordering principle: **prove before you build, finish before you expand.** Each phase ends with a dated certification note appended to `CURRENT_STATE_AND_WAY_FORWARD.md`, per existing convention.

### Days 1–30 — "Prove what exists" (target: L3 foundation)

| # | Action | Exit criteria | Effort |
|---|---|---|---|
| 1.1 | **Commit the Stage 3 working tree.** Review, test (`go test ./...` per touched module), and commit the uncommitted StatGovernance/StatData/StatFederation/StatCitizen work in logical commits | Clean `git status`; all touched modules' tests pass locally | Small |
| 1.2 | **Prove CI green.** Push, watch `integration.yml`, fix every red job until the full workflow passes on `master` | Green run recorded; branch protection requiring CI enabled | Small–Medium |
| 1.3 | **Add the full-stack certification job** to CI: compose up with placeholder secrets → poll `/health` + `/ready` across all ~46 services → fail on any non-200 → teardown. Reuse the Stage 1 manual procedure as the script | Job green in CI on a clean runner | Medium |
| 1.4 | **Correct doc drift.** Update `PHASE_SERVICE_MATRIX.md` (StatCollect now in Compose; add StatCitizen) and reconcile `PROJECT_PROGRESS.md` percentages with the honest IP/IMP model | Matrix and progress doc match `docker ps` output | Small |
| 1.5 | **Start the tenant/authorization suite** (`tests/tenancy/`): one reusable scenario runner asserting (a) malformed `X-Workspace-ID` → 400, (b) no credentials → 401, (c) cross-workspace read → 403/404, for every service, driven off a service/port manifest | Runner covers the 10 later services + PMS/RMS/StatChat/Governance | Medium |

### Days 31–60 — "Finish Stage 3, harden ops"

| # | Action | Exit criteria | Effort |
|---|---|---|---|
| 2.1 | **Ship browser entry points** for object conversations in the 4 frontends with checked-in sources (StatGovernance done in working tree → then StatSpatial/StatTrust/StatOps; StatData/GeoIntel/BPM/StatIoT/AI-Autonomy/Learning lack frontend sources — document as explicit non-goals rather than silently open) | A user clicks a discussion button in each shipped frontend and reaches the canonical conversation | Medium |
| 2.2 | **Complete Stage 2 follow-ups:** tenant branding adopted by later-phase UIs; statdata contracts/quality child scoping; geointel drones/rasters/scenes scoping | Matrix Stage 2 section marked closed with date | Medium |
| 2.3 | **Broad durable-consumer rollout** — every service that publishes consumes via named consumer groups with retry-to-DLQ (pattern already certified in statdata/statiot/ai-autonomy/federation/trust/ops) | Event catalog section per service updated; DLQ verification live | Medium |
| 2.4 | **Expand tenancy suite to 100% of deployed services** and wire it into CI as a nightly + PR job (needs the full-stack job from 1.3 as its harness) | Tenancy job green; any new service without coverage fails CI | Medium |
| 2.5 | **StatChat Phase 3 acceptance:** enable browser-based E2E (Playwright) for core flows; production TURN checklist (public address + TLS) | E2E green in CI for messaging basics; TURN runbook updated with live evidence | Large |
| 2.6 | **Production posture certification:** dated checklist run — secrets rotated/strong, `ANALYTICS_REQUIRE_AUTH=true`, notebook execution off, TLS/reverse-proxy front, backup/restore drill executed once and logged | `docs/PRODUCTION_READINESS.md` gains a dated "certified" section | Medium |

### Days 61–90 — "Open Stage 4" (only if Days 1–60 are certified)

| # | Action | Exit criteria | Effort |
|---|---|---|---|
| 3.1 | **Pick ONE product slice as a template** — recommended: PMS LogFrame/ToC/donor (highest user value, schema already exists). Execute it against the full Definition of Done: migrations on clean env, workspace scoping, events, tests, compose, runbook | Acceptance scenario completable by a user without manual DB intervention | Large |
| 3.2 | **Codify the Definition of Done as a checklist script/doc** so every subsequent Stage 4 slice (RMS DOI/IRB → Statistics → Governance) follows the identical path | Checklist committed; PMS slice certified against it | Small |
| 3.3 | **Second slice** (RMS DOI/citations/IRB) using the template | Same exit criteria as 3.1 | Large |
| 3.4 | **Maturity gate review:** re-run this assessment; if CI + tenancy suite + full-stack health are all green for 30 consecutive days, declare L3 and queue the next product slices | Updated assessment appended here | Small |

**Explicitly NOT in the 90 days** (kept parked, per the Way Forward): LLM gateway/embeddings/RAG (Stage 5), all 12 spec-only phases, marketplace/low-code, mobile offline, and any new service. Expansion before the proof layer exists would *lower* maturity.

## 5. Top Risks

1. **Uncommitted work ages badly** — the Stage 3 working tree is the most valuable unpushed asset; commit it first.
2. **No green CI = silent rot** — 15+ Go modules and 11 frontends change between manual verifications.
3. **Doc drift compounds** — 50 phase files + 6 status docs; a dated matrix reconciliation now prevents a full rewrite later.
4. **Frontends are untested** — build-only CI means UI regressions ship unnoticed; even 3–4 smoke tests per major frontend would change the risk profile.
5. **Docker Hub/environment dependency** — the 2 Sep checkpoint noted image rebuild stalling on dependency download; pin/verify base images in the certification job.

## 6. How the Assistant Executes This

Each line item above is directly executable in this workspace: code review + commit sequencing (1.1), CI authoring and debugging (1.2–1.3), doc reconciliation (1.4), test-suite authoring (1.5, 2.4), frontend feature work (2.1), event-bus rollout (2.3), E2E scaffolding (2.5), Stage 4 slice implementation (3.1–3.3), and dated certification updates throughout.



