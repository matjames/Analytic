-- ══════════════════════════════════════════════════════════════
-- STATGATE WORKFLOW ENGINE
-- ══════════════════════════════════════════════════════════════
-- Implements the directive's "Build Workflows, Not Pages" principle.
-- Objects (projects, reports, research) move through defined workflow
-- stages with role-based transitions and automatic notifications.
-- ══════════════════════════════════════════════════════════════

-- ── Workflow Templates ───────────────────────────────────────
-- Defines the stages and allowed transitions for each object type.
CREATE TABLE IF NOT EXISTS workflow_templates (
    id              VARCHAR(128) PRIMARY KEY,
    name            TEXT NOT NULL,
    object_type     VARCHAR(64) NOT NULL,   -- project | report | research
    description     TEXT,
    stages          JSONB NOT NULL,          -- [{name, label, roles, actions}]
    initial_stage   VARCHAR(64) NOT NULL,
    tenant_id       VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_workflow_templates_object ON workflow_templates(object_type);

-- ── Workflow Instances ───────────────────────────────────────
-- Tracks the current stage of each object in its workflow.
CREATE TABLE IF NOT EXISTS workflow_instances (
    id              VARCHAR(128) PRIMARY KEY,
    template_id     VARCHAR(128) NOT NULL REFERENCES workflow_templates(id),
    object_type     VARCHAR(64) NOT NULL,
    object_id       VARCHAR(255) NOT NULL,
    current_stage   VARCHAR(64) NOT NULL,
    tenant_id       VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
    initiated_by    VARCHAR(128),
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (object_type, object_id)
);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_object ON workflow_instances(object_type, object_id);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_stage ON workflow_instances(current_stage);

-- ── Workflow Transition Log ──────────────────────────────────
-- Audit trail of every stage transition.
CREATE TABLE IF NOT EXISTS workflow_transitions (
    id              SERIAL PRIMARY KEY,
    instance_id     VARCHAR(128) NOT NULL REFERENCES workflow_instances(id),
    from_stage      VARCHAR(64) NOT NULL,
    to_stage        VARCHAR(64) NOT NULL,
    action          VARCHAR(64) NOT NULL,
    performed_by    VARCHAR(128),
    comment         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_workflow_transitions_instance ON workflow_transitions(instance_id);

-- ── Seed Default Workflow Templates ──────────────────────────
INSERT INTO workflow_templates (id, name, object_type, description, stages, initial_stage) VALUES
(
    'wf-project-lifecycle',
    'Research Project Lifecycle',
    'project',
    'Create a project → Invite collaborators → Collect data → Discuss findings → Analyse data → Generate reports → Submit for approval → Publish findings → Present results → Archive the project',
    '[
        {"name": "planning", "label": "Planning", "roles": ["admin","analyst","manager"], "actions": [{"action": "start", "to": "active"}]},
        {"name": "active", "label": "Active — Data Collection", "roles": ["admin","analyst","manager"], "actions": [{"action": "review", "to": "review"}]},
        {"name": "review", "label": "Review & Analysis", "roles": ["admin","analyst","manager"], "actions": [{"action": "approve", "to": "completed"}, {"action": "reject", "to": "active"}]},
        {"name": "completed", "label": "Completed", "roles": ["admin","manager"], "actions": [{"action": "archive", "to": "archived"}]},
        {"name": "archived", "label": "Archived", "roles": ["admin"], "actions": []}
    ]'::jsonb,
    'planning'
),

(
    'wf-report-approval',
    'Report Approval Process',
    'report',
    'Draft → Submit for approval → Review → Approve/Reject → Publish → Archive',
    '[
        {"name": "draft", "label": "Draft", "roles": ["admin","analyst"], "actions": [{"action": "submit", "to": "submitted"}]},
        {"name": "submitted", "label": "Submitted for Approval", "roles": ["admin","analyst"], "actions": [{"action": "approve", "to": "approved"}, {"action": "reject", "to": "draft"}]},
        {"name": "approved", "label": "Approved", "roles": ["admin","manager"], "actions": [{"action": "publish", "to": "published"}]},
        {"name": "published", "label": "Published", "roles": ["admin","manager"], "actions": [{"action": "archive", "to": "archived"}]},
        {"name": "archived", "label": "Archived", "roles": ["admin"], "actions": []}
    ]'::jsonb,
    'draft'
),

(
    'wf-research-lifecycle',
    'Research Study Lifecycle',
    'research',
    'Proposed → Ethics Review → Approved → Active → Analysis → Published → Archived',
    '[
        {"name": "proposed", "label": "Proposed", "roles": ["admin","analyst","researcher"], "actions": [{"action": "submit_ethics", "to": "ethics_review"}]},
        {"name": "ethics_review", "label": "Ethics Review", "roles": ["admin","manager"], "actions": [{"action": "approve_ethics", "to": "approved"}, {"action": "reject_ethics", "to": "proposed"}]},
        {"name": "approved", "label": "Approved", "roles": ["admin","researcher"], "actions": [{"action": "start", "to": "active"}]},
        {"name": "active", "label": "Active — Data Collection", "roles": ["admin","researcher"], "actions": [{"action": "analyse", "to": "analysis"}]},
        {"name": "analysis", "label": "Analysis", "roles": ["admin","analyst","researcher"], "actions": [{"action": "publish", "to": "published"}]},
        {"name": "published", "label": "Published", "roles": ["admin","manager"], "actions": [{"action": "archive", "to": "archived"}]},
        {"name": "archived", "label": "Archived", "roles": ["admin"], "actions": []}
    ]'::jsonb,
    'proposed'
) ON CONFLICT (id) DO NOTHING;
