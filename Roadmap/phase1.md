# PHASE 1 — PROJECT FOUNDATION & DEVELOPMENT ENVIRONMENT

## Objective

Establish the engineering foundation upon which every module of StatGate is built.

This phase covers repository organisation, developer environment, coding standards, CI/CD, Docker infrastructure, monitoring foundations, security baseline, and project governance.

Nothing else shall be developed before this phase is complete.

---

## Vision

StatGate is an **Enterprise Evidence Intelligence Platform** — not a collection of modules. Every decision made in this phase must support a single, unified platform where organisations collect, integrate, govern, analyse, visualise, communicate, and transform data into trusted evidence.

---

## Tech Stack (Adopted)

| Layer | Technology |
|---|---|
| Primary backend language | Go (1.22+) |
| Go web framework | Gin |
| Frontend (analytical workspace) | Python / Flask + Jinja2 templates |
| Frontend (module UIs) | React 18 + Vite + TypeScript |
| Central portal | Next.js 14 |
| Database | PostgreSQL 15 |
| Cache / event bus | Redis 7 |
| Object storage | MinIO |
| Containerisation | Docker + Docker Compose |
| Monitoring | Prometheus + Grafana + Loki + Alertmanager |
| CI/CD | GitHub Actions |

---

## Repository Structure (Adopted)

```
Analytic/                         ← workspace root
├── backend/                      ← Go analytics core (:8082)
│   ├── cmd/server/
│   └── internal/
│       ├── abac/                 ← ABAC engine
│       ├── assets/               ← asset manager
│       ├── lakehouse/            ← in-memory lakehouse + anomaly worker
│       └── semantic/             ← semantic indicator registry
├── frontend/                     ← Flask analytics UI (:5000)
│   ├── app.py                    ← main application + all routes
│   ├── engine.py                 ← Pandas/DuckDB analysis engine
│   ├── agentic_engine.py         ← rule-based automated reports
│   ├── schema_healer.py          ← AI-assisted schema health
│   └── kaggle_connector.py       ← Kaggle dataset import
├── appluancher/                  ← Next.js launcher portal (:3006)
├── stage_register/               ← Field Operations Registry
│   ├── go-backend/               ← Go API (:9090)
│   └── frontend/                 ← React UI (:3007)
├── helpdesk-master/              ← Operations Helpdesk
│   ├── backend/                  ← Node/Express API (:5006)
│   └── frontend/                 ← React UI (:3005)
├── StatChat/                     ← Collaboration platform
│   ├── backend/                  ← Go API + WebSockets (:4000)
│   └── frontend/                 ← React/Vite UI (:3009)
├── PMS/                          ← Project Management System
│   ├── backend/                  ← Go API (:8091)
│   └── frontend/                 ← React/Vite UI (:3010)
├── RMS/                          ← Research Management System
│   ├── backend/                  ← Go API (:8092)
│   └── frontend/                 ← React/Vite UI (:3011)
├── StatGovernance/               ← Governance & Compliance
│   ├── backend/                  ← Go API (:8093)
│   └── frontend/                 ← React/Vite UI (:3012)
├── StatSpatial/                  ← GIS Platform
│   └── backend/                  ← Go API (:8094) — stub
├── StatCollect/                  ← Field data collection
│   └── (Go ODK adapter :8080)
├── collect-master/               ← Android field capture app (ODK fork)
├── enterprise/
│   ├── core/                     ← Enterprise integration layer (:8096)
│   ├── search/                   ← Cross-app search (:8095)
│   └── design-system/            ← Shared UI component library
├── monitoring/                   ← Prometheus / Grafana / Loki / Alertmanager
├── docker/                       ← PostgreSQL bootstrap DDL (11 SQL scripts)
├── docker-compose.yml            ← Full platform compose file
├── .env.example                  ← Single config source of truth
└── README.md
```

---

## Standard Service Pattern

Every Go backend service in StatGate follows this pattern:

