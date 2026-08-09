-- Authoritative state for Phase VII Enterprise Intelligence.
-- Redis is deliberately not used as the system of record for investigations.
CREATE TABLE IF NOT EXISTS enterprise_ai_investigations (
    id VARCHAR(100) PRIMARY KEY,
    title TEXT NOT NULL,
    question TEXT NOT NULL,
    owner_id VARCHAR(100) NOT NULL,
    tenant_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100),
    project_id VARCHAR(100),
    source_application VARCHAR(100),
    source_object_type VARCHAR(100),
    source_object_id VARCHAR(100),
    priority VARCHAR(40),
    correlation_id VARCHAR(100),
    status VARCHAR(40) NOT NULL,
    evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    response_id VARCHAR(100),
    recommendation_id VARCHAR(100),
    task_id VARCHAR(100),
    workflow_id VARCHAR(100),
    decision_id VARCHAR(100),
    outcome TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_enterprise_ai_investigations_tenant_owner ON enterprise_ai_investigations (tenant_id, owner_id, updated_at DESC);
