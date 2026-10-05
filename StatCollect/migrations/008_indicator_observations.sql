-- Versioned, publishable SDG observations with source-run provenance.
CREATE TABLE IF NOT EXISTS indicator_observations (
  id TEXT PRIMARY KEY,
  indicator_code TEXT NOT NULL,
  period TEXT NOT NULL,
  geography_code TEXT NOT NULL DEFAULT 'UGA',
  value DOUBLE PRECISION NOT NULL,
  unit TEXT NOT NULL,
  version INT NOT NULL DEFAULT 1,
  is_current BOOLEAN NOT NULL DEFAULT true,
  status TEXT NOT NULL DEFAULT 'published',
  source_form_id TEXT NOT NULL,
  processing_run_id TEXT NOT NULL,
  source_count INT NOT NULL DEFAULT 0,
  methodology JSONB NOT NULL DEFAULT '{}',
  published_by TEXT NOT NULL DEFAULT 'admin',
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_indicator_observations_scope
  ON indicator_observations(tenant_id, workspace_id, indicator_code, period, geography_code);
CREATE INDEX IF NOT EXISTS idx_indicator_observations_current
  ON indicator_observations(tenant_id, workspace_id, is_current, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_indicator_observations_run
  ON indicator_observations(tenant_id, workspace_id, processing_run_id);
