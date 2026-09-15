package models

import "time"

// ObjectLink defines cross-application relationships in the StatGate ecosystem
type ObjectLink struct {
	ID           int                    `json:"id"`
	SourceType   string                 `json:"source_type"`
	SourceID     string                 `json:"source_id"`
	TargetType   string                 `json:"target_type"`
	TargetID     string                 `json:"target_id"`
	RelationType string                 `json:"relation_type"`
	TenantID     string                 `json:"tenant_id"`
	WorkspaceID  string                 `json:"workspace_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy    string                 `json:"created_by,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// AuditLog captures security and operational activities across App 12
type AuditLog struct {
	ID             string                 `json:"id"`
	Action         string                 `json:"action"`
	ResourceType   string                 `json:"resource_type"`
	ResourceID     string                 `json:"resource_id"`
	ActorID        string                 `json:"actor_id"`
	ActorTenantID  string                 `json:"actor_tenant_id"`
	WorkspaceID    string                 `json:"workspace_id,omitempty"`
	Status         string                 `json:"status"` // SUCCESS, FAILED, DENIED
	Details        map[string]interface{} `json:"details,omitempty"`
	IPAddress      string                 `json:"ip_address,omitempty"`
	EventTimestamp time.Time              `json:"event_timestamp"`
}

// EventEnvelope standardizes event messaging across statgate:events
type EventEnvelope struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Source    string                 `json:"source"`
	TenantID  string                 `json:"tenant_id"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// HealthResponse represents standard health endpoint response
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}
