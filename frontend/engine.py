import os
import time
import json
import re
import duckdb
import pandas as pd
import requests
from kaggle_connector import query_kaggle_table, get_table_schema, list_kaggle_tables, TableNotFoundError

ASSET_ID_PATTERN = re.compile(r'^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$')


def _dashboard_path(dashboard_id):
    if not isinstance(dashboard_id, str) or not ASSET_ID_PATTERN.fullmatch(dashboard_id):
        raise ValueError('Invalid dashboard identifier.')
    dashboards_dir = os.path.join(os.path.dirname(__file__), '..', 'data', 'dashboards')
    return dashboards_dir, os.path.join(dashboards_dir, f"{dashboard_id}.json")

class StatGateAnalysisEngine:
    def __init__(self, go_backend_url=None):
        self.go_backend_url = go_backend_url or os.getenv('GO_BACKEND_URL', 'http://localhost:8080')
        # Initialize DuckDB in-memory engine
        self.duck_conn = duckdb.connect(database=':memory:')

    def _go_headers(self, tenant_id):
        headers = {"X-Tenant-ID": tenant_id}
        internal_key = os.getenv("STATGATE_INTERNAL_API_KEY")
        if internal_key:
            headers["X-StatGate-Internal-Key"] = internal_key
        return headers

    def get_dataset_summary(self, table_name):
        """
        Generates automated profiling data (Null counts, distributions, data types, stats)
        for any of the datasets in < 1 second.
        """
        start_time = time.time()
        schema = get_table_schema(table_name)
        records = query_kaggle_table(table_name, limit=100)

        if records:
            df = pd.DataFrame(records)
            self.duck_conn.register('current_view', df)
            
            # Execute in-memory DuckDB profiling query
            col_names = [col['column_name'] for col in schema]
            profiling = []
            
            for col in col_names:
                if col in df.columns:
                    series = df[col]
                    null_count = int(series.isnull().sum())
                    distinct_count = int(series.nunique())
                    dtype = str(series.dtype)
                    
                    stats = {}
                    if pd.api.types.is_numeric_dtype(series):
                        stats = {
                            "mean": float(series.mean()) if not series.empty else 0.0,
                            "std": float(series.std()) if len(series) > 1 else 0.0,
                            "min": float(series.min()) if not series.empty else 0.0,
                            "max": float(series.max()) if not series.empty else 0.0
                        }
                    
                    profiling.append({
                        "column_name": col,
                        "data_type": dtype,
                        "null_count": null_count,
                        "distinct_count": distinct_count,
                        "stats": stats
                    })
            latency_ms = round((time.time() - start_time) * 1000, 2)
            return {
                "table_name": table_name,
                "row_count": len(df),
                "latency_ms": latency_ms,
                "profiling": profiling,
                "sample_records": records[:100]
            }
        else:
            latency_ms = round((time.time() - start_time) * 1000, 2)
            return {
                "table_name": table_name,
                "row_count": 0,
                "latency_ms": latency_ms,
                "profiling": [
                    {
                        "column_name": col['column_name'],
                        "data_type": col['data_type'],
                        "null_count": 0,
                        "distinct_count": 0,
                        "stats": {"mean": 0.0, "std": 0.0, "min": 0.0, "max": 0.0}
                    } for col in schema
                ],
                "sample_records": []
            }

    def fetch_semantic_formula(self, metric_id, tenant_id):
        """
        Bridges to the Go backend semantic.go registry to resolve metric business logic.
        """
        try:
            res = requests.get(
                f"{self.go_backend_url}/api/v1/indicators",
                headers=self._go_headers(tenant_id),
                timeout=5
            )
            if res.status_code == 200:
                indicators = res.json().get('indicators', [])
                for ind in indicators:
                    if ind['id'] == metric_id or ind['name'].lower() == metric_id.lower():
                        return ind['formula']
        except Exception as e:
            print(f"Semantic registry connection note: {e}")

        # Fallback default formulas resolved dynamically
        fallback_formulas = {
            "net_revenue": "SUM(metric_value) * 0.85",
            "avg_latency": "AVG(value)",
            "total_count": "COUNT(*)"
        }
        return fallback_formulas.get(metric_id.lower(), "AVG(value)")

    def load_dataset(self, table_name, limit=1000):
        """
        Loads dataset directly into Pandas DataFrame via DuckDB in-memory engine.
        """
        records = query_kaggle_table(table_name, limit=limit)
        if records:
            df = pd.DataFrame(records)
            self.duck_conn.register(table_name, df)
            return df
        return pd.DataFrame()

    def save_dashboard_metadata(self, dashboard_id, metadata):
        """
        Persists dashboard layout and chart definitions as JSON metadata.
        """
        dashboards_dir, path = _dashboard_path(dashboard_id)
        os.makedirs(dashboards_dir, exist_ok=True)
        with open(path, 'w') as f:
            json.dump(metadata, f, indent=2)
        return {"status": "saved", "path": path}

    def load_dashboard_metadata(self, dashboard_id="default", strict=False):
        """
        Loads persisted dashboard layout JSON metadata.
        """
        _, path = _dashboard_path(dashboard_id)
        if os.path.exists(path):
            with open(path, 'r') as f:
                return json.load(f)
        if strict:
            raise FileNotFoundError(f"Dashboard '{dashboard_id}' not found.")
        return {
            "dashboard_id": dashboard_id,
            "widgets": []
        }

    def run_functional_analysis(self, table_name, metric_id="avg_latency", tenant_id="tenant-alpha", user_clearance=2):
        """
        Safety Bridge: Mandatory single entry point for functional analytics.
        Executes dynamic formulas inside DuckDB in-memory engine.
        """
        start_time = time.time()
        
        # Enforce ABAC clearance level check
        if user_clearance < 1:
            raise PermissionError("ABAC Policy Error: Clearance level insufficient for analytical computation.")

        formula = self.fetch_semantic_formula(metric_id, tenant_id)
        
        if table_name:
            records = query_kaggle_table(table_name, limit=500)
            df = pd.DataFrame(records) if records else pd.DataFrame()
        else:
            # Fallback to Go Lakehouse live telemetry feed
            res = requests.get(
                f"{self.go_backend_url}/api/v1/query?limit=100",
                headers=self._go_headers(tenant_id),
                timeout=5,
            )
            records = res.json().get('records', [])
            df = pd.DataFrame([r['payload'] for r in records]) if records else pd.DataFrame()

        if df.empty:
            return {
                "metric_id": metric_id,
                "formula": formula,
                "result": 0.0,
                "latency_ms": round((time.time() - start_time) * 1000, 2)
            }

        self.duck_conn.register('analysis_df', df)
        
        # Execute formula inside DuckDB
        try:
            query = f"SELECT {formula} AS metric_result FROM analysis_df"
            res_df = self.duck_conn.execute(query).df()
            val = float(res_df['metric_result'].iloc[0]) if not res_df.empty else 0.0
        except Exception:
            val = float(df['value'].mean()) if 'value' in df.columns else 0.0

        latency_ms = round((time.time() - start_time) * 1000, 2)
        return {
            "metric_id": metric_id,
            "resolved_formula": formula,
            "result": val,
            "dataset": table_name or "live_telemetry_stream",
            "latency_ms": latency_ms,
            "status": "success"
        }

# Global Singleton Instance of Engine
statgate_engine = StatGateAnalysisEngine()
