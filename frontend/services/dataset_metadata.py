"""
dataset_metadata.py — StatGate Dataset Metadata & Search Service
Implements the directive's requirement that datasets must be:
discoverable, searchable, and carry rich business metadata.

Adds a dataset_metadata table to PostgreSQL with fields for owner,
steward, description, tags, classification, and full-text search.
"""
import os
import json
import logging
from datetime import datetime, timezone

logger = logging.getLogger(__name__)

try:
    import psycopg2
    import psycopg2.extras
    PSYCOPG2_AVAILABLE = True
except ImportError:
    PSYCOPG2_AVAILABLE = False
    psycopg2 = None


def _get_conn():
    if not PSYCOPG2_AVAILABLE:
        return None
    try:
        return psycopg2.connect(
            host=os.getenv('KAGGLE_DB_HOST', 'localhost'),
            port=int(os.getenv('KAGGLE_DB_PORT', '5432')),
            dbname=os.getenv('KAGGLE_DB_NAME', 'statgate_ml_staging'),
            user=os.getenv('KAGGLE_DB_USER', 'Kaggle'),
            password=os.getenv('KAGGLE_DB_PASSWORD', ''),
            sslmode=os.getenv('KAGGLE_DB_SSLMODE', 'disable'),
        )
    except Exception as exc:
        logger.warning('Dataset metadata DB connection failed: %s', exc)
        return None


def _ensure_table(conn):
    with conn.cursor() as cur:
        cur.execute("""
            CREATE TABLE IF NOT EXISTS dataset_metadata (
                table_name      VARCHAR(128) PRIMARY KEY,
                owner_id        VARCHAR(128),
                steward_id      VARCHAR(128),
                description     TEXT,
                classification  VARCHAR(32) DEFAULT 'public',
                tags            JSONB NOT NULL DEFAULT '[]',
                row_count       BIGINT DEFAULT 0,
                col_count       INT DEFAULT 0,
                latest_snapshot TIMESTAMPTZ,
                created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
        """)
        cur.execute("""
            CREATE INDEX IF NOT EXISTS idx_dataset_metadata_class ON dataset_metadata(classification)
        """)
    conn.commit()


def upsert_dataset_metadata(table_name, owner_id=None, steward_id=None,
                            description='', classification='public',
                            tags=None, row_count=0, col_count=0):
    """Create or update metadata for a dataset."""
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_table(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                INSERT INTO dataset_metadata (table_name, owner_id, steward_id, description, classification, tags, row_count, col_count, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, CURRENT_TIMESTAMP)
                ON CONFLICT (table_name) DO UPDATE SET
                    owner_id = EXCLUDED.owner_id,
                    steward_id = EXCLUDED.steward_id,
                    description = EXCLUDED.description,
                    classification = EXCLUDED.classification,
                    tags = EXCLUDED.tags,
                    row_count = EXCLUDED.row_count,
                    col_count = EXCLUDED.col_count,
                    updated_at = CURRENT_TIMESTAMP
                RETURNING *
            """, (table_name, owner_id, steward_id, description, classification,
                  json.dumps(tags or []), row_count, col_count))
            row = cur.fetchone()
        conn.commit()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[DatasetMetadata] Upsert failed: %s', exc)
        return None
    finally:
        conn.close()


def get_dataset_metadata(table_name):
    """Get metadata for a single dataset."""
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_table(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("SELECT * FROM dataset_metadata WHERE table_name = %s", (table_name,))
            row = cur.fetchone()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[DatasetMetadata] Get failed: %s', exc)
        return None
    finally:
        conn.close()


def list_dataset_metadata(classification=None):
    """List all dataset metadata records, optionally filtered by classification."""
    conn = _get_conn()
    if conn is None:
        return []
    try:
        _ensure_table(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            if classification:
                cur.execute("SELECT * FROM dataset_metadata WHERE classification = %s ORDER BY table_name",
                            (classification,))
            else:
                cur.execute("SELECT * FROM dataset_metadata ORDER BY table_name")
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[DatasetMetadata] List failed: %s', exc)
        return []
    finally:
        conn.close()


def search_datasets(query, limit=20):
    """
    Full-text search across dataset table names, descriptions, tags,
    owners, and stewards.  Implements the directive's requirement that
    datasets must be "searchable".
    """
    conn = _get_conn()
    if conn is None:
        return []
    try:
        _ensure_table(conn)
        like = f"%{query}%"
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                SELECT * FROM dataset_metadata
                WHERE table_name ILIKE %s
                   OR description ILIKE %s
                   OR owner_id ILIKE %s
                   OR steward_id ILIKE %s
                   OR classification ILIKE %s
                   OR tags::text ILIKE %s
                ORDER BY table_name
                LIMIT %s
            """, (like, like, like, like, like, like, limit))
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[DatasetMetadata] Search failed: %s', exc)
        return []
    finally:
        conn.close()


def remove_dataset_metadata(table_name):
    """Remove metadata for a dataset."""
    conn = _get_conn()
    if conn is None:
        return False
    try:
        _ensure_table(conn)
        with conn.cursor() as cur:
            cur.execute("DELETE FROM dataset_metadata WHERE table_name = %s", (table_name,))
        conn.commit()
        return True
    except Exception as exc:
        logger.warning('[DatasetMetadata] Remove failed: %s', exc)
        return False
    finally:
        conn.close()