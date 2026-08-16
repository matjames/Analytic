-- Create PMS database and schema
CREATE DATABASE pms;

-- Password injected from environment (PMS_DB_PASSWORD), never hardcoded.
\getenv PMS_PW PMS_DB_PASSWORD
SELECT format('CREATE ROLE "PMS" WITH LOGIN PASSWORD %L', :'PMS_PW')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'PMS') \gexec

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE pms TO "PMS";

-- Connect to pms database and setup schema
\c pms

-- Grant schema-level permissions
GRANT ALL ON SCHEMA public TO "PMS";
CREATE SCHEMA IF NOT EXISTS pms AUTHORIZATION "PMS";
ALTER DEFAULT PRIVILEGES IN SCHEMA pms GRANT ALL ON TABLES TO "PMS";
ALTER DEFAULT PRIVILEGES IN SCHEMA pms GRANT ALL ON SEQUENCES TO "PMS";
ALTER ROLE "PMS" SET search_path TO pms, public;

-- Create projects table
CREATE TABLE IF NOT EXISTS pms.projects (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    stage VARCHAR(50) NOT NULL DEFAULT 'Concept',
    progress FLOAT DEFAULT 0.0,
    org VARCHAR(255),
    portfolio VARCHAR(255),
    programme VARCHAR(255),
    owner VARCHAR(255),
    start_date DATE,
    end_date DATE,
    target_geo TEXT,
    tags TEXT[],
    budget_total FLOAT DEFAULT 0.0,
    spent_total FLOAT DEFAULT 0.0,
    risks_count INTEGER DEFAULT 0,
    issues_count INTEGER DEFAULT 0,
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create project members table
CREATE TABLE IF NOT EXISTS pms.project_members (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(255),
    email VARCHAR(255),
    avatar_url TEXT,
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create tasks table
CREATE TABLE IF NOT EXISTS pms.tasks (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    parent_id VARCHAR(36),
    wbs_code VARCHAR(50),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'Todo',
    priority VARCHAR(50) DEFAULT 'Medium',
    start_date DATE,
    end_date DATE,
    progress FLOAT DEFAULT 0.0,
    assigned_to VARCHAR(255),
    dependencies TEXT[],
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create budget lines table
CREATE TABLE IF NOT EXISTS pms.budget_lines (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    category VARCHAR(100),
    description TEXT,
    source VARCHAR(100),
    amount FLOAT DEFAULT 0.0,
    spent FLOAT DEFAULT 0.0,
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create risks table
CREATE TABLE IF NOT EXISTS pms.risks (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    category VARCHAR(100),
    probability VARCHAR(50),
    impact VARCHAR(50),
    mitigation TEXT,
    status VARCHAR(50) DEFAULT 'Active',
    owner VARCHAR(255),
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create documents table
CREATE TABLE IF NOT EXISTS pms.documents (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50),
    size VARCHAR(50),
    uploaded_by VARCHAR(255),
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    url TEXT
);

-- Create meetings table
CREATE TABLE IF NOT EXISTS pms.meetings (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    date_time TIMESTAMP,
    location VARCHAR(255),
    attendees TEXT[],
    agenda TEXT,
    minutes TEXT,
    action_items TEXT[],
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create surveys table
CREATE TABLE IF NOT EXISTS pms.surveys (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'Template',
    target_sample INTEGER DEFAULT 0,
    submissions INTEGER DEFAULT 0,
    progress FLOAT DEFAULT 0.0,
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create chat messages table
CREATE TABLE IF NOT EXISTS pms.chat_messages (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    channel VARCHAR(100) DEFAULT 'general',
    sender VARCHAR(255),
    role VARCHAR(255),
    message TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create helpdesk tickets table
CREATE TABLE IF NOT EXISTS pms.helpdesk_tickets (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'Open',
    priority VARCHAR(50) DEFAULT 'Medium',
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_projects_code ON pms.projects(code);
CREATE INDEX IF NOT EXISTS idx_projects_stage ON pms.projects(stage);
CREATE INDEX IF NOT EXISTS idx_tasks_project ON pms.tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_chat_project ON pms.chat_messages(project_id);
CREATE INDEX IF NOT EXISTS idx_helpdesk_project ON pms.helpdesk_tickets(project_id);

-- Insert seed data
INSERT INTO pms.projects (id, code, name, description, stage, progress, org, portfolio, programme, owner, start_date, end_date, target_geo, tags, budget_total, spent_total, risks_count, issues_count, created_time) VALUES
('proj-1', 'SG-2026-NCD', 'National Non-Communicable Diseases Survey 2026', 'Comprehensive nationwide survey mapping NCD prevalence, risk factors, and healthcare access across all 12 provinces.', 'Implementation', 42.5, 'StatGate Ministry Alliance', 'Public Health Intelligence', 'National Surveys Programme', 'Dr. Sarah Jenkins', '2026-02-15', '2026-11-30', 'National Coverage (All Provinces)', ARRAY['NCD', 'Health', 'Survey', 'National'], 450000, 192500, 2, 1, CURRENT_TIMESTAMP - INTERVAL '120 days'),
('proj-2', 'SG-2026-AGRI', 'Agricultural Productivity & Food Security Census', 'Assessing smallholder farmer yields, irrigation tech adoption, and market access metrics in the northern agricultural belt.', 'Planning', 12.0, 'StatGate Agritech Directorate', 'Economic & Resource Intelligence', 'Rural Development Initiative', 'Marcus Vance', '2026-09-01', '2027-03-31', 'Northern and Eastern Agricultural Zones', ARRAY['Agriculture', 'Food Security', 'Census'], 320000, 15000, 1, 0, CURRENT_TIMESTAMP - INTERVAL '30 days')
ON CONFLICT (id) DO NOTHING;

-- Insert members
INSERT INTO pms.project_members (id, project_id, name, role, email, avatar_url) VALUES
('m-1', 'proj-1', 'Dr. Sarah Jenkins', 'Project Manager', 's.jenkins@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=sarah'),
('m-2', 'proj-1', 'David Chen', 'Research Lead', 'd.chen@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=david'),
('m-3', 'proj-1', 'Amara Oke', 'Finance Officer', 'a.oke@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=amara'),
('m-4', 'proj-1', 'Carlos Gomez', 'Field Supervisor', 'c.gomez@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=carlos'),
('m-5', 'proj-2', 'Marcus Vance', 'Project Manager', 'm.vance@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=marcus'),
('m-6', 'proj-2', 'Elena Rostova', 'Monitoring & Evaluation Officer', 'e.rostova@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=elena')
ON CONFLICT (id) DO NOTHING;

-- Insert tasks
INSERT INTO pms.tasks (id, project_id, wbs_code, title, description, status, priority, start_date, end_date, progress, assigned_to) VALUES
('t1-1', 'proj-1', '1.1', 'Survey Protocol Design', 'Finalize ethics board approvals and design NCD methodology.', 'Done', 'High', '2026-02-15', '2026-03-10', 100, 'David Chen'),
('t1-2', 'proj-1', '1.2', 'Digital Questionnaire Deployment', 'Build digital survey forms in StatCollect framework.', 'Done', 'Medium', '2026-03-11', '2026-04-05', 100, 'David Chen'),
('t1-3', 'proj-1', '2.1', 'Field Staff Training', 'Conduct training seminars for 48 enumerators across regions.', 'Done', 'High', '2026-04-10', '2026-04-25', 100, 'Carlos Gomez'),
('t1-4', 'proj-1', '2.2', 'Data Collection Phase 1', 'Launch household field surveys in Western and Southern provinces.', 'In Progress', 'High', '2026-05-01', '2026-08-30', 75, 'Carlos Gomez'),
('t1-5', 'proj-1', '3.1', 'Interim Analysis & Midterm Report', 'Compile preliminary findings for the Department of Health.', 'Todo', 'Medium', '2026-09-01', '2026-09-30', 0, 'Dr. Sarah Jenkins'),
('t2-1', 'proj-2', '1.1', 'Define Census Scope & Indicators', 'Consult Ministry of Agriculture to lock agricultural indicators.', 'In Progress', 'High', '2026-08-01', '2026-08-25', 50, 'Marcus Vance'),
('t2-2', 'proj-2', '1.2', 'Budget Approval and Fund Allocation', 'Secure formal sign-off for rural development grants.', 'Todo', 'High', '2026-08-26', '2026-09-10', 0, 'Elena Rostova')
ON CONFLICT (id) DO NOTHING;

-- Insert budget lines
INSERT INTO pms.budget_lines (id, project_id, category, description, source, amount, spent) VALUES
('b1-1', 'proj-1', 'Personnel', 'Field enumerator allowances and PM salary', 'State Fund', 220000, 110000),
('b1-2', 'proj-1', 'Travel', 'Fuel, transport hire, logistics for remote visits', 'State Fund', 80000, 45000),
('b1-3', 'proj-1', 'Equipment', 'Tablet PCs for offline data collection', 'WHO Grant', 100000, 32500),
('b1-4', 'proj-1', 'Supplies', 'PPE, training materials, printed documentation', 'WHO Grant', 50000, 5000),
('b2-1', 'proj-2', 'Personnel', 'Enumerator training staff fees', 'FAO Grant', 150000, 5000),
('b2-2', 'proj-2', 'Travel', 'Logistics for northern remote districts', 'FAO Grant', 120000, 10000),
('b2-3', 'proj-2', 'Indirect', 'Operational overhead and mapping software', 'State Fund', 50000, 0)
ON CONFLICT (id) DO NOTHING;

-- Insert risks
INSERT INTO pms.risks (id, project_id, description, category, probability, impact, mitigation, status, owner) VALUES
('r1-1', 'proj-1', 'Delayed access approval in remote districts', 'Operational', 'Medium', 'High', 'Establish coordination with local chieftains and rural clinics early.', 'Active', 'Carlos Gomez'),
('r1-2', 'proj-1', 'Tablet hardware failures in humid field conditions', 'Technical', 'Low', 'Medium', 'Procure rugged waterproof cases and supply field backups.', 'Mitigated', 'David Chen'),
('r2-1', 'proj-2', 'Heavy rain season blocking road access', 'External', 'High', 'High', 'Schedule survey windows to avoid peak monsoon months.', 'Active', 'Marcus Vance')
ON CONFLICT (id) DO NOTHING;

-- Insert documents
INSERT INTO pms.documents (id, project_id, name, type, size, uploaded_by, uploaded_at, url) VALUES
('d1-1', 'proj-1', 'NCD_Survey_Protocol_v2.pdf', 'PDF', '2.4 MB', 'Dr. Sarah Jenkins', CURRENT_TIMESTAMP - INTERVAL '100 days', '#'),
('d1-2', 'proj-1', 'Survey_Budget_Approved_2026.xlsx', 'Excel', '1.1 MB', 'Amara Oke', CURRENT_TIMESTAMP - INTERVAL '95 days', '#'),
('d2-1', 'proj-2', 'Agri_Security_Proposal.pdf', 'PDF', '4.8 MB', 'Marcus Vance', CURRENT_TIMESTAMP - INTERVAL '28 days', '#')
ON CONFLICT (id) DO NOTHING;

-- Insert meetings
INSERT INTO pms.meetings (id, project_id, title, date_time, location, attendees, agenda, minutes, action_items) VALUES
('mt1-1', 'proj-1', 'NCD Mid-Point Progress Check-in', '2026-08-10T10:00:00Z', 'Conference Room B / StatChat Video', ARRAY['Sarah Jenkins', 'David Chen', 'Carlos Gomez'], '1. Review western region submission rates. 2. Address tablet sync issues. 3. Midterm draft timeline.', '', ARRAY['David to push offline patch', 'Carlos to verify region coverage']),
('mt2-1', 'proj-2', 'Stakeholder Alignment Session', '2026-08-18T14:00:00Z', 'Executive Suite', ARRAY['Marcus Vance', 'Elena Rostova'], 'Finalizing indicators for regional agricultural policy matching.', '', ARRAY['Elena to circulate draft list of questions'])
ON CONFLICT (id) DO NOTHING;

-- Insert surveys
INSERT INTO pms.surveys (id, project_id, name, status, target_sample, submissions, progress) VALUES
('s1-1', 'proj-1', 'NCD Household Survey Form', 'Active', 5000, 2125, 42.5),
('s2-1', 'proj-2', 'Smallholder Farmer Yield Questionnaire', 'Template', 3000, 0, 0.0)
ON CONFLICT (id) DO NOTHING;

-- Insert chat messages
INSERT INTO pms.chat_messages (id, project_id, channel, sender, role, message, timestamp) VALUES
('ch1-1', 'proj-1', 'general', 'Dr. Sarah Jenkins', 'Project Manager', 'Welcome team! Let''s use this workspace chat room for fast status checkups.', CURRENT_TIMESTAMP - INTERVAL '119 days'),
('ch1-2', 'proj-1', 'announcements', 'Amara Oke', 'Finance Officer', 'First grant disbursement cleared! Tablet hardware procurement order is now initialized.', CURRENT_TIMESTAMP - INTERVAL '95 days'),
('ch1-3', 'proj-1', 'general', 'Carlos Gomez', 'Field Supervisor', 'Quick update: enumerators in southern region report excellent community cooperation so far.', CURRENT_TIMESTAMP - INTERVAL '50 days'),
('ch2-1', 'proj-2', 'general', 'Marcus Vance', 'Project Manager', 'Starting census layout design. Looking for regional crop list docs.', CURRENT_TIMESTAMP - INTERVAL '20 days')
ON CONFLICT (id) DO NOTHING;

-- Insert helpdesk tickets
INSERT INTO pms.helpdesk_tickets (id, project_id, title, description, status, priority, created_by, created_at) VALUES
('hd1-1', 'proj-1', 'StatCollect sync failing on remote offline tablets', 'Enumerators in Sector 4 are getting timed-out error while trying to upload stored surveys.', 'In Progress', 'High', 'Carlos Gomez', CURRENT_TIMESTAMP - INTERVAL '5 days')
ON CONFLICT (id) DO NOTHING;

-- LogFrames & Results Frameworks
CREATE TABLE IF NOT EXISTS pms.logframes (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pms.logframe_items (
    id VARCHAR(36) PRIMARY KEY,
    logframe_id VARCHAR(36) REFERENCES pms.logframes(id) ON DELETE CASCADE,
    level VARCHAR(50) NOT NULL,
    code VARCHAR(50),
    description TEXT NOT NULL,
    indicators TEXT,
    means_of_verification TEXT,
    assumptions TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Theory of Change
CREATE TABLE IF NOT EXISTS pms.theory_of_change (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
    inputs TEXT[],
    activities TEXT[],
    outputs TEXT[],
    outcomes_short TEXT[],
    outcomes_long TEXT[],
    impact TEXT,
    assumptions TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Donors / Funding Partners CRM
CREATE TABLE IF NOT EXISTS pms.donors (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE,
    type VARCHAR(100),
    contact_person VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    website TEXT,
    total_funding FLOAT DEFAULT 0.0,
    currency VARCHAR(10) DEFAULT 'USD',
    status VARCHAR(50) DEFAULT 'Active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);