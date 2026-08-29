# StatGate / Analytic — Monorepo Code-Quality & Hardcoding Audit

Audited: `c:\Users\PC\Desktop\Analytic` (git repo `matjames/Analytic`, last commit `161e227`)
Method: automated pattern scan of **first-party source + config files** in every top-level project
directory (Python / Go / Java-Kotlin / JS-TS / C-Fortran / SQL / YAML / JSON / properties / shell /
PowerShell), **excluding** `node_modules`, `.gocache`, `__pycache__`, `.venv`, `.git`, `.next`,
minified/vendor bundles (`*.min.js`, `dist/`, `public/assets`, `bootstrap`, `vendor/`, `*.map`).

> **Read-me-first notes**
> - `collect-master/` is the **upstream open-source ODK Collect app** (third-party). It drives a
>   large share of the raw counts (94 of 128 TODO files, nearly all `example.com` test fixtures,
>   all *lorem ipsum* hits). Those are upstream code, **not** team-authored defects.
> - The repo already ships a secret-scan tool (`scripts/secret-scan.ps1`), `docs/security/` docs and
>   `.gitignore`; most real `.env` files are **gitignored** (verified: root `.env` and all
>   `.env.example` files are NOT tracked by git).

---

## 1. Aggregate results (non-vendor, non-minified files only)

