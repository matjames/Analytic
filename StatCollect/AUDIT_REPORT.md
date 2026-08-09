# StatCollect Audit Report

**Date:** 2026-08-06  
**Auditor:** Automated Code Review  
**Scope:** `StatCollect/` — StatGate Data Collection Adapter  
**Files Audited:** Core backend, migrations, configuration, templates  

---

## 1. Executive Summary

StatCollect is a **Go-based HTTP adapter** that accepts ODK-style multipart form submissions, persists them to PostgreSQL (with optional S3 storage), and integrates deeply into the **StatGate Enterprise Evidence Intelligence Platform** via:

- **Registry JWT** for unified identity
- **Redis pub/sub** event bus (`statgate:events`)
- **StatChat** object-linked discussion backbone
- **Cross-module** publishing to Research, Projects, Statistics, GIS, Reporting, Documents

The codebase is mature, feature-rich, and well-structured, but carries **significant security debt** (hardcoded secrets, weak defaults, unsafe JSON handling) and **technical debt** (single 2,000-line server file, minimal error context, no request validation middleware).

**Risk Rating:** **HIGH** (production-blocking security issues must be resolved before deployment)

---

## 2. Architecture Overview

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Runtime** | Go 1.21+ | Standard library `net/http` |
| **Database** | PostgreSQL via `pgx/v5` | Connection pooling, migrations auto-run |
| **Storage** | Local FS or S3-compatible | Pluggable `Store` interface |
| **Auth** | API keys + Registry JWT + Internal service key | Multi-layer but inconsistently enforced |
| **Eventing** | Redis pub/sub (`go-redis/v9`) | Optional, feature-flagged |
| **Integrations** | HTTP clients to StatChat, Registry, platform modules | Timeout-bounded, fire-and-forget goroutines |

### Request Flow

```
Client → /submission (multipart)
    ├─ Auth: API key / Internal key / Registry JWT
    ├─ Parse multipart (200 MB limit)
    ├─ Extract XML + attachments
    ├─ Idempotency check (instance_id)
    ├─ GPS enforcement (mandatory)
    ├─ QA flagging (field_video missing)
    ├─ Persist: S3/FS + PostgreSQL
    ├─ Publish: Redis event + StatChat link + platform webhooks
    └─ Response: 201 Created
```

---

## 3. Security Audit

### 3.1 Critical Findings

#### [CRITICAL] Hardcoded Default Secrets
**File:** `config.go:173`, `README.md:72`
```go
internalKey := os.Getenv("STATGATE_INTERNAL_API_KEY")
if internalKey == "" {
    internalKey = "REDACTED_PLACEHOLDER"
}
```
- **Impact:** Any deployment that fails to set the env var uses a publicly documented default. An attacker can call **every admin endpoint** (`/admin/*`).
- **Fix:** Remove the default. Return an error on startup if the variable is missing.

#### [CRITICAL] Hardcoded JWT Secret Fallback
**File:** `README.md:71`
```
STATGATE_REGISTRY_JWT_SECRET = REDACTED_PLACEHOLDER
```
- **Impact:** If the env var is missing, `registry.go` disables integration silently, **bypassing identity checks**. Submissions lose `submitted_by` provenance.
- **Fix:** Do not fall back. Fail startup if JWT secret is required but missing.

#### [HIGH] Admin Endpoint Authorization Bypass Risk
**File:** `server.go:1130`
```go
if !checkAdminKey(r) && !checkAPIKey(r) {
```
`/admin/submissions/export` allows **any API key** (not just admin). An enumerator with a basic API key can export all submissions.
- **Fix:** Require `checkAdminKey(r)` only.

#### [HIGH] Inconsistent Authorization Matrix
| Endpoint | Current Check | Expected |
|----------|--------------|----------|
| `/admin/submissions/export` | Admin **or** API key | Admin only |
| `/templates` (GET) | Any API key | Public or authenticated |
| `/templates/library` | None | Should be public (OK) |
| `/templates/import` | Admin key | OK |
| `/admin/events` | Admin key | OK |
| `/admin/objects/links` | Admin key | OK |

