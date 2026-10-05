-- Phase 6 questionnaire/template ownership for workspace-aware production.
ALTER TABLE templates ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE template_versions ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE template_versions ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';

CREATE INDEX IF NOT EXISTS idx_templates_tenant_workspace
  ON templates(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_template_versions_tenant_workspace
  ON template_versions(tenant_id, workspace_id);
