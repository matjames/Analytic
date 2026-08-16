-- ═══════════════════════════════════════════════════════════════════
-- PHASE 15 — ENTERPRISE DOCUMENT MANAGEMENT (EDMS), RECORDS & ARCHIVES
-- ═══════════════════════════════════════════════════════════════════

-- ─── Folder & Repository Structure ─────────────────────────────────

CREATE TABLE IF NOT EXISTS document_repositories (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    repository_type VARCHAR(64) DEFAULT 'General',  -- 'Policies','Research','Publications','Contracts','Legal','Media'
    access_level VARCHAR(32) DEFAULT 'INTERNAL',
    owner_department VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS document_folders (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    path VARCHAR(512) NOT NULL UNIQUE,
    parent_path VARCHAR(512),
    repository_id VARCHAR(64) REFERENCES document_repositories(id) ON DELETE SET NULL,
    description TEXT,
    access_level VARCHAR(32) DEFAULT 'INTERNAL',
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Document Repository ────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS enterprise_documents (
    id VARCHAR(64) PRIMARY KEY,
    document_number VARCHAR(64) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    folder_path VARCHAR(512) DEFAULT '/',
    folder_id VARCHAR(64) REFERENCES document_folders(id) ON DELETE SET NULL,
    category VARCHAR(64) NOT NULL,
    version VARCHAR(32) DEFAULT 'v1.0',
    author_name VARCHAR(255) NOT NULL,
    department VARCHAR(255),
    classification VARCHAR(32) DEFAULT 'INTERNAL',   -- PUBLIC, INTERNAL, CONFIDENTIAL, RESTRICTED, TOP_SECRET
    status VARCHAR(32) DEFAULT 'Draft',              -- Draft, In_Review, Approved, Published, Archived, Legal_Hold
    file_size_kb INT DEFAULT 0,
    checkout_user VARCHAR(255),
    checkout_at TIMESTAMP WITH TIME ZONE,
    sha256_hash VARCHAR(128) NOT NULL,
    signed_by JSONB DEFAULT '[]'::jsonb,
    tags JSONB DEFAULT '[]'::jsonb,
    content TEXT,
    watermark TEXT,
    expiry_date DATE,
    download_count INT DEFAULT 0,
    view_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_enterprise_documents_folder ON enterprise_documents(folder_path);
CREATE INDEX IF NOT EXISTS idx_enterprise_documents_status ON enterprise_documents(status);
CREATE INDEX IF NOT EXISTS idx_enterprise_documents_category ON enterprise_documents(category);

-- ─── Version History ────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS document_versions (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    version_number VARCHAR(32) NOT NULL,
    change_summary TEXT,
    author_name VARCHAR(255),
    content_snapshot TEXT,
    sha256_hash VARCHAR(128) NOT NULL,
    file_size_kb INT DEFAULT 0,
    is_major_version BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Document Metadata ──────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS document_metadata (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    key VARCHAR(128) NOT NULL,
    value TEXT NOT NULL,
    data_type VARCHAR(32) DEFAULT 'string',  -- string, number, date, boolean
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_doc_metadata_docid ON document_metadata(document_id);

-- ─── Document Tags ──────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS document_tags (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) UNIQUE NOT NULL,
    category VARCHAR(64),
    color VARCHAR(16) DEFAULT '#6366f1',
    usage_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Document Classifications ───────────────────────────────────────

CREATE TABLE IF NOT EXISTS document_classifications (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    classification_level VARCHAR(32) NOT NULL,  -- PUBLIC, INTERNAL, CONFIDENTIAL, RESTRICTED
    classification_reason TEXT,
    classified_by VARCHAR(255),
    review_date DATE,
    declassification_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Document Comments ──────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS document_comments (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    author_name VARCHAR(255) NOT NULL,
    author_role VARCHAR(128),
    comment TEXT NOT NULL,
    comment_type VARCHAR(32) DEFAULT 'General',  -- General, Review, Approval, Annotation
    resolved BOOLEAN DEFAULT FALSE,
    parent_id VARCHAR(64),  -- for threaded replies
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Standardized Templates ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS document_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(64) NOT NULL,
    description TEXT,
    default_content TEXT NOT NULL,
    tags JSONB DEFAULT '[]'::jsonb,
    usage_count INT DEFAULT 0,
    is_official BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Digital Signatures & Cryptographic Audit Stamps ───────────────

CREATE TABLE IF NOT EXISTS digital_signatures (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    document_number VARCHAR(64) NOT NULL,
    signer_name VARCHAR(255) NOT NULL,
    signer_role VARCHAR(255) NOT NULL,
    signature_type VARCHAR(64) NOT NULL,  -- Executive_Approval, Ethics_Clearance, Financial_Signoff, Chief_Statistician
    verification_hash VARCHAR(128) NOT NULL,
    qr_verification_url TEXT NOT NULL,
    certificate_chain TEXT,
    signed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Approval History ───────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS document_approval_history (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    document_number VARCHAR(64) NOT NULL,
    action VARCHAR(32) NOT NULL,  -- Submitted, Approved, Rejected, Recalled, Escalated
    actor_name VARCHAR(255) NOT NULL,
    actor_role VARCHAR(128),
    comments TEXT,
    previous_status VARCHAR(32),
    new_status VARCHAR(32),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Document Permissions ───────────────────────────────────────────

CREATE TABLE IF NOT EXISTS document_permissions (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    grantee_type VARCHAR(32) NOT NULL,  -- User, Role, Department, Public
    grantee_id VARCHAR(128) NOT NULL,
    permissions JSONB DEFAULT '["read"]'::jsonb,  -- ["read","write","sign","delete","share"]
    granted_by VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── ISO 15489 Records Retention Schedules ─────────────────────────

CREATE TABLE IF NOT EXISTS retention_policies (
    id VARCHAR(64) PRIMARY KEY,
    category VARCHAR(255) NOT NULL,
    retention_period_years INT NOT NULL,
    disposition_action VARCHAR(64) NOT NULL,  -- Permanent_National_Archive, Review_For_Destruction, Declassify_To_Public
    legal_authority VARCHAR(255),
    review_cycle_years INT DEFAULT 5,
    applies_to_categories JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Legal Holds ────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS legal_holds (
    id VARCHAR(64) PRIMARY KEY,
    case_reference VARCHAR(64) UNIQUE NOT NULL,
    reason TEXT NOT NULL,
    custodian VARCHAR(255) NOT NULL,
    issued_by VARCHAR(255),
    issued_date DATE DEFAULT CURRENT_DATE,
    expected_release_date DATE,
    status VARCHAR(32) DEFAULT 'Active',  -- Active, Released, Under_Review
    scope_description TEXT,
    affected_departments JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Archive Management ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS archives (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    archive_type VARCHAR(64) DEFAULT 'Operational',  -- Operational, Long_Term, Cold_Storage, Cloud, Disaster_Recovery
    location_description TEXT,
    retention_policy_id VARCHAR(64) REFERENCES retention_policies(id) ON DELETE SET NULL,
    document_count INT DEFAULT 0,
    size_mb NUMERIC(12,2) DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS archive_locations (
    id VARCHAR(64) PRIMARY KEY,
    archive_id VARCHAR(64) REFERENCES archives(id) ON DELETE CASCADE,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    archived_by VARCHAR(255),
    archive_reason TEXT,
    disposition_date DATE,
    disposition_action VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Disposition Workflows ──────────────────────────────────────────

CREATE TABLE IF NOT EXISTS disposition_workflows (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    retention_policy_id VARCHAR(64) REFERENCES retention_policies(id) ON DELETE SET NULL,
    scheduled_disposition_date DATE NOT NULL,
    disposition_action VARCHAR(64) NOT NULL,
    status VARCHAR(32) DEFAULT 'Scheduled',  -- Scheduled, Under_Review, Approved, Executed, Cancelled
    reviewed_by VARCHAR(255),
    approved_by VARCHAR(255),
    executed_at TIMESTAMP WITH TIME ZONE,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Knowledge Articles & Wiki ──────────────────────────────────────

CREATE TABLE IF NOT EXISTS knowledge_articles (
    id VARCHAR(64) PRIMARY KEY,
    article_number VARCHAR(64) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    article_type VARCHAR(64) DEFAULT 'Knowledge_Article',  -- Knowledge_Article, Policy, SOP, FAQ, Procedure, Research_Note
    content TEXT NOT NULL,
    summary TEXT,
    author_name VARCHAR(255) NOT NULL,
    department VARCHAR(255),
    status VARCHAR(32) DEFAULT 'Draft',  -- Draft, Published, Archived
    tags JSONB DEFAULT '[]'::jsonb,
    related_document_ids JSONB DEFAULT '[]'::jsonb,
    view_count INT DEFAULT 0,
    helpful_votes INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS wiki_pages (
    id VARCHAR(64) PRIMARY KEY,
    page_number VARCHAR(64) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    content TEXT NOT NULL,
    parent_slug VARCHAR(255),
    author_name VARCHAR(255) NOT NULL,
    status VARCHAR(32) DEFAULT 'Published',
    tags JSONB DEFAULT '[]'::jsonb,
    view_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── OCR Results ────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS ocr_results (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    extracted_text TEXT,
    confidence_score NUMERIC(5,2),
    language_detected VARCHAR(32),
    page_count INT DEFAULT 1,
    processing_status VARCHAR(32) DEFAULT 'Pending',  -- Pending, Processing, Completed, Failed
    ocr_engine VARCHAR(64) DEFAULT 'Tesseract',
    processing_time_ms INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ─── Full-Text Search Index ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS document_search_index (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) REFERENCES enterprise_documents(id) ON DELETE CASCADE,
    indexed_content TEXT NOT NULL,
    search_vector TSVECTOR,
    last_indexed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_doc_search_vector ON document_search_index USING GIN(search_vector);

-- ─── Media & Digital Assets ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS digital_assets (
    id VARCHAR(64) PRIMARY KEY,
    asset_name VARCHAR(255) NOT NULL,
    asset_type VARCHAR(64) NOT NULL,  -- Image, Audio, Video, GIS_File, Statistical_Output, AI_Model
    file_path TEXT NOT NULL,
    file_size_kb INT DEFAULT 0,
    mime_type VARCHAR(128),
    uploaded_by VARCHAR(255),
    department VARCHAR(255),
    description TEXT,
    tags JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

