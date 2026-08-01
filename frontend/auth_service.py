import os

import jwt


def _get_claim(claims, *names):
    for name in names:
        value = claims.get(name)
        if value not in (None, ''):
            return value
    return None


def _clearance_for_role(role):
    normalized = (role or 'public').lower()
    if normalized in {'admin', 'superadmin', 'platform_admin'}:
        return 5
    if normalized in {'district', 'district_admin', 'tenant_admin'}:
        return 4
    if normalized in {'analyst', 'editor', 'operator'}:
        return 3
    if normalized in {'viewer', 'read_only', 'guest'}:
        return 2
    return 1


def normalize_registry_claims(claims):
    """Normalize Registry JWT claims into the identity shape Analytics expects."""
    if not isinstance(claims, dict):
        raise ValueError('Registry token payload must be a JSON object.')

    user_id = _get_claim(claims, 'userId', 'user_id', 'sub', 'email', 'username')
    if user_id is None:
        raise ValueError('Registry token missing userId.')

    role = _get_claim(claims, 'role', 'roles', 'userRole') or 'public'
    tenant_id = _get_claim(claims, 'tenantId', 'tenant_id')
    district_id = _get_claim(claims, 'districtId', 'district_id')

    identity = {
        'userId': user_id,
        'role': role,
    }
    if tenant_id is not None:
        identity['tenantId'] = tenant_id
    if district_id is not None:
        identity['districtId'] = district_id
    if tenant_id is None and district_id is not None:
        identity['tenantId'] = f"district:{district_id}"
    return identity


def validate_registry_token(token, jwt_secret=None):
    """Validate a Registry JWT and return a normalized identity payload."""
    if not token:
        raise ValueError('Missing Registry token.')

    secret = jwt_secret or os.getenv('STATGATE_REGISTRY_JWT_SECRET', '')
    if not secret:
        raise ValueError('STATGATE_REGISTRY_JWT_SECRET is not configured.')

    try:
        claims = jwt.decode(token, secret, algorithms=['HS256'], options={'require': ['exp']})
    except jwt.PyJWTError as exc:
        raise ValueError('Registry token validation failed.') from exc

    return normalize_registry_claims(claims)


def analytics_context(identity):
    identity = identity or {}
    registry_role = identity.get('role', 'public')
    district_id = identity.get('districtId')
    tenant_id = identity.get('tenantId') or (f"district:{district_id}" if district_id else None) or 'statgate-national'
    return {
        'user_id': identity.get('userId'),
        'role': 'admin' if registry_role == 'admin' else 'analyst',
        'registry_role': registry_role,
        'district_id': district_id,
        'tenant_id': tenant_id,
        'clearance': _clearance_for_role(registry_role),
    }


def get_tenant_headers(identity=None, internal_key=None, default_tenant_id='tenant-alpha', default_user_role='analyst', default_user_clearance='2'):
    context = analytics_context(identity) if identity else {
        'tenant_id': default_tenant_id,
        'role': default_user_role,
        'clearance': default_user_clearance,
        'registry_role': default_user_role,
    }

    headers = {
        'X-Tenant-ID': context['tenant_id'],
        'X-User-Role': context.get('registry_role', context['role']),
        'X-User-Clearance': str(context['clearance']),
        'Content-Type': 'application/json',
    }

    if identity and identity.get('userId') is not None:
        headers['X-User-ID'] = str(identity['userId'])

    if internal_key:
        headers['X-StatGate-Internal-Key'] = internal_key

    return headers
