"""
platform_objects.py — StatGate Platform Object Service
Provides CRUD operations for Projects, Reports, and Research Studies.
These are first-class platform objects that can be linked, discussed,
and workflowed — implementing the directive's "Every Object Should
Be Connected" principle.
"""
import os
import json
import uuid
import logging
import datetime

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
        logger.warning('Platform objects DB connection failed: %s', exc)
        return None


def _ensure_tables(conn):
    with conn.cursor() as cur:
        cur.execute("""
            CREATE TABLE IF NOT EXISTS projects (
                id              VARCHAR(128) PRIMARY KEY,
                title           TEXT NOT NULL,
                description     TEXT,
                status          VARCHAR(32) NOT NULL DEFAULT 'planning',
                owner_id        VARCHAR(128),
                tenant_id       VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
                workspace_id    VARCHAR(128),
                collaborators   JSONB NOT NULL DEFAULT '[]',
                tags            JSONB NOT NULL DEFAULT '[]',
                start_date      DATE,
                end_date        DATE,
                created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
        """)
        cur.execute("""
            CREATE TABLE IF NOT EXISTS reports (
                id              VARCHAR(128) PRIMARY KEY,
                title           TEXT NOT NULL,
                content         TEXT,
                summary         TEXT,
                status          VARCHAR(32) NOT NULL DEFAULT 'draft',
                author_id       VARCHAR(128),
                reviewer_id     VARCHAR(128),
                tenant_id       VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
                workspace_id    VARCHAR(128),
                project_id      VARCHAR(128),
                version_tag     VARCHAR(32) NOT NULL DEFAULT '1.0.0',
                tags            JSONB NOT NULL DEFAULT '[]',
                approved_at     TIMESTAMPTZ,
                published_at    TIMESTAMPTZ,
                created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
        """)
        cur.execute("""
            CREATE TABLE IF NOT EXISTS research_studies (
                id                  VARCHAR(128) PRIMARY KEY,
                title               TEXT NOT NULL,
                protocol            TEXT,
                description         TEXT,
                status              VARCHAR(32) NOT NULL DEFAULT 'proposed',
                principal_investigator VARCHAR(128),
                tenant_id           VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
                workspace_id        VARCHAR(128),
                project_id          VARCHAR(128),
                ethics_status       VARCHAR(32) NOT NULL DEFAULT 'pending',
                data_collection_status VARCHAR(32) NOT NULL DEFAULT 'not_started',
                collaborators       JSONB NOT NULL DEFAULT '[]',
                tags                JSONB NOT NULL DEFAULT '[]',
                start_date          DATE,
                end_date            DATE,
                created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
        """)
        cur.execute("ALTER TABLE projects ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)")
        cur.execute("ALTER TABLE reports ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)")
        cur.execute("ALTER TABLE research_studies ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)")
        cur.execute("CREATE INDEX IF NOT EXISTS idx_projects_tenant_workspace ON projects(tenant_id, workspace_id)")
        cur.execute("CREATE INDEX IF NOT EXISTS idx_reports_tenant_workspace ON reports(tenant_id, workspace_id)")
        cur.execute("CREATE INDEX IF NOT EXISTS idx_research_tenant_workspace ON research_studies(tenant_id, workspace_id)")
    conn.commit()


# ══════════════════════════════════════════════════════════════
#  PROJECTS
# ══════════════════════════════════════════════════════════════

