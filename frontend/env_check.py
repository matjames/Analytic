import importlib
import os
import re
from pathlib import Path

REQUIRED_ENV_KEYS = [
    'STATGATE_INTERNAL_API_KEY',
    'FLASK_SECRET_KEY',
    'STATGATE_REGISTRY_JWT_SECRET',
]

REQUIRED_DB_KEYS = [
    'KAGGLE_DB_HOST',
    'KAGGLE_DB_NAME',
    'KAGGLE_DB_USER',
    'KAGGLE_DB_PASSWORD',
    'KAGGLE_STAGING_SCHEMA',
]


def _read_env_file(path):
    if not path or not os.path.exists(path):
        return {}

    values = {}
    for line in Path(path).read_text(encoding='utf-8').splitlines():
        if not line.strip() or line.strip().startswith('#'):
            continue
        if '=' not in line:
            continue
        key, value = line.split('=', 1)
        values[key.strip()] = value.strip()
    return values


def collect_missing_keys(env_file_path):
    env_path = Path(env_file_path)
    values = _read_env_file(env_path)
    missing = []
    required = REQUIRED_ENV_KEYS + REQUIRED_DB_KEYS

    for name in required:
        if not values.get(name):
            missing.append(name)

    return missing


def find_missing_dependencies(packages):
    missing = []
    for name in packages:
        try:
            importlib.import_module(name)
        except ModuleNotFoundError:
            missing.append(name)
    return missing


def build_startup_report(env_file_path='.env'):
    missing = collect_missing_keys(env_file_path)
    dependency_status = find_missing_dependencies(['flask', 'requests', 'pandas', 'duckdb', 'psycopg2'])
    return {
        'missing_env_keys': missing,
        'missing_python_dependencies': dependency_status,
        'ready': not missing and not dependency_status,
    }


if __name__ == '__main__':
    report = build_startup_report()
    print(report)
