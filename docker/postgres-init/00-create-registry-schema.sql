-- Registry identity bootstrap (stage_register/go-backend).
-- Creates the base `users` table the Registry expects; the service's
-- ensureIdentitySecuritySchema (configs/db.go) then adds MFA/refresh/
-- invitation/branding tables on startup. Idempotent.
SELECT 'CREATE DATABASE kaggle' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'kaggle') \gexec

\c kaggle

CREATE TABLE IF NOT EXISTS users (
	id BIGSERIAL PRIMARY KEY,
	email TEXT UNIQUE NOT NULL,
	username TEXT UNIQUE NOT NULL,
	password TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'viewer',
	first_name TEXT NOT NULL DEFAULT '',
	last_name TEXT NOT NULL DEFAULT '',
	organisation TEXT NOT NULL DEFAULT '',
	district_id BIGINT,
	mfa_secret TEXT NOT NULL DEFAULT '',
	mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
	email_verified BOOLEAN NOT NULL DEFAULT TRUE,
	must_change_password BOOLEAN NOT NULL DEFAULT FALSE,
	"createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	"updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
