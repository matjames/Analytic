-- Additive workspace boundary for every persisted StatFederation domain.
-- NULL preserves legacy, tenant-wide records until they are assigned explicitly.
SET search_path TO statfederation, public;

CREATE TABLE IF NOT EXISTS object_links (
    id            SERIAL PRIMARY KEY,
    source_type   VARCHAR(64)  NOT NULL,
    source_id     VARCHAR(255) NOT NULL,
    target_type   VARCHAR(64)  NOT NULL,
    target_id     VARCHAR(255) NOT NULL,
    relationship  VARCHAR(64)  NOT NULL DEFAULT 'related',
    tenant_id     VARCHAR(64)  NOT NULL DEFAULT 'tenant-alpha',
    workspace_id  VARCHAR(128),
    created_by    VARCHAR(128),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE federated_nodes ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128);
ALTER TABLE data_sharing_agreements ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128);
ALTER TABLE national_indicators ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128);
ALTER TABLE distributed_queries ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128);
ALTER TABLE metadata_vocabularies ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128);
ALTER TABLE diplomatic_treaties ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128);
ALTER TABLE international_reports ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128);
ALTER TABLE federated_search_indices ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128);
ALTER TABLE compliance_audit_logs ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128);
ALTER TABLE object_links ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128);

DO $$
DECLARE
    legacy_constraint_name TEXT;
BEGIN
    SELECT c.conname
    INTO legacy_constraint_name
    FROM pg_constraint c
    JOIN pg_class t ON t.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = t.relnamespace
    WHERE n.nspname = 'statfederation'
      AND t.relname = 'object_links'
      AND c.contype = 'u'
      AND (
          SELECT ARRAY_AGG(a.attname ORDER BY keys.ordinality)
          FROM UNNEST(c.conkey) WITH ORDINALITY AS keys(attnum, ordinality)
          JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = keys.attnum
      ) = ARRAY['source_type', 'source_id', 'target_type', 'target_id', 'relationship']::name[];

    IF legacy_constraint_name IS NOT NULL THEN
        EXECUTE FORMAT('ALTER TABLE statfederation.object_links DROP CONSTRAINT %I', legacy_constraint_name);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_fed_nodes_workspace ON federated_nodes(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_dsa_workspace ON data_sharing_agreements(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_indicators_workspace ON national_indicators(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_dist_queries_workspace ON distributed_queries(initiator_tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_metadata_vocab_workspace ON metadata_vocabularies(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_treaties_workspace ON diplomatic_treaties(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_intl_reports_workspace ON international_reports(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_fed_search_workspace ON federated_search_indices(tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_comp_audit_workspace ON compliance_audit_logs(actor_tenant_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_object_links_workspace ON object_links(tenant_id, workspace_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_object_links_workspace_unique
ON object_links (
    source_type,
    source_id,
    target_type,
    target_id,
    relationship,
    tenant_id,
    COALESCE(workspace_id, '')
);