#### [MEDIUM] Unvalidated JSON in `SaveTemplate`
**File:** `server.go:839-843`
```go
var schemaCheck map[string]interface{}
if err := json.Unmarshal(t.Schema, &schemaCheck); err != nil {
    http.Error(w, "invalid schema: "+err.Error(), http.StatusBadRequest)
    return
}
```
Only checks that the JSON is parseable. Does not validate:
- `sections` array exists
- Each section has `id`, `title`, `fields`
- Each field has `key`, `label`, `type`
- No SQL injection vectors in field keys (though pgx parameterized queries mitigate this)

#### [MEDIUM] Default Database Credentials
**File:** `config.go:63-67`
```go
user := "statcollect"
pass := "Statgate"
```
Hardcoded defaults in source code. If the operator forgets to set `STATCOLLECT_DB_*`, the app connects with well-known credentials.
- **Fix:** Remove defaults. Require explicit configuration or fail.

#### [MEDIUM] Missing Rate Limiting & Request Size Validation
- `/submission` accepts up to **200 MB** (`r.ParseMultipartForm(200 << 20)`). No per-user rate limit.
- No timeout on the overall handler (only on Redis/HTTP client calls).
- **Impact:** Memory exhaustion, DoS.

#### [LOW] `defer file.Close()` Inside Loop
**File:** `server.go:434-439`
```go
for key, fhs := range mf.File {
    for _, fh := range fhs {
        f, err := fh.Open()
        // ...
        f.Close()
    }
}
```
Not using `defer` inside loops is correct here (would exhaust file descriptors), but the pattern is fragile. Document the intent.

### 3.2 Positive Security Patterns
- **Idempotency:** `instance_id` uniqueness prevents double-processing.
- **HMAC JWT validation:** `registry.go:54-59` correctly restricts to `SigningMethodHMAC`.
- **SQL parameterization:** All queries use `$1`, `$2` (pgx).
- **Context timeouts:** Every DB/Redis call uses `context.WithTimeout`.
- **Graceful shutdown:** Listens for `SIGTERM`, closes Redis and HTTP server.

---

## 4. Code Quality & Technical Debt

### 4.1 God Object: `server.go` (2,023 lines)
The entire HTTP layer lives in one file with 40+ handlers. This violates **Single Responsibility Principle**.

**Recommendation:** Split by domain:
```
internal/server/
  handlers/
    submission.go
    admin.go
    templates.go
    phase_y.go      // devices, assignments, etc.
  routes.go
```

### 4.2 Package-Level Globals
```go
var store Store
var cfg *Config
var dbPool *pgxpool.Pool
var eventBus *EventBus
var statChat *StatChatIntegration
var registry *RegistryIdentity
```
- **Impact:** Impossible to test in parallel; hidden dependencies; race conditions if `Run()` is ever called twice.
- **Fix:** Inject dependencies via a `Server` struct.

### 4.3 Error Handling: Silent Failures
Many DB functions return `nil` when `dbPool == nil`:
```go
func SaveSubmissionToDB(...) error {
    if dbPool == nil {
        return nil  // Swallows the error
    }
```
This makes "no database configured" silently succeed, corrupting idempotency and audit guarantees.

**Fix:** Return `fmt.Errorf("database not initialized")` and let the handler return 500.

### 4.4 Logging Quality
- Uses `log.Printf` everywhere (no structured logging).
- No request IDs, correlation IDs, or tenant context in logs.
- Sensitive data (XML payloads) are not logged, which is good.

**Fix:** Adopt `zap` or `logrus` with structured fields (`request_id`, `tenant_id`, `instance_id`).

### 4.5 Test Coverage
- **No unit tests** found in `internal/server/`.
- Only integration scripts (`tests/integration/run_integration.sh`) and sample XML files.
- The `tests/` directory contains questionnaire fixtures, not Go tests.

