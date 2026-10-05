-- Durable GSBPM process outputs for weighting and subsequent official tabulation.
CREATE TABLE IF NOT EXISTS processing_runs (
  id TEXT PRIMARY KEY,
  form_id TEXT NOT NULL,
  survey_id TEXT,
  method TEXT NOT NULL,
  weight_variable TEXT,
  configuration JSONB NOT NULL DEFAULT '{}',
  input_count INT NOT NULL DEFAULT 0,
  approved_count INT NOT NULL DEFAULT 0,
  output_count INT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'completed',
  error_message TEXT,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  created_by TEXT NOT NULL DEFAULT 'admin',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS processing_run_records (
  id BIGSERIAL PRIMARY KEY,
  processing_run_id TEXT NOT NULL REFERENCES processing_runs(id) ON DELETE CASCADE,
  submission_instance_id TEXT NOT NULL,
  form_id TEXT NOT NULL,
  values JSONB NOT NULL DEFAULT '{}',
  weight DOUBLE PRECISION NOT NULL DEFAULT 1.0,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (processing_run_id, submission_instance_id)
);

CREATE INDEX IF NOT EXISTS idx_processing_runs_scope
  ON processing_runs(tenant_id, workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_processing_runs_form
  ON processing_runs(tenant_id, workspace_id, form_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_processing_run_records_scope
  ON processing_run_records(tenant_id, workspace_id, processing_run_id);
