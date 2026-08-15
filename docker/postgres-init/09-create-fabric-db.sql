-- ══════════════════════════════════════════════════════════════
-- STATGATE PHASE IX — INSTITUTIONAL DATA & KNOWLEDGE FABRIC
-- ══════════════════════════════════════════════════════════════
-- Tables providing universal object identity, governed relationship
-- graphs, data catalogue, data dictionary, semantic mappings,
-- knowledge versioning, and dynamic application registration.
-- ══════════════════════════════════════════════════════════════

-- ── 1. Canonical Objects ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS canonical_objects (
    canonical_id        VARCHAR(255) PRIMARY KEY, -- e.g. tenant:source_app:object_type:object_id
    object_id           VARCHAR(128) NOT NULL,
    object_type         VARCHAR(64)  NOT NULL,
    source_application  VARCHAR(64)  NOT NULL,
    tenant_id           VARCHAR(64)  NOT NULL DEFAULT 'tenant_uganda_inst',
    organization_id     VARCHAR(128),
    project_id          VARCHAR(128),
    title               TEXT         NOT NULL,
    description         TEXT,
    created_by          VARCHAR(128),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version             INT          NOT NULL DEFAULT 1,
    status              VARCHAR(32)  NOT NULL DEFAULT 'active',
    classification      VARCHAR(32)  NOT NULL DEFAULT 'verified',
    -- verified | derived | ai_recommendation | decision | unverified
    sensitivity         VARCHAR(32)  NOT NULL DEFAULT 'official',
    -- public | internal | official | confidential | restricted
    canonical_url       TEXT,
    correlation_id      VARCHAR(128),
    metadata            JSONB        NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_canonical_type_id ON canonical_objects(object_type, object_id);
CREATE INDEX IF NOT EXISTS idx_canonical_tenant ON canonical_objects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_canonical_source ON canonical_objects(source_application);
CREATE INDEX IF NOT EXISTS idx_canonical_project ON canonical_objects(project_id);

-- ── 2. Governed Fabric Relationships ──────────────────────────
CREATE TABLE IF NOT EXISTS fabric_relationships (
    id                  VARCHAR(128) PRIMARY KEY,
    from_type           VARCHAR(64)  NOT NULL,
    from_id             VARCHAR(128) NOT NULL,
    to_type             VARCHAR(64)  NOT NULL,
    to_id               VARCHAR(128) NOT NULL,
    relation_type       VARCHAR(64)  NOT NULL, 
    -- belongs_to | derived_from | depends_on | documents | resolves | triggered_by | monitors | governed_by | assigned_to | located_at | verified_by | informs
    confidence          NUMERIC(3,2) NOT NULL DEFAULT 1.00, -- 0.00 to 1.00
    source              VARCHAR(64)  NOT NULL DEFAULT 'enterprise',
    provenance          TEXT,
    lifecycle_status    VARCHAR(32)  NOT NULL DEFAULT 'active', -- active | deprecated | revoked
    created_by          VARCHAR(128),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    audit_history       JSONB        NOT NULL DEFAULT '[]',
    tenant_id           VARCHAR(64)  NOT NULL DEFAULT 'tenant_uganda_inst',
    UNIQUE (from_type, from_id, to_type, to_id, relation_type)
);

CREATE INDEX IF NOT EXISTS idx_fabric_rel_from ON fabric_relationships(from_type, from_id);
CREATE INDEX IF NOT EXISTS idx_fabric_rel_to ON fabric_relationships(to_type, to_id);
CREATE INDEX IF NOT EXISTS idx_fabric_rel_type ON fabric_relationships(relation_type);

-- ── 3. Governed Knowledge Items & Versioning ──────────────────
CREATE TABLE IF NOT EXISTS governed_knowledge (
    id                  VARCHAR(128) PRIMARY KEY,
    title               TEXT         NOT NULL,
    summary             TEXT         NOT NULL,
    content             TEXT         NOT NULL,
    category            VARCHAR(64)  NOT NULL DEFAULT 'article',
    classification      VARCHAR(32)  NOT NULL DEFAULT 'verified',
    -- verified | derived | ai_recommendation | decision | unverified
    author              VARCHAR(128) NOT NULL,
    reviewer            VARCHAR(128),
    approval_status     VARCHAR(32)  NOT NULL DEFAULT 'published', -- draft | review | approved | published | superseded | archived
    version             INT          NOT NULL DEFAULT 1,
    effective_date      TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expiry_date         TIMESTAMPTZ,
    superseded_by       VARCHAR(128),
    source_references   JSONB        NOT NULL DEFAULT '[]',
    change_log          JSONB        NOT NULL DEFAULT '[]',
    tags                JSONB        NOT NULL DEFAULT '[]',
    tenant_id           VARCHAR(64)  NOT NULL DEFAULT 'tenant_uganda_inst',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gov_know_status ON governed_knowledge(approval_status);
CREATE INDEX IF NOT EXISTS idx_gov_know_class ON governed_knowledge(classification);

-- ── 4. Enterprise Data Catalogue ──────────────────────────────
CREATE TABLE IF NOT EXISTS data_catalogue (
    id                  VARCHAR(128) PRIMARY KEY,
    name                TEXT         NOT NULL,
    description         TEXT,
    owner               VARCHAR(128) NOT NULL,
    source_application  VARCHAR(64)  NOT NULL,
    organization_id     VARCHAR(128) NOT NULL DEFAULT 'MOH_UG',
    data_domain         VARCHAR(64)  NOT NULL DEFAULT 'Health Statistics',
    update_frequency    VARCHAR(32)  NOT NULL DEFAULT 'Daily',
    last_refresh        TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    record_count        BIGINT       NOT NULL DEFAULT 0,
    quality_score       NUMERIC(5,2) NOT NULL DEFAULT 98.40,
    completeness        NUMERIC(5,2) NOT NULL DEFAULT 99.10,
    coverage            NUMERIC(5,2) NOT NULL DEFAULT 98.50,
    sensitivity         VARCHAR(32)  NOT NULL DEFAULT 'internal',
    geographic_coverage TEXT         NOT NULL DEFAULT 'Uganda (135 Districts)',
    temporal_coverage   TEXT         NOT NULL DEFAULT '2024-2026',
    schema_definition   JSONB        NOT NULL DEFAULT '[]',
    variables_count     INT          NOT NULL DEFAULT 0,
    lineage_source      TEXT,
    freshness_status    VARCHAR(32)  NOT NULL DEFAULT 'live', -- live | recent | delayed | stale | unavailable
    tenant_id           VARCHAR(64)  NOT NULL DEFAULT 'tenant_uganda_inst',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_catalogue_source ON data_catalogue(source_application);
CREATE INDEX IF NOT EXISTS idx_catalogue_domain ON data_catalogue(data_domain);

-- ── 5. Enterprise Data Dictionary ─────────────────────────────
CREATE TABLE IF NOT EXISTS data_dictionary (
    id                  VARCHAR(128) PRIMARY KEY,
    canonical_name      VARCHAR(128) NOT NULL,
    aliases             JSONB        NOT NULL DEFAULT '[]', -- ['health_facility', 'site', 'service_point']
    definition          TEXT         NOT NULL,
    data_type           VARCHAR(64)  NOT NULL DEFAULT 'string',
    unit                VARCHAR(64),
    permissible_values  JSONB        NOT NULL DEFAULT '[]',
    source_applications JSONB        NOT NULL DEFAULT '["statcollect", "pms", "rms", "analytics"]',
    owner               VARCHAR(128) NOT NULL DEFAULT 'National Statistics Directorate',
    related_indicators  JSONB        NOT NULL DEFAULT '[]',
    related_datasets    JSONB        NOT NULL DEFAULT '[]',
    version             INT          NOT NULL DEFAULT 1,
    status              VARCHAR(32)  NOT NULL DEFAULT 'approved',
    tenant_id           VARCHAR(64)  NOT NULL DEFAULT 'tenant_uganda_inst',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dict_canonical ON data_dictionary(canonical_name);

-- ── 6. Semantic Concept Mappings ──────────────────────────────
CREATE TABLE IF NOT EXISTS semantic_mappings (
    id                  VARCHAR(128) PRIMARY KEY,
    source_term         VARCHAR(128) NOT NULL,
    canonical_concept   VARCHAR(128) NOT NULL,
    source_application  VARCHAR(64)  NOT NULL,
    standard_code       VARCHAR(64), -- e.g. DHIS2_FACILITY_CODE, LOINC_XYZ
    confidence          NUMERIC(3,2) NOT NULL DEFAULT 1.00,
    approved_by         VARCHAR(128),
    tenant_id           VARCHAR(64)  NOT NULL DEFAULT 'tenant_uganda_inst',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_semantic_term ON semantic_mappings(source_term);
CREATE INDEX IF NOT EXISTS idx_semantic_concept ON semantic_mappings(canonical_concept);

-- ── 7. Dynamic Application Registry ───────────────────────────
CREATE TABLE IF NOT EXISTS application_registry (
    application_id      VARCHAR(64) PRIMARY KEY,
    name                VARCHAR(128) NOT NULL,
    version             VARCHAR(32)  NOT NULL,
    owner               VARCHAR(128) NOT NULL,
    api_url             TEXT         NOT NULL,
    ui_url              TEXT         NOT NULL,
    health_endpoint     TEXT         NOT NULL,
    capabilities        JSONB        NOT NULL DEFAULT '[]',
    supported_objects   JSONB        NOT NULL DEFAULT '[]',
    supported_events    JSONB        NOT NULL DEFAULT '[]',
    auth_method         VARCHAR(32)  NOT NULL DEFAULT 'bearer_jwt',
    status              VARCHAR(32)  NOT NULL DEFAULT 'active',
    last_heartbeat      TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── 8. Institutional Decisions Memory ─────────────────────────
CREATE TABLE IF NOT EXISTS institutional_decisions (
    id                  VARCHAR(128) PRIMARY KEY,
    title               TEXT         NOT NULL,
    context             TEXT         NOT NULL,
    problem_statement   TEXT         NOT NULL,
    evidence_sources    JSONB        NOT NULL DEFAULT '[]',
    decided_by          VARCHAR(128) NOT NULL,
    decision_timestamp  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    alternatives_tested JSONB        NOT NULL DEFAULT '[]',
    action_tasks        JSONB        NOT NULL DEFAULT '[]',
    expected_outcome    TEXT         NOT NULL,
    actual_outcome      TEXT,
    outcome_status      VARCHAR(32)  NOT NULL DEFAULT 'monitoring', -- monitoring | achieved | partially_achieved | deviated | resolved
    verified_at         TIMESTAMPTZ,
    tenant_id           VARCHAR(64)  NOT NULL DEFAULT 'tenant_uganda_inst',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inst_dec_status ON institutional_decisions(outcome_status);
CREATE INDEX IF NOT EXISTS idx_inst_dec_actor ON institutional_decisions(decided_by);
