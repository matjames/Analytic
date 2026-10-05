# PHASE 5 — RESEARCH ECOSYSTEM & SCIENTIFIC WORKFLOW

## Objective

Build the complete research ecosystem supporting the entire scientific lifecycle from idea conception through publication, preservation, and knowledge dissemination.

The Research Management System (RMS) manages proposals, ethics review, funding, literature, datasets, surveys, publications, and team collaboration — all linked to the StatGate identity and communication backbone.

---

## Vision

StatGate becomes the primary platform where researchers design, conduct, analyse, publish, and preserve their work. No external research management tool is needed. Every research object connects to projects, datasets, GIS layers, surveys, and statistical outputs.

---

## Current State (28 September 2026)

The first Phase 5 completion slice is deployed and live-certified on RMS ports `8092` and `3011`. Existing research lifecycle records remain available, and the new scholarly-governance workflows are workspace-aware through Registry JWT plus `X-Workspace-ID` context.

Completed and certified:

- IRB/ethics committee registration, status, approval validity, workspace ownership, and deletion.
- Ethics applications with committee assignment and the existing submission, review, approval, amendment, renewal, and compliance fields.
- Local DOI registry with generated or supplied identifiers, status, author metadata, citation formatting (APA, Chicago, Harvard, Vancouver, and BibTeX), and full lifecycle operations.
- DOI provider registration workflow with idempotent local registration, persisted retry state, and configurable DataCite and Crossref adapters.
- Open-science repository records with resource type, access level, licence, repository, URLs, keywords, and full lifecycle operations.
- Journal submission tracking across draft, submitted, review, revision, acceptance, rejection, and withdrawal states.
- Conference submission tracking across abstract, presentation type, review, acceptance, rejection, and withdrawal states.
- Preservation and archive records with repository, access URL, checksum, retention date, and verification state.
- Knowledge-transfer queue with explicit publish-to-Knowledge-Portal handoff and returned portal content ID.
- Governed readiness assistant with evidence-based recommendations from the study's proposal, ethics, task, dataset, and publication records.
- Provider-backed research assistant with workspace-grounded proposal planning, literature synthesis, and research-gap analysis tasks.
- Governed publication-readiness scoring over study metadata, proposal, ethics, dataset, publication, DOI, preservation, and knowledge-transfer evidence.
- Governed statistical planning recommendations over registered dataset metadata, outcome type, study design, survey structure, and missing-data evidence.
- Governed research-quality review for metadata completeness, proposal content, ethics status, dataset metadata, DOI links, overdue work, and preservation state.
- Deterministic language review for bounded whitespace, capitalization, and long-sentence guidance, with an explicit non-LLM disclaimer.
- Ethics committee member invitations, role assignment, approval, suspension/rejection/revocation, and removal with authenticated audit fields.
- Conference event calendar, registration, attendance types, cancellation, and checked-in status.
- Knowledge Portal editorial review, moderation decisions, publication visibility control, and workspace-scoped graph links.

The migration is additive and compatibility-safe for the earlier RMS schemas already present in Docker PostgreSQL. Backend `go test ./...`, `go vet ./...`, frontend production build, deployed health/readiness, selected-workspace CRUD, unauthenticated rejection, and non-member workspace rejection all pass for this slice.

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
The research workspace also exposes `GET /api/research/:id/assistant`, `GET /api/research/:id/publication-readiness`, `GET /api/research/:id/quality-review`, and `POST /api/research/:id/language-review` for governed readiness, quality, and bounded language review.
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

## Remaining Scope

### Scientific Workflow Features
Ethics committee membership administration is now operational for workspace-owned committees; the remaining scientific workflow items below are external provider and integration work.
The older membership wording in the list below is retained as roadmap history and is superseded by the certified implementation and acceptance item above.
Conference event scheduling, registration, and attendance are now operational inside RMS; external conference-provider federation remains a later integration.
Knowledge Portal moderation, editorial review, publication archiving, and graph-link enrichment are now operational through protected workspace-scoped APIs; external editorial policy federation remains a later integration.
DOI provider registration is now operational with local idempotency plus DataCite and Crossref deposit adapters; credentialed provider deployment remains provider-specific release configuration.
The older external-DOI wording in the list below is retained as roadmap history and is superseded by the certified provider workflow and acceptance item above.
- **External DOI registration** — local DOI assignment and provider adapters are operational; credentialed Crossref/DataCite delivery remains deployment configuration
- **Ethics committee membership administration** — member data is stored, but member-level invitations, roles, and approvals remain
- **Conference integrations** — calendar, registration, and attendance workflows remain outside RMS
- **Knowledge governance** — portal moderation, editorial review, and knowledge-graph enrichment remain in later knowledge phases

### AI Features (legacy planning bullets)
Implementation status: deterministic publication-readiness scoring, research-quality checks, bounded language review, and provider-backed research assistance and language-editing contracts are operational; credentialed provider deployment and domain-specific methodological validation remain.
- **Research AI Assistant** — provider-backed proposal planning, literature synthesis, and gap analysis are implemented; approved LLM endpoint deployment remains
- **Statistical recommendations** — transparent planning and workspace-safe Analytics Core handoff are operational; execution remains dataset-binding dependent
- **Publication readiness review** — deterministic governed review is operational; publisher-specific validation remains
- **Language editing assistant** — deterministic review and an opt-in provider-editing contract are operational; credentialed provider deployment remains
- **Research quality review** — deterministic governed review is operational; domain-specific methodological validation remains

### Integration Gaps
### Current AI Status
The three review capabilities above are now operational as deterministic governed workflows, and provider-backed research assistance plus opt-in language editing have workspace-safe contracts; the remaining AI scope is credentialed LLM deployment and domain-specific methodological validation.
- **StatCollect → RMS** — workspace-safe submission links, internal ingestion, and automatic event delivery are operational and live-certified
- **Analytics Core → RMS** — workspace-safe planning and tabulation handoff are operational; durable dataset binding and executed-result certification remain
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
- [x] DOI registry and local DOI assignment operational
- [x] DOI provider registration workflow operational
- [x] Citation engine (APA, Chicago, Harvard, Vancouver, BibTeX) operational
- [x] IRB committee management and formal approval records operational
- [x] Journal submission tracking operational
- [x] Conference submission tracking operational
- [x] Conference calendar, registration, and attendance operational
- [x] Open science repository operational
- [x] Research preservation and archive tracking operational
- [x] Knowledge transfer to the shared Knowledge Portal operational
- [x] Knowledge Portal moderation, editorial review, and graph enrichment operational
- [x] Governed publication-readiness review operational
- [x] Provider-backed research-assistance workflow operational
- [x] Governed statistical planning recommendations operational
- [x] Analytics Core statistical handoff contract operational
- [x] Governed research-quality review operational
- [x] Deterministic language review operational
- [x] Provider-backed language editing contract operational
- [x] Ethics committee membership and approval administration operational
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
