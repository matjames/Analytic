import os

from services.notebook_service import build_kernel_session_scope, is_notebook_execution_enabled


def test_notebook_execution_disabled_by_default():
    os.environ.pop('ENABLE_NOTEBOOK_EXECUTION', None)
    os.environ.pop('STATGATE_ENV', None)

    assert is_notebook_execution_enabled() is False


def test_notebook_execution_can_be_enabled_in_dev_for_trusted_runs():
    os.environ['ENABLE_NOTEBOOK_EXECUTION'] = 'true'
    os.environ['STATGATE_ENV'] = 'development'

    assert is_notebook_execution_enabled() is True


def test_build_kernel_session_scope_includes_runtime_contract():
    scope = build_kernel_session_scope(
        session_id='abc',
        tenant_id='tenant-42',
        pd_module={'DataFrame': object},
        requests_module={'get': object},
        engine_module={'name': 'engine'},
        statgate_module={'name': 'statgate'},
        list_tables_fn=lambda: ['alpha'],
    )

    assert scope['session_id'] == 'abc'
    assert scope['tenant_id'] == 'tenant-42'
    assert scope['list_kaggle_tables'] is not None
    assert scope['engine'] == {'name': 'engine'}


def test_notebook_execution_is_blocked_in_production_even_when_enabled():
    os.environ['ENABLE_NOTEBOOK_EXECUTION'] = 'true'
    os.environ['STATGATE_ENV'] = 'production'

    assert is_notebook_execution_enabled() is False
