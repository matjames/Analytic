package main

import "time"

// ============================================================================
// P20: DEVOPS, CI/CD & CONTINUOUS DELIVERY
// ============================================================================

type PipelineRun struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	WorkspaceID  string          `json:"workspace_id,omitempty"`
	RepoName     string          `json:"repo_name"`
	Branch       string          `json:"branch"`
	CommitSHA    string          `json:"commit_sha"`
	CommitMsg    string          `json:"commit_msg"`
	Author       string          `json:"author"`
	Status       string          `json:"status"` // PENDING, RUNNING, SUCCESS, FAILED, CANCELLED
	Stages       []PipelineStage `json:"stages"`
	StartedAt    time.Time       `json:"started_at"`
	FinishedAt   *time.Time      `json:"finished_at,omitempty"`
	DurationSecs int             `json:"duration_secs"`
	ArtifactURL  string          `json:"artifact_url,omitempty"`
}

type PipelineStage struct {
	Name         string     `json:"name"`   // Lint & Security, Unit Tests, Container Build, Integration Tests, Artifact Push
	Status       string     `json:"status"` // PENDING, RUNNING, SUCCESS, FAILED, SKIPPED
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	DurationSecs int        `json:"duration_secs"`
	Logs         []string   `json:"logs"`
}

type DeploymentRecord struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	WorkspaceID      string    `json:"workspace_id,omitempty"`
	ServiceName      string    `json:"service_name"`
	Version          string    `json:"version"`
	Environment      string    `json:"environment"` // DEVELOPMENT, STAGING, PRODUCTION_SOVEREIGN, PRODUCTION_CLOUD
	Strategy         string    `json:"strategy"`    // ROLLING, BLUE_GREEN, CANARY
	TargetCluster    string    `json:"target_cluster"`
	Status           string    `json:"status"` // DEPLOYING, ACTIVE, ROLLBACK_IN_PROGRESS, ROLLED_BACK, FAILED
	CanaryPercentage int       `json:"canary_percentage"`
	DeployedBy       string    `json:"deployed_by"`
	DeployedAt       time.Time `json:"deployed_at"`
	RollbackVersion  string    `json:"rollback_version,omitempty"`
}

type ReleaseTrain struct {
	ID            string    `json:"id"`
	ReleaseTag    string    `json:"release_tag"`
	Title         string    `json:"title"`
	IncludedApps  []string  `json:"included_apps"`
	TargetDate    time.Time `json:"target_date"`
	ApprovalState string    `json:"approval_state"` // DRAFT, IN_REVIEW, APPROVED, RELEASED
	SignoffLead   string    `json:"signoff_lead"`
	Changelog     []string  `json:"changelog"`
}

type EnvironmentConfig struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenant_id"`
	EnvName    string            `json:"env_name"` // dev, staging, prod-gov, prod-cloud
	ClusterURL string            `json:"cluster_url"`
	Variables  map[string]string `json:"variables"`
	SecretRefs []string          `json:"secret_refs"`
	UpdatedAt  time.Time         `json:"updated_at"`
	UpdatedBy  string            `json:"updated_by"`
}

// ============================================================================
// P25: SOVEREIGN & MULTI-CLOUD INFRASTRUCTURE
// ============================================================================

type CloudCluster struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Provider       string    `json:"provider"`      // NITA_U_GOVCLOUD, ON_PREM_DC, AWS_AFRICA_SOUTH, AZURE_SOUTH_AFRICA, HYBRID
	Region         string    `json:"region"`        // kampala-central, entebbe-dr, af-south-1, southafricanorth
	Status         string    `json:"status"`        // HEALTHY, DEGRADED, PROVISIONING, OFFLINE
	DataBoundary   string    `json:"data_boundary"` // NATIONAL_SOVEREIGN, REGIONAL_RESTRICTED, OPEN_FEDERATION
	TotalNodes     int       `json:"total_nodes"`
	AllocatedCPU   string    `json:"allocated_cpu"`
	AllocatedRAM   string    `json:"allocated_ram"`
	StorageUsageGB float64   `json:"storage_usage_gb"`
	K8sVersion     string    `json:"k8s_version"`
	LastHeartbeat  time.Time `json:"last_heartbeat"`
}

type CloudCostRecord struct {
	ID              string  `json:"id"`
	MonthYear       string  `json:"month_year"` // 2026-08
	Provider        string  `json:"provider"`
	ServiceName     string  `json:"service_name"`
	CostUSD         float64 `json:"cost_usd"`
	BudgetLimitUSD  float64 `json:"budget_limit_usd"`
	OptimizationTip string  `json:"optimization_tip"`
}

// ============================================================================
// P34: PRODUCTION READINESS & SRE AUTOMATION
// ============================================================================

type ProductionReadinessCheck struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"` // INFRASTRUCTURE, SECURITY, PERFORMANCE, BACKUP_DR, DOCUMENTATION
	CheckName   string    `json:"check_name"`
	Status      string    `json:"status"` // PASSED, FAILED, WARNING, PENDING
	Details     string    `json:"details"`
	Auditor     string    `json:"auditor"`
	LastAuditAt time.Time `json:"last_audit_at"`
}

