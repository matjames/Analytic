# StatGate Enterprise Integration Layer

This directory contains the shared enterprise services that unify the StatGate platform.

## Services

### 1. Enterprise Search Service (`search/`)
Aggregates search results across all StatGate applications:
- Staff (Registry)
- Projects (PMS)
- Research (RMS)
- Surveys (PMS/RMS)
- Datasets (RMS)
- Reports (PMS/RMS)
- Meetings (PMS/RMS)
- Documents (PMS/RMS)
- StatChat messages
- HelpDesk tickets
- Facilities (Registry)
- Organizations (Registry)

### 2. Enterprise Core (`core/`) — Phase II
The intelligent, event-driven enterprise operating environment.

### 3. Enterprise Design System (`design-system/`) — Phase II
A centralized UI library for all StatGate applications:
- Buttons, Cards, Forms, Dialogs, Tables, Badges
- Notifications with priority, category, source app, deep links, actions, attachments
- Layouts, Icons, Typography, Spacing
- Themes (light/dark), Accessibility
- Design tokens for colors, spacing, typography, radii, shadows

**Capabilities:**

| # | Capability | Endpoints |
|---|-----------|-----------|
| 1 | **Enterprise Event Bus** | `POST /api/events`, `GET /api/events`, `GET /api/events/types`, `GET /api/events/:id`, `GET/POST /api/events/subscriptions` |
| 2 | **Global Notification Centre** | `GET/POST /api/notifications`, `PUT /api/notifications/:id/read`, `PUT /api/notifications/read-all`, `PUT /api/notifications/:id/archive`, `DELETE /api/notifications/:id` |
| 3 | **Enterprise Timeline** | `GET/POST /api/timeline`, `GET /api/timeline/entity/:entity/:id`, `GET /api/timeline/user/:user` |
| 4 | **Enterprise Dashboard Engine** | `GET/POST /api/dashboards`, `PUT/DELETE /api/dashboards/:id`, `POST /api/dashboards/:id/share`, `GET /api/dashboards/:id/widgets/:widgetId/data` |
| 5 | **Enterprise Widget Library** | `GET /api/widgets`, `GET /api/widgets/:type/preview` |
| 6 | **Universal File Service** | `GET/POST /api/files`, `GET /api/files/:id`, `GET /api/files/:id/download`, `PUT /api/files/:id`, `DELETE /api/files/:id`, `GET /api/files/:id/versions` |
| 7 | **Enterprise Permissions** | `GET/POST /api/permissions/policies`, `GET /api/permissions/check`, `GET/POST /api/permissions/roles`, `GET/POST /api/permissions/grants`, `POST /api/permissions/grants/:id/revoke`, `POST /api/permissions/grants/:id/delegate` |
| 8 | **Enterprise Calendar** | `GET/POST /api/calendar`, `GET/PUT/DELETE /api/calendar/:id` |
| 9 | **Enterprise Reporting Engine** | `GET/POST /api/reports`, `GET /api/reports/:id`, `GET /api/reports/:id/download`, `DELETE /api/reports/:id` |
| 10 | **AI Preparation** | `GET /api/ai/catalog`, `GET /api/ai/catalog/:app`, `GET /api/ai/schema` |
| 11 | **API Governance** | `GET /api/apis`, `GET /api/apis/health`, `GET /api/apis/audit`, `GET /api/openapi.json` |
| 12 | **Monitoring & Observability** | `GET /api/monitoring/services`, `GET /api/monitoring/metrics`, `GET /api/monitoring/health` |

**Event Types Supported:**
- `user.created`, `user.updated`
- `project.created`, `project.closed`, `project.updated`, `project.milestone`
- `survey.published`, `survey.submitted`
- `dataset.updated`, `dataset.created`
- `research.approved`, `research.created`, `research.stage_changed`
- `meeting.scheduled`, `meeting.cancelled`
- `ticket.raised`, `ticket.closed`, `ticket.updated`
- `report.generated`, `dashboard.refreshed`
- `file.uploaded`, `file.updated`
- `approval.requested`, `approval.approved`, `approval.rejected`

**Event-driven automation:**
- Every event is recorded in the Enterprise Timeline
- Events automatically generate notifications with deep links
- Meetings, milestones, and SLA deadlines auto-populate the Enterprise Calendar
- Report completion triggers notifications
- All events are audited

**Widget Library:**
- KPI Card, Trend Chart, Table, Map, Timeline
- Task List, Activity Feed, Calendar, Heat Map
- Gauge, Metric Tile

**Architecture:**
```
┌─────────────────────────────────────────────────┐
│           Enterprise Workspace (App Launcher)    │
├─────────────────────────────────────────────────┤
│  Unified Search │ Enterprise Core │ Dashboards  │
│     (8095)      │     (8096)      │             │
├─────────────────────────────────────────────────┤
│         Enterprise Integration Services          │
├──────────┬──────────┬──────────┬────────────────┤
│   PMS    │   RMS    │ StatChat │  Registry      │
│  (8091)  │  (8092)  │  (4000)  │  (9090)        │
└──────────┴──────────┴──────────┴────────────────┘
```

**Storage:**
- Redis: Event bus, notifications, timeline, dashboards, files metadata, calendar, permissions
- Filesystem: Uploaded files (configurable via `ENTERPRISE_UPLOAD_DIR`)
- In-memory: Report store (production would use Postgres)

## Getting Started

### 1. Start the Enterprise Search service:
```bash
cd enterprise/search
go run cmd/server/main.go
```

### 2. Start the Enterprise Core service:
```bash
cd enterprise/core
go run .
# or
powershell -File start-enterprise-core.ps1
```

### 3. The services run on:
- Enterprise Search: `8095`
- Enterprise Core: `8096`

### 4. Query unified search:
```bash
curl "http://localhost:8095/api/search?q=malaria"
```

### 5. Publish an enterprise event:
```bash
curl -X POST http://localhost:8096/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "project.created",
    "source": "pms",
    "object_type": "project",
    "object_id": "PRJ-001",
    "actor": "admin",
    "payload": {"name": "Malaria Surveillance", "owner": "admin"}
  }'
```

### 6. Get enterprise timeline:
```bash
curl "http://localhost:8096/api/timeline?limit=20"
```

### 7. Get notifications for a user:
```bash
curl "http://localhost:8096/api/notifications?user_id=admin"
```

### 8. Create a configurable dashboard:
```bash
curl -X POST http://localhost:8096/api/dashboards \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Executive Overview",
    "owner_type": "organization",
    "layout": [
      {"id": "w1", "widget_id": "kpi1", "type": "kpi_card", "title": "Projects", "x": 0, "y": 0, "w": 2, "h": 2},
      {"id": "w2", "widget_id": "trend1", "type": "trend_chart", "title": "Activity Trend", "x": 2, "y": 0, "w": 4, "h": 3}
    ]
  }'
```

### 9. Upload a file:
```bash
curl -X POST http://localhost:8096/api/files \
  -F "file=@report.pdf" \
  -F "project_id=PRJ-001"
```

### 10. Check service health:
```bash
curl "http://localhost:8096/api/monitoring/services"
```

## Docker

Both services are included in `docker-compose.yml`:

```bash
docker compose up -d statgate-enterprise statgate-enterprise-core
```
