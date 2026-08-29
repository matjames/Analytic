package store

import (
	"context"

	"statdata-backend/internal/models"
)

// Store specifies the comprehensive data access contract for StatData
type Store interface {
	// ─── Data Catalog & Sources (P37) ──────────────────────────────────────────
	CreateDataset(ctx context.Context, ds *models.Dataset) error
	GetDatasetByID(ctx context.Context, id string) (*models.Dataset, error)
	GetDatasetByURN(ctx context.Context, urn string) (*models.Dataset, error)
	ListDatasets(ctx context.Context, tenantID, domain, classification, workspaceID string, limit, offset int) ([]*models.Dataset, int64, error)
	UpdateDataset(ctx context.Context, ds *models.Dataset) error
	DeleteDataset(ctx context.Context, id string) error

	CreateDataSource(ctx context.Context, src *models.DataSource) error
	GetDataSourceByID(ctx context.Context, id string) (*models.DataSource, error)
	ListDataSources(ctx context.Context, tenantID, sourceType, workspaceID string) ([]*models.DataSource, error)
	UpdateDataSource(ctx context.Context, src *models.DataSource) error

	// ─── Schema Registry (P37) ────────────────────────────────────────────────
	RegisterSchema(ctx context.Context, schema *models.SchemaDefinition) error
	GetSchemaByID(ctx context.Context, id string) (*models.SchemaDefinition, error)
	GetLatestSchemaBySubject(ctx context.Context, subject, tenantID string) (*models.SchemaDefinition, error)
	ListSchemas(ctx context.Context, tenantID string) ([]*models.SchemaDefinition, error)

	// ─── Data Contracts (P37) ─────────────────────────────────────────────────
	CreateDataContract(ctx context.Context, contract *models.DataContract) error
	GetDataContractByID(ctx context.Context, id string) (*models.DataContract, error)
	ListDataContracts(ctx context.Context, tenantID, datasetID string) ([]*models.DataContract, error)
	UpdateDataContract(ctx context.Context, contract *models.DataContract) error

	// ─── Data Quality & Governance (P37) ───────────────────────────────────────
	CreateQualityRule(ctx context.Context, rule *models.DataQualityRule) error
	ListQualityRules(ctx context.Context, tenantID, datasetID string) ([]*models.DataQualityRule, error)
	SaveQualityReport(ctx context.Context, report *models.DataQualityReport) error
	GetLatestQualityReport(ctx context.Context, datasetID string) (*models.DataQualityReport, error)
	ListQualityReports(ctx context.Context, tenantID, datasetID string, limit int) ([]*models.DataQualityReport, error)

	// ─── Data Lineage (P37) ───────────────────────────────────────────────────
	RecordLineageNode(ctx context.Context, node *models.LineageNode) error
	RecordLineageEdge(ctx context.Context, edge *models.LineageEdge) error
	GetLineageGraph(ctx context.Context, rootID, tenantID string, depth int) (*models.LineageGraph, error)

	// ─── Data Pipelines & Streaming (P37) ─────────────────────────────────────
	CreatePipeline(ctx context.Context, p *models.DataPipeline) error
	GetPipelineByID(ctx context.Context, id string) (*models.DataPipeline, error)
	ListPipelines(ctx context.Context, tenantID, status, workspaceID string) ([]*models.DataPipeline, error)
	UpdatePipeline(ctx context.Context, p *models.DataPipeline) error
	DeletePipeline(ctx context.Context, id string) error

	RecordPipelineRun(ctx context.Context, run *models.PipelineRun) error
	GetPipelineRunByID(ctx context.Context, id string) (*models.PipelineRun, error)
	ListPipelineRuns(ctx context.Context, pipelineID, tenantID string, limit int) ([]*models.PipelineRun, error)
	UpdatePipelineRun(ctx context.Context, run *models.PipelineRun) error

	CreateStreamingJob(ctx context.Context, job *models.StreamingJob) error
	GetStreamingJobByID(ctx context.Context, id string) (*models.StreamingJob, error)
	ListStreamingJobs(ctx context.Context, tenantID, status, workspaceID string) ([]*models.StreamingJob, error)
	UpdateStreamingJob(ctx context.Context, job *models.StreamingJob) error

	// ─── Feature Store (P37) ──────────────────────────────────────────────────
	CreateFeatureView(ctx context.Context, fv *models.FeatureView) error
	GetFeatureViewByID(ctx context.Context, id string) (*models.FeatureView, error)
	ListFeatureViews(ctx context.Context, tenantID, entityName string) ([]*models.FeatureView, error)
	SaveFeatureRecords(ctx context.Context, records []models.FeatureRecord) error
	GetOnlineFeatures(ctx context.Context, featureViewID, entityKey, tenantID string) (*models.FeatureVector, error)

	// ─── Scientific & Statistical Computing (P38) ─────────────────────────────
	CreateNotebookSession(ctx context.Context, nb *models.NotebookSession) error
	GetNotebookSessionByID(ctx context.Context, id string) (*models.NotebookSession, error)
	ListNotebookSessions(ctx context.Context, tenantID, language string) ([]*models.NotebookSession, error)
	UpdateNotebookSession(ctx context.Context, nb *models.NotebookSession) error

	CreateExperiment(ctx context.Context, exp *models.Experiment) error
	GetExperimentByID(ctx context.Context, id string) (*models.Experiment, error)
	ListExperiments(ctx context.Context, tenantID, domain string) ([]*models.Experiment, error)

	RecordExperimentRun(ctx context.Context, run *models.ExperimentRun) error
	GetExperimentRunByID(ctx context.Context, id string) (*models.ExperimentRun, error)
	ListExperimentRuns(ctx context.Context, experimentID, tenantID string) ([]*models.ExperimentRun, error)

	CreateRegisteredModel(ctx context.Context, rm *models.RegisteredModel) error
	GetRegisteredModelByID(ctx context.Context, id string) (*models.RegisteredModel, error)
	ListRegisteredModels(ctx context.Context, tenantID, domain string) ([]*models.RegisteredModel, error)
	CreateModelVersion(ctx context.Context, mv *models.ModelVersion) error
	ListModelVersions(ctx context.Context, modelID, tenantID string) ([]*models.ModelVersion, error)
	UpdateModelStage(ctx context.Context, versionID string, stage models.ModelStage) error

	RegisterComputeNode(ctx context.Context, node *models.ComputeNode) error
	ListComputeNodes(ctx context.Context, tenantID string) ([]*models.ComputeNode, error)
	CreateComputeJob(ctx context.Context, job *models.ComputeJob) error
	GetComputeJobByID(ctx context.Context, id string) (*models.ComputeJob, error)
	ListComputeJobs(ctx context.Context, tenantID, status string) ([]*models.ComputeJob, error)
	UpdateComputeJob(ctx context.Context, job *models.ComputeJob) error

	// ─── Enterprise Search & Retrieval (P47) ──────────────────────────────────
	CreateSearchIndex(ctx context.Context, idx *models.SearchIndex) error
	GetSearchIndex(ctx context.Context, indexName, tenantID string) (*models.SearchIndex, error)
	IndexDocument(ctx context.Context, doc *models.IndexedDocument) error
	DeleteIndexedDocument(ctx context.Context, indexName, documentID string) error
	GetIndexedDocument(ctx context.Context, indexName, documentID string) (*models.IndexedDocument, error)
	Search(ctx context.Context, req *models.HybridSearchRequest) (*models.SearchResponse, error)
	GetSuggestions(ctx context.Context, prefix, tenantID string, limit int) ([]string, error)

	CreateSavedSearch(ctx context.Context, ss *models.SavedSearch) error
	ListSavedSearches(ctx context.Context, tenantID, createdBy string) ([]*models.SavedSearch, error)

	// ─── Compliance & Cross-App Primitives ────────────────────────────────────
	LogAuditEvent(ctx context.Context, entry *models.AuditLog) error
	ListAuditLogs(ctx context.Context, tenantID, resourceType string, limit int) ([]*models.AuditLog, error)
	CreateObjectLink(ctx context.Context, link *models.ObjectLink) error
	GetObjectLinks(ctx context.Context, sourceType, sourceID, tenantID string) ([]*models.ObjectLink, error)
}