**Impact:** Refactoring is high-risk. Auth changes, DB schema changes, and handler logic changes have no safety net.

---

## 5. Database Design Review

### 5.1 Schema Strengths
- **JSONB** for `meta`, `schema`, `payload` — flexible and queryable.
- **Foreign keys** with `ON DELETE CASCADE` on attachments, validations, longitudinal links.
- **Indexes** on all foreign keys and high-cardinality filters (`tenant_id`, `status`, `form_id`).
- **Idempotency** guaranteed by `UNIQUE (instance_id)`.

### 5.2 Schema Concerns

#### [MEDIUM] `submissions.xml` as TEXT
```sql
xml TEXT
```
Full XML documents are stored as text in PostgreSQL. For large submissions with video metadata, this can bloat the DB.

**Fix:** Store raw XML in S3/MinIO and keep only a reference + parsed JSONB `meta` in Postgres.

#### [LOW] Missing `updated_at` Trigger
`submissions` has `updated_at` column but no auto-update trigger. The app sets it manually in a few places but misses it in others.

**Fix:**
```sql
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER submissions_updated_at BEFORE UPDATE ON submissions
FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

#### [LOW] `event_log.payload` Growth
Unbounded JSONB event log. In high-volume deployments, this table will grow without cleanup.

**Fix:** Add partition by `created_at` (monthly) or a TTL cleanup job.

### 5.3 Missing Tables vs. Code
`db.go` references `longitudinal_links`, `integrations`, `plugins`, `business_rules` — these exist in the migration. Good.

However, `db.go` does **not** expose CRUD for:
- `integrations`
- `plugins`
- `business_rules`
- `dashboard_subscriptions`
- `stats_exports`
- `gis_sync_queue`
- `document_links`
- `platform_publish_log`

These tables are defined in SQL but **unreachable via API**. Either they are placeholders for future work, or the API is incomplete.

---

## 6. Integration Design

### 6.1 Event Bus (`events.go`)
- **Pattern:** Fire-and-forget `Publish()` with 3-second timeout.
- **Issue:** No retry, no dead-letter queue. If Redis is down, the event is lost silently.
- **Issue:** `Publish()` logs errors but does not return them to the caller. Submission still returns 201 even if event bus fails.

**Fix:** At minimum, expose a health indicator (`eventBus.Health()`) and surface failures in `fullHealthHandler`.

### 6.2 StatChat (`statchat.go`)
- Uses `X-API-Key` header with `cfg.InternalAPIKey` for service-to-service auth.
- **Issue:** `urlQueryEscape` is reimplemented instead of using `net/url`. Risk of divergence from standard behavior.
- **Fix:** Use `url.QueryEscape`.

### 6.3 Registry (`registry.go`)
- JWT parsing is local (no remote introspection by default).
- `verifyWithRegistry` exists but is **never called**. It should be the primary path; local parsing should be a cache/fallback.
- **Issue:** `IdentityFromToken` tries multiple claim keys (`sub`, `userId`, `user_id`) — suggests inconsistent JWT producers in the ecosystem.

---

## 7. Business Logic Observations

### 7.1 GPS Enforcement
**File:** `server.go:310-320`
```go
hasGPS := strings.Contains(xmlStr, "<gps_coordinates>") &&
    !strings.Contains(xmlStr, "<gps_coordinates></gps_coordinates>")
