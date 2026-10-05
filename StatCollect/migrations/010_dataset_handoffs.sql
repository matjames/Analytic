-- Durable dataset lineage handoffs from StatCollect into Analytics Core.
CREATE TABLE IF NOT EXISTS analytics_dataset_handoffs (
  id BIGSERIAL PRIMARY KEY,
  dataset_id TEXT NOT NULL,
  form_id TEXT NOT NULL,
  processing_run_id TEXT NOT NULL,
  target_asset_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  http_status INT NOT NULL DEFAULT 0,
  attempts INT NOT NULL DEFAULT 0,
  record_count INT NOT NULL DEFAULT 0,
  schedule_id TEXT,
  schedule_status TEXT,
  error_message TEXT,
  lineage JSONB NOT NULL DEFAULT '{}',
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (dataset_id, processing_run_id, tenant_id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_analytics_dataset_handoffs_scope
  ON analytics_dataset_handoffs(tenant_id, workspace_id, updated_at DESC);
