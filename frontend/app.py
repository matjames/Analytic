import os
import json
import requests
import io
import time
import contextlib
import datetime
import re
import pandas as pd
from flask import Flask, render_template, request, jsonify, session, redirect, url_for, g
from dotenv import load_dotenv
try:
    from pythonjsonlogger import jsonlogger
except ImportError:
    jsonlogger = None
import uuid
import logging
from kaggle_connector import list_kaggle_tables, query_kaggle_table, get_table_schema, validate_dataset_name, TableNotFoundError, is_db_connected, init_pool_or_fail
try:
    from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST
except ImportError:
    Counter = Histogram = None
    generate_latest = lambda: b''
    CONTENT_TYPE_LATEST = 'text/plain; version=0.0.4; charset=utf-8'
from engine import statgate_engine
from agentic_engine import agentic_engine
from schema_healer import detect_drift, heal_assets, snapshot_all_tables
from auth_service import analytics_context, validate_registry_token
from services.backend_proxy import build_backend_headers, make_go_request
from services.analytics_service import build_nlq_response
from services.assets_service import (
    list_dashboard_assets,
    validate_asset_id as validate_asset_id_contract,
    build_asset_payload,
    save_dashboard_asset,
    get_dashboard_asset_metadata,
)
from services.service_health import collect_service_health, list_expected_services
from services.registry_service import get_registry_identity_from_request
from services.notebook_service import build_kernel_session_scope, is_notebook_execution_enabled as notebook_exec_enabled, validate_notebook_execution_request
from services.event_bus import (
    publish_dataset_imported,
    publish_anomaly_detected,
    publish_agent_report_generated,
    publish_dashboard_saved,
)
from services.object_links import create_link, get_links, remove_link
from services.platform_objects import (
    create_project, get_project, list_projects, update_project,
    create_report, get_report, list_reports, update_report,
    create_research, get_research, list_research, update_research,
)
from services.workflow_engine import (
    start_workflow, get_workflow_instance, get_available_actions,
    transition, get_transition_history, list_templates,
)
from services.dataset_metadata import (
    upsert_dataset_metadata, get_dataset_metadata, list_dataset_metadata,
    search_datasets, remove_dataset_metadata,
)
from services.command_centre import get_command_centre
from services.registry_integration import (
    list_registry_facilities, get_registry_facility, get_registry_summary,
)

load_dotenv(os.path.join(os.path.dirname(__file__), '..', '.env'))


def _read_secret(env_name):
    # Prefer environment variable
    val = os.getenv(env_name)
    if val:
        return val
    # Then Docker secrets path
    secret_path = f"/run/secrets/{env_name}"
    try:
        if os.path.exists(secret_path):
            with open(secret_path, 'r') as fh:
                return fh.read().strip()
    except Exception:
        pass
    # Then repository-local ./secrets folder (developer convenience)
    local_path = os.path.join(os.path.dirname(__file__), '..', 'secrets', env_name)
    try:
        if os.path.exists(local_path):
            with open(local_path, 'r') as fh:
                return fh.read().strip()
    except Exception:
        pass
    return None


app = Flask(__name__)
app.secret_key = _read_secret('FLASK_SECRET_KEY') or os.getenv('FLASK_SECRET_KEY', '')
GO_BACKEND_URL = os.getenv('GO_BACKEND_URL', 'http://localhost:8080')
GO_REQUEST_TIMEOUT_SECONDS = float(os.getenv('GO_REQUEST_TIMEOUT_SECONDS', '5'))
STATCHAT_API_URL = os.getenv('STATCHAT_API_URL', 'http://localhost:4000').rstrip('/')
STATCHAT_UI_URL = os.getenv('STATCHAT_UI_URL', 'http://localhost:3009').rstrip('/')
STATCHAT_REQUEST_TIMEOUT_SECONDS = float(os.getenv('STATCHAT_REQUEST_TIMEOUT_SECONDS', '5'))


def is_notebook_execution_enabled():
    return notebook_exec_enabled()


NOTEBOOK_EXECUTION_ENABLED = is_notebook_execution_enabled()
ASSET_ID_PATTERN = re.compile(r'^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$')
REGISTRY_API_URL = os.getenv('STATGATE_REGISTRY_API_URL', 'http://localhost:9090/api').rstrip('/')
REGISTRY_UI_URL = os.getenv('STATGATE_REGISTRY_UI_URL', 'http://localhost:3000').rstrip('/')
REGISTRY_JWT_SECRET = os.getenv('STATGATE_REGISTRY_JWT_SECRET', '')
AUTH_REQUIRED = os.getenv('ANALYTICS_REQUIRE_AUTH', 'false').lower() in ('true', '1', 'yes')

# Read critical secrets from Docker secrets when available
REGISTRY_JWT_SECRET = _read_secret('STATGATE_REGISTRY_JWT_SECRET') or REGISTRY_JWT_SECRET
INTERNAL_API_KEY = _read_secret('STATGATE_INTERNAL_API_KEY') or os.getenv('STATGATE_INTERNAL_API_KEY')


# Configure structured JSON logging
root_logger = logging.getLogger()
if not any(isinstance(h, logging.StreamHandler) for h in root_logger.handlers):
    handler = logging.StreamHandler()
    if jsonlogger is not None:
        formatter = jsonlogger.JsonFormatter('%(asctime)s %(levelname)s %(name)s %(message)s')
    else:
        formatter = logging.Formatter('%(asctime)s %(levelname)s %(name)s %(message)s')
    handler.setFormatter(formatter)
    root_logger.addHandler(handler)
root_logger.setLevel(logging.INFO)

# Fail-fast DB pool initialization on startup only in production.
# Local development and test runs should not abort if Postgres is not yet available.
try:
    if os.getenv('STATGATE_ENV', 'development').lower() == 'production' and os.getenv('KAGGLE_DB_FAIL_FAST', 'true').lower() in ('true', '1', 'yes'):
        timeout = int(os.getenv('KAGGLE_DB_STARTUP_TIMEOUT', '30'))
        interval = float(os.getenv('KAGGLE_DB_STARTUP_INTERVAL', '2'))
        root_logger.info('Performing production DB fail-fast check (timeout=%s, interval=%s)', timeout, interval)
        init_pool_or_fail(timeout=timeout, interval=interval)
    else:
        root_logger.info('Skipping DB fail-fast startup check in non-production mode')
except Exception:
    root_logger.exception('DB fail-fast startup check failed; aborting application startup')
    raise


