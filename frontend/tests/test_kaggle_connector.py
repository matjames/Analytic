import importlib
import os


def reload_module():
    import kaggle_connector
    return importlib.reload(kaggle_connector)


def test_list_kaggle_tables_raises_when_db_unavailable_in_production():
    os.environ['STATGATE_ENV'] = 'production'
    os.environ.pop('KAGGLE_ALLOW_FALLBACK', None)

    module = reload_module()
    original_retry = module._retry
    module._retry = lambda *args, **kwargs: (_ for _ in ()).throw(RuntimeError('db unavailable'))
    try:
        try:
            module.list_kaggle_tables()
            assert False, 'expected RuntimeError when DB is unavailable in production'
        except RuntimeError:
            pass
    finally:
        module._retry = original_retry
        os.environ.pop('STATGATE_ENV', None)


def test_query_kaggle_table_requires_an_explicit_db_error_in_dev():
    os.environ.pop('STATGATE_ENV', None)
    os.environ.pop('KAGGLE_ALLOW_FALLBACK', None)

    module = reload_module()
    original_retry = module._retry
    module._retry = lambda *args, **kwargs: (_ for _ in ()).throw(RuntimeError('db unavailable'))
    try:
        try:
            module.query_kaggle_table('covid_19_data', limit=10)
            assert False, 'expected explicit DB error instead of hidden fallback'
        except RuntimeError:
            pass
    finally:
        module._retry = original_retry


def test_get_table_schema_returns_fallback_in_dev_when_enabled():
    os.environ.pop('STATGATE_ENV', None)
    os.environ['KAGGLE_ALLOW_FALLBACK'] = 'true'

    module = reload_module()
    original_retry = module._retry
    module._retry = lambda *args, **kwargs: (_ for _ in ()).throw(RuntimeError('db unavailable'))
    try:
        schema = module.get_table_schema('covid_19_data')
        assert isinstance(schema, list)
        assert any(col['column_name'] == 'id' for col in schema)
    finally:
        module._retry = original_retry
        os.environ.pop('KAGGLE_ALLOW_FALLBACK', None)


def test_get_table_schema_raises_in_production_when_db_unavailable():
    os.environ['STATGATE_ENV'] = 'production'
    os.environ.pop('KAGGLE_ALLOW_FALLBACK', None)

    module = reload_module()
    original_retry = module._retry
    module._retry = lambda *args, **kwargs: (_ for _ in ()).throw(RuntimeError('db unavailable'))
    try:
        try:
            module.get_table_schema('covid_19_data')
            assert False, 'expected RuntimeError when schema lookup is unavailable in production'
        except RuntimeError:
            pass
    finally:
        module._retry = original_retry
        os.environ.pop('STATGATE_ENV', None)


def test_get_table_schema_raises_when_table_not_found():
    os.environ.pop('STATGATE_ENV', None)
    os.environ.pop('KAGGLE_ALLOW_FALLBACK', None)

    module = reload_module()
    original_retry = module._retry
    original_get_conn = module._get_conn_from_pool

    class FakeCursor:
        def execute(self, *_args, **_kwargs):
            pass

        def fetchall(self):
            return []

        def close(self):
            pass

    class FakeConn:
        def cursor(self, cursor_factory=None):
            return FakeCursor()

    module._retry = lambda fn, *args, attempts=3, backoff=0.2, **kwargs: fn(*args, **kwargs)
    module._get_conn_from_pool = lambda: FakeConn()
    try:
        try:
            module.get_table_schema('no_such_table')
            assert False, 'expected module.TableNotFoundError when the table does not exist'
        except module.TableNotFoundError as exc:
            assert 'not found in schema' in str(exc)
    finally:
        module._retry = original_retry
        module._get_conn_from_pool = original_get_conn


def test_dataset_name_validation_rejects_invalid_names():
    module = reload_module()
    try:
        module.validate_dataset_name('Invalid-Name')
        assert False, 'expected ValueError for invalid dataset name'
    except ValueError as exc:
        assert 'Invalid analytics dataset name' in str(exc)

    try:
        module.validate_dataset_name('1starter')
        assert False, 'expected ValueError for dataset name starting with a digit'
    except ValueError as exc:
        assert 'Invalid analytics dataset name' in str(exc)


def test_list_kaggle_tables_filters_invalid_names(monkeypatch):
    os.environ.pop('STATGATE_ENV', None)
    os.environ.pop('KAGGLE_ALLOW_FALLBACK', None)

    module = reload_module()
    original_retry = module._retry
    original_get_conn = module._get_conn_from_pool

    class FakeCursor:
        def execute(self, *_args, **_kwargs):
            pass

        def fetchall(self):
            return [('covid_19_data',), ('InvalidName',), ('good_table',)]

        def close(self):
            pass

    class FakeConn:
        def cursor(self, cursor_factory=None):
            return FakeCursor()

    module._retry = lambda fn, *args, attempts=3, backoff=0.3, **kwargs: fn(*args, **kwargs)
    module._get_conn_from_pool = lambda: FakeConn()
    try:
        tables = module.list_kaggle_tables()
        assert 'covid_19_data' in tables
        assert 'good_table' in tables
        assert 'InvalidName' not in tables
    finally:
        module._retry = original_retry
        module._get_conn_from_pool = original_get_conn
