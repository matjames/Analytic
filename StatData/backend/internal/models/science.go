package models

import "time"

// ModelStage defines lifecycle stage of an ML/statistical model
type ModelStage string

const (
	ModelStageNone       ModelStage = "NONE"
	ModelStageStaging    ModelStage = "STAGING"
	ModelStageProduction ModelStage = "PRODUCTION"
	ModelStageArchived   ModelStage = "ARCHIVED"
)

// NotebookSession represents an active or saved scientific notebook workspace
type NotebookSession struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Language    string                 `json:"language"` // PYTHON, R, JULIA, SQL
	KernelState string                 `json:"kernel_state"` // IDLE, BUSY, STOPPED, DEAD
	DatasetRefs []string               `json:"dataset_refs,omitempty"`
	Cells       []NotebookCell         `json:"cells"`
	Variables   map[string]string      `json:"variables,omitempty"`
	TenantID    string                 `json:"tenant_id"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// NotebookCell represents a unit of computation in a scientific notebook
type NotebookCell struct {
	ID           string                 `json:"id"`
	CellType     string                 `json:"cell_type"` // CODE, MARKDOWN, RAW
	Source       string                 `json:"source"`
	Output       string                 `json:"output,omitempty"`
	RichOutput   map[string]interface{} `json:"rich_output,omitempty"` // charts, tables, HTML
	ExecutionCount int                  `json:"execution_count"`
	Status       string                 `json:"status"` // IDLE, RUNNING, SUCCESS, ERROR
	ExecutionTimeMs int64               `json:"execution_time_ms"`
}

// NotebookCellExecution represents an incoming code execution request
type NotebookCellExecution struct {
	SessionID string `json:"session_id"`
	CellID    string `json:"cell_id"`
	Source    string `json:"source"`
	Language  string `json:"language"`
}

// Experiment represents a trackable scientific or ML experiment
type Experiment struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Domain      string                 `json:"domain"` // EPIDEMIOLOGY, ECONOMETRICS, DEMOGRAPHICS, LLM_FINE_TUNING
	Tags        []string               `json:"tags,omitempty"`
	ArtifactURI string                 `json:"artifact_uri,omitempty"`
	TenantID    string                 `json:"tenant_id"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ExperimentRun represents a single trial / training run
type ExperimentRun struct {
	ID           string                 `json:"id"`
	ExperimentID string                 `json:"experiment_id"`
	RunName      string                 `json:"run_name"`
	Status       string                 `json:"status"` // RUNNING, COMPLETED, FAILED
	Parameters   map[string]interface{} `json:"parameters"`
	Metrics      map[string]float64     `json:"metrics"`
	Tags         map[string]string      `json:"tags,omitempty"`
	Artifacts    []ModelArtifact        `json:"artifacts,omitempty"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	DurationMs   int64                  `json:"duration_ms"`
	TenantID     string                 `json:"tenant_id"`
	CreatedBy    string                 `json:"created_by"`
}

// ModelArtifact represents files produced by an experiment run
type ModelArtifact struct {
	Name        string `json:"name"`
	ArtifactURI string `json:"artifact_uri"`
	SizeBytes   int64  `json:"size_bytes"`
	Checksum    string `json:"checksum"`
}

// RegisteredModel represents a centralized ML/Statistical model in the model registry
type RegisteredModel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Domain      string                 `json:"domain"`
	Framework   string                 `json:"framework"` // PYTORCH, TENSORFLOW, SCIKIT_LEARN, STAN, R_STAT
	LatestStage ModelStage             `json:"latest_stage"`
	Versions    []ModelVersion         `json:"versions,omitempty"`
	TenantID    string                 `json:"tenant_id"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ModelVersion represents an immutable tagged model release
type ModelVersion struct {
	ID             string                 `json:"id"`
	ModelID        string                 `json:"model_id"`
	Version        int                    `json:"version"`
	Stage          ModelStage             `json:"stage"` // NONE, STAGING, PRODUCTION, ARCHIVED
	SourceRunID    string                 `json:"source_run_id,omitempty"`
	ArtifactURI    string                 `json:"artifact_uri"`
	MetricsSummary map[string]float64     `json:"metrics_summary,omitempty"`
	InputSchema    string                 `json:"input_schema,omitempty"`
	OutputSchema   string                 `json:"output_schema,omitempty"`
	Description    string                 `json:"description,omitempty"`
	TenantID       string                 `json:"tenant_id"`
	CreatedBy      string                 `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ComputeNode represents a cluster node available for scientific workloads
type ComputeNode struct {
	ID          string    `json:"id"`
	Hostname    string    `json:"hostname"`
	IPAddress   string    `json:"ip_address"`
	TotalCPUs   int       `json:"total_cpus"`
	AllocCPUs   int       `json:"alloc_cpus"`
	TotalRAMGB  float64   `json:"total_ram_gb"`
	AllocRAMGB  float64   `json:"alloc_ram_gb"`
	TotalGPUs   int       `json:"total_gpus"`
	AllocGPUs   int       `json:"alloc_gpus"`
	Status      string    `json:"status"` // READY, BUSY, OFFLINE, MAINTENANCE
	TenantID    string    `json:"tenant_id"`
	LastPing    time.Time `json:"last_ping"`
}

// ComputeJob represents a scheduled computational task dispatched to the cluster
type ComputeJob struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	JobType     string                 `json:"job_type"` // MONTE_CARLO, BAYESIAN_INFERENCE, EPIDEMIOLOGICAL_SIM, MODEL_TRAINING
	AssignedNode string                `json:"assigned_node,omitempty"`
	RequiredCPUs int                   `json:"required_cpus"`
	RequiredRAMGB float64              `json:"required_ram_gb"`
	RequiredGPUs int                   `json:"required_gpus"`
	Status      string                 `json:"status"` // QUEUED, RUNNING, COMPLETED, FAILED
	Params      map[string]interface{} `json:"params,omitempty"`
	OutputData  map[string]interface{} `json:"output_data,omitempty"`
	TenantID    string                 `json:"tenant_id"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}
