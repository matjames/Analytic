import importlib.util
from pathlib import Path


def _load_env_check_module():
    module_path = Path(__file__).resolve().parents[1] / 'env_check.py'
    spec = importlib.util.spec_from_file_location('env_check', module_path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_collect_missing_keys_flags_required_values(tmp_path):
    env_file = tmp_path / '.env'
    env_file.write_text(
        'STATGATE_INTERNAL_API_KEY=abc\n'
        'FLASK_SECRET_KEY=def\n'
        'STATGATE_REGISTRY_JWT_SECRET=ghi\n'
        'KAGGLE_DB_HOST=localhost\n'
        'KAGGLE_DB_USER=app\n'
        'KAGGLE_DB_PASSWORD=secret\n'
        'KAGGLE_DB_NAME=statgate_ml_staging\n'
        'KAGGLE_STAGING_SCHEMA=ml_staging\n'
    )

    env_check = _load_env_check_module()
    missing = env_check.collect_missing_keys(env_file)

    assert missing == []


def test_collect_missing_keys_detects_missing_required_values(tmp_path):
    env_file = tmp_path / '.env'
    env_file.write_text('STATGATE_INTERNAL_API_KEY=abc\n')

    env_check = _load_env_check_module()
    missing = env_check.collect_missing_keys(env_file)

    assert 'FLASK_SECRET_KEY' in missing
    assert 'STATGATE_REGISTRY_JWT_SECRET' in missing


def test_dependency_probe_reports_missing_packages(monkeypatch):
    env_check = _load_env_check_module()

    def fake_import(name):
        if name == 'flask':
            raise ModuleNotFoundError()
        return object()

    monkeypatch.setattr(env_check.importlib, 'import_module', fake_import)
    missing = env_check.find_missing_dependencies(['flask', 'requests'])

    assert 'flask' in missing
    assert 'requests' not in missing
