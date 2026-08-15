package main

import "time"

// ─── Criticality & Status Enums ───────────────────────────────────

type CriticalityTier string

const (
	Tier0_MissionCritical CriticalityTier = "Tier 0" // Core institutional failure
	Tier1_Critical        CriticalityTier = "Tier 1" // Workflows significantly affected
	Tier2_Important       CriticalityTier = "Tier 2" // Productivity affected, core runs
	Tier3_Supporting      CriticalityTier = "Tier 3" // Limited impact
)

type ComplianceStatus string

const (
	ComplianceCompliant ComplianceStatus = "COMPLIANT"
	ComplianceAtRisk    ComplianceStatus = "AT RISK"
	ComplianceBreached  ComplianceStatus = "BREACHED"
	ComplianceNotTested ComplianceStatus = "NOT TESTED"
)

type OperationalStatus string

const (
	OpStatusHealthy     OperationalStatus = "HEALTHY"
	OpStatusDegraded    OperationalStatus = "DEGRADED"
	OpStatusUnavailable OperationalStatus = "UNAVAILABLE"
	OpStatusRecovering  OperationalStatus = "RECOVERING"
)

// ─── Service Resilience Profile ───────────────────────────────────

type ServiceResilienceProfile struct {
	ServiceID          string            `json:"service_id"`
	ServiceName        string            `json:"service_name"`
	CriticalityTier    CriticalityTier   `json:"criticality_tier"`
	BusinessOwner      string            `json:"business_owner"`
	TechnicalOwner     string            `json:"technical_owner"`
	Dependencies       []string          `json:"dependencies"`
	HealthEndpoint     string            `json:"health_endpoint"`
	ReadinessEndpoint  string            `json:"readiness_endpoint"`
	LivenessEndpoint   string            `json:"liveness_endpoint"`
	RTOTargetSec       int               `json:"rto_target_sec"`
	RPOTargetSec       int               `json:"rpo_target_sec"`
	ActualRTOSec       int               `json:"actual_rto_sec"`
	ActualRPOSec       int               `json:"actual_rpo_sec"`
	AvailabilityPct    float64           `json:"availability_pct"`
	LastDrillDate      string            `json:"last_drill_date,omitempty"`
	LastDrillStatus    ComplianceStatus  `json:"last_drill_status"`
	ComplianceStatus   ComplianceStatus  `json:"compliance_status"`
	OperationalStatus  OperationalStatus `json:"operational_status"`
	ConfidenceScore    int               `json:"confidence_score"`
	RecoveryProcedure  string            `json:"recovery_procedure"`
	BackupRequirement  string            `json:"backup_requirement"`
	CreatedAt          string            `json:"created_at"`
	UpdatedAt          string            `json:"updated_at"`
}

// ─── Disaster Recovery Drills ─────────────────────────────────────

type DrillMode string

const (
	DrillModeSimulation     DrillMode = "SIMULATION"
	DrillModeControlledTest DrillMode = "CONTROLLED_TEST"
	DrillModeProduction     DrillMode = "PRODUCTION"
)

type DrillStatus string

const (
	DrillScheduled DrillStatus = "SCHEDULED"
	DrillRunning   DrillStatus = "RUNNING"
	DrillCompleted DrillStatus = "COMPLETED"
	DrillFailed    DrillStatus = "FAILED"
	DrillCancelled DrillStatus = "CANCELLED"
)

type DrillResult string

const (
	DrillResultSuccess  DrillResult = "SUCCESS"
	DrillResultPartial  DrillResult = "PARTIAL"
	DrillResultBreached DrillResult = "BREACHED"
	DrillResultFailed   DrillResult = "FAILED"
	DrillResultPending  DrillResult = "PENDING"
)

type RecoveryDrill struct {
	ID             string              `json:"id"`
	Title          string              `json:"title"`
	ServiceID      string              `json:"service_id"`
	ServiceName    string              `json:"service_name,omitempty"`
	Scenario       string              `json:"scenario"`
	Mode           DrillMode           `json:"mode"`
	Status         DrillStatus         `json:"status"`
	RTOTargetSec   int                 `json:"rto_target_sec"`
	RPOTargetSec   int                 `json:"rpo_target_sec"`
	MeasuredRTOSec int                 `json:"measured_rto_sec"`
	MeasuredRPOSec int                 `json:"measured_rpo_sec"`
	DrillResult    DrillResult         `json:"drill_result"`
	StartedAt      string              `json:"started_at,omitempty"`
	CompletedAt    string              `json:"completed_at,omitempty"`
	ConductedBy    string              `json:"conducted_by"`
	EvidenceID     string              `json:"evidence_id,omitempty"`
	Summary        string              `json:"summary"`
	Steps          []RecoveryDrillStep `json:"steps,omitempty"`
	CreatedAt      string              `json:"created_at"`
}

