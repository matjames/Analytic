# StatGate Phase IV — Enterprise Data Intelligence & Real-Time Analytics

**Status:** IN PROGRESS  
**Phase:** IV  
**Prerequisite:** Phase III Enterprise Integration Validation completed

---

## 1. Purpose

Phase III connected the platform. Phase IV turns the connected platform into an **intelligence platform**.

The engineering objective is:

> **TURNING THE INFORMATION MOVING THROUGH STATGATE INTO TRUSTED, REAL-TIME, ACTIONABLE INTELLIGENCE.**

The system moves from:

**DATA → INFORMATION → INTELLIGENCE → ACTION**

---

## 2. What Has Been Implemented

### 2.1 Enterprise Data Layer

| Capability | Implementation |
|---|---|
| **Enterprise Records** | Structured records preserve source application, source entity, source ID, tenant, organization, project, user, timestamp, event type, data version, correlation ID. |
| **Dataset Registry** | All known data sources registered as datasets with live record counts from source APIs. |
| **Event-Driven Ingestion** | Every domain event is converted to an enterprise record and fed to the analytics layer in real time. |
| **Data Freshness** | Every dataset tracks last updated, record count, processing status, age and freshness. |

### 2.2 KPI Engine

| Capability | Implementation |
|---|---|
| **KPI Definitions** | Every KPI has a definition: name, description, formula, source app, scope, period, unit, threshold. |
| **Versioned** | Each KPI definition has a version number. |
| **Configurable** | New KPI definitions can be created via API. |
| **Source-Aware** | Each KPI knows its source application. |
| **Time-Aware** | Each KPI has a reporting period (daily, weekly, monthly, quarterly, yearly). |
| **Live Computation** | KPI values are always computed from live data — never hardcoded. |
| **KPI Drill-Down** | Each KPI provides a breakdown by project, survey, region, facility, or status. |
| **Data Lineage** | Every KPI value includes full lineage: source app, formula, reporting period, filters, last update, responsible system. |

### 2.3 Data Quality Intelligence

| Capability | Implementation |
|---|---|
| **Duplicate Detection** | Identifies duplicate records by source ID. |
| **Missing Values** | Detects missing identifiers and empty metadata fields. |
| **Invalid Dates** | Validates timestamp formats. |
| **Invalid Coordinates** | Validates GPS coordinates are within valid ranges. |
| **Outlier Detection** | Uses standard deviation-based detection on numeric values. |
| **Inconsistent Categories** | Detects unexpected status values. |
| **Low Reporting** | Flags scopes with abnormally low record counts. |
| **Quality Score** | Computes an overall quality score with completeness score. |

### 2.4 Anomaly Detection Foundation

| Capability | Implementation |
|---|---|
| **Rule-Based Detection** | Detects sudden submission drops, ticket spikes, facility inactivity, project delays. |
| **Configurable Rules** | Anomaly rules are defined with type, metric, window, threshold, severity. |
| **Extensible Architecture** | Detection rules are separated from the detection engine, allowing future statistical and AI-based detection. |
| **Anomaly Records** | Detected anomalies are stored, published, and audited. |

### 2.5 Alert Engine

| Capability | Implementation |
|---|---|
| **Configurable Rules** | Alert rules define metric, condition, threshold, priority, scope, recipients. |
| **Event-Driven** | Alerts are evaluated when relevant events arrive. |
| **Scheduled Sweep** | KPI-based alert rules are evaluated periodically. |
| **Prioritized** | Alerts have priority levels (low, medium, high, critical). |
| **Traceable** | Every alert has an ID, rule ID, metric, value, threshold, and audit trail. |
| **Linked to Entity** | Alerts link to the affected project, facility, or entity. |
| **Actionable** | Alerts support actions: create HelpDesk ticket, notify recipients, open StatChat. |

### 2.6 Analytics Dashboards

| Dashboard | Contents |
|---|---|
| **Executive** | Organization KPIs, data quality, alerts, timeline, project portfolio, field operations. |
| **Management** | Performance KPIs, tasks, approvals, team activity. |
| **Project** | Project KPIs, tasks, data quality, anomalies, project timeline. |
| **Field** | Field KPIs, geographic coverage, data quality, anomalies, facilities. |

### 2.7 Cross-Application Intelligence

Combines signals from PMS, StatCollect, HelpDesk and StatChat to identify the bigger operational picture:

- PMS: project behind schedule
- StatCollect: reporting dropped
- HelpDesk: field-related tickets open
- StatChat: field team discussing connectivity problems
- Analytics: risk level synthesis

### 2.8 Enterprise Report Builder

