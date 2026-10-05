-- Migration 000002: real dataset record storage (data plane)
-- Datasets hold their actual rows here as JSONB; pipelines read from and
-- write to this table, and catalog row counts are derived from it.
CREATE TABLE IF NOT EXISTS statdata.dataset_records (
    id BIGSERIAL PRIMARY KEY,
    dataset_id VARCHAR(64) NOT NULL,
    record JSONB NOT NULL,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dataset_records_dataset ON statdata.dataset_records (dataset_id, id);