type SLOBudget struct {
	ID                   string    `json:"id"`
	ServiceName          string    `json:"service_name"`
	SLOName              string    `json:"slo_name"`       // Availability, Latency_P99, Error_Rate
	TargetPercent        float64   `json:"target_percent"` // 99.95
	CurrentPercent       float64   `json:"current_percent"`
	ErrorBudgetRemaining float64   `json:"error_budget_remaining"` // %
	BurnRateStatus       string    `json:"burn_rate_status"`       // NORMAL, ELEVATED, CRITICAL
	Period               string    `json:"period"`                 // 30d
	LastCalculated       time.Time `json:"last_calculated"`
}

type RunbookExecution struct {
	ID            string                 `json:"id"`
	RunbookName   string                 `json:"runbook_name"`
	TriggerReason string                 `json:"trigger_reason"`
	TargetService string                 `json:"target_service"`
	TriggeredBy   string                 `json:"triggered_by"`
	Status        string                 `json:"status"` // RUNNING, SUCCESS, FAILED
	StepsExecuted []string               `json:"steps_executed"`
	OutputLogs    string                 `json:"output_logs"`
	ExecutedAt    time.Time              `json:"executed_at"`
	Parameters    map[string]interface{} `json:"parameters"`
}

type DisasterRecoveryDrill struct {
	ID             string    `json:"id"`
	DrillName      string    `json:"drill_name"`
	PrimarySite    string    `json:"primary_site"`
	DRSite         string    `json:"dr_site"`
	RTOAchievedMin float64   `json:"rto_achieved_min"`
	RPOAchievedSec float64   `json:"rpo_achieved_sec"`
	Status         string    `json:"status"` // PASSED, PARTIAL, FAILED
	ConductedAt    time.Time `json:"conducted_at"`
	ConductedBy    string    `json:"conducted_by"`
	Notes          string    `json:"notes"`
}

// ============================================================================
// P49: INTEGRATED OPERATIONS CENTER (IOC) & OBSERVABILITY
// ============================================================================

type ServiceTelemetry struct {
	AppName        string             `json:"app_name"`
	Port           int                `json:"port"`
	Endpoint       string             `json:"endpoint"`
	Status         string             `json:"status"` // UP, DOWN, DEGRADED
	LatencyMs      int                `json:"latency_ms"`
	CPUUsagePct    float64            `json:"cpu_usage_pct"`
	MemoryUsageMB  float64            `json:"memory_usage_mb"`
	UptimeSecs     int64              `json:"uptime_secs"`
	ActiveRequests int                `json:"active_requests"`
	ErrorRate5xx   float64            `json:"error_rate_5xx"`
	LastScrapedAt  time.Time          `json:"last_scraped_at"`
	CustomMetrics  map[string]float64 `json:"custom_metrics,omitempty"`
}

type CentralizedLog struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"` // INFO, WARN, ERROR, FATAL, DEBUG
	SourceApp string                 `json:"source_app"`
	Message   string                 `json:"message"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type DistributedTraceSpan struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	ServiceName  string            `json:"service_name"`
	Operation    string            `json:"operation"`
	DurationMs   int               `json:"duration_ms"`
	StatusCode   int               `json:"status_code"`
	StartTime    time.Time         `json:"start_time"`
	Tags         map[string]string `json:"tags"`
}

type CMDBItem struct {
	ID           string    `json:"id"`
	ItemName     string    `json:"item_name"`
	ItemType     string    `json:"item_type"` // MICROSERVICE, DATABASE, REDIS_CLUSTER, LOAD_BALANCER, K8S_INGRESS, VM
	Environment  string    `json:"environment"`
	HostOrURL    string    `json:"host_or_url"`
	OwnerTeam    string    `json:"owner_team"`
	Criticality  string    `json:"criticality"` // TIER_0_MISSION_CRITICAL, TIER_1_HIGH, TIER_2_MEDIUM
	Dependencies []string  `json:"dependencies"`
	Status       string    `json:"status"` // OPERATIONAL, MAINTENANCE, DECOMMISSIONED
	LastUpdated  time.Time `json:"last_updated"`
}

type AlertIncident struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Severity       string     `json:"severity"` // P1_CRITICAL, P2_HIGH, P3_MEDIUM, P4_LOW
	Status         string     `json:"status"`   // FIRING, ACKNOWLEDGED, RESOLVED, SUPPRESSED
	SourceApp      string     `json:"source_app"`
	Description    string     `json:"description"`
	TriggeredAt    time.Time  `json:"triggered_at"`
	AcknowledgedBy string     `json:"acknowledged_by,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	RunbookLink    string     `json:"runbook_link,omitempty"`
}

type PublicStatusComponent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Group     string    `json:"group"`  // Core Services, Data Collection, Intelligence & GIS, Trust & Governance
	Status    string    `json:"status"` // OPERATIONAL, DEGRADED_PERFORMANCE, PARTIAL_OUTAGE, MAJOR_OUTAGE
	Uptime90d float64   `json:"uptime_90d"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OperationsOverviewSummary struct {
	PlatformHealthScore  float64 `json:"platform_health_score"` // 99.8%
	ActiveServicesCount  int     `json:"active_services_count"`
	TotalClusters        int     `json:"total_clusters"`
	FiringAlertsCount    int     `json:"firing_alerts_count"`
	RunningPipelines     int     `json:"running_pipelines"`
	AvgResponseLatencyMs int     `json:"avg_response_latency_ms"`
	MonthlyCloudSpendUSD float64 `json:"monthly_cloud_spend_usd"`
	ReadinessPassRate    float64 `json:"readiness_pass_rate"`
}
