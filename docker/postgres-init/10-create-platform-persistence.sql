-- ═══════════════════════════════════════════════════════════════════
-- Phase X — Enterprise Core Platform Persistence Schema
-- File: 10-create-platform-persistence.sql
-- ═══════════════════════════════════════════════════════════════════
-- Creates the statgate_enterprise database and populates it with
-- the authoritative platform persistence schema.
-- This runs at container startup via postgres-init/ ordering.
-- All statements are idempotent (CREATE IF NOT EXISTS / DO NOTHING).
-- ═══════════════════════════════════════════════════════════════════

-- Create dedicated database for Enterprise Core platform state
SELECT 'CREATE DATABASE statgate_enterprise'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'statgate_enterprise')\gexec

\connect statgate_enterprise;

-- ── Platform Notifications ────────────────────────────────────────
-- Durable notification store that survives Redis restarts.
-- Redis remains the real-time delivery mechanism.

CREATE TABLE IF NOT EXISTS platform_notifications (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL,
    tenant_id        TEXT NOT NULL DEFAULT '',
    title            TEXT NOT NULL,
    body             TEXT NOT NULL,
    priority         TEXT NOT NULL DEFAULT 'medium',
    category         TEXT NOT NULL DEFAULT 'system',
    source_app       TEXT NOT NULL DEFAULT '',
    source_entity    TEXT NOT NULL DEFAULT '',
    source_entity_id TEXT NOT NULL DEFAULT '',
    read             BOOLEAN NOT NULL DEFAULT FALSE,
    archived         BOOLEAN NOT NULL DEFAULT FALSE,
    deep_link        TEXT NOT NULL DEFAULT '',
    metadata         JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_notifications_user
    ON platform_notifications(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_platform_notifications_tenant
    ON platform_notifications(tenant_id);

CREATE INDEX IF NOT EXISTS idx_platform_notifications_unread
    ON platform_notifications(user_id) WHERE read = FALSE AND archived = FALSE;

-- ── Platform Timeline ─────────────────────────────────────────────
-- Institutional activity log — every significant action by every actor.

CREATE TABLE IF NOT EXISTS platform_timeline (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT '',
    tenant_id   TEXT NOT NULL DEFAULT '',
    application TEXT NOT NULL DEFAULT '',
    entity      TEXT NOT NULL DEFAULT '',
    entity_id   TEXT NOT NULL DEFAULT '',
    action      TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    project_id  TEXT NOT NULL DEFAULT '',
    org_id      TEXT NOT NULL DEFAULT '',
    metadata    JSONB,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_timeline_ts
    ON platform_timeline(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_platform_timeline_entity
    ON platform_timeline(entity, entity_id);
CREATE INDEX IF NOT EXISTS idx_platform_timeline_user
    ON platform_timeline(user_id);
CREATE INDEX IF NOT EXISTS idx_platform_timeline_project
    ON platform_timeline(project_id) WHERE project_id != '';

-- ── Platform Audit Log ────────────────────────────────────────────
-- Immutable, append-only institutional audit record.
-- Never updated after insert. Partitioned by month in production.

CREATE TABLE IF NOT EXISTS platform_audit_log (
    id             BIGSERIAL PRIMARY KEY,
    actor          TEXT NOT NULL DEFAULT '',
    action         TEXT NOT NULL,
    resource       TEXT NOT NULL DEFAULT '',
    resource_id    TEXT NOT NULL DEFAULT '',
    tenant_id      TEXT NOT NULL DEFAULT '',
    correlation_id TEXT NOT NULL DEFAULT '',
    request_id     TEXT NOT NULL DEFAULT '',
    outcome        TEXT NOT NULL DEFAULT 'success',
    metadata       JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_audit_actor
    ON platform_audit_log(actor);
CREATE INDEX IF NOT EXISTS idx_platform_audit_resource
    ON platform_audit_log(resource, resource_id);
CREATE INDEX IF NOT EXISTS idx_platform_audit_ts
    ON platform_audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_audit_correlation
    ON platform_audit_log(correlation_id) WHERE correlation_id != '';

-- ── Platform Events ───────────────────────────────────────────────
-- Durable domain event store. Events are written here BEFORE Redis publish.
-- Supports replay, DLQ inspection, and causal chain tracing.

CREATE TABLE IF NOT EXISTS platform_events (
    id             TEXT PRIMARY KEY,
    event_type     TEXT NOT NULL,
    source         TEXT NOT NULL DEFAULT '',
    object_type    TEXT NOT NULL DEFAULT '',
    object_id      TEXT NOT NULL DEFAULT '',
    actor          TEXT NOT NULL DEFAULT '',
    tenant_id      TEXT NOT NULL DEFAULT '',
    project_id     TEXT NOT NULL DEFAULT '',
    org_id         TEXT NOT NULL DEFAULT '',
    correlation_id TEXT NOT NULL DEFAULT '',
    causation_id   TEXT NOT NULL DEFAULT '',
    schema_version INTEGER NOT NULL DEFAULT 1,
    payload        JSONB,
    status         TEXT NOT NULL DEFAULT 'published'
        CHECK (status IN ('published', 'processed', 'failed', 'retried', 'replayed')),
    retry_count    INTEGER NOT NULL DEFAULT 0,
    last_error     TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_platform_events_type
    ON platform_events(event_type);
CREATE INDEX IF NOT EXISTS idx_platform_events_status
    ON platform_events(status) WHERE status IN ('failed', 'published');
CREATE INDEX IF NOT EXISTS idx_platform_events_correlation
    ON platform_events(correlation_id) WHERE correlation_id != '';
CREATE INDEX IF NOT EXISTS idx_platform_events_ts
    ON platform_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_events_object
    ON platform_events(object_type, object_id);

-- ── Platform Service Registry ─────────────────────────────────────
-- Authoritative list of all active StatGate microservices.
-- Services register via the internal API and send periodic heartbeats.

CREATE TABLE IF NOT EXISTS platform_service_registry (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    display_name   TEXT NOT NULL DEFAULT '',
    description    TEXT NOT NULL DEFAULT '',
    api_url        TEXT NOT NULL DEFAULT '',
    ui_url         TEXT NOT NULL DEFAULT '',
    health_url     TEXT NOT NULL DEFAULT '',
    version        TEXT NOT NULL DEFAULT '',
    capabilities   JSONB,
    status         TEXT NOT NULL DEFAULT 'registered'
        CHECK (status IN ('registered', 'active', 'degraded', 'offline')),
    registered_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat TIMESTAMPTZ,
    UNIQUE(name)
);

CREATE INDEX IF NOT EXISTS idx_platform_registry_status
    ON platform_service_registry(status);

-- ── Comments: Data Governance ─────────────────────────────────────
COMMENT ON TABLE platform_notifications   IS 'Phase X: Durable notification store, dual-written with Redis.';
COMMENT ON TABLE platform_timeline        IS 'Phase X: Institutional activity timeline, append-only.';
COMMENT ON TABLE platform_audit_log       IS 'Phase X: Immutable audit trail. Never delete rows in production.';
COMMENT ON TABLE platform_events          IS 'Phase X: Durable domain event store. Source of truth for event replay.';
COMMENT ON TABLE platform_service_registry IS 'Phase X: Authoritative StatGate service registry.';