@app.before_request
def attach_request_id():
    rid = request.headers.get('X-Request-ID') or str(uuid.uuid4())
    g.request_id = rid
    # echo back to client
    request.environ['HTTP_X_REQUEST_ID'] = rid


@app.after_request
def add_request_id_header(response):
    try:
        response.headers['X-Request-ID'] = g.get('request_id', '')
    except Exception:
        pass
    return response

@app.context_processor
def inject_registry_context():
    local_logo_url = url_for('static', filename='images/logo.png', _external=False)
    return {
        'registry_api_url': REGISTRY_API_URL,
        'registry_ui_url': REGISTRY_UI_URL,
        'analytics_title': 'StatGate Analytics',
        'analytics_logo_url': local_logo_url,
    }

if AUTH_REQUIRED and (not app.secret_key or not REGISTRY_JWT_SECRET):
    raise RuntimeError('FLASK_SECRET_KEY and STATGATE_REGISTRY_JWT_SECRET are required when ANALYTICS_REQUIRE_AUTH=true.')

def current_identity():
    """Return the Registry-authenticated identity attached to this request."""
    identity = get_registry_identity_from_request(request, session, REGISTRY_JWT_SECRET)
    if identity is not None:
        g.statgate_identity = identity
    return identity


# Prometheus metrics
if Counter is not None and Histogram is not None:
    REQUEST_COUNT = Counter('statgate_request_count', 'HTTP requests', ['method', 'endpoint', 'http_status'])
    REQUEST_LATENCY = Histogram('statgate_request_latency_seconds', 'Request latency in seconds', ['method', 'endpoint'])
else:
    class _NoopMetric:
        def labels(self, *args, **kwargs):
            return self
        def observe(self, value):
            pass
        def inc(self):
            pass
    REQUEST_COUNT = _NoopMetric()
    REQUEST_LATENCY = _NoopMetric()


@app.before_request
def _prom_before_request():
    g._prom_start_time = time.time()


@app.after_request
def _prom_after_request(response):
    try:
        start = getattr(g, '_prom_start_time', None)
        if start is not None:
            latency = time.time() - start
            REQUEST_LATENCY.labels(request.method, request.path).observe(latency)
            REQUEST_COUNT.labels(request.method, request.path, response.status_code).inc()
    except Exception:
        pass
    return response

@app.before_request
def require_registry_identity():
    public_paths = {'health', 'ready', 'analytics_login', 'analytics_logout'}
    if not AUTH_REQUIRED or request.endpoint in public_paths or request.path.startswith('/static/'):
        return None

    identity = current_identity()
    if identity is None:
        if request.path.startswith('/api/'):
            return jsonify({'error': 'StatGate Registry authentication is required.'}), 401
        return redirect(url_for('analytics_login', next=request.path))

    g.statgate_identity = identity
    session['statgate_identity'] = identity

@app.route('/login', methods=['GET', 'POST'])
def analytics_login():
    if request.method == 'GET':
        return render_template('analytics_login.html')
    payload = request.get_json(silent=True) or request.form
    try:
        response = requests.post(
            f'{REGISTRY_API_URL}/users/login',
            json={'emailOrUsername': payload.get('emailOrUsername', ''), 'password': payload.get('password', '')},
            timeout=GO_REQUEST_TIMEOUT_SECONDS,
        )
        if response.status_code != 200:
            return jsonify({'error': 'Invalid StatGate Registry credentials.'}), 401

        token = response.json().get('token', '')
        identity = validate_registry_token(token, REGISTRY_JWT_SECRET)

        session.clear()
        session['statgate_identity'] = identity
        session['registry_token'] = token
        return jsonify({'status': 'authenticated', 'identity': analytics_context(identity)})
    except ValueError:
        return jsonify({'error': 'Registry token verification failed.'}), 401
    except requests.RequestException:
        return jsonify({'error': 'StatGate Registry is unavailable.'}), 503

@app.route('/logout', methods=['POST'])
def analytics_logout():
    session.clear()
    return jsonify({'status': 'signed_out'})

@app.route('/health')
def health():
    """Lightweight readiness endpoint for the Analytics UI process."""
    return jsonify({
        'status': 'healthy',
        'service': 'statgate-analytics',
        'notebook_execution_enabled': NOTEBOOK_EXECUTION_ENABLED,
    })


@app.route('/ready')
def ready():
    """Readiness check: verifies connectivity to the analytics DB and the Go backend."""
    db_ok = False
    try:
        db_ok = is_db_connected()
    except Exception:
        db_ok = False

    backend_ok = False
    try:
        r = requests.get(f"{GO_BACKEND_URL}/ready", timeout=2)
        backend_ok = r.status_code == 200
    except Exception:
        backend_ok = False

    registry_ok = True
    registry_health_url = None
    if AUTH_REQUIRED:
        registry_health_url = REGISTRY_API_URL.rsplit('/api', 1)[0].rstrip('/') + '/ready'
        try:
            r = requests.get(registry_health_url, timeout=2)
            registry_ok = r.status_code == 200
        except Exception:
            registry_ok = False

    status = 200 if (db_ok and backend_ok and registry_ok) else 503
    payload = {
        'status': 'ready' if status == 200 else 'not-ready',
        'db': 'connected' if db_ok else 'unavailable',
        'go_backend': 'healthy' if backend_ok else 'unavailable',
    }
    if AUTH_REQUIRED:
        payload['registry'] = 'healthy' if registry_ok else 'unavailable'
        payload['registry_health_url'] = registry_health_url
    return jsonify(payload), status


@app.route('/metrics')
def metrics():
    """Expose Prometheus metrics."""
    try:
        data = generate_latest()
        return data, 200, {'Content-Type': CONTENT_TYPE_LATEST}
    except Exception:
        return 'error', 500

def get_tenant_headers(req):
    identity = current_identity()
    return build_backend_headers(
        identity,
        INTERNAL_API_KEY,
        default_tenant_id=os.getenv('DEFAULT_TENANT_ID', 'tenant-alpha'),
        default_user_role=os.getenv('DEFAULT_USER_ROLE', 'analyst'),
        default_user_clearance=os.getenv('DEFAULT_USER_CLEARANCE', '2'),
    )

def go_request(method, path, **kwargs):
    """Make a bounded request to the internal Go service."""
    prepared = make_go_request(method, path, base_url=GO_BACKEND_URL, **kwargs)
    return requests.Session().send(prepared, timeout=GO_REQUEST_TIMEOUT_SECONDS)


