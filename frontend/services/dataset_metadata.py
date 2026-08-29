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
                table_name      VARCHAR(128) NOT NULL,
                tenant_id       VARCHAR(128) NOT NULL DEFAULT 'tenant-alpha',
                workspace_id    VARCHAR(128),
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
        cur.execute("ALTER TABLE dataset_metadata ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)")
        cur.execute("ALTER TABLE dataset_metadata ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(128) NOT NULL DEFAULT 'tenant-alpha'")
        cur.execute("""
            DO $$
            DECLARE
                legacy_constraint_name TEXT;
            BEGIN
                SELECT c.conname
                INTO legacy_constraint_name
                FROM pg_constraint c
                JOIN pg_class t ON t.oid = c.conrelid
                WHERE t.relname = 'dataset_metadata'
                  AND c.contype IN ('p', 'u')
                  AND (
                      SELECT ARRAY_AGG(a.attname ORDER BY keys.ordinality)
                      FROM UNNEST(c.conkey) WITH ORDINALITY AS keys(attnum, ordinality)
                      JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = keys.attnum
                  ) = ARRAY['table_name'];

                IF legacy_constraint_name IS NOT NULL THEN
                    EXECUTE FORMAT('ALTER TABLE dataset_metadata DROP CONSTRAINT %I', legacy_constraint_name);
                END IF;
            END $$;
        """)
        cur.execute("CREATE INDEX IF NOT EXISTS idx_dataset_metadata_tenant_workspace ON dataset_metadata(tenant_id, workspace_id)")
        cur.execute("CREATE INDEX IF NOT EXISTS idx_dataset_metadata_workspace ON dataset_metadata(workspace_id)")
        cur.execute("""
            CREATE UNIQUE INDEX IF NOT EXISTS idx_dataset_metadata_scope_unique
            ON dataset_metadata (tenant_id, COALESCE(workspace_id, ''), table_name)
        """)
    conn.commit()


def upsert_dataset_metadata(table_name, owner_id=None, steward_id=None,
                            description='', classification='public',
                            tags=None, row_count=0, col_count=0, workspace_id=None, tenant_id='tenant-alpha'):
    """Create or update metadata for a dataset."""
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_table(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                INSERT INTO dataset_metadata (table_name, tenant_id, workspace_id, owner_id, steward_id, description, classification, tags, row_count, col_count, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, CURRENT_TIMESTAMP)
                ON CONFLICT DO NOTHING
                RETURNING *
            """, (table_name, tenant_id, workspace_id, owner_id, steward_id, description, classification,
                  json.dumps(tags or []), row_count, col_count))
            row = cur.fetchone()
            if row is None:
                cur.execute("""
                    UPDATE dataset_metadata SET
                    owner_id = %s,
                    steward_id = %s,
                    description = %s,
                    classification = %s,
                    tags = %s,
                    row_count = %s,
                    col_count = %s,
                    updated_at = CURRENT_TIMESTAMP
                WHERE table_name = %s
                  AND tenant_id = %s
                  AND workspace_id IS NOT DISTINCT FROM %s
                RETURNING *
                """, (owner_id, steward_id, description, classification, json.dumps(tags or []),
                      row_count, col_count, table_name, tenant_id, workspace_id))
                row = cur.fetchone()
        conn.commit()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[DatasetMetadata] Upsert failed: %s', exc)
        return None
    finally:
        conn.close()


def get_dataset_metadata(table_name, workspace_id=None, tenant_id='tenant-alpha'):
    """Get metadata for a single dataset."""
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_table(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("SELECT * FROM dataset_metadata WHERE table_name = %s AND tenant_id = %s AND (%s IS NULL OR workspace_id = %s)",
                        (table_name, tenant_id, workspace_id, workspace_id))
            row = cur.fetchone()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[DatasetMetadata] Get failed: %s', exc)
        return None
    finally:
        conn.close()


def list_dataset_metadata(classification=None, workspace_id=None, tenant_id='tenant-alpha'):
    """List all dataset metadata records, optionally filtered by classification."""
    conn = _get_conn()
    if conn is None:
        return []
    try:
        _ensure_table(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            query = "SELECT * FROM dataset_metadata WHERE tenant_id = %s AND (%s IS NULL OR workspace_id = %s)"
            args = [tenant_id, workspace_id, workspace_id]
            if classification:
                query += " AND classification = %s"
                args.append(classification)
            query += " ORDER BY table_name"
            cur.execute(query, args)
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[DatasetMetadata] List failed: %s', exc)
        return []
    finally:
        conn.close()


def search_datasets(query, limit=20, workspace_id=None, tenant_id='tenant-alpha'):
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
                WHERE tenant_id = %s AND (%s IS NULL OR workspace_id = %s)
                  AND (table_name ILIKE %s
                   OR description ILIKE %s
                   OR owner_id ILIKE %s
                   OR steward_id ILIKE %s
                   OR classification ILIKE %s
                   OR tags::text ILIKE %s)
                ORDER BY table_name
                LIMIT %s
            """, (tenant_id, workspace_id, workspace_id, like, like, like, like, like, like, limit))
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[DatasetMetadata] Search failed: %s', exc)
        return []
    finally:
        conn.close()


def remove_dataset_metadata(table_name, workspace_id=None, tenant_id='tenant-alpha'):
    """Remove metadata for a dataset."""
    conn = _get_conn()
    if conn is None:
        return False
    try:
        _ensure_table(conn)
        with conn.cursor() as cur:
            cur.execute("DELETE FROM dataset_metadata WHERE table_name = %s AND tenant_id = %s AND (%s IS NULL OR workspace_id = %s)",
                        (table_name, tenant_id, workspace_id, workspace_id))
        conn.commit()
        return True
    except Exception as exc:
        logger.warning('[DatasetMetadata] Remove failed: %s', exc)
        return False
    finally:
        conn.close()
