# PHASE 2 — IDENTITY, ORGANIZATION & PLATFORM FOUNDATION

## Objective

Establish the complete enterprise identity infrastructure of StatGate.

This phase creates the core organisational structure — users, organisations, facilities, org-units, departments, roles, JWT identity — upon which every other module operates.

Every user, organisation, workspace, dataset, AI interaction, research activity, and collaboration session inherits the identity model established here.

---

## Vision

Every module in StatGate must know **who** is acting, **which organisation** they belong to, and **what they are allowed to do**. This phase provides those answers platform-wide through one authoritative identity source.

---

## Service: Field Operations Registry

**Repository:** `stage_register/`  
**Backend:** Go + Gin → `stage_register/go-backend/` (:9090)  
**Frontend:** React + Vite → `stage_register/frontend/` (:3007)  
**Database:** `statgate` (shared PostgreSQL instance)  
**Role:** JWT signing authority for the entire StatGate platform

---

## What Exists ✅

### Backend (`stage_register/go-backend/`)

**Auth & User Management**
```
POST /api/users/register       ← user registration
POST /api/users/login          ← login → issues Registry JWT
POST /api/users/change-password
GET  /api/users/me             ← current user profile (auth required)
GET  /api/users                ← user directory (auth required)
GET  /api/users/:id            ← user detail (auth required)
PUT  /api/users/:id            ← update user (auth required)
DELETE /api/users/:id          ← delete user (auth required)
GET  /api/internal/users       ← internal service directory (service credential)
```

**Facilities / Organisations**
```
GET  /api/facilities/public              ← public facility list
GET  /api/facilities/public/:id
GET  /api/facilities/public/export
GET  /api/facilities/summary/ownership-by-level
GET  /api/facilities/summary/ownership-totals
GET  /api/facilities/filters
GET  /api/facilities/distribution/ownership
GET  /api/facilities/distribution/level
GET  /api/facilities/distribution/authority
```

**Organisation Units (Administrative Hierarchy)**
```
GET  /api/orgunits                          ← (basic auth)
GET  /api/orgunits/tree
GET  /api/orgunits/level/:level
GET  /api/orgunits/district/:id/facilities
GET  /api/orgunits/subcounty/:id/facilities
GET  /api/orgunits/:id
GET  /api/orgunits/:id/children
```

**Reference Data**
```
GET  /api/authority        ← authority types
GET  /api/level            ← facility levels
GET  /api/ownership        ← ownership types
GET  /api/adminunits       ← administrative units
GET  /api/adminunits/tree/public
GET  /api/adminunits/districts/public
GET  /api/adminlevel/public
```

**MFL Integration**
```
GET  /api/mfl/facilities   ← Master Facility List integration (basic auth)
GET  /api/mfl/level
GET  /api/mfl/authority
GET  /api/mfl/ownership
GET  /api/mfl/adminunits
```

**Authenticated facility management**
```
GET    /api/facilities       ← list (auth)
GET    /api/facilities/:id
POST   /api/facilities       ← create
PUT    /api/facilities/:id   ← update
DELETE /api/facilities/:id   ← delete
POST   /api/facilities/import/csv    ← bulk CSV import
GET    /api/facilities/:id/history   ← change history
```

**Document management**
```
GET  /api/documents
GET  /api/documents/:id
GET  /api/documents/:id/download
```

**Middleware**
- `middleware.AuthRequired()` — validates Registry JWT on protected routes
- `middleware.InternalServiceRequired()` — validates service-to-service credential
- `middleware.BasicAuthRequired()` — for MFL/OrgUnits endpoints

### Shared Identity Components

**ABAC Engine** (`backend/internal/abac/abac.go`)
- Attribute-Based Access Control (ABAC) policy evaluation
- Used by the analytics core to enforce row-level permissions