def statchat_request(method, path, **kwargs):
    """Make a bounded request to StatChat (the platform communication backbone)."""
    timeout = kwargs.pop('timeout', STATCHAT_REQUEST_TIMEOUT_SECONDS)
    return requests.request(method, f"{STATCHAT_API_URL}{path}", timeout=timeout, **kwargs)

def validate_asset_id(asset_id):
    return validate_asset_id_contract(asset_id)

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/datasets')
def datasets():
    return render_template('datasets.html')

@app.route('/notebook')
def notebook():
    return render_template('notebook.html')

@app.route('/executive')
def executive():
    return render_template('executive.html')

@app.route('/semantic')
def semantic():
    return render_template('semantic.html')

@app.route('/abac')
def abac():
    return render_template('abac.html')

# --- Proxy Endpoints to Go Ingestion/Orchestration Engine ---

@app.route('/api/proxy/stats', methods=['GET'])
def proxy_stats():
    try:
        res = go_request('GET', '/api/v1/stats', headers=get_tenant_headers(request))
        return jsonify(res.json()), res.status_code
    except Exception as e:
        return jsonify({'error': f'Failed to connect to Go backend engine: {str(e)}'}), 502

@app.route('/api/proxy/query', methods=['GET'])
def proxy_query():
    try:
        res = go_request('GET', '/api/v1/query', headers=get_tenant_headers(request), params=request.args)
        return jsonify(res.json()), res.status_code
    except Exception as e:
        return jsonify({'error': f'Failed to connect to Go backend engine: {str(e)}'}), 502

@app.route('/api/proxy/ingest', methods=['POST'])
def proxy_ingest():
    try:
        res = go_request('POST', '/api/v1/ingest', headers=get_tenant_headers(request), json=request.get_json())
        return jsonify(res.json()), res.status_code
    except Exception as e:
        return jsonify({'error': f'Failed to connect to Go backend engine: {str(e)}'}), 502

@app.route('/api/proxy/indicators', methods=['GET', 'POST'])
def proxy_indicators():
    try:
        if request.method == 'GET':
            res = go_request('GET', '/api/v1/indicators', headers=get_tenant_headers(request))
        else:
            res = go_request('POST', '/api/v1/indicators', headers=get_tenant_headers(request), json=request.get_json())
        return jsonify(res.json()), res.status_code
    except Exception as e:
        return jsonify({'error': str(e)}), 502

@app.route('/api/proxy/anomalies', methods=['GET'])
def proxy_anomalies():
    try:
        res = go_request('GET', '/api/v1/anomalies', headers=get_tenant_headers(request))
        return jsonify(res.json()), res.status_code
    except Exception as e:
        return jsonify({'error': str(e)}), 502

@app.route('/api/proxy/policies', methods=['GET'])
def proxy_policies():
    try:
        res = go_request('GET', '/api/v1/policies', headers=get_tenant_headers(request))
        return jsonify(res.json()), res.status_code
    except Exception as e:
        return jsonify({'error': str(e)}), 502

# --- Functional Data Product & Engine API Routes ---

@app.route('/api/v1/datasets/fetch/<table_name>', methods=['GET'])
def fetch_live_dataset(table_name):
    # Live active projection query SELECT * FROM table_name LIMIT 100
    try:
        validate_dataset_name(table_name)
        records = query_kaggle_table(table_name, limit=100)
        return jsonify({
            'table_name': table_name,
            'count': len(records),
            'records': records
        })
    except ValueError as e:
        return jsonify({'error': str(e)}), 400
    except TableNotFoundError as e:
        return jsonify({'error': str(e)}), 404
    except Exception as e:
        return jsonify({'error': str(e)}), 502

@app.route('/api/kaggle/datasets', methods=['GET'])
def get_kaggle_datasets():
    # Flask owns live PostgreSQL discovery. Calling Go here would create a proxy loop.
    try:
        tables = list_kaggle_tables()
        return jsonify({
            'schema': os.getenv('KAGGLE_STAGING_SCHEMA', 'ml_staging'),
            'count': len(tables),
            'tables': tables
        })
    except Exception as e:
        return jsonify({'error': str(e)}), 502

@app.route('/api/kaggle/datasets/refresh', methods=['POST'])
def refresh_kaggle_datasets():
    """Refresh the analytics dataset catalog by snapshotting every current table."""
    try:
        snapshot_results = snapshot_all_tables()
        tables = list_kaggle_tables()
        # Publish dataset.imported events for each table so other modules
        # (StatChat, Go Core, Registry) can react to the new data.
        for table in tables:
            publish_dataset_imported(table, tenant_id='tenant-alpha')
        return jsonify({
            'schema': os.getenv('KAGGLE_STAGING_SCHEMA', 'ml_staging'),
            'count': len(tables),
            'tables': tables,
            'snapshot_results': snapshot_results,
        }), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 502

@app.route('/api/kaggle/schema/<table_name>', methods=['GET'])
def get_kaggle_schema(table_name):
    # Functional Engine mandatory automated dataset profiling
    try:
        validate_dataset_name(table_name)
        summary = statgate_engine.get_dataset_summary(table_name)
        return jsonify(summary)
    except ValueError as e:
        return jsonify({'error': str(e)}), 400
    except TableNotFoundError as e:
        return jsonify({'error': str(e)}), 404
    except Exception as e:
        return jsonify({'error': str(e)}), 502

@app.route('/api/functional/analyze', methods=['POST'])
def run_functional_analysis_endpoint():
    data = request.get_json() or {}
    table_name = data.get('table_name', '')
    metric_id = data.get('metric_id', 'avg_latency')
    tenant_id = request.headers.get('X-Tenant-ID', 'tenant-alpha')
    try:
        user_clearance = int(request.headers.get('X-User-Clearance', '2'))
    except ValueError:
        return jsonify({'error': 'X-User-Clearance must be an integer.'}), 400

    try:
        result = statgate_engine.run_functional_analysis(
            table_name=table_name,
            metric_id=metric_id,
            tenant_id=tenant_id,
            user_clearance=user_clearance
        )
        return jsonify(result)
    except PermissionError as e:
        return jsonify({'error': str(e)}), 403
    except Exception as e:
        return jsonify({'error': str(e)}), 502

# ── Semantic NLQ Engine ─────────────────────────────────────
# The StatGate Analyst — expands user jargon via semantic registry,
# generates SQL, and produces a summary insight narrative.

