import os

os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
os.environ.setdefault('ANALYTICS_REQUIRE_AUTH', 'true')

from services.backend_proxy import build_backend_headers, make_go_request


def test_build_backend_headers_includes_identity_and_internal_key():
    headers = build_backend_headers(
        identity={'userId': 42, 'role': 'district', 'districtId': 'district-17'},
        internal_key='internal-shared-key',
    )

    assert headers['X-Tenant-ID'] == 'district:district-17'
    assert headers['X-User-Role'] == 'district'
    assert headers['X-User-Clearance'] == '4'
    assert headers['X-StatGate-Internal-Key'] == 'internal-shared-key'


def test_make_go_request_uses_backend_url_and_default_tenant_headers():
    request = make_go_request('GET', '/api/v1/stats', base_url='http://example.test', headers={'X-Test': '1'})

    assert request.url == 'http://example.test/api/v1/stats'
    assert request.headers['X-Test'] == '1'
