import os
import importlib


def reload_module():
    import engine
    return importlib.reload(engine)


def test_get_dataset_summary_raises_table_not_found(monkeypatch):
    engine_module = reload_module()
    os.environ.pop('STATGATE_ENV', None)
    os.environ.pop('KAGGLE_ALLOW_FALLBACK', None)

    monkeypatch.setattr(engine_module, 'get_table_schema', lambda table_name: (_ for _ in ()).throw(engine_module.TableNotFoundError(f'Table "{table_name}" not found in schema "ml_staging".')))
    monkeypatch.setattr(engine_module, 'query_kaggle_table', lambda table_name, limit=100: [])

    engine = engine_module.StatGateAnalysisEngine()

    try:
        engine.get_dataset_summary('no_such_table')
        assert False, 'expected TableNotFoundError to propagate'
    except engine_module.TableNotFoundError as exc:
        assert 'not found in schema' in str(exc)


def test_get_dataset_summary_profiles_columns_and_records(monkeypatch):
    engine_module = reload_module()

    monkeypatch.setattr(engine_module, 'get_table_schema', lambda table_name: [
        {'column_name': 'id', 'data_type': 'integer', 'is_nullable': 'NO'},
        {'column_name': 'value', 'data_type': 'double precision', 'is_nullable': 'YES'},
    ])
    monkeypatch.setattr(engine_module, 'query_kaggle_table', lambda table_name, limit=100: [
        {'id': 1, 'value': 10.0},
        {'id': 2, 'value': 20.0},
        {'id': 3, 'value': 30.0},
    ])

    engine = engine_module.StatGateAnalysisEngine()
    result = engine.get_dataset_summary('covid_19_data')

    assert result['table_name'] == 'covid_19_data'
    assert result['row_count'] == 3
    assert isinstance(result['profiling'], list)
    assert len(result['profiling']) == 2
    assert result['sample_records'][0]['id'] == 1
    assert any(col['column_name'] == 'value' for col in result['profiling'])
    value_stats = next(col for col in result['profiling'] if col['column_name'] == 'value')
    assert value_stats['stats']['mean'] == 20.0
    assert value_stats['stats']['min'] == 10.0
    assert value_stats['stats']['max'] == 30.0
