package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() (*sql.DB, error) {
	host := getEnv("DB_HOST", getEnv("POSTGRES_HOST", "postgres"))
	port := getEnv("DB_PORT", getEnv("POSTGRES_PORT", "5432"))
	user := getEnv("DB_USER", getEnv("POSTGRES_USER", "Kaggle"))
	password := getEnv("DB_PASSWORD", getEnv("POSTGRES_PASSWORD", "statgate-dev-secret"))
	dbname := getEnv("DB_NAME", "stattrust")
	sslmode := getEnv("DB_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	database, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Postgres init warning (using in-memory store): %v", err)
		return nil, err
	}

	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)

	if err := database.Ping(); err != nil {
		log.Printf("Postgres ping failed (using in-memory store): %v", err)
		return nil, err
	}

	log.Println("Connected to PostgreSQL for StatTrust")
	db = database
	runMigrations()
	return db, nil
}

func runMigrations() {
	if db == nil {
		return
	}

	schema := `
	CREATE TABLE IF NOT EXISTS security_incidents (
		id VARCHAR(100) PRIMARY KEY,
		tenant_id VARCHAR(64) DEFAULT 'tenant-alpha',
		title VARCHAR(255) NOT NULL,
		severity VARCHAR(50) NOT NULL,
		status VARCHAR(50) NOT NULL,
		threat_category VARCHAR(100),
		source_app VARCHAR(100),
		source_ip VARCHAR(50),
		affected_asset VARCHAR(255),
		assigned_to VARCHAR(100),
		details JSONB DEFAULT '{}',
		remediation_log JSONB DEFAULT '[]',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		resolved_at TIMESTAMPTZ
	);

	CREATE TABLE IF NOT EXISTS privacy_consents (
		id VARCHAR(100) PRIMARY KEY,
		tenant_id VARCHAR(64) DEFAULT 'tenant-alpha',
		subject_id VARCHAR(100) NOT NULL,
		subject_name VARCHAR(255),
		data_scope VARCHAR(100) NOT NULL,
		purpose VARCHAR(255) NOT NULL,
		status VARCHAR(50) NOT NULL,
		granted_at TIMESTAMPTZ DEFAULT NOW(),
		expires_at TIMESTAMPTZ
	);

	CREATE TABLE IF NOT EXISTS audit_ledger (
		index BIGSERIAL PRIMARY KEY,
		tenant_id VARCHAR(64) DEFAULT 'tenant-alpha',
		prev_hash VARCHAR(64) NOT NULL,
		record_hash VARCHAR(64) NOT NULL,
		merkle_root VARCHAR(64) NOT NULL,
		event_type VARCHAR(100) NOT NULL,
		source_app VARCHAR(100) NOT NULL,
		actor_id VARCHAR(255) NOT NULL,
		payload JSONB NOT NULL,
		timestamp TIMESTAMPTZ DEFAULT NOW(),
		nonce BIGINT DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS verifiable_credentials (
		id VARCHAR(100) PRIMARY KEY,
		tenant_id VARCHAR(64) DEFAULT 'tenant-alpha',
		credential_id VARCHAR(255) UNIQUE NOT NULL,
		issuer_did VARCHAR(255) NOT NULL,
		holder_did VARCHAR(255) NOT NULL,
		holder_name VARCHAR(255),
		issuance_date TIMESTAMPTZ DEFAULT NOW(),
		expiration_date TIMESTAMPTZ,
		credential_subject JSONB NOT NULL,
		proof JSONB NOT NULL,
		status VARCHAR(50) DEFAULT 'ACTIVE'
	);

	CREATE TABLE IF NOT EXISTS artifact_provenance (
		id VARCHAR(100) PRIMARY KEY,
		tenant_id VARCHAR(64) DEFAULT 'tenant-alpha',
		artifact_id VARCHAR(255) UNIQUE NOT NULL,
		artifact_name VARCHAR(255) NOT NULL,
		artifact_type VARCHAR(100) NOT NULL,
		origin_app VARCHAR(100) NOT NULL,
		current_owner VARCHAR(255),
		sha256_checksum VARCHAR(64) NOT NULL,
		tsa_timestamp TIMESTAMPTZ DEFAULT NOW(),
		custody_chain JSONB DEFAULT '[]',
		integrity_state VARCHAR(50) DEFAULT 'VERIFIED',
		ledger_index BIGINT
	);

	CREATE TABLE IF NOT EXISTS encryption_keys (
		id VARCHAR(100) PRIMARY KEY,
		tenant_id VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
		algorithm VARCHAR(50) NOT NULL,
		purpose VARCHAR(100) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
		key_ref VARCHAR(255) NOT NULL,
		rotation_cycle VARCHAR(50) DEFAULT 'MONTHLY',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		expires_at TIMESTAMPTZ NOT NULL,
		rotated_at TIMESTAMPTZ,
		rotated_by VARCHAR(255),
		previous_key_id VARCHAR(100),
		ledger_index BIGINT
	);

	CREATE TABLE IF NOT EXISTS pki_certificates (
		id VARCHAR(100) PRIMARY KEY,
		tenant_id VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
		common_name VARCHAR(255) NOT NULL,
		organization VARCHAR(255) NOT NULL,
		organizational_unit VARCHAR(255),
		country VARCHAR(10) DEFAULT 'UG',
		subject_did VARCHAR(255) NOT NULL,
		certificate_pem TEXT NOT NULL,
		public_key_hex VARCHAR(128) NOT NULL,
		serial_number VARCHAR(100) UNIQUE NOT NULL,
		issued_by VARCHAR(255) NOT NULL,
		not_before TIMESTAMPTZ DEFAULT NOW(),
		not_after TIMESTAMPTZ NOT NULL,
		key_usage JSONB DEFAULT '[]',
		status VARCHAR(50) DEFAULT 'VALID',
		revocation_reason TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		ledger_index BIGINT
	);

	CREATE TABLE IF NOT EXISTS timestamp_records (
		id VARCHAR(100) PRIMARY KEY,
		tenant_id VARCHAR(64) NOT NULL DEFAULT 'tenant-alpha',
		artifact_id VARCHAR(255) NOT NULL,
		artifact_type VARCHAR(100) NOT NULL,
		sha256_hash VARCHAR(64) NOT NULL,
		issued_at TIMESTAMPTZ DEFAULT NOW(),
		tsa_signature_hex TEXT NOT NULL,
		tsa_public_key VARCHAR(128) NOT NULL,
		policy_oid VARCHAR(100) DEFAULT '1.3.6.1.4.1.99999.1.1',
		status VARCHAR(50) DEFAULT 'VALID',
		ledger_index BIGINT
	);

	CREATE TABLE IF NOT EXISTS trust_registry (
		did VARCHAR(255) PRIMARY KEY,
		tenant_id VARCHAR(64) DEFAULT 'tenant-alpha',
		organization_name VARCHAR(255) NOT NULL,
		trust_level VARCHAR(100) NOT NULL,
		public_keys JSONB DEFAULT '[]',
		authorized_scopes JSONB DEFAULT '[]',
		status VARCHAR(50) DEFAULT 'ACTIVE',
		registered_at TIMESTAMPTZ DEFAULT NOW()
	);
	`
	_, err := db.Exec(schema)
	if err != nil {
		log.Printf("Error executing StatTrust DB migrations: %v", err)
	} else {
		log.Println("StatTrust DB schema migrations applied successfully.")
	}
}
