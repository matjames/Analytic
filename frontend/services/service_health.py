"""
service_health.py — StatGate Service Registry & Health Check
Loads the service registry from config/services.json (data-driven)
instead of hardcoding service lists in code.  Falls back to a bundled
default list if the config file is missing.

This implements the directive's "Extract Configuration to Data"
principle: adding/removing a service is a config change, not a code change.
"""
import json
import os
import socket
from pathlib import Path

import requests

CONFIG_PATH = Path(__file__).resolve().parent.parent / 'config' / 'services.json'

# Fallback defaults (used only if config/services.json is missing).
_FALLBACK_SERVICES = [
    {"id": "analytics", "name": "StatGate Analytics", "ui": "http://localhost:5000", "health": "http://statgate-analytics:5000/health"},
    {"id": "core", "name": "StatGate Core", "ui": "http://localhost:8082", "health": "http://statgate-core:8080/health"},
    {"id": "registry", "name": "Registry API", "ui": "http://localhost:3007", "health": "http://statgate-registry-api:9090/health"},
    {"id": "postgres", "name": "Postgres DB", "ui": None, "health": "tcp://postgres:5432"},
    {"id": "redis", "name": "Redis", "ui": None, "health": "tcp://redis:6379"},
    {"id": "minio", "name": "MinIO", "ui": "http://localhost:9001", "health": "http://minio:9000/minio/health/ready"},
    {"id": "prometheus", "name": "Prometheus", "ui": "http://localhost:9095", "health": "http://prometheus:9090/-/healthy"},
    {"id": "grafana", "name": "Grafana", "ui": "http://localhost:3003", "health": "http://grafana:3000/api/health"},
    {"id": "alertmanager", "name": "Alertmanager", "ui": "http://localhost:9093", "health": "http://alertmanager:9093/-/healthy"},
    {"id": "statchat", "name": "StatChat", "ui": "http://localhost:3009", "health": "http://statchat-backend:4000/health"},
    {"id": "helpdesk", "name": "Operations Helpdesk", "ui": "http://localhost:3005", "health": "http://statgate-helpdesk-api:5000/api-docs"},
    {"id": "pms", "name": "Projects Management System", "ui": "http://localhost:3010", "health": "http://statgate-pms-api:8080/health"},
    {"id": "rms", "name": "Research Management System", "ui": "http://localhost:3011", "health": "http://statgate-rms-api:8080/health"},
]


def _load_services():
    """Load the service registry from config/services.json, falling back to defaults."""
    try:
        if CONFIG_PATH.exists():
            with open(CONFIG_PATH, 'r', encoding='utf-8') as f:
                data = json.load(f)
            services = data.get('services', [])
            if services:
                return services
    except Exception:
        pass
    return _FALLBACK_SERVICES


def list_expected_services():
    """Return the list of expected services (data-driven from config)."""
    return _load_services()


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
