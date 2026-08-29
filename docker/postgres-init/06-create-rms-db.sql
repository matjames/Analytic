-- ═══════════════════════════════════════════════════════════
--  StatGate Research Management System (RMS)
--  PostgreSQL initialization script
-- ═══════════════════════════════════════════════════════════

-- Create RMS database
SELECT 'CREATE DATABASE rms' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'rms') \gexec

-- Password injected from environment (RMS_DB_PASSWORD), never hardcoded.
\getenv RMS_PW RMS_DB_PASSWORD
SELECT format('CREATE ROLE "RMS" WITH LOGIN PASSWORD %L', :'RMS_PW')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'RMS') \gexec

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE rms TO "RMS";

-- Connect to rms database and setup schema
\c rms

-- Grant schema-level permissions
GRANT ALL ON SCHEMA public TO "RMS";
CREATE SCHEMA IF NOT EXISTS rms AUTHORIZATION "RMS";
ALTER DEFAULT PRIVILEGES IN SCHEMA rms GRANT ALL ON TABLES TO "RMS";
ALTER DEFAULT PRIVILEGES IN SCHEMA rms GRANT ALL ON SEQUENCES TO "RMS";
ALTER ROLE "RMS" SET search_path TO rms, public;

-- ─── Research Projects ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.research_projects (
    id                 VARCHAR(36)   PRIMARY KEY,
    code               VARCHAR(50)   UNIQUE NOT NULL,
    name               VARCHAR(255)  NOT NULL,
    description        TEXT,
    type               VARCHAR(100)  DEFAULT 'Research Project',
    stage              VARCHAR(100)  NOT NULL DEFAULT 'Research Idea',
    progress           FLOAT         DEFAULT 0.0,
    principal_investigator VARCHAR(255),
    owner              VARCHAR(255),
    organisation       VARCHAR(255),
    portfolio          VARCHAR(255),
    programme          VARCHAR(255),
    start_date         DATE,
    end_date           DATE,
    target_geo         TEXT,
    budget_total       FLOAT         DEFAULT 0.0,
    spent_total        FLOAT         DEFAULT 0.0,
    tags               TEXT[],
    pms_project_id     VARCHAR(36),
    statchat_room_id   VARCHAR(255),
    created_time       TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_time       TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- ─── Research Members ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.research_members (
    id           VARCHAR(36)  PRIMARY KEY,
    research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    role         VARCHAR(255),
    email        VARCHAR(255),
    avatar_url   TEXT,
    department   VARCHAR(255),
    phone        VARCHAR(50),
    location     VARCHAR(255),
    since        DATE,
    created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Proposals ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.proposals (
    id                 VARCHAR(36)  PRIMARY KEY,
    research_id        VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    title              VARCHAR(255) NOT NULL,
    version            VARCHAR(50)  DEFAULT '1.0',
    status             VARCHAR(50)  DEFAULT 'Draft',
    description        TEXT,
    draft_content      TEXT,
    background         TEXT,
    objectives         TEXT,
    methodology        TEXT,
    timeline_details   TEXT,
    budget_details     TEXT,
    reviewer_comments  TEXT,
    submitted_by       VARCHAR(255),
    reviewed_by        VARCHAR(255),
    approved_by        VARCHAR(255),
    submitted_at       TIMESTAMP,
    approved_at        TIMESTAMP,
    created_time       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_time       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Ethics Applications ───────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.ethics_applications (
    id                 VARCHAR(36)  PRIMARY KEY,
    research_id        VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    irb_name           VARCHAR(255) NOT NULL,
    status             VARCHAR(50)  DEFAULT 'Pending',
    submission_date    DATE,
    approval_date      DATE,
    expiry_date        DATE,
    certificate_number VARCHAR(100),
    comments           TEXT,
    amendment_notes    TEXT,
    renewal_notes      TEXT,
    compliance_notes   TEXT,
    created_time       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_time       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Grants / Funding ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.grants (
    id                 VARCHAR(36)  PRIMARY KEY,
    research_id        VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    opportunity_name   VARCHAR(255) NOT NULL,
    status             VARCHAR(50)  DEFAULT 'Applied',
    donor_name         VARCHAR(255),
    contract_number    VARCHAR(100),
    budget_allocated   FLOAT        DEFAULT 0.0,
    spent              FLOAT        DEFAULT 0.0,
    currency           VARCHAR(20)  DEFAULT 'USD',
    start_date         DATE,
    end_date           DATE,
    reporting_schedule TEXT,
    deliverables       TEXT,
    notes              TEXT,
    created_time       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_time       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Literature Library ────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.literature (
    id             VARCHAR(36)  PRIMARY KEY,
    research_id    VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    title          VARCHAR(255) NOT NULL,
    authors        VARCHAR(255),
    journal        VARCHAR(255),
    doi            VARCHAR(100),
    pub_year       INTEGER,
    citation       TEXT,
    keywords       VARCHAR(255),
    category       VARCHAR(100),
    tags           TEXT[],
    notes          TEXT,
    url            TEXT,
    reading_status VARCHAR(50)  DEFAULT 'Unread',
    created_time   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Datasets ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.datasets (
    id               VARCHAR(36)  PRIMARY KEY,
    research_id      VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    description      TEXT,
    version          VARCHAR(50)  DEFAULT '1.0',
    status           VARCHAR(50)  DEFAULT 'Draft',
    source_type      VARCHAR(100),
    collection_method VARCHAR(100),
    metadata_info    TEXT,
    variables_dict   TEXT,
    access_level     VARCHAR(50)  DEFAULT 'Internal',
    download_url     TEXT,
    statcollect_id   VARCHAR(255),
    created_time     TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_time     TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Publications ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.publications (
    id                  VARCHAR(36)  PRIMARY KEY,
    research_id         VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    title               VARCHAR(255) NOT NULL,
    pub_type            VARCHAR(100) DEFAULT 'Manuscript',
    authors             VARCHAR(255),
    journal             VARCHAR(255),
    status              VARCHAR(50)  DEFAULT 'Drafting',
    peer_review_comments TEXT,
    revision_history    TEXT,
    doi                 VARCHAR(100),
    acceptance_date     DATE,
    published_date      DATE,
    affiliation         VARCHAR(255),
    created_time        TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_time        TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Research Tasks ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.tasks (
    id           VARCHAR(36)  PRIMARY KEY,
    research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    title        VARCHAR(255) NOT NULL,
    description  TEXT,
    status       VARCHAR(50)  DEFAULT 'Todo',
    priority     VARCHAR(50)  DEFAULT 'Medium',
    start_date   DATE,
    end_date     DATE,
    progress     FLOAT        DEFAULT 0.0,
    assigned_to  VARCHAR(255),
    created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Meetings ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.meetings (
    id           VARCHAR(36)  PRIMARY KEY,
    research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    title        VARCHAR(255) NOT NULL,
    date_time    TIMESTAMP,
    location     VARCHAR(255),
    agenda       TEXT,
    decisions    TEXT[],
    attendees    TEXT[],
    status       VARCHAR(50)  DEFAULT 'Scheduled',
    created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Risks ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.risks (
    id           VARCHAR(36)  PRIMARY KEY,
    research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    description  TEXT         NOT NULL,
    status       VARCHAR(50)  DEFAULT 'Open',
    impact       VARCHAR(50)  DEFAULT 'Medium',
    likelihood   VARCHAR(50)  DEFAULT 'Medium',
    mitigation   TEXT,
    due_date     DATE,
    owner        VARCHAR(255),
    created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Chat Messages ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.chat_messages (
    id           VARCHAR(36)  PRIMARY KEY,
    research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    sender       VARCHAR(255) NOT NULL,
    channel      VARCHAR(100) DEFAULT 'general',
    message      TEXT         NOT NULL,
    created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Audit Logs ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.audit_logs (
    id           VARCHAR(36)  PRIMARY KEY,
    research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
    action       VARCHAR(255) NOT NULL,
    performed_by VARCHAR(255) NOT NULL,
    details      TEXT,
    created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ─── Ethics Committees / IRB ───────────────────────────────
CREATE TABLE IF NOT EXISTS rms.ethics_committees (
    id                VARCHAR(36) PRIMARY KEY,
    name              VARCHAR(255) NOT NULL,
    code              VARCHAR(50) UNIQUE,
    institution       VARCHAR(255),
    chair_person      VARCHAR(255),
    email             VARCHAR(255),
    phone             VARCHAR(50),
    approval_validity INTEGER DEFAULT 12,
    status            VARCHAR(50) DEFAULT 'Active',
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ─── DOI Records ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.doi_records (
    id          VARCHAR(36) PRIMARY KEY,
    research_id VARCHAR(36) REFERENCES rms.research_projects(id) ON DELETE SET NULL,
    doi         VARCHAR(100) UNIQUE NOT NULL,
    title       VARCHAR(255) NOT NULL,
    authors     VARCHAR(255),
    journal     VARCHAR(255),
    year        INTEGER,
    url         TEXT,
    abstract    TEXT,
    keywords    TEXT,
    output_type VARCHAR(100) DEFAULT 'Journal Article',
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ─── Open Access Repository ─────────────────────────────────
CREATE TABLE IF NOT EXISTS rms.open_access_repo (
    id            VARCHAR(36) PRIMARY KEY,
    research_id   VARCHAR(36) REFERENCES rms.research_projects(id) ON DELETE SET NULL,
    title         VARCHAR(255) NOT NULL,
    description   TEXT,
    resource_type VARCHAR(100) DEFAULT 'Dataset',
    url           TEXT NOT NULL,
    license       VARCHAR(100) DEFAULT 'CC BY 4.0',
    access_level  VARCHAR(50) DEFAULT 'Open',
    keywords      TEXT,
    repo_name     VARCHAR(100),
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Runtime migrations connect as RMS, so bootstrap objects must be owned by it.
DO $$
DECLARE
    obj RECORD;
BEGIN
    FOR obj IN
        SELECT c.relkind, n.nspname, c.relname
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'rms' AND c.relkind IN ('r', 'p', 'S')
    LOOP
        IF obj.relkind = 'S' THEN
            EXECUTE format('ALTER SEQUENCE %I.%I OWNER TO %I', obj.nspname, obj.relname, 'RMS');
        ELSE
            EXECUTE format('ALTER TABLE %I.%I OWNER TO %I', obj.nspname, obj.relname, 'RMS');
        END IF;
    END LOOP;
END $$;

