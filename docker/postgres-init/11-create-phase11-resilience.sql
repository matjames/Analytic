-- ═══════════════════════════════════════════════════════════════════
-- StatGate Phase XI — Institutional Resilience, Continuity & Assurance Schema
-- File: 11-create-phase11-resilience.sql
-- Database: statgate_enterprise
-- ═══════════════════════════════════════════════════════════════════

\connect statgate_enterprise;

-- ── 1. Service Resilience Profiles ─────────────────────────────────
CREATE TABLE IF NOT EXISTS service_resilience_profiles (
    service_id          TEXT PRIMARY KEY,
    service_name        TEXT NOT NULL,
    criticality_tier    TEXT NOT NULL DEFAULT 'Tier 1' CHECK (criticality_tier IN ('Tier 0', 'Tier 1', 'Tier 2', 'Tier 3')),
    business_owner      TEXT NOT NULL DEFAULT 'Institutional Operations',
    technical_owner     TEXT NOT NULL DEFAULT 'Platform Reliability Team',
    dependencies        JSONB DEFAULT '[]'::jsonb,
    health_endpoint     TEXT NOT NULL DEFAULT '',
    readiness_endpoint  TEXT NOT NULL DEFAULT '',
    liveness_endpoint   TEXT NOT NULL DEFAULT '',
    rto_target_sec      INTEGER NOT NULL DEFAULT 300,  -- 5 min target
    rpo_target_sec      INTEGER NOT NULL DEFAULT 60,   -- 1 min target
    actual_rto_sec      INTEGER NOT NULL DEFAULT 0,
    actual_rpo_sec      INTEGER NOT NULL DEFAULT 0,
    availability_pct    NUMERIC(5,2) NOT NULL DEFAULT 99.90,
    last_drill_date     TIMESTAMPTZ,
    last_drill_status   TEXT NOT NULL DEFAULT 'NOT TESTED' CHECK (last_drill_status IN ('COMPLIANT', 'AT RISK', 'BREACHED', 'NOT TESTED')),
    compliance_status   TEXT NOT NULL DEFAULT 'COMPLIANT' CHECK (compliance_status IN ('COMPLIANT', 'AT RISK', 'BREACHED', 'NOT TESTED')),
    operational_status  TEXT NOT NULL DEFAULT 'HEALTHY' CHECK (operational_status IN ('HEALTHY', 'DEGRADED', 'UNAVAILABLE', 'RECOVERING')),
    confidence_score    INTEGER NOT NULL DEFAULT 85 CHECK (confidence_score BETWEEN 0 AND 100),
    recovery_procedure  TEXT NOT NULL DEFAULT '',
    backup_requirement  TEXT NOT NULL DEFAULT 'Continuous WAL + Hourly Snapshot',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_resilience_criticality ON service_resilience_profiles(criticality_tier);
CREATE INDEX IF NOT EXISTS idx_resilience_compliance ON service_resilience_profiles(compliance_status);
CREATE INDEX IF NOT EXISTS idx_resilience_status ON service_resilience_profiles(operational_status);

-- ── 2. Recovery Drills & Steps ────────────────────────────────────
CREATE TABLE IF NOT EXISTS recovery_drills (
    id                  TEXT PRIMARY KEY,
    title               TEXT NOT NULL,
    service_id          TEXT NOT NULL REFERENCES service_resilience_profiles(service_id) ON DELETE CASCADE,
    scenario            TEXT NOT NULL,
    mode                TEXT NOT NULL DEFAULT 'SIMULATION' CHECK (mode IN ('SIMULATION', 'CONTROLLED_TEST', 'PRODUCTION')),
    status              TEXT NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN ('SCHEDULED', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED')),
    rto_target_sec      INTEGER NOT NULL DEFAULT 300,
    rpo_target_sec      INTEGER NOT NULL DEFAULT 60,
    measured_rto_sec    INTEGER NOT NULL DEFAULT 0,
    measured_rpo_sec    INTEGER NOT NULL DEFAULT 0,
    drill_result        TEXT NOT NULL DEFAULT 'PENDING' CHECK (drill_result IN ('SUCCESS', 'PARTIAL', 'BREACHED', 'FAILED', 'PENDING')),
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    conducted_by        TEXT NOT NULL DEFAULT '',
    evidence_id         TEXT,
    summary             TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_drills_service ON recovery_drills(service_id);
CREATE INDEX IF NOT EXISTS idx_drills_status ON recovery_drills(status);
CREATE INDEX IF NOT EXISTS idx_drills_date ON recovery_drills(created_at DESC);

CREATE TABLE IF NOT EXISTS recovery_drill_steps (
    id                  TEXT PRIMARY KEY,
    drill_id            TEXT NOT NULL REFERENCES recovery_drills(id) ON DELETE CASCADE,
    step_number         INTEGER NOT NULL,
    name                TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    action_type         TEXT NOT NULL DEFAULT 'VERIFY',
    status              TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'RUNNING', 'PASSED', 'FAILED', 'SKIPPED')),
    duration_ms         INTEGER NOT NULL DEFAULT 0,
    output              TEXT NOT NULL DEFAULT '',
    error_message       TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_drill_steps_drill ON recovery_drill_steps(drill_id, step_number);

-- ── 3. Backup Assurance Registry & Restore Tests ───────────────────
CREATE TABLE IF NOT EXISTS backup_registry (
    id                  TEXT PRIMARY KEY,
    service_id          TEXT NOT NULL REFERENCES service_resilience_profiles(service_id) ON DELETE CASCADE,
    backup_type         TEXT NOT NULL CHECK (backup_type IN ('FULL_SNAPSHOT', 'INCREMENTAL_WAL', 'LOGICAL_DUMP', 'CONFIG_STATE')),
    data_scope          TEXT NOT NULL,
    storage_location    TEXT NOT NULL,
    encryption_status   TEXT NOT NULL DEFAULT 'AES-256-GCM',
    retention_policy    TEXT NOT NULL DEFAULT '30_DAYS_IMMUTABLE',
    size_bytes          BIGINT NOT NULL DEFAULT 0,
    checksum_sha256     TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'VERIFIED' CHECK (status IN ('CREATED', 'VERIFIED', 'RESTORE_TESTED', 'CORRUPT', 'EXPIRED')),
    last_restore_test   TIMESTAMPTZ,
    restore_verified    BOOLEAN NOT NULL DEFAULT FALSE,
    certified_by        TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backup_service ON backup_registry(service_id);
CREATE INDEX IF NOT EXISTS idx_backup_status ON backup_registry(status);

CREATE TABLE IF NOT EXISTS restore_tests (
    id                  TEXT PRIMARY KEY,
    backup_id           TEXT NOT NULL REFERENCES backup_registry(id) ON DELETE CASCADE,
    service_id          TEXT NOT NULL,
    test_mode           TEXT NOT NULL DEFAULT 'ISOLATED_SANDBOX' CHECK (test_mode IN ('ISOLATED_SANDBOX', 'STAGING_VERIFICATION', 'SHADOW_RESTORATION')),
    status              TEXT NOT NULL DEFAULT 'PASSED' CHECK (status IN ('RUNNING', 'PASSED', 'INTEGRITY_FAILED', 'SERVICE_FAILED')),
    duration_sec        INTEGER NOT NULL DEFAULT 0,
    records_restored    BIGINT NOT NULL DEFAULT 0,
    checksum_match      BOOLEAN NOT NULL DEFAULT TRUE,
    service_probe_pass  BOOLEAN NOT NULL DEFAULT TRUE,
    evidence_id         TEXT,
    conducted_by        TEXT NOT NULL DEFAULT '',
    notes               TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_restore_backup ON restore_tests(backup_id);
CREATE INDEX IF NOT EXISTS idx_restore_status ON restore_tests(status);

-- ── 4. Data Integrity Checks & Results ────────────────────────────
CREATE TABLE IF NOT EXISTS integrity_checks (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    category            TEXT NOT NULL CHECK (category IN ('ORPHAN_RECORDS', 'TENANT_ISOLATION', 'CANONICAL_IDENTITY', 'EVENT_ORDERING', 'AUDIT_CHAIN', 'DLQ_ACCUMULATION', 'SERVICE_REGISTRY')),
    description         TEXT NOT NULL DEFAULT '',
    query_assertion     TEXT NOT NULL DEFAULT '',
    severity            TEXT NOT NULL DEFAULT 'HIGH' CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS integrity_results (
    id                  TEXT PRIMARY KEY,
    check_id            TEXT NOT NULL REFERENCES integrity_checks(id) ON DELETE CASCADE,
    check_name          TEXT NOT NULL,
    category            TEXT NOT NULL,
    status              TEXT NOT NULL CHECK (status IN ('PASSED', 'WARNING', 'FAILED')),
    anomalies_count     INTEGER NOT NULL DEFAULT 0,
    details             JSONB DEFAULT '{}'::jsonb,
    duration_ms         INTEGER NOT NULL DEFAULT 0,
    run_by              TEXT NOT NULL DEFAULT 'AUTONOMOUS_ENGINE',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_integrity_results_check ON integrity_results(check_id);
CREATE INDEX IF NOT EXISTS idx_integrity_results_date ON integrity_results(created_at DESC);

-- ── 5. Autonomous Incident Lifecycle & Actions ─────────────────────
CREATE TABLE IF NOT EXISTS incidents (
    id                  TEXT PRIMARY KEY,
    title               TEXT NOT NULL,
    severity            TEXT NOT NULL CHECK (severity IN ('INFO', 'WARNING', 'HIGH', 'CRITICAL', 'CATASTROPHIC')),
    status              TEXT NOT NULL DEFAULT 'DETECTED' CHECK (status IN ('DETECTED', 'TRIAGED', 'ACKNOWLEDGED', 'MITIGATING', 'RECOVERING', 'VALIDATING', 'RESOLVED', 'CLOSED')),
    service_id          TEXT NOT NULL,
    detection_source    TEXT NOT NULL DEFAULT 'AUTONOMOUS_ENGINE',
    root_cause_summary  TEXT NOT NULL DEFAULT '',
    affected_components JSONB DEFAULT '[]'::jsonb,
    correlation_id      TEXT NOT NULL DEFAULT '',
    detected_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at     TIMESTAMPTZ,
    acknowledged_by     TEXT NOT NULL DEFAULT '',
    resolved_at         TIMESTAMPTZ,
    resolved_by         TEXT NOT NULL DEFAULT '',
    closed_at           TIMESTAMPTZ,
    closed_by           TEXT NOT NULL DEFAULT '',
    rto_impact_sec      INTEGER NOT NULL DEFAULT 0,
    rpo_impact_sec      INTEGER NOT NULL DEFAULT 0,
    evidence_id         TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents(severity);
CREATE INDEX IF NOT EXISTS idx_incidents_service ON incidents(service_id);
CREATE INDEX IF NOT EXISTS idx_incidents_date ON incidents(created_at DESC);

CREATE TABLE IF NOT EXISTS incident_events (
    id                  BIGSERIAL PRIMARY KEY,
    incident_id         TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    from_status         TEXT NOT NULL,
    to_status           TEXT NOT NULL,
    actor               TEXT NOT NULL,
    reason              TEXT NOT NULL DEFAULT '',
    evidence            JSONB DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_incident_events_inc ON incident_events(incident_id);

CREATE TABLE IF NOT EXISTS incident_actions (
    id                  TEXT PRIMARY KEY,
    incident_id         TEXT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    action_type         TEXT NOT NULL,
    description         TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'COMPLETED' CHECK (status IN ('PROPOSED', 'IN_PROGRESS', 'COMPLETED', 'FAILED')),
    executed_by         TEXT NOT NULL,
    output              TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── 6. Runbooks & Execution Records ───────────────────────────────
CREATE TABLE IF NOT EXISTS runbooks (
    id                  TEXT PRIMARY KEY,
    title               TEXT NOT NULL,
    incident_type       TEXT NOT NULL,
    affected_services   JSONB DEFAULT '[]'::jsonb,
    detection_method    TEXT NOT NULL DEFAULT '',
    immediate_actions   JSONB DEFAULT '[]'::jsonb,
    recovery_steps      JSONB DEFAULT '[]'::jsonb,
    validation_checks   JSONB DEFAULT '[]'::jsonb,
    rollback_procedure  TEXT NOT NULL DEFAULT '',
    escalation_path     TEXT NOT NULL DEFAULT '',
    evidence_required   TEXT NOT NULL DEFAULT '',
    closure_criteria    TEXT NOT NULL DEFAULT '',
    version             INTEGER NOT NULL DEFAULT 1,
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS runbook_executions (
    id                  TEXT PRIMARY KEY,
    runbook_id          TEXT NOT NULL REFERENCES runbooks(id) ON DELETE CASCADE,
    incident_id         TEXT REFERENCES incidents(id) ON DELETE SET NULL,
    executed_by         TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'IN_PROGRESS' CHECK (status IN ('IN_PROGRESS', 'COMPLETED', 'FAILED', 'ABORTED')),
    step_results        JSONB DEFAULT '[]'::jsonb,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ,
    evidence_id         TEXT
);

-- ── 7. Resilience Evidence Vault ──────────────────────────────────
CREATE TABLE IF NOT EXISTS resilience_evidence (
    id                  TEXT PRIMARY KEY,
    activity_type       TEXT NOT NULL CHECK (activity_type IN ('RECOVERY_DRILL', 'BACKUP_RESTORE_TEST', 'INCIDENT_RECOVERY', 'EVENT_REPLAY', 'INTEGRITY_AUDIT', 'SECURITY_INVESTIGATION')),
    target_service      TEXT NOT NULL,
    actor               TEXT NOT NULL,
    result_status       TEXT NOT NULL CHECK (result_status IN ('VERIFIED_SUCCESS', 'COMPLIANT_WITH_WARNINGS', 'BREACHED', 'FAILED')),
    metrics             JSONB DEFAULT '{}'::jsonb,
    logs_summary        TEXT NOT NULL DEFAULT '',
    checksum_digest     TEXT NOT NULL DEFAULT '',
    audit_reference     TEXT NOT NULL DEFAULT '',
    verification_hash   TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_evidence_type ON resilience_evidence(activity_type);
CREATE INDEX IF NOT EXISTS idx_evidence_service ON resilience_evidence(target_service);
CREATE INDEX IF NOT EXISTS idx_evidence_date ON resilience_evidence(created_at DESC);

-- ── 8. Resilience Metrics Snapshots ───────────────────────────────
CREATE TABLE IF NOT EXISTS resilience_metrics (
    id                  BIGSERIAL PRIMARY KEY,
    resilience_score    INTEGER NOT NULL CHECK (resilience_score BETWEEN 0 AND 100),
    availability_score  INTEGER NOT NULL,
    rto_score           INTEGER NOT NULL,
    rpo_score           INTEGER NOT NULL,
    backup_score        INTEGER NOT NULL,
    integrity_score     INTEGER NOT NULL,
    event_score         INTEGER NOT NULL,
    incident_score      INTEGER NOT NULL,
    active_incidents    INTEGER NOT NULL DEFAULT 0,
    untested_services   INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_resilience_metrics_ts ON resilience_metrics(created_at DESC);

COMMENT ON TABLE service_resilience_profiles IS 'Phase XI: Institutional RTO/RPO targets, criticality tiers, and reliability ownership.';
COMMENT ON TABLE recovery_drills             IS 'Phase XI: Disaster recovery simulations and controlled execution records.';
COMMENT ON TABLE backup_registry            IS 'Phase XI: Authoritative backup inventory and restore test verification registry.';
COMMENT ON TABLE integrity_checks           IS 'Phase XI: Automated institutional data integrity assertions.';
COMMENT ON TABLE incidents                  IS 'Phase XI: Formal incident lifecycle state machine and audit trail.';
COMMENT ON TABLE runbooks                   IS 'Phase XI: Machine-readable recovery runbook library.';
COMMENT ON TABLE resilience_evidence        IS 'Phase XI: Cryptographically verified institutional evidence vault.';
