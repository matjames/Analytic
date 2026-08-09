-- ══════════════════════════════════════════════════════════════
-- STATGATE DATASET METADATA & SEARCH
-- ══════════════════════════════════════════════════════════════
-- Implements the directive's requirement that datasets must be
-- discoverable, searchable, and carry rich business metadata.
-- ══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS dataset_metadata (
    table_name      VARCHAR(128) PRIMARY KEY,
    owner_id        VARCHAR(128),
    steward_id      VARCHAR(128),
    description     TEXT,
    classification  VARCHAR(32) DEFAULT 'public',
    tags            JSONB NOT NULL DEFAULT '[]',
    row_count       BIGINT DEFAULT 0,
    col_count       INT DEFAULT 0,
    latest_snapshot TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dataset_metadata_class ON dataset_metadata(classification);