"""
event_bus.py — StatGate Cross-Module Event Bus
Publishes domain events to Redis for consumption by other StatGate services
(StatChat, Go Core, Registry, etc.).

Event schema:
{
    "event_type": "dataset.imported" | "report.generated" | "anomaly.detected" | ...,
    "source": "analytics" | "core" | "registry" | "statchat",
    "object_type": "dataset" | "report" | "project" | "dashboard" | ...,
    "object_id": "covid_19_data" | "report-123" | ...,
    "tenant_id": "tenant-alpha",
    "payload": { ... },
    "timestamp": "2026-08-05T12:00:00Z"
}
"""
import json
import os
import time
import logging
import datetime

logger = logging.getLogger(__name__)

# Redis connection (lazy import so the module can be used without redis installed)
try:
    import redis
    REDIS_AVAILABLE = True
except ImportError:
    REDIS_AVAILABLE = False
    redis = None

EVENT_CHANNEL = os.getenv('STATGATE_EVENT_CHANNEL', 'statgate:events')


def _get_redis_client():
    """Return a Redis client or None if Redis is unavailable."""
    if not REDIS_AVAILABLE:
        return None
    try:
        return redis.Redis(
            host=os.getenv('REDIS_HOST', 'localhost'),
            port=int(os.getenv('REDIS_PORT', '6379')),
            db=int(os.getenv('REDIS_DB', '0')),
            socket_timeout=2,
            socket_connect_timeout=2,
            decode_responses=True,
        )
    except Exception as exc:
        logger.warning('Redis client init failed: %s', exc)
        return None


def publish_event(event_type, object_type, object_id, tenant_id='tenant-alpha', payload=None):
    """
    Publish a domain event to the StatGate event bus.

    Args:
        event_type: e.g. 'dataset.imported', 'report.generated', 'anomaly.detected'
        object_type: e.g. 'dataset', 'report', 'project', 'dashboard'
        object_id: e.g. 'covid_19_data', 'report-123'
        tenant_id: tenant scope
        payload: dict of additional event data
    """
    if not event_type or not object_type or not object_id:
        logger.warning('Event bus: event_type, object_type, and object_id are required')
        return False

    event = {
        'event_type': event_type,
        'source': 'analytics',
        'object_type': object_type,
        'object_id': object_id,
        'tenant_id': tenant_id,
        'payload': payload or {},
        'timestamp': datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z'),
    }

    client = _get_redis_client()
    if client is None:
        # Redis unavailable — log the event for observability
        logger.info('[EventBus] Redis unavailable, event logged: %s', json.dumps(event))
        return False

    try:
        client.publish(EVENT_CHANNEL, json.dumps(event))
        logger.info('[EventBus] Published %s for %s:%s', event_type, object_type, object_id)
        return True
    except Exception as exc:
        logger.warning('[EventBus] Publish failed: %s', exc)
        return False


# ── Convenience publishers for common domain events ──

def publish_dataset_imported(table_name, tenant_id='tenant-alpha', row_count=0, extra=None):
    """Publish a dataset.imported event."""
    payload = {'table_name': table_name, 'row_count': row_count}
    if extra:
        payload.update(extra)
    return publish_event(
        'dataset.imported',
        'dataset',
        table_name,
        tenant_id=tenant_id,
        payload=payload,
    )


def publish_anomaly_detected(alert, tenant_id='tenant-alpha'):
    """Publish an anomaly.detected event from an alert dict."""
    return publish_event(
        'anomaly.detected',
        'alert',
        alert.get('id', 'alert-unknown'),
        tenant_id=tenant_id,
        payload=alert,
    )


def publish_agent_report_generated(report_id, tenant_id='tenant-alpha', goal='', extra=None):
    """Publish an agent.report.generated event."""
    payload = {'report_id': report_id, 'goal': goal}
    if extra:
        payload.update(extra)
    return publish_event(
        'agent.report.generated',
        'report',
        report_id,
        tenant_id=tenant_id,
        payload=payload,
    )


def publish_dashboard_saved(dashboard_id, tenant_id='tenant-alpha', extra=None):
    """Publish a dashboard.saved event."""
    return publish_event(
        'dashboard.saved',
        'dashboard',
        dashboard_id,
        tenant_id=tenant_id,
        payload=extra or {},
    )