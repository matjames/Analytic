"""
schema_healer.py — Self-Healing Data Pipeline
Detects schema drift, infers column mappings, and auto-patches analytical assets.
"""
import os
import re
import json
import datetime
from kaggle_connector import get_table_schema

SNAPSHOT_DIR = os.path.join(os.path.dirname(__file__), '..', 'data', 'schema_snapshots')
DASHBOARDS_DIR = os.path.join(os.path.dirname(__file__), '..', 'data', 'dashboards')

# Semantic column synonym groups — used for auto-mapping on drift
SYNONYM_GROUPS = [
    {"names": ["country", "country_name", "nation", "country_per_region", "country_region"], "canonical": "country"},
    {"names": ["date", "dt", "observationdate", "timestamp", "created_at", "record_date"], "canonical": "date"},
    {"names": ["value", "metric_value", "score", "amount", "quantity", "count"],           "canonical": "value"},
    {"names": ["region", "province", "state", "province_per_state", "area"],               "canonical": "region"},
    {"names": ["deaths", "fatalities", "mortality_count", "death_count"],                  "canonical": "deaths"},
    {"names": ["confirmed", "cases", "total_cases", "case_count", "infections"],           "canonical": "confirmed"},
    {"names": ["recovered", "recovery", "recoveries", "healed"],                           "canonical": "recovered"},
    {"names": ["gdp", "gross_domestic_product", "gdp_value", "gdp_usd"],                   "canonical": "gdp"},
]


def _save_snapshot(table_name: str, schema: list):
    os.makedirs(SNAPSHOT_DIR, exist_ok=True)
    path = os.path.join(SNAPSHOT_DIR, f"{table_name}.json")
    with open(path, "w") as f:
        json.dump({
            "table_name":   table_name,
            "snapshotted_at": datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z'),
            "columns":      schema,
        }, f, indent=2)


def _load_snapshot(table_name: str) -> list:
    path = os.path.join(SNAPSHOT_DIR, f"{table_name}.json")
    if not os.path.exists(path):
        return []
    with open(path) as f:
        return json.load(f).get("columns", [])


def _normalize_column_name(col_name: str) -> str:
    normalized = re.sub(r'[^a-z0-9]+', '_', col_name.strip().lower())
    for group in SYNONYM_GROUPS:
        if normalized in [name.lower() for name in group["names"]]:
            return group["canonical"]
    return normalized


def _find_canonical(col_name: str) -> str:
    return _normalize_column_name(col_name)


def _infer_column_mapping(removed_cols: list, added_cols: list) -> tuple[list, dict]:
    renamed_cols = []
    mapping = {}
    unmatched_added = set(added_cols)

    for removed in removed_cols:
        canonical_removed = _find_canonical(removed)
        best_match = None

        for added in list(unmatched_added):
            if _find_canonical(added) == canonical_removed:
                best_match = added
                break

        if best_match:
            renamed_cols.append({"from": removed, "to": best_match})
            mapping[removed] = best_match
            unmatched_added.remove(best_match)

    return renamed_cols, mapping


def detect_drift(table_name: str) -> dict:
    """
    Compare live information_schema against last saved snapshot.
    Returns: { drifted, added_cols, removed_cols, renamed_cols, mapping }
    """
    live_schema = get_table_schema(table_name)
    old_schema = _load_snapshot(table_name)

    live_cols = {c["column_name"]: c["data_type"] for c in live_schema}
    old_cols = {c["column_name"]: c["data_type"] for c in old_schema}

    added_cols = [c for c in live_cols if c not in old_cols]
    removed_cols = [c for c in old_cols if c not in live_cols]

    renamed_cols, mapping = _infer_column_mapping(removed_cols, added_cols)
    drifted = bool(added_cols or removed_cols)

    # Save fresh snapshot after detecting drift
    _save_snapshot(table_name, live_schema)

    return {
        "table_name": table_name,
        "drifted": drifted,
        "added_cols": added_cols,
        "removed_cols": removed_cols,
        "renamed_cols": renamed_cols,
        "auto_mapping": mapping,
        "live_columns": list(live_cols.keys()),
        "checked_at": datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z'),
    }


def _replace_columns_in_json(data, mapping):
    if isinstance(data, dict):
        updated = {}
        for key, value in data.items():
            new_key = mapping.get(key, key)
            updated[new_key] = _replace_columns_in_json(value, mapping)
        return updated
    if isinstance(data, list):
        return [_replace_columns_in_json(item, mapping) for item in data]
    if isinstance(data, str):
        for old_col, new_col in mapping.items():
            if data == old_col:
                return new_col
        return data
    return data


def heal_assets(drift_report: dict) -> dict:
    """
    Auto-patch saved analytical assets that reference any drifted columns.
    Updates content_definition JSON blobs to use the new column names.
    """
    if not drift_report["drifted"] or not drift_report["auto_mapping"]:
        return {"patched": 0, "assets": []}

    mapping = drift_report["auto_mapping"]
    dashboards_dir = DASHBOARDS_DIR
    patched = []

    if not os.path.exists(dashboards_dir):
        return {"patched": 0, "assets": []}

    for fname in os.listdir(dashboards_dir):
        if not fname.endswith(".json"):
            continue
        fpath = os.path.join(dashboards_dir, fname)
        with open(fpath) as f:
            try:
                content = json.load(f)
            except json.JSONDecodeError:
                continue

        updated = _replace_columns_in_json(content, mapping)
        if updated != content:
            with open(fpath, "w") as f:
                json.dump(updated, f, indent=2)
            patched.append(fname)

    return {"patched": len(patched), "assets": patched}


def snapshot_all_tables() -> dict:
    """Snapshot every known table — run on startup to initialise the baseline."""
    from kaggle_connector import list_kaggle_tables
    tables  = list_kaggle_tables()
    results = {}
    for t in tables:
        try:
            schema = get_table_schema(t)
            _save_snapshot(t, schema)
            results[t] = "ok"
        except Exception as e:
            results[t] = str(e)
    return results
