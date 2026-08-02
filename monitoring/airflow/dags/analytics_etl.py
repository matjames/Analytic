from airflow import DAG
from airflow.operators.python import PythonOperator
from datetime import datetime
import requests
import logging

def check_analytics():
    url = "http://statgate-analytics:5000/health"
    try:
        r = requests.get(url, timeout=5)
        logging.info("Analytics health %s: %s", url, r.status_code)
    except Exception as e:
        logging.exception("Failed to reach analytics service: %s", e)

with DAG(dag_id="analytics_etl", start_date=datetime(2023, 1, 1), schedule_interval="@daily", catchup=False) as dag:
    t1 = PythonOperator(task_id="check_analytics_health", python_callable=check_analytics)
