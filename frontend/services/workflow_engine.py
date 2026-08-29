"""
workflow_engine.py — StatGate Workflow Engine
Implements the directive's "Build Workflows, Not Pages" principle.
Objects (projects, reports, research) move through defined workflow
stages with role-based transitions and automatic event publishing.
"""
import os
import json
import uuid
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
        logger.warning('Workflow engine DB connection failed: %s', exc)
        return None


def _ensure_tables(conn):
    with conn.cursor() as cur:
        cur.execute("""
            CREATE TABLE IF NOT EXISTS workflow_templates (
                id              VARCHAR(128) PRIMARY KEY,
                name            TEXT NOT NULL,
                object_type     VARCHAR(64) NOT NULL,
                description     TEXT,
                stages          JSONB NOT NULL,
                initial_stage   VARCHAR(64) NOT NULL,
                tenant_id       VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
                workspace_id    VARCHAR(128),
                created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
        """)
        cur.execute("""
            CREATE TABLE IF NOT EXISTS workflow_instances (
                id              VARCHAR(128) PRIMARY KEY,
                template_id     VARCHAR(128) NOT NULL,
                object_type     VARCHAR(64) NOT NULL,
                object_id       VARCHAR(255) NOT NULL,
                current_stage   VARCHAR(64) NOT NULL,
                tenant_id       VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
                initiated_by    VARCHAR(128),
                completed_at    TIMESTAMPTZ,
                created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                UNIQUE (object_type, object_id)
            )
        """)
        cur.execute("""
            CREATE TABLE IF NOT EXISTS workflow_transitions (
                id              SERIAL PRIMARY KEY,
                instance_id     VARCHAR(128) NOT NULL,
                from_stage      VARCHAR(64) NOT NULL,
                to_stage        VARCHAR(64) NOT NULL,
                action          VARCHAR(64) NOT NULL,
                performed_by    VARCHAR(128),
                comment         TEXT,
                created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
        """)
        cur.execute("ALTER TABLE workflow_instances ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)")
        cur.execute("CREATE INDEX IF NOT EXISTS idx_workflow_instances_scope ON workflow_instances(tenant_id, workspace_id)")
    conn.commit()


# ── Template Management ──

def get_template_for_object(object_type):
    """Return the workflow template for the given object type."""
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("SELECT * FROM workflow_templates WHERE object_type = %s LIMIT 1", (object_type,))
            row = cur.fetchone()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Workflow] Get template failed: %s', exc)
        return None
    finally:
        conn.close()


def list_templates():
    """List all workflow templates."""
    conn = _get_conn()
    if conn is None:
        return []
    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("SELECT * FROM workflow_templates ORDER BY object_type")
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[Workflow] List templates failed: %s', exc)
        return []
    finally:
        conn.close()


# ── Instance Management ──

