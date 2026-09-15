# PHASE 12 — MONITORING, EVALUATION, RESULTS MANAGEMENT & IMPACT ASSESSMENT

## Objective

Develop the complete Monitoring and Evaluation (M&E) ecosystem for tracking organizational performance, projects, programs, policies, indicators, and development outcomes.

The platform supports continuous monitoring, periodic evaluations (baseline/midline/endline), adaptive management, Results Frameworks, and evidence-based decision-making.

---

## Vision

Every project, program, strategic plan, and national/institutional policy shall be measurable in real time through structured indicators, verifiable evidence, and automated performance signals.

---

## Services Involved

| Service | Technology | Port |
|---|---|---|
| **Enterprise Core (M&E & Strategic Layer)** | Go + Gin | :8096 |
| **Analytics UI / Command Centre** | Python/Flask + React | :5000 / :3006 |
| **PMS / RMS Integration Layer** | Go + Gin | :8091 / :8092 |

---

## What Exists ✅

### 1. Strategic Objectives & KPI Framework (`enterprise/core/`)
- **Strategic Hierarchy Models (`phase12_models_strategy.go`):**
  - Institutional Goals → Strategic Objectives → Programmes → KPIs
- **KPI Management Engine (`analytics_kpi.go`, `phase12_kpi*.go`):**
  ```
  GET    /api/analytics/kpis                     ← list KPI definitions
  POST   /api/analytics/kpis                     ← create KPI definition
  GET    /api/analytics/kpis/:id                 ← get KPI detail & status
  GET    /api/analytics/kpis/:id/drilldown       ← multidimensional breakdown
  POST   /api/analytics/kpis/:id/refresh         ← recalculate measurement
  GET    /api/intelligence/objectives            ← strategic objectives tree
  POST   /api/intelligence/objectives            ← create strategic objective
  ```
- **Measurement Engine:** Automated periodic measurement capture, baseline vs target tracking, variance thresholds.

### 2. Institutional Signals & Condition Engine (`phase12_conditions*.go`)
- Real-time intelligence signals (`intelligence_signals`, `institutional_conditions`)
- Status levels: `OPTIMAL` → `WATCH` → `ELEVATED` → `CRITICAL` → `EMERGENCY`
- Explainability engine outputting audit trail of why a KPI/condition changed state

### 3. Executive Command Centre (`command_centre.go`)
- 6-view executive command centre aggregating project health, research milestones, risk alerts, and strategic KPI progress.

---

## What is Missing ❌

### 1. Results Framework & Planning Tools
- **Logical Framework (LogFrame) Designer:**
  ```
  GET    /api/me/logframes/:id                   ← LogFrame matrix (Impact, Outcome, Output, Activity)
  POST   /api/me/logframes                       ← create LogFrame
  PUT    /api/me/logframes/:id                   ← update indicators & means of verification
  ```
- **Theory of Change (ToC) Visualizer:** Pathway diagram defining inputs, activities, outputs, outcomes, assumptions, and causal links.
- **Results Framework Matrix:** Institutional scorecard aligning multi-program outcomes to national/SDG goals.

### 2. Evaluation Study Management
- **Study Lifecycle Engine:**
  ```
  GET    /api/me/evaluations                     ← list evaluation studies
  POST   /api/me/evaluations                     ← schedule baseline/midline/endline study
  GET    /api/me/evaluations/:id/protocols       ← evaluation methodology & tools
  POST   /api/me/evaluations/:id/findings        ← record evaluation findings & ratings
  ```
- **Beneficiary Tracking Engine:** Disaggregated demographic beneficiary tracking across interventions.
- **Qualitative M&E Modules:**
  - Most Significant Change (MSC) story bank
  - Outcome Harvesting registry

### 3. Recommendations & Adaptive Learning
- **Recommendations Tracker:**
  ```
  GET    /api/me/recommendations                 ← list evaluation recommendations
  POST   /api/me/recommendations                 ← register management action response
  PUT    /api/me/recommendations/:id/status      ← track adoption and corrective action closure
  ```
- **SDG & National Development Plan Alignment:** Pre-built SDG indicator catalog with target progress scorecards.

---

## Database Tables (`enterprise` / `statgate`)

```sql
-- Existing Core
strategic_objectives (id, title, description, parent_id, weight, target_date, status)
kpi_definitions (id, objective_id, code, name, formula, data_source, target_value, baseline_value, unit, frequency)
kpi_measurements (id, kpi_id, measured_value, period_start, period_end, source_evidence_id, measured_at)
intelligence_signals (id, domain, signal_type, severity, confidence, payload, generated_at)

-- Missing / To Build
me_logframes (id, entity_type, entity_id, narrative_summary, assumptions, created_at)
me_logframe_items (id, logframe_id, level, description, indicators JSONB, mov JSONB, assumptions TEXT)
me_evaluations (id, entity_id, type, lead_evaluator, term_of_reference, budget, start_date, end_date, report_url, status)
me_beneficiaries (id, project_id, individual_id, household_id, gender, age_group, location_id, intervention_type, verified)
me_recommendations (id, evaluation_id, recommendation_text, priority, responsible_unit, action_plan, status, deadline)
```

---

## Integration Points

| Module | Integration Flow |
|---|---|
| **PMS (:8091)** | Outputs from project deliverables feed directly into LogFrame milestone achievements |
| **StatCollect (:8080)** | Survey datasets serve as automated verification evidence for indicator measurement values |
| **StatGovernance (:8093)** | Evaluation recommendations and compliance actions automatically linked to audit CAPA logs |
| **Enterprise AI (:8096)** | Generates automated narrative progress summaries and performance deviation forecasts |

---

## Acceptance Criteria

- [x] Strategic objectives hierarchy and KPI registry operational
- [x] KPI measurement recalculation and drilldown operational
- [x] Institutional condition engine computing health signals
- [x] Executive command centre operational
- [x] LogFrame builder operational (Impact → Outcome → Output → Activity)
- [x] Theory of Change visual workflow builder operational
- [ ] Baseline, Midline, and Endline evaluation management operational
- [ ] Beneficiary disaggregation tracking operational
- [ ] Evaluation recommendations tracker with management action workflow operational
- [ ] Pre-seeded SDG 1–17 indicator framework operational

---

## Ports & Services

| Component | Port |
|---|---|
| Enterprise Core M&E APIs (Go) | :8096 |
| Analytical Hub / Command Centre (Flask/React) | :5000 / :3006 |

---

## Estimated Duration

10 weeks

## Milestone

Enterprise Monitoring & Evaluation Platform Complete. Real-time indicators, LogFrames, evaluations, and strategic impact monitored across all platform activities.
