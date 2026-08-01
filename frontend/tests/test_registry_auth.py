import os
import time

os.environ.setdefault('FLASK_SECRET_KEY', 'test-secret')
os.environ.setdefault('STATGATE_REGISTRY_JWT_SECRET', 'test-secret')
os.environ.setdefault('ANALYTICS_REQUIRE_AUTH', 'true')

import jwt

import app as analytics_app


def _make_token(claims):
    return jwt.encode({**claims, 'exp': int(time.time()) + 3600}, 'test-secret', algorithm='HS256')


def test_validate_registry_token_builds_identity():
    token = _make_token({'userId': 42, 'role': 'district', 'districtId': 'district-17'})

    identity = analytics_app.validate_registry_token(token)

    assert identity['userId'] == 42
    assert identity['role'] == 'district'
    assert identity['districtId'] == 'district-17'


def test_analytics_context_maps_registry_claims_to_platform_context():
    identity = {'userId': 42, 'role': 'district', 'districtId': 'district-17'}

    context = analytics_app.analytics_context(identity)

    assert context['tenant_id'] == 'district:district-17'
    assert context['registry_role'] == 'district'
    assert context['clearance'] == 4


def test_notebook_execution_is_disabled_by_default_and_in_production():
    os.environ.pop('ENABLE_NOTEBOOK_EXECUTION', None)
    os.environ.pop('STATGATE_ENV', None)
    assert analytics_app.is_notebook_execution_enabled() is False

    os.environ['ENABLE_NOTEBOOK_EXECUTION'] = 'true'
    os.environ['STATGATE_ENV'] = 'production'
    assert analytics_app.is_notebook_execution_enabled() is False