def start_workflow(object_type, object_id, initiated_by=None, tenant_id='tenant-alpha', workspace_id=None):
    """Start a workflow for an object. Returns the instance or None."""
    template = get_template_for_object(object_type)
    if template is None:
        logger.info('[Workflow] No template for object_type=%s', object_type)
        return None

    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        instance_id = f"wf-inst-{uuid.uuid4().hex[:12]}"
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                INSERT INTO workflow_instances (id, template_id, object_type, object_id, current_stage, tenant_id, workspace_id, initiated_by)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (object_type, object_id) DO NOTHING
                RETURNING *
            """, (instance_id, template['id'], object_type, object_id,
                  template['initial_stage'], tenant_id, workspace_id, initiated_by))
            row = cur.fetchone()
        conn.commit()
        return dict(row) if row else None
    except Exception as exc:
        logger.warning('[Workflow] Start failed: %s', exc)
        return None
    finally:
        conn.close()


def get_workflow_instance(object_type, object_id, tenant_id=None, workspace_id=None):
    """Get the workflow instance for an object."""
    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            query = "SELECT * FROM workflow_instances WHERE object_type = %s AND object_id = %s"
            args = [object_type, object_id]
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
        logger.warning('[Workflow] Get instance failed: %s', exc)
        return None
    finally:
        conn.close()


def get_available_actions(object_type, object_id, user_role='analyst', tenant_id=None, workspace_id=None):
    """
    Return the actions available for the current stage of the object's workflow,
    filtered by the user's role.
    """
    instance = get_workflow_instance(object_type, object_id, tenant_id=tenant_id, workspace_id=workspace_id)
    if instance is None:
        return []

    template = get_template_for_object(object_type)
    if template is None:
        return []

    stages = template.get('stages', [])
    if isinstance(stages, str):
        stages = json.loads(stages)

    current_stage = instance['current_stage']
    for stage in stages:
        if stage.get('name') == current_stage:
            roles = stage.get('roles', [])
            if user_role in roles or 'admin' == user_role:
                return stage.get('actions', [])
            return []
    return []


def transition(object_type, object_id, action, performed_by=None, comment='', tenant_id='tenant-alpha', workspace_id=None):
    """
    Execute a workflow transition. Returns the updated instance or None.
    Also publishes a workflow.transition event.
    """
    instance = get_workflow_instance(object_type, object_id, tenant_id=tenant_id, workspace_id=workspace_id)
    if instance is None:
        return None

    template = get_template_for_object(object_type)
    if template is None:
        return None

    stages = template.get('stages', [])
    if isinstance(stages, str):
        stages = json.loads(stages)

    current_stage = instance['current_stage']
    to_stage = None
    for stage in stages:
        if stage.get('name') == current_stage:
            for act in stage.get('actions', []):
                if act.get('action') == action:
                    to_stage = act.get('to')
                    break
            break

    if to_stage is None:
        logger.warning('[Workflow] Action %s not valid for stage %s', action, current_stage)
        return None

    conn = _get_conn()
    if conn is None:
        return None
    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            # Log the transition
            cur.execute("""
                INSERT INTO workflow_transitions (instance_id, from_stage, to_stage, action, performed_by, comment)
                VALUES (%s, %s, %s, %s, %s, %s)
            """, (instance['id'], current_stage, to_stage, action, performed_by, comment))

            # Update the instance
            completed = "NULL"
            if to_stage in ('archived', 'completed'):
                completed = "CURRENT_TIMESTAMP"
            cur.execute(f"""
                UPDATE workflow_instances
                SET current_stage = %s, updated_at = CURRENT_TIMESTAMP, completed_at = {completed}
                WHERE id = %s
                RETURNING *
            """, (to_stage, instance['id']))
            row = cur.fetchone()
        conn.commit()

        result = dict(row) if row else None

        # Publish event so other modules (StatChat notifications, dashboard updates) can react.
        if result:
            try:
                from services.event_bus import publish_event
                publish_event(
                    'workflow.transition',
                    object_type,
                    object_id,
                    tenant_id=tenant_id,
                    payload={
                        'from_stage': current_stage,
                        'to_stage': to_stage,
                        'action': action,
                        'performed_by': performed_by,
                        'comment': comment,
                    },
                )
            except Exception:
                pass

        return result
    except Exception as exc:
        logger.warning('[Workflow] Transition failed: %s', exc)
        return None
    finally:
        conn.close()


def get_transition_history(object_type, object_id, tenant_id=None, workspace_id=None):
    """Return the full transition history for an object's workflow."""
    instance = get_workflow_instance(object_type, object_id, tenant_id=tenant_id, workspace_id=workspace_id)
    if instance is None:
        return []
    conn = _get_conn()
    if conn is None:
        return []
    try:
        _ensure_tables(conn)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                SELECT * FROM workflow_transitions WHERE instance_id = %s ORDER BY created_at ASC
            """, (instance['id'],))
            rows = cur.fetchall()
        return [dict(r) for r in rows]
    except Exception as exc:
        logger.warning('[Workflow] History failed: %s', exc)
        return []
    finally:
        conn.close()
