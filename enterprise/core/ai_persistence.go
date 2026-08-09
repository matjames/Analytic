package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	_ "github.com/lib/pq"
)

// aiDB is the authoritative persistence repository for enterprise AI records.
// Redis remains useful for events and caching but is never the source of truth.
var aiDB *sql.DB

func enterpriseAIDatabaseURL() string {
	if dsn := getEnvValue("ENTERPRISE_AI_DATABASE_URL"); dsn != "" {
		return dsn
	}
	host := getEnv("ENTERPRISE_DB_HOST", "")
	if host == "" {
		return ""
	}
	port, database := getEnv("ENTERPRISE_DB_PORT", "5432"), getEnv("ENTERPRISE_DB_NAME", "statgate_ml_staging")
	user, password := getEnv("ENTERPRISE_DB_USER", ""), getEnv("ENTERPRISE_DB_PASSWORD", "")
	sslmode := getEnv("ENTERPRISE_DB_SSLMODE", "disable")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", url.QueryEscape(user), url.QueryEscape(password), host, port, database, sslmode)
}

func initAIPersistence() {
	dsn := enterpriseAIDatabaseURL()
	if dsn == "" {
		log.Printf("enterprise AI persistence disabled: ENTERPRISE_DB_HOST or ENTERPRISE_AI_DATABASE_URL is not configured")
		return
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("enterprise AI persistence unavailable: %v", err)
		return
	}
	if err = db.Ping(); err != nil {
		log.Printf("enterprise AI persistence unavailable: %v", err)
		_ = db.Close()
		return
	}
	aiDB = db
	if err := ensureAIStorageSchema(); err != nil {
		log.Printf("enterprise AI persistence schema unavailable: %v", err)
		_ = db.Close()
		aiDB = nil
		return
	}
	if err := loadInvestigationsFromDB(); err != nil {
		log.Printf("enterprise AI investigation hydration failed: %v", err)
	}
}

func ensureAIStorageSchema() error {
	_, err := aiDB.Exec(`CREATE TABLE IF NOT EXISTS enterprise_ai_investigations (id VARCHAR(100) PRIMARY KEY,title TEXT NOT NULL,question TEXT NOT NULL,owner_id VARCHAR(100) NOT NULL,tenant_id VARCHAR(100) NOT NULL,organization_id VARCHAR(100),project_id VARCHAR(100),status VARCHAR(40) NOT NULL,evidence JSONB NOT NULL DEFAULT '[]'::jsonb,response_id VARCHAR(100),recommendation_id VARCHAR(100),task_id VARCHAR(100),workflow_id VARCHAR(100),decision_id VARCHAR(100),outcome TEXT,created_at TIMESTAMPTZ NOT NULL,updated_at TIMESTAMPTZ NOT NULL); ALTER TABLE enterprise_ai_investigations ADD COLUMN IF NOT EXISTS source_application VARCHAR(100), ADD COLUMN IF NOT EXISTS source_object_type VARCHAR(100), ADD COLUMN IF NOT EXISTS source_object_id VARCHAR(100), ADD COLUMN IF NOT EXISTS priority VARCHAR(40), ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(100); CREATE INDEX IF NOT EXISTS idx_enterprise_ai_investigations_tenant_owner ON enterprise_ai_investigations (tenant_id, owner_id, updated_at DESC);`)
	return err
}

func loadInvestigationsFromDB() error {
	rows, err := aiDB.Query(`SELECT id, title, question, owner_id, tenant_id, COALESCE(organization_id,''), COALESCE(project_id,''), COALESCE(source_application,''), COALESCE(source_object_type,''), COALESCE(source_object_id,''), COALESCE(priority,''), COALESCE(correlation_id,''), status, evidence, COALESCE(response_id,''), COALESCE(recommendation_id,''), COALESCE(task_id,''), COALESCE(workflow_id,''), COALESCE(decision_id,''), COALESCE(outcome,''), created_at::text, updated_at::text FROM enterprise_ai_investigations`)
	if err != nil {
		return err
	}
	defer rows.Close()
	loaded := make(map[string]AIInvestigation)
	for rows.Next() {
		var item AIInvestigation
		var evidence []byte
		if err := rows.Scan(&item.ID, &item.Title, &item.Question, &item.Owner, &item.TenantID, &item.OrganizationID, &item.ProjectID, &item.SourceApp, &item.SourceObjectType, &item.SourceObjectID, &item.Priority, &item.CorrelationID, &item.Status, &evidence, &item.ResponseID, &item.RecommendationID, &item.TaskID, &item.WorkflowID, &item.DecisionID, &item.Outcome, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return err
		}
		_ = json.Unmarshal(evidence, &item.Evidence)
		loaded[item.ID] = item
	}
	investigationStore.Lock()
	for id, item := range loaded {
		investigationStore.items[id] = item
	}
	investigationStore.Unlock()
	return rows.Err()
}

func persistInvestigationToDB(item AIInvestigation) error {
	if aiDB == nil {
		return nil
	}
	evidence, err := json.Marshal(item.Evidence)
	if err != nil {
		return err
	}
	_, err = aiDB.Exec(`INSERT INTO enterprise_ai_investigations (id,title,question,owner_id,tenant_id,organization_id,project_id,source_application,source_object_type,source_object_id,priority,correlation_id,status,evidence,response_id,recommendation_id,task_id,workflow_id,decision_id,outcome,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),NULLIF($10,''),NULLIF($11,''),$12,$13,$14,NULLIF($15,''),NULLIF($16,''),NULLIF($17,''),NULLIF($18,''),NULLIF($19,''),NULLIF($20,''),$21,$22) ON CONFLICT (id) DO UPDATE SET source_application=EXCLUDED.source_application,source_object_type=EXCLUDED.source_object_type,source_object_id=EXCLUDED.source_object_id,priority=EXCLUDED.priority,correlation_id=EXCLUDED.correlation_id,status=EXCLUDED.status,evidence=EXCLUDED.evidence,recommendation_id=EXCLUDED.recommendation_id,task_id=EXCLUDED.task_id,workflow_id=EXCLUDED.workflow_id,decision_id=EXCLUDED.decision_id,outcome=EXCLUDED.outcome,updated_at=EXCLUDED.updated_at`, item.ID, item.Title, item.Question, item.Owner, item.TenantID, item.OrganizationID, item.ProjectID, item.SourceApp, item.SourceObjectType, item.SourceObjectID, item.Priority, item.CorrelationID, item.Status, evidence, item.ResponseID, item.RecommendationID, item.TaskID, item.WorkflowID, item.DecisionID, item.Outcome, item.CreatedAt, item.UpdatedAt)
	return err
}
