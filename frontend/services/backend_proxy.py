import os

import requests

from auth_service import get_tenant_headers


def build_backend_headers(identity=None, internal_key=None, default_tenant_id='tenant-alpha', default_user_role='analyst', default_user_clearance='2'):
    return get_tenant_headers(
        identity,
        internal_key,
        default_tenant_id=default_tenant_id,
        default_user_role=default_user_role,
        default_user_clearance=default_user_clearance,
    )


def make_go_request(method, path, base_url=None, **kwargs):
    base_url = base_url or os.getenv('GO_BACKEND_URL', 'http://localhost:8080')
    timeout = kwargs.pop('timeout', float(os.getenv('GO_REQUEST_TIMEOUT_SECONDS', '5')))
    request = requests.Request(method, f"{base_url.rstrip('/')}{path}", **kwargs)
    prepared = request.prepare()
    prepared.timeout = timeout
    return prepared
