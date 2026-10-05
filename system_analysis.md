# StatCitizen / StatGate System — Full Code Analysis

> Focused exclusively on actual source code. MD docs excluded.

---

## 🔬 Test Run Results

```
9/9 PASS  (2.305s)  — github.com/statgate/statcitizen
```

| Test | Result |
|------|--------|
| TestCitizenEventUsesCanonicalDurableEnvelope | ✅ PASS |
| TestHealthAndReadiness | ✅ PASS |
| TestCitizenSessionCreation | ✅ PASS |
| TestExplicitConsentRequirement | ✅ PASS |
| TestPIIHashingAndMinimization | ✅ PASS |
| TestServiceRatingDimensions | ✅ PASS |
| TestGovernedAIQueryInsufficientEvidenceFallback | ✅ PASS |
| TestUniversalObjectContextStructure | ✅ PASS |
| TestOfflineDraftQueuing | ✅ PASS |

---

## 🏗️ Architecture Overview

The system is a **multi-service StatGate Sovereign Intelligence Platform**.

```
Analytic/
├── StatCitizen/          ← Citizen participation platform (Go, Port 8115)
│   ├── backend/          ← Go 1.22, Gin, PostgreSQL, Redis (9 source files)
│   └── frontend/         ← Vanilla HTML/CSS/JS (3 files, ~92KB total)
├── backend/              ← Main enterprise platform (Go, internal/ sub-packages)
├── frontend/             ← Python Flask app (app.py 68KB, agentic_engine.py 20KB)
├── statgate-lib/         ← Shared Go library (events, auth, audit, database, etc.)
├── StatChat/             ← Chat service
├── StatCollect/          ← Survey/collection service
├── StatData/             ← Data analytics service
├── StatFederation/       ← Federation service
├── StatGovernance/       ← Governance service
├── StatIoT/              ← IoT bridge
├── StatOps/              ← Operations service
├── StatSpatial/          ← Spatial/GIS service
├── StatTrust/            ← Trust service
├── PMS/                  ← Project Management System
├── RMS/                  ← Registry Management System
└── docker-compose.yml    ← 71KB, orchestrates the entire platform
```

---

## 📦 StatCitizen — Detailed Code Analysis

### Stack
- **Language**: Go 1.22
- **HTTP Framework**: gin-gonic/gin v1.10
- **Database**: PostgreSQL (lib/pq), pool of 20 max connections
- **Event Bus**: Redis (go-redis/v9) via shared `statgate-lib/events`
- **Auth**: JWT (golang-jwt/jwt/v5) for enterprise admin, custom HMAC session tokens for citizens
- **Dependencies**: godotenv, gin-cors

### Source Files

| File | Lines | Purpose |
|------|-------|---------|
| `main.go` | 115 | Server init, middleware chain, graceful shutdown |
| `config.go` | 166 | Environment-driven config, prod secret validation |
| `models.go` | 429 | All domain models (15 structs) |
| `persistence.go` | 572 | DB init, migrations (2), all schema DDL |
| `routes.go` | 741 | Route registration + 14 inline handlers |
| `citizen_identity.go` | 993 | All citizen-facing handlers (session, feedback, reports, cases, rating) |
| `consultations.go` | 545 | Consultation CRUD, response submission, service ratings |
| `middleware.go` | 424 | Rate limiting, JWT, session, CORS, security headers, audit |
| `ai.go` | 231 | Governed AI assistant (evidence-grounded, citation-based) |
| `events.go` | 238 | Redis event bus, dead-letter queue, retry worker |
| `integrations.go` | 229 | Enterprise fabric registration, HelpDesk integration, Universal Context |
| `workers.go` | 59 | Offline sync worker (60s ticker) |
| `statcitizen_test.go` | 281 | 8 integration tests |
| `durable_events_test.go` | 13 | 1 event envelope test |

---

## ✅ What Is Fully Working (Verified by Tests)

### 1. Anonymous Citizen Sessions
- `POST /api/statcitizen/v1/session` → 201 Created
- Generates 32-byte cryptographically secure token (base64)
- Hashes IP with HMAC-SHA256 (never stored raw)
- 24-hour TTL, persisted to `citizen_sessions` table
- Async last_seen update on each lookup

### 2. Privacy-by-Design — PII Handling
- Email and phone are **never stored raw** — always HMAC-SHA256 hashed
- Normalization (lowercase + trim) before hashing → consistent identity matching
- `consent_granted: false` → **hard 400 reject** at every endpoint
- Verified by `TestPIIHashingAndMinimization` and `TestExplicitConsentRequirement`

### 3. Governed AI Assistant
- Evidence-grounded: only reads from `public_publications` (status=`published`)
- No hallucination: returns `insufficient_evidence: true` with zero confidence when no public records match
- Citations included with canonical IDs
- Query for confidential/internal data → fallback triggered, tested by `TestGovernedAIQueryInsufficientEvidenceFallback`