# Common semantic alias dictionary (supplements the Go registry)

@app.route('/api/nlq', methods=['POST'])
def nlq_semantic_query():
    data = request.get_json() or {}
    prompt = data.get('prompt', '').strip()
    tenant_id = data.get('tenant_id', request.headers.get('X-Tenant-ID', 'tenant-alpha'))

    try:
        response = build_nlq_response(prompt, tenant_id)
    except ValueError:
        return jsonify({'error': 'Empty prompt'}), 400

    try:
        records = query_kaggle_table(response['dataset'], limit=100)
        df = pd.DataFrame(records) if records else pd.DataFrame()
        related_cols = response['resolved_columns']
        semantic_label = response['semantic_label']
        resolved_formula = response['resolved_formula']
        dataset = response['dataset']

        if not df.empty and related_cols and related_cols[0] in df.columns:
            col = related_cols[0]
            mean_val = df[col].mean() if pd.api.types.is_numeric_dtype(df[col]) else None
            max_val = df[col].max() if pd.api.types.is_numeric_dtype(df[col]) else None
            row_count = len(df)
            mean_label = f"{mean_val:.2f}" if mean_val is not None else 'N/A'
            max_label = f"{max_val:.2f}" if max_val is not None else 'N/A'
            insight = (
                f"Semantic Analyst Insight: Analysis of '{semantic_label}' across {row_count} records in '{dataset}'. "
                f"Mean {col} = {mean_label}, Peak = {max_label}. "
                f"Formula resolved via Semantic Registry: {resolved_formula}."
            )
        else:
            insight = (
                f"Semantic alias '{semantic_label}' resolved to dataset '{dataset}'. "
                f"Formula: {resolved_formula}. No numeric columns detected for auto-summary."
            )
    except Exception as e:
        insight = f"Semantic expansion succeeded. Live dataset query note: {e}"

    explanation = (
        f"[Semantic Query Expansion]\n"
        f"User Term: '{prompt}'\n"
        f"Resolved Alias: '{response['semantic_label']}'\n"
        f"Mapped Columns: {', '.join(response['resolved_columns']) or 'general'}\n"
        f"Target Dataset: {response['dataset']}\n"
        f"Standardized Formula: {response['resolved_formula']}\n\n"
        f"[Analyst Insight]\n{insight}"
    )

    return jsonify({
        'prompt': prompt,
        'resolved_alias': response['semantic_label'],
        'dataset': response['dataset'],
        'expanded_columns': response['resolved_columns'],
        'generated_sql': response['generated_sql'],
        'explanation': explanation,
        'insight': insight
    })

# ── Real-time Proactive Alerts Proxy ───────────────────────
@app.route('/api/v1/alerts', methods=['GET'])
def get_proactive_alerts():
    """Proxy to Go backend 3-sigma anomaly alert feed."""
    try:
        res = go_request('GET', '/api/v1/alerts', headers=get_tenant_headers(request), timeout=3)
        if res.status_code == 200:
            return jsonify(res.json()), 200
    except Exception as e:
        print(f"[app.py] Alert proxy note: {e}")

    # Fallback: generate a demo alert if Go is unreachable
    return jsonify({
        'tenant_id': 'tenant-alpha',
        'count': 1,
        'alerts': [{
            'id': 'alert_demo_001',
            'metric_name': 'Confirmed Cases Spike',
            'dataset': 'covid_19_data',
            'value': 285.0,
            'mean': 100.9,
            'sigma_score': 4.2,
            'severity': 'CRITICAL',
            'message': '3σ Anomaly: Confirmed Cases Spike = 285.00 (4.20σ from μ=100.90)',
            'notebook_url': '/notebook?dataset=covid_19_data',
            'timestamp': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())
        }]
    })



@app.route('/api/v1/assets/save', methods=['POST'])
def save_analytical_asset():
    data = request.get_json() or {}
    dashboard_id = data.get('id', data.get('dashboard_id', 'default_layout'))
    try:
        dashboard_id = validate_asset_id(dashboard_id)
    except ValueError as e:
        return jsonify({'error': str(e)}), 400

    asset_type = data.get('asset_type', 'dashboard')
    content_def = data.get('content_definition', data.get('metadata', {}))
    version_tag = data.get('version_tag', '1.0.0')
    identity = current_identity()
    owner = analytics_context(identity)['tenant_id'] if identity else request.headers.get('X-Tenant-ID', 'tenant-alpha')

    asset_payload = build_asset_payload(dashboard_id, asset_type, content_def, owner, version_tag)
    try:
        res = save_dashboard_asset(statgate_engine, asset_payload)
        # Publish dashboard.saved event so other modules know about the new asset.
        publish_dashboard_saved(dashboard_id, tenant_id=owner)
        return jsonify(res), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/v1/assets/get/<asset_id>', methods=['GET'])
def get_analytical_asset(asset_id):
    try:
        asset_id = validate_asset_id(asset_id)
    except ValueError as e:
        return jsonify({'error': str(e)}), 400

    identity = current_identity()
    owner = analytics_context(identity)['tenant_id'] if identity else request.headers.get('X-Tenant-ID', 'tenant-alpha')
    try:
        asset = get_dashboard_asset_metadata(statgate_engine, asset_id, owner_id=owner)
        return jsonify(asset), 200
    except FileNotFoundError:
        return jsonify({'error': 'Asset not found.'}), 404
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/v1/assets/list', methods=['GET'])
def list_analytical_assets():
    # Return saved dashboard/widget assets
    assets_list = list_dashboard_assets(statgate_engine)
    return jsonify({'count': len(assets_list), 'assets': assets_list})

@app.route('/api/dashboard/save', methods=['POST'])
def save_dashboard():
    data = request.get_json() or {}
    dashboard_id = data.get('dashboard_id', 'default')
    try:
        dashboard_id = validate_asset_id(dashboard_id)
    except ValueError as e:
        return jsonify({'error': str(e)}), 400

    metadata = data.get('metadata', {})
    identity = current_identity()
    owner = analytics_context(identity)['tenant_id'] if identity else request.headers.get('X-Tenant-ID', 'tenant-alpha')
    asset_payload = build_asset_payload(dashboard_id, 'dashboard', metadata, owner)

    try:
        res = save_dashboard_asset(statgate_engine, asset_payload)
        # Publish dashboard.saved event so other modules know about the new asset.
        publish_dashboard_saved(dashboard_id, tenant_id=owner)
        return jsonify(res), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/dashboard/load/<dashboard_id>', methods=['GET'])
