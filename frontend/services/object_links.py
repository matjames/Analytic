"""
object_links.py — StatGate Object Linkage Framework
Provides a universal "Connections" API so that every major object in
StatGate (datasets, reports, projects, dashboards, meetings, tasks,
documents, etc.) can be linked to any other object.

This implements the directive's "Every Object Should Be Connected"
principle.  Links are stored in the shared PostgreSQL `object_links`
table (created by docker/postgres-init/02-create-object-links.sql).
"""
import os
import logging

logger = logging.getLogger(__name__)

try:
    import psycopg2
    import psycopg2.extras
    PSYCOPG2_AVAILABLE = True
except ImportError:
    PSYCOPG2_AVAILABLE = False
    psycopg2 = None


def _get_conn():
    """Return a PostgreSQL connection or None if unavailable."""
    if not PSYCOPG2_AVAILABLE:
        return None
    try:
        return psycopg2.connect(
            host=os.getenv('KAGGLE_DB_HOST', 'localhost'),
            port=int(os.getenv('KAGGLE_DB_PORT', '5432')),
            dbname=os.getenv('KAGGLE_DB_NAME', 'statgate_ml_staging'),
            user=os.getenv('KAGGLE_DB_USER', 'Kaggle'),
            password=os.getenv('KAGGLE_DB_PASSWORD', 'REDACTED_PLACEHOLDER'),
            sslmode=os.getenv('KAGGLE_DB_SSLMODE', 'disable'),
        )
    except Exception as exc:
        logger.warning('Object links DB connection failed: %s', exc)
        return None


def _ensure_table(conn):
    """Ensure the object_links table exists (idempotent)."""
    with conn.cursor() as cur:
        cur.execute("""
            CREATE TABLE IF NOT EXISTS object_links (
                id            SERIAL PRIMARY KEY,
                source_type   VARCHAR(64)  NOT NULL,
                source_id     VARCHAR(255) NOT NULL,
                target_type   VARCHAR(64)  NOT NULL,
                target_id     VARCHAR(255) NOT NULL,
                relationship  VARCHAR(64)  NOT NULL DEFAULT 'related',
                tenant_id     VARCHAR(64)  NOT NULL DEFAULT 'tenant-alpha',
                created_by    VARCHAR(128),
                created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
                UNIQUE (source_type, source_id, target_type, target_id, relationship)
            )
        """)
        cur.execute("CREATE INDEX IF NOT EXISTS idx_object_links_source ON object_links(source_type, source_id)")
        cur.execute("CREATE INDEX IF NOT EXISTS idx_object_links_target ON object_links(target_type, target_id)")
        cur.execute("CREATE INDEX IF NOT EXISTS idx_object_links_tenant ON object_links(tenant_id)")
    conn.commit()


def create_link(source_type, source_id, target_type, target_id,
                relationship='related', tenant_id='tenant-alpha', created_by=None):
    """
    Create a link between two platform objects.
    Returns True on success, False if the link already exists or DB unavailable.
    """
    if not all([source_type, source_id, target_type, target_id]):
        logger.warning('Object link: source and target are required')
        return False

    conn = _get_conn()
    if conn is None:
        logger.info('[ObjectLinks] DB unavailable, link not persisted: %s:%s -> %s:%s',
                    source_type, source_id, target_type, target_id)
        return False

    try:
        _ensure_table(conn)
        with conn.cursor() as cur:
            cur.execute("""
                INSERT INTO object_links (source_type, source_id, target_type, target_id, relationship, tenant_id, created_by)
                VALUES (%s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (source_type, source_id, target_type, target_id, relationship) DO NOTHING
            """, (source_type, source_id, target_type, target_id, relationship, tenant_id, created_by))
        conn.commit()
        return True
    except Exception as exc:
        logger.warning('[ObjectLinks] Create link failed: %s', exc)
        return False
    finally:
        conn.close()


def get_links(object_type, object_id, tenant_id='tenant-alpha'):
    """
    Return all links where the given object is either the source or target.
    Returns a list of dicts with the connected object and relationship.
    """
    conn = _get_conn()
    if conn is None:
        return []

    try:
        _ensure_table(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                SELECT source_type, source_id, target_type, target_id, relationship, created_at
                FROM object_links
                WHERE tenant_id = %s
                  AND ((source_type = %s AND source_id = %s) OR (target_type = %s AND target_id = %s))
                ORDER BY created_at DESC
            """, (tenant_id, object_type, object_id, object_type, object_id))
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[ObjectLinks] Get links failed: %s', exc)
        return []
    finally:
        conn.close()


def remove_link(source_type, source_id, target_type, target_id, relationship='related'):
    """Remove a link between two objects."""
    conn = _get_conn()
    if conn is None:
        return False
    try:
        _ensure_table(conn)
        with conn.cursor() as cur:
            cur.execute("""
                DELETE FROM object_links
                WHERE source_type = %s AND source_id = %s
                  AND target_type = %s AND target_id = %s
                  AND relationship = %s
            """, (source_type, source_id, target_type, target_id, relationship))
        conn.commit()
        return True
    except Exception as exc:
        logger.warning('[ObjectLinks] Remove link failed: %s', exc)
        return False
    finally:
        conn.close()