**Permissions Engine** (`enterprise/core/permissions.go`)
- Canonical role catalog aligned with Registry JWT roles, default permissions, grants, delegation of authority
- APIs: `GET/POST /api/permissions/policies`, `/api/permissions/roles`, `/api/permissions/grants`

**Audit Log** (`enterprise/core/audit_log.go`)
- Cross-platform audit trail API: `GET /api/apis/audit`

**Notifications Framework** (`enterprise/core/notifications.go`)
- Platform-wide notifications: `GET/POST /api/notifications`, `PUT /api/notifications/:id/read`

---

## What is Missing ❌

### Security (Critical)
- **Email verification delivery** — SMTP-backed verification delivery is implemented and tested with a fake SMTP server; real provider credentials and provider smoke remain deployment gates
- **OAuth 2.0 / OIDC** — optional authorization-code login path exists with provider discovery, signed state, token exchange, userinfo mapping, and Registry JWT issuance; real provider smoke remains a deployment gate
- **Geo-login detection** — privacy-preserving country/region login hints and anomaly flags are stored from trusted proxy headers; external IP intelligence provider enrichment remains optional
- **Device intelligence** — browser metadata, privacy-preserving network fingerprints, country/region hints, anomaly flags, last-seen timestamps, and revocation now exist
- **Brute force protection** — Redis-backed distributed enforcement exists for sensitive Registry endpoints and is certified against live Redis

### Identity Features
- **Organisation Branding** — Registry, App Launcher, PMS, RMS, StatGovernance, and StatChat consume Registry tenant branding; later services remain
- **Workspace Switcher** — Registry and supported module shells have selection/context propagation and membership verification; PMS/RMS root, child, dashboard, and search paths are scoped, while later services remain
- **Organisation Switcher** — App Launcher has portal but no organisation switcher
- **User invitation system** — token acceptance, administration, SMTP delivery, and plaintext invitation templates exist; real provider smoke remains a deployment gate
- **Authenticated organisation enforcement** — tenant-admin user and invitation administration derives organisation from the Registry JWT; platform admins retain cross-organisation administration
- **User preference management** — personal timezone, theme, density, locale, and email notification preferences are persisted through the Registry settings API

### Frontend Gaps
- **Full identity admin console** — core directory, security settings, session management, invitations, branding, roles, permissions, and audit viewing exist; advanced delegation remains

---

## Database Tables (Adopted — `statgate` database)

```sql
-- Core identity (Registry-owned)
users, user_profiles, user_preferences
organisations, organisation_settings, organisation_branding
facilities, facility_history
org_units, org_unit_levels
admin_units, admin_levels
roles, permissions, user_roles, user_permissions
sessions, login_attempts, api_keys
documents, document_versions
-- Shared (Enterprise Core)
audit_logs, activity_logs, notifications, object_links
```

---

## Next to Build

1. **Email provider certification** — configure the production SMTP provider and verify registration/resend/invitation delivery against that provider
2. **OAuth 2.0 / OIDC provider certification** — configure the external identity provider, verify callback/userinfo mapping, and certify role/tenant claims in the target deployment
3. **Distributed abuse controls** — move rate limits and device/session intelligence to shared infrastructure
4. **Cross-module authorization** — enforce workspace membership and object-level grants in every supported backend using the canonical Enterprise Core permission catalog

---

## Acceptance Criteria

