-- ==============================================================================
-- STATGATE APP 12: DATA ENGINEERING, SCIENCE & SEARCH (P37 / P38 / P47)
-- SCHEMA DEFINITION & MIGRATION SCRIPT
-- ==============================================================================

CREATE SCHEMA IF NOT EXISTS statdata;

-- 1. Central Data Catalog & Datasets
CREATE TABLE IF NOT EXISTS statdata.datasets (
    id VARCHAR(64) PRIMARY KEY,
    urn VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    domain VARCHAR(64) NOT NULL,
    classification VARCHAR(32) NOT NULL DEFAULT 'INTERNAL',
    owner_team VARCHAR(128) NOT NULL,
    owner_email VARCHAR(128) NOT NULL,
    format VARCHAR(32) NOT NULL DEFAULT 'PARQUET',
    storage_uri TEXT NOT NULL,
    schema_id VARCHAR(64),
    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    quality_score NUMERIC(5,2) DEFAULT 100.0,
    row_count BIGINT DEFAULT 0,
    size_bytes BIGINT DEFAULT 0,
    tags JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_datasets_tenant_domain ON statdata.datasets(tenant_id, domain);
CREATE INDEX IF NOT EXISTS idx_datasets_classification ON statdata.datasets(classification);

-- 2. Data Sources
CREATE TABLE IF NOT EXISTS statdata.data_sources (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    source_type VARCHAR(64) NOT NULL,
    connection_uri TEXT,
    auth_type VARCHAR(32) DEFAULT 'NONE',
    credentials JSONB DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    last_tested_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. Schema Registry
CREATE TABLE IF NOT EXISTS statdata.schema_registry (
    id VARCHAR(128) PRIMARY KEY,
    subject VARCHAR(128) NOT NULL,
    version INT NOT NULL,
    schema_type VARCHAR(32) NOT NULL DEFAULT 'JSON_SCHEMA',
    schema_content TEXT NOT NULL,
    compatibility VARCHAR(32) NOT NULL DEFAULT 'BACKWARD',
    description TEXT,
    fields JSONB DEFAULT '[]'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(subject, version, tenant_id)
);

-- 4. Data Contracts
CREATE TABLE IF NOT EXISTS statdata.data_contracts (
    id VARCHAR(64) PRIMARY KEY,
    dataset_id VARCHAR(64) NOT NULL REFERENCES statdata.datasets(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    producer_team VARCHAR(128) NOT NULL,
    consumer_team VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    max_latency_mins INT DEFAULT 60,
    min_quality_rate NUMERIC(5,2) DEFAULT 95.0,
    expected_volume BIGINT DEFAULT 0,
    sla_config JSONB DEFAULT '{}'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    valid_from TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 5. Data Quality Rules & Reports
CREATE TABLE IF NOT EXISTS statdata.data_quality_rules (
    id VARCHAR(64) PRIMARY KEY,
    dataset_id VARCHAR(64) NOT NULL REFERENCES statdata.datasets(id) ON DELETE CASCADE,
    rule_name VARCHAR(255) NOT NULL,
    rule_type VARCHAR(64) NOT NULL,
    target_field VARCHAR(128),
    parameters JSONB DEFAULT '{}'::jsonb,
    severity VARCHAR(32) NOT NULL DEFAULT 'ERROR',
    is_enabled BOOLEAN DEFAULT TRUE,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statdata.data_quality_reports (
    id VARCHAR(64) PRIMARY KEY,
    dataset_id VARCHAR(64) NOT NULL REFERENCES statdata.datasets(id) ON DELETE CASCADE,
    pipeline_run_id VARCHAR(64),
    status VARCHAR(32) NOT NULL,
    quality_score NUMERIC(5,2) NOT NULL,
    total_rules INT NOT NULL,
    passed_rules INT NOT NULL,
    failed_rules INT NOT NULL,
    rule_results JSONB DEFAULT '[]'::jsonb,
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default'
);

-- 6. Data Lineage
CREATE TABLE IF NOT EXISTS statdata.lineage_nodes (
    id VARCHAR(128) PRIMARY KEY,
    urn VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS statdata.lineage_edges (
    id VARCHAR(128) PRIMARY KEY,
    source_node_id VARCHAR(128) NOT NULL,
    target_node_id VARCHAR(128) NOT NULL,
    relation_type VARCHAR(64) NOT NULL,
    pipeline_id VARCHAR(64),
    transformation TEXT,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 7. Data Pipelines & Runs
CREATE TABLE IF NOT EXISTS statdata.pipelines (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    pipeline_type VARCHAR(32) NOT NULL DEFAULT 'ETL',
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    cron_schedule VARCHAR(64),
    source_dataset_id VARCHAR(64),
    target_dataset_id VARCHAR(64),
    stages JSONB NOT NULL DEFAULT '[]'::jsonb,
    config JSONB DEFAULT '{}'::jsonb,
    max_retries INT DEFAULT 3,
    timeout_seconds INT DEFAULT 3600,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statdata.pipeline_runs (
    id VARCHAR(64) PRIMARY KEY,
    pipeline_id VARCHAR(64) NOT NULL REFERENCES statdata.pipelines(id) ON DELETE CASCADE,
    pipeline_name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,
    trigger_type VARCHAR(32) NOT NULL DEFAULT 'MANUAL',
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMPTZ,
    duration_ms BIGINT DEFAULT 0,
    records_read BIGINT DEFAULT 0,
    records_written BIGINT DEFAULT 0,
    records_rejected BIGINT DEFAULT 0,
    stage_runs JSONB DEFAULT '[]'::jsonb,
    error_message TEXT,
    metrics JSONB DEFAULT '{}'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    triggered_by VARCHAR(128) NOT NULL
);

-- 8. Streaming Jobs
CREATE TABLE IF NOT EXISTS statdata.streaming_jobs (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    source_topic VARCHAR(255) NOT NULL,
    target_sink VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'RUNNING',
    throughput_msg_sec NUMERIC(10,2) DEFAULT 0.0,
    lag_records BIGINT DEFAULT 0,
    config JSONB DEFAULT '{}'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    last_checkpoint TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 9. Feature Store
CREATE TABLE IF NOT EXISTS statdata.feature_views (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL UNIQUE,
    entity_name VARCHAR(128) NOT NULL,
    description TEXT,
    ttl_seconds BIGINT DEFAULT 86400,
    features JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_query TEXT,
    online_store BOOLEAN DEFAULT TRUE,
    offline_sink TEXT,
    tags JSONB DEFAULT '[]'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statdata.feature_records (
    entity_key VARCHAR(128) NOT NULL,
    feature_view_id VARCHAR(64) NOT NULL REFERENCES statdata.feature_views(id) ON DELETE CASCADE,
    values JSONB NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    PRIMARY KEY(entity_key, feature_view_id)
);

-- 10. Scientific Notebooks & Experiments (P38)
CREATE TABLE IF NOT EXISTS statdata.notebook_sessions (
    id VARCHAR(64) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    language VARCHAR(32) NOT NULL DEFAULT 'PYTHON',
    kernel_state VARCHAR(32) NOT NULL DEFAULT 'IDLE',
    dataset_refs JSONB DEFAULT '[]'::jsonb,
    cells JSONB NOT NULL DEFAULT '[]'::jsonb,
    variables JSONB DEFAULT '{}'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statdata.experiments (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    domain VARCHAR(64) NOT NULL,
    tags JSONB DEFAULT '[]'::jsonb,
    artifact_uri TEXT,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statdata.experiment_runs (
    id VARCHAR(64) PRIMARY KEY,
    experiment_id VARCHAR(64) NOT NULL REFERENCES statdata.experiments(id) ON DELETE CASCADE,
    run_name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'RUNNING',
    parameters JSONB DEFAULT '{}'::jsonb,
    metrics JSONB DEFAULT '{}'::jsonb,
    tags JSONB DEFAULT '{}'::jsonb,
    artifacts JSONB DEFAULT '[]'::jsonb,
    start_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMPTZ,
    duration_ms BIGINT DEFAULT 0,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL
);

-- 11. ML Model Registry
CREATE TABLE IF NOT EXISTS statdata.registered_models (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    domain VARCHAR(64) NOT NULL,
    framework VARCHAR(64) NOT NULL DEFAULT 'PYTORCH',
    latest_stage VARCHAR(32) NOT NULL DEFAULT 'NONE',
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statdata.model_versions (
    id VARCHAR(128) PRIMARY KEY,
    model_id VARCHAR(64) NOT NULL REFERENCES statdata.registered_models(id) ON DELETE CASCADE,
    version INT NOT NULL,
    stage VARCHAR(32) NOT NULL DEFAULT 'STAGING',
    source_run_id VARCHAR(64),
    artifact_uri TEXT NOT NULL,
    metrics_summary JSONB DEFAULT '{}'::jsonb,
    input_schema TEXT,
    output_schema TEXT,
    description TEXT,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(model_id, version)
);

-- 12. Compute Cluster Management
CREATE TABLE IF NOT EXISTS statdata.compute_nodes (
    id VARCHAR(64) PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL UNIQUE,
    ip_address VARCHAR(64) NOT NULL,
    total_cpus INT NOT NULL DEFAULT 16,
    alloc_cpus INT NOT NULL DEFAULT 0,
    total_ram_gb NUMERIC(8,2) NOT NULL DEFAULT 64.0,
    alloc_ram_gb NUMERIC(8,2) NOT NULL DEFAULT 0.0,
    total_gpus INT NOT NULL DEFAULT 0,
    alloc_gpus INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'READY',
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    last_ping TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statdata.compute_jobs (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    job_type VARCHAR(64) NOT NULL,
    assigned_node VARCHAR(255),
    required_cpus INT DEFAULT 2,
    required_ram_gb NUMERIC(8,2) DEFAULT 8.0,
    required_gpus INT DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'QUEUED',
    params JSONB DEFAULT '{}'::jsonb,
    output_data JSONB DEFAULT '{}'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ
);

-- 13. Enterprise Search & Indexing (P47)
CREATE TABLE IF NOT EXISTS statdata.search_indexes (
    id VARCHAR(64) PRIMARY KEY,
    index_name VARCHAR(128) NOT NULL UNIQUE,
    document_count BIGINT DEFAULT 0,
    dimension INT DEFAULT 384,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    last_indexed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statdata.indexed_documents (
    id VARCHAR(128) PRIMARY KEY,
    index_name VARCHAR(128) NOT NULL DEFAULT 'statgate_global',
    resource_id VARCHAR(128) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    domain VARCHAR(64) NOT NULL,
    classification VARCHAR(32) NOT NULL DEFAULT 'INTERNAL',
    owner VARCHAR(128) NOT NULL DEFAULT 'system',
    tags JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    vector JSONB DEFAULT '[]'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_indexed_documents_type ON statdata.indexed_documents(resource_type);
CREATE INDEX IF NOT EXISTS idx_indexed_documents_domain ON statdata.indexed_documents(domain);

CREATE TABLE IF NOT EXISTS statdata.saved_searches (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    query TEXT NOT NULL,
    filters JSONB DEFAULT '{}'::jsonb,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 14. Compliance & Object Links
CREATE TABLE IF NOT EXISTS statdata.audit_logs (
    id VARCHAR(64) PRIMARY KEY,
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    actor_id VARCHAR(128) NOT NULL,
    actor_tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    status VARCHAR(32) NOT NULL DEFAULT 'SUCCESS',
    details JSONB DEFAULT '{}'::jsonb,
    ip_address VARCHAR(64),
    event_timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statdata.object_links (
    id SERIAL PRIMARY KEY,
    source_type VARCHAR(64) NOT NULL,
    source_id VARCHAR(64) NOT NULL,
    target_type VARCHAR(64) NOT NULL,
    target_id VARCHAR(64) NOT NULL,
    relation_type VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_by VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_object_links_src ON statdata.object_links(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_object_links_tgt ON statdata.object_links(target_type, target_id);
