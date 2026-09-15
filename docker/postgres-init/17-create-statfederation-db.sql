-- ══════════════════════════════════════════════════════════════════════════
-- STATGATE APP 6: NATIONAL & GLOBAL FEDERATION (PHASES 23 & 30)
-- ══════════════════════════════════════════════════════════════════════════
-- Initializes the dedicated statfederation database and schema with strict
-- data sovereignty, fail-closed access controls, and cross-module linkage.
-- ══════════════════════════════════════════════════════════════════════════

SELECT 'CREATE DATABASE statfederation' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'statfederation') \gexec

\getenv statfederation_pw STATFEDERATION_DB_PASSWORD
SELECT format('CREATE ROLE "StatFederation" WITH LOGIN PASSWORD %L', :'statfederation_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'StatFederation') \gexec

GRANT ALL PRIVILEGES ON DATABASE statfederation TO "StatFederation";

\c statfederation

-- Schema-level permissions
GRANT ALL ON SCHEMA public TO "StatFederation";
CREATE SCHEMA IF NOT EXISTS statfederation AUTHORIZATION "StatFederation";
ALTER DEFAULT PRIVILEGES IN SCHEMA statfederation GRANT ALL ON TABLES TO "StatFederation";
ALTER DEFAULT PRIVILEGES IN SCHEMA statfederation GRANT ALL ON SEQUENCES TO "StatFederation";
ALTER ROLE "StatFederation" SET search_path TO statfederation, public;

SET search_path TO statfederation, public;
SET ROLE "StatFederation";

-- 1. Federated Nodes Registry (NSS & International)
CREATE TABLE IF NOT EXISTS federated_nodes (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(64) NOT NULL UNIQUE,
    node_type VARCHAR(32) NOT NULL, -- 'nss_agency', 'ministry', 'district', 'regional_bloc', 'international_org', 'academic_partner'
    jurisdiction VARCHAR(64) NOT NULL DEFAULT 'NATIONAL', -- 'NATIONAL', 'EAST_AFRICA', 'AFRICAN_UNION', 'GLOBAL'
    endpoint_url VARCHAR(512) NOT NULL,
    health_status VARCHAR(32) NOT NULL DEFAULT 'HEALTHY', -- 'HEALTHY', 'DEGRADED', 'UNREACHABLE', 'MAINTENANCE'
    trust_level VARCHAR(32) NOT NULL DEFAULT 'STANDARD',  -- 'TIER_1_SOVEREIGN', 'TIER_2_DOMESTIC_AGENCY', 'TIER_3_REGIONAL_PARTNER', 'TIER_4_UNRESTRICTED'
    public_key TEXT,
    protocols JSONB NOT NULL DEFAULT '["SDMX-REST", "StatGate-JSON", "GraphQL"]'::jsonb,
    capabilities JSONB NOT NULL DEFAULT '["AGGREGATE_QUERY", "MICRODATA_EXCHANGE", "METADATA_HARMONIZATION"]'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    contact_email VARCHAR(255),
    last_heartbeat TIMESTAMPTZ,
    latency_ms INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_fed_nodes_type ON federated_nodes(node_type);
CREATE INDEX IF NOT EXISTS idx_fed_nodes_status ON federated_nodes(health_status);
CREATE INDEX IF NOT EXISTS idx_fed_nodes_tenant ON federated_nodes(tenant_id);

-- 2. Data Sharing Agreements (DSAs)
CREATE TABLE IF NOT EXISTS data_sharing_agreements (
    id VARCHAR(64) PRIMARY KEY,
    dsa_number VARCHAR(64) NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    provider_node_id VARCHAR(64) NOT NULL REFERENCES federated_nodes(id) ON DELETE RESTRICT,
    consumer_node_id VARCHAR(64) NOT NULL REFERENCES federated_nodes(id) ON DELETE RESTRICT,
    status VARCHAR(32) NOT NULL DEFAULT 'DRAFT', -- 'DRAFT', 'UNDER_REVIEW', 'ACTIVE', 'SUSPENDED', 'EXPIRED', 'REVOKED'
    access_tier VARCHAR(32) NOT NULL DEFAULT 'RESTRICTED', -- 'PUBLIC_OPEN', 'INTER_AGENCY', 'DIPLOMATIC_CONFIDENTIAL', 'STRICT_SOVEREIGN'
    permitted_domains JSONB NOT NULL DEFAULT '[]'::jsonb, -- e.g. ["agriculture", "health", "demographics", "sdg"]
    classification_allowed VARCHAR(32) NOT NULL DEFAULT 'CONFIDENTIAL',
    requires_approval BOOLEAN NOT NULL DEFAULT TRUE,
    purpose TEXT NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ NOT NULL,
    rate_limit_per_min INT NOT NULL DEFAULT 120,
    daily_quota INT NOT NULL DEFAULT 10000,
    current_daily_usage INT NOT NULL DEFAULT 0,
    governance_approved_by VARCHAR(255),
    governance_approved_at TIMESTAMPTZ,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dsa_provider ON data_sharing_agreements(provider_node_id);
CREATE INDEX IF NOT EXISTS idx_dsa_consumer ON data_sharing_agreements(consumer_node_id);
CREATE INDEX IF NOT EXISTS idx_dsa_status ON data_sharing_agreements(status);
CREATE INDEX IF NOT EXISTS idx_dsa_tenant ON data_sharing_agreements(tenant_id);

-- 3. National Indicator Repository (NSS & SDG)
CREATE TABLE IF NOT EXISTS national_indicators (
    id VARCHAR(64) PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    domain VARCHAR(64) NOT NULL, -- 'SDG', 'ECONOMIC', 'DEMOGRAPHIC', 'HEALTH', 'AGRICULTURE', 'ENVIRONMENT'
    sdmx_dimension VARCHAR(128),
    lead_agency_id VARCHAR(64) REFERENCES federated_nodes(id) ON DELETE SET NULL,
    calculation_method TEXT,
    frequency VARCHAR(32) NOT NULL DEFAULT 'ANNUAL', -- 'MONTHLY', 'QUARTERLY', 'ANNUAL', 'BIENNIAL', 'DECENNIAL'
    target_value NUMERIC(18, 4),
    current_value NUMERIC(18, 4),
    baseline_value NUMERIC(18, 4),
    baseline_year INT,
    unit_of_measure VARCHAR(64),
    tier VARCHAR(16) NOT NULL DEFAULT 'TIER_1', -- 'TIER_1', 'TIER_2', 'TIER_3'
    disaggregation_dimensions JSONB NOT NULL DEFAULT '["GENDER", "DISTRICT", "URBAN_RURAL", "AGE_GROUP"]'::jsonb,
    is_official_statistic BOOLEAN NOT NULL DEFAULT TRUE,
    calendar_release_date DATE,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_indicators_domain ON national_indicators(domain);
CREATE INDEX IF NOT EXISTS idx_indicators_agency ON national_indicators(lead_agency_id);
CREATE INDEX IF NOT EXISTS idx_indicators_tenant ON national_indicators(tenant_id);

-- 4. Distributed Query Execution Ledger
CREATE TABLE IF NOT EXISTS distributed_queries (
    id VARCHAR(64) PRIMARY KEY,
    query_name VARCHAR(255) NOT NULL,
    initiator_user_id VARCHAR(64) NOT NULL,
    initiator_tenant_id VARCHAR(64) NOT NULL,
    target_nodes JSONB NOT NULL DEFAULT '[]'::jsonb,
    query_syntax JSONB NOT NULL,
    execution_strategy VARCHAR(32) NOT NULL DEFAULT 'SCATTER_GATHER', -- 'SCATTER_GATHER', 'PIPELINE', 'POINT_TO_POINT'
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'DISPATCHED', 'PARTIAL_SUCCESS', 'COMPLETED', 'FAILED', 'TIMED_OUT'
    dispatch_timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_timestamp TIMESTAMPTZ,
    total_records_retrieved INT DEFAULT 0,
    execution_time_ms INT DEFAULT 0,
    node_responses JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_summary TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dist_queries_status ON distributed_queries(status);
CREATE INDEX IF NOT EXISTS idx_dist_queries_tenant ON distributed_queries(initiator_tenant_id);

-- 5. Harmonized Metadata Vocabularies & Concept Schemes
CREATE TABLE IF NOT EXISTS metadata_vocabularies (
    id VARCHAR(64) PRIMARY KEY,
    vocabulary_name VARCHAR(255) NOT NULL,
    standard_framework VARCHAR(64) NOT NULL DEFAULT 'SDMX_2.1', -- 'SDMX_2.1', 'DDI_3.3', 'DCAT_AP', 'ISO_19115'
    source_agency VARCHAR(64) NOT NULL,
    target_canonical_concept VARCHAR(128) NOT NULL,
    source_concept_term VARCHAR(128) NOT NULL,
    mapping_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    transformation_expression TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'APPROVED',
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_metadata_vocab_agency ON metadata_vocabularies(source_agency);
CREATE INDEX IF NOT EXISTS idx_metadata_vocab_concept ON metadata_vocabularies(target_canonical_concept);

-- 6. Diplomatic Treaties & Bilateral Evidence Pacts
CREATE TABLE IF NOT EXISTS diplomatic_treaties (
    id VARCHAR(64) PRIMARY KEY,
    treaty_code VARCHAR(64) NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    partner_states JSONB NOT NULL DEFAULT '[]'::jsonb, -- e.g. ["Kenya", "Rwanda", "Uganda", "Tanzania"]
    jurisdiction VARCHAR(64) NOT NULL DEFAULT 'EAST_AFRICAN_COMMUNITY',
    framework_type VARCHAR(64) NOT NULL, -- 'DATA_SOVEREIGNTY_PACT', 'MALABO_CONVENTION', 'AU_AGENDA_2063', 'SDG_ACCORD'
    status VARCHAR(32) NOT NULL DEFAULT 'IN_FORCE', -- 'DRAFT', 'RATIFIED', 'IN_FORCE', 'SUSPENDED', 'TERMINATED'
    ratification_date DATE,
    expiry_date DATE,
    governing_body VARCHAR(255) NOT NULL,
    compliance_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    data_localization_required BOOLEAN NOT NULL DEFAULT FALSE,
    encryption_standard VARCHAR(64) NOT NULL DEFAULT 'AES_256_GCM',
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_treaties_code ON diplomatic_treaties(treaty_code);
CREATE INDEX IF NOT EXISTS idx_treaties_status ON diplomatic_treaties(status);

-- 7. Multilateral International Reports
CREATE TABLE IF NOT EXISTS international_reports (
    id VARCHAR(64) PRIMARY KEY,
    report_title VARCHAR(255) NOT NULL,
    destination_body VARCHAR(64) NOT NULL, -- 'UN_DESA', 'AU_STATAFRIC', 'EAC_SECRETARIAT', 'WHO', 'WORLD_BANK'
    reporting_period VARCHAR(32) NOT NULL, -- '2026-Q2', '2026-ANNUAL', etc.
    status VARCHAR(32) NOT NULL DEFAULT 'DRAFT', -- 'DRAFT', 'VALIDATED', 'TRANSMITTED', 'ACCEPTED', 'REJECTED'
    submission_hash VARCHAR(128),
    transferred_indicators JSONB NOT NULL DEFAULT '[]'::jsonb,
    compliance_passed BOOLEAN NOT NULL DEFAULT TRUE,
    compliance_notes TEXT,
    submitted_by VARCHAR(128),
    submitted_at TIMESTAMPTZ,
    acknowledgement_receipt JSONB,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_intl_reports_body ON international_reports(destination_body);
CREATE INDEX IF NOT EXISTS idx_intl_reports_status ON international_reports(status);

-- 8. Federated Search Remote Indices
CREATE TABLE IF NOT EXISTS federated_search_indices (
    id VARCHAR(64) PRIMARY KEY,
    node_id VARCHAR(64) NOT NULL REFERENCES federated_nodes(id) ON DELETE CASCADE,
    resource_type VARCHAR(64) NOT NULL, -- 'DATASET', 'INDICATOR', 'RESEARCH_PAPER', 'POLICY_BRIEF', 'GEOSPATIAL_LAYER'
    remote_resource_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    abstract TEXT,
    keywords JSONB NOT NULL DEFAULT '[]'::jsonb,
    classification VARCHAR(32) NOT NULL DEFAULT 'PUBLIC',
    temporal_coverage VARCHAR(64),
    spatial_coverage VARCHAR(128),
    direct_access_url VARCHAR(512),
    cached_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default'
);

CREATE INDEX IF NOT EXISTS idx_fed_search_node ON federated_search_indices(node_id);
CREATE INDEX IF NOT EXISTS idx_fed_search_type ON federated_search_indices(resource_type);
CREATE INDEX IF NOT EXISTS idx_fed_search_title ON federated_search_indices USING gin(to_tsvector('english', title || ' ' || COALESCE(abstract, '')));

-- 9. Transboundary Policy Compliance Audit Logs
CREATE TABLE IF NOT EXISTS compliance_audit_logs (
    id VARCHAR(64) PRIMARY KEY,
    event_timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor_user_id VARCHAR(64) NOT NULL,
    actor_tenant_id VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL, -- 'CROSS_BORDER_DISPATCH', 'DSA_EVALUATION', 'MICRODATA_EXPORT', 'PII_REDACTION', 'SOVEREIGNTY_VIOLATION'
    source_jurisdiction VARCHAR(64) NOT NULL,
    target_jurisdiction VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    decision VARCHAR(32) NOT NULL, -- 'ALLOWED', 'DENIED', 'REDACTED_AND_ALLOWED'
    applied_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    redacted_fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    policy_hash VARCHAR(128) NOT NULL,
    reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_comp_audit_actor ON compliance_audit_logs(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_comp_audit_action ON compliance_audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_comp_audit_time ON compliance_audit_logs(event_timestamp);

-- 10. Cross-Application Object Links
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
