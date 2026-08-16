# PHASE 4 — PROJECT, PROGRAM & PORTFOLIO MANAGEMENT

## Objective

Develop the complete Project Management ecosystem enabling organisations to plan, execute, monitor, evaluate, and archive projects, programmes, grants, and portfolios.

Every statistical activity, research study, survey, field operation, and institutional initiative shall be managed through this module.

---

## Vision

StatGate shall become a complete Enterprise Project Management (EPM) platform. Projects transition smoothly from proposal → implementation → monitoring → reporting → closure → institutional learning. The PMS is the operational backbone, integrating seamlessly with Research, Statistics, AI, GIS, Communication (StatChat), and Reporting.

---

## Service: PMS (Project Management System)

**Repository:** `PMS/`  
**Backend:** Go + Gin → `PMS/backend/` (:8091)  
**Frontend:** React 18 + Vite + TypeScript → `PMS/frontend/` (:3010)  
**Database:** `pms` (PostgreSQL 15, schema: `pms`)  
**Auth:** Registry JWT middleware on all routes  
**Redis:** Redis event bus for cross-module events

---

## Backend Structure

```
PMS/backend/
├── main.go         ← Gin server, DB init, Redis init, Prometheus, CORS, health
├── database.go     ← PostgreSQL connection + migrateDB() (inline SQL, IF NOT EXISTS)
├── routes.go       ← RegisterRoutes() grouped by resource
├── db_handlers.go  ← all database handler functions
├── models.go       ← Go structs for all entities
├── middleware.go   ← registryAuthMiddleware(), getEnv()
└── events.go       ← Redis event publishing helpers
```

---

## What Exists ✅

### API Routes (`PMS/backend/routes.go`)

**Enterprise Hierarchy**
```
GET    /api/organizations              ← list organisations
POST   /api/organizations              ← create organisation
GET    /api/portfolios                 ← list portfolios
POST   /api/portfolios                 ← create portfolio
GET    /api/programmes                 ← list programmes
POST   /api/programmes                 ← create programme
```

**Projects (Core)**
```
GET    /api/projects                   ← list projects (with filters)
POST   /api/projects                   ← create project
GET    /api/projects/:id               ← project workspace (full detail)
PUT    /api/projects/:id               ← update project
DELETE /api/projects/:id               ← delete project
PUT    /api/projects/:id/stage         ← advance project stage
GET    /api/projects/:id/audit         ← project audit trail
GET    /api/projects/:id/calendar      ← project calendar events
GET    /api/projects/:id/reports       ← project reports list
POST   /api/projects/:id/reports       ← generate project report
GET    /api/projects/:id/relationships ← cross-module object links
```

**Activities, Components, Deliverables, Milestones**
```
GET    /api/projects/:id/components    ← project components
POST   /api/components                 ← create component
GET    /api/projects/:id/activities    ← activities list
POST   /api/activities                 ← create activity
GET    /api/projects/:id/deliverables  ← deliverables list
POST   /api/deliverables               ← create deliverable
GET    /api/projects/:id/milestones    ← milestones list
POST   /api/milestones                 ← create milestone
```

**Tasks**
```
POST   /api/tasks                      ← create task
PUT    /api/tasks/:id                  ← update task
DELETE /api/tasks/:id                  ← delete task
PUT    /api/tasks/:id/status           ← update task status
```

**Team Members**
```
POST   /api/members                    ← add team member
DELETE /api/members/:id                ← remove team member
```

**Budget & Finance**
```
POST   /api/budgets                    ← create budget line
PUT    /api/budgets/:id                ← update budget line
DELETE /api/budgets/:id                ← delete budget line
GET    /api/projects/:id/funding       ← funding sources
POST   /api/funding                    ← create funding source
GET    /api/projects/:id/cost-centres  ← cost centres
POST   /api/cost-centres               ← create cost centre
GET    /api/projects/:id/budget-revisions ← budget revisions
POST   /api/budget-revisions           ← create budget revision
GET    /api/projects/:id/procurement   ← procurement references
POST   /api/procurement                ← create procurement reference
```

**Risks & Issues**
```
POST   /api/risks                      ← create risk
PUT    /api/risks/:id                  ← update risk
DELETE /api/risks/:id                  ← delete risk
GET    /api/projects/:id/issues        ← issues list
POST   /api/issues                     ← create issue
PUT    /api/issues/:id                 ← update issue
GET    /api/projects/:id/assumptions   ← assumptions list
POST   /api/assumptions                ← create assumption
GET    /api/projects/:id/lessons       ← lessons learned
POST   /api/lessons                    ← create lesson
GET    /api/projects/:id/corrective-actions ← corrective actions
POST   /api/corrective-actions         ← create corrective action
```

