package main

// ─── Phase V: Enterprise Decision & Workflow Automation Models ─────
// This file defines the core data structures for the StatGate
// Enterprise Workflow Engine, Approval Engine, Task Management,
// Escalation Engine, SLA Management, Decision Records, Action Centre
// and Process Analytics services.

// ─── Workflow Engine ───────────────────────────────────────────────
// A WorkflowDefinition is a configurable, reusable workflow that can
// be triggered by enterprise events or manually. Workflows are
// permission-aware, auditable, recoverable and human-supervised.

type WorkflowDefinition struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Category      string                 `json:"category"` // project | survey | research | helpdesk | data_quality | general
	Version       int                    `json:"version"`
	Status        string                 `json:"status"` // draft | active | deprecated
	Trigger       WorkflowTrigger        `json:"trigger"`
	Conditions    []WorkflowCondition    `json:"conditions,omitempty"`
	Steps         []WorkflowStep         `json:"steps"`
	Assignment    WorkflowAssignment     `json:"assignment,omitempty"`
	Deadline      *WorkflowDeadline      `json:"deadline,omitempty"`
	Escalation    *EscalationPolicy      `json:"escalation,omitempty"`
	Completion    *WorkflowCompletion    `json:"completion,omitempty"`
	Approval      *ApprovalConfig        `json:"approval,omitempty"`
	RequiresHuman bool                   `json:"requires_human"` // human-in-the-loop control
	OwnerRole     string                 `json:"owner_role,omitempty"`
	Scope         string                 `json:"scope,omitempty"` // organization | department | region | project | team | facility
	ScopeID       string                 `json:"scope_id,omitempty"`
	Permissions   []string               `json:"permissions,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy     string                 `json:"created_by,omitempty"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

type WorkflowTrigger struct {
	Type        string                 `json:"type"` // event | manual | schedule | alert | anomaly
	EventType   string                 `json:"event_type,omitempty"`
	SourceApp   string                 `json:"source_app,omitempty"`
	Schedule    string                 `json:"schedule,omitempty"` // cron-like description
	AlertRuleID string                 `json:"alert_rule_id,omitempty"`
	AnomalyRule string                 `json:"anomaly_rule,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
}

type WorkflowCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // equals | not_equals | contains | greater_than | less_than | exists | in
	Value    interface{} `json:"value,omitempty"`
}

type WorkflowStep struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`                 // action | approval | task | notification | condition | wait
	Action      string                 `json:"action,omitempty"`     // create_task | create_ticket | notify | create_chat | update_entity | create_calendar | create_decision
	TargetApp   string                 `json:"target_app,omitempty"` // pms | rms | helpdesk | statchat | enterprise
	Params      map[string]interface{} `json:"params,omitempty"`
	Assignee    string                 `json:"assignee,omitempty"`
	Role        string                 `json:"role,omitempty"`
	Deadline    string                 `json:"deadline,omitempty"`
	NextOnPass  string                 `json:"next_on_pass,omitempty"`
	NextOnFail  string                 `json:"next_on_fail,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty"`
	TimeoutSecs int                    `json:"timeout_secs,omitempty"`
	Order       int                    `json:"order"`
}

type WorkflowAssignment struct {
	Type          string `json:"type"` // user | role | manager | supervisor | project_owner | auto
	UserID        string `json:"user_id,omitempty"`
	Role          string `json:"role,omitempty"`
	Fallback      string `json:"fallback,omitempty"`
	AllowReassign bool   `json:"allow_reassign"`
}

type WorkflowDeadline struct {
	DurationHours int    `json:"duration_hours"`
	From          string `json:"from"` // start | step_completion | trigger
	ReminderHours int    `json:"reminder_hours,omitempty"`
}

type WorkflowCompletion struct {
	Type        string             `json:"type"` // all_steps | condition | manual | metric_improved
	Condition   *WorkflowCondition `json:"condition,omitempty"`
	Metric      string             `json:"metric,omitempty"`
	Threshold   float64            `json:"threshold,omitempty"`
	Operator    string             `json:"operator,omitempty"`
	VerifyAfter string             `json:"verify_after,omitempty"`
}

