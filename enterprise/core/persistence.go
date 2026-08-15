package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// ─── Database Pool ────────────────────────────────────────────────

var dbPool *sql.DB

// initDB opens the PostgreSQL connection pool and verifies connectivity.
// In production (STATGATE_ENV=production) with FAIL_FAST_WITHOUT_DB=true,
// failure is fatal. In development it is a logged warning only.
func initDB() {
	dsn := buildDSN()
	if dsn == "" {
		log.Println("persistence: STATGATE_DB_URL not set — database persistence disabled")
		return
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		handleDBInitError(fmt.Errorf("failed to open database: %w", err))
		return
	}

	// Pool configuration
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		handleDBInitError(fmt.Errorf("database ping failed: %w", err))
		return
	}

	dbPool = db
	log.Println("persistence: PostgreSQL connected")

	// Run schema ensures — idempotent DDL
	if err := ensurePlatformSchema(db); err != nil {
		log.Printf("persistence: schema migration warning: %v", err)
	}
}

func buildDSN() string {
	if url := os.Getenv("STATGATE_DB_URL"); url != "" {
		return url
	}
	host := getEnvValue("STATGATE_DB_HOST")
	if host == "" {
		return ""
	}
	port := getEnvValue("STATGATE_DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := getEnvValue("STATGATE_DB_USER")
	pass := getEnvValue("STATGATE_DB_PASSWORD")
	dbname := getEnvValue("STATGATE_DB_NAME")
	if dbname == "" {
		dbname = "statgate_enterprise"
	}
	sslmode := getEnvValue("STATGATE_DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, dbname, sslmode)
}

func handleDBInitError(err error) {
	env := os.Getenv("STATGATE_ENV")
	failFast := os.Getenv("FAIL_FAST_WITHOUT_DB") == "true"
	if env == "production" || failFast {
		log.Fatalf("persistence: [FATAL] %v", err)
	}
	log.Printf("persistence: [WARNING] %v — running without database persistence", err)
}

// ─── Schema Migration (Idempotent DDL) ───────────────────────────

func ensurePlatformSchema(db *sql.DB) error {
	statements := []string{
		// Platform notifications — durable, survives Redis restarts
		`CREATE TABLE IF NOT EXISTS platform_notifications (
			id            TEXT PRIMARY KEY,
			user_id       TEXT NOT NULL,
			tenant_id     TEXT NOT NULL DEFAULT '',
			title         TEXT NOT NULL,
			body          TEXT NOT NULL,
			priority      TEXT NOT NULL DEFAULT 'medium',
			category      TEXT NOT NULL DEFAULT 'system',
			source_app    TEXT NOT NULL DEFAULT '',
			source_entity TEXT NOT NULL DEFAULT '',
			source_entity_id TEXT NOT NULL DEFAULT '',
			read          BOOLEAN NOT NULL DEFAULT FALSE,
			archived      BOOLEAN NOT NULL DEFAULT FALSE,
			deep_link     TEXT NOT NULL DEFAULT '',
			metadata      JSONB,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_notifications_user ON platform_notifications(user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_notifications_tenant ON platform_notifications(tenant_id)`,

		// Platform timeline — institutional activity log
		`CREATE TABLE IF NOT EXISTS platform_timeline (
			id            TEXT PRIMARY KEY,
			user_id       TEXT NOT NULL DEFAULT '',
			tenant_id     TEXT NOT NULL DEFAULT '',
			application   TEXT NOT NULL DEFAULT '',
			entity        TEXT NOT NULL DEFAULT '',
			entity_id     TEXT NOT NULL DEFAULT '',
			action        TEXT NOT NULL,
			description   TEXT NOT NULL DEFAULT '',
			project_id    TEXT NOT NULL DEFAULT '',
			org_id        TEXT NOT NULL DEFAULT '',
			metadata      JSONB,
			timestamp     TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_timeline_ts ON platform_timeline(timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_timeline_entity ON platform_timeline(entity, entity_id)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_timeline_user ON platform_timeline(user_id)`,

		// Platform audit log — immutable, append-only record of all significant actions
		`CREATE TABLE IF NOT EXISTS platform_audit_log (
			id            BIGSERIAL PRIMARY KEY,
			actor         TEXT NOT NULL DEFAULT '',
			action        TEXT NOT NULL,
			resource      TEXT NOT NULL DEFAULT '',
			resource_id   TEXT NOT NULL DEFAULT '',
			tenant_id     TEXT NOT NULL DEFAULT '',
			correlation_id TEXT NOT NULL DEFAULT '',
			request_id    TEXT NOT NULL DEFAULT '',
			outcome       TEXT NOT NULL DEFAULT 'success',
			metadata      JSONB,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_audit_actor ON platform_audit_log(actor)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_audit_resource ON platform_audit_log(resource, resource_id)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_audit_ts ON platform_audit_log(created_at DESC)`,

		// Platform events — durable event store for replay and DLQ
		`CREATE TABLE IF NOT EXISTS platform_events (
			id             TEXT PRIMARY KEY,
			event_type     TEXT NOT NULL,
			source         TEXT NOT NULL DEFAULT '',
			object_type    TEXT NOT NULL DEFAULT '',
			object_id      TEXT NOT NULL DEFAULT '',
			actor          TEXT NOT NULL DEFAULT '',
			tenant_id      TEXT NOT NULL DEFAULT '',
			project_id     TEXT NOT NULL DEFAULT '',
			org_id         TEXT NOT NULL DEFAULT '',
			correlation_id TEXT NOT NULL DEFAULT '',
			causation_id   TEXT NOT NULL DEFAULT '',
			schema_version INTEGER NOT NULL DEFAULT 1,
			payload        JSONB,
			status         TEXT NOT NULL DEFAULT 'published',
			retry_count    INTEGER NOT NULL DEFAULT 0,
			last_error     TEXT NOT NULL DEFAULT '',
			created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			processed_at   TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_events_type ON platform_events(event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_events_status ON platform_events(status)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_events_correlation ON platform_events(correlation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_platform_events_ts ON platform_events(created_at DESC)`,

		// Service registry — authoritative list of all StatGate services
		`CREATE TABLE IF NOT EXISTS platform_service_registry (
			id            TEXT PRIMARY KEY,
			name          TEXT NOT NULL,
			display_name  TEXT NOT NULL DEFAULT '',
			description   TEXT NOT NULL DEFAULT '',
			api_url       TEXT NOT NULL DEFAULT '',
			ui_url        TEXT NOT NULL DEFAULT '',
			health_url    TEXT NOT NULL DEFAULT '',
			version       TEXT NOT NULL DEFAULT '',
			capabilities  JSONB,
			status        TEXT NOT NULL DEFAULT 'registered',
			registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_heartbeat TIMESTAMPTZ
		)`,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("schema statement failed: %w", err)
		}
	}
	log.Println("persistence: platform schema ensured")
	return nil
}

// ─── Helper ───────────────────────────────────────────────────────

// marshalJSON marshals a value to JSON string — used across persistence helpers.
func marshalJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
