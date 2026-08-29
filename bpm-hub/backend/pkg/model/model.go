package model

import "time"

// ProcessDefinition is a BPMN-ish workflow: nodes + transitions (P48).
type ProcessDefinition struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	WorkspaceID string                 `json:"workspace_id,omitempty"`
	Name        string                 `json:"name"`
	Key         string                 `json:"key"`
	Version     int                    `json:"version"`
	Description string                 `json:"description,omitempty"`
	StartNode   string                 `json:"start_node"`
	Nodes       map[string]interface{} `json:"nodes,omitempty"`
	Transitions map[string]interface{} `json:"transitions,omitempty"`
	Status      string                 `json:"status"` // draft|active|retired
	CreatedBy   string                 `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ProcessInstance is a running instance of a process definition (P48).
type ProcessInstance struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	WorkspaceID  string                 `json:"workspace_id,omitempty"`
	DefinitionID string                 `json:"definition_id"`
	Status       string                 `json:"status"` // created|running|completed|errored|cancelled
	CurrentNode  string                 `json:"current_node"`
	Context      map[string]interface{} `json:"context,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	EndedAt      *time.Time             `json:"ended_at,omitempty"`
	CreatedBy    string                 `json:"created_by,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// CaseInstance is a case-management record aggregating related items (P48).
type CaseInstance struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // open|in_progress|on_hold|closed
	Priority  string    `json:"priority"`
	Owner     string    `json:"owner,omitempty"`
	DueAt     *time.Time `json:"due_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CaseItem is an object/record attached to a case (P48).
type CaseItem struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	CaseID     string                 `json:"case_id"`
	ItemType   string                 `json:"item_type"`
	Content    map[string]interface{} `json:"content,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// WorkItem is a task created while executing a process or case (P48 task mgmt).
type WorkItem struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	InstanceID  string                 `json:"instance_id,omitempty"`
	NodeID      string                 `json:"node_id,omitempty"`
	Name        string                 `json:"name"`
	Assignee    string                 `json:"assignee,omitempty"`
	Status      string                 `json:"status"` // todo|doing|done|failed|skipped
	Payload     map[string]interface{} `json:"payload,omitempty"`
	DueAt       *time.Time             `json:"due_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ActivityLog is a process-mining event: who did what, when, where (P48).
type ActivityLog struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	InstanceID string                 `json:"instance_id,omitempty"`
	CaseID     string                 `json:"case_id,omitempty"`
	Action     string                 `json:"action"`
	NodeID     string                 `json:"node_id,omitempty"`
	Actor      string                 `json:"actor,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
	At         time.Time              `json:"at"`
}

// AutomationRule triggers actions when a matching domain event is seen (P48
// automation service).
type AutomationRule struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	Name         string                 `json:"name"`
	TriggerEvent string                 `json:"trigger_event"`
	Condition    map[string]interface{} `json:"condition,omitempty"`
	Actions      map[string]interface{} `json:"actions,omitempty"`
	Enabled      bool                   `json:"enabled"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ObjectLink mirrors the shared object_links contract.
type ObjectLink struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	SourceType   string    `json:"source_type"`
	SourceID     string    `json:"source_id"`
	TargetType   string    `json:"target_type"`
	TargetID     string    `json:"target_id"`
	Relationship string    `json:"relationship,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}