| Category | Files | Match-lines | Notes |
|---|---|---|---|
| `HTTP_URL` (any `http(s)://`) | 1,219 | 11,331 | dominated by JSON config / legit endpoints — §4 |
| `LOCALHOST` (localhost / 127.0.0.1) | 193 | 748 | dev-defaults; a few big offenders — §4 |
| `MARKER_TODO` (TODO/FIXME/HACK/XXX/WIP/TBD) | 128 | 210 | §2 |
| `PLACEHOLDER_STUB` (stub / not-implemented / dummy) | 54 | 177 | §2 |
| `HARD_IP` (public IP literals) | 25 | 37 | §4 |
| `HARD_PORT` (default port literals) | 17 | 30 | §4 |
| `DB_CONNSTR` (db://connection strings) | 15 | 16 | §3 |
| `CRED_ASSIGN` (key= "…" cred-like lines) | 10 | 29 | §3 |
| `EXAMPLE_DOMAIN` (example.com / demo keys) | 35 | 138 | mostly ODK tests; §5 |
| `LOREM` (lorem ipsum / foobar) | 6 | 19 | all in collect-master (upstream) |
| `DUMMY_TOKEN` (changeme / your_secret…) | 2 | 7 | §5 |
| `JWT_TOKEN` / `KEY_MATERIAL` (private keys) | 0 | 0 | none found in scanned code extensions |

---

## 2. Placeholders / stubs / markers

**Top first-party TODO offenders (by match lines):**
- `appluancher/` — source + `.next` build chunks (remove `.next` build artifacts from repo / gitignore)
- `PMS/frontend/src/components/KanbanBoard.jsx` (3), `PMS/backend/db_handlers.go` (3), `PMS/backend/database.go` (3)
- `RMS/frontend/src/components/TasksTab.jsx` (5)
- `stage_register/frontend/src/pages/initiator/RequestCreate.jsx` (2)
- `StatCollect/internal/server/server.go` (2), `StatChat/frontend/src/components/TasksPanel.tsx` (2)
- `bpm-hub/backend/pkg/api/handlers_bpm.go` (2), `docker/postgres-init/02-create-pms-db.sql` (3)
- `StatGovernance/backend/main.go`

---

## 3. Hardcoded credentials & secrets (SECURITY — highest priority)

### 3a. Credential assignments (first-party, non-vendor)
| File:line | Detail | Risk |
|---|---|---|
| `k8s/00-namespace.yaml:56` | `POSTGRES_PASSWORD: "statgate_k8s_password_prod"` (16 refs in file) | **CRITICAL** — real DB password hardcoded in committed k8s manifest |
| `StatChat/backend/pkg/store/collaboration.go:254` | seeded room with hardcoded `Password: "SG-4821"`, `URL:"statchat.local/room/..."` | hardcoded seed creds |
| `StatChat/start-backend.ps1:16` | `$env:DATABASE_URL = "postgres://Statchat:Statgate@localhost:5432/statchat?sslmode=disable"` | hardcoded DB user/pass in startup script |
| `StatSpatial/frontend/src/App.jsx:67` | DLP-scan demo string `api_key = "sk_live_uganda_data_secret"` | test/sample, confirm not real |
| `backend/cmd/server/main_test.go:39` | `internalAPIKey: "test-secret"` | test fixture (OK) |
| `collect-master/*` | `PASSWORD = "PASSWORD"`, `KEY_PASSWORD = "password"` | field-name constants (OK) |

### 3b. `.env` / `.env.example` files present in working tree (NOT git-tracked)
21 `.env*` files found. Several contain **real, non-empty credentials**:
- Root `./.env`, `StatCollect/.env`, `builder-main/.env`, `StatChat/.env`, `PMS/.env`, `RMS/.env`,
  `stage_register/go-backend/.env` — populated DB passwords, `STATCOLLECT_ADMIN_KEYS=dev-admin-key-2026`,
  JWT secrets, API keys.
- Several **`.env.example` files carry real values instead of placeholders**: `ai-autonomy/backend/.env.example`,
  `bpm-hub/backend/.env.example`, `geointel/backend/.env.example`, `knowledge-portal/backend/.env.example`
  (DB password len 34, JWT secret len 31), `learning-crm/backend/.env.example` (DB password len 28),
  `StatChat/backend/.env.example`.

These are untracked (not cloneable) but sit in the repo on disk: **rotate any that were ever shared**
and move real secrets to a vault/secrets manager.

### 3c. Key material
- No `-----BEGIN PRIVATE KEY-----` literals in scanned source.
- Working tree (not committed): `collect-master/debug.keystore` (tracked), `collect-master/collect_app/src/main/res/raw/isrgrootx1.pem` (trust root, benign).

---

## 4. Hardcoded configuration (URLs / ports / IPs / localhost)

**Service catalog (central hardcoded registry):**
- `frontend/config/services.json` — 22 services hardcoded with `http://localhost:<port>` UI + Docker-internal
  `http://statgate-*:<port>/health` hostnames. **Not env-driven** — should be parameterized.
- `appluancher/src/utils/mockData.ts` (~533 lines) — app-launcher ships a **mock dataset of ~24 services
  with 96 localhost URL/health/doc entries** for the launcher UI.
- `helpdesk-master/frontend/src/config/launcherApps.js`, `stage_register/frontend/src/config/launcherApps.js`,
  similar `launcherApps.js` in the Stat services (≈10 localhost refs each) — duplicated hardcoded launcher configs.
- `enterprise/core/*.go` (fabric.go 19, dashboards.go 11, service_registry.go 9, governance.go 7…) — many
  localhost service URLs embedded in Go.

**Hardcoded deployment IPs / SSH users (build scripts):**
- `helpdesk-master/frontend/package.json:37` — deploy `scp -rP 20020 ./build/* frank@172.27.1.123:...`
- `stage_register/frontend/package.json:51` — deploy `scp -r -P 20020 ... frank@172.27.1.240:...`
- `StatChat/backend/pkg/api/handlers.go:1386` — loopback allow-list checks (by design)

**Default port literals (fallback defaults — mostly OK-by-design):** `port = "5432"` repeated across
`backend`, `ai-autonomy`, `bpm-hub`, `knowledge-portal`, `learning-crm`, `StatChat`, `StatSpatial`,
`geointel`, `statgate-lib/database/database.go`; `6379` (redis) in `enterprise/core/redis.go`,
`statgate-lib/events/events.go`, `StatCollect/config.go`; `8080` in `backend`, `builder-main/server/main.go`.
Config is **duplicated per service** — could be centralized in `statgate-lib`.

**IP literals:** mostly `0.0.0.0` bind addresses (intended) and test IPs. Notable real: the two `172.27.1.x`
deploy hosts above; `StatData mem_store.go` sample `10.0.4.12`; `StatTrust mem_store.go:204` SourceIP
`197.239.4.18` (test).

---

## 5. Dummy / example / synthetic data
- **Example domains** are almost entirely **legit test fixtures** in `collect-master` (upstream ODK tests) —
  e.g. `ValidatorTest.kt` (55), `BulkFinalizationTest.kt` (18). Not defects.
- `builder-main/server/sensitive_data_test.go` — redaction-sensitive tests using `"example.com"`,
  `"your_password"` (DUMMY marker, correct in tests).
- `StatCollect/tests/load/*.sql` — synthetic survey load data with `@example.invalid` (by design).
- `StatTrust` DLP demo data `api_key = "sk_live_uganda_data_secret"` (test sample).

---

## 6. CRUD (List/Create → Get → Update → Delete) route coverage per service

Go backends use a consistent gin/mux pattern (`r.HandleFunc(path, Handler).Methods(...)`).
**Full CRUD = L/C/G/U/D on a resource.**

| Backend | Resource routes | Full CRUD? |
|---|---|---|
| `ai-autonomy` | twins, models, agents (L/C/G/U/D); pipelines (L/C); predictions (L/G) | ✅ (+ action suite) |
| `bpm-hub` | processes (L/C/G), process instances (POST/start) | ◐ |
| `enterprise` | documents, retention, demo A/D (L/C); **OCR = stub/mocked** | ◐ |
| `backend` (analytics core) | ingest, query, stats, indicators, anomalies, policies, datasets, assets, alerts, agents | ◐ mostly R/POST |
| `PMS` / `RMS` | users, projects, tasks (db_handlers.go) | ✅ |
| `StatSpatial` | admin-units, organizations (L/C/G/U/D); layers/features/nodes (L/C) | ✅ for admin-units/orgs |
| `StatCollect` | submissions CRUD + admin/submission/delete, templates (L/C/detail/clone/…) | ✅ |
| `helpdesk-master` | submissions, templates, attachment, admin (get/delete/validate) | ✅ |
| `StatChat` | rooms/collaboration (seeded SQL) + handlers | ✅ |
| `StatData/StatIoT/StatTrust/StatOps/StatGovernance/StatFederation` | service route sets + mem stores | per-service |

Coverage is **uneven**: `ai-autonomy` and `StatSpatial` are full-CRUD; several services expose only
READ (LIST/GET) + POST; `enterprise` ships OCR as a **mocked endpoint** rather than real processing.

---

## 7. Recommended actions (priority order)
1. **Remove hardcoded production secrets** from `k8s/00-namespace.yaml` (DB superuser password) → use
   Kubernetes Secrets + external vault; **rotate** the exposed value.
2. **Scrub real creds from working-tree `.env*`/`.env.example`** (replace with placeholders) and centralize
   real secrets in a vault; keep secret-to-env wiring.
3. **Parameterize the hardcoded service catalog**: replace `frontend/config/services.json`, appluncher
   `mockData.ts`, and duplicated `launcherApps.js` with env-driven discovery (ConfigMap / registry).
4. **Externalize deploy/build scripts**: pull `frank@172.27.1.x` hosts + SSH ports out of `package.json`
   into CI variables (helpdesk-master, stage_register).
5. **Finish the enterprise `OCR` stub** (canned text + `[OCR_PENDING]`) or gate behind a feature flag;
   return proper not-found for unknown graph queries in `phase12_graph_queries.go`.
6. **Centralize per-service defaults** (DB `5432`, Redis `6379`, `8080`) in `statgate-lib` config module.
7. **Housekeeping**: gitignore + remove `.next` build artifacts from `appluancher`; keep
   `collect-master` upstream updates clean.

### Appendix — how this was produced
- `Get-ChildItem` inventory of all 50 top-level dirs.
- `audit_scan.ps1` → per-dir regex + line + count JSON dumps under `.gemini/d_*.json`, aggregated into
  the tables above; `Select-String` used for the CRUD route inventory.
- Git: `git ls-files` used to separate tracked vs untracked sensitive files.

**Concrete placeholder / stubbed implementations (first-party):**
- `enterprise/core/phase15_edms.go:602` — **OCR Service Stub**; `handleSubmitOCR` echoes a hardcoded
  `[OCR_PENDING]` result and `handleGetOCRResult` returns a **canned OCR result with fabricated text**
  (`"National Statistical Data Governance Policy…"`, ConfidenceScore 94.7). Placeholder data.
- `enterprise/core/phase12_graph_queries.go:64` — unknown graph queries return
  `fmt.Errorf("query %q is not implemented")`.
- `StatChat/frontend/src/hooks/useCall.ts:18` — comment: `turn.example.com:3478` is a *placeholder that
  silently breaks calls* → dead/broken integration.