from airflow import DAG
from airflow.operators.python import PythonOperator
from datetime import datetime
import requests
import logging

def check_statchat():
    url = "http://statchat-backend:4000/health"
    try:
        r = requests.get(url, timeout=5)
        logging.info("StatChat health %s: %s", url, r.status_code)
    except Exception as e:
        logging.exception("Failed to reach StatChat service: %s", e)

with DAG(dag_id="statchat_sync", start_date=datetime(2023, 1, 1), schedule_interval="@hourly", catchup=False) as dag:
    t1 = PythonOperator(task_id="check_statchat_health", python_callable=check_statchat)
