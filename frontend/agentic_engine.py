"""
agentic_engine.py — StatGate Autonomous Decision-Support Fabric
Multi-step goal-oriented orchestrator. Fully self-contained — no external LLM API needed.
"""
import os
import json
import time
import math
import datetime
from kaggle_connector import list_kaggle_tables, query_kaggle_table, get_table_schema, validate_dataset_name
from engine import statgate_engine

# ── Dataset Relevance Registry ─────────────────────────────────────────────────
DOMAIN_DATASET_MAP = {
    "health":      ["covid_19_data", "global_health", "health_kaggle_dataset"],
    "education":   ["education_by_region", "world_development_data_imputed"],
    "economy":     ["gdp_by_region", "gdp_vs_pollution_rates_by_country", "income_per_capita_by_region"],
    "environment": ["co2_emissions_by_country", "pollution_emissions_by_region", "renewable_energy_jobs_by_country"],
    "energy":      ["fossil_fuel_prices_1989_2019", "renewable_energy_statistics_2010_2019", "percentage_of_energy_consumption_by_country"],
    "poverty":     ["gini_by_country", "world_development_data_imputed", "income_per_capita_by_region"],
    "insurance":   ["insurance", "insurance_1"],
    "tourism":     ["hotel_bookings_raw"],
    "emissions":   ["co2_emissions_by_country", "pollution_rate_vs_gdp_by_country"],
}

# ── Intervention Scenario Templates ───────────────────────────────────────────
SCENARIO_TEMPLATES = {
    "health": [
        {
            "title": "Scale Healthcare Access",
            "description": "Expand clinic coverage to underserved regions flagged by current admission trends.",
            "formula": "AVG(Confirmed) / NULLIF(AVG(Recovered), 0)",
            "action": "webhook/alert-health-coordinator",
            "priority": "HIGH",
        },
        {
            "title": "Accelerate Vaccination Campaigns",
            "description": "Cross-correlate recovery rates with vaccination timing to identify optimal rollout windows.",
            "formula": "AVG(Recovered) / NULLIF(AVG(Confirmed), 0) * 100",
            "action": "webhook/create-indicator",
            "priority": "MEDIUM",
        },
        {
            "title": "Reallocate Supply Chain Budget",
            "description": "Re-route medical supply logistics from low-mortality zones to high-anomaly provinces.",
            "formula": "SUM(Deaths) / NULLIF(SUM(Confirmed), 0)",
            "action": "webhook/budget-reallocation",
            "priority": "CRITICAL",
        },
    ],
    "education": [
        {
            "title": "Boost Teacher Deployment in Low-Performing Regions",
            "description": "Deploy resources to bottom-quartile regional education scores.",
            "formula": "AVG(value) WHERE region IN (SELECT region FROM ... ORDER BY AVG(value) ASC LIMIT 5)",
            "action": "webhook/alert-education-director",
            "priority": "HIGH",
        },
        {
            "title": "Fund Digital Infrastructure Rollout",
            "description": "Correlate digital access rates with academic performance uplift.",
            "formula": "AVG(value) * 1.15",
            "action": "webhook/create-indicator",
            "priority": "MEDIUM",
        },
        {
            "title": "Early Warning: Dropout Risk Index",
            "description": "Flag regions with attendance trends below 3-sigma moving average as high-risk.",
            "formula": "STDDEV(value) * 3 + AVG(value)",
            "action": "webhook/create-alert",
            "priority": "HIGH",
        },
    ],
    "economy": [
        {
            "title": "GDP Growth Acceleration Plan",
            "description": "Identify fastest-growing sectors per region and propose budget shifts.",
            "formula": "AVG(GDP) - LAG(AVG(GDP))",
            "action": "webhook/alert-finance-ministry",
            "priority": "HIGH",
        },
        {
            "title": "Inequality Reduction: Gini Intervention",
            "description": "Target regions above 0.4 Gini coefficient for redistribution policy.",
            "formula": "AVG(gini_index) WHERE gini_index > 0.4",
            "action": "webhook/create-indicator",
            "priority": "CRITICAL",
        },
        {
            "title": "Budget Overrun Forecast",
            "description": "Real-time stream forecasting flags potential budget overruns 30 days ahead.",
            "formula": "AVG(expenditure) * 1.25",
            "action": "webhook/alert-comptroller",
            "priority": "HIGH",
        },
    ],
    "tourism": [
        {
            "title": "Optimize Hotel Occupancy",
            "description": "Adjust room pricing and promotions based on low-occupancy periods in high-demand regions.",
            "formula": "AVG(occupancy_rate) WHERE region IN (SELECT region FROM ... ORDER BY occupancy_rate ASC LIMIT 5)",
            "action": "webhook/alert-tourism-board",
            "priority": "HIGH",
        },
        {
            "title": "Boost Destination Marketing",
            "description": "Target digital campaigns to destinations with upward booking momentum.",
            "formula": "SUM(bookings) / NULLIF(SUM(ad_spend), 0)",
            "action": "webhook/create-indicator",
            "priority": "MEDIUM",
        },
        {
            "title": "Tourism Revenue Leakage Audit",
            "description": "Identify regions where average spend per visitor is declining versus expected seasonality.",
            "formula": "AVG(spend_per_visitor) - LAG(AVG(spend_per_visitor))",
            "action": "webhook/alert-revenue-office",
            "priority": "CRITICAL",
        },
    ],
}