def create_project(title, description='', owner_id=None, tenant_id='tenant-alpha',
                   collaborators=None, tags=None, start_date=None, end_date=None, workspace_id=None):
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        project_id = f"project-{uuid.uuid4().hex[:12]}"
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                INSERT INTO projects (id, title, description, owner_id, tenant_id, workspace_id, collaborators, tags, start_date, end_date)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                RETURNING *
            """, (project_id, title, description, owner_id, tenant_id, workspace_id,
                  json.dumps(collaborators or []), json.dumps(tags or []),
                  start_date, end_date))
            row = cur.fetchone()
        conn.commit()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Projects] Create failed: %s', exc)
        return None
    finally:
        conn.close()


def get_project(project_id, tenant_id=None, workspace_id=None):
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            query = "SELECT * FROM projects WHERE id = %s"
            args = [project_id]
            if tenant_id is not None:
                query += " AND tenant_id = %s"
                args.append(tenant_id)
            if workspace_id:
                query += " AND workspace_id = %s"
                args.append(workspace_id)
            cur.execute(query, args)
            row = cur.fetchone()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Projects] Get failed: %s', exc)
        return None
    finally:
        conn.close()


def list_projects(tenant_id='tenant-alpha', status=None, workspace_id=None):
    conn = _get_conn()
    if conn is None:
        return []
    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            query = "SELECT * FROM projects WHERE tenant_id = %s"
            args = [tenant_id]
            if workspace_id:
                query += " AND workspace_id = %s"
                args.append(workspace_id)
            if status:
                query += " AND status = %s"
                args.append(status)
            query += " ORDER BY updated_at DESC"
            cur.execute(query, args)
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[Projects] List failed: %s', exc)
        return []
    finally:
        conn.close()


def update_project(project_id, tenant_id=None, workspace_id=None, **fields):
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        allowed = {'title', 'description', 'status', 'owner_id', 'collaborators', 'tags', 'start_date', 'end_date'}
        sets = []
        args = []
        for k, v in fields.items():
            if k in allowed:
                if k in ('collaborators', 'tags'):
                    v = json.dumps(v)
                sets.append(f"{k} = %s")
                args.append(v)
        if not sets:
            return get_project(project_id, tenant_id=tenant_id, workspace_id=workspace_id)
        sets.append("updated_at = CURRENT_TIMESTAMP")
        where = "id = %s"
        where_args = [project_id]
        if tenant_id is not None:
            where += " AND tenant_id = %s"
            where_args.append(tenant_id)
        if workspace_id:
            where += " AND workspace_id = %s"
            where_args.append(workspace_id)
        args.extend(where_args)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(f"UPDATE projects SET {', '.join(sets)} WHERE {where} RETURNING *", args)
            row = cur.fetchone()
        conn.commit()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Projects] Update failed: %s', exc)
        return None
    finally:
        conn.close()


# ══════════════════════════════════════════════════════════════
#  REPORTS
# ══════════════════════════════════════════════════════════════

def create_report(title, content='', summary='', author_id=None, tenant_id='tenant-alpha',
                  project_id=None, tags=None, workspace_id=None):
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        report_id = f"report-{uuid.uuid4().hex[:12]}"
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                INSERT INTO reports (id, title, content, summary, author_id, tenant_id, workspace_id, project_id, tags)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                RETURNING *
            """, (report_id, title, content, summary, author_id, tenant_id, workspace_id, project_id,
                  json.dumps(tags or [])))
            row = cur.fetchone()
        conn.commit()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Reports] Create failed: %s', exc)
        return None
    finally:
        conn.close()


def get_report(report_id, tenant_id=None, workspace_id=None):
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            query = "SELECT * FROM reports WHERE id = %s"
            args = [report_id]
            if tenant_id is not None:
                query += " AND tenant_id = %s"
                args.append(tenant_id)
            if workspace_id:
                query += " AND workspace_id = %s"
                args.append(workspace_id)
            cur.execute(query, args)
            row = cur.fetchone()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Reports] Get failed: %s', exc)
        return None
    finally:
        conn.close()


def list_reports(tenant_id='tenant-alpha', status=None, project_id=None, workspace_id=None):
    conn = _get_conn()
    if conn is None:
        return []
    try:
        _ensure_tables(conn)
        query = "SELECT * FROM reports WHERE tenant_id = %s"
        args = [tenant_id]
        if workspace_id:
            query += " AND workspace_id = %s"
            args.append(workspace_id)
        if status:
            query += " AND status = %s"
            args.append(status)
        if project_id:
            query += " AND project_id = %s"
            args.append(project_id)
        query += " ORDER BY updated_at DESC"
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(query, args)
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[Reports] List failed: %s', exc)
        return []
    finally:
        conn.close()


