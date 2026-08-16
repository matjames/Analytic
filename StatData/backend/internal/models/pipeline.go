package models

import "time"

// PipelineType defines processing paradigm
type PipelineType string

const (
	PipelineTypeETL       PipelineType = "ETL"
	PipelineTypeELT       PipelineType = "ELT"
	PipelineTypeStreaming PipelineType = "STREAMING"
	PipelineTypeReverseETL PipelineType = "REVERSE_ETL"
	PipelineTypeCDC       PipelineType = "CDC"
)

// PipelineStatus defines lifecycle status
type PipelineStatus string

const (
	PipelineStatusActive   PipelineStatus = "ACTIVE"
	PipelineStatusPaused   PipelineStatus = "PAUSED"
	PipelineStatusFailed   PipelineStatus = "FAILED"
	PipelineStatusArchived PipelineStatus = "ARCHIVED"
)

// RunStatus defines execution state
type RunStatus string

const (
	RunStatusPending   RunStatus = "PENDING"
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusSuccess   RunStatus = "SUCCESS"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusCancelled RunStatus = "CANCELLED"
)

// DataPipeline represents a workflow definition for data processing
type DataPipeline struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	PipelineType   PipelineType           `json:"pipeline_type"`
	Status         PipelineStatus         `json:"status"`
	CronSchedule   string                 `json:"cron_schedule,omitempty"`
	SourceDatasetID string                `json:"source_dataset_id,omitempty"`
	TargetDatasetID string                `json:"target_dataset_id,omitempty"`
	Stages         []PipelineStage        `json:"stages"`
	Config         map[string]interface{} `json:"config,omitempty"`
	MaxRetries     int                    `json:"max_retries"`
	TimeoutSeconds int                    `json:"timeout_seconds"`
	TenantID       string                 `json:"tenant_id"`
	CreatedBy      string                 `json:"created_by"`
	LastRunAt      *time.Time             `json:"last_run_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// PipelineStage defines an atomic transformation or validation step in a pipeline DAG
type PipelineStage struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	StageType      string                 `json:"stage_type"` // EXTRACT, TRANSFORM, LOAD, VALIDATE, ENRICH, ANONYMIZE
	DependsOn      []string               `json:"depends_on,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
	Script         string                 `json:"script,omitempty"`
}

// PipelineRun records an execution instance of a data pipeline
type PipelineRun struct {
	ID              string                 `json:"id"`
	PipelineID      string                 `json:"pipeline_id"`
	PipelineName    string                 `json:"pipeline_name"`
	Status          RunStatus              `json:"status"`
	TriggerType     string                 `json:"trigger_type"` // MANUAL, SCHEDULED, EVENT, CDC
	StartedAt       time.Time              `json:"started_at"`
	FinishedAt      *time.Time             `json:"finished_at,omitempty"`
	DurationMs      int64                  `json:"duration_ms"`
	RecordsRead     int64                  `json:"records_read"`
	RecordsWritten  int64                  `json:"records_written"`
	RecordsRejected int64                  `json:"records_rejected"`
	StageRuns       []StageRun             `json:"stage_runs,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	Metrics         map[string]interface{} `json:"metrics,omitempty"`
	TenantID        string                 `json:"tenant_id"`
	TriggeredBy     string                 `json:"triggered_by"`
}

// StageRun captures state of an individual stage execution
type StageRun struct {
	StageID        string                 `json:"stage_id"`
	StageName      string                 `json:"stage_name"`
	Status         RunStatus              `json:"status"`
	DurationMs     int64                  `json:"duration_ms"`
	RecordsProcessed int64                `json:"records_processed"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	OutputSummary  map[string]interface{} `json:"output_summary,omitempty"`
}

// StreamingJob represents a continuous real-time streaming data ingestion job
type StreamingJob struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	SourceTopic     string                 `json:"source_topic"`
	TargetSink      string                 `json:"target_sink"`
	Status          string                 `json:"status"` // RUNNING, STOPPED, FAILED
	ThroughputMsgSec float64               `json:"throughput_msg_sec"`
	LagRecords      int64                  `json:"lag_records"`
	Config          map[string]interface{} `json:"config,omitempty"`
	TenantID        string                 `json:"tenant_id"`
	LastCheckpoint  time.Time              `json:"last_checkpoint"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// CDCEvent captures a Change Data Capture stream mutation event
type CDCEvent struct {
	EventID        string                 `json:"event_id"`
	Table          string                 `json:"table"`
	Schema         string                 `json:"schema"`
	Operation      string                 `json:"operation"` // INSERT, UPDATE, DELETE
	Before         map[string]interface{} `json:"before,omitempty"`
	After          map[string]interface{} `json:"after,omitempty"`
	SequenceOffset int64                  `json:"sequence_offset"`
	Timestamp      time.Time              `json:"timestamp"`
	TenantID       string                 `json:"tenant_id"`
}