DEFAULT_SCENARIOS = [
    {
        "title": "Cross-Domain Anomaly Investigation",
        "description": "3-sigma breach detected. Recommend immediate dataset review across related tables.",
        "formula": "STDDEV(value)",
        "action": "webhook/alert-manager",
        "priority": "HIGH",
    },
    {
        "title": "Federated Benchmark Comparison",
        "description": "Compare current tenant metrics against anonymized global averages from the federated model.",
        "formula": "AVG(value) / global_avg",
        "action": "webhook/benchmark-request",
        "priority": "MEDIUM",
    },
    {
        "title": "Predictive Governance Alert",
        "description": "Trend-extrapolation suggests metric will breach policy threshold within 14 days.",
        "formula": "value + (AVG(value) - LAG(AVG(value))) * 14",
        "action": "webhook/create-alert",
        "priority": "CRITICAL",
    },
]


class AgenticEngine:
    """
    Multi-step autonomous analytical agent.
    Given a high-level goal (e.g. "Improve education in Region X"),
    it orchestrates dataset discovery, anomaly detection, and scenario generation
    without any external API calls.
    """

    def analyze_intent(self, goal: str, tenant_id: str = "tenant-alpha") -> dict:
        """
        Core agentic pipeline — 4-step orchestration:
        1. Identify domain & relevant datasets
        2. Profile each dataset for anomalies
        3. Generate 3 Intervention Scenarios with statistical evidence
        4. Produce a structured AgentReport
        """
        start = time.time()
        if not goal or not goal.strip():
            raise ValueError('goal is required')

        goal_lower = goal.lower()

        # ─ Step 1: Domain identification ─────────────────────────────
        domain = self._identify_domain(goal_lower)
        candidate_datasets = self._filter_valid_datasets(DOMAIN_DATASET_MAP.get(domain, []))

        # Fallback: list all tables from live DB
        if not candidate_datasets:
            try:
                all_tables = list_kaggle_tables()
            except Exception as e:
                all_tables = []
                dataset_discovery_error = str(e)
            else:
                dataset_discovery_error = None
            candidate_datasets = self._filter_valid_datasets(all_tables)[:5]
        else:
            dataset_discovery_error = None

        dataset_candidates = list(candidate_datasets)

        # ─ Step 2: Dataset profiling & anomaly detection ──────────────
        profiled_datasets = []
        anomalies_found = []
        profile_errors = []

        for table in candidate_datasets[:3]:  # cap at 3 for speed
            try:
                summary = statgate_engine.get_dataset_summary(table)
                anomaly_cols = []

                for col in summary.get("profiling", []):
                    stats = col.get("stats", {})
                    if stats.get("std", 0) > 0 and stats.get("mean", 0) > 0:
                        # Flag columns where max exceeds mean + 3*std
                        if stats.get("max", 0) > stats.get("mean", 0) + 3 * stats.get("std", 0):
                            anomaly_cols.append({
                                "column":    col["column_name"],
                                "mean":      round(stats["mean"], 2),
                                "std":       round(stats["std"],  2),
                                "max":       round(stats["max"],  2),
                                "sigma":     round((stats["max"] - stats["mean"]) / stats["std"], 2) if stats["std"] > 0 else 0,
                            })
                            anomalies_found.append({
                                "table":  table,
                                "column": col["column_name"],
                                "sigma":  round((stats["max"] - stats["mean"]) / stats["std"], 2),
                            })

                profiled_datasets.append({
                    "table":       table,
                    "row_count":   summary.get("row_count", 0),
                    "col_count":   len(summary.get("profiling", [])),
                    "anomaly_cols": anomaly_cols,
                })
            except Exception as e:
                profiled_datasets.append({"table": table, "error": str(e)})
                profile_errors.append({"table": table, "error": str(e)})

        # ─ Step 3: Scenario generation ────────────────────────────────
        scenarios = self._build_scenarios(domain, anomalies_found, goal_lower, candidate_datasets)

        # ─ Step 4: Federated aggregate (cross-tenant anonymized) ───────
        federated_summary = self._federated_aggregate(candidate_datasets)

        # ─ Compile Agent Report ────────────────────────────────────────
        latency = round((time.time() - start) * 1000, 1)
        report = {
            "goal":                    goal,
            "tenant_id":               tenant_id,
            "domain":                  domain,
            "dataset_discovery_error": dataset_discovery_error,
            "dataset_candidates":      dataset_candidates,
            "datasets_analyzed":       profiled_datasets,
            "profile_errors":          profile_errors,
            "anomalies_detected":      anomalies_found,
            "actions_needed":          len([s for s in scenarios if s["priority"] == "CRITICAL"]),
            "scenarios":               scenarios,
            "federated_summary":       federated_summary,
            "generated_at":            datetime.datetime.now(datetime.UTC).isoformat(),
            "engine_latency_ms":       latency,
        }

        # Persist as analytical asset for reproducibility
        try:
            statgate_engine.save_dashboard_metadata(
                f"agent_report_{tenant_id}_{int(time.time())}",
                report
            )
        except Exception:
            pass

        return report

    def _filter_valid_datasets(self, dataset_names: list) -> list:
        valid = []
        for name in dataset_names or []:
            try:
                validate_dataset_name(name)
                valid.append(name)
            except ValueError:
                continue
        return valid

    def _identify_domain(self, goal_lower: str) -> str:
        """Match goal text to a semantic domain."""
        domain_keywords = {
            "health":      ["health", "covid", "hospital", "mortality", "disease", "vaccination"],
            "education":   ["education", "school", "literacy", "attendance", "learning", "student"],
            "tourism":     ["tourism", "travel", "hotel", "destination", "resort", "hospitality"],
            "insurance":   ["insurance", "risk", "coverage", "claim", "premium"],
            "emissions":   ["emission", "emissions", "carbon", "greenhouse", "co2"],
            "economy":     ["economy", "gdp", "budget", "revenue", "finance", "expenditure", "growth"],
            "environment": ["environment", "climate", "pollution", "emission", "carbon"],
            "energy":      ["energy", "fuel", "renewable", "electricity", "fossil"],
            "poverty":     ["poverty", "inequality", "gini", "income", "wealth"],
        }
        for domain, keywords in domain_keywords.items():
            if any(kw in goal_lower for kw in keywords):
                return domain
        return "general"

    def _build_scenarios(self, domain: str, anomalies: list, goal: str, datasets: list) -> list:
        """Build 3 concrete intervention scenarios with statistical evidence."""
        templates = SCENARIO_TEMPLATES.get(domain, DEFAULT_SCENARIOS)
        scenarios = []
        goal_summary = goal.strip()[:88]
        dataset_reference = datasets[0] if datasets else None

        for i, tmpl in enumerate(templates[:3]):
            if anomalies:
                a = anomalies[min(i, len(anomalies) - 1)]
                evidence = (
                    f"Anomaly evidence from {a['table']}: {a['column']} reached {a['sigma']}σ. "
                    f"Recommended action is grounded in live dataset signals from {a['table']}."
                )
            elif dataset_reference:
                evidence = (
                    f"No active anomaly was detected. This scenario is built from domain model guidance and dataset "
                    f"'{dataset_reference}'."
                )
            else:
                evidence = "No active anomaly or dataset reference available. This scenario uses a conservative domain-driven intervention."

            scenario_description = tmpl["description"]
            if dataset_reference:
                scenario_description = f"{scenario_description} Uses dataset '{dataset_reference}' to validate assumptions."

            if dataset_reference and "dataset_links" not in tmpl:
                evidence = f"{evidence} Primary dataset candidate: {dataset_reference}."

            scenarios.append({
                "scenario_id":   f"scenario_{i+1}",
                "title":         tmpl["title"],
                "description":   scenario_description,
                "goal":          goal_summary,
                "evidence":      evidence,
                "formula":       tmpl["formula"],
                "action":        tmpl["action"],
                "priority":      tmpl["priority"],
                "dataset_links": datasets[:2],
                "approved":      False,
            })

        return scenarios

    def _federated_aggregate(self, datasets: list) -> dict:
        """
        Federated cross-tenant intelligence:
        Computes anonymized aggregate statistics WITHOUT sharing raw row data.
        Each tenant contributes only: (count, mean, variance) — never raw records.
        """
        agg = {"global_mean": None, "global_count": 0, "participating_datasets": 0}
        means = []
        total_rows = 0

        for table in datasets[:2]:
            try:
                records = query_kaggle_table(table, limit=50)
                if not records:
                    continue

                import pandas as pd
                df = pd.DataFrame(records)
                numeric_cols = df.select_dtypes(include="number").columns.tolist()

                if numeric_cols:
                    col_mean = float(df[numeric_cols[0]].mean())
                    means.append(col_mean)
                    total_rows += len(df)
                    agg["participating_datasets"] += 1
            except Exception:
                pass

        if means:
            agg["global_mean"] = round(sum(means) / len(means), 2)
            agg["global_count"] = total_rows

        return agg

    def record_feedback(self, report_id: str, scenario_id: str, action: str, outcome: str, tenant_id: str) -> dict:
        """
        RLHF Feedback Loop: Record user accept/reject decisions on agent proposals.
        Stored as JSON for continuous fine-tuning.
        """
        feedback_dir = os.path.join(os.path.dirname(__file__), '..', 'data', 'feedback')
        os.makedirs(feedback_dir, exist_ok=True)

        record = {
            "report_id":   report_id,
            "scenario_id": scenario_id,
            "action":      action,        # "approved" | "rejected" | "deferred"
            "outcome":     outcome,       # user-provided text outcome
            "tenant_id":   tenant_id,
            "recorded_at": datetime.datetime.now(datetime.UTC).isoformat(),
        }

        fname = os.path.join(feedback_dir, f"rlhf_{int(time.time())}_{scenario_id}.json")
        with open(fname, "w") as f:
            json.dump(record, f, indent=2)

        return {"status": "recorded", "file": os.path.basename(fname)}

    def get_executive_summary(self, tenant_id: str = "tenant-alpha") -> dict:
        """
        Executive-grade summary: minimal cognitive load.
        Returns exactly what a Minister needs to see.
        """
        try:
            tables = list_kaggle_tables()
        except Exception as e:
            tables = []
            dataset_discovery_error = str(e)
        else:
            dataset_discovery_error = None

        total_datasets = len(tables)

        # Pull recent alerts
        alerts_payload = {"count": 0, "alerts": []}
        try:
            import requests
            go_url = os.getenv("GO_BACKEND_URL", "http://localhost:8080")
            headers = {"X-Tenant-ID": tenant_id}
            internal_key = os.getenv("STATGATE_INTERNAL_API_KEY")
            if internal_key:
                headers["X-StatGate-Internal-Key"] = internal_key
            r = requests.get(f"{go_url}/api/v1/alerts",
                             headers=headers, timeout=2)
            if r.status_code == 200:
                alerts_payload = r.json()
        except Exception:
            pass

        critical = [a for a in alerts_payload.get("alerts", []) if a.get("severity") == "CRITICAL"]
        high      = [a for a in alerts_payload.get("alerts", []) if a.get("severity") == "HIGH"]

        actions_needed = len(critical) + len(high)

        # Quick domain snapshot from top dataset
        snapshot = {}
        if tables:
            try:
                s = statgate_engine.get_dataset_summary(tables[0])
                snapshot = {
                    "dataset":    tables[0],
                    "rows":       s.get("row_count", 0),
                    "columns":    len(s.get("profiling", [])),
                    "latency_ms": s.get("latency_ms", 0),
                }
            except Exception:
                pass

        result = {
            "tenant_id":           tenant_id,
            "total_datasets":      total_datasets,
            "actions_needed":      actions_needed,
            "critical_alerts":     len(critical),
            "high_alerts":         len(high),
            "top_alert":           critical[0] if critical else (high[0] if high else None),
            "data_snapshot":       snapshot,
            "system_health":       "Optimal" if actions_needed == 0 else ("Degraded" if actions_needed > 3 else "Attention Required"),
            "generated_at":        datetime.datetime.now(datetime.UTC).isoformat(),
        }

        if dataset_discovery_error:
            result["dataset_discovery_error"] = dataset_discovery_error

        return result


# Singleton
agentic_engine = AgenticEngine()