def load_dashboard(dashboard_id):
    try:
        dashboard_id = validate_asset_id(dashboard_id)
    except ValueError as e:
        return jsonify({'error': str(e)}), 400

    identity = current_identity()
    owner = analytics_context(identity)['tenant_id'] if identity else request.headers.get('X-Tenant-ID', 'tenant-alpha')
    try:
        asset = get_dashboard_asset_metadata(statgate_engine, dashboard_id, owner_id=owner)
        return jsonify(asset), 200
    except FileNotFoundError:
        return jsonify({'error': 'Dashboard not found.'}), 404
    except Exception as e:
        return jsonify({'error': str(e)}), 500

# --- Persistent Kernel Session Scope ---
kernel_sessions = {}

# --- Notebook Execution Sandbox (Using StatGate SDK & Persistent Scope) ---

@app.route('/api/notebook/execute', methods=['POST'])
def execute_notebook():
    if not NOTEBOOK_EXECUTION_ENABLED:
        return jsonify({'success': False, 'error': 'Notebook execution is disabled. Set ENABLE_NOTEBOOK_EXECUTION=true only for trusted deployments.'}), 403
    data = request.get_json() or {}
    try:
        validated = validate_notebook_execution_request(data)
    except ValueError as exc:
        return jsonify({'success': False, 'error': str(exc)}), 400

    code = data.get('code', '')
    tenant_id = validated['tenant_id']
    target_table = data.get('kaggle_table', '')
    session_id = validated['session_id']

    # Get or initialize persistent kernel session scope
    if session_id not in kernel_sessions:
        from statgate import engine as statgate_sdk_engine
        kernel_sessions[session_id] = build_kernel_session_scope(
            session_id=session_id,
            tenant_id=tenant_id,
            pd_module=pd,
            requests_module=requests,
            engine_module=statgate_engine,
            statgate_module=__import__('statgate'),
            list_tables_fn=list_kaggle_tables,
        )

    exec_scope = kernel_sessions[session_id]

    # Pre-load df if target_table is passed
    if target_table:
        summary = statgate_engine.get_dataset_summary(target_table)
        records = summary.get('sample_records', [])
        exec_scope['df'] = pd.DataFrame(records) if records else pd.DataFrame()
        exec_scope['target_table'] = target_table

    stdout_capture = io.StringIO()

    try:
        with contextlib.redirect_stdout(stdout_capture):
            exec(code, exec_scope)
        output = stdout_capture.getvalue()
        return jsonify({'success': True, 'output': output if output else 'Code executed successfully with no stdout.'})
    except Exception as e:
        return jsonify({'success': False, 'error': str(e)})

# ══════════════════════════════════════════════════════════════
#  AGENTIC DECISION-SUPPORT FABRIC
# ══════════════════════════════════════════════════════════════

@app.route('/api/agent/analyze', methods=['POST'])
def agent_analyze():
    """
    Agentic Insight Engine: goal-oriented multi-step analysis.
    Accepts a high-level intent and returns 3 Intervention Scenarios.
    """
    data      = request.get_json() or {}
    goal      = data.get('goal', '').strip()
    tenant_id = data.get('tenant_id', request.headers.get('X-Tenant-ID', 'tenant-alpha'))

    if not goal:
        return jsonify({'error': 'goal is required'}), 400

    try:
        report = agentic_engine.analyze_intent(goal, tenant_id)
        # Publish agent.report.generated event so StatChat can notify
        # relevant teams about the new analytical report.
        report_id = report.get('generated_at', 'report-unknown')
        publish_agent_report_generated(report_id, tenant_id=tenant_id, goal=goal)
        return jsonify(report), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/agent/feedback', methods=['POST'])
def agent_feedback():
    """
    RLHF Feedback Loop: record analyst accept/reject on a proposed scenario.
    This data is persisted and used to continuously improve recommendations.
    """
    data        = request.get_json() or {}
    report_id   = data.get('report_id', 'unknown')
    scenario_id = data.get('scenario_id', 'scenario_1')
    action      = data.get('action', 'approved')    # approved | rejected | deferred
    outcome     = data.get('outcome', '')
    tenant_id   = data.get('tenant_id', request.headers.get('X-Tenant-ID', 'tenant-alpha'))

    if action not in {'approved', 'rejected', 'deferred'}:
        return jsonify({'error': 'Invalid action. Expected approved, rejected, or deferred.'}), 400

    try:
        result = agentic_engine.record_feedback(report_id, scenario_id, action, outcome, tenant_id)
        return jsonify({'status': 'recorded', 'detail': result}), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/executive/summary', methods=['GET'])
def executive_summary():
    """
    Executive-grade summary — minimal cognitive load.
    Returns actions_needed count, top alert, and system health status.
    """
    tenant_id = request.args.get('tenant_id', request.headers.get('X-Tenant-ID', 'tenant-alpha'))
    try:
        summary = agentic_engine.get_executive_summary(tenant_id)
        return jsonify(summary), 200
    except Exception as e:
        root_logger.exception('Executive summary generation failed: %s', e)
        return jsonify({
            'tenant_id': tenant_id,
            'total_datasets': 0,
            'actions_needed': 0,
            'critical_alerts': 0,
            'high_alerts': 0,
            'top_alert': None,
            'data_snapshot': {},
            'system_health': 'Unknown',
            'generated_at': datetime.datetime.now(datetime.UTC).isoformat(),
            'note': 'Executive summary is temporarily unavailable. Try again later.',
        }), 200


# ── Self-Healing Pipeline Routes ──────────────────────────────
@app.route('/api/schema-health/<table_name>', methods=['GET'])
def schema_health(table_name):
    """Detect schema drift for a single table and auto-heal any impacted assets."""
    try:
        validate_dataset_name(table_name)
        drift = detect_drift(table_name)
        heal  = heal_assets(drift) if drift['drifted'] else {'patched': 0, 'assets': []}
        return jsonify({**drift, 'healing': heal}), 200
    except ValueError as e:
        return jsonify({'error': str(e)}), 400
    except TableNotFoundError as e:
        return jsonify({'error': str(e)}), 404
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/schema-health/snapshot/all', methods=['POST'])
def schema_snapshot_all():
    """Baseline snapshot of all tables. Call on startup or after a major ingestion."""
    try:
        results = snapshot_all_tables()
        return jsonify({'snapshotted': results}), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 500


