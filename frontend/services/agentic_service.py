DOMAIN_KEYWORDS = {
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


def identify_domain(goal_text):
    goal_lower = (goal_text or '').lower()
    for domain, keywords in DOMAIN_KEYWORDS.items():
        if any(keyword in goal_lower for keyword in keywords):
            return domain
    return 'general'


def build_scenarios(domain, anomalies=None, goal=None, datasets=None):
    templates = SCENARIO_TEMPLATES.get(domain, DEFAULT_SCENARIOS)
    anomalies = anomalies or []
    datasets = datasets or []
    scenarios = []

    for index, template in enumerate(templates[:3]):
        evidence = ''
        if anomalies:
            anomaly = anomalies[min(index, len(anomalies) - 1)]
            evidence = f"Statistical evidence: {anomaly['column']} = {anomaly['sigma']}σ deviation in {anomaly['table']}."
        else:
            evidence = 'No active anomaly — proactive scenario based on domain model.'

        scenarios.append({
            'scenario_id': f'scenario_{index + 1}',
            'title': template['title'],
            'description': template['description'],
            'evidence': evidence,
            'formula': template['formula'],
            'action': template['action'],
            'priority': template['priority'],
            'dataset_links': datasets[:2],
            'approved': False,
        })

    return scenarios
