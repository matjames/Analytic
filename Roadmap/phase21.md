# PHASE 21 — TENANT ISOLATION, RBAC ENFORCEMENT & PERSONA BOUNDARIES

## Objective

Establish comprehensive tenant isolation, role-based access control (RBAC) enforcement, and persona boundary validation across the StatGate platform. This phase ensures that all multi-tenant data access is scoped by authenticated organization context, role-based permission checks are consistently applied across all sensitive endpoints, and frontend route guards prevent unauthorized screen access.

---

# Vision

Every request shall be authenticated.

Every query shall be tenant-scoped.

Every action shall be authorized.

Every screen shall be guarded.

No user shall access data outside their organization.

No user shall perform actions beyond their role.

---

# Major Modules

Tenant Isolation Middleware

Organization Context Enforcement

Role-Based Access Control (RBAC)

Permission Matrix

Frontend Route Guards

Persona Boundary Validation

Cross-Organization Access Audit

---

# Backend Deliverables

## Tenant Isolation (P0)
- Global registration of tenant isolation middleware in Gin router
- Refactored query parameter extraction to use authenticated organization context
- Removal of caller-supplied `organization_id` parameters from BI, GIS, Field, and Data Management handlers
- `orgIDForContext()` helper function for consistent org ID extraction

## RBAC & Role Catalog (P1)
- Centralized role constants in `models/roles.go` with 13 built-in roles
- Database CHECK constraint on `User.Role` field enforcing valid role values
- Role-based permission checks across BI, GIS, Field, and Data Management handlers
- Distinct permission boundaries for Viewers, Analysts, Enumerators, Supervisors, and Administrators
- `canWriteBI()`, `canWriteGIS()`, `canWriteField()`, `canSuperviseField()`, `canWriteDataManagement()` guard functions

## Role Constants
| Role | Identifier | Access Level |
|------|-----------|--------------|
| Admin | `admin` | Full system access |
| Platform Admin | `platform_admin` | Cross-organization administrative access |
| DevOps | `devops` | Infrastructure and deployment access |
| Analyst | `analyst` | Data analysis and BI write access |
| Data Steward | `data_steward` | Data management write access |
| Researcher | `researcher` | Research module access |
| Field Manager | `field_manager` | Field operations management |
| Supervisor | `supervisor` | Field supervision and quality review |
| Enumerator | `enumerator` | Field data collection |
| Inspector | `inspector` | Inspection access |
| Health Worker | `health_worker` | Health data access |
| Monitoring Officer | `monitoring_officer` | M&E module access |
| Viewer | `viewer` | Read-only access |

---

# Flask Deliverables

## Frontend Auth Guards (P1)
- `require_role()` decorator for role-based route protection
- `require_permission()` decorator for permission-based route protection
- Integration with Flask-Login for session-based authentication
- Flash messages for unauthorized access attempts
- Redirect to login for unauthenticated users
- Redirect to dashboard for unauthorized role/permission attempts

---

# Database Deliverables

- `User.Role` CHECK constraint enforcing valid role values
- RBAC tables: `roles`, `permissions`, `role_permissions`, `user_permissions`
- Organization-scoped queries via `organization_id` foreign keys

---

# AI Features

- Role-aware AI recommendations
- Permission-based AI feature access
- Organization-scoped AI analytics

---

# Acceptance Criteria

✓ Tenant isolation middleware globally registered
✓ All BI, GIS, Field, and Data Management handlers use authenticated org context
✓ No caller-supplied `organization_id` parameters accepted
✓ Role constants centralized and database-constrained
✓ Permission checks enforced on all sensitive write endpoints
✓ Frontend route guards prevent unauthorized screen access
✓ Cross-organization access blocked (except platform admins and devops)
✓ Build passes with `go build ./...` and `go vet ./...`

Estimated Duration

4 Weeks

Milestone

Multi-Tenant Security & RBAC Foundation Complete.