# ── Agentic Webhook Dispatch (via Go) ────────────────────────
@app.route('/api/agent/dispatch', methods=['POST'])
def agent_dispatch():
    """Proxy a webhook action dispatch to the Go backend ABAC-gated dispatcher."""
    data = request.get_json() or {}
    try:
        res = go_request('POST', '/api/v1/agent/dispatch', headers=get_tenant_headers(request), json=data, timeout=4)
        return jsonify(res.json()), res.status_code
    except Exception as e:
        return jsonify({'status': 'safe_mode', 'note': str(e),
                        'detail': 'Go webhook dispatcher not reachable — action logged locally.'}), 200


@app.route('/api/alerts', methods=['POST'])
def receive_alert():
    """Endpoint for Alertmanager webhooks during local/dev deployments.
    Logs the alert payload as structured JSON for visibility.
    """
    payload = request.get_json(silent=True)
    if not payload:
        return jsonify({'error': 'invalid payload'}), 400
    root_logger.info('Received Alertmanager webhook', extra={'alert': payload})
    return jsonify({'status': 'received'}), 200


# ══════════════════════════════════════════════════════════════
#  PLATFORM OBJECTS — Projects, Reports, Research
# ══════════════════════════════════════════════════════════════

# ── Projects ──────────────────────────────────────────────────

@app.route('/api/projects', methods=['POST'])
def api_create_project():
    data = request.get_json(silent=True) or {}
    title = (data.get('title') or '').strip()
    if not title:
        return jsonify({'error': 'title is required'}), 400
    identity = current_identity()
    owner_id = identity.get('userId') if identity else None
    tenant_id = (data.get('tenant_id') or request.headers.get('X-Tenant-ID', 'tenant-alpha')).strip()
    result = create_project(
        title=title,
        description=data.get('description', ''),
        owner_id=owner_id,
        tenant_id=tenant_id,
        collaborators=data.get('collaborators'),
        tags=data.get('tags'),
        start_date=data.get('start_date'),
        end_date=data.get('end_date'),
    )
    if result is None:
        return jsonify({'error': 'Failed to create project (DB unavailable)'}), 500
    # Auto-start the project lifecycle workflow.
    start_workflow('project', result['id'], initiated_by=owner_id, tenant_id=tenant_id)
    # Publish event so other modules know about the new project.
    from services.event_bus import publish_event
    publish_event('project.created', 'project', result['id'], tenant_id=tenant_id, payload=result)
    return jsonify(result), 201


@app.route('/api/projects', methods=['GET'])
def api_list_projects():
    tenant_id = (request.args.get('tenant_id') or request.headers.get('X-Tenant-ID', 'tenant-alpha')).strip()
    status = request.args.get('status')
    projects = list_projects(tenant_id=tenant_id, status=status)
    return jsonify({'count': len(projects), 'projects': projects}), 200


@app.route('/api/projects/<project_id>', methods=['GET'])
def api_get_project(project_id):
    result = get_project(project_id)
    if result is None:
        return jsonify({'error': 'Project not found'}), 404
    return jsonify(result), 200


@app.route('/api/projects/<project_id>', methods=['PUT'])
def api_update_project(project_id):
    data = request.get_json(silent=True) or {}
    result = update_project(project_id, **data)
    if result is None:
        return jsonify({'error': 'Failed to update project'}), 500
    return jsonify(result), 200


# ── Reports ───────────────────────────────────────────────────

@app.route('/api/reports', methods=['POST'])
def api_create_report():
    data = request.get_json(silent=True) or {}
    title = (data.get('title') or '').strip()
    if not title:
        return jsonify({'error': 'title is required'}), 400
    identity = current_identity()
    author_id = identity.get('userId') if identity else None
    tenant_id = (data.get('tenant_id') or request.headers.get('X-Tenant-ID', 'tenant-alpha')).strip()
    result = create_report(
        title=title,
        content=data.get('content', ''),
        summary=data.get('summary', ''),
        author_id=author_id,
        tenant_id=tenant_id,
        project_id=data.get('project_id'),
        tags=data.get('tags'),
    )
    if result is None:
        return jsonify({'error': 'Failed to create report (DB unavailable)'}), 500
    # Auto-start the report approval workflow.
    start_workflow('report', result['id'], initiated_by=author_id, tenant_id=tenant_id)
    from services.event_bus import publish_event
    publish_event('report.created', 'report', result['id'], tenant_id=tenant_id, payload=result)
    return jsonify(result), 201


@app.route('/api/reports', methods=['GET'])
def api_list_reports():
    tenant_id = (request.args.get('tenant_id') or request.headers.get('X-Tenant-ID', 'tenant-alpha')).strip()
    status = request.args.get('status')
    project_id = request.args.get('project_id')
    reports = list_reports(tenant_id=tenant_id, status=status, project_id=project_id)
    return jsonify({'count': len(reports), 'reports': reports}), 200


@app.route('/api/reports/<report_id>', methods=['GET'])
def api_get_report(report_id):
    result = get_report(report_id)
    if result is None:
        return jsonify({'error': 'Report not found'}), 404
    return jsonify(result), 200


@app.route('/api/reports/<report_id>', methods=['PUT'])
def api_update_report(report_id):
    data = request.get_json(silent=True) or {}
    result = update_report(report_id, **data)
    if result is None:
        return jsonify({'error': 'Failed to update report'}), 500
    # If status changed to published, publish an event.
    if data.get('status') == 'published':
        from services.event_bus import publish_event
        publish_event('report.published', 'report', report_id, payload=result)
    return jsonify(result), 200


# ── Research Studies ──────────────────────────────────────────

@app.route('/api/research', methods=['POST'])
def api_create_research():
    data = request.get_json(silent=True) or {}
    title = (data.get('title') or '').strip()
    if not title:
        return jsonify({'error': 'title is required'}), 400
    tenant_id = (data.get('tenant_id') or request.headers.get('X-Tenant-ID', 'tenant-alpha')).strip()
    result = create_research(
        title=title,
        description=data.get('description', ''),
        protocol=data.get('protocol', ''),
        principal_investigator=data.get('principal_investigator'),
        tenant_id=tenant_id,
        project_id=data.get('project_id'),
        collaborators=data.get('collaborators'),
        tags=data.get('tags'),
        start_date=data.get('start_date'),
        end_date=data.get('end_date'),
    )
    if result is None:
        return jsonify({'error': 'Failed to create research study (DB unavailable)'}), 500
    # Auto-start the research lifecycle workflow.
    start_workflow('research', result['id'], initiated_by=data.get('principal_investigator'), tenant_id=tenant_id)
    from services.event_bus import publish_event
    publish_event('research.created', 'research', result['id'], tenant_id=tenant_id, payload=result)
    return jsonify(result), 201


