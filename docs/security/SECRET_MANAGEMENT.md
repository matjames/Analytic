# StatGate Secret Management (SG-SEC-2026-08)

## Removed from the repository

- Root .env and all per-service .env files -> placeholders only.
- secrets/ directory -> removed; README explains runtime injection.
- docker-compose.yml -> secrets parameterized with "must be set".
- docker/postgres-init/*.sql -> passwords injected via \getenv.
- monitoring configs (Trino catalog, Loki S3, Grafana, Airflow, Superset,
  MinIO, MLflow) -> environment injected.
- Code (Go/Node/Python) -> hardcoded secrets and default passwords removed.

## Rotation

Every credential that was previously committed MUST be treated as
compromised and rotated:

- Postgres passwords (KAGGLE_DB_PASSWORD, HELPDESK_DB_PASSWORD,
  REGISTRY_DB_PASSWORD, STATCHAT_DB_PASSWORD, PMS_DB_PASSWORD,
  RMS_DB_PASSWORD, GOVERNANCE_DB_PASSWORD, STATCOLLECT_DB_PASSWORD)
- STATGATE_REGISTRY_JWT_SECRET / JWT_SECRET (Registry signing)
- STATGATE_INTERNAL_API_KEY
- FLASK_SECRET_KEY, HELPDESK_JWT_SECRET / SECRETKEY
- MinIO, Grafana, Airflow, Superset credentials
- BASIC_AUTH_USERNAME / BASIC_AUTH_PASSWORD (Registry MFL)
- Redis password, JupyterHub cookie secret

Generation: openssl rand -hex 32

## Runtime sources (in priority order)

1. Orchestrator / platform secret manager (recommended in production)
2. Docker secrets mounted at /run/secrets/<NAME>
3. Process environment variables

## Pre-commit / CI

scripts/secret-scan.ps1 is wired in the CI security job; it fails the build
when secret markings or default credentials appear in tracked files.

## Git history

Because several values were previously committed, rotate FIRST and only then
pursue an approved history rewrite if policy requires removing them.