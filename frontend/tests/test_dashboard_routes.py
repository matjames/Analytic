import os
import json
import app as analytics_module


def test_dashboard_save_and_load_routes(tmp_path, monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')

    dashboard_data = {'widgets': [{'id': 'w1', 'type': 'chart'}]}
    payload = {'dashboard_id': 'tenant-alpha.dashboard_test', 'metadata': dashboard_data}

    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    # Patch persistence to avoid writing to production data store
    def fake_save_dashboard_metadata(dashboard_id, metadata):
        path = tmp_path / f"{dashboard_id}.json"
        path.write_text(json.dumps(metadata))
        return {'status': 'saved', 'path': str(path)}

    def fake_load_dashboard_metadata(dashboard_id, strict=False):
        path = tmp_path / f"{dashboard_id}.json"
        if not path.exists():
            if strict:
                raise FileNotFoundError()
            return {}
        return json.loads(path.read_text())

    monkeypatch.setattr(analytics_module.statgate_engine, 'save_dashboard_metadata', fake_save_dashboard_metadata)
    monkeypatch.setattr(analytics_module.statgate_engine, 'load_dashboard_metadata', fake_load_dashboard_metadata)

    response = client.post('/api/dashboard/save', json=payload)
    assert response.status_code == 200
    assert response.json['status'] == 'saved'

    response = client.get('/api/dashboard/load/tenant-alpha.dashboard_test')
    assert response.status_code == 200
    assert response.json['content_definition'] == dashboard_data


def test_dashboard_load_missing_returns_404(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')

    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    def fake_load_dashboard_metadata(dashboard_id, strict=False):
        if strict:
            raise FileNotFoundError()
        return {}

    monkeypatch.setattr(analytics_module.statgate_engine, 'load_dashboard_metadata', fake_load_dashboard_metadata)

    response = client.get('/api/dashboard/load/tenant-alpha.nonexistent')
    assert response.status_code == 404
    assert response.json['error'] == 'Dashboard not found.'


def test_get_kaggle_schema_returns_summary(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    monkeypatch.setattr(analytics_module.statgate_engine, 'get_dataset_summary', lambda table_name: {'dataset': table_name, 'columns': ['id', 'value']})

    response = client.get('/api/kaggle/schema/covid_19_data')
    assert response.status_code == 200
    assert response.json['dataset'] == 'covid_19_data'
    assert response.json['columns'] == ['id', 'value']


def test_get_kaggle_schema_handles_engine_error(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    def fail_summary(table_name):
        raise RuntimeError('engine unavailable')

    monkeypatch.setattr(analytics_module.statgate_engine, 'get_dataset_summary', fail_summary)

    response = client.get('/api/kaggle/schema/covid_19_data')
    assert response.status_code == 502
    assert response.json['error'] == 'engine unavailable'


def test_run_functional_analysis_endpoint_returns_result(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    def run_analysis(table_name, metric_id, tenant_id, user_clearance):
        return {'table': table_name, 'metric': metric_id, 'tenant': tenant_id, 'clearance': user_clearance}

    monkeypatch.setattr(analytics_module.statgate_engine, 'run_functional_analysis', run_analysis)

    response = client.post('/api/functional/analyze', json={'table_name': 'covid_19_data', 'metric_id': 'avg_latency'}, headers={'X-User-Clearance': '3'})
    assert response.status_code == 200
    assert response.json['table'] == 'covid_19_data'
    assert response.json['metric'] == 'avg_latency'
    assert response.json['clearance'] == 3


def test_run_functional_analysis_endpoint_invalid_clearance(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    response = client.post('/api/functional/analyze', json={'table_name': 'covid_19_data'}, headers={'X-User-Clearance': 'not-int'})
    assert response.status_code == 400
    assert response.json['error'] == 'X-User-Clearance must be an integer.'


def test_schema_health_endpoint_detects_drift_and_heals_assets(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    monkeypatch.setattr(analytics_module, 'detect_drift', lambda table_name: {'drifted': True, 'table_name': table_name, 'details': {'renamed_columns': {'old': 'new'}}})
    monkeypatch.setattr(analytics_module, 'heal_assets', lambda drift_report: {'patched': 1, 'assets': ['tenant-alpha.dashboard_test']})

    response = client.get('/api/schema-health/covid_19_data')
    assert response.status_code == 200
    assert response.json['drifted'] is True
    assert response.json['healing']['patched'] == 1
    assert response.json['healing']['assets'] == ['tenant-alpha.dashboard_test']


def test_get_kaggle_schema_returns_404_for_missing_table(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    def missing_dataset(table_name):
        raise analytics_module.TableNotFoundError(f'Table "{table_name}" not found in schema "ml_staging".')

    monkeypatch.setattr(analytics_module.statgate_engine, 'get_dataset_summary', missing_dataset)

    response = client.get('/api/kaggle/schema/no_such_table')
    assert response.status_code == 404
    assert 'not found in schema' in response.json['error']


def test_get_kaggle_schema_returns_400_for_invalid_dataset_name(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    response = client.get('/api/kaggle/schema/Invalid-Name')
    assert response.status_code == 400
    assert 'Invalid analytics dataset name' in response.json['error']


def test_agent_feedback_rejects_invalid_action(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    response = client.post('/api/agent/feedback', json={
        'report_id': 'report_1',
        'scenario_id': 'scenario_1',
        'action': 'invalid_action',
        'outcome': 'test',
        'tenant_id': 'tenant-alpha'
    })

    assert response.status_code == 400
    assert 'Invalid action' in response.json['error']


def test_agent_analyze_requires_goal(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    response = client.post('/api/agent/analyze', json={})
    assert response.status_code == 400
    assert 'goal is required' in response.json['error']


def test_agent_analyze_returns_report(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    def fake_analyze_intent(goal, tenant_id='tenant-alpha'):
        return {
            'goal': goal,
            'domain': 'health',
            'scenarios': [],
            'actions_needed': 0,
            'generated_at': '2026-08-01T00:00:00Z'
        }

    monkeypatch.setattr(analytics_module.agentic_engine, 'analyze_intent', fake_analyze_intent)

    response = client.post('/api/agent/analyze', json={'goal': 'Improve education outcomes'})
    assert response.status_code == 200
    assert response.json['goal'] == 'Improve education outcomes'
    assert response.json['domain'] == 'health'


def test_executive_summary_returns_fallback_on_error(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    monkeypatch.setattr(analytics_module, 'agentic_engine', analytics_module.agentic_engine)
    monkeypatch.setattr(analytics_module.agentic_engine, 'get_executive_summary', lambda tenant_id: (_ for _ in ()).throw(Exception('backend unavailable')))

    response = client.get('/api/executive/summary')
    assert response.status_code == 200
    assert response.json['system_health'] == 'Unknown'
    assert 'note' in response.json
    assert 'temporarily unavailable' in response.json['note']


def test_get_kaggle_datasets_returns_table_list_with_mocked_data(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    monkeypatch.setattr(analytics_module, 'list_kaggle_tables', lambda: ['covid_19_data', 'new_dataset'])

    response = client.get('/api/kaggle/datasets')
    assert response.status_code == 200
    assert response.json['schema'] == 'ml_staging'
    assert response.json['count'] == 2
    assert response.json['tables'] == ['covid_19_data', 'new_dataset']


def test_get_kaggle_datasets_returns_table_list(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    monkeypatch.setattr(analytics_module, 'list_kaggle_tables', lambda: ['covid_19_data', 'new_dataset'])
    client = analytics_module.app.test_client()

    response = client.get('/api/kaggle/datasets')
    assert response.status_code == 200
    assert response.json['schema'] == 'ml_staging'
    assert response.json['count'] == 2
    assert response.json['tables'] == ['covid_19_data', 'new_dataset']


def test_fetch_live_dataset_returns_404_for_missing_table(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    def missing_table(table_name, limit=100):
        raise analytics_module.TableNotFoundError(f'Table "{table_name}" not found in schema "ml_staging".')

    monkeypatch.setattr(analytics_module, 'query_kaggle_table', missing_table)

    response = client.get('/api/v1/datasets/fetch/no_such_table')
    assert response.status_code == 404
    assert 'not found in schema' in response.json['error']


def test_fetch_live_dataset_returns_400_for_invalid_name(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    response = client.get('/api/v1/datasets/fetch/Invalid-Name')
    assert response.status_code == 400
    assert 'Invalid analytics dataset name' in response.json['error']


def test_refresh_kaggle_datasets_returns_new_snapshots(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    monkeypatch.setattr(analytics_module, 'snapshot_all_tables', lambda: {'covid_19_data': 'ok', 'new_dataset': 'ok'})
    monkeypatch.setattr(analytics_module, 'list_kaggle_tables', lambda: ['covid_19_data', 'new_dataset'])

    response = client.post('/api/kaggle/datasets/refresh')
    assert response.status_code == 200
    assert response.json['count'] == 2
    assert 'new_dataset' in response.json['tables']
    assert response.json['snapshot_results']['new_dataset'] == 'ok'


def test_schema_health_returns_404_for_missing_table(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    def missing_drift(table_name):
        raise analytics_module.TableNotFoundError(f'Table "{table_name}" not found in schema "ml_staging".')

    monkeypatch.setattr(analytics_module, 'detect_drift', missing_drift)

    response = client.get('/api/schema-health/no_such_table')
    assert response.status_code == 404
    assert 'not found in schema' in response.json['error']


def test_schema_snapshot_all_returns_results(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    monkeypatch.setattr(analytics_module, 'snapshot_all_tables', lambda: {'covid_19_data': 'snapshotted'})

    response = client.post('/api/schema-health/snapshot/all')
    assert response.status_code == 200
    assert response.json['snapshotted'] == {'covid_19_data': 'snapshotted'}
