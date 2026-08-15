"""
registry_integration.py — StatGate Registry ↔ Analytics Integration
Bridges the Field Operations Registry database (`kaggle`, REGISTRY_DB_*)
into the Analytics ecosystem so that facility data becomes analytical
datasets that can be profiled, discussed via StatChat, and linked as
platform objects.

Implements the directive: "Field Operations Registry Isolated" fix —
facilities no longer live in a silo; they flow into the unified platform.
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


def _get_registry_conn():
    """Connect to the Registry database (`kaggle`), using REGISTRY_DB_* env vars."""
    if not PSYCOPG2_AVAILABLE:
        return None
    try:
        return psycopg2.connect(
            host=os.getenv('REGISTRY_DB_HOST', 'localhost'),
            port=int(os.getenv('REGISTRY_DB_PORT', '5432')),
            dbname=os.getenv('REGISTRY_DB_NAME', 'kaggle'),
            user=os.getenv('REGISTRY_DB_USER', 'Kaggle'),
            password=os.getenv('REGISTRY_DB_PASSWORD', ''),
            sslmode=os.getenv('REGISTRY_DB_SSLMODE', 'disable'),
        )
    except Exception as exc:
        logger.warning('Registry DB connection failed: %s', exc)
        return None


def list_registry_facilities(limit=100, offset=0, district_id=None):
    """
    List facilities from the Registry database.  Returns a list of dicts
    so Analytics can treat facilities as first-class analytical objects.
    """
    conn = _get_registry_conn()
    if conn is None:
        return []

    try:
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            # Discover the facilities table name from common Registry schemas.
            cur.execute("""
                SELECT table_name FROM information_schema.tables
                WHERE table_schema = 'public'
                  AND (table_name LIKE '%facilit%' OR table_name LIKE '%facility%' OR table_name LIKE '%health%facilit%')
                LIMIT 1
            """)
            row = cur.fetchone()
            if not row:
                return []
            table = row['table_name']

            query = f"SELECT * FROM {table}"
            args = []
            if district_id:
                # Common column names for district filtering.
                query += " WHERE district_id = %s OR district = %s OR admin_unit_id = %s"
                args = [district_id, district_id, district_id]
            query += " ORDER BY 1 LIMIT %s OFFSET %s"
            args += [limit, offset]

            cur.execute(query, args)
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[RegistryIntegration] List facilities failed: %s', exc)
        return []
    finally:
        conn.close()


def get_registry_facility(facility_id):
    """Get a single facility from the Registry database."""
    conn = _get_registry_conn()
    if conn is None:
        return None

    try:
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                SELECT table_name FROM information_schema.tables
                WHERE table_schema = 'public'
                  AND (table_name LIKE '%facilit%' OR table_name LIKE '%facility%' OR table_name LIKE '%health%facilit%')
                LIMIT 1
            """)
            row = cur.fetchone()
            if not row:
                return None
            table = row['table_name']

            cur.execute(f"SELECT * FROM {table} WHERE id = %s OR facility_id = %s OR code = %s LIMIT 1",
                        (facility_id, facility_id, facility_id))
            row = cur.fetchone()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[RegistryIntegration] Get facility failed: %s', exc)
        return None
    finally:
        conn.close()


def get_registry_summary():
    """Return summary counts from the Registry for command centre views."""
    conn = _get_registry_conn()
    if conn is None:
        return {'facilities': 0, 'districts': 0}

    try:
        with conn.cursor() as cur:
            # Facilities
            cur.execute("""
                SELECT COUNT(*) FROM information_schema.tables
                WHERE table_schema = 'public'
                  AND (table_name LIKE '%facilit%' OR table_name LIKE '%facility%' OR table_name LIKE '%health%facilit%')
            """)
            has_facilities = cur.fetchone()[0] > 0

            facility_count = 0
            if has_facilities:
                cur.execute("""
                    SELECT table_name FROM information_schema.tables
                    WHERE table_schema = 'public'
                      AND (table_name LIKE '%facilit%' OR table_name LIKE '%facility%' OR table_name LIKE '%health%facilit%')
                    LIMIT 1
                """)
                table = cur.fetchone()[0]
                cur.execute(f"SELECT COUNT(*) FROM {table}")
                facility_count = cur.fetchone()[0]

            # Districts (admin units at level 2 typically)
            district_count = 0
            cur.execute("""
                SELECT COUNT(*) FROM information_schema.tables
                WHERE table_schema = 'public' AND table_name IN ('adminunits', 'admin_units', 'admin_units_history')
            """)
            has_admin = cur.fetchone()[0] > 0
            if has_admin:
                cur.execute("SELECT COUNT(*) FROM adminunits WHERE level = 2")
                district_count = cur.fetchone()[0]

        return {'facilities': facility_count, 'districts': district_count}
    except Exception as exc:
        logger.warning('[RegistryIntegration] Summary failed: %s', exc)
        return {'facilities': 0, 'districts': 0}
    finally:
        conn.close()