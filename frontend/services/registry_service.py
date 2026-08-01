from flask import g as flask_g

from auth_service import validate_registry_token


def get_registry_identity_from_request(request, session, jwt_secret):
    """Resolve the authenticated registry identity from session or bearer token."""
    identity = getattr(request, 'g', None)
    if identity is not None:
        statgate_identity = getattr(identity, 'statgate_identity', None)
        if statgate_identity is not None:
            return statgate_identity

    try:
        statgate_identity = getattr(flask_g, 'statgate_identity', None)
    except RuntimeError:
        statgate_identity = None
    if statgate_identity is not None:
        return statgate_identity

    identity = session.get('statgate_identity') if session else None
    if identity is not None:
        return identity

    auth_header = request.headers.get('Authorization', '')
    if auth_header.lower().startswith('bearer '):
        token = auth_header.split(None, 1)[1].strip()
        try:
            return validate_registry_token(token, jwt_secret)
        except ValueError:
            return None
    return None
