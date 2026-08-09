# StatGate PMS — Enterprise Projects Management System

The **StatGate Projects Management System (PMS)** is the operational heartbeat of the StatGate enterprise platform. It provides complete lifecycle management for projects, programmes, grants, consultancy assignments, field operations, research initiatives, and organizational activities.

## Architecture

```
PMS/
├── backend/              # Go API service (PostgreSQL-backed)
│   ├── main.go           # Entry point with DB init + healthcheck
│   ├── database.go       # PostgreSQL connection layer + schema migration
│   ├── models.go         # Enterprise data models
│   ├── routes.go         # HTTP route registration (60+ endpoints)
│   └── db_handlers.go    # Database-backed REST handlers
├── frontend/             # React + Vite UI
│   └── src/
│       ├── App.jsx       # Main application with workspace management
│       ├── components/   # 18+ UI components
│       └── App.css       # StatGate unified theme
└── Dockerfile            # Container build
```

## Enterprise Project Hierarchy

The system supports the full enterprise hierarchy:

```
Organization → Portfolio → Programme → Project → Component → Activity → Task → Deliverable → Output → Outcome → Impact
```

## Project Lifecycle

Every project progresses through configurable stages with workflow automation:

```
Concept → Proposal → Planning → Approval → Funding → Implementation → Monitoring → Evaluation → Closure → Archive
```

Each stage transition triggers automated workflows including notifications, document creation, dashboard activation, and reporting.

## Database Schema

PMS uses its own dedicated PostgreSQL database (`pms`) with 25+ tables:

- `organizations` — Enterprise organization registry
- `portfolios` — Portfolio management
- `programmes` — Programme management
- `projects` — Project registry with full lifecycle management
- `components` — Project components (WBS level 1)
- `activities` — Project activities (WBS level 2)
- `deliverables` — Project deliverables
- `milestones` — Project milestones
- `project_members` — Team member assignments
- `tasks` — Work breakdown structure (WBS) tasks
- `budget_lines` — Financial allocations and spending
- `funding_sources` — Donors and funding commitments
- `cost_centres` — Cost centre management
- `budget_revisions` — Budget revision history
- `procurement_refs` — Procurement references
- `risks` — Risk register with mitigations
- `issues` — Issue tracking with resolutions
- `assumptions` — Project assumptions
- `lessons_learned` — Knowledge capture
- `corrective_actions` — Corrective action tracking
- `documents` — Project document registry
- `meetings` — Meeting schedule with attendees and action items
- `surveys` — StatCollect survey instruments
- `chat_messages` — Project communication channels
- `helpdesk_tickets` — Support incidents linked to projects
- `workflow_rules` — Configurable workflow automation rules
- `permissions` — Role-based access control
- `audit_logs` — Complete audit trail
- `reports` — Generated project reports
- `calendar_events` — Project calendar events

## API Endpoints

### Projects
- `GET /api/projects` — List all projects
- `POST /api/projects` — Create project with auto-provisioned workspace
- `GET /api/projects/:id` — Get full project workspace with all related data
- `PUT /api/projects/:id` — Update project details
- `DELETE /api/projects/:id` — Delete project
- `PUT /api/projects/:id/stage` — Transition project lifecycle stage
- `GET /api/projects/:id/audit` — Get project audit logs
- `GET /api/projects/:id/calendar` — Get project calendar events
- `GET /api/projects/:id/reports` — Get project reports
- `POST /api/projects/:id/reports` — Generate report from live data
- `GET /api/projects/:id/relationships` — Get all connected object counts
- `GET /api/projects/:id/dashboard` — Get project dashboard metrics

### Enterprise Hierarchy
- `GET/POST /api/organizations` — Organization management
- `GET/POST /api/portfolios` — Portfolio management
- `GET/POST /api/programmes` — Programme management
- `GET/POST /api/components` — Component management
- `GET/POST /api/activities` — Activity management
- `GET/POST /api/deliverables` — Deliverable management
- `GET/POST /api/milestones` — Milestone management

