package configs

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func ConnectDB() {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("REGISTRY_DB_HOST"),
		os.Getenv("REGISTRY_DB_PORT"),
		os.Getenv("REGISTRY_DB_USER"),
		os.Getenv("REGISTRY_DB_PASSWORD"),
		os.Getenv("REGISTRY_DB_NAME"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	DB = db
	if _, err := DB.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_secret TEXT NOT NULL DEFAULT '';
		ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE;
		ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT TRUE;
		CREATE TABLE IF NOT EXISTS user_invitations (
			id BIGSERIAL PRIMARY KEY,
			email TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'viewer',
			organisation TEXT NOT NULL DEFAULT '',
			token_hash TEXT NOT NULL UNIQUE,
			invited_by BIGINT NOT NULL REFERENCES users(id),
			expires_at TIMESTAMPTZ NOT NULL,
			accepted_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_user_invitations_pending ON user_invitations (email, expires_at) WHERE accepted_at IS NULL;
		CREATE TABLE IF NOT EXISTS refresh_sessions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			revoked_at TIMESTAMPTZ,
			user_agent TEXT NOT NULL DEFAULT '',
			ip_hash TEXT NOT NULL DEFAULT '',
			last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		ALTER TABLE refresh_sessions ADD COLUMN IF NOT EXISTS user_agent TEXT NOT NULL DEFAULT '';
		ALTER TABLE refresh_sessions ADD COLUMN IF NOT EXISTS ip_hash TEXT NOT NULL DEFAULT '';
		ALTER TABLE refresh_sessions ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
		ALTER TABLE refresh_sessions ADD COLUMN IF NOT EXISTS geo_country TEXT NOT NULL DEFAULT '';
		ALTER TABLE refresh_sessions ADD COLUMN IF NOT EXISTS geo_region TEXT NOT NULL DEFAULT '';
		ALTER TABLE refresh_sessions ADD COLUMN IF NOT EXISTS geo_anomaly BOOLEAN NOT NULL DEFAULT FALSE;
		CREATE INDEX IF NOT EXISTS idx_refresh_sessions_user ON refresh_sessions (user_id, expires_at) WHERE revoked_at IS NULL;
		CREATE INDEX IF NOT EXISTS idx_refresh_sessions_geo_anomaly ON refresh_sessions (user_id, created_at) WHERE geo_anomaly = TRUE;
		CREATE TABLE IF NOT EXISTS email_verification_tokens (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_user ON email_verification_tokens (user_id, expires_at) WHERE used_at IS NULL;
		CREATE TABLE IF NOT EXISTS user_preferences (
			user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			locale TEXT NOT NULL DEFAULT 'en',
			timezone TEXT NOT NULL DEFAULT 'UTC',
			theme TEXT NOT NULL DEFAULT 'system',
			density TEXT NOT NULL DEFAULT 'comfortable',
			email_notifications BOOLEAN NOT NULL DEFAULT TRUE,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			code_hash TEXT NOT NULL UNIQUE,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_mfa_recovery_codes_user ON mfa_recovery_codes (user_id) WHERE used_at IS NULL;
		CREATE TABLE IF NOT EXISTS organisation_branding (
			tenant_id TEXT PRIMARY KEY,
			display_name TEXT NOT NULL DEFAULT '',
			logo_url TEXT NOT NULL DEFAULT '',
			primary_color TEXT NOT NULL DEFAULT '#0f766e',
			secondary_color TEXT NOT NULL DEFAULT '#0f3d3e',
			updated_by BIGINT REFERENCES users(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`); err != nil {
		panic(fmt.Errorf("ensure identity security schema: %w", err))
	}
	fmt.Println("✅ Database connected")
}
