package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var dbPool *sql.DB

// initDB opens the PostgreSQL connection pool and verifies connectivity.
func initDB(cfg *Config) {
	dsn := buildDSN(cfg)
	if dsn == "" {
		if cfg.FailFastWithoutDB {
			log.Fatal("persistence: database configuration missing and FAIL_FAST_WITHOUT_DB=true")
		}
		log.Println("persistence: DB not configured — running without persistence (not suitable for production)")
		return
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		handleDBError(cfg, fmt.Errorf("failed to open database: %w", err))
		return
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		handleDBError(cfg, fmt.Errorf("database ping failed: %w", err))
		return
	}

	dbPool = db
	log.Println("persistence: StatCitizen PostgreSQL connected")
}

func buildDSN(cfg *Config) string {
	if cfg.DBHost == "" {
		return ""
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)
}

func handleDBError(cfg *Config, err error) {
	if cfg.Env == "production" || cfg.FailFastWithoutDB {
		log.Fatalf("persistence: [FATAL] %v", err)
	}
	log.Printf("persistence: [WARNING] %v — running without database persistence", err)
}

func closeDB() {
	if dbPool != nil {
		_ = dbPool.Close()
	}
}

// runMigrations applies all pending migrations in order.
// Uses a migrations table to track what has been applied.
func runMigrations() error {
	if dbPool == nil {
		return nil
	}

	// Ensure migrations table exists
	_, err := dbPool.Exec(`
		CREATE TABLE IF NOT EXISTS statcitizen_migrations (
			id          SERIAL PRIMARY KEY,
			name        TEXT NOT NULL UNIQUE,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	// Apply each migration in order
	for _, m := range allMigrations() {
		var exists bool
		row := dbPool.QueryRow(`SELECT EXISTS(SELECT 1 FROM statcitizen_migrations WHERE name = $1)`, m.Name)
		if err := row.Scan(&exists); err != nil {
			return fmt.Errorf("checking migration %s: %w", m.Name, err)
		}
		if exists {
			continue
		}

		log.Printf("migrations: applying %s", m.Name)
		if _, err := dbPool.Exec(m.SQL); err != nil {
			return fmt.Errorf("applying migration %s: %w", m.Name, err)
		}
		if _, err := dbPool.Exec(`INSERT INTO statcitizen_migrations (name) VALUES ($1)`, m.Name); err != nil {
			return fmt.Errorf("recording migration %s: %w", m.Name, err)
		}
		log.Printf("migrations: applied %s", m.Name)
	}

	return nil
}

type migration struct {
	Name string
	SQL  string
}

func allMigrations() []migration {
	return []migration{
		{
			Name: "001_initial_schema",
			SQL: `
-- ─── Citizen Sessions ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS citizen_sessions (
	id           TEXT PRIMARY KEY,
	token        TEXT NOT NULL UNIQUE,
	tenant_id    TEXT NOT NULL DEFAULT '',
	ip_hash      TEXT,
	user_agent   TEXT,
	created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	expires_at   TIMESTAMPTZ NOT NULL,
	last_seen    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_citizen_sessions_token ON citizen_sessions(token);
CREATE INDEX IF NOT EXISTS idx_citizen_sessions_tenant ON citizen_sessions(tenant_id);

-- ─── Registered Citizens ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS registered_citizens (
	id              TEXT PRIMARY KEY,
	tenant_id       TEXT NOT NULL DEFAULT '',
	identity_type   TEXT NOT NULL DEFAULT 'registered',
	display_name    TEXT,
	email_hash      TEXT,
	phone_hash      TEXT,
	email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
	phone_verified  BOOLEAN NOT NULL DEFAULT FALSE,
	org_id          TEXT,
	district        TEXT,
	status          TEXT NOT NULL DEFAULT 'active',
	created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_registered_citizens_tenant ON registered_citizens(tenant_id);
CREATE INDEX IF NOT EXISTS idx_registered_citizens_email_hash ON registered_citizens(email_hash);

-- ─── Citizen Consent ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS citizen_consents (
	id              TEXT PRIMARY KEY,
	tenant_id       TEXT NOT NULL DEFAULT '',
	session_id      TEXT,
	citizen_id      TEXT,
	purpose         TEXT NOT NULL,
	data_categories TEXT[] NOT NULL DEFAULT '{}',
	visibility      TEXT NOT NULL DEFAULT 'institution',
	retention       TEXT NOT NULL DEFAULT '1_year',
	sensitivity     TEXT NOT NULL DEFAULT 'internal',
	granted         BOOLEAN NOT NULL DEFAULT FALSE,
	granted_at      TIMESTAMPTZ,
	withdrawn_at    TIMESTAMPTZ,
	expires_at      TIMESTAMPTZ,
	downstream_use  TEXT[] NOT NULL DEFAULT '{}',
	metadata        JSONB,
	created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_citizen_consents_tenant ON citizen_consents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_citizen_consents_citizen ON citizen_consents(citizen_id);
CREATE INDEX IF NOT EXISTS idx_citizen_consents_session ON citizen_consents(session_id);

-- ─── Feedback Categories ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS feedback_categories (
	id          TEXT PRIMARY KEY,
	tenant_id   TEXT NOT NULL DEFAULT '',
	name        TEXT NOT NULL,
	slug        TEXT NOT NULL,
	description TEXT,
	parent_id   TEXT,
	active      BOOLEAN NOT NULL DEFAULT TRUE,
	sort_order  INT NOT NULL DEFAULT 0,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(tenant_id, slug)
);

-- ─── Feedback Records ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS feedback_records (
	id                   TEXT PRIMARY KEY,
	canonical_id         TEXT NOT NULL UNIQUE,
	tenant_id            TEXT NOT NULL DEFAULT '',
	session_id           TEXT,
	citizen_id           TEXT,
	consent_id           TEXT NOT NULL,
	category_id          TEXT,
	subject              TEXT NOT NULL,
	description          TEXT NOT NULL,
	district             TEXT,
	facility_id          TEXT,
	project_id           TEXT,
	service_id           TEXT,
	priority             TEXT NOT NULL DEFAULT 'medium',
	sensitivity          TEXT NOT NULL DEFAULT 'internal',
	status               TEXT NOT NULL DEFAULT 'submitted',
	assigned_institution TEXT,
	source               TEXT NOT NULL DEFAULT 'web',
	anonymous            BOOLEAN NOT NULL DEFAULT FALSE,
	visibility           TEXT NOT NULL DEFAULT 'institution',
	resolution           TEXT,
	response             TEXT,
	correlation_id       TEXT NOT NULL,
	metadata             JSONB,
	created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	resolved_at          TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_feedback_tenant ON feedback_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_feedback_status ON feedback_records(status);
CREATE INDEX IF NOT EXISTS idx_feedback_citizen ON feedback_records(citizen_id);
CREATE INDEX IF NOT EXISTS idx_feedback_correlation ON feedback_records(correlation_id);
CREATE INDEX IF NOT EXISTS idx_feedback_created ON feedback_records(created_at DESC);

-- ─── Report Categories ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS report_categories (
	id          TEXT PRIMARY KEY,
	tenant_id   TEXT NOT NULL DEFAULT '',
	name        TEXT NOT NULL,
	slug        TEXT NOT NULL,
	description TEXT,
	active      BOOLEAN NOT NULL DEFAULT TRUE,
	sort_order  INT NOT NULL DEFAULT 0,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(tenant_id, slug)
);

-- ─── Citizen Reports ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS citizen_reports (
	id               TEXT PRIMARY KEY,
	canonical_id     TEXT NOT NULL UNIQUE,
	tenant_id        TEXT NOT NULL DEFAULT '',
	session_id       TEXT,
	citizen_id       TEXT,
	consent_id       TEXT NOT NULL,
	category_id      TEXT,
	title            TEXT NOT NULL,
	description      TEXT NOT NULL,
	location_text    TEXT,
	district         TEXT,
	subcounty        TEXT,
	parish           TEXT,
	latitude         DOUBLE PRECISION,
	longitude        DOUBLE PRECISION,
	location_consent BOOLEAN NOT NULL DEFAULT FALSE,
	facility_id      TEXT,
	project_id       TEXT,
	service_id       TEXT,
	priority         TEXT NOT NULL DEFAULT 'medium',
	severity         TEXT NOT NULL DEFAULT 'medium',
	anonymous        BOOLEAN NOT NULL DEFAULT FALSE,
	contact_preference TEXT,
	status           TEXT NOT NULL DEFAULT 'submitted',
	helpdesk_ticket_id TEXT,
	enterprise_case_id TEXT,
	correlation_id   TEXT NOT NULL,
	source           TEXT NOT NULL DEFAULT 'web',
	metadata         JSONB,
	created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	resolved_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_reports_tenant ON citizen_reports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON citizen_reports(status);
CREATE INDEX IF NOT EXISTS idx_reports_district ON citizen_reports(district);
CREATE INDEX IF NOT EXISTS idx_reports_citizen ON citizen_reports(citizen_id);
CREATE INDEX IF NOT EXISTS idx_reports_correlation ON citizen_reports(correlation_id);
CREATE INDEX IF NOT EXISTS idx_reports_created ON citizen_reports(created_at DESC);

-- ─── Attachment Records ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS attachment_records (
	id           TEXT PRIMARY KEY,
	tenant_id    TEXT NOT NULL DEFAULT '',
	entity_type  TEXT NOT NULL,
	entity_id    TEXT NOT NULL,
	file_name    TEXT NOT NULL,
	mime_type    TEXT NOT NULL,
	size_bytes   BIGINT NOT NULL DEFAULT 0,
	storage_path TEXT NOT NULL,
	checksum     TEXT,
	scan_status  TEXT NOT NULL DEFAULT 'pending',
	uploaded_by  TEXT,
	created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_attachments_entity ON attachment_records(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_attachments_tenant ON attachment_records(tenant_id);

-- ─── Consultations ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS consultations (
	id                  TEXT PRIMARY KEY,
	canonical_id        TEXT NOT NULL UNIQUE,
	tenant_id           TEXT NOT NULL DEFAULT '',
	institution_id      TEXT,
	title               TEXT NOT NULL,
	description         TEXT NOT NULL,
	category            TEXT,
	open_date           TIMESTAMPTZ NOT NULL,
	close_date          TIMESTAMPTZ NOT NULL,
	status              TEXT NOT NULL DEFAULT 'draft',
	allow_anonymous     BOOLEAN NOT NULL DEFAULT TRUE,
	require_verified    BOOLEAN NOT NULL DEFAULT FALSE,
	statcollect_form_id TEXT,
	participant_count   INT NOT NULL DEFAULT 0,
	response_count      INT NOT NULL DEFAULT 0,
	created_by          TEXT NOT NULL,
	published_by        TEXT,
	published_at        TIMESTAMPTZ,
	correlation_id      TEXT NOT NULL,
	metadata            JSONB,
	created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_consultations_tenant ON consultations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_consultations_status ON consultations(status);
CREATE INDEX IF NOT EXISTS idx_consultations_dates ON consultations(open_date, close_date);

-- ─── Consultation Questions ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS consultation_questions (
	id               TEXT PRIMARY KEY,
	consultation_id  TEXT NOT NULL REFERENCES consultations(id) ON DELETE CASCADE,
	tenant_id        TEXT NOT NULL DEFAULT '',
	text             TEXT NOT NULL,
	type             TEXT NOT NULL DEFAULT 'text',
	options          TEXT[] NOT NULL DEFAULT '{}',
	required         BOOLEAN NOT NULL DEFAULT FALSE,
	sort_order       INT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_consult_questions_consult ON consultation_questions(consultation_id);

-- ─── Consultation Responses ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS consultation_responses (
	id               TEXT PRIMARY KEY,
	tenant_id        TEXT NOT NULL DEFAULT '',
	consultation_id  TEXT NOT NULL,
	session_id       TEXT,
	citizen_id       TEXT,
	consent_id       TEXT NOT NULL,
	anonymous        BOOLEAN NOT NULL DEFAULT FALSE,
	answers          JSONB NOT NULL DEFAULT '{}',
	comment          TEXT,
	correlation_id   TEXT NOT NULL,
	created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_consult_responses_consult ON consultation_responses(consultation_id);
CREATE INDEX IF NOT EXISTS idx_consult_responses_tenant ON consultation_responses(tenant_id);

-- ─── Rating Dimensions ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS rating_dimensions (
	id          TEXT PRIMARY KEY,
	tenant_id   TEXT NOT NULL DEFAULT '',
	slug        TEXT NOT NULL,
	label       TEXT NOT NULL,
	description TEXT,
	min_score   INT NOT NULL DEFAULT 1,
	max_score   INT NOT NULL DEFAULT 5,
	active      BOOLEAN NOT NULL DEFAULT TRUE,
	sort_order  INT NOT NULL DEFAULT 0,
	UNIQUE(tenant_id, slug)
);

-- ─── Service Ratings ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS service_ratings (
	id             TEXT PRIMARY KEY,
	canonical_id   TEXT NOT NULL UNIQUE,
	tenant_id      TEXT NOT NULL DEFAULT '',
	session_id     TEXT,
	citizen_id     TEXT,
	consent_id     TEXT NOT NULL,
	service_id     TEXT NOT NULL,
	service_name   TEXT NOT NULL,
	facility_id    TEXT,
	district       TEXT,
	ratings        JSONB NOT NULL DEFAULT '{}',
	overall_score  DOUBLE PRECISION NOT NULL DEFAULT 0,
	comment        TEXT,
	anonymous      BOOLEAN NOT NULL DEFAULT FALSE,
	source         TEXT NOT NULL DEFAULT 'web',
	correlation_id TEXT NOT NULL,
	metadata       JSONB,
	created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ratings_tenant ON service_ratings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ratings_service ON service_ratings(service_id);
CREATE INDEX IF NOT EXISTS idx_ratings_facility ON service_ratings(facility_id);
CREATE INDEX IF NOT EXISTS idx_ratings_created ON service_ratings(created_at DESC);

-- ─── Public Publications ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public_publications (
	id               TEXT PRIMARY KEY,
	canonical_id     TEXT NOT NULL UNIQUE,
	tenant_id        TEXT NOT NULL DEFAULT '',
	category         TEXT NOT NULL,
	title            TEXT NOT NULL,
	summary          TEXT NOT NULL,
	body             TEXT,
	source           TEXT NOT NULL,
	source_object_id TEXT,
	source_app       TEXT,
	publisher        TEXT NOT NULL,
	approved_by      TEXT,
	status           TEXT NOT NULL DEFAULT 'draft',
	published_at     TIMESTAMPTZ,
	expires_at       TIMESTAMPTZ,
	tags             TEXT[] NOT NULL DEFAULT '{}',
	geo_scope        TEXT[] NOT NULL DEFAULT '{}',
	language         TEXT NOT NULL DEFAULT 'en',
	metadata         JSONB,
	created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_publications_tenant ON public_publications(tenant_id);
CREATE INDEX IF NOT EXISTS idx_publications_status ON public_publications(status);
CREATE INDEX IF NOT EXISTS idx_publications_category ON public_publications(category);
CREATE INDEX IF NOT EXISTS idx_publications_published ON public_publications(published_at DESC);

-- ─── Citizen Cases ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS citizen_cases (
	id               TEXT PRIMARY KEY,
	tenant_id        TEXT NOT NULL DEFAULT '',
	session_id       TEXT,
	citizen_id       TEXT,
	submission_type  TEXT NOT NULL,
	submission_id    TEXT NOT NULL,
	status           TEXT NOT NULL DEFAULT 'submitted',
	status_message   TEXT,
	response         TEXT,
	correlation_id   TEXT NOT NULL,
	created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	resolved_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_cases_submission ON citizen_cases(submission_type, submission_id);
CREATE INDEX IF NOT EXISTS idx_cases_citizen ON citizen_cases(citizen_id);
CREATE INDEX IF NOT EXISTS idx_cases_session ON citizen_cases(session_id);
CREATE INDEX IF NOT EXISTS idx_cases_correlation ON citizen_cases(correlation_id);

-- ─── Geographic Locations ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS geo_locations (
	id          TEXT PRIMARY KEY,
	tenant_id   TEXT NOT NULL DEFAULT '',
	level       TEXT NOT NULL,
	code        TEXT,
	name        TEXT NOT NULL,
	parent_id   TEXT,
	latitude    DOUBLE PRECISION,
	longitude   DOUBLE PRECISION,
	facility_id TEXT,
	active      BOOLEAN NOT NULL DEFAULT TRUE,
	UNIQUE(tenant_id, level, code)
);
CREATE INDEX IF NOT EXISTS idx_geo_tenant ON geo_locations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_geo_parent ON geo_locations(parent_id);

-- ─── Citizen Audit ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS citizen_audit_log (
	id             TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
	tenant_id      TEXT NOT NULL DEFAULT '',
	actor          TEXT NOT NULL,
	actor_type     TEXT NOT NULL DEFAULT 'citizen',
	action         TEXT NOT NULL,
	resource_type  TEXT NOT NULL,
	resource_id    TEXT NOT NULL,
	correlation_id TEXT NOT NULL,
	outcome        TEXT NOT NULL DEFAULT 'success',
	metadata       JSONB,
	created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_citizen_audit_tenant ON citizen_audit_log(tenant_id);
CREATE INDEX IF NOT EXISTS idx_citizen_audit_actor ON citizen_audit_log(actor);
CREATE INDEX IF NOT EXISTS idx_citizen_audit_created ON citizen_audit_log(created_at DESC);

-- ─── Event Dead Letter ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS event_dead_letter (
	id           TEXT PRIMARY KEY,
	tenant_id    TEXT NOT NULL DEFAULT '',
	event_type   TEXT NOT NULL,
	payload      JSONB NOT NULL,
	retry_count  INT NOT NULL DEFAULT 0,
	last_error   TEXT,
	status       TEXT NOT NULL DEFAULT 'pending',
	created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dlq_status ON event_dead_letter(status);
CREATE INDEX IF NOT EXISTS idx_dlq_created ON event_dead_letter(created_at DESC);

-- ─── Offline Drafts ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS offline_drafts (
	id             TEXT PRIMARY KEY,
	tenant_id      TEXT NOT NULL DEFAULT '',
	session_id     TEXT NOT NULL,
	draft_type     TEXT NOT NULL,
	payload        JSONB NOT NULL,
	retry_count    INT NOT NULL DEFAULT 0,
	last_error     TEXT,
	status         TEXT NOT NULL DEFAULT 'pending',
	correlation_id TEXT NOT NULL,
	created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_drafts_session ON offline_drafts(session_id);
CREATE INDEX IF NOT EXISTS idx_drafts_status ON offline_drafts(status);

-- ─── Admin Settings ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS admin_settings (
	id         TEXT PRIMARY KEY,
	tenant_id  TEXT NOT NULL DEFAULT '',
	key        TEXT NOT NULL,
	value      TEXT NOT NULL,
	category   TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(tenant_id, key)
);
`,
		},
		{
			Name: "002_seed_categories",
			SQL: `
-- Seed default feedback categories (configurable, not hardcoded in logic)
INSERT INTO feedback_categories (id, tenant_id, name, slug, description, sort_order)
VALUES
	('fc_services',       '', 'Service Delivery',    'service_delivery',    'Issues related to service delivery', 1),
	('fc_facilities',     '', 'Facilities',           'facilities',          'Issues related to facilities and infrastructure', 2),
	('fc_programmes',     '', 'Programmes',           'programmes',          'Issues related to government programmes', 3),
	('fc_staff',          '', 'Staff Conduct',        'staff_conduct',       'Issues related to staff behaviour or conduct', 4),
	('fc_accessibility',  '', 'Accessibility',        'accessibility',       'Issues accessing services', 5),
	('fc_information',    '', 'Information Access',   'information_access',  'Difficulty accessing public information', 6),
	('fc_other',          '', 'Other',                'other',               'Other feedback', 99)
ON CONFLICT (tenant_id, slug) DO NOTHING;

-- Seed default report categories
INSERT INTO report_categories (id, tenant_id, name, slug, description, sort_order)
VALUES
	('rc_service',        '', 'Service Problem',     'service_problem',     'Report a problem with a service', 1),
	('rc_infrastructure', '', 'Infrastructure',      'infrastructure',      'Report infrastructure issues', 2),
	('rc_facility',       '', 'Facility Issue',      'facility_issue',      'Report a facility problem', 3),
	('rc_programme',      '', 'Programme Concern',   'programme_concern',   'Report concerns about a programme', 4),
	('rc_community',      '', 'Community Issue',     'community_issue',     'Report a community problem', 5),
	('rc_emergency',      '', 'Emergency',           'emergency',           'Report an emergency situation', 6),
	('rc_other',          '', 'Other',               'other',               'Other report', 99)
ON CONFLICT (tenant_id, slug) DO NOTHING;

-- Seed default rating dimensions (configurable)
INSERT INTO rating_dimensions (id, tenant_id, slug, label, description, sort_order)
VALUES
	('rd_satisfaction',   '', 'satisfaction',   'Overall Satisfaction',  'How satisfied were you overall?', 1),
	('rd_accessibility',  '', 'accessibility',  'Accessibility',         'How accessible was the service?', 2),
	('rd_wait_time',      '', 'wait_time',       'Waiting Time',          'How reasonable was the waiting time?', 3),
	('rd_availability',   '', 'availability',   'Service Availability',  'Was the service available when needed?', 4),
	('rd_staff',          '', 'staff',           'Staff Experience',      'How was your experience with staff?', 5),
	('rd_quality',        '', 'quality',         'Quality',               'How would you rate the quality?', 6),
	('rd_outcome',        '', 'outcome',         'Outcome',               'Did you achieve your intended outcome?', 7)
ON CONFLICT (tenant_id, slug) DO NOTHING;
`,
		},
	}
}
