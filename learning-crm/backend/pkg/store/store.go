package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var db *sql.DB

// DB returns the shared handle for the Enterprise Audit Service.
func DB() *sql.DB { return db }

// IsReady reports database connectivity for the probes.
func IsReady() bool {
	if db == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return db.PingContext(ctx) == nil
}

// Init opens the connection pool and applies the schema (auto-migration).
func Init(dsn string) error {
	if dsn == "" {
		return errors.New("database DSN must be provided")
	}
	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	if err = db.Ping(); err != nil {
		return err
	}
	if err := ensureSchema(context.Background()); err != nil {
		return err
	}
	return ensureSchemaP35(context.Background())
}

func ensureSchema(ctx context.Context) error {
	stmts := []string{
		// ── P24: LMS & CPD ──
		`CREATE TABLE IF NOT EXISTS courses (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, title TEXT NOT NULL,
			slug TEXT NOT NULL, description TEXT, category TEXT, language TEXT,
			credits INT NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'draft',
			created_by TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_courses_tenant ON courses(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS modules (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, course_id TEXT NOT NULL,
			title TEXT NOT NULL, order_index INT NOT NULL DEFAULT 0, content JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_modules_course ON modules(course_id)`,
		`CREATE TABLE IF NOT EXISTS enrollments (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, course_id TEXT NOT NULL,
			user_id TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'enrolled',
			progress DOUBLE PRECISION NOT NULL DEFAULT 0, completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE (tenant_id, course_id, user_id))`,
		`CREATE TABLE IF NOT EXISTS assessments (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, course_id TEXT NOT NULL,
			title TEXT NOT NULL, kind TEXT NOT NULL DEFAULT 'quiz', config JSONB NOT NULL DEFAULT '{}'::jsonb,
			passing_score DOUBLE PRECISION NOT NULL DEFAULT 0, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS attempts (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, assessment_id TEXT NOT NULL,
			user_id TEXT NOT NULL, answers JSONB NOT NULL DEFAULT '{}'::jsonb,
			score DOUBLE PRECISION NOT NULL DEFAULT 0, passed BOOLEAN NOT NULL DEFAULT false,
			completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS certificates (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, course_id TEXT NOT NULL,
			user_id TEXT NOT NULL, number TEXT NOT NULL UNIQUE, kind TEXT NOT NULL DEFAULT 'course',
			credits INT NOT NULL DEFAULT 0, issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			expires_at TIMESTAMPTZ, verify_hash TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS badges (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, course_id TEXT NOT NULL,
			user_id TEXT NOT NULL, badge_type TEXT NOT NULL, issued_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		// ── P26: Commercial CRM ──
		`CREATE TABLE IF NOT EXISTS leads (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			email TEXT NOT NULL, company TEXT, source TEXT, stage TEXT NOT NULL DEFAULT 'lead',
			value DOUBLE PRECISION NOT NULL DEFAULT 0, owner_id TEXT,
			converted BOOLEAN NOT NULL DEFAULT false, converted_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_leads_tenant ON leads(tenant_id, stage)`,
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			type TEXT, industry TEXT, website TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS opportunities (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, account_id TEXT, lead_id TEXT,
			name TEXT NOT NULL, stage TEXT NOT NULL DEFAULT 'discovery',
			amount DOUBLE PRECISION NOT NULL DEFAULT 0, owner_id TEXT, expected_close TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS partners (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT 'reseller', status TEXT NOT NULL DEFAULT 'active',
			contact_email TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS service_requests (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, title TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT 'support', requester TEXT NOT NULL,
			priority TEXT NOT NULL DEFAULT 'medium', status TEXT NOT NULL DEFAULT 'open',
			assignee TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS invoices (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, number TEXT NOT NULL UNIQUE,
			account_id TEXT, partner_id TEXT, currency TEXT NOT NULL DEFAULT 'UGX',
			amount DOUBLE PRECISION NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'draft',
			synced BOOLEAN NOT NULL DEFAULT false, issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			due_at TIMESTAMPTZ, paid_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return nil
}

func newID() string   { return uuid.New().String() }
func nowT() time.Time { return time.Now().UTC() }

func jsonB(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// lower is a local helper; strings stays imported occasionally.
func lower(s string) string { return strings.ToLower(s) }

func ensureSchemaP35(ctx context.Context) error {
	stmts := []string{
		// ── P35: Stakeholder Engagement & Governance Stewardship ──
		`CREATE TABLE IF NOT EXISTS stakeholders (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			kind TEXT NOT NULL, tier TEXT, engagement_level TEXT,
			contact_email TEXT, notes TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_stakeholders_tenant ON stakeholders(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS engagements (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, stakeholder_id TEXT NOT NULL,
			kind TEXT NOT NULL, title TEXT NOT NULL, happened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			notes TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_engagements_stake ON engagements(stakeholder_id)`,
		`CREATE TABLE IF NOT EXISTS stewardship_actions (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, action TEXT NOT NULL,
			entity_type TEXT, entity_id TEXT, owner TEXT, status TEXT NOT NULL DEFAULT 'open',
			due_at TIMESTAMPTZ, updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_stewardship_tenant ON stewardship_actions(tenant_id, status)`,
		// ── Cross-app object linkage contract ──
		`CREATE TABLE IF NOT EXISTS object_links (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL,
			source_type TEXT NOT NULL, source_id TEXT NOT NULL,
			target_type TEXT NOT NULL, target_id TEXT NOT NULL,
			relationship TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_links_source ON object_links(source_type, source_id)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return nil
}