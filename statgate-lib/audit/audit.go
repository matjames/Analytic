package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Record defines the authoritative enterprise audit schema.
type Record struct {
	ID            string                 `json:"id"`
	Who           string                 `json:"who"`            // User ID / Principal
	What          string                 `json:"what"`           // Action name (e.g., "project.created", "dataset.updated")
	When          time.Time              `json:"when"`           // UTC Timestamp
	Where         string                 `json:"where"`          // Endpoint / component / host
	Application   string                 `json:"application"`    // pms | rms | statcollect | registry | etc.
	TenantID      string                 `json:"tenant_id"`      // Tenant Context
	OrgID         string                 `json:"org_id,omitempty"`
	ObjectID      string                 `json:"object_id"`      // Target entity identifier
	ObjectType    string                 `json:"object_type"`    // Entity classification
	PreviousState map[string]interface{} `json:"previous_state,omitempty"`
	NewState      map[string]interface{} `json:"new_state,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Service provides audit logging capabilities.
type Service struct {
	db          *sql.DB
	application string
}

// NewService creates a new Audit Service.
func NewService(db *sql.DB, application string) *Service {
	s := &Service{
		db:          db,
		application: application,
	}
	if db != nil {
		_ = s.EnsureSchema(context.Background())
	}
	return s
}

// EnsureSchema creates the immutable enterprise audit table if it does not exist.
func (s *Service) EnsureSchema(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	query := `
	CREATE TABLE IF NOT EXISTS enterprise_audit_log (
		id TEXT PRIMARY KEY,
		who TEXT NOT NULL,
		what TEXT NOT NULL,
		"when" TIMESTAMPTZ NOT NULL,
		"where" TEXT NOT NULL,
		application TEXT NOT NULL,
		tenant_id TEXT NOT NULL,
		org_id TEXT,
		object_id TEXT NOT NULL,
		object_type TEXT NOT NULL,
		previous_state JSONB,
		new_state JSONB,
		request_id TEXT,
		correlation_id TEXT,
		ip_address TEXT,
		user_agent TEXT,
		metadata JSONB,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_audit_tenant ON enterprise_audit_log(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_audit_object ON enterprise_audit_log(object_id);
	CREATE INDEX IF NOT EXISTS idx_audit_when ON enterprise_audit_log("when");
	CREATE INDEX IF NOT EXISTS idx_audit_app ON enterprise_audit_log(application);
	`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// Log writes an immutable audit record.
func (s *Service) Log(ctx context.Context, r Record) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.When.IsZero() {
		r.When = time.Now().UTC()
	}
	if r.Application == "" {
		r.Application = s.application
	}
	if r.TenantID == "" {
		r.TenantID = "default"
	}

	prevJSON, _ := json.Marshal(r.PreviousState)
	newJSON, _ := json.Marshal(r.NewState)
	metaJSON, _ := json.Marshal(r.Metadata)

	if s.db != nil {
		query := `
		INSERT INTO enterprise_audit_log (
			id, who, what, "when", "where", application, tenant_id, org_id,
			object_id, object_type, previous_state, new_state, request_id,
			correlation_id, ip_address, user_agent, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		`
		_, err := s.db.ExecContext(ctx, query,
			r.ID, r.Who, r.What, r.When, r.Where, r.Application, r.TenantID, r.OrgID,
			r.ObjectID, r.ObjectType, prevJSON, newJSON, r.RequestID,
			r.CorrelationID, r.IPAddress, r.UserAgent, metaJSON, time.Now().UTC(),
		)
		if err != nil {
			log.Printf("[Audit:DB-Error] Failed to insert audit record %s: %v", r.ID, err)
			return err
		}
	} else {
		// Output structured audit JSON to stdout for log aggregators
		recordBytes, _ := json.Marshal(r)
		log.Printf("[AUDIT_RECORD] %s", string(recordBytes))
	}

	return nil
}

// ExtractContext extracts client IP and request metadata from HTTP requests.
func ExtractContext(r *http.Request) (ip string, userAgent string) {
	ip = r.Header.Get("X-Forwarded-For")
	if ip != "" {
		ip = strings.Split(ip, ",")[0]
	} else {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	userAgent = r.Header.Get("User-Agent")
	return strings.TrimSpace(ip), userAgent
}

// GlobalService provides a default singleton for services.
var GlobalService = NewService(nil, "statgate-app")

// InitGlobal initializes the global audit service with an active DB.
func InitGlobal(db *sql.DB, appName string) {
	GlobalService = NewService(db, appName)
}
