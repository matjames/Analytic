-- ═══════════════════════════════════════════════════════════
--  StatGate Institutional Governance Platform (StatGovernance)
--  PostgreSQL initialization script - Phase VIII
-- ═══════════════════════════════════════════════════════════

-- Create StatGovernance database
SELECT 'CREATE DATABASE statgovernance' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'statgovernance') \gexec

-- Password injected from environment (GOVERNANCE_DB_PASSWORD), never hardcoded.
\getenv GOVERNANCE_PW GOVERNANCE_DB_PASSWORD
SELECT format('CREATE ROLE "StatGovernance" WITH LOGIN PASSWORD %L', :'GOVERNANCE_PW')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'StatGovernance') \gexec

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE statgovernance TO "StatGovernance";
GRANT ALL PRIVILEGES ON DATABASE statgovernance TO statgate;

-- Connect to statgovernance database and setup schema
\c statgovernance

-- Grant schema-level permissions
GRANT ALL ON SCHEMA public TO "StatGovernance";
CREATE SCHEMA IF NOT EXISTS statgovernance AUTHORIZATION "StatGovernance";
ALTER DEFAULT PRIVILEGES IN SCHEMA statgovernance GRANT ALL ON TABLES TO "StatGovernance";
ALTER DEFAULT PRIVILEGES IN SCHEMA statgovernance GRANT ALL ON SEQUENCES TO "StatGovernance";
ALTER ROLE "StatGovernance" SET search_path TO statgovernance, public;

-- Also allow statgate user access
GRANT ALL ON SCHEMA statgovernance TO statgate;
ALTER DEFAULT PRIVILEGES IN SCHEMA statgovernance GRANT ALL ON TABLES TO statgate;
ALTER DEFAULT PRIVILEGES IN SCHEMA statgovernance GRANT ALL ON SEQUENCES TO statgate;