### Tasks & Team
- `POST /api/tasks` — Create task
- `PUT /api/tasks/:id` — Update task
- `DELETE /api/tasks/:id` — Delete task
- `PUT /api/tasks/:id/status` — Update task status/progress
- `POST /api/members` — Add team member
- `DELETE /api/members/:id` — Remove team member

### Budget & Finance
- `POST/PUT/DELETE /api/budgets` — Budget line management
- `GET/POST /api/funding` — Funding source management
- `GET/POST /api/cost-centres` — Cost centre management
- `GET/POST /api/budget-revisions` — Budget revision management
- `GET/POST /api/procurement` — Procurement reference management

### Risks & Issues
- `POST/PUT/DELETE /api/risks` — Risk management
- `GET/POST/PUT /api/issues` — Issue management
- `GET/POST /api/assumptions` — Assumption management
- `GET/POST /api/lessons` — Lessons learned management
- `GET/POST /api/corrective-actions` — Corrective action management

### Other
- `POST/DELETE /api/documents` — Document management
- `POST/PUT /api/meetings` — Meeting management
- `POST/PUT /api/surveys` — Survey management
- `POST /api/chats` — Chat message posting
- `POST/PUT /api/helpdesk` — HelpDesk ticket management
- `GET/POST /api/workflow-rules` — Workflow rule management
- `GET/POST /api/permissions` — Permission management
- `GET /api/search?q=` — Enterprise-wide search
- `GET /api/dashboard` — Global dashboard metrics

## Integration with StatGate Ecosystem

### StatChat Integration
Every project workspace has integrated chat channels (`general`, `announcements`, `meetings`) that persist to the PMS database. Project events automatically appear in StatChat.

### Registry Integration
PMS teams, geographic regions, and organizational data align with the **StatGate Field Operations Registry** for unified field identity.

### Helpdesk Integration
Project workspaces surface **Operations Helpdesk** tickets in real time, giving project managers direct visibility into technical incidents and escalations.

### StatCollect Integration
Projects own survey templates, active surveys, field teams, assignments, submissions, and approved datasets. Survey progress updates project progress automatically.

### Analytics Integration
Every project has live dashboards showing completion status, budget utilization, survey progress, team performance, KPI achievement, timeline adherence, and risk status.

### App Launcher
PMS is registered in the **StatGate App Launcher** at `http://localhost:3010`, accessible via the 9-dot app switcher.

## Running

### With Docker (Production)

```bash
# From the Analytics root
docker compose up -d postgres statgate-pms-api statgate-pms-ui
```

- PMS UI: `http://localhost:3010`
- PMS API: `http://localhost:8091`
- Health Check: `http://localhost:8091/health`

### Local Development

```bash
# Backend
cd PMS/backend
go mod tidy
go run main.go

# Frontend (from PMS/frontend)
npm install
npm run dev
```

The frontend development server runs on `http://localhost:5175` and connects to the API at `http://localhost:8091`.

## Environment Variables

### Backend
- `PMS_DB_HOST` — PostgreSQL host (default: `postgres`)
- `PMS_DB_PORT` — PostgreSQL port (default: `5432`)
- `PMS_DB_NAME` — Database name (default: `pms`)
- `PMS_DB_USER` — Database user (default: `PMS`)
- `PMS_DB_PASSWORD` — Database password (default: `Statgate`)
- `PORT` — HTTP listen port (default: `8080`)
- `STATCHAT_API_URL` — StatChat backend URL for integration
- `HELPDESK_API_URL` — Helpdesk backend URL for integration
- `STATGATE_REGISTRY_API_URL` — Registry API URL for unified identity

### Frontend
- `VITE_PMS_API_URL` — Backend API base URL (default: `http://localhost:8091`)

## Version
v2.0.0 — Enterprise Projects Management System