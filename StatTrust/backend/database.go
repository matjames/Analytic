package main

import (
	"database/sql"
	"encoding/json"
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
	loadTrustPersistence()
	return db, nil
}

func persistLedgerBlock(block AuditLedgerBlock) {
	if db == nil {
		return
	}
	payload, err := json.Marshal(block.Payload)
	if err != nil {
		log.Printf("StatTrust ledger payload serialization failed: %v", err)
		return
	}
	err = db.QueryRow(`
		INSERT INTO audit_ledger
			(chain_index, tenant_id, workspace_id, prev_hash, record_hash, merkle_root, event_type, source_app, actor_id, payload, timestamp, nonce)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING storage_id`,
		block.Index, block.TenantID, nullableWorkspace(block.WorkspaceID), block.PrevHash,
		block.RecordHash, block.MerkleRoot, block.EventType, block.SourceApp, block.ActorID,
		payload, block.Timestamp, block.Nonce).Scan(&block.StorageID)
	if err != nil {
		log.Printf("StatTrust ledger persistence failed: %v", err)
	}
}

func persistProvenance(p ArtifactProvenance) {
	if db == nil {
		return
	}
	chain, err := json.Marshal(p.CustodyChain)
	if err != nil {
		log.Printf("StatTrust provenance serialization failed: %v", err)
		return
	}
	_, err = db.Exec(`
		INSERT INTO artifact_provenance
			(id, tenant_id, workspace_id, artifact_id, artifact_name, artifact_type, origin_app, current_owner, sha256_checksum, tsa_timestamp, custody_chain, integrity_state, ledger_index)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id,
			workspace_id = EXCLUDED.workspace_id,
			artifact_id = EXCLUDED.artifact_id,
			artifact_name = EXCLUDED.artifact_name,
			artifact_type = EXCLUDED.artifact_type,
			origin_app = EXCLUDED.origin_app,
			current_owner = EXCLUDED.current_owner,
			sha256_checksum = EXCLUDED.sha256_checksum,
			tsa_timestamp = EXCLUDED.tsa_timestamp,
			custody_chain = EXCLUDED.custody_chain,
			integrity_state = EXCLUDED.integrity_state,
			ledger_index = EXCLUDED.ledger_index`,
		p.ID, p.TenantID, nullableWorkspace(p.WorkspaceID), p.ArtifactID, p.ArtifactName, p.ArtifactType,
		p.OriginApp, p.CurrentOwner, p.Sha256Checksum, p.TSATimestamp, chain, p.IntegrityState, p.LedgerIndex)
	if err != nil {
		log.Printf("StatTrust provenance persistence failed: %v", err)
	}
}

func loadTrustPersistence() {
	if db == nil || globalStore == nil {
		return
	}
	rows, err := db.Query(`
		SELECT storage_id, chain_index, tenant_id, COALESCE(workspace_id, ''), prev_hash, record_hash, merkle_root,
			event_type, source_app, actor_id, payload, timestamp, nonce
		FROM audit_ledger ORDER BY storage_id`)
	if err == nil {
		defer rows.Close()
		globalStore.mu.Lock()
		for rows.Next() {
			var block AuditLedgerBlock
			var payload []byte
			if err := rows.Scan(&block.StorageID, &block.Index, &block.TenantID, &block.WorkspaceID, &block.PrevHash,
				&block.RecordHash, &block.MerkleRoot, &block.EventType, &block.SourceApp, &block.ActorID, &payload,
				&block.Timestamp, &block.Nonce); err != nil {
				log.Printf("StatTrust ledger load failed: %v", err)
				continue
			}
			if err := json.Unmarshal(payload, &block.Payload); err == nil {
				globalStore.ledger = append(globalStore.ledger, block)
			}
		}
		globalStore.mu.Unlock()
	} else {
		log.Printf("StatTrust ledger load skipped: %v", err)
	}

	rows, err = db.Query(`
		SELECT id, tenant_id, COALESCE(workspace_id, ''), artifact_id, artifact_name, artifact_type, origin_app,
			current_owner, sha256_checksum, tsa_timestamp, custody_chain, integrity_state, ledger_index
		FROM artifact_provenance`)
	if err != nil {
		log.Printf("StatTrust provenance load skipped: %v", err)
		return
	}
	defer rows.Close()
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	for rows.Next() {
		var p ArtifactProvenance
		var chain []byte
		if err := rows.Scan(&p.ID, &p.TenantID, &p.WorkspaceID, &p.ArtifactID, &p.ArtifactName, &p.ArtifactType,
			&p.OriginApp, &p.CurrentOwner, &p.Sha256Checksum, &p.TSATimestamp, &chain, &p.IntegrityState, &p.LedgerIndex); err != nil {
			log.Printf("StatTrust provenance load failed: %v", err)
			continue
		}
		if err := json.Unmarshal(chain, &p.CustodyChain); err == nil {
			globalStore.provenance[provenanceKey(p.TenantID, p.WorkspaceID, p.ArtifactID)] = p
		}
	}
}

