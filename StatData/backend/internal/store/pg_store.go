package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"statdata-backend/internal/models"
)

// PGStore implements Store using PostgreSQL with statdata schema isolation
type PGStore struct {
	db *sql.DB
}

// NewPGStore creates PostgreSQL storage adapter
func NewPGStore(db *sql.DB) *PGStore {
	p := &PGStore{db: db}
	p.ensureSchema()
	return p
}

// ensureSchema self-provisions the tables owned by this service and the
// workspace scoping columns (Stage 2: identity/security closure). Idempotent;
// safe on every startup and against a fresh database.
func (p *PGStore) ensureSchema() {
	stmts := []string{
		`CREATE SCHEMA IF NOT EXISTS statdata`,
		`CREATE TABLE IF NOT EXISTS statdata.datasets (
			id VARCHAR(64) PRIMARY KEY,
			urn VARCHAR(255) UNIQUE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			domain VARCHAR(64),
			classification VARCHAR(32),
			owner_team VARCHAR(128),
			owner_email VARCHAR(255),
			format VARCHAR(32),
			storage_uri TEXT,
			schema_id VARCHAR(64),
			version VARCHAR(32),
			quality_score DOUBLE PRECISION DEFAULT 0,
			row_count BIGINT DEFAULT 0,
			size_bytes BIGINT DEFAULT 0,
			tags JSONB,
			metadata JSONB,
			tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
			created_by VARCHAR(128),
			workspace_id VARCHAR(128),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS statdata.pipelines (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			pipeline_type VARCHAR(32),
			status VARCHAR(32),
			cron_schedule VARCHAR(64),
			source_dataset_id VARCHAR(64),
			target_dataset_id VARCHAR(64),
			stages JSONB,
			config JSONB,
			max_retries INT DEFAULT 0,
			timeout_seconds INT DEFAULT 0,
			tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
			created_by VARCHAR(128),
			last_run_at TIMESTAMPTZ,
			workspace_id VARCHAR(128),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE statdata.datasets ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`ALTER TABLE statdata.pipelines ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`ALTER TABLE statdata.data_sources ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`ALTER TABLE statdata.streaming_jobs ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		`ALTER TABLE statdata.feature_views ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(128)`,
		// ── Remaining domains (previously missing entirely) ──
		`CREATE TABLE IF NOT EXISTS statdata.data_sources (
			id VARCHAR(64) PRIMARY KEY, name VARCHAR(255) NOT NULL, source_type VARCHAR(32),
			connection_uri TEXT, auth_type VARCHAR(32), credentials JSONB, status VARCHAR(32),
			tenant_id VARCHAR(64) NOT NULL DEFAULT 'default', workspace_id VARCHAR(128),
			last_tested_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS statdata.schema_registry (
			id VARCHAR(64) PRIMARY KEY, subject VARCHAR(128) NOT NULL, version INT NOT NULL,
			schema_type VARCHAR(32), schema_content TEXT, compatibility VARCHAR(32),
			description TEXT, fields JSONB, tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
			created_by VARCHAR(128), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS statdata.data_contracts (
			id VARCHAR(64) PRIMARY KEY, dataset_id VARCHAR(64), title VARCHAR(255) NOT NULL,
			producer_team VARCHAR(128), consumer_team VARCHAR(128), status VARCHAR(32),
			max_latency_mins INT, min_quality_rate DOUBLE PRECISION, expected_volume BIGINT,
			sla_config JSONB, tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
			valid_from TIMESTAMPTZ, valid_until TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS statdata.data_quality_rules (
			id VARCHAR(64) PRIMARY KEY, dataset_id VARCHAR(64), rule_name VARCHAR(128) NOT NULL,
			rule_type VARCHAR(32), target_field VARCHAR(128), parameters JSONB,
			severity VARCHAR(16), is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
			tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS statdata.data_quality_reports (
			id VARCHAR(64) PRIMARY KEY, dataset_id VARCHAR(64), pipeline_run_id VARCHAR(64),
			status VARCHAR(32), quality_score DOUBLE PRECISION, total_rules INT,
			passed_rules INT, failed_rules INT, rule_results JSONB,
			evaluated_at TIMESTAMPTZ, tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS statdata.lineage_nodes (
			id VARCHAR(64) PRIMARY KEY, urn VARCHAR(255), type VARCHAR(32), name VARCHAR(255),
			domain VARCHAR(64), tenant_id VARCHAR(64) NOT NULL DEFAULT 'default', metadata JSONB)`,
		`CREATE TABLE IF NOT EXISTS statdata.lineage_edges (
			id VARCHAR(64) PRIMARY KEY, source_node_id VARCHAR(64), target_node_id VARCHAR(64),
			relation_type VARCHAR(32), pipeline_id VARCHAR(64), transformation TEXT,
			tenant_id VARCHAR(64) NOT NULL DEFAULT 'default')`,
		`CREATE TABLE IF NOT EXISTS statdata.pipeline_runs (
			id VARCHAR(64) PRIMARY KEY, pipeline_id VARCHAR(64), pipeline_name VARCHAR(255),
			status VARCHAR(32), trigger_type VARCHAR(32), started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ, duration_ms BIGINT, records_read BIGINT,
			records_written BIGINT, records_rejected BIGINT, stage_runs JSONB,
			error_message TEXT, metrics JSONB, tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
			triggered_by VARCHAR(128), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS statdata.streaming_jobs (
			id VARCHAR(64) PRIMARY KEY, name VARCHAR(255) NOT NULL, source_topic VARCHAR(128),
			target_sink VARCHAR(128), status VARCHAR(32), throughput_msg_sec DOUBLE PRECISION DEFAULT 0,
			lag_records BIGINT DEFAULT 0, config JSONB,
			tenant_id VARCHAR(64) NOT NULL DEFAULT 'default', workspace_id VARCHAR(128),
			last_checkpoint TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS statdata.feature_views (
			id VARCHAR(64) PRIMARY KEY, name VARCHAR(255) NOT NULL, entity_name VARCHAR(128),
			description TEXT, ttl_seconds BIGINT DEFAULT 0, features JSONB,
			source_query TEXT, online_store BOOLEAN DEFAULT FALSE, offline_sink TEXT,
			tags JSONB, tenant_id VARCHAR(64) NOT NULL DEFAULT 'default', workspace_id VARCHAR(128),
			created_by VARCHAR(128), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS statdata.feature_records (
			entity_key VARCHAR(128) NOT NULL, feature_view_id VARCHAR(64) NOT NULL,
			"values" JSONB NOT NULL DEFAULT '{}'::jsonb, timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
			PRIMARY KEY (entity_key, feature_view_id))`,
		`CREATE TABLE IF NOT EXISTS statdata.notebook_sessions (
			id VARCHAR(64) PRIMARY KEY, title VARCHAR(255), language VARCHAR(32),
			kernel_state VARCHAR(32), dataset_refs JSONB, cells JSONB, variables JSONB,
			tenant_id VARCHAR(64) NOT NULL DEFAULT 'default', created_by VARCHAR(128),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS statdata.experiments (
			id VARCHAR(64) PRIMARY KEY, name VARCHAR(255) NOT NULL, description TEXT,
			domain VARCHAR(64), tags JSONB, artifact_uri TEXT,
			tenant_id VARCHAR(64) NOT NULL DEFAULT 'default', created_by VARCHAR(128),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
	}
	for _, stmt := range stmts {
		if _, err := p.db.Exec(stmt); err != nil {
			log.Printf("statdata: schema ensure failed: %v", err)
		}
	}
}

// ─── Data Catalog & Sources Implementations ─────────────────────────────────

func (p *PGStore) CreateDataset(ctx context.Context, ds *models.Dataset) error {
	if ds.ID == "" {
		ds.ID = "ds-" + uuid.New().String()[:8]
	}
	if ds.URN == "" {
		ds.URN = fmt.Sprintf("urn:statgate:dataset:%s:%s", ds.Domain, ds.ID)
	}
	tagsJSON, _ := json.Marshal(ds.Tags)
	metaJSON, _ := json.Marshal(ds.Metadata)

	query := `
		INSERT INTO statdata.datasets (
			id, urn, name, description, domain, classification,
			owner_team, owner_email, format, storage_uri, schema_id,
			version, quality_score, row_count, size_bytes, tags,
			metadata, tenant_id, created_by, workspace_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			quality_score = EXCLUDED.quality_score,
			row_count = EXCLUDED.row_count,
			size_bytes = EXCLUDED.size_bytes,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := p.db.ExecContext(ctx, query,
		ds.ID, ds.URN, ds.Name, ds.Description, ds.Domain, string(ds.Classification),
		ds.OwnerTeam, ds.OwnerEmail, ds.Format, ds.StorageURI, ds.SchemaID,
		ds.Version, ds.QualityScore, ds.RowCount, ds.SizeBytes, tagsJSON,
		metaJSON, ds.TenantID, ds.CreatedBy, ds.WorkspaceID,
	)
	return err
}

func (p *PGStore) GetDatasetByID(ctx context.Context, id string) (*models.Dataset, error) {
	query := `
		SELECT id, urn, name, description, domain, classification,
		       owner_team, owner_email, format, storage_uri, COALESCE(schema_id, ''),
		       version, quality_score, row_count, size_bytes, tags,
		       metadata, tenant_id, created_by, COALESCE(workspace_id, ''), created_at, updated_at
		FROM statdata.datasets WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var ds models.Dataset
	var tagsJSON, metaJSON []byte
	var classStr string

	err := row.Scan(
		&ds.ID, &ds.URN, &ds.Name, &ds.Description, &ds.Domain, &classStr,
		&ds.OwnerTeam, &ds.OwnerEmail, &ds.Format, &ds.StorageURI, &ds.SchemaID,
		&ds.Version, &ds.QualityScore, &ds.RowCount, &ds.SizeBytes, &tagsJSON,
		&metaJSON, &ds.TenantID, &ds.CreatedBy, &ds.WorkspaceID, &ds.CreatedAt, &ds.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	ds.Classification = models.DataClassification(classStr)
	_ = json.Unmarshal(tagsJSON, &ds.Tags)
	_ = json.Unmarshal(metaJSON, &ds.Metadata)
	return &ds, nil
}

func (p *PGStore) GetDatasetByURN(ctx context.Context, urn string) (*models.Dataset, error) {
	query := `
		SELECT id, urn, name, description, domain, classification,
		       owner_team, owner_email, format, storage_uri, COALESCE(schema_id, ''),
		       version, quality_score, row_count, size_bytes, tags,
		       metadata, tenant_id, created_by, COALESCE(workspace_id, ''), created_at, updated_at
		FROM statdata.datasets WHERE urn = $1
	`
	row := p.db.QueryRowContext(ctx, query, urn)
	var ds models.Dataset
	var tagsJSON, metaJSON []byte
	var classStr string

	err := row.Scan(
		&ds.ID, &ds.URN, &ds.Name, &ds.Description, &ds.Domain, &classStr,
		&ds.OwnerTeam, &ds.OwnerEmail, &ds.Format, &ds.StorageURI, &ds.SchemaID,
		&ds.Version, &ds.QualityScore, &ds.RowCount, &ds.SizeBytes, &tagsJSON,
		&metaJSON, &ds.TenantID, &ds.CreatedBy, &ds.WorkspaceID, &ds.CreatedAt, &ds.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	ds.Classification = models.DataClassification(classStr)
	_ = json.Unmarshal(tagsJSON, &ds.Tags)
	_ = json.Unmarshal(metaJSON, &ds.Metadata)
	return &ds, nil
}

func (p *PGStore) ListDatasets(ctx context.Context, tenantID, domain, classification, workspaceID string, limit, offset int) ([]*models.Dataset, int64, error) {
	var countQuery strings.Builder
	countQuery.WriteString("SELECT COUNT(*) FROM statdata.datasets WHERE 1=1")

	var query strings.Builder
	query.WriteString(`
		SELECT id, urn, name, description, domain, classification,
		       owner_team, owner_email, format, storage_uri, COALESCE(schema_id, ''),
		       version, quality_score, row_count, size_bytes, tags,
		       metadata, tenant_id, created_by, COALESCE(workspace_id, ''), created_at, updated_at
		FROM statdata.datasets WHERE 1=1
	`)

	args := make([]interface{}, 0)
	argIdx := 1

	if tenantID != "" && tenantID != "default" {
		countQuery.WriteString(fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx))
		query.WriteString(fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx))
		args = append(args, tenantID)
		argIdx++
	}
	if workspaceID != "" {
		countQuery.WriteString(fmt.Sprintf(" AND workspace_id = $%d", argIdx))
		query.WriteString(fmt.Sprintf(" AND workspace_id = $%d", argIdx))
		args = append(args, workspaceID)
		argIdx++
	}
	if domain != "" {
		countQuery.WriteString(fmt.Sprintf(" AND domain = $%d", argIdx))
		query.WriteString(fmt.Sprintf(" AND domain = $%d", argIdx))
		args = append(args, domain)
		argIdx++
	}
	if classification != "" {
		countQuery.WriteString(fmt.Sprintf(" AND classification = $%d", argIdx))
		query.WriteString(fmt.Sprintf(" AND classification = $%d", argIdx))
		args = append(args, classification)
		argIdx++
	}

	var total int64
	err := p.db.QueryRowContext(ctx, countQuery.String(), args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query.WriteString(" ORDER BY created_at DESC")
	if limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset))
	}

	rows, err := p.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	res := make([]*models.Dataset, 0)
	for rows.Next() {
		var ds models.Dataset
		var tagsJSON, metaJSON []byte
		var classStr string

		if err := rows.Scan(
			&ds.ID, &ds.URN, &ds.Name, &ds.Description, &ds.Domain, &classStr,
			&ds.OwnerTeam, &ds.OwnerEmail, &ds.Format, &ds.StorageURI, &ds.SchemaID,
			&ds.Version, &ds.QualityScore, &ds.RowCount, &ds.SizeBytes, &tagsJSON,
			&metaJSON, &ds.TenantID, &ds.CreatedBy, &ds.WorkspaceID, &ds.CreatedAt, &ds.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		ds.Classification = models.DataClassification(classStr)
		_ = json.Unmarshal(tagsJSON, &ds.Tags)
		_ = json.Unmarshal(metaJSON, &ds.Metadata)
		res = append(res, &ds)
	}
	return res, total, nil
}

func (p *PGStore) UpdateDataset(ctx context.Context, ds *models.Dataset) error {
	tagsJSON, _ := json.Marshal(ds.Tags)
	metaJSON, _ := json.Marshal(ds.Metadata)

	query := `
		UPDATE statdata.datasets SET
			name = $1, description = $2, domain = $3, classification = $4,
			owner_team = $5, owner_email = $6, format = $7, storage_uri = $8,
			schema_id = $9, version = $10, quality_score = $11, row_count = $12,
			size_bytes = $13, tags = $14, metadata = $15, updated_at = CURRENT_TIMESTAMP,
			workspace_id = $16
		WHERE id = $17
	`
	_, err := p.db.ExecContext(ctx, query,
		ds.Name, ds.Description, ds.Domain, string(ds.Classification),
		ds.OwnerTeam, ds.OwnerEmail, ds.Format, ds.StorageURI,
		ds.SchemaID, ds.Version, ds.QualityScore, ds.RowCount,
		ds.SizeBytes, tagsJSON, metaJSON, ds.WorkspaceID, ds.ID,
	)
	return err
}

func (p *PGStore) DeleteDataset(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM statdata.datasets WHERE id = $1", id)
	return err
}

func (p *PGStore) CreateDataSource(ctx context.Context, src *models.DataSource) error {
	if src.ID == "" {
		src.ID = "src-" + uuid.New().String()[:8]
	}
	credJSON, _ := json.Marshal(src.Credentials)
	query := `
		INSERT INTO statdata.data_sources (
			id, name, source_type, connection_uri, auth_type,
			credentials, status, tenant_id, workspace_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			connection_uri = EXCLUDED.connection_uri,
			status = EXCLUDED.status,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := p.db.ExecContext(ctx, query,
		src.ID, src.Name, src.SourceType, src.ConnectionURI, src.AuthType,
		credJSON, src.Status, src.TenantID, src.WorkspaceID,
	)
	return err
}

func (p *PGStore) GetDataSourceByID(ctx context.Context, id string) (*models.DataSource, error) {
	query := `
		SELECT id, name, source_type, connection_uri, auth_type,
		       credentials, status, tenant_id, last_tested_at, created_at, updated_at
		FROM statdata.data_sources WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var s models.DataSource
	var credJSON []byte
	err := row.Scan(
		&s.ID, &s.Name, &s.SourceType, &s.ConnectionURI, &s.AuthType,
		&credJSON, &s.Status, &s.TenantID, &s.LastTestedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(credJSON, &s.Credentials)
	return &s, nil
}

func (p *PGStore) ListDataSources(ctx context.Context, tenantID, sourceType, workspaceID string) ([]*models.DataSource, error) {
	query := `
		SELECT id, name, source_type, connection_uri, auth_type,
		       credentials, status, tenant_id, last_tested_at, created_at, updated_at
		FROM statdata.data_sources WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if workspaceID != "" {
		query += fmt.Sprintf(" AND (workspace_id = $%d OR workspace_id = '')", argIdx)
		args = append(args, workspaceID)
		argIdx++
	}
	if sourceType != "" {
		query += fmt.Sprintf(" AND source_type = $%d", argIdx)
		args = append(args, sourceType)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.DataSource, 0)
	for rows.Next() {
		var s models.DataSource
		var credJSON []byte
		if err := rows.Scan(
			&s.ID, &s.Name, &s.SourceType, &s.ConnectionURI, &s.AuthType,
			&credJSON, &s.Status, &s.TenantID, &s.LastTestedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(credJSON, &s.Credentials)
		res = append(res, &s)
	}
	return res, nil
}

func (p *PGStore) UpdateDataSource(ctx context.Context, src *models.DataSource) error {
	credJSON, _ := json.Marshal(src.Credentials)
	query := `
		UPDATE statdata.data_sources SET
			name = $1, source_type = $2, connection_uri = $3, auth_type = $4,
			credentials = $5, status = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
	`
	_, err := p.db.ExecContext(ctx, query,
		src.Name, src.SourceType, src.ConnectionURI, src.AuthType,
		credJSON, src.Status, src.ID,
	)
	return err
}

// ─── Schema Registry Implementations ────────────────────────────────────────

func (p *PGStore) RegisterSchema(ctx context.Context, schema *models.SchemaDefinition) error {
	if schema.ID == "" {
		schema.ID = fmt.Sprintf("sch-%s-v%d", schema.Subject, schema.Version)
	}
	fieldsJSON, _ := json.Marshal(schema.Fields)
	query := `
		INSERT INTO statdata.schema_registry (
			id, subject, version, schema_type, schema_content,
			compatibility, description, fields, tenant_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			schema_content = EXCLUDED.schema_content,
			fields = EXCLUDED.fields,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := p.db.ExecContext(ctx, query,
		schema.ID, schema.Subject, schema.Version, schema.SchemaType,
		schema.SchemaContent, schema.Compatibility, schema.Description,
		fieldsJSON, schema.TenantID, schema.CreatedBy,
	)
	return err
}

func (p *PGStore) GetSchemaByID(ctx context.Context, id string) (*models.SchemaDefinition, error) {
	query := `
		SELECT id, subject, version, schema_type, schema_content,
		       compatibility, COALESCE(description, ''), fields, tenant_id, created_by,
		       created_at, updated_at
		FROM statdata.schema_registry WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var s models.SchemaDefinition
	var fieldsJSON []byte
	err := row.Scan(
		&s.ID, &s.Subject, &s.Version, &s.SchemaType, &s.SchemaContent,
		&s.Compatibility, &s.Description, &fieldsJSON, &s.TenantID, &s.CreatedBy,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(fieldsJSON, &s.Fields)
	return &s, nil
}

func (p *PGStore) GetLatestSchemaBySubject(ctx context.Context, subject, tenantID string) (*models.SchemaDefinition, error) {
	query := `
		SELECT id, subject, version, schema_type, schema_content,
		       compatibility, COALESCE(description, ''), fields, tenant_id, created_by,
		       created_at, updated_at
		FROM statdata.schema_registry
		WHERE subject = $1
		ORDER BY version DESC LIMIT 1
	`
	row := p.db.QueryRowContext(ctx, query, subject)
	var s models.SchemaDefinition
	var fieldsJSON []byte
	err := row.Scan(
		&s.ID, &s.Subject, &s.Version, &s.SchemaType, &s.SchemaContent,
		&s.Compatibility, &s.Description, &fieldsJSON, &s.TenantID, &s.CreatedBy,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(fieldsJSON, &s.Fields)
	return &s, nil
}

func (p *PGStore) ListSchemas(ctx context.Context, tenantID string) ([]*models.SchemaDefinition, error) {
	query := `
		SELECT id, subject, version, schema_type, schema_content,
		       compatibility, COALESCE(description, ''), fields, tenant_id, created_by,
		       created_at, updated_at
		FROM statdata.schema_registry
		ORDER BY subject ASC, version DESC
	`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.SchemaDefinition, 0)
	for rows.Next() {
		var s models.SchemaDefinition
		var fieldsJSON []byte
		if err := rows.Scan(
			&s.ID, &s.Subject, &s.Version, &s.SchemaType, &s.SchemaContent,
			&s.Compatibility, &s.Description, &fieldsJSON, &s.TenantID, &s.CreatedBy,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(fieldsJSON, &s.Fields)
		res = append(res, &s)
	}
	return res, nil
}

// ─── Data Contracts Implementations ─────────────────────────────────────────

func (p *PGStore) CreateDataContract(ctx context.Context, contract *models.DataContract) error {
	if contract.ID == "" {
		contract.ID = "contract-" + uuid.New().String()[:8]
	}
	slaJSON, _ := json.Marshal(contract.SLAConfig)
	query := `
		INSERT INTO statdata.data_contracts (
			id, dataset_id, title, producer_team, consumer_team,
			status, max_latency_mins, min_quality_rate, expected_volume,
			sla_config, tenant_id, valid_from, valid_until
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := p.db.ExecContext(ctx, query,
		contract.ID, contract.DatasetID, contract.Title, contract.ProducerTeam,
		contract.ConsumerTeam, contract.Status, contract.MaxLatencyMins,
		contract.MinQualityRate, contract.ExpectedVolume, slaJSON,
		contract.TenantID, contract.ValidFrom, contract.ValidUntil,
	)
	return err
}

func (p *PGStore) GetDataContractByID(ctx context.Context, id string) (*models.DataContract, error) {
	query := `
		SELECT id, dataset_id, title, producer_team, consumer_team,
		       status, max_latency_mins, min_quality_rate, expected_volume,
		       sla_config, tenant_id, valid_from, valid_until, created_at, updated_at
		FROM statdata.data_contracts WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var c models.DataContract
	var slaJSON []byte
	err := row.Scan(
		&c.ID, &c.DatasetID, &c.Title, &c.ProducerTeam, &c.ConsumerTeam,
		&c.Status, &c.MaxLatencyMins, &c.MinQualityRate, &c.ExpectedVolume,
		&slaJSON, &c.TenantID, &c.ValidFrom, &c.ValidUntil, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(slaJSON, &c.SLAConfig)
	return &c, nil
}

func (p *PGStore) ListDataContracts(ctx context.Context, tenantID, datasetID string) ([]*models.DataContract, error) {
	query := `
		SELECT id, dataset_id, title, producer_team, consumer_team,
		       status, max_latency_mins, min_quality_rate, expected_volume,
		       sla_config, tenant_id, valid_from, valid_until, created_at, updated_at
		FROM statdata.data_contracts WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if datasetID != "" {
		query += fmt.Sprintf(" AND dataset_id = $%d", argIdx)
		args = append(args, datasetID)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.DataContract, 0)
	for rows.Next() {
		var c models.DataContract
		var slaJSON []byte
		if err := rows.Scan(
			&c.ID, &c.DatasetID, &c.Title, &c.ProducerTeam, &c.ConsumerTeam,
			&c.Status, &c.MaxLatencyMins, &c.MinQualityRate, &c.ExpectedVolume,
			&slaJSON, &c.TenantID, &c.ValidFrom, &c.ValidUntil, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(slaJSON, &c.SLAConfig)
		res = append(res, &c)
	}
	return res, nil
}

func (p *PGStore) UpdateDataContract(ctx context.Context, contract *models.DataContract) error {
	slaJSON, _ := json.Marshal(contract.SLAConfig)
	query := `
		UPDATE statdata.data_contracts SET
			title = $1, producer_team = $2, consumer_team = $3,
			status = $4, max_latency_mins = $5, min_quality_rate = $6,
			expected_volume = $7, sla_config = $8, updated_at = CURRENT_TIMESTAMP
		WHERE id = $9
	`
	_, err := p.db.ExecContext(ctx, query,
		contract.Title, contract.ProducerTeam, contract.ConsumerTeam,
		contract.Status, contract.MaxLatencyMins, contract.MinQualityRate,
		contract.ExpectedVolume, slaJSON, contract.ID,
	)
	return err
}

// ─── Data Quality & Governance Implementations ──────────────────────────────

func (p *PGStore) CreateQualityRule(ctx context.Context, rule *models.DataQualityRule) error {
	if rule.ID == "" {
		rule.ID = "qr-" + uuid.New().String()[:8]
	}
	paramsJSON, _ := json.Marshal(rule.Parameters)
	query := `
		INSERT INTO statdata.data_quality_rules (
			id, dataset_id, rule_name, rule_type, target_field,
			parameters, severity, is_enabled, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := p.db.ExecContext(ctx, query,
		rule.ID, rule.DatasetID, rule.RuleName, rule.RuleType,
		rule.TargetField, paramsJSON, rule.Severity, rule.IsEnabled, rule.TenantID,
	)
	return err
}

func (p *PGStore) ListQualityRules(ctx context.Context, tenantID, datasetID string) ([]*models.DataQualityRule, error) {
	query := `
		SELECT id, dataset_id, rule_name, rule_type, COALESCE(target_field, ''),
		       parameters, severity, is_enabled, tenant_id, created_at, updated_at
		FROM statdata.data_quality_rules WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if datasetID != "" {
		query += fmt.Sprintf(" AND dataset_id = $%d", argIdx)
		args = append(args, datasetID)
		argIdx++
	}
	query += " ORDER BY created_at ASC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.DataQualityRule, 0)
	for rows.Next() {
		var r models.DataQualityRule
		var paramsJSON []byte
		if err := rows.Scan(
			&r.ID, &r.DatasetID, &r.RuleName, &r.RuleType, &r.TargetField,
			&paramsJSON, &r.Severity, &r.IsEnabled, &r.TenantID, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(paramsJSON, &r.Parameters)
		res = append(res, &r)
	}
	return res, nil
}

func (p *PGStore) SaveQualityReport(ctx context.Context, report *models.DataQualityReport) error {
	if report.ID == "" {
		report.ID = "qrep-" + uuid.New().String()[:8]
	}
	resultsJSON, _ := json.Marshal(report.RuleResults)
	query := `
		INSERT INTO statdata.data_quality_reports (
			id, dataset_id, pipeline_run_id, status, quality_score,
			total_rules, passed_rules, failed_rules, rule_results,
			evaluated_at, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := p.db.ExecContext(ctx, query,
		report.ID, report.DatasetID, report.PipelineRunID, report.Status,
		report.QualityScore, report.TotalRules, report.PassedRules,
		report.FailedRules, resultsJSON, report.EvaluatedAt, report.TenantID,
	)
	return err
}

func (p *PGStore) GetLatestQualityReport(ctx context.Context, datasetID string) (*models.DataQualityReport, error) {
	query := `
		SELECT id, dataset_id, COALESCE(pipeline_run_id, ''), status, quality_score,
		       total_rules, passed_rules, failed_rules, rule_results,
		       evaluated_at, tenant_id
		FROM statdata.data_quality_reports
		WHERE dataset_id = $1
		ORDER BY evaluated_at DESC LIMIT 1
	`
	row := p.db.QueryRowContext(ctx, query, datasetID)
	var r models.DataQualityReport
	var resultsJSON []byte
	err := row.Scan(
		&r.ID, &r.DatasetID, &r.PipelineRunID, &r.Status, &r.QualityScore,
		&r.TotalRules, &r.PassedRules, &r.FailedRules, &resultsJSON,
		&r.EvaluatedAt, &r.TenantID,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(resultsJSON, &r.RuleResults)
	return &r, nil
}

func (p *PGStore) ListQualityReports(ctx context.Context, tenantID, datasetID string, limit int) ([]*models.DataQualityReport, error) {
	query := `
		SELECT id, dataset_id, COALESCE(pipeline_run_id, ''), status, quality_score,
		       total_rules, passed_rules, failed_rules, rule_results,
		       evaluated_at, tenant_id
		FROM statdata.data_quality_reports WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if datasetID != "" {
		query += fmt.Sprintf(" AND dataset_id = $%d", argIdx)
		args = append(args, datasetID)
		argIdx++
	}
	query += " ORDER BY evaluated_at DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.DataQualityReport, 0)
	for rows.Next() {
		var r models.DataQualityReport
		var resultsJSON []byte
		if err := rows.Scan(
			&r.ID, &r.DatasetID, &r.PipelineRunID, &r.Status, &r.QualityScore,
			&r.TotalRules, &r.PassedRules, &r.FailedRules, &resultsJSON,
			&r.EvaluatedAt, &r.TenantID,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(resultsJSON, &r.RuleResults)
		res = append(res, &r)
	}
	return res, nil
}

// ─── Data Lineage Implementations ───────────────────────────────────────────

func (p *PGStore) RecordLineageNode(ctx context.Context, node *models.LineageNode) error {
	metaJSON, _ := json.Marshal(node.Metadata)
	query := `
		INSERT INTO statdata.lineage_nodes (id, urn, type, name, domain, tenant_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			metadata = EXCLUDED.metadata
	`
	_, err := p.db.ExecContext(ctx, query, node.ID, node.URN, node.Type, node.Name, node.Domain, node.TenantID, metaJSON)
	return err
}

func (p *PGStore) RecordLineageEdge(ctx context.Context, edge *models.LineageEdge) error {
	if edge.ID == "" {
		edge.ID = fmt.Sprintf("edge-%s-%s", edge.SourceNodeID, edge.TargetNodeID)
	}
	query := `
		INSERT INTO statdata.lineage_edges (
			id, source_node_id, target_node_id, relation_type, pipeline_id, transformation, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := p.db.ExecContext(ctx, query,
		edge.ID, edge.SourceNodeID, edge.TargetNodeID, edge.RelationType,
		edge.PipelineID, edge.Transformation, edge.TenantID,
	)
	return err
}

func (p *PGStore) GetLineageGraph(ctx context.Context, rootID, tenantID string, depth int) (*models.LineageGraph, error) {
	// Recursive CTE to traverse graph up to requested depth
	query := `
		WITH RECURSIVE lineage_traverse AS (
			SELECT source_node_id, target_node_id, id, relation_type, pipeline_id, transformation, 1 as depth
			FROM statdata.lineage_edges
			WHERE source_node_id = $1 OR target_node_id = $1

			UNION

			SELECT e.source_node_id, e.target_node_id, e.id, e.relation_type, e.pipeline_id, e.transformation, lt.depth + 1
			FROM statdata.lineage_edges e
			INNER JOIN lineage_traverse lt ON (e.source_node_id = lt.target_node_id OR e.target_node_id = lt.source_node_id)
			WHERE lt.depth < $2
		)
		SELECT DISTINCT id, source_node_id, target_node_id, relation_type, COALESCE(pipeline_id, ''), COALESCE(transformation, '')
		FROM lineage_traverse
	`
	rows, err := p.db.QueryContext(ctx, query, rootID, depth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	edges := make([]models.LineageEdge, 0)
	nodeIDSet := make(map[string]bool)
	nodeIDSet[rootID] = true

	for rows.Next() {
		var edge models.LineageEdge
		if err := rows.Scan(&edge.ID, &edge.SourceNodeID, &edge.TargetNodeID, &edge.RelationType, &edge.PipelineID, &edge.Transformation); err != nil {
			return nil, err
		}
		edges = append(edges, edge)
		nodeIDSet[edge.SourceNodeID] = true
		nodeIDSet[edge.TargetNodeID] = true
	}

	nodes := make([]models.LineageNode, 0)
	for nID := range nodeIDSet {
		var n models.LineageNode
		var metaJSON []byte
		err := p.db.QueryRowContext(ctx, "SELECT id, urn, type, name, domain, tenant_id, metadata FROM statdata.lineage_nodes WHERE id = $1", nID).
			Scan(&n.ID, &n.URN, &n.Type, &n.Name, &n.Domain, &n.TenantID, &metaJSON)
		if err == nil {
			_ = json.Unmarshal(metaJSON, &n.Metadata)
			nodes = append(nodes, n)
		}
	}

	return &models.LineageGraph{
		RootID: rootID,
		Nodes:  nodes,
		Edges:  edges,
	}, nil
}

// ─── Data Pipelines & Streaming Implementations ─────────────────────────────

func (p *PGStore) CreatePipeline(ctx context.Context, pipe *models.DataPipeline) error {
	if pipe.ID == "" {
		pipe.ID = "pipe-" + uuid.New().String()[:8]
	}
	stagesJSON, _ := json.Marshal(pipe.Stages)
	configJSON, _ := json.Marshal(pipe.Config)

	query := `
		INSERT INTO statdata.pipelines (
			id, name, description, pipeline_type, status,
			cron_schedule, source_dataset_id, target_dataset_id,
			stages, config, max_retries, timeout_seconds,
			tenant_id, created_by, workspace_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := p.db.ExecContext(ctx, query,
		pipe.ID, pipe.Name, pipe.Description, string(pipe.PipelineType), string(pipe.Status),
		pipe.CronSchedule, pipe.SourceDatasetID, pipe.TargetDatasetID,
		stagesJSON, configJSON, pipe.MaxRetries, pipe.TimeoutSeconds,
		pipe.TenantID, pipe.CreatedBy, pipe.WorkspaceID,
	)
	return err
}

func (p *PGStore) GetPipelineByID(ctx context.Context, id string) (*models.DataPipeline, error) {
	query := `
		SELECT id, name, description, pipeline_type, status,
		       COALESCE(cron_schedule, ''), COALESCE(source_dataset_id, ''), COALESCE(target_dataset_id, ''),
		       stages, config, max_retries, timeout_seconds,
		       tenant_id, created_by, COALESCE(workspace_id, ''), last_run_at, created_at, updated_at
		FROM statdata.pipelines WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var pipe models.DataPipeline
	var stagesJSON, configJSON []byte
	var pType, pStat string

	err := row.Scan(
		&pipe.ID, &pipe.Name, &pipe.Description, &pType, &pStat,
		&pipe.CronSchedule, &pipe.SourceDatasetID, &pipe.TargetDatasetID,
		&stagesJSON, &configJSON, &pipe.MaxRetries, &pipe.TimeoutSeconds,
		&pipe.TenantID, &pipe.CreatedBy, &pipe.WorkspaceID, &pipe.LastRunAt, &pipe.CreatedAt, &pipe.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	pipe.PipelineType = models.PipelineType(pType)
	pipe.Status = models.PipelineStatus(pStat)
	_ = json.Unmarshal(stagesJSON, &pipe.Stages)
	_ = json.Unmarshal(configJSON, &pipe.Config)
	return &pipe, nil
}

func (p *PGStore) ListPipelines(ctx context.Context, tenantID, status, workspaceID string) ([]*models.DataPipeline, error) {
	query := `
		SELECT id, name, description, pipeline_type, status,
		       COALESCE(cron_schedule, ''), COALESCE(source_dataset_id, ''), COALESCE(target_dataset_id, ''),
		       stages, config, max_retries, timeout_seconds,
		       tenant_id, created_by, COALESCE(workspace_id, ''), last_run_at, created_at, updated_at
		FROM statdata.pipelines WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if workspaceID != "" {
		query += fmt.Sprintf(" AND workspace_id = $%d", argIdx)
		args = append(args, workspaceID)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.DataPipeline, 0)
	for rows.Next() {
		var pipe models.DataPipeline
		var stagesJSON, configJSON []byte
		var pType, pStat string

		if err := rows.Scan(
			&pipe.ID, &pipe.Name, &pipe.Description, &pType, &pStat,
			&pipe.CronSchedule, &pipe.SourceDatasetID, &pipe.TargetDatasetID,
			&stagesJSON, &configJSON, &pipe.MaxRetries, &pipe.TimeoutSeconds,
			&pipe.TenantID, &pipe.CreatedBy, &pipe.WorkspaceID, &pipe.LastRunAt, &pipe.CreatedAt, &pipe.UpdatedAt,
		); err != nil {
			return nil, err
		}
		pipe.PipelineType = models.PipelineType(pType)
		pipe.Status = models.PipelineStatus(pStat)
		_ = json.Unmarshal(stagesJSON, &pipe.Stages)
		_ = json.Unmarshal(configJSON, &pipe.Config)
		res = append(res, &pipe)
	}
	return res, nil
}

func (p *PGStore) UpdatePipeline(ctx context.Context, pipe *models.DataPipeline) error {
	stagesJSON, _ := json.Marshal(pipe.Stages)
	configJSON, _ := json.Marshal(pipe.Config)
	query := `
		UPDATE statdata.pipelines SET
			name = $1, description = $2, pipeline_type = $3, status = $4,
			cron_schedule = $5, source_dataset_id = $6, target_dataset_id = $7,
			stages = $8, config = $9, max_retries = $10, timeout_seconds = $11,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $12
	`
	_, err := p.db.ExecContext(ctx, query,
		pipe.Name, pipe.Description, string(pipe.PipelineType), string(pipe.Status),
		pipe.CronSchedule, pipe.SourceDatasetID, pipe.TargetDatasetID,
		stagesJSON, configJSON, pipe.MaxRetries, pipe.TimeoutSeconds, pipe.ID,
	)
	return err
}

func (p *PGStore) DeletePipeline(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM statdata.pipelines WHERE id = $1", id)
	return err
}

func (p *PGStore) RecordPipelineRun(ctx context.Context, run *models.PipelineRun) error {
	if run.ID == "" {
		run.ID = "run-" + uuid.New().String()[:8]
	}
	stageRunsJSON, _ := json.Marshal(run.StageRuns)
	metricsJSON, _ := json.Marshal(run.Metrics)

	query := `
		INSERT INTO statdata.pipeline_runs (
			id, pipeline_id, pipeline_name, status, trigger_type,
			started_at, finished_at, duration_ms, records_read,
			records_written, records_rejected, stage_runs,
			error_message, metrics, tenant_id, triggered_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	_, err := p.db.ExecContext(ctx, query,
		run.ID, run.PipelineID, run.PipelineName, string(run.Status), run.TriggerType,
		run.StartedAt, run.FinishedAt, run.DurationMs, run.RecordsRead,
		run.RecordsWritten, run.RecordsRejected, stageRunsJSON,
		run.ErrorMessage, metricsJSON, run.TenantID, run.TriggeredBy,
	)
	return err
}

func (p *PGStore) GetPipelineRunByID(ctx context.Context, id string) (*models.PipelineRun, error) {
	query := `
		SELECT id, pipeline_id, pipeline_name, status, trigger_type,
		       started_at, finished_at, duration_ms, records_read,
		       records_written, records_rejected, stage_runs,
		       COALESCE(error_message, ''), metrics, tenant_id, triggered_by
		FROM statdata.pipeline_runs WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var run models.PipelineRun
	var stageRunsJSON, metricsJSON []byte
	var statusStr string

	err := row.Scan(
		&run.ID, &run.PipelineID, &run.PipelineName, &statusStr, &run.TriggerType,
		&run.StartedAt, &run.FinishedAt, &run.DurationMs, &run.RecordsRead,
		&run.RecordsWritten, &run.RecordsRejected, &stageRunsJSON,
		&run.ErrorMessage, &metricsJSON, &run.TenantID, &run.TriggeredBy,
	)
	if err != nil {
		return nil, err
	}
	run.Status = models.RunStatus(statusStr)
	_ = json.Unmarshal(stageRunsJSON, &run.StageRuns)
	_ = json.Unmarshal(metricsJSON, &run.Metrics)
	return &run, nil
}

func (p *PGStore) ListPipelineRuns(ctx context.Context, pipelineID, tenantID string, limit int) ([]*models.PipelineRun, error) {
	query := `
		SELECT id, pipeline_id, pipeline_name, status, trigger_type,
		       started_at, finished_at, duration_ms, records_read,
		       records_written, records_rejected, stage_runs,
		       COALESCE(error_message, ''), metrics, tenant_id, triggered_by
		FROM statdata.pipeline_runs WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if pipelineID != "" {
		query += fmt.Sprintf(" AND pipeline_id = $%d", argIdx)
		args = append(args, pipelineID)
		argIdx++
	}
	query += " ORDER BY started_at DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.PipelineRun, 0)
	for rows.Next() {
		var run models.PipelineRun
		var stageRunsJSON, metricsJSON []byte
		var statusStr string

		if err := rows.Scan(
			&run.ID, &run.PipelineID, &run.PipelineName, &statusStr, &run.TriggerType,
			&run.StartedAt, &run.FinishedAt, &run.DurationMs, &run.RecordsRead,
			&run.RecordsWritten, &run.RecordsRejected, &stageRunsJSON,
			&run.ErrorMessage, &metricsJSON, &run.TenantID, &run.TriggeredBy,
		); err != nil {
			return nil, err
		}
		run.Status = models.RunStatus(statusStr)
		_ = json.Unmarshal(stageRunsJSON, &run.StageRuns)
		_ = json.Unmarshal(metricsJSON, &run.Metrics)
		res = append(res, &run)
	}
	return res, nil
}

func (p *PGStore) UpdatePipelineRun(ctx context.Context, run *models.PipelineRun) error {
	stageRunsJSON, _ := json.Marshal(run.StageRuns)
	metricsJSON, _ := json.Marshal(run.Metrics)
	query := `
		UPDATE statdata.pipeline_runs SET
			status = $1, finished_at = $2, duration_ms = $3,
			records_read = $4, records_written = $5, records_rejected = $6,
			stage_runs = $7, error_message = $8, metrics = $9
		WHERE id = $10
	`
	_, err := p.db.ExecContext(ctx, query,
		string(run.Status), run.FinishedAt, run.DurationMs,
		run.RecordsRead, run.RecordsWritten, run.RecordsRejected,
		stageRunsJSON, run.ErrorMessage, metricsJSON, run.ID,
	)
	return err
}

func (p *PGStore) CreateStreamingJob(ctx context.Context, job *models.StreamingJob) error {
	if job.ID == "" {
		job.ID = "stream-" + uuid.New().String()[:8]
	}
	configJSON, _ := json.Marshal(job.Config)
	query := `
		INSERT INTO statdata.streaming_jobs (
			id, name, source_topic, target_sink, status,
			throughput_msg_sec, lag_records, config, tenant_id, workspace_id, last_checkpoint
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		job.ID, job.Name, job.SourceTopic, job.TargetSink, job.Status,
		job.ThroughputMsgSec, job.LagRecords, configJSON, job.TenantID, job.WorkspaceID,
	)
	return err
}

func (p *PGStore) GetStreamingJobByID(ctx context.Context, id string) (*models.StreamingJob, error) {
	query := `
		SELECT id, name, source_topic, target_sink, status,
		       throughput_msg_sec, lag_records, config, tenant_id, last_checkpoint,
		       created_at, updated_at
		FROM statdata.streaming_jobs WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var j models.StreamingJob
	var configJSON []byte
	err := row.Scan(
		&j.ID, &j.Name, &j.SourceTopic, &j.TargetSink, &j.Status,
		&j.ThroughputMsgSec, &j.LagRecords, &configJSON, &j.TenantID, &j.LastCheckpoint,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(configJSON, &j.Config)
	return &j, nil
}

func (p *PGStore) ListStreamingJobs(ctx context.Context, tenantID, status, workspaceID string) ([]*models.StreamingJob, error) {
	query := `
		SELECT id, name, source_topic, target_sink, status,
		       throughput_msg_sec, lag_records, config, tenant_id, last_checkpoint,
		       created_at, updated_at
		FROM statdata.streaming_jobs WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if workspaceID != "" {
		query += fmt.Sprintf(" AND (workspace_id = $%d OR workspace_id = '')", argIdx)
		args = append(args, workspaceID)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.StreamingJob, 0)
	for rows.Next() {
		var j models.StreamingJob
		var configJSON []byte
		if err := rows.Scan(
			&j.ID, &j.Name, &j.SourceTopic, &j.TargetSink, &j.Status,
			&j.ThroughputMsgSec, &j.LagRecords, &configJSON, &j.TenantID, &j.LastCheckpoint,
			&j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(configJSON, &j.Config)
		res = append(res, &j)
	}
	return res, nil
}

func (p *PGStore) UpdateStreamingJob(ctx context.Context, job *models.StreamingJob) error {
	configJSON, _ := json.Marshal(job.Config)
	query := `
		UPDATE statdata.streaming_jobs SET
			name = $1, status = $2, throughput_msg_sec = $3,
			lag_records = $4, config = $5, last_checkpoint = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
	`
	_, err := p.db.ExecContext(ctx, query,
		job.Name, job.Status, job.ThroughputMsgSec,
		job.LagRecords, configJSON, job.ID,
	)
	return err
}

// ─── Feature Store Implementations ──────────────────────────────────────────

func (p *PGStore) CreateFeatureView(ctx context.Context, fv *models.FeatureView) error {
	if fv.ID == "" {
		fv.ID = "fv-" + uuid.New().String()[:8]
	}
	featuresJSON, _ := json.Marshal(fv.Features)
	tagsJSON, _ := json.Marshal(fv.Tags)

	query := `
		INSERT INTO statdata.feature_views (
			id, name, entity_name, description, ttl_seconds,
			features, source_query, online_store, offline_sink,
			tags, tenant_id, created_by, workspace_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := p.db.ExecContext(ctx, query,
		fv.ID, fv.Name, fv.EntityName, fv.Description, fv.TTLSeconds,
		featuresJSON, fv.SourceQuery, fv.OnlineStore, fv.OfflineSink,
		tagsJSON, fv.TenantID, fv.CreatedBy, fv.WorkspaceID,
	)
	return err
}

func (p *PGStore) GetFeatureViewByID(ctx context.Context, id string) (*models.FeatureView, error) {
	query := `
		SELECT id, name, entity_name, description, ttl_seconds,
		       features, COALESCE(source_query, ''), online_store, COALESCE(offline_sink, ''),
		       tags, tenant_id, created_by, COALESCE(workspace_id, ''), created_at, updated_at
		FROM statdata.feature_views WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var fv models.FeatureView
	var featJSON, tagsJSON []byte
	err := row.Scan(
		&fv.ID, &fv.Name, &fv.EntityName, &fv.Description, &fv.TTLSeconds,
		&featJSON, &fv.SourceQuery, &fv.OnlineStore, &fv.OfflineSink,
		&tagsJSON, &fv.TenantID, &fv.CreatedBy, &fv.WorkspaceID, &fv.CreatedAt, &fv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(featJSON, &fv.Features)
	_ = json.Unmarshal(tagsJSON, &fv.Tags)
	return &fv, nil
}

func (p *PGStore) ListFeatureViews(ctx context.Context, tenantID, entityName, workspaceID string) ([]*models.FeatureView, error) {
	query := `
		SELECT id, name, entity_name, description, ttl_seconds,
		       features, COALESCE(source_query, ''), online_store, COALESCE(offline_sink, ''),
		       tags, tenant_id, created_by, COALESCE(workspace_id, ''), created_at, updated_at
		FROM statdata.feature_views WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if workspaceID != "" {
		query += fmt.Sprintf(" AND (workspace_id = $%d OR workspace_id = '')", argIdx)
		args = append(args, workspaceID)
		argIdx++
	}
	if entityName != "" {
		query += fmt.Sprintf(" AND entity_name = $%d", argIdx)
		args = append(args, entityName)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.FeatureView, 0)
	for rows.Next() {
		var fv models.FeatureView
		var featJSON, tagsJSON []byte
		if err := rows.Scan(
			&fv.ID, &fv.Name, &fv.EntityName, &fv.Description, &fv.TTLSeconds,
			&featJSON, &fv.SourceQuery, &fv.OnlineStore, &fv.OfflineSink,
			&tagsJSON, &fv.TenantID, &fv.CreatedBy, &fv.WorkspaceID, &fv.CreatedAt, &fv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(featJSON, &fv.Features)
		_ = json.Unmarshal(tagsJSON, &fv.Tags)
		res = append(res, &fv)
	}
	return res, nil
}

func (p *PGStore) SaveFeatureRecords(ctx context.Context, records []models.FeatureRecord) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO statdata.feature_records (
			entity_key, feature_view_id, "values", timestamp, tenant_id
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (entity_key, feature_view_id) DO UPDATE SET
			"values" = EXCLUDED."values",
			timestamp = EXCLUDED.timestamp
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, rec := range records {
		valJSON, _ := json.Marshal(rec.Values)
		ts := rec.Timestamp
		if ts.IsZero() {
			ts = time.Now().UTC()
		}
		if _, err := stmt.ExecContext(ctx, rec.EntityKey, rec.FeatureViewID, valJSON, ts, rec.TenantID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *PGStore) GetOnlineFeatures(ctx context.Context, featureViewID, entityKey, tenantID string) (*models.FeatureVector, error) {
	query := `
		SELECT entity_key, "values", timestamp
		FROM statdata.feature_records
		WHERE feature_view_id = $1 AND entity_key = $2
	`
	row := p.db.QueryRowContext(ctx, query, featureViewID, entityKey)
	var vec models.FeatureVector
	var valJSON []byte
	err := row.Scan(&vec.EntityKey, &valJSON, &vec.RetrievedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(valJSON, &vec.Features)
	return &vec, nil
}

// ─── Scientific Computing Implementations ───────────────────────────────────

func (p *PGStore) CreateNotebookSession(ctx context.Context, nb *models.NotebookSession) error {
	if nb.ID == "" {
		nb.ID = "nb-" + uuid.New().String()[:8]
	}
	cellsJSON, _ := json.Marshal(nb.Cells)
	refsJSON, _ := json.Marshal(nb.DatasetRefs)
	varsJSON, _ := json.Marshal(nb.Variables)

	query := `
		INSERT INTO statdata.notebook_sessions (
			id, title, language, kernel_state, dataset_refs,
			cells, variables, tenant_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := p.db.ExecContext(ctx, query,
		nb.ID, nb.Title, nb.Language, nb.KernelState, refsJSON,
		cellsJSON, varsJSON, nb.TenantID, nb.CreatedBy,
	)
	return err
}

func (p *PGStore) GetNotebookSessionByID(ctx context.Context, id string) (*models.NotebookSession, error) {
	query := `
		SELECT id, title, language, kernel_state, dataset_refs,
		       cells, variables, tenant_id, created_by, created_at, updated_at
		FROM statdata.notebook_sessions WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var nb models.NotebookSession
	var cellsJSON, refsJSON, varsJSON []byte

	err := row.Scan(
		&nb.ID, &nb.Title, &nb.Language, &nb.KernelState, &refsJSON,
		&cellsJSON, &varsJSON, &nb.TenantID, &nb.CreatedBy, &nb.CreatedAt, &nb.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(cellsJSON, &nb.Cells)
	_ = json.Unmarshal(refsJSON, &nb.DatasetRefs)
	_ = json.Unmarshal(varsJSON, &nb.Variables)
	return &nb, nil
}

func (p *PGStore) ListNotebookSessions(ctx context.Context, tenantID, language string) ([]*models.NotebookSession, error) {
	query := `
		SELECT id, title, language, kernel_state, dataset_refs,
		       cells, variables, tenant_id, created_by, created_at, updated_at
		FROM statdata.notebook_sessions WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if language != "" {
		query += fmt.Sprintf(" AND language = $%d", argIdx)
		args = append(args, language)
		argIdx++
	}
	query += " ORDER BY updated_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.NotebookSession, 0)
	for rows.Next() {
		var nb models.NotebookSession
		var cellsJSON, refsJSON, varsJSON []byte
		if err := rows.Scan(
			&nb.ID, &nb.Title, &nb.Language, &nb.KernelState, &refsJSON,
			&cellsJSON, &varsJSON, &nb.TenantID, &nb.CreatedBy, &nb.CreatedAt, &nb.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(cellsJSON, &nb.Cells)
		_ = json.Unmarshal(refsJSON, &nb.DatasetRefs)
		_ = json.Unmarshal(varsJSON, &nb.Variables)
		res = append(res, &nb)
	}
	return res, nil
}

func (p *PGStore) UpdateNotebookSession(ctx context.Context, nb *models.NotebookSession) error {
	cellsJSON, _ := json.Marshal(nb.Cells)
	varsJSON, _ := json.Marshal(nb.Variables)
	query := `
		UPDATE statdata.notebook_sessions SET
			title = $1, kernel_state = $2, cells = $3, variables = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`
	_, err := p.db.ExecContext(ctx, query, nb.Title, nb.KernelState, cellsJSON, varsJSON, nb.ID)
	return err
}

func (p *PGStore) CreateExperiment(ctx context.Context, exp *models.Experiment) error {
	if exp.ID == "" {
		exp.ID = "exp-" + uuid.New().String()[:8]
	}
	tagsJSON, _ := json.Marshal(exp.Tags)
	query := `
		INSERT INTO statdata.experiments (
			id, name, description, domain, tags, artifact_uri, tenant_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := p.db.ExecContext(ctx, query,
		exp.ID, exp.Name, exp.Description, exp.Domain,
		tagsJSON, exp.ArtifactURI, exp.TenantID, exp.CreatedBy,
	)
	return err
}

func (p *PGStore) GetExperimentByID(ctx context.Context, id string) (*models.Experiment, error) {
	query := `
		SELECT id, name, description, domain, tags, COALESCE(artifact_uri, ''),
		       tenant_id, created_by, created_at, updated_at
		FROM statdata.experiments WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var exp models.Experiment
	var tagsJSON []byte
	err := row.Scan(
		&exp.ID, &exp.Name, &exp.Description, &exp.Domain, &tagsJSON,
		&exp.ArtifactURI, &exp.TenantID, &exp.CreatedBy, &exp.CreatedAt, &exp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(tagsJSON, &exp.Tags)
	return &exp, nil
}

func (p *PGStore) ListExperiments(ctx context.Context, tenantID, domain string) ([]*models.Experiment, error) {
	query := `
		SELECT id, name, description, domain, tags, COALESCE(artifact_uri, ''),
		       tenant_id, created_by, created_at, updated_at
		FROM statdata.experiments WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if domain != "" {
		query += fmt.Sprintf(" AND domain = $%d", argIdx)
		args = append(args, domain)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.Experiment, 0)
	for rows.Next() {
		var exp models.Experiment
		var tagsJSON []byte
		if err := rows.Scan(
			&exp.ID, &exp.Name, &exp.Description, &exp.Domain, &tagsJSON,
			&exp.ArtifactURI, &exp.TenantID, &exp.CreatedBy, &exp.CreatedAt, &exp.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tagsJSON, &exp.Tags)
		res = append(res, &exp)
	}
	return res, nil
}

func (p *PGStore) RecordExperimentRun(ctx context.Context, run *models.ExperimentRun) error {
	if run.ID == "" {
		run.ID = "exprun-" + uuid.New().String()[:8]
	}
	paramJSON, _ := json.Marshal(run.Parameters)
	metricsJSON, _ := json.Marshal(run.Metrics)
	tagsJSON, _ := json.Marshal(run.Tags)
	artifactsJSON, _ := json.Marshal(run.Artifacts)

	query := `
		INSERT INTO statdata.experiment_runs (
			id, experiment_id, run_name, status, parameters,
			metrics, tags, artifacts, start_time, end_time,
			duration_ms, tenant_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := p.db.ExecContext(ctx, query,
		run.ID, run.ExperimentID, run.RunName, run.Status, paramJSON,
		metricsJSON, tagsJSON, artifactsJSON, run.StartTime, run.EndTime,
		run.DurationMs, run.TenantID, run.CreatedBy,
	)
	return err
}

func (p *PGStore) GetExperimentRunByID(ctx context.Context, id string) (*models.ExperimentRun, error) {
	query := `
		SELECT id, experiment_id, run_name, status, parameters,
		       metrics, tags, artifacts, start_time, end_time,
		       duration_ms, tenant_id, created_by
		FROM statdata.experiment_runs WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var run models.ExperimentRun
	var paramJSON, metricsJSON, tagsJSON, artifactsJSON []byte
	err := row.Scan(
		&run.ID, &run.ExperimentID, &run.RunName, &run.Status, &paramJSON,
		&metricsJSON, &tagsJSON, &artifactsJSON, &run.StartTime, &run.EndTime,
		&run.DurationMs, &run.TenantID, &run.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(paramJSON, &run.Parameters)
	_ = json.Unmarshal(metricsJSON, &run.Metrics)
	_ = json.Unmarshal(tagsJSON, &run.Tags)
	_ = json.Unmarshal(artifactsJSON, &run.Artifacts)
	return &run, nil
}

func (p *PGStore) ListExperimentRuns(ctx context.Context, experimentID, tenantID string) ([]*models.ExperimentRun, error) {
	query := `
		SELECT id, experiment_id, run_name, status, parameters,
		       metrics, tags, artifacts, start_time, end_time,
		       duration_ms, tenant_id, created_by
		FROM statdata.experiment_runs WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if experimentID != "" {
		query += fmt.Sprintf(" AND experiment_id = $%d", argIdx)
		args = append(args, experimentID)
		argIdx++
	}
	query += " ORDER BY start_time DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.ExperimentRun, 0)
	for rows.Next() {
		var run models.ExperimentRun
		var paramJSON, metricsJSON, tagsJSON, artifactsJSON []byte
		if err := rows.Scan(
			&run.ID, &run.ExperimentID, &run.RunName, &run.Status, &paramJSON,
			&metricsJSON, &tagsJSON, &artifactsJSON, &run.StartTime, &run.EndTime,
			&run.DurationMs, &run.TenantID, &run.CreatedBy,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(paramJSON, &run.Parameters)
		_ = json.Unmarshal(metricsJSON, &run.Metrics)
		_ = json.Unmarshal(tagsJSON, &run.Tags)
		_ = json.Unmarshal(artifactsJSON, &run.Artifacts)
		res = append(res, &run)
	}
	return res, nil
}

func (p *PGStore) CreateRegisteredModel(ctx context.Context, rm *models.RegisteredModel) error {
	if rm.ID == "" {
		rm.ID = "model-" + uuid.New().String()[:8]
	}
	query := `
		INSERT INTO statdata.registered_models (
			id, name, description, domain, framework, latest_stage, tenant_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := p.db.ExecContext(ctx, query,
		rm.ID, rm.Name, rm.Description, rm.Domain,
		rm.Framework, string(rm.LatestStage), rm.TenantID, rm.CreatedBy,
	)
	return err
}

func (p *PGStore) GetRegisteredModelByID(ctx context.Context, id string) (*models.RegisteredModel, error) {
	query := `
		SELECT id, name, description, domain, framework, latest_stage,
		       tenant_id, created_by, created_at, updated_at
		FROM statdata.registered_models WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var rm models.RegisteredModel
	var stageStr string
	err := row.Scan(
		&rm.ID, &rm.Name, &rm.Description, &rm.Domain, &rm.Framework,
		&stageStr, &rm.TenantID, &rm.CreatedBy, &rm.CreatedAt, &rm.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	rm.LatestStage = models.ModelStage(stageStr)
	return &rm, nil
}

func (p *PGStore) ListRegisteredModels(ctx context.Context, tenantID, domain string) ([]*models.RegisteredModel, error) {
	query := `
		SELECT id, name, description, domain, framework, latest_stage,
		       tenant_id, created_by, created_at, updated_at
		FROM statdata.registered_models WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if domain != "" {
		query += fmt.Sprintf(" AND domain = $%d", argIdx)
		args = append(args, domain)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.RegisteredModel, 0)
	for rows.Next() {
		var rm models.RegisteredModel
		var stageStr string
		if err := rows.Scan(
			&rm.ID, &rm.Name, &rm.Description, &rm.Domain, &rm.Framework,
			&stageStr, &rm.TenantID, &rm.CreatedBy, &rm.CreatedAt, &rm.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rm.LatestStage = models.ModelStage(stageStr)
		res = append(res, &rm)
	}
	return res, nil
}

func (p *PGStore) CreateModelVersion(ctx context.Context, mv *models.ModelVersion) error {
	if mv.ID == "" {
		mv.ID = fmt.Sprintf("%s-v%d", mv.ModelID, mv.Version)
	}
	metricsJSON, _ := json.Marshal(mv.MetricsSummary)
	query := `
		INSERT INTO statdata.model_versions (
			id, model_id, version, stage, source_run_id,
			artifact_uri, metrics_summary, input_schema, output_schema,
			description, tenant_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := p.db.ExecContext(ctx, query,
		mv.ID, mv.ModelID, mv.Version, string(mv.Stage), mv.SourceRunID,
		mv.ArtifactURI, metricsJSON, mv.InputSchema, mv.OutputSchema,
		mv.Description, mv.TenantID, mv.CreatedBy,
	)
	return err
}

func (p *PGStore) ListModelVersions(ctx context.Context, modelID, tenantID string) ([]*models.ModelVersion, error) {
	query := `
		SELECT id, model_id, version, stage, COALESCE(source_run_id, ''),
		       artifact_uri, metrics_summary, COALESCE(input_schema, ''), COALESCE(output_schema, ''),
		       COALESCE(description, ''), tenant_id, created_by, created_at, updated_at
		FROM statdata.model_versions
		WHERE model_id = $1
		ORDER BY version DESC
	`
	rows, err := p.db.QueryContext(ctx, query, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.ModelVersion, 0)
	for rows.Next() {
		var mv models.ModelVersion
		var metricsJSON []byte
		var stageStr string
		if err := rows.Scan(
			&mv.ID, &mv.ModelID, &mv.Version, &stageStr, &mv.SourceRunID,
			&mv.ArtifactURI, &metricsJSON, &mv.InputSchema, &mv.OutputSchema,
			&mv.Description, &mv.TenantID, &mv.CreatedBy, &mv.CreatedAt, &mv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		mv.Stage = models.ModelStage(stageStr)
		_ = json.Unmarshal(metricsJSON, &mv.MetricsSummary)
		res = append(res, &mv)
	}
	return res, nil
}

func (p *PGStore) UpdateModelStage(ctx context.Context, versionID string, stage models.ModelStage) error {
	query := `
		UPDATE statdata.model_versions SET stage = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2
	`
	_, err := p.db.ExecContext(ctx, query, string(stage), versionID)
	return err
}

func (p *PGStore) RegisterComputeNode(ctx context.Context, node *models.ComputeNode) error {
	if node.ID == "" {
		node.ID = "node-" + uuid.New().String()[:8]
	}
	query := `
		INSERT INTO statdata.compute_nodes (
			id, hostname, ip_address, total_cpus, alloc_cpus,
			total_ram_gb, alloc_ram_gb, total_gpus, alloc_gpus,
			status, tenant_id, last_ping
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			alloc_cpus = EXCLUDED.alloc_cpus,
			alloc_ram_gb = EXCLUDED.alloc_ram_gb,
			alloc_gpus = EXCLUDED.alloc_gpus,
			status = EXCLUDED.status,
			last_ping = CURRENT_TIMESTAMP
	`
	_, err := p.db.ExecContext(ctx, query,
		node.ID, node.Hostname, node.IPAddress, node.TotalCPUs, node.AllocCPUs,
		node.TotalRAMGB, node.AllocRAMGB, node.TotalGPUs, node.AllocGPUs,
		node.Status, node.TenantID,
	)
	return err
}

func (p *PGStore) ListComputeNodes(ctx context.Context, tenantID string) ([]*models.ComputeNode, error) {
	query := `
		SELECT id, hostname, ip_address, total_cpus, alloc_cpus,
		       total_ram_gb, alloc_ram_gb, total_gpus, alloc_gpus,
		       status, tenant_id, last_ping
		FROM statdata.compute_nodes
		ORDER BY hostname ASC
	`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.ComputeNode, 0)
	for rows.Next() {
		var n models.ComputeNode
		if err := rows.Scan(
			&n.ID, &n.Hostname, &n.IPAddress, &n.TotalCPUs, &n.AllocCPUs,
			&n.TotalRAMGB, &n.AllocRAMGB, &n.TotalGPUs, &n.AllocGPUs,
			&n.Status, &n.TenantID, &n.LastPing,
		); err != nil {
			return nil, err
		}
		res = append(res, &n)
	}
	return res, nil
}

func (p *PGStore) CreateComputeJob(ctx context.Context, job *models.ComputeJob) error {
	if job.ID == "" {
		job.ID = "cjob-" + uuid.New().String()[:8]
	}
	paramJSON, _ := json.Marshal(job.Params)
	query := `
		INSERT INTO statdata.compute_jobs (
			id, name, job_type, assigned_node, required_cpus,
			required_ram_gb, required_gpus, status, params,
			tenant_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'QUEUED', $8, $9, $10)
	`
	_, err := p.db.ExecContext(ctx, query,
		job.ID, job.Name, job.JobType, job.AssignedNode, job.RequiredCPUs,
		job.RequiredRAMGB, job.RequiredGPUs, paramJSON, job.TenantID, job.CreatedBy,
	)
	return err
}

func (p *PGStore) GetComputeJobByID(ctx context.Context, id string) (*models.ComputeJob, error) {
	query := `
		SELECT id, name, job_type, COALESCE(assigned_node, ''), required_cpus,
		       required_ram_gb, required_gpus, status, params, output_data,
		       tenant_id, created_by, created_at, completed_at
		FROM statdata.compute_jobs WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var j models.ComputeJob
	var paramJSON, outJSON []byte
	err := row.Scan(
		&j.ID, &j.Name, &j.JobType, &j.AssignedNode, &j.RequiredCPUs,
		&j.RequiredRAMGB, &j.RequiredGPUs, &j.Status, &paramJSON, &outJSON,
		&j.TenantID, &j.CreatedBy, &j.CreatedAt, &j.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(paramJSON, &j.Params)
	_ = json.Unmarshal(outJSON, &j.OutputData)
	return &j, nil
}

func (p *PGStore) ListComputeJobs(ctx context.Context, tenantID, status string) ([]*models.ComputeJob, error) {
	query := `
		SELECT id, name, job_type, COALESCE(assigned_node, ''), required_cpus,
		       required_ram_gb, required_gpus, status, params, output_data,
		       tenant_id, created_by, created_at, completed_at
		FROM statdata.compute_jobs WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.ComputeJob, 0)
	for rows.Next() {
		var j models.ComputeJob
		var paramJSON, outJSON []byte
		if err := rows.Scan(
			&j.ID, &j.Name, &j.JobType, &j.AssignedNode, &j.RequiredCPUs,
			&j.RequiredRAMGB, &j.RequiredGPUs, &j.Status, &paramJSON, &outJSON,
			&j.TenantID, &j.CreatedBy, &j.CreatedAt, &j.CompletedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(paramJSON, &j.Params)
		_ = json.Unmarshal(outJSON, &j.OutputData)
		res = append(res, &j)
	}
	return res, nil
}

func (p *PGStore) UpdateComputeJob(ctx context.Context, job *models.ComputeJob) error {
	outJSON, _ := json.Marshal(job.OutputData)
	query := `
		UPDATE statdata.compute_jobs SET
			status = $1, assigned_node = $2, output_data = $3, completed_at = $4
		WHERE id = $5
	`
	_, err := p.db.ExecContext(ctx, query, job.Status, job.AssignedNode, outJSON, job.CompletedAt, job.ID)
	return err
}

// ─── Enterprise Search Implementations ──────────────────────────────────────

func (p *PGStore) CreateSearchIndex(ctx context.Context, idx *models.SearchIndex) error {
	if idx.ID == "" {
		idx.ID = "idx-" + uuid.New().String()[:8]
	}
	query := `
		INSERT INTO statdata.search_indexes (
			id, index_name, document_count, dimension, status, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := p.db.ExecContext(ctx, query, idx.ID, idx.IndexName, idx.DocumentCount, idx.Dimension, idx.Status, idx.TenantID)
	return err
}

func (p *PGStore) GetSearchIndex(ctx context.Context, indexName, tenantID string) (*models.SearchIndex, error) {
	query := `
		SELECT id, index_name, document_count, dimension, status, tenant_id, last_indexed_at, created_at
		FROM statdata.search_indexes WHERE index_name = $1
	`
	row := p.db.QueryRowContext(ctx, query, indexName)
	var idx models.SearchIndex
	err := row.Scan(&idx.ID, &idx.IndexName, &idx.DocumentCount, &idx.Dimension, &idx.Status, &idx.TenantID, &idx.LastIndexedAt, &idx.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &idx, nil
}

func (p *PGStore) IndexDocument(ctx context.Context, doc *models.IndexedDocument) error {
	if doc.ID == "" {
		doc.ID = "doc-" + uuid.New().String()[:8]
	}
	tagsJSON, _ := json.Marshal(doc.Tags)
	metaJSON, _ := json.Marshal(doc.Metadata)
	vecJSON, _ := json.Marshal(doc.Vector)

	query := `
		INSERT INTO statdata.indexed_documents (
			id, index_name, resource_id, resource_type, title,
			content, domain, classification, owner, tags,
			metadata, vector, tenant_id, indexed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			vector = EXCLUDED.vector,
			indexed_at = CURRENT_TIMESTAMP
	`
	_, err := p.db.ExecContext(ctx, query,
		doc.ID, doc.IndexName, doc.ResourceID, doc.ResourceType, doc.Title,
		doc.Content, doc.Domain, doc.Classification, doc.Owner, tagsJSON,
		metaJSON, vecJSON, doc.TenantID,
	)
	return err
}

func (p *PGStore) DeleteIndexedDocument(ctx context.Context, indexName, documentID string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM statdata.indexed_documents WHERE id = $1", documentID)
	return err
}

func (p *PGStore) GetIndexedDocument(ctx context.Context, indexName, documentID string) (*models.IndexedDocument, error) {
	query := `
		SELECT id, index_name, resource_id, resource_type, title,
		       content, domain, classification, owner, tags,
		       metadata, vector, tenant_id, indexed_at
		FROM statdata.indexed_documents WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, documentID)
	var doc models.IndexedDocument
	var tagsJSON, metaJSON, vecJSON []byte
	err := row.Scan(
		&doc.ID, &doc.IndexName, &doc.ResourceID, &doc.ResourceType, &doc.Title,
		&doc.Content, &doc.Domain, &doc.Classification, &doc.Owner, &tagsJSON,
		&metaJSON, &vecJSON, &doc.TenantID, &doc.IndexedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(tagsJSON, &doc.Tags)
	_ = json.Unmarshal(metaJSON, &doc.Metadata)
	_ = json.Unmarshal(vecJSON, &doc.Vector)
	return &doc, nil
}

func (p *PGStore) Search(ctx context.Context, req *models.HybridSearchRequest) (*models.SearchResponse, error) {
	start := time.Now()
	// Fetch matching documents and calculate scores
	query := `
		SELECT id, resource_id, resource_type, title, content, domain,
		       classification, metadata, vector, indexed_at
		FROM statdata.indexed_documents WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if req.TenantID != "" && req.TenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, req.TenantID)
		argIdx++
	}
	if req.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, req.ResourceType)
		argIdx++
	}
	if req.Domain != "" {
		query += fmt.Sprintf(" AND domain = $%d", argIdx)
		args = append(args, req.Domain)
		argIdx++
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	queryTokens := strings.Fields(strings.ToLower(req.Query))
	results := make([]models.SearchResultItem, 0)
	facetDomain := make(map[string]int64)
	facetType := make(map[string]int64)

	alpha := req.Alpha
	if alpha <= 0 && len(req.Vector) > 0 {
		alpha = 0.5
	}

	for rows.Next() {
		var doc models.IndexedDocument
		var metaJSON, vecJSON []byte
		if err := rows.Scan(
			&doc.ID, &doc.ResourceID, &doc.ResourceType, &doc.Title, &doc.Content,
			&doc.Domain, &doc.Classification, &metaJSON, &vecJSON, &doc.IndexedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metaJSON, &doc.Metadata)
		_ = json.Unmarshal(vecJSON, &doc.Vector)

		docText := strings.ToLower(fmt.Sprintf("%s %s", doc.Title, doc.Content))
		bm25Score := 0.0
		for _, token := range queryTokens {
			if strings.Contains(docText, token) {
				bm25Score += 1.0
			}
		}

		vecScore := 0.0
		if len(req.Vector) > 0 && len(doc.Vector) > 0 {
			vecScore = cosineSimilarity(req.Vector, doc.Vector)
		}

		finalScore := (1.0-alpha)*bm25Score + alpha*vecScore
		if len(queryTokens) == 0 && len(req.Vector) == 0 {
			finalScore = 1.0
		}

		if finalScore > 0.0 || (len(queryTokens) == 0 && len(req.Vector) == 0) {
			snippet := doc.Content
			if len(snippet) > 180 {
				snippet = snippet[:180] + "..."
			}
			results = append(results, models.SearchResultItem{
				DocumentID:     doc.ID,
				ResourceID:     doc.ResourceID,
				ResourceType:   doc.ResourceType,
				Title:          doc.Title,
				Snippet:        snippet,
				Domain:         doc.Domain,
				Classification: doc.Classification,
				Score:          finalScore,
				BM25Score:      bm25Score,
				VectorScore:    vecScore,
				Metadata:       doc.Metadata,
				IndexedAt:      doc.IndexedAt,
			})
			facetDomain[doc.Domain]++
			facetType[doc.ResourceType]++
		}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	totalHits := int64(len(results))
	offset := req.Offset
	if offset > len(results) {
		results = []models.SearchResultItem{}
	} else {
		end := len(results)
		if offset+limit < end {
			end = offset + limit
		}
		results = results[offset:end]
	}

	facets := []models.FacetResult{
		{Field: "domain", Counts: facetDomain},
		{Field: "resource_type", Counts: facetType},
	}

	return &models.SearchResponse{
		Query:       req.Query,
		TotalHits:   totalHits,
		ExecutionMs: time.Since(start).Milliseconds(),
		Results:     results,
		Facets:      facets,
	}, nil
}

func (p *PGStore) GetSuggestions(ctx context.Context, prefix, tenantID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `
		SELECT title FROM statdata.indexed_documents
		WHERE LOWER(title) LIKE LOWER($1)
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, prefix+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	suggestions := make([]string, 0)
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err == nil {
			suggestions = append(suggestions, title)
		}
	}
	return suggestions, nil
}

func (p *PGStore) CreateSavedSearch(ctx context.Context, ss *models.SavedSearch) error {
	if ss.ID == "" {
		ss.ID = "ss-" + uuid.New().String()[:8]
	}
	filtersJSON, _ := json.Marshal(ss.Filters)
	query := `
		INSERT INTO statdata.saved_searches (
			id, name, query, filters, tenant_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := p.db.ExecContext(ctx, query, ss.ID, ss.Name, ss.Query, filtersJSON, ss.TenantID, ss.CreatedBy)
	return err
}

func (p *PGStore) ListSavedSearches(ctx context.Context, tenantID, createdBy string) ([]*models.SavedSearch, error) {
	query := `
		SELECT id, name, query, filters, tenant_id, created_by, created_at
		FROM statdata.saved_searches WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id = 'default')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if createdBy != "" {
		query += fmt.Sprintf(" AND created_by = $%d", argIdx)
		args = append(args, createdBy)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.SavedSearch, 0)
	for rows.Next() {
		var ss models.SavedSearch
		var filtJSON []byte
		if err := rows.Scan(&ss.ID, &ss.Name, &ss.Query, &filtJSON, &ss.TenantID, &ss.CreatedBy, &ss.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(filtJSON, &ss.Filters)
		res = append(res, &ss)
	}
	return res, nil
}

// ─── Compliance & Cross-App Primitives ──────────────────────────────────────

func (p *PGStore) LogAuditEvent(ctx context.Context, entry *models.AuditLog) error {
	if entry.ID == "" {
		entry.ID = "audit-" + uuid.New().String()[:8]
	}
	detailsJSON, _ := json.Marshal(entry.Details)
	query := `
		INSERT INTO statdata.audit_logs (
			id, action, resource_type, resource_id, actor_id,
			actor_tenant_id, status, details, ip_address, event_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		entry.ID, entry.Action, entry.ResourceType, entry.ResourceID,
		entry.ActorID, entry.ActorTenantID, entry.Status, detailsJSON, entry.IPAddress,
	)
	return err
}

func (p *PGStore) ListAuditLogs(ctx context.Context, tenantID, resourceType string, limit int) ([]*models.AuditLog, error) {
	query := `
		SELECT id, action, resource_type, resource_id, actor_id,
		       actor_tenant_id, status, details, COALESCE(ip_address, ''), event_timestamp
		FROM statdata.audit_logs WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1
	if tenantID != "" && tenantID != "default" {
		query += fmt.Sprintf(" AND (actor_tenant_id = $%d OR actor_tenant_id = 'default' OR actor_tenant_id = '')", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if resourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, resourceType)
		argIdx++
	}
	query += " ORDER BY event_timestamp DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.AuditLog, 0)
	for rows.Next() {
		var entry models.AuditLog
		var detJSON []byte
		if err := rows.Scan(
			&entry.ID, &entry.Action, &entry.ResourceType, &entry.ResourceID,
			&entry.ActorID, &entry.ActorTenantID, &entry.Status, &detJSON,
			&entry.IPAddress, &entry.EventTimestamp,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(detJSON, &entry.Details)
		res = append(res, &entry)
	}
	return res, nil
}

func (p *PGStore) CreateObjectLink(ctx context.Context, link *models.ObjectLink) error {
	metaJSON, _ := json.Marshal(link.Metadata)
	query := `
		INSERT INTO statdata.object_links (
			source_type, source_id, target_type, target_id, relation_type,
			tenant_id, metadata, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := p.db.ExecContext(ctx, query,
		link.SourceType, link.SourceID, link.TargetType, link.TargetID,
		link.RelationType, link.TenantID, metaJSON, link.CreatedBy,
	)
	return err
}

func (p *PGStore) GetObjectLinks(ctx context.Context, sourceType, sourceID, tenantID string) ([]*models.ObjectLink, error) {
	query := `
		SELECT id, source_type, source_id, target_type, target_id, relation_type,
		       tenant_id, metadata, COALESCE(created_by, ''), created_at
		FROM statdata.object_links
		WHERE (source_type = $1 AND source_id = $2) OR (target_type = $1 AND target_id = $2)
	`
	args := []interface{}{sourceType, sourceID}
	if tenantID != "" && tenantID != "default" {
		query += " AND (tenant_id = $3 OR tenant_id = 'default' OR tenant_id = '')"
		args = append(args, tenantID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]*models.ObjectLink, 0)
	for rows.Next() {
		var l models.ObjectLink
		var metaJSON []byte
		if err := rows.Scan(
			&l.ID, &l.SourceType, &l.SourceID, &l.TargetType, &l.TargetID,
			&l.RelationType, &l.TenantID, &metaJSON, &l.CreatedBy, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metaJSON, &l.Metadata)
		res = append(res, &l)
	}
	return res, nil
}
