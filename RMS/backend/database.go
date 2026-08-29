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
	_, _ = DB.Exec(`
		CREATE TABLE IF NOT EXISTS rms.ethics_committees (
			id           VARCHAR(36)  PRIMARY KEY,
			name         VARCHAR(255) NOT NULL,
			institution  VARCHAR(255) NOT NULL,
			chair_person VARCHAR(255),
			email        VARCHAR(255),
			members      TEXT[],
			active       BOOLEAN      DEFAULT TRUE,
			created_time TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
		)`)

	_, _ = DB.Exec(`
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
		)`)

	_, _ = DB.Exec(`
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
		)`)

	log.Println("RMS database migration completed successfully")
	return nil
}
