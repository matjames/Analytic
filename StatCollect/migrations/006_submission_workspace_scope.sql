-- Phase 6 field submissions must follow the same tenant/workspace boundary
-- as templates, surveys, and official-statistics outputs.
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';

CREATE INDEX IF NOT EXISTS idx_submissions_tenant_workspace
  ON submissions(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_submissions_form_scope
  ON submissions(tenant_id, workspace_id, form_id);
CREATE INDEX IF NOT EXISTS idx_attachments_submission_scope
  ON attachments(tenant_id, workspace_id, submission_instance_id);