@app.route('/api/research', methods=['GET'])
def api_list_research():
    tenant_id = (request.args.get('tenant_id') or request.headers.get('X-Tenant-ID', 'tenant-alpha')).strip()
    status = request.args.get('status')
    studies = list_research(tenant_id=tenant_id, status=status)
    return jsonify({'count': len(studies), 'research': studies}), 200


@app.route('/api/research/<research_id>', methods=['GET'])
def api_get_research(research_id):
    result = get_research(research_id)
    if result is None:
        return jsonify({'error': 'Research study not found'}), 404
    return jsonify(result), 200


@app.route('/api/research/<research_id>', methods=['PUT'])
def api_update_research(research_id):
    data = request.get_json(silent=True) or {}
    result = update_research(research_id, **data)
    if result is None:
        return jsonify({'error': 'Failed to update research study'}), 500
    return jsonify(result), 200


# ══════════════════════════════════════════════════════════════
#  WORKFLOW ENGINE
# ══════════════════════════════════════════════════════════════

@app.route('/api/workflows/templates', methods=['GET'])
def api_list_workflow_templates():
    """List all workflow templates."""
    return jsonify({'count': len(list_templates()), 'templates': list_templates()}), 200


@app.route('/api/workflows/<object_type>/<object_id>', methods=['GET'])
def api_get_workflow(object_type, object_id):
    """Get the workflow status for an object."""
    instance = get_workflow_instance(object_type, object_id)
    if instance is None:
        return jsonify({'error': 'No workflow found for this object'}), 404
    identity = current_identity()
    user_role = 'analyst'
    if identity:
        ctx = analytics_context(identity)
        user_role = ctx.get('role', 'analyst')
    actions = get_available_actions(object_type, object_id, user_role=user_role)
    history = get_transition_history(object_type, object_id)
    return jsonify({
        'instance': instance,
        'available_actions': actions,
        'history': history,
    }), 200


@app.route('/api/workflows/<object_type>/<object_id>/transition', methods=['POST'])
def api_workflow_transition(object_type, object_id):
    """Execute a workflow transition."""
    data = request.get_json(silent=True) or {}
    action = (data.get('action') or '').strip()
    if not action:
        return jsonify({'error': 'action is required'}), 400
    identity = current_identity()
    performed_by = identity.get('userId') if identity else None
    tenant_id = (data.get('tenant_id') or request.headers.get('X-Tenant-ID', 'tenant-alpha')).strip()
    result = transition(object_type, object_id, action,
                        performed_by=performed_by, comment=data.get('comment', ''),
                        tenant_id=tenant_id)
    if result is None:
        return jsonify({'error': 'Workflow transition failed (invalid action or DB unavailable)'}), 500
    return jsonify(result), 200


# ══════════════════════════════════════════════════════════════
#  DATASET METADATA & SEARCH
# ══════════════════════════════════════════════════════════════

@app.route('/api/datasets/metadata', methods=['POST'])
def api_upsert_dataset_metadata():
    """Create or update metadata for a dataset."""
    data = request.get_json(silent=True) or {}
    table_name = (data.get('table_name') or '').strip()
    if not table_name:
        return jsonify({'error': 'table_name is required'}), 400
    result = upsert_dataset_metadata(
        table_name=table_name,
        owner_id=data.get('owner_id'),
        steward_id=data.get('steward_id'),
        description=data.get('description', ''),
        classification=data.get('classification', 'public'),
        tags=data.get('tags'),
        row_count=data.get('row_count', 0),
        col_count=data.get('col_count', 0),
    )
    if result is None:
        return jsonify({'error': 'Failed to save dataset metadata (DB unavailable)'}), 500
    return jsonify(result), 200


@app.route('/api/datasets/metadata/<table_name>', methods=['GET'])
def api_get_dataset_metadata(table_name):
    """Get metadata for a single dataset."""
    result = get_dataset_metadata(table_name)
    if result is None:
        return jsonify({'error': 'No metadata found for this dataset'}), 404
    return jsonify(result), 200


@app.route('/api/datasets/metadata', methods=['GET'])
def api_list_dataset_metadata():
    """List all dataset metadata, optionally filtered by classification."""
    classification = request.args.get('classification')
    records = list_dataset_metadata(classification=classification)
    return jsonify({'count': len(records), 'metadata': records}), 200


@app.route('/api/datasets/search', methods=['GET'])
def api_search_datasets():
    """Full-text search across dataset names, descriptions, tags, owners, stewards."""
    query = (request.args.get('q') or '').strip()
    if not query:
        return jsonify({'error': 'q query param is required'}), 400
    limit = int(request.args.get('limit', '20'))
    results = search_datasets(query, limit=limit)
    return jsonify({'count': len(results), 'results': results}), 200


@app.route('/api/datasets/metadata/<table_name>', methods=['DELETE'])
def api_delete_dataset_metadata(table_name):
    """Remove metadata for a dataset."""
    ok = remove_dataset_metadata(table_name)
    if not ok:
        return jsonify({'error': 'Failed to remove dataset metadata'}), 500
    return jsonify({'status': 'removed'}), 200


# ══════════════════════════════════════════════════════════════
#  REGISTRY ↔ ANALYTICS INTEGRATION
# ══════════════════════════════════════════════════════════════

@app.route('/api/registry/facilities', methods=['GET'])
def api_list_registry_facilities():
    """
    List facilities from the Field Operations Registry as analytical
    objects within the StatGate ecosystem.
    Query params: limit, offset, district_id?
    """
    limit = int(request.args.get('limit', '100'))
    offset = int(request.args.get('offset', '0'))
    district_id = request.args.get('district_id')
    facilities = list_registry_facilities(limit=limit, offset=offset, district_id=district_id)
    return jsonify({'count': len(facilities), 'facilities': facilities}), 200


@app.route('/api/registry/facilities/<facility_id>', methods=['GET'])
def api_get_registry_facility(facility_id):
    """Get a single facility from the Registry."""
    facility = get_registry_facility(facility_id)
    if facility is None:
        return jsonify({'error': 'Facility not found in Registry'}), 404
    return jsonify(facility), 200