-- ─── 1. Policy Management ────────────────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.policies (
    id                  VARCHAR(36)   PRIMARY KEY,
    policy_number       VARCHAR(50)   UNIQUE NOT NULL,
    title               VARCHAR(255)  NOT NULL,
    summary             TEXT,
    content             TEXT          NOT NULL,
    category            VARCHAR(100)  NOT NULL, -- Governance, Data, Security, Operations, Ethics, Financial, HR
    scope               VARCHAR(100)  DEFAULT 'Organization-Wide', -- Organization-Wide, Departmental, Project, Facility
    classification      VARCHAR(50)   DEFAULT 'Internal', -- Public, Internal, Confidential, Restricted
    version             VARCHAR(20)   NOT NULL DEFAULT '1.0',
    status              VARCHAR(50)   NOT NULL DEFAULT 'Draft', -- Draft, Under Review, Approval, Approved, Published, Active, Under Revision, Superseded, Archived
    owner               VARCHAR(255)  NOT NULL,
    owner_id            VARCHAR(36),
    department          VARCHAR(100)  NOT NULL,
    effective_date      DATE,
    review_date         DATE,
    expiry_date         DATE,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    org_id              VARCHAR(50),
    related_regulations TEXT[],
    related_risks       TEXT[],
    related_controls    TEXT[],
    related_projects    TEXT[],
    related_research    TEXT[],
    related_datasets    TEXT[],
    approved_by         VARCHAR(255),
    approved_at         TIMESTAMP,
    published_at        TIMESTAMP,
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statgovernance.policy_versions (
    id                  VARCHAR(36)   PRIMARY KEY,
    policy_id           VARCHAR(36)   REFERENCES statgovernance.policies(id) ON DELETE CASCADE,
    version             VARCHAR(20)   NOT NULL,
    title               VARCHAR(255)  NOT NULL,
    content             TEXT          NOT NULL,
    change_summary      TEXT,
    changed_by          VARCHAR(255)  NOT NULL,
    approved_by         VARCHAR(255),
    status              VARCHAR(50)   NOT NULL,
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 2. Standard Operating Procedures (SOPs) ─────────────────
CREATE TABLE IF NOT EXISTS statgovernance.sops (
    id                  VARCHAR(36)   PRIMARY KEY,
    sop_number          VARCHAR(50)   UNIQUE NOT NULL,
    title               VARCHAR(255)  NOT NULL,
    process             VARCHAR(255)  NOT NULL,
    purpose             TEXT          NOT NULL,
    scope               TEXT          NOT NULL,
    responsibilities    TEXT,
    procedure_steps     JSONB         DEFAULT '[]'::jsonb,
    required_evidence   TEXT[],
    related_policy_id   VARCHAR(36)   REFERENCES statgovernance.policies(id) ON DELETE SET NULL,
    related_control_id  VARCHAR(36),
    review_frequency    VARCHAR(50)   DEFAULT 'Annual',
    version             VARCHAR(20)   DEFAULT '1.0',
    status              VARCHAR(50)   DEFAULT 'Active',
    owner               VARCHAR(255)  NOT NULL,
    department          VARCHAR(100)  NOT NULL,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    approved_by         VARCHAR(255),
    approved_at         TIMESTAMP,
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 3. Regulations & Compliance Obligations ─────────────────
CREATE TABLE IF NOT EXISTS statgovernance.regulations (
    id                  VARCHAR(36)   PRIMARY KEY,
    code                VARCHAR(50)   UNIQUE NOT NULL,
    name                VARCHAR(255)  NOT NULL,
    regulatory_authority VARCHAR(255) NOT NULL,
    jurisdiction        VARCHAR(100)  DEFAULT 'National',
    category            VARCHAR(100)  NOT NULL, -- Data Protection, Public Health, Statistics, Cybersecurity, Financial
    description         TEXT,
    source_url          TEXT,
    effective_date      DATE,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statgovernance.compliance_obligations (
    id                  VARCHAR(36)   PRIMARY KEY,
    obligation_code     VARCHAR(50)   UNIQUE NOT NULL,
    regulation_id       VARCHAR(36)   REFERENCES statgovernance.regulations(id) ON DELETE CASCADE,
    requirement         TEXT          NOT NULL,
    applicable_scope    VARCHAR(255),
    responsible_dept    VARCHAR(100)  NOT NULL,
    responsible_person  VARCHAR(255)  NOT NULL,
    frequency           VARCHAR(50)   DEFAULT 'Continuous', -- Continuous, Quarterly, Bi-Annual, Annual, Ad-Hoc
    compliance_status   VARCHAR(50)   NOT NULL DEFAULT 'Compliant', -- Not Assessed, Compliant, Partially Compliant, Non-Compliant, Not Applicable, Under Review
    evidence_requirement TEXT,
    evidence_location   TEXT,
    violation_risk      VARCHAR(50)   DEFAULT 'High', -- Critical, High, Medium, Low
    corrective_action   TEXT,
    deadline            DATE,
    review_date         DATE,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statgovernance.compliance_assessments (
    id                  VARCHAR(36)   PRIMARY KEY,
    assessment_code     VARCHAR(50)   UNIQUE NOT NULL,
    title               VARCHAR(255)  NOT NULL,
    scope               TEXT          NOT NULL,
    assessor            VARCHAR(255)  NOT NULL,
    assessor_id         VARCHAR(36),
    status              VARCHAR(50)   DEFAULT 'In Progress', -- Created, Assigned, Evidence Collection, Assessment, Review, Approval, Completed
    score_percentage    FLOAT         DEFAULT 0.0,
    findings_count      INTEGER       DEFAULT 0,
    recommendations     TEXT,
    start_date          DATE,
    completion_date     DATE,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statgovernance.assessment_items (
    id                  VARCHAR(36)   PRIMARY KEY,
    assessment_id       VARCHAR(36)   REFERENCES statgovernance.compliance_assessments(id) ON DELETE CASCADE,
    obligation_id       VARCHAR(36)   REFERENCES statgovernance.compliance_obligations(id) ON DELETE CASCADE,
    status              VARCHAR(50)   NOT NULL, -- Compliant, Partially Compliant, Non-Compliant, Not Applicable
    notes               TEXT,
    evidence_ref        VARCHAR(255),
    findings            TEXT,
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 4. Risk Management & Register ───────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.risks (
    id                  VARCHAR(36)   PRIMARY KEY,
    risk_code           VARCHAR(50)   UNIQUE NOT NULL,
    title               VARCHAR(255)  NOT NULL,
    description         TEXT          NOT NULL,
    category            VARCHAR(100)  NOT NULL, -- Strategic, Operational, Compliance, Information Security, Financial, Reputational, Data Integrity
    owner               VARCHAR(255)  NOT NULL,
    owner_id            VARCHAR(36),
    organization        VARCHAR(255)  DEFAULT 'StatGate Authority',
    department          VARCHAR(100),
    project_id          VARCHAR(36),
    research_id         VARCHAR(36),
    probability         INTEGER       NOT NULL CHECK (probability >= 1 AND probability <= 5), -- 1 (Rare) to 5 (Almost Certain)
    impact              INTEGER       NOT NULL CHECK (impact >= 1 AND impact <= 5),           -- 1 (Insignificant) to 5 (Catastrophic)
    inherent_risk_score INTEGER       GENERATED ALWAYS AS (probability * impact) STORED,
    inherent_risk_level VARCHAR(20)   NOT NULL, -- Low (1-4), Medium (5-9), High (10-14), Critical (15-25)
    treatment_strategy  VARCHAR(50)   DEFAULT 'Mitigate', -- Mitigate, Avoid, Transfer, Accept
    mitigation_actions  TEXT,
    residual_probability INTEGER      CHECK (residual_probability >= 1 AND residual_probability <= 5),
    residual_impact     INTEGER       CHECK (residual_impact >= 1 AND residual_impact <= 5),
    residual_risk_score INTEGER,
    residual_risk_level VARCHAR(20),
    status              VARCHAR(50)   NOT NULL DEFAULT 'Active', -- Identified, Assessed, In Treatment, Monitored, Closed
    target_date         DATE,
    review_date         DATE,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statgovernance.risk_treatments (
    id                  VARCHAR(36)   PRIMARY KEY,
    risk_id             VARCHAR(36)   REFERENCES statgovernance.risks(id) ON DELETE CASCADE,
    title               VARCHAR(255)  NOT NULL,
    description         TEXT,
    action_type         VARCHAR(50)   DEFAULT 'Preventive',
    responsible_person  VARCHAR(255)  NOT NULL,
    due_date            DATE,
    status              VARCHAR(50)   DEFAULT 'Pending', -- Pending, In Progress, Completed, Verified
    enterprise_task_id  VARCHAR(50),
    completion_date     DATE,
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 5. Control Library & Testing ────────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.controls (
    id                  VARCHAR(36)   PRIMARY KEY,
    control_code        VARCHAR(50)   UNIQUE NOT NULL,
    name                VARCHAR(255)  NOT NULL,
    description         TEXT          NOT NULL,
    objective           TEXT          NOT NULL,
    owner               VARCHAR(255)  NOT NULL,
    department          VARCHAR(100)  NOT NULL,
    control_type        VARCHAR(50)   NOT NULL, -- Preventive, Detective, Corrective, Directive, Compensating
    frequency           VARCHAR(50)   NOT NULL DEFAULT 'Monthly', -- Continuous, Daily, Weekly, Monthly, Quarterly, Annual
    test_method         VARCHAR(100)  DEFAULT 'Automated Probe', -- Inspection, Observation, Inquiry, Reperformance, Automated Probe
    evidence_requirement TEXT,
    effectiveness       VARCHAR(50)   NOT NULL DEFAULT 'Effective', -- Effective, Partially Effective, Ineffective, Not Tested
    related_risk_id     VARCHAR(36)   REFERENCES statgovernance.risks(id) ON DELETE SET NULL,
    related_policy_id   VARCHAR(36)   REFERENCES statgovernance.policies(id) ON DELETE SET NULL,
    last_tested         TIMESTAMP,
    next_test_date      DATE,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statgovernance.control_tests (
    id                  VARCHAR(36)   PRIMARY KEY,
    control_id          VARCHAR(36)   REFERENCES statgovernance.controls(id) ON DELETE CASCADE,
    tester              VARCHAR(255)  NOT NULL,
    test_date           TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    result              VARCHAR(50)   NOT NULL, -- Pass, Partial Pass, Fail
    effectiveness       VARCHAR(50)   NOT NULL, -- Effective, Partially Effective, Ineffective
    findings            TEXT,
    evidence_ref        VARCHAR(255),
    remediation_task_id VARCHAR(50),
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 6. Audit Management & Findings ──────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.audits (
    id                  VARCHAR(36)   PRIMARY KEY,
    audit_code          VARCHAR(50)   UNIQUE NOT NULL,
    title               VARCHAR(255)  NOT NULL,
    audit_type          VARCHAR(100)  NOT NULL, -- Internal Audit, External Compliance, ISO/IEC 27001, Data Protection, Financial, Operational
    scope               TEXT          NOT NULL,
    objectives          TEXT          NOT NULL,
    lead_auditor        VARCHAR(255)  NOT NULL,
    auditors            TEXT[],
    auditees            TEXT[],
    department          VARCHAR(100)  NOT NULL,
    start_date          DATE,
    end_date            DATE,
    status              VARCHAR(50)   NOT NULL DEFAULT 'Planned', -- Planned, Scheduled, In Progress, Findings, Management Response, Corrective Action, Verification, Closed
    findings_count      INTEGER       DEFAULT 0,
    critical_findings   INTEGER       DEFAULT 0,
    summary             TEXT,
    report_file_id      VARCHAR(50),
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statgovernance.audit_findings (
    id                  VARCHAR(36)   PRIMARY KEY,
    finding_code        VARCHAR(50)   UNIQUE NOT NULL,
    audit_id            VARCHAR(36)   REFERENCES statgovernance.audits(id) ON DELETE CASCADE,
    title               VARCHAR(255)  NOT NULL,
    description         TEXT          NOT NULL,
    severity            VARCHAR(50)   NOT NULL, -- Critical, High, Medium, Low, Observation
    source              VARCHAR(100)  DEFAULT 'Audit', -- Audit, Compliance Assessment, Control Test, Incident, Whistleblower
    owner               VARCHAR(255)  NOT NULL,
    department          VARCHAR(100)  NOT NULL,
    policy_id           VARCHAR(36)   REFERENCES statgovernance.policies(id) ON DELETE SET NULL,
    control_id          VARCHAR(36)   REFERENCES statgovernance.controls(id) ON DELETE SET NULL,
    risk_id             VARCHAR(36)   REFERENCES statgovernance.risks(id) ON DELETE SET NULL,
    recommendation      TEXT          NOT NULL,
    management_response TEXT,
    due_date            DATE          NOT NULL,
    status              VARCHAR(50)   NOT NULL DEFAULT 'Open', -- Open, In Remediation, Remediation Submitted, Verified, Closed
    enterprise_task_id  VARCHAR(50),
    closure_evidence    TEXT,
    closed_by           VARCHAR(255),
    closed_at           TIMESTAMP,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 7. Corrective & Preventive Actions (CAPA) ────────────────
CREATE TABLE IF NOT EXISTS statgovernance.corrective_actions (
    id                  VARCHAR(36)   PRIMARY KEY,
    action_code         VARCHAR(50)   UNIQUE NOT NULL,
    title               VARCHAR(255)  NOT NULL,
    description         TEXT          NOT NULL,
    source_type         VARCHAR(50)   NOT NULL, -- Finding, Risk, Control Failure, Incident, Assessment
    source_id           VARCHAR(36)   NOT NULL,
    owner               VARCHAR(255)  NOT NULL,
    priority            VARCHAR(50)   NOT NULL DEFAULT 'High', -- Critical, High, Medium, Low
    due_date            DATE          NOT NULL,
    enterprise_task_id  VARCHAR(50),
    status              VARCHAR(50)   NOT NULL DEFAULT 'Pending', -- Pending, In Progress, Completed, Verified, Closed
    completion_date     DATE,
    verification_notes  TEXT,
    verified_by         VARCHAR(255),
    verified_at         TIMESTAMP,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 8. Governance Committees & Meetings ─────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.governance_committees (
    id                  VARCHAR(36)   PRIMARY KEY,
    name                VARCHAR(255)  NOT NULL,
    code                VARCHAR(50)   UNIQUE NOT NULL,
    mandate             TEXT          NOT NULL,
    chairperson         VARCHAR(255)  NOT NULL,
    secretary           VARCHAR(255)  NOT NULL,
    meeting_frequency   VARCHAR(50)   DEFAULT 'Monthly',
    statchat_channel_id VARCHAR(100),
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statgovernance.committee_members (
    id                  VARCHAR(36)   PRIMARY KEY,
    committee_id        VARCHAR(36)   REFERENCES statgovernance.governance_committees(id) ON DELETE CASCADE,
    name                VARCHAR(255)  NOT NULL,
    role                VARCHAR(100)  NOT NULL, -- Chairperson, Secretary, Member, Technical Advisor, Observer
    email               VARCHAR(255)  NOT NULL,
    department          VARCHAR(100),
    joined_date         DATE          DEFAULT CURRENT_DATE,
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statgovernance.governance_meetings (
    id                  VARCHAR(36)   PRIMARY KEY,
    committee_id        VARCHAR(36)   REFERENCES statgovernance.governance_committees(id) ON DELETE CASCADE,
    title               VARCHAR(255)  NOT NULL,
    meeting_date        TIMESTAMP     NOT NULL,
    location            VARCHAR(255)  DEFAULT 'StatGate Boardroom / StatChat Video',
    calendar_event_id   VARCHAR(50),
    status              VARCHAR(50)   DEFAULT 'Scheduled', -- Scheduled, In Progress, Completed, Cancelled
    agenda              TEXT,
    minutes             TEXT,
    attendees           TEXT[],
    decisions_count     INTEGER       DEFAULT 0,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 9. Governance Decisions ─────────────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.governance_decisions (
    id                  VARCHAR(36)   PRIMARY KEY,
    decision_code       VARCHAR(50)   UNIQUE NOT NULL,
    title               VARCHAR(255)  NOT NULL,
    context             TEXT          NOT NULL,
    options_considered  JSONB         DEFAULT '[]'::jsonb,
    risks_evaluated     TEXT,
    recommendation      TEXT,
    final_decision      TEXT          NOT NULL,
    decision_maker      VARCHAR(255)  NOT NULL,
    committee_id        VARCHAR(36)   REFERENCES statgovernance.governance_committees(id) ON DELETE SET NULL,
    meeting_id          VARCHAR(36)   REFERENCES statgovernance.governance_meetings(id) ON DELETE SET NULL,
    policy_id           VARCHAR(36)   REFERENCES statgovernance.policies(id) ON DELETE SET NULL,
    status              VARCHAR(50)   DEFAULT 'Approved', -- Proposed, Under Review, Approved, Implemented, Superseded
    ai_advisory         BOOLEAN       DEFAULT FALSE,
    human_approved      BOOLEAN       DEFAULT TRUE,
    enterprise_decision_id VARCHAR(50),
    effective_date      DATE          DEFAULT CURRENT_DATE,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 10. Evidence Records & Vault ────────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.evidence_records (
    id                  VARCHAR(36)   PRIMARY KEY,
    title               VARCHAR(255)  NOT NULL,
    description         TEXT,
    category            VARCHAR(100)  NOT NULL, -- Audit Report, Test Result, Approval Log, Meeting Minutes, Dataset Lineage, Regulatory Certificate
    source_application  VARCHAR(100)  NOT NULL, -- StatGovernance, PMS, RMS, StatCollect, StatChat, HelpDesk, Registry
    related_entity_type VARCHAR(50)   NOT NULL, -- policy, risk, control, audit, finding, decision, assessment
    related_entity_id   VARCHAR(36)   NOT NULL,
    file_id             VARCHAR(50),  -- References Enterprise Universal File Service
    file_name           VARCHAR(255),
    file_url            TEXT,
    mime_type           VARCHAR(100),
    file_size           BIGINT,
    checksum_sha256     VARCHAR(64),
    classification      VARCHAR(50)   DEFAULT 'Internal',
    version             INTEGER       DEFAULT 1,
    uploaded_by         VARCHAR(255)  NOT NULL,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 11. Data Governance & Privacy ───────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.data_governance_records (
    id                  VARCHAR(36)   PRIMARY KEY,
    dataset_name        VARCHAR(255)  NOT NULL,
    dataset_source      VARCHAR(100)  NOT NULL, -- StatCollect, RMS, Analytics, PMS, Registry
    data_owner          VARCHAR(255)  NOT NULL,
    data_steward        VARCHAR(255)  NOT NULL,
    classification      VARCHAR(50)   NOT NULL DEFAULT 'Confidential', -- Public, Internal, Confidential, Highly Restricted
    sensitivity_level   VARCHAR(50)   NOT NULL DEFAULT 'Medium',       -- Low, Medium, High, Special Category (PII/Health)
    retention_period    VARCHAR(50)   NOT NULL DEFAULT '7 Years',
    access_rules        TEXT          NOT NULL,
    lineage_reference   TEXT,
    quality_threshold   FLOAT         DEFAULT 90.0,
    sharing_agreements  TEXT,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statgovernance.privacy_assessments (
    id                  VARCHAR(36)   PRIMARY KEY,
    activity_name       VARCHAR(255)  NOT NULL,
    data_controller     VARCHAR(255)  NOT NULL,
    lawful_basis        VARCHAR(100)  NOT NULL, -- Consent, Legal Obligation, Public Interest, Vital Interests, Contractual
    personal_data_types TEXT[],
    retention_rules     TEXT          NOT NULL,
    cross_border_transfer BOOLEAN     DEFAULT FALSE,
    dpia_status         VARCHAR(50)   DEFAULT 'Approved', -- Required, In Review, Approved, Mitigations Needed
    breach_records_count INTEGER      DEFAULT 0,
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 12. Delegations of Authority ────────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.delegations (
    id                  VARCHAR(36)   PRIMARY KEY,
    delegator_id        VARCHAR(36)   NOT NULL,
    delegator_name      VARCHAR(255)  NOT NULL,
    delegate_id         VARCHAR(36)   NOT NULL,
    delegate_name       VARCHAR(255)  NOT NULL,
    role_scope          VARCHAR(100)  NOT NULL, -- Acting Officer, Delegated Approver, Temporary Reviewer, Audit Delegate, Risk Owner Delegate
    scope_details       TEXT          NOT NULL,
    start_date          TIMESTAMP     NOT NULL,
    end_date            TIMESTAMP     NOT NULL,
    reason              TEXT          NOT NULL,
    status              VARCHAR(50)   NOT NULL DEFAULT 'Active', -- Active, Expired, Revoked
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── 13. Audit Trails & Logs ─────────────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.audit_logs (
    id                  VARCHAR(36)   PRIMARY KEY,
    entity_type         VARCHAR(50)   NOT NULL,
    entity_id           VARCHAR(36)   NOT NULL,
    action              VARCHAR(100)  NOT NULL,
    actor               VARCHAR(255)  NOT NULL,
    actor_id            VARCHAR(36),
    previous_state      JSONB,
    new_state           JSONB,
    details             TEXT,
    ip_address          VARCHAR(50),
    correlation_id      VARCHAR(100),
    tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
    timestamp           TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── Indexes for Performance & Scalability ───────────────────
CREATE INDEX IF NOT EXISTS idx_policies_status ON statgovernance.policies(status);
CREATE INDEX IF NOT EXISTS idx_policies_tenant ON statgovernance.policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_risks_level ON statgovernance.risks(inherent_risk_level);
CREATE INDEX IF NOT EXISTS idx_risks_tenant ON statgovernance.risks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_controls_effectiveness ON statgovernance.controls(effectiveness);
CREATE INDEX IF NOT EXISTS idx_obligations_status ON statgovernance.compliance_obligations(compliance_status);
CREATE INDEX IF NOT EXISTS idx_findings_severity ON statgovernance.audit_findings(severity);
CREATE INDEX IF NOT EXISTS idx_findings_status ON statgovernance.audit_findings(status);
CREATE INDEX IF NOT EXISTS idx_capa_status ON statgovernance.corrective_actions(status);
CREATE INDEX IF NOT EXISTS idx_delegations_active ON statgovernance.delegations(status, end_date);
CREATE INDEX IF NOT EXISTS idx_evidence_entity ON statgovernance.evidence_records(related_entity_type, related_entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON statgovernance.audit_logs(entity_type, entity_id);

-- ─── 14. Whistleblower Reports (Encrypted & Anonymous) ────────
CREATE TABLE IF NOT EXISTS statgovernance.whistleblower_reports (
    id            VARCHAR(36) PRIMARY KEY,
    title         VARCHAR(255) NOT NULL,
    description   TEXT NOT NULL,
    category      VARCHAR(100) NOT NULL,
    severity      VARCHAR(50) DEFAULT 'Medium',
    is_anonymous  BOOLEAN DEFAULT true,
    status        VARCHAR(50) DEFAULT 'New',
    investigator  VARCHAR(255),
    outcome       TEXT,
    tenant_id     VARCHAR(50) DEFAULT 'tenant-alpha',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ─── 15. Conflict of Interest Declarations ────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.conflict_declarations (
    id              VARCHAR(36) PRIMARY KEY,
    declarant_name  VARCHAR(255) NOT NULL,
    declarant_role  VARCHAR(255) NOT NULL,
    conflict_type   VARCHAR(100) NOT NULL,
    description     TEXT NOT NULL,
    project_name    VARCHAR(255),
    has_conflict    BOOLEAN DEFAULT true,
    mitigation_plan TEXT,
    status          VARCHAR(50) DEFAULT 'Under Review',
    reviewed_by     VARCHAR(255),
    tenant_id       VARCHAR(50) DEFAULT 'tenant-alpha',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ─── 16. Feature Flags ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.feature_flags (
    id          VARCHAR(36) PRIMARY KEY,
    name        VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    environment VARCHAR(50) DEFAULT 'Production',
    is_enabled  BOOLEAN DEFAULT false,
    updated_by  VARCHAR(255),
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ─── 17. Centralized System Parameters ────────────────────────
CREATE TABLE IF NOT EXISTS statgovernance.system_parameters (
    id          VARCHAR(36) PRIMARY KEY,
    key         VARCHAR(100) UNIQUE NOT NULL,
    value       TEXT NOT NULL,
    category    VARCHAR(100) DEFAULT 'General',
    description TEXT,
    updated_by  VARCHAR(255),
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


