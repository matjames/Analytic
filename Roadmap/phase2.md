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
- Roles, permissions, grants, delegation of authority
- APIs: `GET/POST /api/permissions/policies`, `/api/permissions/roles`, `/api/permissions/grants`

**Audit Log** (`enterprise/core/audit_log.go`)
- Cross-platform audit trail API: `GET /api/apis/audit`

**Notifications Framework** (`enterprise/core/notifications.go`)
- Platform-wide notifications: `GET/POST /api/notifications`, `PUT /api/notifications/:id/read`

---

## What is Missing ❌

### Security (Critical)
- **MFA (Multi-Factor Authentication)** — not implemented in any service
- **Email verification** on registration — not implemented; registration is immediate
- **Refresh tokens** — not implemented; JWT sessions expire without renewal path
- **OAuth 2.0 / OIDC** — not implemented; external IdP integration not present
- **Geo-login detection** — not implemented
- **Device tracking / session tracking** — not implemented
- **Brute force protection** — no login attempt rate limiting on Registry API

### Identity Features
- **Organisation Branding** — logos, colours, custom themes per org not implemented
- **Workspace Switcher** — no multi-workspace concept in UI
- **Organisation Switcher** — App Launcher has portal but no organisation switcher
- **User invitation system** — no email-based invitation flow
- **User preference management** — user preferences not stored or served

### Frontend Gaps
- **Full identity admin console** — no unified user directory, security settings, session management UI
- **Role & permission management UI** — not present for end-users
- **Activity logs / audit log viewer** — not in Registry React frontend

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

1. **MFA** — TOTP-based two-factor authentication on Registry login
2. **Refresh tokens** — JWT refresh endpoint on Registry
3. **User invitation flow** — invite by email, token-based acceptance
4. **Email verification** — confirm email on registration before activation
5. **Organisation branding** — `org_settings` with logo/colour, served to App Launcher
6. **Admin console UI** — role management, user directory, audit log viewer in Registry frontend

---

## Acceptance Criteria

- [x] Users can register, login, and receive a Registry JWT
- [x] All other services validate the Registry JWT correctly
- [x] Facilities, org-units, and administrative boundaries are queryable
- [x] Bulk CSV import for facilities works
- [x] ABAC engine is operational in analytics core
- [x] Audit logs are written on all mutating operations
- [x] Notifications framework operational in enterprise core
- [ ] MFA operational
- [ ] Refresh token flow operational
- [ ] User invitation system operational
- [ ] Organisation branding served to App Launcher
- [ ] Full admin console UI operational

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