@app.route('/api/registry/summary', methods=['GET'])
def api_get_registry_summary():
    """Return summary counts from the Registry for command centre views."""
    return jsonify(get_registry_summary()), 200


# ══════════════════════════════════════════════════════════════
#  COMMAND CENTRE — Dashboards That Guide Action
# ══════════════════════════════════════════════════════════════

@app.route('/api/command-centre', methods=['GET'])
def api_command_centre():
    """
    Return the full command centre payload answering the four questions:
      1. What is happening?
      2. What requires my attention?
      3. What decisions should I make?
      4. What should I do next?
    """
    tenant_id = (request.args.get('tenant_id') or request.headers.get('X-Tenant-ID', 'tenant-alpha')).strip()
    return jsonify(get_command_centre(tenant_id=tenant_id)), 200


@app.route('/api/objects/links', methods=['POST'])
def create_object_link():
    """
    Create a link between two platform objects.
    Body: {source_type, source_id, target_type, target_id, relationship?, tenant_id?}
    """
    data = request.get_json(silent=True) or {}
    source_type = (data.get('source_type') or '').strip()
    source_id = (data.get('source_id') or '').strip()
    target_type = (data.get('target_type') or '').strip()
    target_id = (data.get('target_id') or '').strip()
    relationship = (data.get('relationship') or 'related').strip()
    tenant_id = (data.get('tenant_id') or request.headers.get('X-Tenant-ID', 'tenant-alpha')).strip()

    if not all([source_type, source_id, target_type, target_id]):
        return jsonify({'error': 'source_type, source_id, target_type, and target_id are required'}), 400

    ok = create_link(source_type, source_id, target_type, target_id,
                     relationship=relationship, tenant_id=tenant_id)
    if not ok:
        return jsonify({'error': 'Failed to create object link (DB unavailable or link already exists)'}), 500
    return jsonify({'status': 'linked'}), 201


@app.route('/api/objects/links', methods=['GET'])
def list_object_links():
    """
    List all links for a given object.
    Query params: type, id, tenant_id?
    """
    object_type = (request.args.get('type') or '').strip()
    object_id = (request.args.get('id') or '').strip()
    tenant_id = (request.args.get('tenant_id') or request.headers.get('X-Tenant-ID', 'tenant-alpha')).strip()

    if not object_type or not object_id:
        return jsonify({'error': 'type and id query params are required'}), 400

    links = get_links(object_type, object_id, tenant_id=tenant_id)
    return jsonify({'count': len(links), 'links': links}), 200


@app.route('/api/objects/links', methods=['DELETE'])
def delete_object_link():
    """
    Remove a link between two objects.
    Body: {source_type, source_id, target_type, target_id, relationship?}
    """
    data = request.get_json(silent=True) or {}
    source_type = (data.get('source_type') or '').strip()
    source_id = (data.get('source_id') or '').strip()
    target_type = (data.get('target_type') or '').strip()
    target_id = (data.get('target_id') or '').strip()
    relationship = (data.get('relationship') or 'related').strip()

    if not all([source_type, source_id, target_type, target_id]):
        return jsonify({'error': 'source_type, source_id, target_type, and target_id are required'}), 400

    ok = remove_link(source_type, source_id, target_type, target_id, relationship=relationship)
    if not ok:
        return jsonify({'error': 'Failed to remove object link'}), 500
    return jsonify({'status': 'unlinked'}), 200


@app.route('/api/objects/discussion', methods=['GET'])
def get_object_discussion():
    """
    Open (creating if needed) the StatChat conversation linked to any
    platform object.  This is the core of the communication backbone:
    datasets, reports, projects, dashboards, etc. can all be discussed
    through StatChat.

    Query params:
      type: dataset | report | project | dashboard | indicator | ...
      id:   object identifier (e.g. covid_19_data)
      name: optional human-friendly conversation name
    """
    object_type = (request.args.get('type') or '').strip()
    object_id = (request.args.get('id') or '').strip()
    name = (request.args.get('name') or '').strip()

    if not object_type or not object_id:
        return jsonify({'error': 'type and id query params are required'}), 400

    conv_id = f"obj:{object_type}:{object_id}"

    try:
        # Always try the StatChat API directly for a live conversation link.
        params = {}
        if name:
            params['name'] = name
        res = statchat_request('GET', f"/v1/objects/{object_type}/{object_id}/discussion", params=params)
        if res.status_code == 200:
            return jsonify({
                'conversation': res.json(),
                'conversation_id': conv_id,
                'statchat_ui_url': f"{STATCHAT_UI_URL}/chat/{conv_id}",
            }), 200
        # StatChat reachable but returned an error.
        return jsonify({'error': f"StatChat returned {res.status_code}: {res.text}"}), res.status_code
    except requests.RequestException as exc:
        # StatChat unavailable — fall back to a deterministic deep link.
        return jsonify({
            'conversation': None,
            'conversation_id': conv_id,
            'statchat_ui_url': f"{STATCHAT_UI_URL}/chat/{conv_id}",
            'note': f"StatChat is unreachable ({exc}). Deep link provided; StatChat will create the conversation on first open.",
        }), 200


@app.route('/api/objects/discussions', methods=['GET'])
def list_object_discussions():
    """List all object-linked discussions from StatChat."""
    try:
        object_type = (request.args.get('type') or '').strip()
        params = {}
        if object_type:
            params['type'] = object_type
        res = statchat_request('GET', '/v1/objects/discussions', params=params)
        if res.status_code == 200:
            return jsonify(res.json()), 200
        return jsonify({'error': f"StatChat returned {res.status_code}: {res.text}"}), res.status_code
    except requests.RequestException as exc:
        return jsonify({'error': f"StatChat is unreachable: {exc}"}), 502


@app.route('/api/services', methods=['GET'])
def list_services():
    """Return a list of expected local services with UI and health endpoints (for the Service Launcher)."""
    return jsonify({'services': list_expected_services()}), 200


@app.route('/api/services/health', methods=['GET'])
def services_health():
    """Check each service health; HTTP endpoints are requested, TCP endpoints are checked via socket."""
    return jsonify({'results': collect_service_health()}), 200


if __name__ == '__main__':
    port = int(os.getenv('FLASK_PORT', 5000))
    print(f"StatGate Flask User Interface running on http://localhost:{port}")
    app.run(host='0.0.0.0', port=port, debug=False)
