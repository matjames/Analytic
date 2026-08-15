-- ═══════════════════════════════════════════════════════════════════
-- StatGate Phase XII — Institutional Intelligence & Governed AI Schema
-- File: 12-create-phase12-intelligence.sql
-- Database: statgate_enterprise
-- Security baseline: SG-SEC-2026-08
--
-- This migration is ADDITIVE. It does not modify Phases I–XI tables.
-- Phase XII is the institutional intelligence layer — an identity and
-- relationship projection layer over existing authoritative systems
-- (UOI / Registry / StatGovernance / Data Fabric), NOT a second copy of
-- application databases.
-- ═══════════════════════════════════════════════════════════════════

\connect statgate_enterprise;

-- ── 1. Institutional Object Registry ───────────────────────────────
-- Projection index of UOI-canonical objects. Every row is a projection
-- (identity + relationship layer) of an authoritative source record.
-- The underlying source object is NEVER duplicated here.
CREATE TABLE IF NOT EXISTS institutional_objects (
    id                TEXT PRIMARY KEY,
    tenant_id         TEXT NOT NULL,
    canonical_id      TEXT NOT NULL,
    object_type       TEXT NOT NULL,
    source_system     TEXT NOT NULL,
    source_object_id  TEXT NOT NULL,
    display_name      TEXT NOT NULL DEFAULT '',
    projection_status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (projection_status IN ('ACTIVE', 'STALE', 'DELETED')),
    metadata          JSONB DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, canonical_id)
);

CREATE INDEX IF NOT EXISTS idx_inst_objects_tenant ON institutional_objects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_inst_objects_canonical ON institutional_objects(canonical_id);
CREATE INDEX IF NOT EXISTS idx_inst_objects_type ON institutional_objects(object_type);
CREATE INDEX IF NOT EXISTS idx_inst_objects_source ON institutional_objects(source_system, source_object_id);
CREATE INDEX IF NOT EXISTS idx_inst_objects_updated ON institutional_objects(updated_at DESC);

-- ── 2. Knowledge Graph Edges (PostgreSQL adjacency list) ──────────
-- Approved architecture: indexed adjacency list + recursive CTE traversal.
-- No graph database is introduced. No cross-tenant traversal is possible
-- because every query is scoped by tenant_id AND canonical ids.
CREATE TABLE IF NOT EXISTS institutional_graph_edges (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL,
    subject_canonical_id TEXT NOT NULL,
    relationship_type   TEXT NOT NULL,
    object_canonical_id TEXT NOT NULL,
    metadata            JSONB DEFAULT '{}'::jsonb,
    provenance_type     TEXT NOT NULL DEFAULT 'AUTHORITATIVE' CHECK (provenance_type IN ('AUTHORITATIVE', 'DERIVED', 'INFERRED', 'AI_GENERATED')),
    confidence          NUMERIC(5,4) NOT NULL DEFAULT 1.0 CHECK (confidence >= 0.0 AND confidence <= 1.0),
    source_event_id     TEXT NOT NULL DEFAULT '',
    valid_from          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until         TIMESTAMPTZ,
    created_by          TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (subject_canonical_id <> object_canonical_id)
);

CREATE INDEX IF NOT EXISTS idx_graph_edges_tenant_subject ON institutional_graph_edges(tenant_id, subject_canonical_id);
CREATE INDEX IF NOT EXISTS idx_graph_edges_tenant_object ON institutional_graph_edges(tenant_id, object_canonical_id);
CREATE INDEX IF NOT EXISTS idx_graph_edges_relationship ON institutional_graph_edges(relationship_type);
CREATE INDEX IF NOT EXISTS idx_graph_edges_valid ON institutional_graph_edges(valid_until);