- [x] Users can register, login, and receive a Registry JWT
- [x] All other services validate the Registry JWT correctly
- [x] Facilities, org-units, and administrative boundaries are queryable
- [x] Bulk CSV import for facilities works
- [x] ABAC engine is operational in analytics core
- [x] Audit logs are written on all mutating operations
- [x] Notifications framework operational in enterprise core
- [x] MFA operational
- [x] Refresh token flow operational
- [x] User invitation system operational
- [x] Tenant-admin user and invitation administration derives organisation scope from authenticated identity
- [x] Registry directory authorization tests cover non-admin rejection, internal-service access, tenant scoping, and privileged-role boundaries
- [x] Enterprise Core canonical role/permission catalog aligns with Registry JWT roles and has default-policy tests
- [x] Organisation branding served to App Launcher
- [x] Registry admin/initiator headers consume tenant branding
- [x] StatGovernance and StatChat shell branding consumes Registry tenant branding
- [x] Core admin console UI operational
- [x] Email verification flow implemented
- [x] Optional OIDC authorization-code login path implemented and fake-provider redirect/token/userinfo behavior certified
- [x] SMTP-backed email verification delivery path implemented and fake-SMTP certified
- [x] SMTP-backed invitation email delivery path implemented and fake-SMTP certified
- [x] User preference management implemented
- [x] Session/device visibility implemented
- [x] Geo-login hint capture and cross-country session anomaly detection implemented
- [x] Redis-backed distributed rate limiting implemented and live-Redis certified for sensitive Registry endpoints
- [x] Workspace creation and membership administration UI implemented
- [x] Enterprise Core workspace manager authorization role matrix tested
- [x] Enterprise Core file records carry tenant ownership and list/get/download/update/delete/project-file routes enforce tenant or platform-admin access
- [x] Selected-workspace membership verification in PMS, RMS, and StatChat implemented
- [x] PMS project and RMS research root-object workspace scope implemented
- [x] PMS/RMS root workspace guards applied to child routes
- [x] PMS/RMS workspace guards applied to JSON child references
- [x] PMS/RMS workspace guards applied to direct child-ID writes
- [x] PMS/RMS primary dashboards and search paths workspace-scoped
- [x] StatGovernance workspace membership and frontend header propagation implemented
- [x] Analytics Core telemetry/anomaly workspace scope implemented
- [x] Analytics persistent assets and Python dashboard storage workspace-scoped
- [x] Analytics platform projects, reports, research, and object links workspace-scoped
- [x] Analytics workflow, command-centre, and dataset metadata paths workspace-scoped
- [x] Legacy Analytics routes derive tenant scope from authenticated identity
- [x] Legacy Analytics dataset refresh/ingestion derives tenant/workspace scope and records scoped metadata/events
- [x] Analytics Python runtime compile, smoke, and pytest certification completed
- [x] StatOps pipeline and deployment workspace propagation/scoping implemented
- [x] StatOps frontend production build certified
- [x] StatSpatial workspace validation and frontend propagation implemented
- [x] StatSpatial store-level tenant/workspace ownership migration implemented
- [x] StatSpatial live PostgreSQL ownership migration and scoped-store certification completed
- [x] StatTrust workspace validation, frontend propagation, and tenant-derived incident events implemented
- [x] StatTrust in-memory incident, ledger, and provenance scope enforcement implemented
- [x] StatTrust scoped ledger/provenance PostgreSQL persistence hooks and startup reload implemented
- [x] StatTrust live PostgreSQL ledger/provenance persistence and reload certification completed
- [x] StatFederation workspace validation and federation-node boundary propagation implemented
- [x] StatFederation workspace columns and indexes migration added for all persisted domains
- [x] StatFederation federated-node PostgreSQL workspace predicates and persistence implemented
- [x] StatFederation DSA and national-indicator PostgreSQL workspace predicates and persistence implemented
- [x] StatFederation distributed-query and object-link PostgreSQL workspace predicates and persistence implemented
- [x] StatFederation diplomacy, compliance, and federated-search PostgreSQL workspace predicates and persistence implemented
- [x] StatFederation metadata-vocabulary PostgreSQL workspace predicates and persistence implemented
- [x] StatFederation live PostgreSQL workspace migration and object-link isolation certification completed
- [x] App Launcher selected-workspace handoff implemented

---

## Ports & Services

| Component | Port |
|---|---|
| Registry API (Go) | :9090 |
| Registry UI (React) | :3007 |

---

## Estimated Duration

6–8 weeks

## Milestone

Enterprise Identity Platform complete. Every StatGate service validates identity from one authoritative source.