### 4. Universal Object Context (8-Facet View)
- `GET /api/statcitizen/v1/universal/context/:canonical_id`
- Returns: Overview, Activity, Relationships, Documents, Workflow, Decisions, AIIntelligence, Audit
- Pulls audit trail from DB if available, returns populated struct otherwise
- Tested by `TestUniversalObjectContextStructure`

### 5. Offline Draft Queuing
- `POST /api/statcitizen/v1/offline/drafts` → 202 Accepted
- Stores to `offline_drafts` table, returns `draft_id` + `correlation_id`
- Background sync worker runs every 60s, processes up to 10 drafts at a time (max 5 retries)
- Tested by `TestOfflineDraftQueuing`

### 6. Service Rating Dimensions
- Returns 7 configurable dimensions (satisfaction, accessibility, wait_time, availability, staff, quality, outcome)
- Falls back to hardcoded defaults if DB unavailable
- Tested by `TestServiceRatingDimensions`

### 7. Event Bus with Dead Letter Queue
- Publishes to Redis `statgate:events` channel via `statgate-lib` shared library
- On Redis failure → writes to `event_dead_letter` PostgreSQL table
- Retry worker (60s) re-attempts pending DLQ events with `retry_count < 10`
- Canonical event envelope: `{EventID, EventType, Source, ObjectType, ObjectID, UserID, TenantID, CorrelationID, Payload, Timestamp, Version}`
- Tested by `TestCitizenEventUsesCanonicalDurableEnvelope`

### 8. Database Schema (migration 001 + 002)
Complete schema with:
- `citizen_sessions`, `registered_citizens`, `citizen_consents`
- `feedback_categories` (7 seeded), `feedback_records`
- `report_categories` (7 seeded), `citizen_reports`
- `attachment_records`, `consultations`, `consultation_questions`, `consultation_responses`
- `rating_dimensions` (7 seeded), `service_ratings`
- `public_publications`, `citizen_cases`
- `geo_locations`, `citizen_audit_log`
- `event_dead_letter`, `offline_drafts`, `admin_settings`
- All tables properly indexed

---

## ⚠️ Issues Found in Code

### 🔴 Critical

1. **Broken JSON serialization in `consultations.go`** (Lines 469–494)
   ```go
   func answerJSON(data interface{}) (string, error) {
       import_json_b, err := marshalJSON(data)
       return import_json_b, err
   }
   func marshalJSON(v interface{}) (string, error) {
       import_json := jsonMarshal(v)
       return import_json, nil
   }
   func jsonMarshal(v interface{}) string {
       import_json_b, _ := import_json_marshal(v)
       return import_json_b
   }
   func import_json_marshal(v interface{}) (string, error) {
       // Custom formatter — does NOT use encoding/json
       ...
   }
   ```
   - **Problem**: This custom JSON serializer (`formatJSONMap`) does not escape special characters (quotes, backslashes, newlines) in string values. Inserting feedback with `"` or `\n` in answers will produce malformed JSONB that PostgreSQL will **reject** or silently corrupt.
   - Should use `encoding/json.Marshal` instead.

2. **`handleSubmitConsultationResponse` variable name conflict** (line 205):
   ```go
   import_json_b, _ := answerJSON(req.Answers)
   answers = import_json_b
   ```
   This works but the variable is named `import_json_b` which is a leftover from the broken marshaller chain above. Compiles but fragile.

3. **`handleListFeedback` — count query doesn't respect pagination context** (line 347):
   The total count query for feedback runs on the same `ctx` context that was used for the row query. If rows are slow, the count query's timeout is already partially consumed.

### 🟡 Significant

4. **Rate limiter is in-memory only** — `rateLimitBuckets` map is not distributed
   - Rate limits are per-process, per-IP. With multiple replicas, each instance has its own bucket → effective limit becomes `N × RPM`
   - For single-instance deployments this is fine; multi-replica needs Redis-backed rate limiting.

5. **`workers.go` offline sync doesn't actually re-process the payload**
   ```go
   // Mark as completed once processed
   _, _ = dbPool.ExecContext(uCtx,
       `UPDATE offline_drafts SET status='completed', updated_at=NOW() WHERE id=$1`, d.ID)
   log.Printf("offline_sync: processed offline draft %s (%s)", d.ID, d.DraftType)
   ```
   - The worker marks drafts as `completed` without actually re-submitting them to any handler. The offline sync feature is **incomplete** — it accepts drafts but doesn't process them into real submissions.

6. **`handleVerifyContact` OTP validation is a stub** (line 232):
   ```go
   verified := req.Code != "" && len(req.Code) >= 4
   ```
   - Any 4+ character code is accepted. Real OTP validation is not implemented.

7. **`handleGetAdminAnalytics` has a hardcoded `ratings_avg: 4.2`** (line 674)
   - Not computed from actual data.

8. **`handleCreateConsultation` question options are not stored** (lines 291–297):
   ```go
   optJSON, _ := answerJSON(map[string]interface{}{"opts": q.Options})
   _, _ = dbPool.ExecContext(ctx,
       `INSERT INTO consultation_questions (id,...,type,required,sort_order)
       VALUES($1,$2,$3,$4,$5,$6,$7)`,
       ...)
   _ = optJSON  // options are computed but never inserted!
   ```
   - The INSERT does not include the `options` column — consultation questions will always have empty options arrays.