-- ── 3. Intelligence Signals ────────────────────────────────────────
-- Signals are computed facts consumed from existing systems via the
-- event bus. Signals drive the institutional condition; they are never
-- manually entered as a "condition" itself.
CREATE TABLE IF NOT EXISTS intelligence_signals (
    id            TEXT PRIMARY KEY,
    tenant_id     TEXT NOT NULL,
    signal_type   TEXT NOT NULL,
    domain        TEXT NOT NULL,
    severity      INTEGER NOT NULL CHECK (severity BETWEEN 0 AND 100),
    source_system TEXT NOT NULL,
    source_event_id TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUPERSEDED')),
    payload       JSONB DEFAULT '{}'::jsonb,
    correlation_id TEXT NOT NULL DEFAULT '',
    evidence      TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    evaluated_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_signals_tenant ON intelligence_signals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_signals_domain ON intelligence_signals(domain);
CREATE INDEX IF NOT EXISTS idx_signals_status ON intelligence_signals(status);
CREATE INDEX IF NOT EXISTS idx_signals_ts ON intelligence_signals(created_at DESC);

-- ── 4. Institutional Conditions ────────────────────────────────────
-- Every condition row is a deterministic calculation snapshot with
-- full explainability (supporting signals, affected domains, version,
-- correlation id). A decision-maker can answer "Why is the institution
-- currently ELEVATED?" from this record alone.
CREATE TABLE IF NOT EXISTS institutional_conditions (
    id                    TEXT PRIMARY KEY,
    tenant_id             TEXT NOT NULL,
    condition_level       TEXT NOT NULL CHECK (condition_level IN ('OPTIMAL', 'NOMINAL', 'ATTENTION', 'ELEVATED', 'CRITICAL', 'EMERGENCY', 'UNKNOWN')),
    composite_score       INTEGER NOT NULL CHECK (composite_score BETWEEN 0 AND 100),
    calculation_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    supporting_signals    JSONB DEFAULT '[]'::jsonb,
    affected_domains      JSONB DEFAULT '[]'::jsonb,
    explanation           TEXT NOT NULL DEFAULT '',
    calculation_version   TEXT NOT NULL DEFAULT 'PHASE_XII-1',
    correlation_id        TEXT NOT NULL DEFAULT '',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conditions_tenant_ts ON institutional_conditions(tenant_id, calculation_timestamp DESC);

-- ── 5. Institutional Objectives & KPIs ─────────────────────────────
-- StatGovernance remains the source of truth for governance-owned
-- objectives. These tables hold the Phase XII strategic performance
-- projection consumed/derived from governance + source data, and are
-- correlated to the graph via canonical_id references.
CREATE TABLE IF NOT EXISTS institutional_objectives (
    id               TEXT PRIMARY KEY,
    tenant_id        TEXT NOT NULL,
    canonical_id     TEXT NOT NULL DEFAULT '',
    name             TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    owner            TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'ARCHIVED', 'DRAFT')),
    governance_ref   TEXT NOT NULL DEFAULT '',
    start_date       TIMESTAMPTZ,
    end_date         TIMESTAMPTZ,
    metadata         JSONB DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, canonical_id)
);

CREATE INDEX IF NOT EXISTS idx_objectives_tenant ON institutional_objectives(tenant_id);
CREATE INDEX IF NOT EXISTS idx_objectives_status ON institutional_objectives(status);

CREATE TABLE IF NOT EXISTS kpis (
    id                 TEXT PRIMARY KEY,
    tenant_id          TEXT NOT NULL,
    objective_id       TEXT REFERENCES institutional_objectives(id) ON DELETE CASCADE,
    canonical_id       TEXT NOT NULL DEFAULT '',
    name               TEXT NOT NULL,
    definition         TEXT NOT NULL DEFAULT '',
    owner              TEXT NOT NULL DEFAULT '',
    target             NUMERIC NOT NULL DEFAULT 0,
    measurement_period TEXT NOT NULL DEFAULT 'MONTHLY' CHECK (measurement_period IN ('DAILY', 'WEEKLY', 'MONTHLY', 'QUARTERLY', 'ANNUAL')),
    actual_value       NUMERIC,
    unit               TEXT NOT NULL DEFAULT '',
    source             TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'PAUSED', 'ARCHIVED')),
    data_status        TEXT NOT NULL DEFAULT 'MISSING' CHECK (data_status IN ('ACTUAL', 'ESTIMATED', 'STALE', 'MISSING', 'INVALID')),
    freshness_window_h INTEGER NOT NULL DEFAULT 168,
    last_measured_at   TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, canonical_id)
);