**Documents & Meetings**
```
POST   /api/documents                  ← attach document
DELETE /api/documents/:id              ← remove document
POST   /api/meetings                   ← create meeting
PUT    /api/meetings/:id               ← update meeting
```

**Workflow & Permissions**
```
GET    /api/workflow-rules             ← workflow rule list
POST   /api/workflow-rules             ← create workflow rule
GET    /api/permissions                ← permissions list
POST   /api/permissions                ← create permission
```

**Surveys**
```
POST   /api/surveys                    ← create survey (linked to project)
PUT    /api/surveys/:id                ← update survey
```

**HelpDesk Integration**
```
POST   /api/helpdesk                   ← raise helpdesk ticket from project context
PUT    /api/helpdesk/:id               ← update helpdesk ticket
```

**StatChat Integration**
```
POST   /api/chats                      ← post chat message in project conversation
```

**Dashboard, Search, Activity**
```
GET    /api/dashboard                  ← PMS summary dashboard
GET    /api/projects/:id/dashboard     ← project-level dashboard
GET    /api/search?q=                  ← full-text search across PMS data
GET    /api/activity                   ← enterprise activity timeline
```

### Database Schema (`pms` schema)

```sql
-- Hierarchy
pms.organizations, pms.portfolios, pms.programmes, pms.projects

-- Planning
pms.components, pms.activities, pms.deliverables, pms.milestones
pms.tasks, pms.task_dependencies

-- Team
pms.project_members

-- Finance
pms.budget_lines, pms.funding_sources, pms.cost_centres
pms.budget_revisions, pms.procurement_refs

-- Risk & Issues
pms.risks, pms.issues, pms.assumptions, pms.lessons_learned
pms.corrective_actions

-- Documents & Comms
pms.documents, pms.meetings, pms.surveys, pms.helpdesk_tickets, pms.chat_messages

-- Governance
pms.workflow_rules, pms.permissions, pms.audit_logs, pms.calendar_events, pms.reports
```

---

## What is Missing ❌

### Programme & Portfolio Features
- **Gantt chart view** — no server-side timeline calculation; frontend renders from milestone/activity dates
- **Portfolio dashboard** — aggregated portfolio-level KPIs not built
- **Donor Management** — donors entity not implemented (funding sources exist but no full donor CRM)
- **Grant Management** — grants not tracked separately from funding sources

### Planning Tools
- **Logical Framework (LogFrame)** — not implemented
- **Theory of Change** — not implemented
- **Work Breakdown Structure (WBS)** — no hierarchical WBS tree beyond components/activities
- **Critical Path Analysis** — not implemented
- **Resource planner / resource allocation** — not implemented

### Reporting & AI
- **Project AI Assistant** — no AI-powered risk prediction, schedule optimisation, or budget forecasting
- **Automated progress summaries** — not implemented
- **Donor reporting templates** — not implemented

### GIS Integration
- **Map project locations** — not wired to StatSpatial
- **Field activity tracking on map** — not implemented

---

## Integration Points

| Module | How PMS integrates |
|---|---|
| StatChat | `POST /api/chats` — posts messages to project conversation thread |
| Enterprise Core | Activity timeline, approval workflows, file management |
| RMS | Cross-module object links via `object_links` table |
| StatCollect | Survey submissions linked to project via `object_links` |
| StatSpatial | (planned) Project location mapping |

---

## Acceptance Criteria

- [x] Organisations, Portfolios, Programmes hierarchy operational
- [x] Projects can be created, updated, staged, and deleted
- [x] Activities, components, deliverables, milestones tracked
- [x] Tasks with status management operational
- [x] Risk register operational
- [x] Budget lines and funding sources operational
- [x] Documents attached to projects
- [x] Meetings managed from project workspace
- [x] StatChat integration (chat message from project context)
- [x] HelpDesk ticket from project context
- [x] Project dashboard and search operational
- [x] Activity timeline operational
- [x] Audit trail per project
- [ ] LogFrame builder operational
- [ ] Theory of Change designer operational
- [ ] Portfolio/programme dashboard with aggregated KPIs
- [ ] Donor management operational
- [ ] AI project health assistant operational
- [ ] GIS project location mapping operational

---

## Ports & Services

| Component | Port |
|---|---|
| PMS API (Go) | :8091 |
| PMS UI (React/Vite) | :3010 |

---

## Estimated Duration

8 weeks

## Milestone

Enterprise Project Management Platform complete. Every project, programme, and portfolio tracked through a single system.
