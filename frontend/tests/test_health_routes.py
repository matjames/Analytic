import os
import types
import app as analytics_module


class FakeResponse:
    def __init__(self, status_code=200):
        self.status_code = status_code


def test_health_endpoint_returns_healthy(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    client = analytics_module.app.test_client()

    response = client.get('/health')
    assert response.status_code == 200
    assert response.json['status'] == 'healthy'
    assert response.json['service'] == 'statgate-analytics'


def test_ready_endpoint_checks_dependencies(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)
    monkeypatch.setattr(analytics_module, 'is_db_connected', lambda: True)
    monkeypatch.setattr(analytics_module.requests, 'get', lambda url, timeout=None: FakeResponse(200))
    client = analytics_module.app.test_client()

    response = client.get('/ready')
    assert response.status_code == 200
    assert response.json['status'] == 'ready'
    assert response.json['db'] == 'connected'
    assert response.json['go_backend'] == 'healthy'


def test_ready_endpoint_includes_registry_when_auth_enabled(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', True)
    monkeypatch.setattr(analytics_module, 'REGISTRY_API_URL', 'http://statgate-registry-api:9090/api')
    monkeypatch.setattr(analytics_module, 'is_db_connected', lambda: True)

    def fake_get(url, timeout=None):
        if url.endswith('/ready'):
            return FakeResponse(200)
        return FakeResponse(404)

    monkeypatch.setattr(analytics_module.requests, 'get', fake_get)
    client = analytics_module.app.test_client()

    response = client.get('/ready')
    assert response.status_code == 200
    assert response.json['status'] == 'ready'
    assert response.json['registry'] == 'healthy'
    assert response.json['registry_health_url'] == 'http://statgate-registry-api:9090/ready'


def test_services_routes_return_expected_structures(monkeypatch):
    os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
    os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
    monkeypatch.setattr(analytics_module, 'AUTH_REQUIRED', False)

    monkeypatch.setattr(analytics_module, 'collect_service_health', lambda: [
        {'id': 'analytics', 'status': 'ok'},
        {'id': 'core', 'status': 'ok'},
    ])

    client = analytics_module.app.test_client()

    services_response = client.get('/api/services')
    assert services_response.status_code == 200
    assert 'services' in services_response.json
    assert any(item['id'] == 'analytics' for item in services_response.json['services'])

    health_response = client.get('/api/services/health')
    assert health_response.status_code == 200
    assert health_response.json['results'][0]['id'] == 'analytics'
    assert health_response.json['results'][0]['status'] == 'ok'
