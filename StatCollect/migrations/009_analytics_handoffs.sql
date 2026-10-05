-- Durable handoff status for published official-statistics observations.
CREATE TABLE IF NOT EXISTS analytics_handoffs (
  id BIGSERIAL PRIMARY KEY,
  observation_id TEXT NOT NULL,
  target_module TEXT NOT NULL,
  target_asset_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  http_status INT NOT NULL DEFAULT 0,
  attempts INT NOT NULL DEFAULT 0,
  error_message TEXT,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  workspace_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (observation_id, target_module, tenant_id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_analytics_handoffs_scope
  ON analytics_handoffs(tenant_id, workspace_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_handoffs_observation
  ON analytics_handoffs(observation_id, target_module);
