from flask import Flask, g

from services.registry_service import get_registry_identity_from_request


class DummyRequest:
    def __init__(self, headers=None):
        self.headers = headers or {}
        self.g = type('Ctx', (), {})()


class DummySession(dict):
    pass


def test_get_registry_identity_from_request_reads_session_identity():
    request = DummyRequest()
    session = {'statgate_identity': {'userId': 99, 'role': 'district', 'districtId': 'district-99'}}

    identity = get_registry_identity_from_request(request, session, 'secret')

    assert identity['userId'] == 99
    assert identity['districtId'] == 'district-99'


def test_get_registry_identity_from_request_returns_none_for_missing_auth():
    request = DummyRequest(headers={})

    identity = get_registry_identity_from_request(request, {}, 'secret')

    assert identity is None


def test_get_registry_identity_from_request_reads_flask_g_identity():
    request = DummyRequest(headers={})
    app = Flask(__name__)

    with app.app_context():
        g.statgate_identity = {'userId': 100, 'role': 'district', 'districtId': 'district-100'}
        identity = get_registry_identity_from_request(request, {}, 'secret')

    assert identity['userId'] == 100
    assert identity['districtId'] == 'district-100'
