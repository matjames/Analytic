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

	log.Println("Successfully connected to RMS PostgreSQL database")

	// Auto-migrate schema
	if err := migrateDB(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

func migrateDB() error {
	// Create rms schema
	_, err := DB.Exec(`CREATE SCHEMA IF NOT EXISTS rms AUTHORIZATION "RMS"`)
	if err != nil {
		log.Printf("Warning: could not create schema (may already exist): %v", err)
	}

	// ─── Research Projects ─────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.research_projects (
			id                      VARCHAR(36)  PRIMARY KEY,
			code                    VARCHAR(50)  UNIQUE NOT NULL,
			name                    VARCHAR(255) NOT NULL,
			description             TEXT,
			type                    VARCHAR(100) DEFAULT 'Research Project',
			stage                   VARCHAR(100) NOT NULL DEFAULT 'Research Idea',
			progress                FLOAT        DEFAULT 0.0,
			principal_investigator  VARCHAR(255),
			owner                   VARCHAR(255),
			organisation            VARCHAR(255),
			portfolio               VARCHAR(255),
			programme               VARCHAR(255),
			start_date              DATE,
			end_date                DATE,
			target_geo              TEXT,
			budget_total            FLOAT        DEFAULT 0.0,
			spent_total             FLOAT        DEFAULT 0.0,
			tags                    TEXT[],
			pms_project_id          VARCHAR(36),
			statchat_room_id        VARCHAR(255),
			created_time            TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
			updated_time            TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create research_projects: %w", err)
	}
	for _, statement := range []string{
		`ALTER TABLE rms.research_projects ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`CREATE INDEX IF NOT EXISTS idx_rms_research_projects_workspace ON rms.research_projects(workspace_id)`,
	} {
		if _, err := DB.Exec(statement); err != nil {
			return fmt.Errorf("apply RMS workspace migration: %w", err)
		}
	}

	// ─── Research Members ──────────────────────────────────────
	_, err = DB.Exec(`
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
		)`)
	if err != nil {
		return fmt.Errorf("create research_members: %w", err)
	}

	// ─── Proposals ─────────────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.proposals (
			id                VARCHAR(36)  PRIMARY KEY,
			research_id       VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			title             VARCHAR(255) NOT NULL,
			version           VARCHAR(50)  DEFAULT '1.0',
			status            VARCHAR(50)  DEFAULT 'Draft',
			description       TEXT,
			draft_content     TEXT,
			background        TEXT,
			objectives        TEXT,
			methodology       TEXT,
			timeline_details  TEXT,
			budget_details    TEXT,
			reviewer_comments TEXT,
			submitted_by      VARCHAR(255),
			reviewed_by       VARCHAR(255),
			approved_by       VARCHAR(255),
			submitted_at      TIMESTAMP,
			approved_at       TIMESTAMP,
			created_time      TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
			updated_time      TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create proposals: %w", err)
	}

	// ─── Ethics Applications ───────────────────────────────────
	_, err = DB.Exec(`
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
		)`)
	if err != nil {
		return fmt.Errorf("create ethics_applications: %w", err)
	}

	// ─── Grants / Funding ──────────────────────────────────────
	_, err = DB.Exec(`
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
		)`)
	if err != nil {
		return fmt.Errorf("create grants: %w", err)
	}

	// ─── Literature Library ────────────────────────────────────
	_, err = DB.Exec(`
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
		)`)
	if err != nil {
		return fmt.Errorf("create literature: %w", err)
	}

	// ─── Datasets ──────────────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.datasets (
			id                VARCHAR(36)  PRIMARY KEY,
			research_id       VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			name              VARCHAR(255) NOT NULL,
			description       TEXT,
			version           VARCHAR(50)  DEFAULT '1.0',
			status            VARCHAR(50)  DEFAULT 'Draft',
			source_type       VARCHAR(100),
			collection_method VARCHAR(100),
			metadata_info     TEXT,
			variables_dict    TEXT,
			access_level      VARCHAR(50)  DEFAULT 'Internal',
			download_url      TEXT,
			statcollect_id    VARCHAR(255),
			created_time      TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
			updated_time      TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create datasets: %w", err)
	}

	// ─── Publications ──────────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.publications (
			id                   VARCHAR(36)  PRIMARY KEY,
			research_id          VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			title                VARCHAR(255) NOT NULL,
			pub_type             VARCHAR(100) DEFAULT 'Manuscript',
			authors              VARCHAR(255),
			journal              VARCHAR(255),
			status               VARCHAR(50)  DEFAULT 'Drafting',
			peer_review_comments TEXT,
			revision_history     TEXT,
			doi                  VARCHAR(100),
			acceptance_date      DATE,
			published_date       DATE,
			affiliation          VARCHAR(255),
			created_time         TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
			updated_time         TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create publications: %w", err)
	}

	// ─── Research Tasks ────────────────────────────────────────
	_, err = DB.Exec(`
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
		)`)
	if err != nil {
		return fmt.Errorf("create tasks: %w", err)
	}

	// ─── Meetings ──────────────────────────────────────────────
	_, err = DB.Exec(`
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
		)`)
	if err != nil {
		return fmt.Errorf("create meetings: %w", err)
	}

	// ─── Risks ─────────────────────────────────────────────────
	_, err = DB.Exec(`
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
		)`)
	if err != nil {
		return fmt.Errorf("create risks: %w", err)
	}

	// ─── Issues ─────────────────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.issues (
			id           VARCHAR(36)  PRIMARY KEY,
			research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			title        VARCHAR(255) NOT NULL,
			description  TEXT,
			category     VARCHAR(100),
			priority     VARCHAR(50)  DEFAULT 'Medium',
			status       VARCHAR(50)  DEFAULT 'Open',
			owner        VARCHAR(255),
			due_date     DATE,
			resolution   TEXT,
			created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create issues: %w", err)
	}

	// ─── Documents ─────────────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.documents (
			id           VARCHAR(36)  PRIMARY KEY,
			research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			name         VARCHAR(255) NOT NULL,
			type         VARCHAR(50),
			size         VARCHAR(50),
			uploaded_by  VARCHAR(255),
			uploaded_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
			url          TEXT,
			status       VARCHAR(50)  DEFAULT 'Draft',
			created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create documents: %w", err)
	}

	// ─── Surveys ───────────────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.surveys (
			id            VARCHAR(36)  PRIMARY KEY,
			research_id   VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			name          VARCHAR(255) NOT NULL,
			status        VARCHAR(50)  DEFAULT 'Template',
			target_sample INTEGER     DEFAULT 0,
			submissions   INTEGER     DEFAULT 0,
			progress      FLOAT       DEFAULT 0.0,
			created_time  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create surveys: %w", err)
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.statcollect_links (
			id VARCHAR(36) PRIMARY KEY,
			research_id VARCHAR(36) NOT NULL REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			submission_id VARCHAR(255) NOT NULL,
			form_id VARCHAR(255),
			relationship VARCHAR(100) NOT NULL DEFAULT 'collected-submission',
			status VARCHAR(50) NOT NULL DEFAULT 'received',
			source_event_type VARCHAR(100),
			tenant_id VARCHAR(128) NOT NULL DEFAULT 'default',
			workspace_id VARCHAR(128) NOT NULL,
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_by VARCHAR(255),
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (research_id, submission_id)
		)`)
	if err != nil {
		return fmt.Errorf("create statcollect_links: %w", err)
	}

	// ─── Reports ───────────────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.reports (
			id           VARCHAR(36)  PRIMARY KEY,
			research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			title        VARCHAR(255) NOT NULL,
			type         VARCHAR(100),
			format       VARCHAR(50)  DEFAULT 'PDF',
			generated_by VARCHAR(255),
			generated_at TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
			content      TEXT,
			status       VARCHAR(50)  DEFAULT 'Generated',
			created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create reports: %w", err)
	}

	// ─── Calendar Events ───────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.calendar_events (
			id           VARCHAR(36)  PRIMARY KEY,
			research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			title        VARCHAR(255) NOT NULL,
			description  TEXT,
			event_type   VARCHAR(100),
			start_time   TIMESTAMP,
			end_time     TIMESTAMP,
			location     VARCHAR(255),
			attendees    TEXT[],
			created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create calendar_events: %w", err)
	}

	// ─── Chat Messages ─────────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.chat_messages (
			id           VARCHAR(36)  PRIMARY KEY,
			research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			sender       VARCHAR(255) NOT NULL,
			channel      VARCHAR(100) DEFAULT 'general',
			message      TEXT         NOT NULL,
			created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create chat_messages: %w", err)
	}

	// ─── Audit Logs ────────────────────────────────────────────
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.audit_logs (
			id           VARCHAR(36)  PRIMARY KEY,
			research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			action       VARCHAR(255) NOT NULL,
			performed_by VARCHAR(255) NOT NULL,
			details      TEXT,
			created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create audit_logs: %w", err)
	}

	// ─── Additive column migrations (idempotent) ───────────────
	for _, statement := range []string{
		`ALTER TABLE rms.research_projects ADD COLUMN IF NOT EXISTS pms_project_id VARCHAR(36)`,
		`ALTER TABLE rms.research_projects ADD COLUMN IF NOT EXISTS statchat_room_id VARCHAR(255)`,
		`ALTER TABLE rms.proposals ADD COLUMN IF NOT EXISTS background TEXT`,
		`ALTER TABLE rms.proposals ADD COLUMN IF NOT EXISTS objectives TEXT`,
		`ALTER TABLE rms.proposals ADD COLUMN IF NOT EXISTS methodology TEXT`,
	} {
		if _, err := DB.Exec(statement); err != nil {
			return fmt.Errorf("apply additive RMS migration: %w", err)
		}
	}

	// ─── Phase 5 Extensions: Ethics Committees, DOIs, Open Access Repository ───
	for _, statement := range []string{`
		CREATE TABLE IF NOT EXISTS rms.ethics_committees (
			id           VARCHAR(36)  PRIMARY KEY,
			name         VARCHAR(255) NOT NULL,
			institution  VARCHAR(255) NOT NULL,
			chair_person VARCHAR(255),
			email        VARCHAR(255),
			members      TEXT[],
			active       BOOLEAN      DEFAULT TRUE,
			created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE rms.ethics_committees ADD COLUMN IF NOT EXISTS code VARCHAR(100)`,
		`ALTER TABLE rms.ethics_committees ADD COLUMN IF NOT EXISTS phone VARCHAR(50)`,
		`ALTER TABLE rms.ethics_committees ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'Active'`,
		`ALTER TABLE rms.ethics_committees ADD COLUMN IF NOT EXISTS approval_validity INTEGER DEFAULT 12`,
		`ALTER TABLE rms.ethics_committees ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`ALTER TABLE rms.ethics_committees ADD COLUMN IF NOT EXISTS members TEXT[]`,
		`ALTER TABLE rms.ethics_committees ADD COLUMN IF NOT EXISTS active BOOLEAN DEFAULT TRUE`,
		`ALTER TABLE rms.ethics_committees ADD COLUMN IF NOT EXISTS created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`CREATE INDEX IF NOT EXISTS idx_rms_ethics_committees_workspace ON rms.ethics_committees(workspace_id)`,
		`CREATE TABLE IF NOT EXISTS rms.ethics_committee_members (
			id VARCHAR(36) PRIMARY KEY,
			committee_id VARCHAR(36) NOT NULL REFERENCES rms.ethics_committees(id) ON DELETE CASCADE,
			user_id VARCHAR(255),
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			role VARCHAR(80) DEFAULT 'Reviewer',
			status VARCHAR(80) DEFAULT 'Pending',
			invited_by VARCHAR(255) NOT NULL,
			approved_by VARCHAR(255),
			invited_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			approved_at TIMESTAMP,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rms_ethics_committee_members_committee ON rms.ethics_committee_members(committee_id)`,
	} {
		if _, err := DB.Exec(statement); err != nil {
			return fmt.Errorf("apply ethics committee migration: %w", err)
		}
	}

	for _, statement := range []string{`
		CREATE TABLE IF NOT EXISTS rms.doi_records (
			id             VARCHAR(36)  PRIMARY KEY,
			research_id    VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			publication_id VARCHAR(36),
			doi            VARCHAR(100) UNIQUE NOT NULL,
			title          VARCHAR(255) NOT NULL,
			authors        TEXT[],
			year           INTEGER      DEFAULT 2026,
			publisher      VARCHAR(255) DEFAULT 'StatGate Open Science',
			url            TEXT,
			status         VARCHAR(50)  DEFAULT 'Registered',
			created_time   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rms_doi_records_research ON rms.doi_records(research_id)`,
		`ALTER TABLE rms.doi_records ADD COLUMN IF NOT EXISTS publication_id VARCHAR(36)`,
		`ALTER TABLE rms.doi_records ADD COLUMN IF NOT EXISTS publisher VARCHAR(255)`,
		`ALTER TABLE rms.doi_records ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'Registered'`,
		`ALTER TABLE rms.doi_records ADD COLUMN IF NOT EXISTS provider VARCHAR(50) DEFAULT 'local'`,
		`ALTER TABLE rms.doi_records ADD COLUMN IF NOT EXISTS provider_status VARCHAR(50) DEFAULT 'Not Submitted'`,
		`ALTER TABLE rms.doi_records ADD COLUMN IF NOT EXISTS external_id VARCHAR(255)`,
		`ALTER TABLE rms.doi_records ADD COLUMN IF NOT EXISTS registration_attempts INTEGER DEFAULT 0`,
		`ALTER TABLE rms.doi_records ADD COLUMN IF NOT EXISTS last_registration_error TEXT`,
		`ALTER TABLE rms.doi_records ADD COLUMN IF NOT EXISTS last_attempt_at TIMESTAMP`,
		`ALTER TABLE rms.doi_records ADD COLUMN IF NOT EXISTS created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
	} {
		if _, err := DB.Exec(statement); err != nil {
			return fmt.Errorf("apply DOI migration: %w", err)
		}
	}

	for _, statement := range []string{`
		CREATE TABLE IF NOT EXISTS rms.open_access_repo (
			id           VARCHAR(36)  PRIMARY KEY,
			research_id  VARCHAR(36)  REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			title        VARCHAR(255) NOT NULL,
			abstract     TEXT,
			license      VARCHAR(100) DEFAULT 'CC-BY-4.0',
			access_url   TEXT,
			download_url TEXT,
			file_size    VARCHAR(50),
			format       VARCHAR(50)  DEFAULT 'PDF',
			views        INTEGER      DEFAULT 0,
			downloads    INTEGER      DEFAULT 0,
			created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS description TEXT`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS resource_type VARCHAR(50) DEFAULT 'Dataset'`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS keywords TEXT`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS repo_name VARCHAR(255)`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS access_level VARCHAR(50) DEFAULT 'Open'`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS abstract TEXT`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS access_url TEXT`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS download_url TEXT`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS file_size VARCHAR(50)`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS format VARCHAR(50) DEFAULT 'PDF'`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS views INTEGER DEFAULT 0`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS downloads INTEGER DEFAULT 0`,
		`ALTER TABLE rms.open_access_repo ADD COLUMN IF NOT EXISTS created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`CREATE INDEX IF NOT EXISTS idx_rms_open_access_research ON rms.open_access_repo(research_id)`,
	} {
		if _, err := DB.Exec(statement); err != nil {
			return fmt.Errorf("apply open science migration: %w", err)
		}
	}

	for _, statement := range []string{
		`ALTER TABLE rms.ethics_applications ADD COLUMN IF NOT EXISTS committee_id VARCHAR(36)`,
		`CREATE TABLE IF NOT EXISTS rms.journal_submissions (
			id VARCHAR(36) PRIMARY KEY,
			research_id VARCHAR(36) REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			publication_id VARCHAR(36),
			journal VARCHAR(255) NOT NULL,
			manuscript_title VARCHAR(255) NOT NULL,
			submission_date DATE,
			status VARCHAR(80) DEFAULT 'Draft',
			manuscript_url TEXT,
			corresponding_author VARCHAR(255),
			reviewer_comments TEXT,
			next_action TEXT,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rms_journal_submissions_research ON rms.journal_submissions(research_id)`,
		`CREATE TABLE IF NOT EXISTS rms.research_archives (
			id VARCHAR(36) PRIMARY KEY,
			research_id VARCHAR(36) REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			output_id VARCHAR(36),
			title VARCHAR(255) NOT NULL,
			archive_type VARCHAR(80) DEFAULT 'Dataset',
			repository VARCHAR(255),
			access_url TEXT,
			checksum VARCHAR(255),
			status VARCHAR(80) DEFAULT 'Planned',
			retention_until DATE,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rms_research_archives_research ON rms.research_archives(research_id)`,
		`CREATE TABLE IF NOT EXISTS rms.conference_submissions (
			id VARCHAR(36) PRIMARY KEY,
			research_id VARCHAR(36) REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			conference_name VARCHAR(255) NOT NULL,
			manuscript_title VARCHAR(255) NOT NULL,
			submission_date DATE,
			status VARCHAR(80) DEFAULT 'Draft',
			presentation_type VARCHAR(80) DEFAULT 'Oral',
			abstract_url TEXT,
			reviewer_comments TEXT,
			next_action TEXT,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rms_conference_submissions_research ON rms.conference_submissions(research_id)`,
		`CREATE TABLE IF NOT EXISTS rms.conference_events (
			id VARCHAR(36) PRIMARY KEY,
			research_id VARCHAR(36) REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			event_name VARCHAR(255) NOT NULL,
			start_date DATE NOT NULL,
			end_date DATE NOT NULL,
			location VARCHAR(255),
			registration_url TEXT,
			status VARCHAR(80) DEFAULT 'Planned',
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rms_conference_events_research ON rms.conference_events(research_id)`,
		`CREATE TABLE IF NOT EXISTS rms.conference_attendance (
			id VARCHAR(36) PRIMARY KEY,
			event_id VARCHAR(36) NOT NULL REFERENCES rms.conference_events(id) ON DELETE CASCADE,
			user_id VARCHAR(255),
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			attendance_type VARCHAR(80) DEFAULT 'Delegate',
			status VARCHAR(80) DEFAULT 'Registered',
			registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			checked_in_at TIMESTAMP,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rms_conference_attendance_event ON rms.conference_attendance(event_id)`,
		`CREATE TABLE IF NOT EXISTS rms.knowledge_transfers (
			id VARCHAR(36) PRIMARY KEY,
			research_id VARCHAR(36) REFERENCES rms.research_projects(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			summary TEXT,
			body TEXT,
			tags TEXT[],
			status VARCHAR(80) DEFAULT 'Draft',
			knowledge_id VARCHAR(36),
			published_at TIMESTAMP,
			created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rms_knowledge_transfers_research ON rms.knowledge_transfers(research_id)`,
	} {
		if _, err := DB.Exec(statement); err != nil {
			return fmt.Errorf("apply publication preservation migration: %w", err)
		}
	}

	log.Println("RMS database migration completed successfully")
	return nil
}
