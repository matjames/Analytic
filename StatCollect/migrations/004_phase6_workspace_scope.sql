-- Add workspace ownership to Phase 6 resources without recreating the
-- persistent StatCollect volume. Existing seeded rows remain in the default
-- workspace and are still tenant-scoped.

ALTER TABLE question_bank ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE survey_projects ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE enumeration_areas ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE sample_frames ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE field_supervisors ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE enumerators ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE assignment_plans ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE supervisor_reviews ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE supervisor_reviews ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE census_rounds ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE census_pes ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE census_pes ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE sdg_indicators ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE statistical_publications ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE tabulation_outputs ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE tabulation_outputs ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'default';

CREATE INDEX IF NOT EXISTS idx_question_bank_scope ON question_bank(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_survey_projects_scope ON survey_projects(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_enumeration_areas_scope ON enumeration_areas(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_sample_frames_scope ON sample_frames(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_supervisors_scope ON field_supervisors(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_enumerators_scope ON enumerators(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_assignment_plans_scope ON assignment_plans(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_supervisor_reviews_scope ON supervisor_reviews(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_census_rounds_scope ON census_rounds(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_census_pes_scope ON census_pes(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_sdg_indicators_scope ON sdg_indicators(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_statistical_publications_scope ON statistical_publications(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_tabulation_outputs_scope ON tabulation_outputs(tenant_id, workspace_id);