```
- **Brittle:** Relies on XML string matching. A renamed field (`<location>`) bypasses enforcement.
- **Fix:** Parse XML properly (`encoding/xml`) and check for non-empty element content.

### 7.2 QA Flagging
- `submission.qa_flag` is logged but **not persisted** in the `submissions` table. Supervisors have no way to query "all flagged submissions" without scanning the event log.
- **Fix:** Add a `qa_flags JSONB` column to `submissions` or a dedicated `qa_flags` table.

### 7.3 Sampling Algorithm
**File:** `server.go:1632-1653`
```go
currentSeed = (currentSeed*1103515245 + 12345) & 0x7fffffff
idx := int(currentSeed) % len(frame)
```
- Uses a custom LCG. Not cryptographically secure (not needed here), but **not statistically uniform** for large frames.
- **Fix:** Use `math/rand.NewSource(seed)` or `crypto/rand` for better distribution.

### 7.4 Template MSH Injection
**File:** `templates.go:86-98`
`SaveTemplate` auto-injects the Mandatory Survey Header (MSH) into the schema JSON on every save. This mutates user-authored schemas transparently.

- **Risk:** If `InjectMSHIntoSchema` has a bug, it corrupts the template silently.
- **Fix:** Inject MSH at render time in the frontend, not at save time in the backend.

---

## 8. Configuration & Deployment

### 8.1 `run.bat`
Sets environment variables and runs `go run .`. No health check wait loop. If PostgreSQL is not ready, the app crashes on startup.

**Fix:** Add retry logic in `InitDB` (already has 5s timeout, but no retries).

### 8.2 `docker-compose.yml`
Joins `statgate-network`. No resource limits, no healthcheck on the StatCollect container.

**Fix:** Add:
```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
  interval: 10s
  timeout: 3s
  retries: 3
```

---

## 9. Compliance & Audit Trail

### 9.1 What Is Well Covered
- `event_log` captures cross-module events.
- `submission_validations` records approval/rejection with validator identity.
- `platform_publish_log` (defined in SQL) would track external pushes.

### 9.2 Gaps
- **No `submitted_by` on attachments.** Only the submission header has identity.
- **No content hash** on submissions. Cannot prove data wasn't tampered with post-ingest.
- **No retention policy.** Event logs and attachments grow indefinitely.

---

## 10. Prioritized Recommendations

### P0 — Block Production Deployment
1. **Remove hardcoded secrets** (`config.go:173`, `README.md:72`). Fail startup if missing.
2. **Restrict `/admin/submissions/export`** to admin keys only.
3. **Return error when DB is nil** instead of silent no-op.
4. **Add rate limiting** on `/submission` (e.g., token bucket per API key).

### P1 — High Priority
5. **Split `server.go`** into domain-specific handler files.
6. **Replace package-level globals** with a `Server` struct and dependency injection.
7. **Add structured logging** (request ID, tenant, instance_id).
8. **Implement XML parsing** for GPS enforcement instead of string matching.
9. **Persist QA flags** in the database, not just event log.

### P2 — Medium Priority
10. **Add retry + backoff** for Redis and StatChat/Registry HTTP calls.
11. **Move MSH injection** to frontend or API response layer.
12. **Add database triggers** for `updated_at`.
13. **Expose CRUD APIs** for `integrations`, `plugins`, `business_rules` if they are intended for use.
14. **Add partition strategy** for `event_log` and `platform_publish_log`.

### P3 — Low Priority / Nice-to-Have
15. Replace custom `urlQueryEscape` with `net/url`.
16. Add unit tests for `extractXMLSubmission`, `instanceIDFromXML`, `formIDFromXML`, `mapTypeToXLS`.
17. Add OpenAPI spec validation middleware.
18. Add `request.Context()` propagation to all downstream calls.

---

## 11. Strengths

- **Comprehensive enterprise schema** covering devices, assignments, registries, sampling, workflows, schedules, comments, notifications.
- **Strong StatGate platform integration** design (event bus, unified identity, StatChat).
- **Idempotent submission ingestion** via `instance_id`.
- **Multi-tenant awareness** in most queries.
- **Flexible template system** with import/export to XLSForm and ODK XML.
- **Observability** via Prometheus metrics (`submissionDuration`, counters).

---

## 12. Conclusion

StatCollect is a **capable, platform-native adapter** with a clear vision for enterprise survey data collection. However, it carries **critical security vulnerabilities** (hardcoded secrets, overly permissive export endpoint) that must be resolved before any production exposure. The codebase would benefit significantly from structural refactoring (dependency injection, file splitting) and a test suite to support future evolution.

**Next Step:** Address all P0 findings, then schedule P1 refactoring in the next sprint.