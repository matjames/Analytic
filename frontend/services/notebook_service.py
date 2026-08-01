def is_notebook_execution_enabled():
    enabled = __import__('os').getenv('ENABLE_NOTEBOOK_EXECUTION', 'false').lower() in ('true', 'enable', 'enabled', '1', 'yes')
    if not enabled:
        return False
    if __import__('os').getenv('STATGATE_ENV', 'development').lower() == 'production':
        return False
    return True


def validate_notebook_execution_request(payload):
    if not isinstance(payload, dict):
        raise ValueError('Notebook execution payload must be a JSON object.')

    session_id = payload.get('session_id', 'default_session')
    tenant_id = payload.get('tenant_id', 'tenant-alpha')
    if not isinstance(session_id, str) or not session_id.strip():
        raise ValueError('session_id is required.')
    if not isinstance(tenant_id, str) or not tenant_id.strip():
        raise ValueError('tenant_id is required.')
    if payload.get('code') is not None and not isinstance(payload['code'], str):
        raise ValueError('code must be a string.')
    return {'session_id': session_id, 'tenant_id': tenant_id}


def build_kernel_session_scope(session_id, tenant_id, pd_module, requests_module, engine_module, statgate_module, list_tables_fn):
    return {
        'session_id': session_id,
        'tenant_id': tenant_id,
        'pd': pd_module,
        'requests': requests_module,
        'engine': engine_module,
        'statgate': statgate_module,
        'list_kaggle_tables': list_tables_fn,
    }