CREATE INDEX IF NOT EXISTS idx_kpis_tenant ON kpis(tenant_id);
CREATE INDEX IF NOT EXISTS idx_kpis_objective ON kpis(objective_id);
CREATE INDEX IF NOT EXISTS idx_kpis_status ON kpis(status);

CREATE TABLE IF NOT EXISTS kpi_measurements (
    id           BIGSERIAL PRIMARY KEY,
    kpi_id       TEXT NOT NULL REFERENCES kpis(id) ON DELETE CASCADE,
    tenant_id    TEXT NOT NULL,
    value        NUMERIC NOT NULL,
    unit         TEXT NOT NULL DEFAULT '',
    source       TEXT NOT NULL DEFAULT '',
    estimated    BOOLEAN NOT NULL DEFAULT FALSE,
    status       TEXT NOT NULL DEFAULT 'ACTUAL' CHECK (status IN ('ACTUAL', 'ESTIMATED', 'INVALID')),
    measured_at  TIMESTAMPTZ NOT NULL,
    recorded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()

-- ── 6. Risk Intelligence ───────────────────────────────────────────
-- StatGovernance remains the source of record for authoritative risk
-- registers. risk_events records correlation events ONLY — repeated
-- incidents, declining KPIs, data quality degradation, cross-domain
-- anomalies — which the intelligence layer turns into AI / intelligence
-- signals. It is NOT a competing risk register.
CREATE TABLE IF NOT EXISTS risk_events (
    id                TEXT PRIMARY KEY,
    tenant_id         TEXT NOT NULL,
    event_type        TEXT NOT NULL,
    source_system     TEXT NOT NULL,
    source_event_id   TEXT NOT NULL DEFAULT '',
    risk_reference    TEXT NOT NULL DEFAULT '',
    severity_estimate INTEGER NOT NULL DEFAULT 0 CHECK (severity_estimate BETWEEN 0 AND 100),
    correlated_objects JSONB DEFAULT '[]'::jsonb,
    signal_id         TEXT,
    evidence          TEXT NOT NULL DEFAULT '',
    correlation_id    TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'DETECTED' CHECK (status IN ('DETECTED', 'REVIEWED', 'ESCALATED', 'DISMISSED')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_events_tenant ON risk_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_risk_events_type ON risk_events(event_type);

-- ── 7. AI Recommendations & Audit ─────────────────────────────────
-- AI is advisory only. Every recommendation flows through a human
-- review state machine. No AI recommendation can execute without
-- human authorization for consequential actions.
CREATE TABLE IF NOT EXISTS ai_recommendations (
    id                 TEXT PRIMARY KEY,
    tenant_id          TEXT NOT NULL,
    provider           TEXT NOT NULL,
    model              TEXT NOT NULL DEFAULT '',
    request_id         TEXT NOT NULL DEFAULT '',
    correlation_id     TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'GENERATED' CHECK (status IN ('GENERATED', 'PENDING_REVIEW', 'AUTHORIZED', 'REJECTED', 'EXECUTED', 'CANCELLED', 'FAILED')),
    recommendation     TEXT NOT NULL DEFAULT '',
    confidence         NUMERIC(4,3) NOT NULL DEFAULT 0 CHECK (confidence >= 0.0 AND confidence <= 1.0),
    reasoning_summary  TEXT NOT NULL DEFAULT '',
    supporting_evidence JSONB DEFAULT '[]'::jsonb,
    affected_objects   JSONB DEFAULT '[]'::jsonb,
    risk_level         TEXT NOT NULL DEFAULT '',
    recommended_actions JSONB DEFAULT '[]'::jsonb,
    limitations        JSONB DEFAULT '[]'::jsonb,
    input_classification TEXT NOT NULL DEFAULT 'INTERNAL',
    output_classification TEXT NOT NULL DEFAULT 'INTERNAL',
    ai_label           TEXT NOT NULL DEFAULT 'AI_GENERATED',
    generated_by       TEXT NOT NULL DEFAULT '',
    reviewed_by        TEXT NOT NULL DEFAULT '',
    reviewed_at        TIMESTAMPTZ,
    human_decision     TEXT NOT NULL DEFAULT '',
    executed_at        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_recos_tenant_status ON ai_recommendations(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_ai_recos_corr ON ai_recommendations(correlation_id);
CREATE INDEX IF NOT EXISTS idx_ai_recos_ts ON ai_recommendations(created_at DESC);

-- Dedicated AI audit trail, linked to the platform audit system by
-- correlation_id / request_id / actor. Sensitive prompt content is
-- never stored — only classifications and structured outputs.
CREATE TABLE IF NOT EXISTS ai_audit_log (
    id                    BIGSERIAL PRIMARY KEY,
    provider              TEXT NOT NULL,
    model                 TEXT NOT NULL DEFAULT '',
    request_id            TEXT NOT NULL DEFAULT '',
    correlation_id        TEXT NOT NULL DEFAULT '',
    input_classification  TEXT NOT NULL DEFAULT 'INTERNAL',
    output_classification TEXT NOT NULL DEFAULT 'INTERNAL',
    recommendation_id     TEXT,
    recommendation        TEXT NOT NULL DEFAULT '',
    confidence            NUMERIC(4,3) NOT NULL DEFAULT 0,
    timestamp             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor                 TEXT NOT NULL DEFAULT '',
    tenant_id             TEXT NOT NULL DEFAULT '',
    authorization_status  TEXT NOT NULL DEFAULT 'GENERATED',
    human_decision        TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_ai_audit_tenant ON ai_audit_log(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ai_audit_ts ON ai_audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ai_audit_corr ON ai_audit_log(correlation_id);

COMMENT ON TABLE institutional_objects       IS 'Phase XII: UOI-canonical institutional object projection index (identity/relationship layer, not a second database).';
COMMENT ON TABLE institutional_graph_edges   IS 'Phase XII: PostgreSQL adjacency-list knowledge graph edges with tenant scoping, provenance and validity windows.';
COMMENT ON TABLE intelligence_signals        IS 'Phase XII: Computed signals consumed from existing systems that drive the institutional condition.';
COMMENT ON TABLE institutional_conditions    IS 'Phase XII: Deterministic, explainable institutional condition snapshots with supporting evidence.';
COMMENT ON TABLE institutional_objectives    IS 'Phase XII: Strategic objective projection; StatGovernance remains the source of truth for governance-owned objectives.';
COMMENT ON TABLE kpis                        IS 'Phase XII: KPI framework distinguishing ACTUAL/ESTIMATED/STALE/MISSING/INVALID data.';
COMMENT ON TABLE kpi_measurements            IS 'Phase XII: Append-only KPI measurement history, indexed and partition-ready.';
COMMENT ON TABLE risk_events                 IS 'Phase XII: Risk intelligence correlation events; NOT a competing risk register (StatGovernance is authoritative).';
COMMENT ON TABLE ai_recommendations          IS 'Phase XII: AI recommendation lifecycle (GENERATED -> PENDING_REVIEW -> AUTHORIZED/REJECTED -> EXECUTED/CANCELLED).';
COMMENT ON TABLE ai_audit_log                IS 'Phase XII: Dedicated AI audit trail linked to the platform audit system; never stores sensitive prompts.';


CREATE INDEX IF NOT EXISTS idx_risk_events_ts ON risk_events(created_at DESC);

);

CREATE INDEX IF NOT EXISTS idx_kpi_measurements_kpi_ts ON kpi_measurements(kpi_id, measured_at DESC);
CREATE INDEX IF NOT EXISTS idx_kpi_measurements_tenant ON kpi_measurements(tenant_id);

CREATE INDEX IF NOT EXISTS idx_conditions_level ON institutional_conditions(condition_level);