func nullableWorkspace(workspaceID string) interface{} {
	if workspaceID == "" {
		return nil
	}
	return workspaceID
}

func runMigrations() {
	if db == nil {
		return
	}

	schema := `
	CREATE TABLE IF NOT EXISTS security_incidents (
		id VARCHAR(100) PRIMARY KEY,
		tenant_id VARCHAR(64) DEFAULT 'tenant-alpha',
		workspace_id VARCHAR(128),
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
		storage_id BIGSERIAL PRIMARY KEY,
		chain_index BIGINT NOT NULL,
		tenant_id VARCHAR(64) DEFAULT 'tenant-alpha',
		workspace_id VARCHAR(128),
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
		workspace_id VARCHAR(128),
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
		artifact_id VARCHAR(255) NOT NULL,
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
	for _, table := range []string{"security_incidents", "audit_ledger", "artifact_provenance"} {
		if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)", table)); err != nil {
			log.Printf("Error adding workspace scope to %s: %v", table, err)
		}
	}
	if _, err := db.Exec(`
		ALTER TABLE audit_ledger ADD COLUMN IF NOT EXISTS storage_id BIGSERIAL;
		ALTER TABLE audit_ledger ADD COLUMN IF NOT EXISTS chain_index BIGINT;
	`); err != nil {
		log.Printf("Error migrating audit ledger storage identity: %v", err)
	} else {
		var hasLegacyIndex bool
		err := db.QueryRow(`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'audit_ledger' AND column_name = 'index'
		)`).Scan(&hasLegacyIndex)
		if err != nil {
			log.Printf("Error checking legacy audit ledger index: %v", err)
		} else {
			backfill := `UPDATE audit_ledger SET chain_index = storage_id WHERE chain_index IS NULL`
			if hasLegacyIndex {
				backfill = `UPDATE audit_ledger SET chain_index = "index" WHERE chain_index IS NULL`
			}
			if _, err := db.Exec(backfill); err != nil {
				log.Printf("Error backfilling audit ledger chain index: %v", err)
			}
		}
		if _, err := db.Exec(`
			ALTER TABLE audit_ledger ALTER COLUMN chain_index SET NOT NULL;
			ALTER TABLE audit_ledger DROP CONSTRAINT IF EXISTS audit_ledger_pkey;
			ALTER TABLE audit_ledger ADD CONSTRAINT audit_ledger_pkey PRIMARY KEY (storage_id);
		`); err != nil {
			log.Printf("Error finalizing audit ledger storage identity: %v", err)
		}
	}
	if _, err := db.Exec(`
		ALTER TABLE artifact_provenance DROP CONSTRAINT IF EXISTS artifact_provenance_artifact_id_key;
		CREATE UNIQUE INDEX IF NOT EXISTS artifact_provenance_scope_key
			ON artifact_provenance (tenant_id, COALESCE(workspace_id, ''), artifact_id);
	`); err != nil {
		log.Printf("Error migrating artifact provenance scope uniqueness: %v", err)
	}
}
