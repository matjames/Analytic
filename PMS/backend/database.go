package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB(dsn string) error {
	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PMS PostgreSQL database")

	// Auto-migrate schema if tables don't exist
	if err := migrateDB(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

func migrateDB() error {
	// Create pms schema if not exists
	_, err := DB.Exec(`CREATE SCHEMA IF NOT EXISTS pms AUTHORIZATION "PMS"`)
	if err != nil {
		return err
	}

	// ─── Enterprise Hierarchy Tables ───────────────────────────────
	// Organizations
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.organizations (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			code VARCHAR(50) UNIQUE,
			description TEXT,
			type VARCHAR(100),
			parent_id VARCHAR(36),
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Portfolios
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.portfolios (
			id VARCHAR(36) PRIMARY KEY,
			org_id VARCHAR(36) REFERENCES pms.organizations(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			code VARCHAR(50) UNIQUE,
			description TEXT,
			owner VARCHAR(255),
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Programmes
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.programmes (
			id VARCHAR(36) PRIMARY KEY,
			portfolio_id VARCHAR(36) REFERENCES pms.portfolios(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			code VARCHAR(50) UNIQUE,
			description TEXT,
			owner VARCHAR(255),
			budget_total FLOAT DEFAULT 0.0,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Components
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.components (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			parent_id VARCHAR(36),
			name VARCHAR(255) NOT NULL,
			code VARCHAR(50),
			description TEXT,
			type VARCHAR(100),
			sort_order INTEGER DEFAULT 0,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Activities
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.activities (
			id VARCHAR(36) PRIMARY KEY,
			component_id VARCHAR(36) REFERENCES pms.components(id) ON DELETE CASCADE,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			code VARCHAR(50),
			description TEXT,
			start_date DATE,
			end_date DATE,
			status VARCHAR(50) DEFAULT 'Planned',
			progress FLOAT DEFAULT 0.0,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Deliverables
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.deliverables (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			task_id VARCHAR(36),
			name VARCHAR(255) NOT NULL,
			description TEXT,
			type VARCHAR(100),
			status VARCHAR(50) DEFAULT 'Pending',
			due_date DATE,
			owner VARCHAR(255),
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Milestones
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.milestones (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			due_date DATE,
			status VARCHAR(50) DEFAULT 'Pending',
			owner VARCHAR(255),
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Projects table
	_, err = DB.Exec(`
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
		)`)
	if err != nil {
		return err
	}

	// Project members table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.project_members (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			role VARCHAR(255),
			email VARCHAR(255),
			avatar_url TEXT,
			department VARCHAR(255),
			phone VARCHAR(50),
			location VARCHAR(255),
			since DATE,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Add new columns to existing tables if they don't exist
	for _, statement := range []string{
		`ALTER TABLE pms.project_members ADD COLUMN IF NOT EXISTS department VARCHAR(255)`,
		`ALTER TABLE pms.projects ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`CREATE INDEX IF NOT EXISTS idx_pms_projects_workspace ON pms.projects(workspace_id)`,
		`ALTER TABLE pms.project_members ADD COLUMN IF NOT EXISTS phone VARCHAR(50)`,
		`ALTER TABLE pms.project_members ADD COLUMN IF NOT EXISTS location VARCHAR(255)`,
		`ALTER TABLE pms.project_members ADD COLUMN IF NOT EXISTS since DATE`,
		`ALTER TABLE pms.tasks ADD COLUMN IF NOT EXISTS is_milestone BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE pms.tasks ADD COLUMN IF NOT EXISTS is_deliverable BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE pms.risks ADD COLUMN IF NOT EXISTS due_date DATE`,
		`ALTER TABLE pms.documents ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'Draft'`,
		`ALTER TABLE pms.meetings ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'Scheduled'`,
		`ALTER TABLE pms.meetings ADD COLUMN IF NOT EXISTS decisions TEXT[]`,
	} {
		if _, err := DB.Exec(statement); err != nil {
			return fmt.Errorf("apply additive PMS migration: %w", err)
		}
	}

	// Tasks table
	_, err = DB.Exec(`
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
			is_milestone BOOLEAN DEFAULT FALSE,
			is_deliverable BOOLEAN DEFAULT FALSE,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Budget lines table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.budget_lines (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			category VARCHAR(100),
			description TEXT,
			source VARCHAR(100),
			amount FLOAT DEFAULT 0.0,
			spent FLOAT DEFAULT 0.0,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Funding sources table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.funding_sources (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(100),
			donor VARCHAR(255),
			amount FLOAT DEFAULT 0.0,
			received FLOAT DEFAULT 0.0,
			currency VARCHAR(10) DEFAULT 'USD',
			start_date DATE,
			end_date DATE,
			status VARCHAR(50) DEFAULT 'Active',
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Cost centres table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.cost_centres (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			code VARCHAR(50),
			description TEXT,
			budget FLOAT DEFAULT 0.0,
			spent FLOAT DEFAULT 0.0,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Budget revisions table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.budget_revisions (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			version INTEGER DEFAULT 1,
			description TEXT,
			amount FLOAT DEFAULT 0.0,
			approved_by VARCHAR(255),
			approved_at TIMESTAMP,
			status VARCHAR(50) DEFAULT 'Draft',
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Procurement references table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.procurement_refs (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			reference VARCHAR(100),
			description TEXT,
			vendor VARCHAR(255),
			amount FLOAT DEFAULT 0.0,
			status VARCHAR(50) DEFAULT 'Pending',
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Risks table
	_, err = DB.Exec(`
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
			due_date DATE,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Issues table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.issues (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			category VARCHAR(100),
			priority VARCHAR(50) DEFAULT 'Medium',
			status VARCHAR(50) DEFAULT 'Open',
			owner VARCHAR(255),
			due_date DATE,
			resolution TEXT,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Assumptions table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.assumptions (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			description TEXT NOT NULL,
			category VARCHAR(100),
			status VARCHAR(50) DEFAULT 'Active',
			owner VARCHAR(255),
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Lessons learned table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.lessons_learned (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			category VARCHAR(100),
			impact VARCHAR(100),
			owner VARCHAR(255),
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Corrective actions table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.corrective_actions (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			issue_id VARCHAR(36),
			description TEXT NOT NULL,
			status VARCHAR(50) DEFAULT 'Open',
			owner VARCHAR(255),
			due_date DATE,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Documents table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.documents (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50),
			size VARCHAR(50),
			uploaded_by VARCHAR(255),
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			url TEXT,
			status VARCHAR(50) DEFAULT 'Draft'
		)`)
	if err != nil {
		return err
	}

	// Meetings table
	_, err = DB.Exec(`
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
			status VARCHAR(50) DEFAULT 'Scheduled',
			decisions TEXT[],
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Surveys table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.surveys (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			status VARCHAR(50) DEFAULT 'Template',
			target_sample INTEGER DEFAULT 0,
			submissions INTEGER DEFAULT 0,
			progress FLOAT DEFAULT 0.0,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Chat messages table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.chat_messages (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			channel VARCHAR(100) DEFAULT 'general',
			sender VARCHAR(255),
			role VARCHAR(255),
			message TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Helpdesk tickets table
	_, err = DB.Exec(`
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
		)`)
	if err != nil {
		return err
	}

	// Workflow rules table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.workflow_rules (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			from_stage VARCHAR(50),
			to_stage VARCHAR(50),
			required_role VARCHAR(100),
			auto_actions TEXT[],
			enabled BOOLEAN DEFAULT TRUE,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Permissions table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.permissions (
			id VARCHAR(36) PRIMARY KEY,
			role VARCHAR(100) NOT NULL,
			resource VARCHAR(100) NOT NULL,
			action VARCHAR(50) NOT NULL,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Audit log table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.audit_logs (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36),
			user_name VARCHAR(255),
			action VARCHAR(100),
			entity VARCHAR(100),
			entity_id VARCHAR(36),
			details TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Reports table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.reports (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			type VARCHAR(100),
			format VARCHAR(50) DEFAULT 'PDF',
			generated_by VARCHAR(255),
			generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			content TEXT,
			status VARCHAR(50) DEFAULT 'Generated'
		)`)
	if err != nil {
		return err
	}

	// Calendar events table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.calendar_events (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			event_type VARCHAR(100),
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			location VARCHAR(255),
			attendees TEXT[],
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// ─── Phase 4 Extensions: LogFrames, Theory of Change, Donors ───

	// LogFrames table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.logframes (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// LogFrame Items table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.logframe_items (
			id VARCHAR(36) PRIMARY KEY,
			logframe_id VARCHAR(36) REFERENCES pms.logframes(id) ON DELETE CASCADE,
			level VARCHAR(50) NOT NULL,
			code VARCHAR(50),
			description TEXT NOT NULL,
			indicators TEXT[],
			means_of_verification TEXT[],
			assumptions TEXT[],
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Theory of Change table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.theory_of_change (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(36) REFERENCES pms.projects(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			narrative TEXT,
			inputs TEXT[],
			activities TEXT[],
			outputs TEXT[],
			short_term_outcomes TEXT[],
			long_term_outcomes TEXT[],
			impact TEXT[],
			assumptions TEXT[],
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Donors CRM table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pms.donors (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			code VARCHAR(50) UNIQUE,
			type VARCHAR(100) DEFAULT 'Bilateral',
			contact_person VARCHAR(255),
			email VARCHAR(255),
			phone VARCHAR(50),
			website VARCHAR(255),
			total_funding FLOAT DEFAULT 0.0,
			currency VARCHAR(10) DEFAULT 'USD',
			status VARCHAR(50) DEFAULT 'Active',
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Phase 4 ownership indexes keep project planning and donor records fast
	// when workspace filters are applied to every list and mutation.
	for _, statement := range []string{
		`ALTER TABLE pms.logframes ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`ALTER TABLE pms.logframes ADD COLUMN IF NOT EXISTS created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='pms' AND table_name='logframes' AND column_name='created_at') THEN UPDATE pms.logframes SET created_time = created_at WHERE created_at IS NOT NULL; END IF; END $$`,
		`CREATE INDEX IF NOT EXISTS idx_pms_logframes_project ON pms.logframes(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pms_logframes_workspace ON pms.logframes(workspace_id)`,
		`ALTER TABLE pms.logframe_items ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`ALTER TABLE pms.logframe_items ADD COLUMN IF NOT EXISTS created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='pms' AND table_name='logframe_items' AND column_name='created_at') THEN UPDATE pms.logframe_items SET created_time = created_at WHERE created_at IS NOT NULL; END IF; END $$`,
		`CREATE INDEX IF NOT EXISTS idx_pms_logframe_items_logframe ON pms.logframe_items(logframe_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pms_logframe_items_workspace ON pms.logframe_items(workspace_id)`,
		`ALTER TABLE pms.theory_of_change ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`ALTER TABLE pms.theory_of_change ADD COLUMN IF NOT EXISTS title VARCHAR(255) DEFAULT 'Theory of Change'`,
		`ALTER TABLE pms.theory_of_change ADD COLUMN IF NOT EXISTS narrative TEXT`,
		`ALTER TABLE pms.theory_of_change ADD COLUMN IF NOT EXISTS short_term_outcomes TEXT[]`,
		`ALTER TABLE pms.theory_of_change ADD COLUMN IF NOT EXISTS long_term_outcomes TEXT[]`,
		`ALTER TABLE pms.theory_of_change ADD COLUMN IF NOT EXISTS created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='pms' AND table_name='theory_of_change' AND column_name='outcomes_short') THEN UPDATE pms.theory_of_change SET short_term_outcomes = outcomes_short WHERE short_term_outcomes IS NULL AND outcomes_short IS NOT NULL; END IF; END $$`,
		`DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='pms' AND table_name='theory_of_change' AND column_name='outcomes_long') THEN UPDATE pms.theory_of_change SET long_term_outcomes = outcomes_long WHERE long_term_outcomes IS NULL AND outcomes_long IS NOT NULL; END IF; END $$`,
		`UPDATE pms.theory_of_change SET title = 'Theory of Change' WHERE title IS NULL OR title = ''`,
		`DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='pms' AND table_name='theory_of_change' AND column_name='created_at') THEN UPDATE pms.theory_of_change SET created_time = created_at WHERE created_at IS NOT NULL; END IF; END $$`,
		`CREATE INDEX IF NOT EXISTS idx_pms_theory_of_change_project ON pms.theory_of_change(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pms_theory_of_change_workspace ON pms.theory_of_change(workspace_id)`,
		`ALTER TABLE pms.donors ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`ALTER TABLE pms.donors ADD COLUMN IF NOT EXISTS created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='pms' AND table_name='donors' AND column_name='created_at') THEN UPDATE pms.donors SET created_time = created_at WHERE created_at IS NOT NULL; END IF; END $$`,
		`CREATE INDEX IF NOT EXISTS idx_pms_donors_workspace ON pms.donors(workspace_id)`,
	} {
		if _, err := DB.Exec(statement); err != nil {
			return fmt.Errorf("apply Phase 4 ownership migration: %w", err)
		}
	}

	// Seed data if projects table is empty
	var count int
	err = DB.QueryRow(`SELECT COUNT(*) FROM pms.projects`).Scan(&count)
	if err != nil {
		return err
	}

	// Check if new enterprise tables need seeding (independent of projects)
	var orgCount int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.organizations`).Scan(&orgCount)

	if count == 0 || orgCount == 0 {
		// Insert seed organizations
		_, err = DB.Exec(`
			INSERT INTO pms.organizations (id, name, code, description, type) VALUES
			('org-1', 'StatGate Ministry Alliance', 'SG-MA', 'Primary government alliance for public health and development intelligence', 'Government'),
			('org-2', 'StatGate Agritech Directorate', 'SG-AD', 'Agricultural technology and food security intelligence directorate', 'Government')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed portfolios
		_, err = DB.Exec(`
			INSERT INTO pms.portfolios (id, org_id, name, code, description, owner) VALUES
			('pf-1', 'org-1', 'Public Health Intelligence', 'PHI', 'Public health surveillance, disease mapping, and health systems research', 'Dr. Sarah Jenkins'),
			('pf-2', 'org-2', 'Economic & Resource Intelligence', 'ERI', 'Agricultural productivity, food security, and rural development intelligence', 'Marcus Vance')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed programmes
		_, err = DB.Exec(`
			INSERT INTO pms.programmes (id, portfolio_id, name, code, description, owner, budget_total) VALUES
			('pg-1', 'pf-1', 'National Surveys Programme', 'NSP', 'Nationwide health and demographic survey initiatives', 'Dr. Sarah Jenkins', 450000),
			('pg-2', 'pf-2', 'Rural Development Initiative', 'RDI', 'Agricultural productivity and rural economic development programmes', 'Marcus Vance', 320000)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed projects
		_, err = DB.Exec(`
			INSERT INTO pms.projects (id, code, name, description, stage, progress, org, portfolio, programme, owner, start_date, end_date, target_geo, tags, budget_total, spent_total, risks_count, issues_count, created_time) VALUES
			('proj-1', 'SG-2026-NCD', 'National Non-Communicable Diseases Survey 2026', 'Comprehensive nationwide survey mapping NCD prevalence, risk factors, and healthcare access across all 12 provinces.', 'Implementation', 42.5, 'StatGate Ministry Alliance', 'Public Health Intelligence', 'National Surveys Programme', 'Dr. Sarah Jenkins', '2026-02-15', '2026-11-30', 'National Coverage (All Provinces)', ARRAY['NCD', 'Health', 'Survey', 'National'], 450000, 192500, 2, 1, CURRENT_TIMESTAMP - INTERVAL '120 days'),
			('proj-2', 'SG-2026-AGRI', 'Agricultural Productivity & Food Security Census', 'Assessing smallholder farmer yields, irrigation tech adoption, and market access metrics in the northern agricultural belt.', 'Planning', 12.0, 'StatGate Agritech Directorate', 'Economic & Resource Intelligence', 'Rural Development Initiative', 'Marcus Vance', '2026-09-01', '2027-03-31', 'Northern and Eastern Agricultural Zones', ARRAY['Agriculture', 'Food Security', 'Census'], 320000, 15000, 1, 0, CURRENT_TIMESTAMP - INTERVAL '30 days')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed components
		_, err = DB.Exec(`
			INSERT INTO pms.components (id, project_id, name, code, description, type, sort_order) VALUES
			('c1-1', 'proj-1', 'Survey Design & Protocol', 'C1', 'Methodology, ethics, and survey instrument design', 'Design', 1),
			('c1-2', 'proj-1', 'Field Operations', 'C2', 'Field team deployment, training, and data collection', 'Operations', 2),
			('c1-3', 'proj-1', 'Data Management & Analysis', 'C3', 'Data cleaning, validation, and statistical analysis', 'Analysis', 3),
			('c2-1', 'proj-2', 'Census Planning', 'C1', 'Scope definition, indicator selection, and budget planning', 'Planning', 1),
			('c2-2', 'proj-2', 'Field Deployment', 'C2', 'Enumerator training and field data collection', 'Operations', 2)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed activities
		_, err = DB.Exec(`
			INSERT INTO pms.activities (id, component_id, project_id, name, code, description, start_date, end_date, status, progress) VALUES
			('a1-1', 'c1-1', 'proj-1', 'Ethics Approval Process', 'A1.1', 'Obtain ethics board approvals for survey methodology', '2026-02-15', '2026-03-10', 'Completed', 100),
			('a1-2', 'c1-1', 'proj-1', 'Questionnaire Development', 'A1.2', 'Develop and pilot the NCD survey questionnaire', '2026-03-01', '2026-04-05', 'Completed', 100),
			('a1-3', 'c1-2', 'proj-1', 'Field Staff Recruitment', 'A2.1', 'Recruit and vet 48 enumerators across regions', '2026-04-01', '2026-04-25', 'Completed', 100),
			('a1-4', 'c1-2', 'proj-1', 'Data Collection Phase 1', 'A2.2', 'Launch household surveys in Western and Southern provinces', '2026-05-01', '2026-08-30', 'In Progress', 75),
			('a1-5', 'c1-3', 'proj-1', 'Data Cleaning & Validation', 'A3.1', 'Clean and validate collected survey data', '2026-08-01', '2026-10-15', 'Planned', 0),
			('a2-1', 'c2-1', 'proj-2', 'Define Census Scope', 'A1.1', 'Consult Ministry of Agriculture to lock indicators', '2026-08-01', '2026-08-25', 'In Progress', 50),
			('a2-2', 'c2-1', 'proj-2', 'Budget Approval', 'A1.2', 'Secure formal sign-off for rural development grants', '2026-08-26', '2026-09-10', 'Planned', 0)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed milestones
		_, err = DB.Exec(`
			INSERT INTO pms.milestones (id, project_id, name, description, due_date, status, owner) VALUES
			('ms1-1', 'proj-1', 'Protocol Approval', 'Ethics and methodology approval secured', '2026-03-10', 'Completed', 'Dr. Sarah Jenkins'),
			('ms1-2', 'proj-1', 'Field Team Deployed', 'All 48 enumerators trained and deployed', '2026-04-25', 'Completed', 'Carlos Gomez'),
			('ms1-3', 'proj-1', 'Phase 1 Data Collection Complete', 'Western and Southern provinces surveyed', '2026-08-30', 'In Progress', 'Carlos Gomez'),
			('ms1-4', 'proj-1', 'Final Report Submission', 'Comprehensive NCD report delivered', '2026-11-30', 'Pending', 'Dr. Sarah Jenkins'),
			('ms2-1', 'proj-2', 'Census Scope Locked', 'Indicators and scope approved by ministry', '2026-08-25', 'In Progress', 'Marcus Vance'),
			('ms2-2', 'proj-2', 'Funding Secured', 'All grants and budget allocations confirmed', '2026-09-10', 'Pending', 'Elena Rostova')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed deliverables
		_, err = DB.Exec(`
			INSERT INTO pms.deliverables (id, project_id, task_id, name, description, type, status, due_date, owner) VALUES
			('d1-1', 'proj-1', 't1-1', 'Survey Protocol Document', 'Final approved survey protocol with methodology', 'Document', 'Delivered', '2026-03-10', 'David Chen'),
			('d1-2', 'proj-1', 't1-2', 'Digital Questionnaire', 'Deployed digital survey forms in StatCollect', 'Software', 'Delivered', '2026-04-05', 'David Chen'),
			('d1-3', 'proj-1', 't1-4', 'Phase 1 Dataset', 'Cleaned and validated household survey dataset', 'Dataset', 'In Progress', '2026-08-30', 'Carlos Gomez'),
			('d2-1', 'proj-2', 't2-1', 'Census Scope Document', 'Approved census scope and indicator list', 'Document', 'In Progress', '2026-08-25', 'Marcus Vance')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed members
		_, err = DB.Exec(`
			INSERT INTO pms.project_members (id, project_id, name, role, email, avatar_url, department, phone, location, since) VALUES
			('m-1', 'proj-1', 'Dr. Sarah Jenkins', 'Project Manager', 's.jenkins@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=sarah', 'Project Leadership', '+256 701 234 001', 'Kampala HQ', '2026-02-15'),
			('m-2', 'proj-1', 'David Chen', 'Research Lead', 'd.chen@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=david', 'Data Science', '+256 701 234 002', 'Kampala HQ', '2026-02-15'),
			('m-3', 'proj-1', 'Amara Oke', 'Finance Officer', 'a.oke@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=amara', 'Finance', '+256 701 234 003', 'Kampala HQ', '2026-02-15'),
			('m-4', 'proj-1', 'Carlos Gomez', 'Field Supervisor', 'c.gomez@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=carlos', 'Field Operations', '+256 701 234 004', 'Western Zone', '2026-03-01'),
			('m-5', 'proj-2', 'Marcus Vance', 'Project Manager', 'm.vance@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=marcus', 'Project Leadership', '+256 701 234 005', 'Kampala HQ', '2026-08-01'),
			('m-6', 'proj-2', 'Elena Rostova', 'Monitoring & Evaluation Officer', 'e.rostova@statgate.gov', 'https://api.dicebear.com/7.x/adventurer/svg?seed=elena', 'M&E', '+256 701 234 006', 'Kampala HQ', '2026-08-01')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed tasks
		_, err = DB.Exec(`
			INSERT INTO pms.tasks (id, project_id, wbs_code, title, description, status, priority, start_date, end_date, progress, assigned_to) VALUES
			('t1-1', 'proj-1', '1.1', 'Survey Protocol Design', 'Finalize ethics board approvals and design NCD methodology.', 'Done', 'High', '2026-02-15', '2026-03-10', 100, 'David Chen'),
			('t1-2', 'proj-1', '1.2', 'Digital Questionnaire Deployment', 'Build digital survey forms in StatCollect framework.', 'Done', 'Medium', '2026-03-11', '2026-04-05', 100, 'David Chen'),
			('t1-3', 'proj-1', '2.1', 'Field Staff Training', 'Conduct training seminars for 48 enumerators across regions.', 'Done', 'High', '2026-04-10', '2026-04-25', 100, 'Carlos Gomez'),
			('t1-4', 'proj-1', '2.2', 'Data Collection Phase 1', 'Launch household field surveys in Western and Southern provinces.', 'In Progress', 'High', '2026-05-01', '2026-08-30', 75, 'Carlos Gomez'),
			('t1-5', 'proj-1', '3.1', 'Interim Analysis & Midterm Report', 'Compile preliminary findings for the Department of Health.', 'Todo', 'Medium', '2026-09-01', '2026-09-30', 0, 'Dr. Sarah Jenkins'),
			('t2-1', 'proj-2', '1.1', 'Define Census Scope & Indicators', 'Consult Ministry of Agriculture to lock agricultural indicators.', 'In Progress', 'High', '2026-08-01', '2026-08-25', 50, 'Marcus Vance'),
			('t2-2', 'proj-2', '1.2', 'Budget Approval and Fund Allocation', 'Secure formal sign-off for rural development grants.', 'Todo', 'High', '2026-08-26', '2026-09-10', 0, 'Elena Rostova')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed budget lines
		_, err = DB.Exec(`
			INSERT INTO pms.budget_lines (id, project_id, category, description, source, amount, spent) VALUES
			('b1-1', 'proj-1', 'Personnel', 'Field enumerator allowances and PM salary', 'State Fund', 220000, 110000),
			('b1-2', 'proj-1', 'Travel', 'Fuel, transport hire, logistics for remote visits', 'State Fund', 80000, 45000),
			('b1-3', 'proj-1', 'Equipment', 'Tablet PCs for offline data collection', 'WHO Grant', 100000, 32500),
			('b1-4', 'proj-1', 'Supplies', 'PPE, training materials, printed documentation', 'WHO Grant', 50000, 5000),
			('b2-1', 'proj-2', 'Personnel', 'Enumerator training staff fees', 'FAO Grant', 150000, 5000),
			('b2-2', 'proj-2', 'Travel', 'Logistics for northern remote districts', 'FAO Grant', 120000, 10000),
			('b2-3', 'proj-2', 'Indirect', 'Operational overhead and mapping software', 'State Fund', 50000, 0)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed funding sources
		_, err = DB.Exec(`
			INSERT INTO pms.funding_sources (id, project_id, name, type, donor, amount, received, currency, start_date, end_date, status) VALUES
			('fs1-1', 'proj-1', 'State Health Fund', 'Government', 'Ministry of Health', 300000, 200000, 'USD', '2026-02-15', '2026-11-30', 'Active'),
			('fs1-2', 'proj-1', 'WHO NCD Grant', 'International', 'World Health Organization', 150000, 75000, 'USD', '2026-03-01', '2026-11-30', 'Active'),
			('fs2-1', 'proj-2', 'FAO Agriculture Grant', 'International', 'Food and Agriculture Organization', 200000, 50000, 'USD', '2026-08-01', '2027-03-31', 'Active'),
			('fs2-2', 'proj-2', 'State Rural Development Fund', 'Government', 'Ministry of Agriculture', 120000, 0, 'USD', '2026-08-15', '2027-03-31', 'Pending')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed cost centres
		_, err = DB.Exec(`
			INSERT INTO pms.cost_centres (id, project_id, name, code, description, budget, spent) VALUES
			('cc1-1', 'proj-1', 'Field Operations', 'CC-FO', 'Field data collection operations', 200000, 120000),
			('cc1-2', 'proj-1', 'Data Management', 'CC-DM', 'Data processing and analysis', 100000, 30000),
			('cc1-3', 'proj-1', 'Administration', 'CC-AD', 'Project administration and overhead', 150000, 42500),
			('cc2-1', 'proj-2', 'Planning & Design', 'CC-PD', 'Census planning and design', 100000, 10000),
			('cc2-2', 'proj-2', 'Field Deployment', 'CC-FD', 'Field team deployment and operations', 220000, 5000)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed risks
		_, err = DB.Exec(`
			INSERT INTO pms.risks (id, project_id, description, category, probability, impact, mitigation, status, owner, due_date) VALUES
			('r1-1', 'proj-1', 'Delayed access approval in remote districts', 'Operational', 'Medium', 'High', 'Establish coordination with local chieftains and rural clinics early.', 'Active', 'Carlos Gomez', '2026-06-30'),
			('r1-2', 'proj-1', 'Tablet hardware failures in humid field conditions', 'Technical', 'Low', 'Medium', 'Procure rugged waterproof cases and supply field backups.', 'Mitigated', 'David Chen', '2026-05-15'),
			('r2-1', 'proj-2', 'Heavy rain season blocking road access', 'External', 'High', 'High', 'Schedule survey windows to avoid peak monsoon months.', 'Active', 'Marcus Vance', '2026-10-31')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed issues
		_, err = DB.Exec(`
			INSERT INTO pms.issues (id, project_id, title, description, category, priority, status, owner, due_date, resolution) VALUES
			('i1-1', 'proj-1', 'Tablet shortage in Southern Province', '12 tablets reported faulty, delaying data collection in 3 districts', 'Equipment', 'High', 'Open', 'David Chen', '2026-07-15', ''),
			('i1-2', 'proj-1', 'Enumerator attrition in Western Zone', '4 enumerators resigned, requiring replacement and retraining', 'Human Resources', 'Medium', 'In Progress', 'Carlos Gomez', '2026-07-01', 'Recruitment underway, training scheduled for next week')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed assumptions
		_, err = DB.Exec(`
			INSERT INTO pms.assumptions (id, project_id, description, category, status, owner) VALUES
			('as1-1', 'proj-1', 'Government funding will be disbursed on schedule', 'Financial', 'Active', 'Amara Oke'),
			('as1-2', 'proj-1', 'Community participation rates remain above 80%', 'Community', 'Active', 'Carlos Gomez'),
			('as2-1', 'proj-2', 'FAO grant approval completes by September', 'Financial', 'Active', 'Elena Rostova')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed lessons learned
		_, err = DB.Exec(`
			INSERT INTO pms.lessons_learned (id, project_id, title, description, category, impact, owner) VALUES
			('ll1-1', 'proj-1', 'Early community engagement is critical', 'Districts with early engagement had 30% higher participation rates', 'Community Engagement', 'High', 'Carlos Gomez'),
			('ll1-2', 'proj-1', 'Rugged devices reduce field failures', 'Investing in rugged tablets reduced hardware failure rate by 60%', 'Technology', 'Medium', 'David Chen')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed corrective actions
		_, err = DB.Exec(`
			INSERT INTO pms.corrective_actions (id, project_id, issue_id, description, status, owner, due_date) VALUES
			('ca1-1', 'proj-1', 'i1-1', 'Procure 15 replacement tablets and redistribute from central stock', 'In Progress', 'David Chen', '2026-07-20'),
			('ca1-2', 'proj-1', 'i1-2', 'Fast-track recruitment of 4 new enumerators', 'In Progress', 'Carlos Gomez', '2026-07-10')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed documents
		_, err = DB.Exec(`
			INSERT INTO pms.documents (id, project_id, name, type, size, uploaded_by, uploaded_at, url, status) VALUES
			('d1-1', 'proj-1', 'Project Charter & Terms of Reference', 'Contract', '2.4 MB', 'Dr. Sarah Jenkins', CURRENT_TIMESTAMP - INTERVAL '110 days', '/docs/charter.pdf', 'Approved'),
			('d1-2', 'proj-1', 'Initial Concept Note & Feasibility Study', 'Proposal', '1.1 MB', 'Marcus Vance', CURRENT_TIMESTAMP - INTERVAL '115 days', '/docs/concept.pdf', 'Approved'),
			('d1-3', 'proj-1', 'Approved Budget Framework FY2026', 'Budget', '890 KB', 'Finance Office', CURRENT_TIMESTAMP - INTERVAL '100 days', '/docs/budget.pdf', 'Approved'),
			('d1-4', 'proj-1', 'Field Enumerator Training Manual v2.1', 'Field Manual', '3.2 MB', 'Alice Ouko', CURRENT_TIMESTAMP - INTERVAL '80 days', '/docs/manual.pdf', 'Active'),
			('d2-1', 'proj-2', 'Census Scope Definition Document', 'Proposal', '1.8 MB', 'Marcus Vance', CURRENT_TIMESTAMP - INTERVAL '20 days', '/docs/scope.pdf', 'Draft')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed meetings
		_, err = DB.Exec(`
			INSERT INTO pms.meetings (id, project_id, title, date_time, location, attendees, agenda, minutes, action_items, status, decisions) VALUES
			('mt1-1', 'proj-1', 'Project Kickoff Meeting', CURRENT_TIMESTAMP - INTERVAL '115 days', 'Kampala HQ - Conference Room A', ARRAY['Dr. Sarah Jenkins', 'David Chen', 'Amara Oke', 'Carlos Gomez'], 'Project overview, team roles, timeline review', 'Kickoff completed successfully. All team members briefed on objectives.', ARRAY['Finalize survey protocol', 'Procure field equipment'], 'Completed', ARRAY['Proceed with Phase 1 planning']),
			('mt1-2', 'proj-1', 'Field Operations Review', CURRENT_TIMESTAMP - INTERVAL '45 days', 'Virtual - Zoom', ARRAY['Dr. Sarah Jenkins', 'Carlos Gomez', 'David Chen'], 'Field progress, challenges, and resource needs', 'Reviewed field progress. Southern province ahead of schedule.', ARRAY['Address tablet shortage', 'Recruit additional enumerators'], 'Completed', ARRAY['Prioritize tablet replacement']),
			('mt2-1', 'proj-2', 'Census Planning Workshop', CURRENT_TIMESTAMP - INTERVAL '15 days', 'Kampala HQ - Training Room', ARRAY['Marcus Vance', 'Elena Rostova'], 'Census scope, indicators, and budget planning', 'Workshop completed. Indicators locked with ministry.', ARRAY['Finalize budget submission'], 'In Progress', ARRAY['Submit budget for approval'])
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed surveys
		_, err = DB.Exec(`
			INSERT INTO pms.surveys (id, project_id, name, status, target_sample, submissions, progress) VALUES
			('s1-1', 'proj-1', 'NCD Household Survey 2026', 'Active', 10000, 4250, 42.5),
			('s1-2', 'proj-1', 'Healthcare Access Assessment', 'Template', 5000, 0, 0),
			('s2-1', 'proj-2', 'Agricultural Productivity Census', 'Template', 8000, 0, 0)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed chat messages
		_, err = DB.Exec(`
			INSERT INTO pms.chat_messages (id, project_id, channel, sender, role, message, timestamp) VALUES
			('ch1-1', 'proj-1', 'general', 'Dr. Sarah Jenkins', 'Project Manager', 'Welcome team! Let''s use this workspace chat room for fast status checkups.', CURRENT_TIMESTAMP - INTERVAL '119 days'),
			('ch1-2', 'proj-1', 'announcements', 'Amara Oke', 'Finance Officer', 'First grant disbursement cleared! Tablet hardware procurement order is now initialized.', CURRENT_TIMESTAMP - INTERVAL '95 days'),
			('ch1-3', 'proj-1', 'general', 'Carlos Gomez', 'Field Supervisor', 'Quick update: enumerators in southern region report excellent community cooperation so far.', CURRENT_TIMESTAMP - INTERVAL '50 days'),
			('ch1-4', 'proj-1', 'meetings', 'Dr. Sarah Jenkins', 'Project Manager', 'Field Operations Review scheduled for next week. Agenda posted in Meetings tab.', CURRENT_TIMESTAMP - INTERVAL '48 days'),
			('ch2-1', 'proj-2', 'general', 'Marcus Vance', 'Project Manager', 'Starting census layout design. Looking for regional crop list docs.', CURRENT_TIMESTAMP - INTERVAL '20 days')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed helpdesk tickets
		_, err = DB.Exec(`
			INSERT INTO pms.helpdesk_tickets (id, project_id, title, description, status, priority, created_by, created_at) VALUES
			('hd1-1', 'proj-1', 'Tablet sync failure in field', 'Field tablets failing to sync data to central server', 'In Progress', 'High', 'Carlos Gomez', CURRENT_TIMESTAMP - INTERVAL '10 days'),
			('hd1-2', 'proj-1', 'StatCollect form update request', 'Request to add new question to NCD survey form', 'Open', 'Medium', 'David Chen', CURRENT_TIMESTAMP - INTERVAL '5 days')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed workflow rules
		_, err = DB.Exec(`
			INSERT INTO pms.workflow_rules (id, name, description, from_stage, to_stage, required_role, auto_actions, enabled) VALUES
			('wf-1', 'Concept to Proposal', 'Move project from concept to proposal stage', 'Concept', 'Proposal', 'Project Manager', ARRAY['notify_team', 'create_documents'], TRUE),
			('wf-2', 'Proposal to Planning', 'Move project from proposal to planning', 'Proposal', 'Planning', 'Project Manager', ARRAY['create_workspace', 'create_calendar'], TRUE),
			('wf-3', 'Planning to Approval', 'Submit project for formal approval', 'Planning', 'Approval', 'Programme Director', ARRAY['notify_director', 'create_approval_request'], TRUE),
			('wf-4', 'Approval to Funding', 'Project approved, moving to funding', 'Approval', 'Funding', 'Programme Director', ARRAY['create_budget', 'create_funding_sources'], TRUE),
			('wf-5', 'Funding to Implementation', 'Project funded, starting implementation', 'Funding', 'Implementation', 'Project Manager', ARRAY['create_dashboard', 'create_analytics', 'activate_reporting'], TRUE),
			('wf-6', 'Implementation to Monitoring', 'Project implementation complete, moving to monitoring', 'Implementation', 'Monitoring', 'M&E Officer', ARRAY['create_evaluation_plan', 'schedule_reviews'], TRUE),
			('wf-7', 'Monitoring to Evaluation', 'Project monitoring complete, starting evaluation', 'Monitoring', 'Evaluation', 'M&E Officer', ARRAY['create_evaluation_report'], TRUE),
			('wf-8', 'Evaluation to Closure', 'Project evaluation complete, closing project', 'Evaluation', 'Closure', 'Programme Director', ARRAY['archive_documents', 'final_report'], TRUE),
			('wf-9', 'Closure to Archive', 'Project closed, archiving all records', 'Closure', 'Archive', 'System Administrator', ARRAY['archive_all', 'notify_stakeholders'], TRUE)
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed permissions
		_, err = DB.Exec(`
			INSERT INTO pms.permissions (id, role, resource, action) VALUES
			('perm-1', 'System Administrator', '*', '*'),
			('perm-2', 'Programme Director', 'project', 'read'),
			('perm-3', 'Programme Director', 'project', 'update'),
			('perm-4', 'Programme Director', 'project', 'approve'),
			('perm-5', 'Project Manager', 'project', 'read'),
			('perm-6', 'Project Manager', 'project', 'update'),
			('perm-7', 'Project Manager', 'task', 'create'),
			('perm-8', 'Project Manager', 'task', 'update'),
			('perm-9', 'Project Manager', 'member', 'assign'),
			('perm-10', 'Finance Officer', 'budget', 'read'),
			('perm-11', 'Finance Officer', 'budget', 'update'),
			('perm-12', 'M&E Officer', 'report', 'create'),
			('perm-13', 'M&E Officer', 'report', 'read'),
			('perm-14', 'Research Lead', 'survey', 'create'),
			('perm-15', 'Research Lead', 'survey', 'update'),
			('perm-16', 'Team Lead', 'task', 'update'),
			('perm-17', 'Field Supervisor', 'task', 'read'),
			('perm-18', 'Field Supervisor', 'task', 'update'),
			('perm-19', 'Enumerator', 'survey', 'submit'),
			('perm-20', 'Viewer', 'project', 'read')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed calendar events
		_, err = DB.Exec(`
			INSERT INTO pms.calendar_events (id, project_id, title, description, event_type, start_time, end_time, location, attendees) VALUES
			('ev1-1', 'proj-1', 'Field Data Collection Deadline', 'Phase 1 data collection must be complete', 'Milestone', CURRENT_TIMESTAMP + INTERVAL '20 days', CURRENT_TIMESTAMP + INTERVAL '20 days', 'All Provinces', ARRAY['Carlos Gomez', 'Field Team']),
			('ev1-2', 'proj-1', 'Mid-Term Review Meeting', 'Review progress and address challenges', 'Meeting', CURRENT_TIMESTAMP + INTERVAL '30 days', CURRENT_TIMESTAMP + INTERVAL '30 days', 'Kampala HQ', ARRAY['Dr. Sarah Jenkins', 'David Chen', 'Amara Oke']),
			('ev2-1', 'proj-2', 'Budget Approval Deadline', 'Final budget submission to ministry', 'Deadline', CURRENT_TIMESTAMP + INTERVAL '15 days', CURRENT_TIMESTAMP + INTERVAL '15 days', 'Kampala HQ', ARRAY['Marcus Vance', 'Elena Rostova'])
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed audit logs
		_, err = DB.Exec(`
			INSERT INTO pms.audit_logs (id, project_id, user_name, action, entity, entity_id, details, timestamp) VALUES
			('al1-1', 'proj-1', 'Dr. Sarah Jenkins', 'CREATE', 'project', 'proj-1', 'Project created and workspace initialized', CURRENT_TIMESTAMP - INTERVAL '120 days'),
			('al1-2', 'proj-1', 'Dr. Sarah Jenkins', 'UPDATE', 'project', 'proj-1', 'Project stage changed to Implementation', CURRENT_TIMESTAMP - INTERVAL '60 days'),
			('al1-3', 'proj-1', 'Carlos Gomez', 'CREATE', 'risk', 'r1-1', 'New risk logged: Delayed access approval', CURRENT_TIMESTAMP - INTERVAL '30 days'),
			('al2-1', 'proj-2', 'Marcus Vance', 'CREATE', 'project', 'proj-2', 'Project created and workspace initialized', CURRENT_TIMESTAMP - INTERVAL '30 days')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		// Insert seed reports
		_, err = DB.Exec(`
			INSERT INTO pms.reports (id, project_id, title, type, format, generated_by, generated_at, content, status) VALUES
			('rp1-1', 'proj-1', 'NCD Survey Progress Report - Q2', 'Progress', 'PDF', 'Dr. Sarah Jenkins', CURRENT_TIMESTAMP - INTERVAL '15 days', 'Quarterly progress report for NCD survey', 'Generated'),
			('rp1-2', 'proj-1', 'NCD Survey Financial Report - Q2', 'Financial', 'PDF', 'Amara Oke', CURRENT_TIMESTAMP - INTERVAL '15 days', 'Quarterly financial report for NCD survey', 'Generated'),
			('rp2-1', 'proj-2', 'Census Planning Status Report', 'Progress', 'PDF', 'Marcus Vance', CURRENT_TIMESTAMP - INTERVAL '5 days', 'Status report for census planning phase', 'Generated')
			ON CONFLICT (id) DO NOTHING
		`)
		if err != nil {
			return err
		}

		log.Println("Seed data inserted successfully")
	}

	log.Println("Database schema migrated successfully")
	return nil
}
