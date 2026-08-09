-- ══════════════════════════════════════════════════════════════
-- STATGATE OBJECT LINKAGE FRAMEWORK
-- ══════════════════════════════════════════════════════════════
-- Creates a shared object_links table so that every major object in
-- StatGate (datasets, reports, projects, dashboards, meetings, tasks,
-- documents, etc.) can be linked to any other object.  This is the
-- foundation for the "Every Object Should Be Connected" principle.
-- ══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS object_links (
    id            SERIAL PRIMARY KEY,
    source_type   VARCHAR(64)  NOT NULL,  -- dataset | report | project | dashboard | ...
    source_id     VARCHAR(255) NOT NULL,  -- e.g. covid_19_data
    target_type   VARCHAR(64)  NOT NULL,  -- dataset | report | project | dashboard | ...
    target_id     VARCHAR(255) NOT NULL,  -- e.g. report-123
    relationship  VARCHAR(64)  NOT NULL DEFAULT 'related',  -- related | depends_on | derived_from | ...
    tenant_id     VARCHAR(64)  NOT NULL DEFAULT 'tenant-alpha',
    created_by    VARCHAR(128),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source_type, source_id, target_type, target_id, relationship)
);

CREATE INDEX IF NOT EXISTS idx_object_links_source
    ON object_links(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_object_links_target
    ON object_links(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_object_links_tenant
    ON object_links(tenant_id);