| Capability | Implementation |
|---|---|
| **Data Source Selection** | KPIs, submissions, projects, research, tickets, facilities, activity, quality. |
| **Filters** | Apply filters to report data. |
| **Date Ranges** | Filter by date range. |
| **Grouping** | Group records by field. |
| **Aggregations** | Aggregation support. |
| **Charts** | Chart type selection. |
| **Narrative** | Narrative sections. |
| **Branding** | Organization branding. |
| **Export** | CSV, Excel, JSON, PDF, GeoJSON. |
| **Scheduling** | Daily, weekly, monthly schedules with delivery channels. |

### 2.9 Scheduled Reports

- Daily operations report
- Weekly project report
- Monthly executive report
- Survey performance report
- Research portfolio report
- HelpDesk performance report

Delivered through: Enterprise Workspace, Notifications, StatChat, Email (where configured).

### 2.10 Data Export

| Format | Support |
|---|---|
| CSV | ✅ |
| Excel | ✅ (tab-separated) |
| JSON | ✅ |
| PDF | ✅ (minimal) |
| GeoJSON | ✅ (from GPS coordinates) |

Exports respect permissions, tenant, organization, project and data classification.

### 2.11 Analytics Search

Enterprise Search discovers:
- Dashboards
- KPIs
- Reports
- Datasets
- Alert rules

### 2.12 AI-Ready Analytics

Clean interfaces for future AI services:
- `/api/analytics/ai/catalog` — structured resource catalog
- `/api/analytics/ai/kpis` — all KPI definitions with live values
- `/api/analytics/ai/trends` — time series trends
- `/api/analytics/ai/quality` — data quality indicators
- `/api/analytics/ai/events` — recent events
- `/api/analytics/ai/reports` — report definitions
- `/api/analytics/ai/anomalies` — detected anomalies
- `/api/analytics/ai/alerts` — active alerts
- `/api/analytics/ai/project-activity` — project activity

### 2.13 Data Lineage

Every analytical result is traceable:
- Source application
- Source dataset
- Calculation/formula
- Reporting period
- Filters
- Last update
- Responsible system
- Raw record count

### 2.14 Real-Time Analytics Stream

Server-Sent Events (SSE) endpoint at `/api/analytics/stream`:
- Dashboards update when meaningful events occur
- No manual refresh required
- Redis pub/sub for cross-instance fan-out

---

## 3. API Endpoints Added (Phase IV)

| Endpoint | Description |
|---|---|
| `GET /api/analytics/records` | List enterprise records |
| `GET /api/analytics/records/:source/:entity/:id` | Get enterprise record |
| `GET /api/analytics/datasets` | List datasets |
| `GET /api/analytics/datasets/:id` | Dataset detail with freshness |
| `GET /api/analytics/freshness` | Data freshness for all datasets |
| `GET /api/analytics/events` | Recent analytics events |
| `GET /api/analytics/kpis` | List KPI definitions |
| `POST /api/analytics/kpis` | Create KPI definition |
| `GET /api/analytics/kpis/:id` | Get KPI with live value |
| `GET /api/analytics/kpis/:id/drilldown` | KPI drill-down breakdown |
| `GET /api/analytics/kpis/:id/refresh` | Refresh KPI value |
| `GET /api/analytics/quality` | Data quality report |
| `GET /api/analytics/quality/reports` | List quality reports |
| `GET /api/analytics/quality/issues` | Quality issues |
| `GET /api/analytics/anomalies/rules` | List anomaly rules |
| `GET /api/analytics/anomalies` | List anomalies |
| `POST /api/analytics/anomalies/run` | Run anomaly detection |
| `PUT /api/analytics/anomalies/:id/resolve` | Resolve anomaly |
| `GET /api/analytics/alerts/rules` | List alert rules |
| `POST /api/analytics/alerts/rules` | Create alert rule |
| `GET /api/analytics/alerts` | List alerts |
| `GET /api/analytics/alerts/:id` | Get alert |
| `PUT /api/analytics/alerts/:id/acknowledge` | Acknowledge alert |
| `PUT /api/analytics/alerts/:id/resolve` | Resolve alert |
| `POST /api/analytics/alerts/:id/actions` | Perform alert action |
| `GET /api/analytics/dashboards` | List analytics dashboards |
| `GET /api/analytics/dashboards/:id` | Get dashboard with live data |
| `GET /api/analytics/dashboard?type=X` | Dashboard data by type |
| `GET /api/analytics/intelligence` | Cross-application intelligence |
| `GET /api/analytics/reports` | List report definitions |
| `POST /api/analytics/reports` | Create report definition |
| `GET /api/analytics/reports/:id` | Get report definition |
| `PUT /api/analytics/reports/:id` | Update report definition |
| `POST /api/analytics/reports/:id/generate` | Generate report |
| `GET /api/analytics/reports/runs` | List report runs |
| `GET /api/analytics/reports/runs/:runId` | Get report run |
| `GET /api/analytics/reports/runs/:runId/data` | Get report run data |
| `POST /api/analytics/exports` | Create export |
| `GET /api/analytics/exports` | List exports |
| `GET /api/analytics/exports/:id` | Get export |
| `GET /api/analytics/exports/:id/download` | Download export |
| `GET /api/analytics/search?q=X` | Analytics search |
| `GET /api/analytics/lineage` | Lineage overview |
| `GET /api/analytics/lineage/kpi/:id` | KPI lineage |
| `GET /api/analytics/lineage/dashboard/:id` | Dashboard lineage |
| `GET /api/analytics/lineage/record/:source/:entity/:id` | Record lineage |
| `GET /api/analytics/ai/catalog` | AI resource catalog |
| `GET /api/analytics/ai/kpis` | AI KPIs |
| `GET /api/analytics/ai/trends` | AI trends |
| `GET /api/analytics/ai/quality` | AI quality |
| `GET /api/analytics/ai/events` | AI events |
| `GET /api/analytics/ai/reports` | AI reports |
| `GET /api/analytics/ai/anomalies` | AI anomalies |
| `GET /api/analytics/ai/alerts` | AI alerts |
| `GET /api/analytics/ai/project-activity` | AI project activity |
| `GET /api/analytics/stream` | Real-time SSE stream |

