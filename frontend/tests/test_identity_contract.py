import os
import time

os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
os.environ.setdefault('ANALYTICS_REQUIRE_AUTH', 'true')

import jwt

from auth_service import (
    analytics_context,
    get_tenant_headers,
    normalize_registry_claims,
    validate_registry_token,
)


def _make_token(claims):
    return jwt.encode({**claims, 'exp': int(time.time()) + 3600}, 'test-secret', algorithm='HS256')


def test_normalize_registry_claims_builds_identity_contract():
    claims = {'userId': 42, 'role': 'district', 'districtId': 'district-17'}

    payload = normalize_registry_claims(claims)

    assert payload['userId'] == 42
    assert payload['role'] == 'district'
    assert payload['districtId'] == 'district-17'


def test_validate_registry_token_and_analytics_context_align():
    token = _make_token({'userId': 42, 'role': 'district', 'districtId': 'district-17'})

    identity = validate_registry_token(token, 'test-secret')
    context = analytics_context(identity)

    assert identity['userId'] == 42
    assert context['tenant_id'] == 'district:district-17'
    assert context['registry_role'] == 'district'
    assert context['clearance'] == 4


def test_get_tenant_headers_adds_platform_context_and_internal_key():
    headers = get_tenant_headers(
        identity={'userId': 42, 'role': 'district', 'districtId': 'district-17'},
        internal_key='internal-shared-key',
    )

    assert headers['X-Tenant-ID'] == 'district:district-17'
    assert headers['X-User-Role'] == 'district'
    assert headers['X-User-Clearance'] == '4'
    assert headers['X-StatGate-Internal-Key'] == 'internal-shared-key'


def test_validate_registry_token_supports_tenant_claims_and_role_mapping():
    token = _make_token({'sub': 'user-7', 'role': 'viewer', 'tenantId': 'tenant-7'})

    identity = validate_registry_token(token, 'test-secret')
    context = analytics_context(identity)

    assert identity['userId'] == 'user-7'
    assert identity['tenantId'] == 'tenant-7'
    assert context['tenant_id'] == 'tenant-7'
    assert context['clearance'] == 2
