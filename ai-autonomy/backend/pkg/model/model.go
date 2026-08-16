package model

import "time"

// DigitalTwin is a virtual representation of a real-world entity (project,
// facility, district, population, supply chain...) that can be simulated (P22).
type DigitalTwin struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	EntityType  string                 `json:"entity_type"` // project|facility|district|process
	EntityID    string                 `json:"entity_id,omitempty"` // source object registered in object_links
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	State       map[string]interface{} `json:"state,omitempty"`
	Status      string                 `json:"status"` // active|archived
	CreatedBy   string                 `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Simulation is a run of a digital twin under a scenario (P22).
type Simulation struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	TwinID     string                 `json:"twin_id"`
	Name       string                 `json:"name"`
	Scenario   map[string]interface{} `json:"scenario,omitempty"`
	Input      map[string]interface{} `json:"input,omitempty"`
	Results    map[string]interface{} `json:"results,omitempty"`
	Status     string                 `json:"status"` // queued|running|completed|failed
	Error      string                 `json:"error,omitempty"`
	StartedAt  *time.Time             `json:"started_at,omitempty"`
	FinishedAt *time.Time             `json:"finished_at,omitempty"`
	CreatedBy  string                 `json:"created_by,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// AIModel is a registered predictive model exposed by the runtime (P22).
type AIModel struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	Name      string                 `json:"name"`
	Kind      string                 `json:"kind"` // forecasting|classifier|recommender|simulation|anomaly
	Version   string                 `json:"version"`
	Config    map[string]interface{} `json:"config,omitempty"`
	Status    string                 `json:"status"` // draft|active|retired
	CreatedBy string                 `json:"created_by,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Prediction is an inference produced by a model over a target object (P22).
type Prediction struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	ModelID     string                 `json:"model_id"`
	TargetType  string                 `json:"target_type"`
	TargetID    string                 `json:"target_id"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Prediction  map[string]interface{} `json:"prediction,omitempty"`
	Confidence  float64                `json:"confidence"`
	Horizon     string                 `json:"horizon,omitempty"`
	Status      string                 `json:"status"` // in_progress|completed|failed
	TriggeredBy string                 `json:"triggered_by"` // manual|event
	SourceEvent string                 `json:"source_event,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ─── Multi-Agent System (P31) ────────────────────────────────────────────────

// Agent is an autonomous AI worker (P31).
type Agent struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	Name         string                 `json:"name"`
	Role         string                 `json:"role"` // analyst|researcher|auditor|classifier...
	Persona      map[string]interface{} `json:"persona,omitempty"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
	SafetyPolicy map[string]interface{} `json:"safety_policy,omitempty"`
	Status       string                 `json:"status"` // registered|active|paused|retired
	CreatedBy    string                 `json:"created_by,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// AgentTask is a work item executed by an agent, optionally triggered by events.
type AgentTask struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	AgentID     string                 `json:"agent_id"`
	Name        string                 `json:"name"`
	TaskType    string                 `json:"task_type"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	TriggerType string                 `json:"trigger_type"` // manual|event|system
	TriggeredBy string                 `json:"triggered_by,omitempty"`
	Status      string                 `json:"status"` // queued|running|completed|failed
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	FinishedAt  *time.Time             `json:"finished_at,omitempty"`
}

// AgentMemory is an observation, fact or reflection stored for an agent (P31).
type AgentMemory struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	AgentID   string                 `json:"agent_id"`
	Kind      string                 `json:"kind"` // observation|fact|reflection|decision
	Content   map[string]interface{} `json:"content,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// AgentMessage is a single exchange on the Agent Communications Bus (P31).
type AgentMessage struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	FromAgent string                 `json:"from_agent"`
	ToAgent   string                 `json:"to_agent"`
	Type      string                 `json:"type"` // request|inform|delegate|reply
	Payload   map[string]interface{} `json:"payload,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// ─── Knowledge Graph & Semantic Intelligence (P39) ───────────────────────────

// GraphNode is an entity in the enterprise knowledge graph (P39).
type GraphNode struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	Type          string                 `json:"type"`
	Label         string                 `json:"label"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
	RefObjectType string                 `json:"ref_object_type,omitempty"`
	RefObjectID   string                 `json:"ref_object_id,omitempty"`
	CreatedBy     string                 `json:"created_by,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// GraphEdge is a typed relationship between two nodes (P39).
type GraphEdge struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	SourceNode string                 `json:"source_node"`
	TargetNode string                 `json:"target_node"`
	Predicate  string                 `json:"predicate"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	CreatedBy  string                 `json:"created_by,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// Triple is a semantic statement (subject, predicate, object) with provenance (P39).
type Triple struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Subject    string    `json:"subject"`
	Predicate  string    `json:"predicate"`
	Object     string    `json:"object"`
	Confidence float64   `json:"confidence"`
	Provenance string    `json:"provenance,omitempty"`
	CreatedBy  string    `json:"created_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ContextEntry feeds the contextual intelligence indexer (P39).
type ContextEntry struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	Content    string    `json:"content"`
	Tokens     []string  `json:"tokens,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ContextHit is a scored result from the contextual indexer.
type ContextHit struct {
	EntityType string  `json:"entity_type"`
	EntityID   string  `json:"entity_id"`
	Score      float64 `json:"score"`
	Content    string  `json:"content,omitempty"`
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
// DecisionPipeline is a decision-intelligence workflow definition (P22).
type DecisionPipeline struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Definition  map[string]interface{} `json:"definition,omitempty"`
	Status      string                 `json:"status"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// GovernanceAction is an audited agent/prediction decision with a risk verdict
// enforced by the AI Safety & Governance guardrails (P31).
type GovernanceAction struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	AgentID    string                 `json:"agent_id,omitempty"`
	Action     string                 `json:"action"` // agent_action|prediction_release|simulation
	Decision   map[string]interface{} `json:"decision,omitempty"`
	RiskScore  float64                `json:"risk_score"`
	Verdict    string                 `json:"verdict"` // auto_approved|blocked|human_review
	ReviewedBy string                 `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time             `json:"reviewed_at,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}