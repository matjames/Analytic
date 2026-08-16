# PHASE 5 — RESEARCH ECOSYSTEM & SCIENTIFIC WORKFLOW

## Objective

Build the complete research ecosystem supporting the entire scientific lifecycle from idea conception through publication, preservation, and knowledge dissemination.

The Research Management System (RMS) manages proposals, ethics review, funding, literature, datasets, surveys, publications, and team collaboration — all linked to the StatGate identity and communication backbone.

---

## Vision

StatGate becomes the primary platform where researchers design, conduct, analyse, publish, and preserve their work. No external research management tool is needed. Every research object connects to projects, datasets, GIS layers, surveys, and statistical outputs.

---

## Service: RMS (Research Management System)

**Repository:** `RMS/`  
**Backend:** Go + Gin → `RMS/backend/` (:8092)  
**Frontend:** React 18 + Vite + TypeScript → `RMS/frontend/` (:3011)  
**Database:** `rms` (PostgreSQL 15, schema: `rms`)  
**Auth:** Registry JWT middleware on all routes  
**Redis:** Redis event bus for cross-module events

---

## Backend Structure

```
RMS/backend/
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

### API Routes (`RMS/backend/routes.go`)

**Research Projects (Core)**
```
GET    /api/research                       ← list research projects
POST   /api/research                       ← create research project
GET    /api/research/:id                   ← research workspace (full detail)
PUT    /api/research/:id                   ← update research project
DELETE /api/research/:id                   ← delete research project
PUT    /api/research/:id/stage             ← advance research stage
GET    /api/research/:id/dashboard         ← research project dashboard
GET    /api/research/:id/audit             ← audit trail
```

**Team Members**
```
POST   /api/members                        ← add team member
PUT    /api/members/:id                    ← update member role
DELETE /api/members/:id                    ← remove member
```

**Proposals**
```
GET    /api/research/:id/proposals         ← list proposals
POST   /api/proposals                      ← create proposal
PUT    /api/proposals/:id                  ← update proposal
DELETE /api/proposals/:id                  ← delete proposal
```

**Ethics Review**
```
GET    /api/research/:id/ethics            ← ethics review records
POST   /api/ethics                         ← create ethics review
PUT    /api/ethics/:id                     ← update ethics review
DELETE /api/ethics/:id                     ← delete ethics review
```

**Grants & Funding**
```
GET    /api/research/:id/grants            ← funding / grants list
POST   /api/grants                         ← create grant record
PUT    /api/grants/:id                     ← update grant
DELETE /api/grants/:id                     ← delete grant
```

**Literature Review**
```
GET    /api/research/:id/literature        ← literature references
POST   /api/literature                     ← add literature reference
PUT    /api/literature/:id                 ← update reference
DELETE /api/literature/:id                 ← delete reference
```

**Research Datasets**
```
GET    /api/research/:id/datasets          ← datasets linked to research
POST   /api/datasets                       ← create dataset record
PUT    /api/datasets/:id                   ← update dataset
DELETE /api/datasets/:id                   ← delete dataset
```

**Publications**
```
GET    /api/research/:id/publications      ← publications list
POST   /api/publications                   ← create publication
PUT    /api/publications/:id               ← update publication
DELETE /api/publications/:id               ← delete publication
```

**Tasks**
```
POST   /api/tasks                          ← create task
PUT    /api/tasks/:id                      ← update task
DELETE /api/tasks/:id                      ← delete task
```

**Meetings**
```
POST   /api/meetings                       ← create meeting
PUT    /api/meetings/:id                   ← update meeting
```

**Risks & Issues**
```
POST   /api/risks                          ← create risk
PUT    /api/risks/:id                      ← update risk
DELETE /api/risks/:id                      ← delete risk
GET    /api/research/:id/issues            ← issues list
POST   /api/issues                         ← create issue
PUT    /api/issues/:id                     ← update issue
DELETE /api/issues/:id                     ← delete issue
```

**Documents**
```
GET    /api/research/:id/documents         ← documents attached to research
POST   /api/documents                      ← attach document
DELETE /api/documents/:id                  ← remove document
```

**Surveys**
```
GET    /api/research/:id/surveys           ← surveys linked to research
POST   /api/surveys                        ← create survey link
PUT    /api/surveys/:id                    ← update survey link
DELETE /api/surveys/:id                    ← delete survey link
```

**Reports**
```
GET    /api/research/:id/reports           ← reports list
POST   /api/reports                        ← create report
DELETE /api/reports/:id                    ← delete report
```

**Calendar**
```
GET    /api/research/:id/calendar          ← research calendar events
POST   /api/calendar                       ← create calendar event
PUT    /api/calendar/:id                   ← update calendar event
DELETE /api/calendar/:id                   ← delete calendar event
```

**StatChat Integration**
```
POST   /api/chats                          ← post chat message in research conversation
```

**Dashboard, Search, Activity**
```
GET    /api/dashboard                      ← RMS summary dashboard
GET    /api/search?q=                      ← full-text search across RMS data
GET    /api/activity                       ← enterprise activity timeline
```

### Database Schema (`rms` schema)

```sql
-- Core
rms.research_projects, rms.project_members

