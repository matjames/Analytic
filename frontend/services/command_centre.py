"""
command_centre.py — StatGate Command Centre Service
Implements the directive's requirement that dashboards must answer
four questions:
  1. What is happening?
  2. What requires my attention?
  3. What decisions should I make?
  4. What should I do next?

Aggregates data from across the platform (datasets, alerts, workflows,
projects, reports, research, tasks) into a single actionable view.
"""
import os
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
            password=os.getenv('KAGGLE_DB_PASSWORD', 'REDACTED_PLACEHOLDER'),
            sslmode=os.getenv('KAGGLE_DB_SSLMODE', 'disable'),
        )
    except Exception as exc:
        logger.warning('Command centre DB connection failed: %s', exc)
        return None


def _ensure_tables(conn):
    """Ensure all platform tables exist (idempotent)."""
    with conn.cursor() as cur:
        for ddl in [
            """
            CREATE TABLE IF NOT EXISTS projects (
                id VARCHAR(128) PRIMARY KEY, title TEXT NOT NULL, description TEXT,
                status VARCHAR(32) NOT NULL DEFAULT 'planning', owner_id VARCHAR(128),
                tenant_id VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
                collaborators JSONB NOT NULL DEFAULT '[]', tags JSONB NOT NULL DEFAULT '[]',
                start_date DATE, end_date DATE,
                created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
            """,
            """
            CREATE TABLE IF NOT EXISTS reports (
                id VARCHAR(128) PRIMARY KEY, title TEXT NOT NULL, content TEXT, summary TEXT,
                status VARCHAR(32) NOT NULL DEFAULT 'draft', author_id VARCHAR(128),
                reviewer_id VARCHAR(128), tenant_id VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
                project_id VARCHAR(128), version_tag VARCHAR(32) NOT NULL DEFAULT '1.0.0',
                tags JSONB NOT NULL DEFAULT '[]', approved_at TIMESTAMPTZ, published_at TIMESTAMPTZ,
                created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
            """,
            """
            CREATE TABLE IF NOT EXISTS research_studies (
                id VARCHAR(128) PRIMARY KEY, title TEXT NOT NULL, protocol TEXT, description TEXT,
                status VARCHAR(32) NOT NULL DEFAULT 'proposed',
                principal_investigator VARCHAR(128),
                tenant_id VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
                project_id VARCHAR(128), ethics_status VARCHAR(32) NOT NULL DEFAULT 'pending',
                data_collection_status VARCHAR(32) NOT NULL DEFAULT 'not_started',
                collaborators JSONB NOT NULL DEFAULT '[]', tags JSONB NOT NULL DEFAULT '[]',
                start_date DATE, end_date DATE,
                created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
            """,
            """
            CREATE TABLE IF NOT EXISTS workflow_instances (
                id VARCHAR(128) PRIMARY KEY, template_id VARCHAR(128) NOT NULL,
                object_type VARCHAR(64) NOT NULL, object_id VARCHAR(255) NOT NULL,
                current_stage VARCHAR(64) NOT NULL,
                tenant_id VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
                initiated_by VARCHAR(128), completed_at TIMESTAMPTZ,
                created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                UNIQUE (object_type, object_id)
            )
            """,
            """
            CREATE TABLE IF NOT EXISTS dataset_metadata (
                table_name VARCHAR(128) PRIMARY KEY, owner_id VARCHAR(128), steward_id VARCHAR(128),
                description TEXT, classification VARCHAR(32) DEFAULT 'public',
                tags JSONB NOT NULL DEFAULT '[]', row_count BIGINT DEFAULT 0, col_count INT DEFAULT 0,
                latest_snapshot TIMESTAMPTZ,
                created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
            """,
        ]:
            cur.execute(ddl)
    conn.commit()


