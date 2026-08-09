-- ══════════════════════════════════════════════════════════════
-- STATGATE PLATFORM OBJECTS — Projects, Reports, Research
-- ══════════════════════════════════════════════════════════════
-- These tables implement the directive's "Every Object Should Be
-- Connected" principle by providing first-class platform objects
-- that can be linked, discussed, and workflowed.
-- ══════════════════════════════════════════════════════════════

-- ── Projects ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS projects (
    id              VARCHAR(128) PRIMARY KEY,
    title           TEXT NOT NULL,
    description     TEXT,
    status          VARCHAR(32) NOT NULL DEFAULT 'planning',
    -- planning | active | review | completed | archived
    owner_id        VARCHAR(128),
    tenant_id       VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
    collaborators   JSONB NOT NULL DEFAULT '[]',
    tags            JSONB NOT NULL DEFAULT '[]',
    start_date      DATE,
    end_date        DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_projects_tenant ON projects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_owner  ON projects(owner_id);

-- ── Reports ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS reports (
    id              VARCHAR(128) PRIMARY KEY,
    title           TEXT NOT NULL,
    content         TEXT,
    summary         TEXT,
    status          VARCHAR(32) NOT NULL DEFAULT 'draft',
    -- draft | submitted | approved | published | archived
    author_id       VARCHAR(128),
    reviewer_id     VARCHAR(128),
    tenant_id       VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
    project_id      VARCHAR(128),
    version_tag     VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    tags            JSONB NOT NULL DEFAULT '[]',
    approved_at     TIMESTAMPTZ,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_reports_tenant   ON reports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_reports_status   ON reports(status);
CREATE INDEX IF NOT EXISTS idx_reports_author   ON reports(author_id);
CREATE INDEX IF NOT EXISTS idx_reports_project  ON reports(project_id);

-- ── Research Studies ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS research_studies (
    id                  VARCHAR(128) PRIMARY KEY,
    title               TEXT NOT NULL,
    protocol            TEXT,
    description         TEXT,
    status              VARCHAR(32) NOT NULL DEFAULT 'proposed',
    -- proposed | approved | active | analysis | published | archived
    principal_investigator VARCHAR(128),
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
    project_id          VARCHAR(128),
    ethics_status       VARCHAR(32) NOT NULL DEFAULT 'pending',
    -- pending | approved | exempt | rejected
    data_collection_status VARCHAR(32) NOT NULL DEFAULT 'not_started',
    -- not_started | in_progress | completed
    collaborators       JSONB NOT NULL DEFAULT '[]',
    tags                JSONB NOT NULL DEFAULT '[]',
    start_date          DATE,
    end_date            DATE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_research_tenant  ON research_studies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_research_status  ON research_studies(status);
CREATE INDEX IF NOT EXISTS idx_research_pi      ON research_studies(principal_investigator);
CREATE INDEX IF NOT EXISTS idx_research_project ON research_studies(project_id);