// WorkflowInstance is a running execution of a workflow definition.
type WorkflowInstance struct {
	ID              string                 `json:"id"`
	WorkflowID      string                 `json:"workflow_id"`
	WorkflowName    string                 `json:"workflow_name"`
	Status          string                 `json:"status"` // pending | running | waiting_approval | waiting_task | completed | failed | cancelled | escalated
	CurrentStep     string                 `json:"current_step,omitempty"`
	TriggerEventID  string                 `json:"trigger_event_id,omitempty"`
	TriggerType     string                 `json:"trigger_type,omitempty"`
	EntityType      string                 `json:"entity_type,omitempty"`
	EntityID        string                 `json:"entity_id,omitempty"`
	ProjectID       string                 `json:"project_id,omitempty"`
	OrgID           string                 `json:"organization_id,omitempty"`
	Scope           string                 `json:"scope,omitempty"`
	ScopeID         string                 `json:"scope_id,omitempty"`
	Assignee        string                 `json:"assignee,omitempty"`
	Context         map[string]interface{} `json:"context,omitempty"`
	CorrelationID   string                 `json:"correlation_id,omitempty"`
	StartedAt       string                 `json:"started_at"`
	CompletedAt     string                 `json:"completed_at,omitempty"`
	DueAt           string                 `json:"due_at,omitempty"`
	EscalationLevel int                    `json:"escalation_level,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowExecution records one step execution for audit.
type WorkflowExecution struct {
	ID            string                   `json:"id"`
	InstanceID    string                   `json:"instance_id"`
	WorkflowID    string                   `json:"workflow_id"`
	StepID        string                   `json:"step_id"`
	StepName      string                   `json:"step_name"`
	Action        string                   `json:"action,omitempty"`
	Status        string                   `json:"status"` // pending | running | succeeded | failed | skipped | retrying
	Trigger       string                   `json:"trigger,omitempty"`
	EventType     string                   `json:"event_type,omitempty"`
	Conditions    []map[string]interface{} `json:"conditions_evaluated,omitempty"`
	Actions       []map[string]interface{} `json:"actions_performed,omitempty"`
	User          string                   `json:"user,omitempty"`
	System        string                   `json:"system,omitempty"`
	Result        string                   `json:"result,omitempty"`
	Error         string                   `json:"error,omitempty"`
	CorrelationID string                   `json:"correlation_id,omitempty"`
	RetryCount    int                      `json:"retry_count,omitempty"`
	Timestamp     string                   `json:"timestamp"`
	CompletedAt   string                   `json:"completed_at,omitempty"`
}

// ─── Approval Engine ───────────────────────────────────────────────

type ApprovalConfig struct {
	Type         string   `json:"type"` // single | multiple | sequential | parallel | conditional | delegated
	Approvers    []string `json:"approvers,omitempty"`
	Roles        []string `json:"roles,omitempty"`
	MinApprovals int      `json:"min_approvals,omitempty"`
	ExpiryHours  int      `json:"expiry_hours,omitempty"`
	EscalateTo   string   `json:"escalate_to,omitempty"`
	AllowReject  bool     `json:"allow_reject"`
	AllowChanges bool     `json:"allow_changes"`
}

type ApprovalRequest struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Type            string                 `json:"type"`   // single | multiple | sequential | parallel | conditional | delegated
	Status          string                 `json:"status"` // pending | approved | rejected | changes_requested | expired | escalated
	EntityType      string                 `json:"entity_type,omitempty"`
	EntityID        string                 `json:"entity_id,omitempty"`
	ProjectID       string                 `json:"project_id,omitempty"`
	OrgID           string                 `json:"organization_id,omitempty"`
	Requester       string                 `json:"requester"`
	Approvers       []string               `json:"approvers"`
	CurrentApprover string                 `json:"current_approver,omitempty"`
	MinApprovals    int                    `json:"min_approvals,omitempty"`
	Approvals       []ApprovalDecision     `json:"approvals,omitempty"`
	ExpiresAt       string                 `json:"expires_at,omitempty"`
	EscalatedTo     string                 `json:"escalated_to,omitempty"`
	Context         map[string]interface{} `json:"context,omitempty"`
	WorkflowID      string                 `json:"workflow_id,omitempty"`
	InstanceID      string                 `json:"instance_id,omitempty"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

type ApprovalDecision struct {
	Approver    string `json:"approver"`
	Decision    string `json:"decision"` // approved | rejected | changes_requested | delegated
	Comment     string `json:"comment,omitempty"`
	DelegatedTo string `json:"delegated_to,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// ─── Enterprise Task Management ────────────────────────────────────

type EnterpriseTask struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Status         string                 `json:"status"`   // pending | in_progress | paused | completed | cancelled
	Priority       string                 `json:"priority"` // low | medium | high | critical
	Assignee       string                 `json:"assignee,omitempty"`
	Assigner       string                 `json:"assigner,omitempty"`
	SourceApp      string                 `json:"source_app,omitempty"`
	SourceEntity   string                 `json:"source_entity,omitempty"`
	SourceEntityID string                 `json:"source_entity_id,omitempty"`
	ProjectID      string                 `json:"project_id,omitempty"`
	OrgID          string                 `json:"organization_id,omitempty"`
	FacilityID     string                 `json:"facility_id,omitempty"`
	Region         string                 `json:"region,omitempty"`
	District       string                 `json:"district,omitempty"`
	DueAt          string                 `json:"due_at,omitempty"`
	StartedAt      string                 `json:"started_at,omitempty"`
	PausedAt       string                 `json:"paused_at,omitempty"`
	CompletedAt    string                 `json:"completed_at,omitempty"`
	CancelledAt    string                 `json:"cancelled_at,omitempty"`
	Dependencies   []TaskDependency       `json:"dependencies,omitempty"`
	Comments       []TaskComment          `json:"comments,omitempty"`
	Attachments    []AttachmentRef        `json:"attachments,omitempty"`
	WorkflowID     string                 `json:"workflow_id,omitempty"`
	InstanceID     string                 `json:"instance_id,omitempty"`
	CorrelationID  string                 `json:"correlation_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
}

type TaskDependency struct {
	TaskID   string `json:"task_id"`
	Type     string `json:"type"` // blocks | blocked_by
	Required bool   `json:"required"`
}

type TaskComment struct {
	User      string `json:"user"`
	Comment   string `json:"comment"`
	Timestamp string `json:"timestamp"`
}

// ─── Escalation Engine ─────────────────────────────────────────────

type EscalationPolicy struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Levels      []EscalationLevel `json:"levels"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   string            `json:"created_at"`
}

type EscalationLevel struct {
	Level      int    `json:"level"`
	AfterHours int    `json:"after_hours"`
	NotifyRole string `json:"notify_role,omitempty"`
	NotifyUser string `json:"notify_user,omitempty"`
	Action     string `json:"action,omitempty"` // notify | reassign | cancel | critical
	Message    string `json:"message,omitempty"`
}

// ─── SLA Management ────────────────────────────────────────────────

type SLAConfig struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"` // response | resolution | approval | review | project
	TargetHours      int    `json:"target_hours"`
	WarningHours     int    `json:"warning_hours"`
	EscalationPolicy string `json:"escalation_policy,omitempty"`
	Scope            string `json:"scope,omitempty"`
	ScopeID          string `json:"scope_id,omitempty"`
	Enabled          bool   `json:"enabled"`
	CreatedAt        string `json:"created_at"`
}

