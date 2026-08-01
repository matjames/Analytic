SEMANTIC_ALIASES = {
    "performance": ["confirmed", "recovered", "deaths"],
    "outbreak": ["confirmed", "country_per_region", "observationdate"],
    "attendance": ["value", "region", "country"],
    "mortality": ["deaths", "confirmed"],
    "education": ["value", "region"],
    "health": ["confirmed", "recovered", "deaths"],
    "funding": ["value", "metric_value"],
    "revenue": ["value", "metric_value"],
    "response": ["value", "latency_ms"],
    "latency": ["latency_ms", "value"],
}


def resolve_nlq_term(prompt_lower):
    """Return best matching semantic indicator and related columns."""
    matches = []
    for key, cols in SEMANTIC_ALIASES.items():
        idx = prompt_lower.find(key)
        if idx != -1:
            matches.append((idx, key, cols))

    if not matches:
        return None, []

    matches.sort(key=lambda item: item[0])
    _, best_key, best_cols = matches[0]
    return best_key, best_cols


def infer_dataset_for_prompt(prompt_lower):
    if any(w in prompt_lower for w in ["gdp", "economy", "growth"]):
        return "gdp_by_region"
    if any(w in prompt_lower for w in ["education", "school", "literacy"]):
        return "education_by_region"
    if any(w in prompt_lower for w in ["energy", "fossil", "renewable"]):
        return "renewable_energy_jobs_by_country"
    if any(w in prompt_lower for w in ["health", "covid", "mortality", "cases"]):
        return "covid_19_data"
    return "covid_19_data"


def build_nlq_response(prompt, tenant_id):
    if not prompt or not prompt.strip():
        raise ValueError("Empty prompt")

    prompt_lower = prompt.strip().lower()
    term, related_cols = resolve_nlq_term(prompt_lower)
    semantic_label = term.title() if term else "General Query"
    resolved_formula = f"AVG({related_cols[0]})" if related_cols else "COUNT(*)"
    expanded_cols = ", ".join(related_cols[:3]) if related_cols else "*"
    dataset = infer_dataset_for_prompt(prompt_lower)

    generated_sql = (
        f"SELECT {expanded_cols}, {resolved_formula} AS computed_metric\n"
        f"FROM ml_staging.{dataset}\n"
        f"WHERE tenant_id = '{tenant_id}'\n"
        f"GROUP BY {related_cols[1] if len(related_cols) > 1 else expanded_cols.split(',')[0]}\n"
        f"ORDER BY computed_metric DESC\n"
        f"LIMIT 20;"
    )

    return {
        "semantic_label": semantic_label,
        "dataset": dataset,
        "generated_sql": generated_sql,
        "resolved_columns": related_cols,
        "resolved_formula": resolved_formula,
    }