```go
// main.go
func main() {
    _ = godotenv.Load("../../.env")    // load shared config
    // ... production secret validation (fail-fast)
    if err := InitDB(dsn); err != nil { ... }
    if err := initRedis(); err != nil { ... }

    r := gin.Default()
    // Prometheus metrics middleware
    r.GET("/metrics", gin.WrapH(promhttp.Handler()))
    // Health + readiness endpoints
    r.GET("/health", handleHealth)
    r.GET("/ready",  handleReady)
    // CORS (platform origins only)
    r.Use(cors.New(corsConfig))
    // Registry JWT auth middleware
    r.Use(registryAuthMiddleware())
    RegisterRoutes(r)
    r.Run(":" + port)
}
```

Every service must expose:
- `GET /health` — JSON health response (service, version, db status, uptime, redis)
- `GET /ready`  — 503 if database unavailable
- `GET /metrics` — Prometheus metrics
- Registry JWT validation on all `POST`, `PUT`, `DELETE` routes
- Prometheus request counter + latency histogram

---

## Database Pattern

All databases use PostgreSQL 15. Each service owns its own database and schema:

| Service | Database | Schema |
|---|---|---|
| Analytics Core | `statgate` | `public` / `ml_staging` |
| Field Registry | `statgate` | `public` |
| StatChat | `statchat` | `public` |
| PMS | `pms` | `pms` |
| RMS | `rms` | `rms` |
| StatGovernance | `statgovernance` | `public` |
| Enterprise Core | `statgate` | `enterprise` |
| StatCollect | (submissions in Redis / `statgate`) | — |

All schema migrations are performed in `database.go` using `migrateDB()` — inline SQL with `CREATE TABLE IF NOT EXISTS`. No external migration tool is used.

---

## Identity & Authentication (Adopted Pattern)

All services share a single JWT secret: `STATGATE_REGISTRY_JWT_SECRET`.

The **Field Operations Registry** (`stage_register/go-backend`, :9090) is the **JWT signing authority**. It signs tokens on login. Every other service **validates** those tokens using the shared secret.

```go
// registryAuthMiddleware() in each service
func registryAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractBearerToken(c)
        claims, err := validateJWT(token, os.Getenv("STATGATE_REGISTRY_JWT_SECRET"))
        if err != nil { c.AbortWithStatus(401); return }
        c.Set("user_id",   claims.UserID)
        c.Set("tenant_id", claims.TenantID)
        c.Next()
    }
}
```

**Production rule:** If `STATGATE_REGISTRY_JWT_SECRET` is not set, the service must refuse to start or return 401 on every protected request — never skip authentication.

---

## Standard API Route Structure

Every service organises routes as:

```go
func RegisterRoutes(r *gin.Engine) {
    r.Use(registryAuthMiddleware())
    api := r.Group("/api")
    {
        // ─── Primary resources ──────────────────────────
        // ─── Sub-resources ──────────────────────────────
        // ─── Workflow & Permissions ─────────────────────
        // ─── Dashboard / Analytics ──────────────────────
        // ─── Search ─────────────────────────────────────
        // ─── Enterprise Activity Timeline ───────────────
        // ─── AI Analysis ────────────────────────────────
    }
}
```

Every service implements:

```
GET  /api/dashboard          ← service-specific summary dashboard
GET  /api/search?q=          ← full-text search within service data
GET  /api/activity           ← enterprise activity timeline feed
```

---

## Integration Patterns (Adopted)

### Event Bus
All services publish cross-module events to Redis channel `statgate:events`:
```json
{
  "type":       "submission.received",
  "source":     "statcollect",
  "entity":     "submission",
  "entity_id":  "uuid",
  "tenant_id":  "uuid",
  "timestamp":  "ISO8601"
}
```

