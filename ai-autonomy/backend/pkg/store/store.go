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
	return ensureSchema(context.Background())
}

func ensureSchema(ctx context.Context) error {
	stmts := []string{
		// ── P22: Digital Twins & Simulations ──
		`CREATE TABLE IF NOT EXISTS digital_twins (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			description TEXT, entity_type TEXT NOT NULL, entity_id TEXT,
			parameters JSONB NOT NULL DEFAULT '{}'::jsonb, state JSONB NOT NULL DEFAULT '{}'::jsonb,
			status TEXT NOT NULL DEFAULT 'active', created_by TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_twins_tenant ON digital_twins(tenant_id)`,

		`CREATE TABLE IF NOT EXISTS simulations (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, twin_id TEXT NOT NULL,
			name TEXT NOT NULL, scenario JSONB NOT NULL DEFAULT '{}'::jsonb,
			input JSONB NOT NULL DEFAULT '{}'::jsonb, results JSONB NOT NULL DEFAULT '{}'::jsonb,
			status TEXT NOT NULL DEFAULT 'queued', error TEXT, started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ, created_by TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_simulations_twin ON simulations(twin_id)`,

		// ── P22: Models, Predictions & Pipelines ──
		`CREATE TABLE IF NOT EXISTS ai_models (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			kind TEXT NOT NULL, version TEXT NOT NULL DEFAULT '1.0',
			config JSONB NOT NULL DEFAULT '{}'::jsonb, status TEXT NOT NULL DEFAULT 'draft',
			created_by TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,

		`CREATE TABLE IF NOT EXISTS predictions (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, model_id TEXT NOT NULL,
			target_type TEXT NOT NULL, target_id TEXT NOT NULL,
			fields JSONB NOT NULL DEFAULT '{}'::jsonb, prediction JSONB NOT NULL DEFAULT '{}'::jsonb,
			confidence DOUBLE PRECISION NOT NULL DEFAULT 0, horizon TEXT,
			status TEXT NOT NULL DEFAULT 'in_progress', triggered_by TEXT NOT NULL DEFAULT 'manual',
			source_event TEXT, created_by TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_predictions_model ON predictions(model_id)`,

		`CREATE TABLE IF NOT EXISTS pipelines (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			description TEXT, definition JSONB NOT NULL DEFAULT '{}'::jsonb,
			status TEXT NOT NULL DEFAULT 'active', created_by TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
	// ── P31: Agents, Tasks, Memory, Messages, Governance ──
		`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			role TEXT NOT NULL, persona JSONB NOT NULL DEFAULT '{}'::jsonb,
			capabilities JSONB NOT NULL DEFAULT '{}'::jsonb, safety_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
			status TEXT NOT NULL DEFAULT 'registered', created_by TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_agents_tenant ON agents(tenant_id)`,

		`CREATE TABLE IF NOT EXISTS agent_tasks (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, agent_id TEXT NOT NULL,
			name TEXT NOT NULL, task_type TEXT NOT NULL, payload JSONB NOT NULL DEFAULT '{}'::jsonb,
			trigger_type TEXT NOT NULL DEFAULT 'manual', triggered_by TEXT,
			status TEXT NOT NULL DEFAULT 'queued', result JSONB NOT NULL DEFAULT '{}'::jsonb,
			error TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			started_at TIMESTAMPTZ, finished_at TIMESTAMPTZ)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_agent ON agent_tasks(agent_id, status)`,

		`CREATE TABLE IF NOT EXISTS agent_memory (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, agent_id TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT 'observation', content JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_memory_agent ON agent_memory(agent_id)`,

		`CREATE TABLE IF NOT EXISTS agent_messages (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, from_agent TEXT NOT NULL,
			to_agent TEXT NOT NULL, type TEXT NOT NULL DEFAULT 'inform',
			payload JSONB NOT NULL DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_messages_to ON agent_messages(to_agent)`,

		`CREATE TABLE IF NOT EXISTS governance_actions (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, agent_id TEXT,
			action TEXT NOT NULL, decision JSONB NOT NULL DEFAULT '{}'::jsonb,
			risk_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			verdict TEXT NOT NULL DEFAULT 'human_review', reviewed_by TEXT, reviewed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,

		// ── P39: Knowledge Graph, Triplestore & Context Index ──
		`CREATE TABLE IF NOT EXISTS graph_nodes (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, type TEXT NOT NULL,
			label TEXT NOT NULL, properties JSONB NOT NULL DEFAULT '{}'::jsonb,
			ref_object_type TEXT, ref_object_id TEXT, created_by TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_tenant ON graph_nodes(tenant_id)`,

		`CREATE TABLE IF NOT EXISTS graph_edges (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, source_node TEXT NOT NULL,
			target_node TEXT NOT NULL, predicate TEXT NOT NULL,
			properties JSONB NOT NULL DEFAULT '{}'::jsonb, created_by TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_edges_source ON graph_edges(source_node)`,

		`CREATE TABLE IF NOT EXISTS graph_statements (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, subject TEXT NOT NULL,
			predicate TEXT NOT NULL, object TEXT NOT NULL,
			confidence DOUBLE PRECISION NOT NULL DEFAULT 1,
			provenance TEXT, created_by TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_statements_spo ON graph_statements(subject, predicate, object)`,

		`CREATE TABLE IF NOT EXISTS context_index (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL, content TEXT NOT NULL, tokens JSONB NOT NULL DEFAULT '[]'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_context_entity ON context_index(entity_type, entity_id)`,

		// ── Object linkage (cross-app contract) ──
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
func newID() string   { return uuid.New().String() }
func nowT() time.Time { return time.Now().UTC() }

// jsonString is a small helper used by CRUD functions.
func jsonB(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// tokenize splits free text into lowercase terms for the context indexer.
func tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	seen := map[string]bool{}
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) > 1 && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}