def update_report(report_id, tenant_id=None, workspace_id=None, **fields):
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        allowed = {'title', 'content', 'summary', 'status', 'reviewer_id', 'project_id', 'version_tag', 'tags', 'approved_at', 'published_at'}
        sets = []
        args = []
        for k, v in fields.items():
            if k in allowed:
                if k == 'tags':
                    v = json.dumps(v)
                sets.append(f"{k} = %s")
                args.append(v)
        if not sets:
            return get_report(report_id, tenant_id=tenant_id, workspace_id=workspace_id)
        sets.append("updated_at = CURRENT_TIMESTAMP")
        where = "id = %s"
        where_args = [report_id]
        if tenant_id is not None:
            where += " AND tenant_id = %s"
            where_args.append(tenant_id)
        if workspace_id:
            where += " AND workspace_id = %s"
            where_args.append(workspace_id)
        args.extend(where_args)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(f"UPDATE reports SET {', '.join(sets)} WHERE {where} RETURNING *", args)
            row = cur.fetchone()
        conn.commit()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Reports] Update failed: %s', exc)
        return None
    finally:
        conn.close()


# ══════════════════════════════════════════════════════════════
#  RESEARCH STUDIES
# ══════════════════════════════════════════════════════════════

def create_research(title, description='', protocol='', principal_investigator=None,
                    tenant_id='tenant-alpha', project_id=None, collaborators=None,
                    tags=None, start_date=None, end_date=None, workspace_id=None):
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        research_id = f"research-{uuid.uuid4().hex[:12]}"
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                INSERT INTO research_studies (id, title, description, protocol, principal_investigator,
                    tenant_id, workspace_id, project_id, collaborators, tags, start_date, end_date)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                RETURNING *
            """, (research_id, title, description, protocol, principal_investigator,
                  tenant_id, workspace_id, project_id, json.dumps(collaborators or []),
                  json.dumps(tags or []), start_date, end_date))
            row = cur.fetchone()
        conn.commit()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Research] Create failed: %s', exc)
        return None
    finally:
        conn.close()


def get_research(research_id, tenant_id=None, workspace_id=None):
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            query = "SELECT * FROM research_studies WHERE id = %s"
            args = [research_id]
            if tenant_id is not None:
                query += " AND tenant_id = %s"
                args.append(tenant_id)
            if workspace_id:
                query += " AND workspace_id = %s"
                args.append(workspace_id)
            cur.execute(query, args)
            row = cur.fetchone()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Research] Get failed: %s', exc)
        return None
    finally:
        conn.close()


def list_research(tenant_id='tenant-alpha', status=None, workspace_id=None):
    conn = _get_conn()
    if conn is None:
        return []
    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            query = "SELECT * FROM research_studies WHERE tenant_id = %s"
            args = [tenant_id]
            if workspace_id:
                query += " AND workspace_id = %s"
                args.append(workspace_id)
            if status:
                query += " AND status = %s"
                args.append(status)
            query += " ORDER BY updated_at DESC"
            cur.execute(query, args)
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[Research] List failed: %s', exc)
        return []
    finally:
        conn.close()


def update_research(research_id, tenant_id=None, workspace_id=None, **fields):
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        allowed = {'title', 'description', 'protocol', 'status', 'principal_investigator',
                   'project_id', 'ethics_status', 'data_collection_status',
                   'collaborators', 'tags', 'start_date', 'end_date'}
        sets = []
        args = []
        for k, v in fields.items():
            if k in allowed:
                if k in ('collaborators', 'tags'):
                    v = json.dumps(v)
                sets.append(f"{k} = %s")
                args.append(v)
        if not sets:
            return get_research(research_id, tenant_id=tenant_id, workspace_id=workspace_id)
        sets.append("updated_at = CURRENT_TIMESTAMP")
        where = "id = %s"
        where_args = [research_id]
        if tenant_id is not None:
            where += " AND tenant_id = %s"
            where_args.append(tenant_id)
        if workspace_id:
            where += " AND workspace_id = %s"
            where_args.append(workspace_id)
        args.extend(where_args)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(f"UPDATE research_studies SET {', '.join(sets)} WHERE {where} RETURNING *", args)
            row = cur.fetchone()
        conn.commit()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Research] Update failed: %s', exc)
        return None
    finally:
        conn.close()