type RecoveryDrillStep struct {
	ID           string `json:"id"`
	DrillID      string `json:"drill_id"`
	StepNumber   int    `json:"step_number"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ActionType   string `json:"action_type"`
	Status       string `json:"status"` // PENDING, RUNNING, PASSED, FAILED, SKIPPED
	DurationMs   int    `json:"duration_ms"`
	Output       string `json:"output"`
	ErrorMessage string `json:"error_message,omitempty"`
	CreatedAt    string `json:"created_at"`
}

// ─── Backup Assurance & Restore Tests ─────────────────────────────

type BackupType string

const (
	BackupFullSnapshot   BackupType = "FULL_SNAPSHOT"
	BackupIncrementalWAL BackupType = "INCREMENTAL_WAL"
	BackupLogicalDump    BackupType = "LOGICAL_DUMP"
	BackupConfigState    BackupType = "CONFIG_STATE"
)

type BackupRecord struct {
	ID               string   `json:"id"`
	ServiceID        string   `json:"service_id"`
	ServiceName      string   `json:"service_name,omitempty"`
	BackupType       BackupType `json:"backup_type"`
	DataScope        string   `json:"data_scope"`
	StorageLocation  string   `json:"storage_location"`
	EncryptionStatus string   `json:"encryption_status"`
	RetentionPolicy  string   `json:"retention_policy"`
	SizeBytes        int64    `json:"size_bytes"`
	ChecksumSHA256   string   `json:"checksum_sha256"`
	Status           string   `json:"status"` // CREATED, VERIFIED, RESTORE_TESTED, CORRUPT, EXPIRED
	LastRestoreTest  string   `json:"last_restore_test,omitempty"`
	RestoreVerified  bool     `json:"restore_verified"`
	CertifiedBy      string   `json:"certified_by,omitempty"`
	CreatedAt        string   `json:"created_at"`
}

type RestoreTestRecord struct {
	ID                string `json:"id"`
	BackupID          string `json:"backup_id"`
	ServiceID         string `json:"service_id"`
	TestMode          string `json:"test_mode"` // ISOLATED_SANDBOX, STAGING_VERIFICATION, SHADOW_RESTORATION
	Status            string `json:"status"`    // PASSED, INTEGRITY_FAILED, SERVICE_FAILED
	DurationSec       int    `json:"duration_sec"`
	RecordsRestored   int64  `json:"records_restored"`
	ChecksumMatch     bool   `json:"checksum_match"`
	ServiceProbePass  bool   `json:"service_probe_pass"`
	EvidenceID        string `json:"evidence_id,omitempty"`
	ConductedBy       string `json:"conducted_by"`
	Notes             string `json:"notes"`
	CreatedAt         string `json:"created_at"`
}

// ─── Data Integrity ───────────────────────────────────────────────

type IntegrityCheck struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	QueryAssertion string `json:"query_assertion"`
	Severity       string `json:"severity"`
	Enabled        bool   `json:"enabled"`
	CreatedAt      string `json:"created_at"`
}

type IntegrityResult struct {
	ID             string                 `json:"id"`
	CheckID        string                 `json:"check_id"`
	CheckName      string                 `json:"check_name"`
	Category       string                 `json:"category"`
	Status         string                 `json:"status"` // PASSED, WARNING, FAILED
	AnomaliesCount int                    `json:"anomalies_count"`
	Details        map[string]interface{} `json:"details"`
	DurationMs     int                    `json:"duration_ms"`
	RunBy          string                 `json:"run_by"`
	CreatedAt      string                 `json:"created_at"`
}

// ─── Incident Lifecycle ───────────────────────────────────────────

type IncidentSeverity string

const (
	SeverityInfo         IncidentSeverity = "INFO"
	SeverityWarning      IncidentSeverity = "WARNING"
	SeverityHigh         IncidentSeverity = "HIGH"
	SeverityCritical     IncidentSeverity = "CRITICAL"
	SeverityCatastrophic IncidentSeverity = "CATASTROPHIC"
)

type IncidentStatus string

const (
	IncDetected     IncidentStatus = "DETECTED"
	IncTriaged      IncidentStatus = "TRIAGED"
	IncAcknowledged IncidentStatus = "ACKNOWLEDGED"
	IncMitigating   IncidentStatus = "MITIGATING"
	IncRecovering   IncidentStatus = "RECOVERING"
	IncValidating   IncidentStatus = "VALIDATING"
	IncResolved     IncidentStatus = "RESOLVED"
	IncClosed       IncidentStatus = "CLOSED"
)

type Incident struct {
	ID                 string           `json:"id"`
	Title              string           `json:"title"`
	Severity           IncidentSeverity `json:"severity"`
	Status             IncidentStatus   `json:"status"`
	ServiceID          string           `json:"service_id"`
	ServiceName        string           `json:"service_name,omitempty"`
	DetectionSource    string           `json:"detection_source"`
	RootCauseSummary   string           `json:"root_cause_summary"`
	AffectedComponents []string         `json:"affected_components"`
	CorrelationID      string           `json:"correlation_id"`
	DetectedAt         string           `json:"detected_at"`
	AcknowledgedAt     string           `json:"acknowledged_at,omitempty"`
	AcknowledgedBy     string           `json:"acknowledged_by,omitempty"`
	ResolvedAt         string           `json:"resolved_at,omitempty"`
	ResolvedBy         string           `json:"resolved_by,omitempty"`
	ClosedAt           string           `json:"closed_at,omitempty"`
	ClosedBy           string           `json:"closed_by,omitempty"`
	RTOImpactSec       int              `json:"rto_impact_sec"`
	RPOImpactSec       int              `json:"rpo_impact_sec"`
	EvidenceID         string           `json:"evidence_id,omitempty"`
	Events             []IncidentEvent  `json:"events,omitempty"`
	Actions            []IncidentAction `json:"actions,omitempty"`
	CreatedAt          string           `json:"created_at"`
	UpdatedAt          string           `json:"updated_at"`
}

type IncidentEvent struct {
	ID         int64                  `json:"id"`
	IncidentID string                 `json:"incident_id"`
	FromStatus string                 `json:"from_status"`
	ToStatus   string                 `json:"to_status"`
	Actor      string                 `json:"actor"`
	Reason     string                 `json:"reason"`
	Evidence   map[string]interface{} `json:"evidence,omitempty"`
	CreatedAt  string                 `json:"created_at"`
}

type IncidentAction struct {
	ID          string `json:"id"`
	IncidentID  string `json:"incident_id"`
	ActionType  string `json:"action_type"`
	Description string `json:"description"`
	Status      string `json:"status"`
	ExecutedBy  string `json:"executed_by"`
	Output      string `json:"output"`
	CreatedAt   string `json:"created_at"`
}

// ─── Runbooks ─────────────────────────────────────────────────────

type Runbook struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	IncidentType      string   `json:"incident_type"`
	AffectedServices  []string `json:"affected_services"`
	DetectionMethod   string   `json:"detection_method"`
	ImmediateActions  []string `json:"immediate_actions"`
	RecoverySteps     []string `json:"recovery_steps"`
	ValidationChecks  []string `json:"validation_checks"`
	RollbackProcedure string   `json:"rollback_procedure"`
	EscalationPath    string   `json:"escalation_path"`
	EvidenceRequired  string   `json:"evidence_required"`
	ClosureCriteria   string   `json:"closure_criteria"`
	Version           int      `json:"version"`
	Enabled           bool     `json:"enabled"`
	CreatedAt         string   `json:"created_at"`
}

type RunbookExecution struct {
	ID           string                   `json:"id"`
	RunbookID    string                   `json:"runbook_id"`
	RunbookTitle string                   `json:"runbook_title,omitempty"`
	IncidentID   string                   `json:"incident_id,omitempty"`
	ExecutedBy   string                   `json:"executed_by"`
	Status       string                   `json:"status"` // IN_PROGRESS, COMPLETED, FAILED, ABORTED
	StepResults  []map[string]interface{} `json:"step_results"`
	StartedAt    string                   `json:"started_at"`
	CompletedAt  string                   `json:"completed_at,omitempty"`
	EvidenceID   string                   `json:"evidence_id,omitempty"`
}

// ─── Resilience Evidence ──────────────────────────────────────────

type ResilienceEvidence struct {
	ID               string                 `json:"id"`
	ActivityType     string                 `json:"activity_type"` // RECOVERY_DRILL, BACKUP_RESTORE_TEST, INCIDENT_RECOVERY, EVENT_REPLAY, INTEGRITY_AUDIT, SECURITY_INVESTIGATION
	TargetService    string                 `json:"target_service"`
	Actor            string                 `json:"actor"`
	ResultStatus     string                 `json:"result_status"` // VERIFIED_SUCCESS, COMPLIANT_WITH_WARNINGS, BREACHED, FAILED
	Metrics          map[string]interface{} `json:"metrics"`
	LogsSummary      string                 `json:"logs_summary"`
	ChecksumDigest   string                 `json:"checksum_digest"`
	AuditReference   string                 `json:"audit_reference"`
	VerificationHash string                 `json:"verification_hash"`
	CreatedAt        string                 `json:"created_at"`
}

// ─── Composite Resilience Score ───────────────────────────────────

type ResilienceScoreOverview struct {
	CompositeScore       int                    `json:"composite_score"` // 0 - 100
	OverallAvailability  float64                `json:"overall_availability"`
	AvailabilityScore    int                    `json:"availability_score"`
	RTOScore             int                    `json:"rto_score"`
	RPOScore             int                    `json:"rpo_score"`
	BackupScore          int                    `json:"backup_score"`
	IntegrityScore       int                    `json:"integrity_score"`
	EventScore           int                    `json:"event_score"`
	IncidentScore        int                    `json:"incident_score"`
	ActiveIncidents      int                    `json:"active_incidents"`
	TotalServices        int                    `json:"total_services"`
	CompliantServices    int                    `json:"compliant_services"`
	AtRiskServices       int                    `json:"at_risk_services"`
	BreachedServices     int                    `json:"breached_services"`
	UntestedServices     int                    `json:"untested_services"`
	LastEvaluated        string                 `json:"last_evaluated"`
	Factors              map[string]interface{} `json:"factors"`
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
