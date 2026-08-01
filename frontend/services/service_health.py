import socket

import requests


def list_expected_services():
    return [
        {"id": "analytics", "name": "StatGate Analytics", "ui": "http://localhost:5001", "health": "http://statgate-analytics:5000/health"},
        {"id": "core", "name": "StatGate Core", "ui": "http://localhost:8081", "health": "http://statgate-core:8080/health"},
        {"id": "registry", "name": "Registry API", "ui": "http://localhost:3000", "health": "http://statgate-registry-api:9090/health"},
        {"id": "postgres", "name": "Postgres DB", "ui": None, "health": "tcp://postgres:5432"},
        {"id": "redis", "name": "Redis", "ui": None, "health": "tcp://redis:6379"},
        {"id": "minio", "name": "MinIO", "ui": "http://localhost:9001", "health": "http://minio:9000/minio/health/ready"},
        {"id": "prometheus", "name": "Prometheus", "ui": "http://localhost:9090", "health": "http://prometheus:9090/-/healthy"},
        {"id": "grafana", "name": "Grafana", "ui": "http://localhost:3000", "health": "http://grafana:3000/api/health"},
        {"id": "loki", "name": "Loki", "ui": None, "health": "http://loki:3100/ready"},
        {"id": "alertmanager", "name": "Alertmanager", "ui": "http://localhost:9093", "health": "http://alertmanager:9093/-/healthy"},
        {"id": "mailhog", "name": "MailHog", "ui": "http://localhost:8025", "health": "http://mailhog:8025"},
    ]


def check_service_status(service):
    url = service.get('health')
    status = 'unknown'
    detail = None
    try:
        if url and url.startswith('http'):
            response = requests.get(url, timeout=2)
            status = 'ok' if response.status_code >= 200 and response.status_code < 300 else f'bad({response.status_code})'
        elif url and url.startswith('tcp://'):
            _, endpoint = url.split('://', 1)
            host, port = endpoint.split(':')
            with socket.create_connection((host, int(port)), timeout=2):
                status = 'ok'
    except Exception as exc:
        status = 'down'
        detail = str(exc)
    return {'id': service.get('id'), 'name': service.get('name'), 'status': status, 'detail': detail, 'ui': service.get('ui')}


def collect_service_health():
    return [check_service_status(service) for service in list_expected_services()]
