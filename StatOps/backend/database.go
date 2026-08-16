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
	dbname := getEnv("DB_NAME", "statops")
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

	log.Println("Connected to PostgreSQL for StatOps")
	db = database
	runMigrations()
	return db, nil
}

func runMigrations() {
	if db == nil {
		return
	}

	schema := `
	CREATE TABLE IF NOT EXISTS pipeline_runs (
		id VARCHAR(100) PRIMARY KEY,
		tenant_id VARCHAR(64) DEFAULT 'tenant-alpha',
		repo_name VARCHAR(255) NOT NULL,
		branch VARCHAR(100) NOT NULL,
		commit_sha VARCHAR(64) NOT NULL,
		commit_msg TEXT,
		author VARCHAR(100),
		status VARCHAR(50) NOT NULL,
		stages JSONB DEFAULT '[]',
		started_at TIMESTAMPTZ DEFAULT NOW(),
		finished_at TIMESTAMPTZ,
		duration_secs INT DEFAULT 0,
		artifact_url VARCHAR(255)
	);

	CREATE TABLE IF NOT EXISTS deployment_records (
		id VARCHAR(100) PRIMARY KEY,
		tenant_id VARCHAR(64) DEFAULT 'tenant-alpha',
		service_name VARCHAR(100) NOT NULL,
		version VARCHAR(50) NOT NULL,
		environment VARCHAR(100) NOT NULL,
		strategy VARCHAR(50) NOT NULL,
		target_cluster VARCHAR(100),
		status VARCHAR(50) NOT NULL,
		canary_percentage INT DEFAULT 100,
		deployed_by VARCHAR(100),
		deployed_at TIMESTAMPTZ DEFAULT NOW(),
		rollback_version VARCHAR(50)
	);

	CREATE TABLE IF NOT EXISTS cloud_clusters (
		id VARCHAR(100) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		provider VARCHAR(100) NOT NULL,
		region VARCHAR(100) NOT NULL,
		status VARCHAR(50) NOT NULL,
		data_boundary VARCHAR(100) NOT NULL,
		total_nodes INT DEFAULT 1,
		allocated_cpu VARCHAR(50),
		allocated_ram VARCHAR(50),
		storage_usage_gb NUMERIC(10, 2) DEFAULT 0,
		k8s_version VARCHAR(50),
		last_heartbeat TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS cmdb_items (
		id VARCHAR(100) PRIMARY KEY,
		item_name VARCHAR(255) NOT NULL,
		item_type VARCHAR(100) NOT NULL,
		environment VARCHAR(100) NOT NULL,
		host_or_url VARCHAR(255) NOT NULL,
		owner_team VARCHAR(100),
		criticality VARCHAR(100) NOT NULL,
		dependencies JSONB DEFAULT '[]',
		status VARCHAR(50) DEFAULT 'OPERATIONAL',
		last_updated TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS alert_incidents (
		id VARCHAR(100) PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		severity VARCHAR(50) NOT NULL,
		status VARCHAR(50) NOT NULL,
		source_app VARCHAR(100) NOT NULL,
		description TEXT,
		triggered_at TIMESTAMPTZ DEFAULT NOW(),
		acknowledged_by VARCHAR(100),
		resolved_at TIMESTAMPTZ,
		runbook_link VARCHAR(255)
	);

	CREATE TABLE IF NOT EXISTS slo_budgets (
		id VARCHAR(100) PRIMARY KEY,
		service_name VARCHAR(100) NOT NULL,
		slo_name VARCHAR(100) NOT NULL,
		target_percent NUMERIC(5, 2) NOT NULL,
		current_percent NUMERIC(5, 2) NOT NULL,
		error_budget_remaining NUMERIC(5, 2) NOT NULL,
		burn_rate_status VARCHAR(50) DEFAULT 'NORMAL',
		period VARCHAR(20) DEFAULT '30d',
		last_calculated TIMESTAMPTZ DEFAULT NOW()
	);
	`
	_, err := db.Exec(schema)
	if err != nil {
		log.Printf("Error executing StatOps DB migrations: %v", err)
	} else {
		log.Println("StatOps DB schema migrations applied successfully.")
	}
}