-- Scientific workflow
rms.proposals, rms.ethics_reviews, rms.grants, rms.literature_references
rms.datasets, rms.publications

-- Execution
rms.tasks, rms.meetings, rms.risks, rms.issues
rms.documents, rms.surveys, rms.reports

-- Scheduling & comms
rms.calendar_events, rms.chat_messages

-- Governance
rms.audit_logs
```

---

## What is Missing ❌

### Scientific Workflow Features
- **Institutional Review Board (IRB)** — no IRB committee management or formal IRB approval workflow
- **Ethics committee membership** — not implemented
- **DOI Management / DOI Integration** — no DOI assignment or CrossRef API integration
- **Citation Engine** — literature references exist but no citation format engine (APA, Chicago, Vancouver)
- **Journal submission tracking** — no external journal submission workflow
- **Conference management** — no conference submission or tracking
- **Open Science / Research Repository** — no open-access repository or data sharing portal
- **Research preservation / archiving** — no formal long-term preservation workflow
- **Knowledge transfer** — no mechanism to push research findings to the knowledge base

### AI Features
- **Research AI Assistant** — no LLM-powered proposal writing, literature summaries, or gap analysis
- **Statistical recommendations** — not integrated with analytics core
- **Publication readiness review** — not implemented
- **Language editing assistant** — not implemented
- **Research quality review** — not implemented

### Integration Gaps
- **StatCollect → RMS** — survey submissions not auto-linked to research projects
- **Analytics Core → RMS** — no path for research datasets to be analysed in the analytics workspace
- **StatSpatial → RMS** — field visit geolocation not mapped

---

## Integration Points

| Module | How RMS integrates |
|---|---|
| StatChat | `POST /api/chats` — messages in research conversation thread |
| Enterprise Core | Activity timeline, approval workflows, file management |
| PMS | Cross-module links; research projects can be linked to PMS projects |
| StatCollect | Survey submissions linked to research via `object_links` |
| Analytics Core | Datasets registered in RMS can be loaded into the analytics workspace |
| StatSpatial | (planned) Field visit and site mapping |

---

## Acceptance Criteria

- [x] Research projects can be created, staged, updated, deleted
- [x] Proposal workflow operational
- [x] Ethics review records manageable
- [x] Grant/funding tracking operational
- [x] Literature references manageable
- [x] Datasets linked to research projects
- [x] Publications manageable
- [x] Tasks, meetings, risks, issues operational
- [x] Documents attached to research
- [x] Calendar events per research project
- [x] StatChat integration operational
- [x] Research dashboard and search operational
- [x] Activity timeline and audit trail operational
- [ ] DOI integration operational
- [ ] Citation engine (APA, Chicago) operational
- [ ] IRB committee management and formal approval workflow
- [ ] Journal submission tracking
- [ ] Open science repository operational
- [ ] Research AI assistant operational

---

## Ports & Services

| Component | Port |
|---|---|
| RMS API (Go) | :8092 |
| RMS UI (React/Vite) | :3011 |

---

## Estimated Duration

10 weeks

## Milestone

Scientific Research Platform complete. Proposals, ethics, funding, literature, datasets, and publications managed in one unified system.