9. **`handleUpdateCaseStatus` has unused variable** (line 830):
   ```go
   var result string
   ...
   _ = result
   ```
   Minor but indicates incomplete or copy-paste code.

10. **`nowUTC()` function exists in `citizen_identity.go` but `time.Now().UTC().Format(time.RFC3339)` is also used inline throughout** — minor inconsistency.

### 🟢 Minor / Code Quality

11. **Line 992 of `citizen_identity.go`**:
    ```go
    var _ = strings.Contains
    ```
    Dead code to suppress import — the `strings` import should be used properly or removed.

12. **`handleListFeedback` fetches `citizenID` into a variable (line 311) that is never stored back to the struct** — the scan creates a `*string` for `citizenID` but the column isn't in the SELECT and `fb.CitizenID` is never set from it.

---

## 🔒 Security Assessment

| Area | Status |
|------|--------|
| PII Hashing (email, phone, IP) | ✅ Done correctly |
| JWT admin validation | ✅ HMAC-SHA256 verification |
| Session token (32-byte random) | ✅ Cryptographically secure |
| Consent enforcement at every endpoint | ✅ Hard reject without consent |
| SQL Injection | ✅ Parameterized queries throughout |
| Security headers (CSP, HSTS, X-Frame-Options) | ✅ Applied globally |
| Rate limiting | ✅ Token-bucket per IP (in-memory) |
| Request size limiting | ✅ Configurable (default 10MB) |
| File upload size limiting | ✅ 10MB attachment cap |
| Production secret validation | ✅ Hard fatal if missing |
| Dev mode bypass (JWT secret empty) | ⚠️ Intentional but documented |
| OTP verification | ❌ Stub — any 4-char code accepted |
| CAPTCHA | ❌ Config exists but not enforced |
| Distributed rate limiting | ❌ In-memory only |

---

## 📡 Integration Points

| Integration | How | Status |
|-------------|-----|--------|
| Enterprise Core (`localhost:8096`) | HTTP POST `/api/registry/services` + 60s heartbeat | ✅ Implemented |
| HelpDesk (`localhost:5006`) | Async goroutine on report submit | ✅ Implemented |
| Redis Event Bus | `statgate-lib/events` shared library | ✅ Implemented + DLQ |
| StatCollect surveys | DB join on `statcollect_form_id` | ✅ Partial (lists surveys linked to consultations) |
| PMS / RMS / Governance | Config URLs present | ❌ Not called yet |
| Enterprise Search | Config URL present | ❌ Not called yet |

---

## 📋 Feature Completion Status

| Feature | Backend | Frontend | Tests |
|---------|---------|----------|-------|
| Anonymous citizen sessions | ✅ | ✅ | ✅ |
| Citizen registration + consent | ✅ | ✅ | ✅ |
| Phone/email verification | ⚠️ Stub | ✅ UI | ❌ |
| Feedback submission | ✅ | ✅ | ❌ |
| Reports with GPS (consent-gated) | ✅ | ✅ | ❌ |
| File attachments | ✅ (10MB) | ✅ | ❌ |
| Public consultations | ✅ | ✅ | ❌ |
| Consultation responses | ⚠️ Options bug | ✅ | ❌ |
| Service ratings (multi-dimension) | ✅ | ✅ | ✅ |
| Case tracking / closed-loop | ✅ | ✅ | ❌ |
| Public publications | ✅ | ✅ | ❌ |
| Governed AI assistant | ✅ | ✅ | ✅ |
| Offline draft queuing | ✅ | ✅ | ✅ |
| Offline draft re-processing | ❌ Incomplete | N/A | ❌ |
| Admin analytics | ⚠️ Hardcoded avg | ✅ | ❌ |
| Universal Object Context | ✅ | ✅ | ✅ |
| Event bus + DLQ | ✅ | N/A | ✅ |
| Enterprise fabric registration | ✅ | N/A | ❌ |
| HelpDesk ticket creation | ✅ | N/A | ❌ |

---

## 🎯 Priority Fixes Needed

1. **[P0] Fix JSON marshaller** in `consultations.go` — replace custom `formatJSONMap` chain with `encoding/json.Marshal`
2. **[P0] Fix consultation question options INSERT** — add `options` column to the INSERT statement
3. **[P1] Complete offline sync worker** — worker must actually re-dispatch draft payloads to handlers
4. **[P1] Implement real OTP verification** — integrate SMS/email OTP provider
5. **[P1] Real admin analytics** — compute ratings average from DB, not hardcoded 4.2
6. **[P2] Distributed rate limiting** — move to Redis-backed rate limiter for multi-replica support
7. **[P2] Enable CAPTCHA enforcement** — config exists but middleware never checks it
8. **[P2] Add integration tests** for feedback, reports, consultations, and case tracking