def get_command_centre(tenant_id='tenant-alpha'):
    """
    Build the full command centre payload answering the four questions:
      1. What is happening?      — live status of datasets, projects, reports, research
      2. What requires my attention? — pending approvals, active workflows, critical alerts
      3. What decisions should I make? — workflow actions available, agent scenarios
      4. What should I do next?   — recommended next actions
    """
    conn = _get_conn()
    if conn is None:
        return _empty_command_centre(tenant_id)

    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            # ── 1. What is happening? ──────────────────────────
            cur.execute("SELECT COUNT(*) AS c FROM projects WHERE tenant_id = %s", (tenant_id,))
            project_count = cur.fetchone()['c']
            cur.execute("SELECT COUNT(*) AS c FROM reports WHERE tenant_id = %s", (tenant_id,))
            report_count = cur.fetchone()['c']
            cur.execute("SELECT COUNT(*) AS c FROM research_studies WHERE tenant_id = %s", (tenant_id,))
            research_count = cur.fetchone()['c']
            cur.execute("SELECT COUNT(*) AS c FROM dataset_metadata")
            dataset_count = cur.fetchone()['c']

            # ── 2. What requires my attention? ─────────────────
            # Pending approvals: reports in 'submitted' status
            cur.execute("""
                SELECT id, title, status, updated_at FROM reports
                WHERE tenant_id = %s AND status = 'submitted' ORDER BY updated_at DESC
            """, (tenant_id,))
            pending_approvals = [dict(r) for r in cur.fetchall()]

            # Active workflows: instances not in terminal stages
            cur.execute("""
                SELECT object_type, object_id, current_stage, updated_at
                FROM workflow_instances
                WHERE tenant_id = %s AND current_stage NOT IN ('archived', 'completed', 'published')
                ORDER BY updated_at DESC
            """, (tenant_id,))
            active_workflows = [dict(r) for r in cur.fetchall()]

            # Research awaiting ethics review
            cur.execute("""
                SELECT id, title, ethics_status, status FROM research_studies
                WHERE tenant_id = %s AND ethics_status = 'pending' ORDER BY updated_at DESC
            """, (tenant_id,))
            ethics_reviews = [dict(r) for r in cur.fetchall()]

            # ── 3. What decisions should I make? ───────────────
            # Projects in review stage
            cur.execute("""
                SELECT id, title, status FROM projects
                WHERE tenant_id = %s AND status = 'review' ORDER BY updated_at DESC
            """, (tenant_id,))
            review_projects = [dict(r) for r in cur.fetchall()]

            # ── 4. What should I do next? ──────────────────────
            # Recommended next actions based on workflow state
            next_actions = []
            for wf in active_workflows:
                next_actions.append({
                    'object_type': wf['object_type'],
                    'object_id': wf['object_id'],
                    'current_stage': wf['current_stage'],
                    'action': f"Advance {wf['object_type']} '{wf['object_id']}' from {wf['current_stage']}",
                })
            for r in pending_approvals:
                next_actions.append({
                    'object_type': 'report',
                    'object_id': r['id'],
                    'current_stage': 'submitted',
                    'action': f"Review and approve report '{r['title']}'",
                })
            for r in ethics_reviews:
                next_actions.append({
                    'object_type': 'research',
                    'object_id': r['id'],
                    'current_stage': 'ethics_review',
                    'action': f"Review ethics status for research '{r['title']}'",
                })

        return {
            'tenant_id': tenant_id,
            'generated_at': datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
            # 1. What is happening?
            'what_is_happening': {
                'datasets': dataset_count,
                'projects': project_count,
                'reports': report_count,
                'research_studies': research_count,
            },
            # 2. What requires my attention?
            'requires_attention': {
                'pending_approvals': pending_approvals,
                'active_workflows': active_workflows,
                'ethics_reviews': ethics_reviews,
                'attention_count': len(pending_approvals) + len(active_workflows) + len(ethics_reviews),
            },
            # 3. What decisions should I make?
            'decisions': {
                'review_projects': review_projects,
                'decision_count': len(review_projects),
            },
            # 4. What should I do next?
            'next_actions': next_actions,
            'next_action_count': len(next_actions),
        }
    except Exception as exc:
        logger.warning('[CommandCentre] Build failed: %s', exc)
        return _empty_command_centre(tenant_id)
    finally:
        conn.close()


def _empty_command_centre(tenant_id):
    return {
        'tenant_id': tenant_id,
        'generated_at': datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
        'what_is_happening': {'datasets': 0, 'projects': 0, 'reports': 0, 'research_studies': 0},
        'requires_attention': {'pending_approvals': [], 'active_workflows': [], 'ethics_reviews': [], 'attention_count': 0},
        'decisions': {'review_projects': [], 'decision_count': 0},
        'next_actions': [],
        'next_action_count': 0,
        'note': 'Command centre data unavailable (DB not reachable).',
    }