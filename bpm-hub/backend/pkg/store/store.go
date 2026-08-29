package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var db *sql.DB

func DB() *sql.DB { return db }

func IsReady() bool {
	if db == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return db.PingContext(ctx) == nil
}

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
	return ensureSchema(context.Background())
}

func ensureSchema(ctx context.Context) error {
	stmts := []string{
		// ── BPM definitions & instances (P48) ──
		`CREATE TABLE IF NOT EXISTS process_definitions (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			key TEXT NOT NULL, version INT NOT NULL DEFAULT 1, description TEXT,
			start_node TEXT NOT NULL, nodes JSONB NOT NULL DEFAULT '{}'::jsonb,
			transitions JSONB NOT NULL DEFAULT '{}'::jsonb, status TEXT NOT NULL DEFAULT 'draft',
			created_by TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_pd_tenant ON process_definitions(tenant_id, key)`,
		`CREATE TABLE IF NOT EXISTS process_instances (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, definition_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'created', current_node TEXT NOT NULL,
			context JSONB NOT NULL DEFAULT '{}'::jsonb, started_at TIMESTAMPTZ,
			ended_at TIMESTAMPTZ, created_by TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_pi_tenant ON process_instances(tenant_id, status)`,
		// ── Case management (P48) ──
		`CREATE TABLE IF NOT EXISTS case_instances (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, key TEXT NOT NULL,
			name TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'open',
			priority TEXT NOT NULL DEFAULT 'medium', owner TEXT, due_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_ci_tenant ON case_instances(tenant_id, status)`,
		`CREATE TABLE IF NOT EXISTS case_items (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, case_id TEXT NOT NULL,
			item_type TEXT NOT NULL, content JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		// ── Task management (P48) ──
		`CREATE TABLE IF NOT EXISTS work_items (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, instance_id TEXT,
			node_id TEXT, name TEXT NOT NULL, assignee TEXT,
			status TEXT NOT NULL DEFAULT 'todo', payload JSONB NOT NULL DEFAULT '{}'::jsonb,
			due_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_wi_assignee ON work_items(assignee, status)`,
		// ── Process mining (P48) ──
		`CREATE TABLE IF NOT EXISTS activity_logs (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, instance_id TEXT,
			case_id TEXT, action TEXT NOT NULL, node_id TEXT, actor TEXT,
			meta JSONB NOT NULL DEFAULT '{}'::jsonb, at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_al_instance ON activity_logs(instance_id, at)`,
		// ── Automation service (P48) ──
		`CREATE TABLE IF NOT EXISTS automation_rules (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			trigger_event TEXT NOT NULL, condition JSONB NOT NULL DEFAULT '{}'::jsonb,
			actions JSONB NOT NULL DEFAULT '{}'::jsonb, enabled BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		// ── Object linkage ──
		`CREATE TABLE IF NOT EXISTS object_links (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL,
			source_type TEXT NOT NULL, source_id TEXT NOT NULL,
			target_type TEXT NOT NULL, target_id TEXT NOT NULL,
			relationship TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_links_source ON object_links(source_type, source_id)`,
		// ── Stage 2: workspace scoping ──
		`ALTER TABLE process_definitions ADD COLUMN IF NOT EXISTS workspace_id TEXT`,
		`ALTER TABLE process_instances ADD COLUMN IF NOT EXISTS workspace_id TEXT`,
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

type row interface{ Scan(...interface{}) error }

func jsonUnmarshal(b []byte, v interface{}) {
	if len(b) > 0 {
		_ = json.Unmarshal(b, v)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }