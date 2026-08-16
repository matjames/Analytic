CREATE SCHEMA IF NOT EXISTS runops;
SET search_path TO runops, public;

CREATE TABLE IF NOT EXISTS service_targets (
    id TEXT PRIMARY KEY, name TEXT NOT NULL, base_url TEXT NOT NULL,
    criticality TEXT NOT NULL DEFAULT 'tier-2', enabled BOOLEAN NOT NULL DEFAULT TRUE,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS service_probes (
    id BIGSERIAL PRIMARY KEY, service_id TEXT NOT NULL REFERENCES service_targets(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL, status TEXT NOT NULL, http_status INTEGER NOT NULL DEFAULT 0, latency_ms BIGINT NOT NULL DEFAULT 0,
    detail TEXT NOT NULL DEFAULT '', checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_runops_probe_service_time ON service_probes(service_id, checked_at DESC);
CREATE TABLE IF NOT EXISTS metric_samples (
    id BIGSERIAL PRIMARY KEY, service_id TEXT NOT NULL, metric_name TEXT NOT NULL, value DOUBLE PRECISION NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb, observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_runops_metrics_lookup ON metric_samples(service_id, metric_name, observed_at DESC);
CREATE TABLE IF NOT EXISTS log_entries (
    id BIGSERIAL PRIMARY KEY, service_id TEXT NOT NULL, severity TEXT NOT NULL DEFAULT 'INFO', message TEXT NOT NULL,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb, observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS trace_spans (
    trace_id TEXT NOT NULL, span_id TEXT NOT NULL, parent_span_id TEXT NOT NULL DEFAULT '', service_id TEXT NOT NULL,
    operation TEXT NOT NULL, started_at TIMESTAMPTZ NOT NULL, duration_ms BIGINT NOT NULL, status TEXT NOT NULL DEFAULT 'OK', attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY(trace_id, span_id)
);
CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY, fingerprint TEXT NOT NULL UNIQUE, severity TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'open', source TEXT NOT NULL,
    summary TEXT NOT NULL, labels JSONB NOT NULL DEFAULT '{}'::jsonb, starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), acknowledged_by TEXT, acknowledged_at TIMESTAMPTZ, resolved_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_runops_alert_state ON alerts(status, severity, starts_at DESC);
CREATE TABLE IF NOT EXISTS configuration_items (
    id TEXT PRIMARY KEY, ci_type TEXT NOT NULL, name TEXT NOT NULL, environment TEXT NOT NULL DEFAULT 'production', owner TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active', attributes JSONB NOT NULL DEFAULT '{}'::jsonb, relationships JSONB NOT NULL DEFAULT '[]'::jsonb, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY, pipeline_name TEXT NOT NULL, service_id TEXT NOT NULL, environment TEXT NOT NULL, revision TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued', requested_by TEXT NOT NULL, metadata JSONB NOT NULL DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS cloud_resources (
    id TEXT PRIMARY KEY, provider TEXT NOT NULL, account_id TEXT NOT NULL, region TEXT NOT NULL, resource_type TEXT NOT NULL, name TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'managed', monthly_cost NUMERIC(14,2) NOT NULL DEFAULT 0, tags JSONB NOT NULL DEFAULT '{}'::jsonb, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS clusters (
    id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, provider TEXT NOT NULL, region TEXT NOT NULL, version TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'provisioning', policy JSONB NOT NULL DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS workload_schedules (
    id TEXT PRIMARY KEY, workload_name TEXT NOT NULL, cluster_id TEXT NOT NULL REFERENCES clusters(id), namespace TEXT NOT NULL,
    schedule TEXT NOT NULL, desired_replicas INTEGER NOT NULL DEFAULT 1, status TEXT NOT NULL DEFAULT 'scheduled', constraints JSONB NOT NULL DEFAULT '{}'::jsonb, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS runbooks (
    id TEXT PRIMARY KEY, name TEXT NOT NULL, trigger TEXT NOT NULL, steps JSONB NOT NULL DEFAULT '[]'::jsonb, enabled BOOLEAN NOT NULL DEFAULT TRUE, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS dr_plans (
    id TEXT PRIMARY KEY, name TEXT NOT NULL, scope TEXT NOT NULL, rto_seconds INTEGER NOT NULL, rpo_seconds INTEGER NOT NULL, status TEXT NOT NULL DEFAULT 'ready', plan JSONB NOT NULL DEFAULT '{}'::jsonb, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
