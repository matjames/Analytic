Airflow DAGs for StatGate
=========================

This folder contains a few operational DAGs used by StatGate services.

Files added:

- `analytics_etl.py` — simple daily check of the Analytics service.
- `statchat_sync.py` — hourly check of the StatChat backend.
- `helpdesk_maintenance.py` — daily placeholder for helpdesk maintenance tasks.
- `requirements.txt` — Python packages required by these DAGs.

How to install requirements into the running Airflow container:

1. Copy or mount `requirements.txt` into the container or use the mounted DAGs path. The DAGs folder is mounted to `/opt/airflow/dags` in this compose setup.

2. Exec into the container and install:

```bash
docker compose exec airflow bash
pip install -r /opt/airflow/dags/requirements.txt
```

3. The scheduler should pick up the new DAGs automatically.

Customize the DAGs to add real ETL logic and database operations as needed.