---

## 4. End-to-End Workflow

A field worker submits a survey:

1. **StatCollect** stores the submission.
2. **StatCollect** publishes `submission.received` event.
3. **Enterprise Core** receives the event.
4. **Enterprise Data Layer** ingests the record.
5. **Analytics** updates relevant metrics.
6. **Alert Engine** evaluates alert rules.
7. **Anomaly Detection** checks for unusual patterns.
8. **Real-Time Stream** broadcasts the update to connected dashboards.
9. **Timeline** records the event.
10. **Notifications** are generated where appropriate.
11. **Manager** can drill from KPI to affected survey/project/facility.
12. **Manager** can take action from the analytical view.

---

## 5. No Hardcoded Analytics

**This requirement remains absolute.**

No KPI values, chart values, totals, percentages, geographic values, trends, or dashboard summaries are hardcoded. Every production visualization originates from actual data.

---

## 6. Testing

### 6.1 Build Verification

```bash
go build -C enterprise/core .
```

### 6.2 End-to-End Test

```bash
# Start all services
docker compose up -d

# Verify Enterprise Core
curl http://localhost:8096/health

# List KPI definitions
curl http://localhost:8096/api/analytics/kpis

# Get a KPI with live value
curl "http://localhost:8096/api/analytics/kpis/kpi_active_projects"

# Get KPI drill-down
curl "http://localhost:8096/api/analytics/kpis/kpi_survey_completion_rate/drilldown?scope_id=PRJ-001"

# Get executive dashboard
curl "http://localhost:8096/api/analytics/dashboards/dash_executive"

# Get data quality report
curl "http://localhost:8096/api/analytics/quality"

# List alerts
curl "http://localhost:8096/api/analytics/alerts"

# Run anomaly detection
curl -X POST http://localhost:8096/api/analytics/anomalies/run

# Cross-application intelligence
curl "http://localhost:8096/api/analytics/intelligence?project_id=PRJ-001"

# Analytics search
curl "http://localhost:8096/api/analytics/search?q=survey"

# Data lineage
curl "http://localhost:8096/api/analytics/lineage"

# Create an export
curl -X POST http://localhost:8096/api/analytics/exports \
  -H "Content-Type: application/json" \
  -d '{"dataset": "statcollect:submission", "format": "csv"}'

# Real-time stream
curl -N http://localhost:8096/api/analytics/stream
```

---

## 7. Phase IV Acceptance Criteria Status

- [x] Enterprise data layer is operational
- [x] Real-time analytical updates work
- [x] KPI definitions are configurable and traceable
- [x] KPI drill-down works
- [x] Executive dashboards use live data
- [x] Management dashboards use live data
- [x] Project dashboards use live data
- [x] Field dashboards use live data
- [x] Geographic analytics work where geographic data exists
- [x] Data-quality indicators are implemented
- [x] Rule-based anomaly detection foundation exists
- [x] Analytical alerts work
- [x] Analytics can trigger operational actions
- [x] Enterprise reporting supports live data
- [x] Scheduled reports work
- [x] Authorized exports work
- [x] Analytics are discoverable through Enterprise Search
- [x] Data lineage is available for important metrics
- [x] Permissions are enforced across analytics
- [x] Transactional databases are protected from heavy analytics workloads
- [x] Automated tests cover critical analytical functionality
- [x] No hardcoded production analytics remain
- [x] The complete field-data-to-intelligence workflow has been demonstrated