type SLABreach struct {
	ID          string                 `json:"id"`
	SLAID       string                 `json:"sla_id"`
	SLAName     string                 `json:"sla_name"`
	Type        string                 `json:"type"`
	EntityType  string                 `json:"entity_type,omitempty"`
	EntityID    string                 `json:"entity_id,omitempty"`
	ProjectID   string                 `json:"project_id,omitempty"`
	Status      string                 `json:"status"` // warning | breached | escalated | resolved
	StartedAt   string                 `json:"started_at"`
	DueAt       string                 `json:"due_at"`
	BreachedAt  string                 `json:"breached_at,omitempty"`
	EscalatedTo string                 `json:"escalated_to,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ─── Decision Records ──────────────────────────────────────────────

type DecisionRecord struct {
	ID                  string                 `json:"id"`
	Decision            string                 `json:"decision"`
	Date                string                 `json:"date"`
	DecisionMaker       string                 `json:"decision_maker"`
	Context             string                 `json:"context,omitempty"`
	SupportingEvidence  []string               `json:"supporting_evidence,omitempty"`
	RelatedProject      string                 `json:"related_project,omitempty"`
	RelatedData         map[string]interface{} `json:"related_data,omitempty"`
	SupportingAnalytics map[string]interface{} `json:"supporting_analytics,omitempty"`
	ActionRequired      string                 `json:"action_required,omitempty"`
	ResponsiblePerson   string                 `json:"responsible_person,omitempty"`
	Deadline            string                 `json:"deadline,omitempty"`
	Outcome             string                 `json:"outcome,omitempty"`
	Status              string                 `json:"status"` // open | in_progress | completed | cancelled
	WorkflowID          string                 `json:"workflow_id,omitempty"`
	InstanceID          string                 `json:"instance_id,omitempty"`
	CorrelationID       string                 `json:"correlation_id,omitempty"`
	CreatedAt           string                 `json:"created_at"`
	UpdatedAt           string                 `json:"updated_at"`
}

// ─── AI Recommendations ────────────────────────────────────────────
// AI recommendations are always distinguishable from confirmed
// organizational decisions. They carry a "recommendation" status and
// require human confirmation before becoming decisions.

type AIRecommendation struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"` // action | assignment | escalation | summary | report | bottleneck
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Rationale       string   `json:"rationale"`
	Evidence        []string `json:"evidence,omitempty"`
	SuggestedAction string   `json:"suggested_action,omitempty"`
	Confidence      float64  `json:"confidence"`
	Status          string   `json:"status"` // pending | accepted | dismissed | converted_to_decision
	RelatedEntity   string   `json:"related_entity,omitempty"`
	RelatedEntityID string   `json:"related_entity_id,omitempty"`
	ProjectID       string   `json:"project_id,omitempty"`
	CreatedAt       string   `json:"created_at"`
	ResolvedAt      string   `json:"resolved_at,omitempty"`
}

// ─── Process Analytics ─────────────────────────────────────────────

type ProcessMetric struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	AverageHours   float64 `json:"average_hours"`
	Count          int     `json:"count"`
	OverdueCount   int     `json:"overdue_count"`
	EscalatedCount int     `json:"escalated_count"`
	FailureCount   int     `json:"failure_count"`
	Bottleneck     bool    `json:"bottleneck"`
	LastUpdated    string  `json:"last_updated"`
}
