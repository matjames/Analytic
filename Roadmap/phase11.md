# PHASE 11 — FIELD OPERATIONS, MOBILE ECOSYSTEM & DIGITAL DATA COLLECTION

## Objective

Develop the complete field operations ecosystem supporting online and offline data collection, supervisor consoles, quality assurance, workforce management, and synchronization across field teams.

This phase enables StatGate to become the operational platform for enumerators, supervisors, field officers, inspectors, researchers, and monitoring teams working in connected and disconnected environments.

---

## Vision

Field teams should perform every activity from mobile and web devices, even without internet connectivity. Data synchronization shall be secure, resilient, idempotent, and near real-time once connectivity is restored.

---

## Services & Apps Involved

| Component | Technology | Port / Platform |
|---|---|---|
| **StatCollect API** | Go + Gin | :8080 |
| **collect-master** | Android (Kotlin/Java, ODK Collect v8.6 Fork) | Mobile App |
| **Supervisor / Field Console** | React 18 + Vite + TypeScript (or Admin UI) | :8080 (embedded admin) / Web UI |
| **Enterprise Core Sync** | Go + Gin + Redis Event Bus | :8096 / Redis :6379 |

---

## Service: StatCollect & Field Backend

**Repository:** `StatCollect/`  
**Backend:** Go + Gin (`:8080`)  
**Database:** PostgreSQL (`statgate` / `statcollect`), Redis Event Bus (`statgate:events`)  
**Auth:** Registry JWT + ODK OpenRosa Basic Auth for mobile agents

---

## What Exists ✅

### 1. StatCollect Submission Engine (`StatCollect/`)
- **ODK OpenRosa Form & Submission Protocol:**
  - `GET  /formList` — Serves XML forms list to mobile devices
  - `GET  /manifest` — Manifest for media and form attachments
  - `POST /submission` — Receives ODK multipart form payloads, GPS metadata, and binary media (photos, audio)
- **Submission Management REST API:**
  - `GET    /submissions` — List all submissions with query filters
  - `GET    /submissions/:id` — Full submission payload inspection
  - `GET    /submissions/:id/attachments` — Download attached media
  - `PUT    /submissions/:id/status` — Mark status (`received`, `validated`, `rejected`)
  - `DELETE /submissions/:id` — Delete submission
- **Admin Dashboard UI:** Embedded HTML admin interface for inspecting incoming field records
- **Redis Event Publishing:** Emits `submission.received`, `submission.validated`, `submission.rejected` to `statgate:events`
- **Cross-Module Linkage:** Automatically generates `object_links` connecting submissions to PMS activities and RMS research projects

### 2. collect-master Mobile App (`collect-master/`)
- Upstream ODK Collect v8.6 fork with full Gradle build pipeline
- Offline form caching and background store-and-forward queue
- Hardware sensor integrations: GPS coordinate capture, camera, microphone, barcode/QR scanner
- Field encryption and submission batching

---

## What is Missing ❌

### 1. Supervisor & Workforce Management
- **Enumerator Registry & Device Pairing:** No server-side device enrollment, device heartbeat, or IMEI/hardware identifier registry
- **Supervisor Review Console:** Need structured per-question review, comment injection, and field correction workflows
- **Assignment & Workload Dispatcher:**
  ```
  GET    /api/field/assignments                ← list field assignments
  POST   /api/field/assignments                ← assign EA / forms to enumerator
  PUT    /api/field/assignments/:id/reassign   ← reassign to different officer
  GET    /api/field/enumerators/active         ← live active enumerators
  ```
- **Geofencing & Route Planning:** Validation that submissions occurred within assigned Enumeration Area (EA) polygons

### 2. Form Management & Questionnaire Studio
- **Dynamic XForm Builder API:** Currently requires external XForm XML compilation; needs visual questionnaire builder
- **Question Library Integration:** Ability to pull standard indicator questions from Phase 6 Question Bank

### 3. Quality Assurance & AI Validation
- **GPS Anomaly Detection:** Flag impossible travel speeds or spoofed coordinates
- **Automated Duplicate & Fraud Detection:** Audio duration check, photo similarity check, velocity checks
- **Submission Conflict Resolution Service:** Deterministic handling of concurrent updates on identical instances

---

## Database Tables (`statcollect` / `statgate`)

```sql
-- Existing / Core
submissions (id, form_id, instance_id, status, submitter_id, raw_xml, json_data, submitted_at)
submission_attachments (id, submission_id, file_name, file_type, file_path, size_bytes)
submission_status_history (id, submission_id, previous_status, new_status, changed_by, notes, changed_at)

-- Missing / To Build
field_devices (id, device_id, model, os_version, app_version, assigned_user_id, status, last_seen_at)
field_assignments (id, form_id, enumerator_id, supervisor_id, admin_unit_id, quota, completed_count, status, start_date, due_date)
field_locations (id, user_id, device_id, latitude, longitude, accuracy, battery_level, captured_at)
field_incidents (id, user_id, assignment_id, type, severity, description, status, reported_at)
```

---

## Integration Points

| Module | Integration Flow |
|---|---|
| **Field Registry (:9090)** | Authenticates enumerators and field supervisors via JWT credentials |
| **StatSpatial (:8094)** | Visualizes field tracking, EA boundary verification, and survey heatmaps |
| **StatChat (:4000)** | Emergency field channel broadcasts, instant supervisor-to-enumerator threads |
| **PMS / RMS** | Automatic binding of field submissions to project milestone indicators |

---

## Acceptance Criteria

- [x] StatCollect receives ODK multipart submissions from collect-master
- [x] Submissions are validated and broadcast over Redis event bus
- [x] Admin console inspects submissions and media attachments
- [x] `object_links` created for incoming submissions
- [ ] Device registry and pairing workflow operational
- [ ] Assignment dispatcher and EA allocation operational
- [ ] Supervisor review and rejection/re-interview workflow operational
- [ ] Real-time enumerator location tracking streamed to StatSpatial
- [ ] AI-assisted fraud and duplicate submission detection operational

---

## Ports & Services

| Component | Port |
|---|---|
| StatCollect Backend (Go) | :8080 |
| collect-master (Android) | Mobile Client |

---

## Estimated Duration

14 weeks

## Milestone

Enterprise Mobile & Field Operations Platform Complete. Field data collected securely offline, synced reliably, and supervised in real time.