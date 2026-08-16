# PHASE 8 — BUSINESS INTELLIGENCE, ANALYTICS & DECISION SUPPORT

## Objective

Build the complete analytics ecosystem that transforms raw data into actionable intelligence.

This phase enables users to create dashboards, reports, scorecards, KPIs, forecasts, and executive insights without relying on external Business Intelligence platforms.

StatGate becomes a complete decision-support system.

---

## Vision

Every decision is supported by trusted evidence, real-time analytics, predictive models, and interactive visualisations. The platform eliminates the need for Power BI, Tableau, or similar external tools for institutional reporting.

---

## Services Involved

| Service | Technology | Port |
|---|---|---|
| Enterprise Core (analytics layer) | Go + Gin | :8096 |
| Analytics UI (Flask) | Python / Flask | :5000 |

---

## Service: Enterprise Core (Analytics Layer)

**Relevant files in `enterprise/core/`:**

```
analytics_dashboards.go   ← dashboard CRUD + widget engine
analytics_kpi.go          ← KPI definitions, measurements, drill-down, refresh
analytics_reports.go      ← report generation and scheduling
analytics_alerts.go       ← alert rules and triggered alerts
analytics_anomaly.go      ← anomaly detection engine
analytics_export.go       ← report export (PDF, Excel, CSV)
analytics_stream.go       ← real-time analytics streaming
analytics_ai.go           ← AI-assisted analytics features
command_centre.go         ← executive command centre (6 views)
analytics_models.go       ← data models for all analytics entities
```

---

## What Exists ✅

### Dashboards (`analytics_dashboards.go`)

```
GET    /api/dashboards                         ← list dashboards
GET    /api/dashboards/:id                     ← dashboard detail
POST   /api/dashboards                         ← create dashboard
PUT    /api/dashboards/:id                     ← update dashboard
DELETE /api/dashboards/:id                     ← delete dashboard
POST   /api/dashboards/:id/share               ← share dashboard
GET    /api/dashboards/:id/widgets/:wid/data   ← widget data (live query)
GET    /api/widgets                            ← available widget types
GET    /api/widgets/:type/preview              ← widget type preview
```

**Supported widget types:**
- Line chart, bar chart, area chart, pie/donut chart
- KPI card, gauge, progress bar
- Table, data grid
- Heat map, scatter plot
- Map widget (placeholder for StatSpatial)

### KPI Management (`analytics_kpi.go`)

```
GET    /api/analytics/kpis                     ← KPI definitions list
POST   /api/analytics/kpis                     ← create KPI definition
GET    /api/analytics/kpis/:id                 ← KPI detail with current value
GET    /api/analytics/kpis/:id/drilldown       ← drill-down data for KPI
GET    /api/analytics/kpis/:id/refresh         ← refresh KPI value
POST   /api/analytics/kpis/:id/refresh         ← trigger KPI refresh
```

**KPI features:**
- Definitions with formula, data source, target, threshold, trend direction
- Auto-measurement on schedule
- Drill-down to underlying data
- Traffic-light status (on-track, at-risk, off-track)

### Reports (`analytics_reports.go`)

```
GET    /api/reports                            ← list reports
GET    /api/reports/:id                        ← report detail
POST   /api/reports                            ← create / generate report
GET    /api/reports/:id/download               ← download report (PDF/Excel/CSV)
DELETE /api/reports/:id                        ← delete report
```

**Agentic report engine** (`frontend/agentic_engine.py`):
- Rule-based automated report generation
- Agent feedback loops for quality improvement
- Scheduled report generation
- Report approval workflow

### Anomaly Detection (`analytics_anomaly.go`)

```
GET    /api/analytics/anomalies/rules          ← anomaly detection rules
GET    /api/analytics/anomalies                ← detected anomalies list
POST   /api/analytics/anomalies/run            ← trigger anomaly detection run
PUT    /api/analytics/anomalies/:id/resolve    ← mark anomaly as resolved
```

**3-sigma statistical anomaly detection:**
- Configurable per indicator or dataset column
- Publishes alerts when anomalies are detected
- Integrates with alert rules for notifications

### Alert Rules (`analytics_alerts.go`)

```
GET    /api/analytics/alerts/rules             ← alert rule definitions
POST   /api/analytics/alerts/rules             ← create alert rule
GET    /api/analytics/alerts                   ← triggered alerts list
```

### Export (`analytics_export.go`)

```
GET    /api/analytics/export/:id               ← export dataset to format
```

**Supported formats:** PDF, Excel (XLSX), CSV