### Object Linkage
A shared `object_links` table connects entities across modules:
```sql
CREATE TABLE object_links (
    id          VARCHAR(36) PRIMARY KEY,
    source_app  VARCHAR(100),
    source_type VARCHAR(100),
    source_id   VARCHAR(36),
    target_app  VARCHAR(100),
    target_type VARCHAR(100),
    target_id   VARCHAR(36),
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Collaboration Integration
StatChat creates conversation threads for platform objects automatically:
`obj:<module>:<entity>:<id>` (e.g. `obj:pms:project:uuid`)

---

## What Exists ✅

- Go backend skeleton (Gin, middleware, routing, graceful shutdown, health endpoints)
- Flask analytics UI (`app.py`, blueprints, auth proxy, templates)
- PostgreSQL + Redis wiring via Docker Compose
- Full Docker Compose platform (`docker-compose.yml`, 11 bootstrap SQL scripts)
- MinIO object storage configured
- Prometheus + Grafana + Loki + Alertmanager under `monitoring/`
- App Launcher Next.js portal (`appluancher/`, :3006)
- GitHub Actions CI pipeline (`.github/workflows/`)
- `.env.example` as single config source of truth
- `README.md`, `SECURITY.md`, `DEPLOYMENT.md`
- `STATGATE_ENGINEERING_DIRECTIVE.md` — platform vision document

## What is Missing ❌

- `PROJECT_PROGRESS.md` — current phase, sprint, blockers, risks
- `SYSTEM_COMPLETION.md` — % completion per module
- `CHANGELOG.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`
- `README_ARCHITECTURE.md`, `README_AI.md`, `README_DATA_GOVERNANCE.md`
- `API_GUIDELINES.md`, `DATABASE_GUIDELINES.md`, `UI_GUIDELINES.md`, `DEVELOPER_GUIDE.md`
- Committed secrets rotated out of git history (`secrets/`, `.env` files)
- Committed `.exe` binaries removed from git
- `frontend/Lib` and `frontend/.venv.bak` removed from tracking

---

## Acceptance Criteria

Phase 1 is complete only if:

- [x] All Go modules build and test with `go build ./...` / `go test ./...` (24 modules verified 2026-08-21)
- [ ] Docker Compose starts all services successfully (`docker compose up`)
- [ ] `GET /health` responds 200 on every Go service
- [x] Flask UI loads at `http://localhost:5000`
- [x] App Launcher loads at `http://localhost:3006`
- [x] PostgreSQL connects and every bootstrap script succeeds on a clean PostgreSQL 15 volume
- [ ] Redis connects on all services that require it
- [x] All currently configured Prometheus targets are up (Core, Analytics, Registry, PMS, StatChat, Prometheus)
- [ ] GitHub Actions CI pipeline executes and passes
- [x] No secrets committed in git (`.env`, `secrets/`) in the current index
- [x] No compiled binaries committed (`.exe`) in the current index
- [x] `PROJECT_PROGRESS.md` and `SYSTEM_COMPLETION.md` exist and are maintained

### Verification snapshot — 2026-08-21

- Running and healthy: PostgreSQL, authenticated Redis, Analytics Core/UI, Registry API/UI, StatChat API/UI, PMS API/UI, RMS API/UI, Governance API/UI, Prometheus, Grafana, and Alertmanager.
- The StatGate Next.js launcher is available at `http://localhost:3006`; an independently healthy Compose preview remains available at `http://localhost:3106`.
- Python tests pass in the supported container runtime: 77 passed. All audited module frontends produce production builds.
- Clean PostgreSQL initialization passes all bootstrap scripts from an empty disposable volume.
- PMS, RMS, and Governance readiness probes now report authenticated Redis event-bus connectivity explicitly (`redis: true`).
- Expansion-stack builds uncovered and fixed invalid shared-library Docker contexts in StatFederation, StatOps, and StatTrust. Deployment of that batch is currently blocked by repeated external registry/Go-proxy download resets; their local Go test suites pass.
- Remaining gate: start and health-certify the entire 41-service Compose inventory, verify Redis/event integration for every applicable service, and obtain a passing GitHub Actions run.

---

## Estimated Duration

4–6 weeks

## Milestone

StatGate possesses a production-grade, security-clean engineering foundation capable of supporting all subsequent modules without architectural restructuring.