### Executive Command Centre (`command_centre.go`)

```
GET    /api/command-centre                     ← 6-view executive intelligence hub
GET    /api/command-centre/overview            ← platform overview KPIs
GET    /api/command-centre/risks               ← enterprise risk summary
GET    /api/command-centre/projects            ← project portfolio summary
GET    /api/command-centre/research            ← research portfolio summary
GET    /api/command-centre/operations          ← operational metrics
GET    /api/command-centre/intelligence        ← institutional intelligence signals
```

### Real-Time Analytics Stream (`analytics_stream.go`)

```
GET    /api/analytics/stream                   ← SSE stream of live analytics events
```

### Analytics UI (Flask — `:5000`)

| Route | Description |
|---|---|
| `/` | Analytical dashboard |
| `/datasets` | Dataset catalog and schema explorer |
| `/notebook` | Interactive Python/DuckDB notebook |
| `/executive` | Executive command centre view |
| `/semantic` | Semantic indicator registry |
| `/abac` | ABAC policy matrix |
| `/api/datasets/*` | Dataset CRUD and schema APIs |
| `/api/agent/*` | Agentic report engine |
| `/api/schema-health/*` | Schema health monitoring |

**Analysis Engine** (`engine.py`): Pandas + DuckDB SQL-on-dataframe for ad-hoc analysis

---

## What is Missing ❌

### Self-Service Analytics
- **OLAP / Data Cubes** — no multidimensional cube calculations
- **Pivot Tables** — no server-side pivot; frontend only
- **Ad-hoc query builder** — no drag-and-drop query builder UI
- **Cross-filtering** — not implemented in frontend
- **Drill-through analysis** — implemented in backend, not wired in frontend

### Predictive & Advanced Analytics
- **Predictive Analytics / Forecasting** — anomaly detection exists; statistical forecasting (ARIMA, exponential smoothing) not implemented
- **Scenario Analysis / What-If** — not implemented
- **Simulation** — not implemented
- **Benchmarking** — not implemented

### AI-Powered Analytics
- **Natural Language Queries (NLQ)** — not implemented; no LLM integration for "query in plain English"
- **Conversational Analytics** — not implemented
- **AI-generated narrative insights** — analytics_ai.go exists but LLM not connected
- **Automatic dashboard generation** — not implemented

### Reporting
- **Scheduled report subscriptions** — not implemented (scheduler exists as stub)
- **PowerPoint / Word export** — not implemented
- **Report sharing via email** — not implemented

### Visualisation
- **Sankey diagrams, Waterfall charts, Radar charts** — not implemented
- **Network graphs** — not implemented
- **Geospatial analytics in dashboards** — placeholder only; requires StatSpatial

---

## Integration Points

| Module | Integration |
|---|---|
| Analytics Core (`:8082`) | Data source for Enterprise Core analytics via internal API |
| Enterprise Core (`:8096`) | Hosts dashboard engine, KPI engine, report engine |
| Enterprise Search (`:8095`) | Dashboard and report search |
| StatChat | Alert notifications pushed to conversation threads |
| StatGovernance | Governance KPIs tracked in command centre |
| PMS | Project KPIs surfaced in command centre |
| Phase XII (Phase 13) | Institutional intelligence signals feed command centre |

---

## Acceptance Criteria

- [x] Dashboard builder (CRUD + widget engine) operational
- [x] KPI definitions, measurements, and drill-down operational
- [x] Report generation and download (PDF, Excel, CSV) operational
- [x] Anomaly detection (3-sigma) operational
- [x] Alert rules and triggered alerts operational
- [x] Executive command centre (6 views) operational
- [x] Agentic automated report engine operational
- [x] Analytics UI (Flask) with dataset catalog and notebook operational
- [x] Real-time analytics stream operational
- [ ] Ad-hoc query builder (drag-and-drop) operational
- [ ] Statistical forecasting operational
- [ ] Natural language queries operational
- [ ] Conversational analytics operational
- [ ] Scheduled report subscriptions operational
- [ ] Geospatial analytics in dashboards operational

---

## Ports & Services

| Component | Port |
|---|---|
| Enterprise Core — analytics layer (Go) | :8096 |
| Analytics Core — lakehouse/ABAC/semantic (Go) | :8082 |
| Analytics UI (Flask) | :5000 |

---

## Estimated Duration

12 weeks

## Milestone

Enterprise Analytics Platform complete. Dashboards, KPIs, reports, anomaly detection, and executive intelligence operational without external BI tools.
