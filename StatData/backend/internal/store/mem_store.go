package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"statdata-backend/internal/models"
)

// MemStore provides a high-fidelity in-memory implementation of Store
type MemStore struct {
	mu           sync.RWMutex
	datasets     map[string]*models.Dataset
	dataSources  map[string]*models.DataSource
	schemas      map[string]*models.SchemaDefinition
	contracts    map[string]*models.DataContract
	qualityRules map[string]*models.DataQualityRule
	qualityReps  map[string]*models.DataQualityReport
	lineageNodes map[string]*models.LineageNode
	lineageEdges map[string]*models.LineageEdge
	pipelines    map[string]*models.DataPipeline
	pipelineRuns map[string]*models.PipelineRun
	streamingJobs map[string]*models.StreamingJob
	featureViews map[string]*models.FeatureView
	features     map[string]map[string]*models.FeatureRecord // featureViewID -> entityKey -> record
	notebooks    map[string]*models.NotebookSession
	experiments  map[string]*models.Experiment
	expRuns      map[string]*models.ExperimentRun
	models       map[string]*models.RegisteredModel
	modelVers    map[string]*models.ModelVersion
	computeNodes map[string]*models.ComputeNode
	computeJobs  map[string]*models.ComputeJob
	searchIdxs   map[string]*models.SearchIndex
	indexedDocs  map[string]*models.IndexedDocument // docID -> doc
	savedSearches map[string]*models.SavedSearch
	auditLogs    []*models.AuditLog
	objectLinks  []*models.ObjectLink
}

// NewMemStore instantiates MemStore and seeds default baseline records
func NewMemStore() *MemStore {
	m := &MemStore{
		datasets:     make(map[string]*models.Dataset),
		dataSources:  make(map[string]*models.DataSource),
		schemas:      make(map[string]*models.SchemaDefinition),
		contracts:    make(map[string]*models.DataContract),
		qualityRules: make(map[string]*models.DataQualityRule),
		qualityReps:  make(map[string]*models.DataQualityReport),
		lineageNodes: make(map[string]*models.LineageNode),
		lineageEdges: make(map[string]*models.LineageEdge),
		pipelines:    make(map[string]*models.DataPipeline),
		pipelineRuns: make(map[string]*models.PipelineRun),
		streamingJobs: make(map[string]*models.StreamingJob),
		featureViews: make(map[string]*models.FeatureView),
		features:     make(map[string]map[string]*models.FeatureRecord),
		notebooks:    make(map[string]*models.NotebookSession),
		experiments:  make(map[string]*models.Experiment),
		expRuns:      make(map[string]*models.ExperimentRun),
		models:       make(map[string]*models.RegisteredModel),
		modelVers:    make(map[string]*models.ModelVersion),
		computeNodes: make(map[string]*models.ComputeNode),
		computeJobs:  make(map[string]*models.ComputeJob),
		searchIdxs:   make(map[string]*models.SearchIndex),
		indexedDocs:  make(map[string]*models.IndexedDocument),
		savedSearches: make(map[string]*models.SavedSearch),
		auditLogs:    make([]*models.AuditLog, 0),
		objectLinks:  make([]*models.ObjectLink, 0),
	}
	m.seedDefaultData()
	return m
}

func (m *MemStore) seedDefaultData() {
	now := time.Now().UTC()

	// 1. Data Sources
	srcPostgres := &models.DataSource{
		ID:            "src-pg-primary",
		Name:          "National Statistical Warehouse",
		SourceType:    "POSTGRESQL",
		ConnectionURI: "postgresql://postgres:***@localhost:5432/statgate",
		AuthType:      "BASIC",
		Status:        "ACTIVE",
		TenantID:      "default",
		LastTestedAt:  &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	m.dataSources[srcPostgres.ID] = srcPostgres

	// 2. Schemas
	schHealth := &models.SchemaDefinition{
		ID:            "sch-demographics-v1",
		Subject:       "national_census_demographics",
		Version:       1,
		SchemaType:    "JSON_SCHEMA",
		SchemaContent: `{"type":"object","properties":{"district":{"type":"string"},"population":{"type":"integer"},"median_age":{"type":"number"}},"required":["district","population"]}`,
		Compatibility: "BACKWARD",
		Description:   "National Census Demographics Core Schema",
		Fields: []models.SchemaField{
			{Name: "district", Type: "STRING", Nullable: false, Description: "Administrative district name"},
			{Name: "population", Type: "INTEGER", Nullable: false, Description: "Total enumerated count"},
			{Name: "median_age", Type: "FLOAT", Nullable: true, Description: "Median population age"},
		},
		TenantID:  "default",
		CreatedBy: "admin",
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.schemas[schHealth.ID] = schHealth

	// 3. Datasets
	dsCensus := &models.Dataset{
		ID:             "ds-census-2026",
		URN:            "urn:statgate:dataset:demographics:census_2026",
		Name:           "National Population & Housing Census 2026",
		Description:    "High-resolution national household demographic indicators, population totals, and migration matrices.",
		Domain:         "demographics",
		Classification: models.ClassificationPublic,
		OwnerTeam:      "Demographic Statistics Directorate",
		OwnerEmail:     "census@statistics.gov.statgate",
		Format:         "PARQUET",
		StorageURI:     "s3://statgate-data-lake/demographics/census_2026.parquet",
		SchemaID:       schHealth.ID,
		Version:        "1.0.0",
		QualityScore:   98.5,
		RowCount:       45000000,
		SizeBytes:      1428571428,
		Tags:           []string{"census", "demographics", "official-statistics", "sdg-1"},
		TenantID:       "default",
		CreatedBy:      "census-pipeline",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	m.datasets[dsCensus.ID] = dsCensus

	// 4. Data Quality Rules
	rule1 := &models.DataQualityRule{
		ID:          "qr-census-district-notnull",
		DatasetID:   dsCensus.ID,
		RuleName:    "District Must Not Be Null",
		RuleType:    "NOT_NULL",
		TargetField: "district",
		Severity:    "ERROR",
		IsEnabled:   true,
		TenantID:    "default",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.qualityRules[rule1.ID] = rule1

	// 5. Lineage
	m.lineageNodes[dsCensus.ID] = &models.LineageNode{
		ID:       dsCensus.ID,
		URN:      dsCensus.URN,
		Type:     "DATASET",
		Name:     dsCensus.Name,
		Domain:   dsCensus.Domain,
		TenantID: "default",
	}

	// 6. Data Pipeline
	pipeETL := &models.DataPipeline{
		ID:              "pipe-census-ingest",
		Name:            "National Census Daily Harmonization Pipeline",
		Description:     "Ingests raw enumeration packets from field tablets, runs validation rules, and stores clean Parquet partitions.",
		PipelineType:    models.PipelineTypeETL,
		Status:          models.PipelineStatusActive,
		CronSchedule:    "0 2 * * *",
		SourceDatasetID: "src-pg-primary",
		TargetDatasetID: dsCensus.ID,
		Stages: []models.PipelineStage{
			{ID: "stg-1", Name: "Extract Field Tablets", StageType: "EXTRACT"},
			{ID: "stg-2", Name: "Validate Schema & Range", StageType: "VALIDATE", DependsOn: []string{"stg-1"}},
			{ID: "stg-3", Name: "Anonymize PII", StageType: "ANONYMIZE", DependsOn: []string{"stg-2"}},
			{ID: "stg-4", Name: "Write Lakehouse Parquet", StageType: "LOAD", DependsOn: []string{"stg-3"}},
		},
		MaxRetries:     3,
		TimeoutSeconds: 3600,
		TenantID:       "default",
		CreatedBy:      "dataops-lead",
		LastRunAt:      &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	m.pipelines[pipeETL.ID] = pipeETL

	// 7. Feature Store
	fvDistrict := &models.FeatureView{
		ID:          "fv-district-socioeconomic",
		Name:        "district_socioeconomic_indicators",
		EntityName:  "district_code",
		Description: "Aggregate district features including poverty index, school density, and electricity access",
		TTLSeconds:  86400,
		Features: []models.FeatureDefinition{
			{Name: "poverty_headcount_ratio", DataType: "FLOAT", Description: "Percentage below national poverty threshold"},
			{Name: "school_density_per_10k", DataType: "FLOAT", Description: "Primary and secondary schools per 10,000 residents"},
			{Name: "electrification_rate", DataType: "FLOAT", Description: "Percentage of households connected to grid"},
		},
		OnlineStore: true,
		Tags:        []string{"socioeconomic", "poverty", "planning"},
		TenantID:    "default",
		CreatedBy:   "lead-data-scientist",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.featureViews[fvDistrict.ID] = fvDistrict

	// 8. Scientific Experiment & ML Model
	expEpi := &models.Experiment{
		ID:          "exp-epi-nowcasting",
		Name:        "Epidemiological Outbreak Nowcasting via SEIR Neural ODE",
		Description: "Real-time infectious disease trajectory forecasting using Bayesian neural ordinary differential equations.",
		Domain:      "EPIDEMIOLOGY",
		Tags:        []string{"epidemiology", "seir", "neural-ode", "forecasting"},
		ArtifactURI: "s3://statgate-science-artifacts/experiments/exp-epi-nowcasting",
		TenantID:    "default",
		CreatedBy:   "dr-sarah-epidemiologist",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.experiments[expEpi.ID] = expEpi

	mlModel := &models.RegisteredModel{
		ID:          "model-maternal-risk-v1",
		Name:        "Maternal Health High-Risk Predictor",
		Description: "Ensemble Gradient Boosted Trees model predicting high-risk pregnancy complications from antenatal records.",
		Domain:      "HEALTHCARE",
		Framework:   "SCIKIT_LEARN",
		LatestStage: models.ModelStageProduction,
		TenantID:    "default",
		CreatedBy:   "dr-sarah-epidemiologist",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.models[mlModel.ID] = mlModel

	// 9. Search Index and Documents
	idxMain := &models.SearchIndex{
		ID:            "idx-statgate-global",
		IndexName:     "statgate_global",
		DocumentCount: 3,
		Dimension:     384,
		Status:        "ACTIVE",
		TenantID:      "default",
		LastIndexedAt: now,
		CreatedAt:     now,
	}
	m.searchIdxs[idxMain.IndexName] = idxMain

	doc1 := &models.IndexedDocument{
		ID:             "doc-census-2026",
		IndexName:      "statgate_global",
		ResourceID:     dsCensus.ID,
		ResourceType:   "DATASET",
		Title:          dsCensus.Name,
		Content:        dsCensus.Description + " population, demographics, migration, census count, statistical indicators.",
		Domain:         dsCensus.Domain,
		Classification: string(dsCensus.Classification),
		Owner:          dsCensus.OwnerTeam,
		Tags:           dsCensus.Tags,
		Vector:         []float32{0.12, 0.45, 0.78, 0.23, 0.89},
		TenantID:       "default",
		IndexedAt:      now,
	}
	m.indexedDocs[doc1.ID] = doc1

	doc2 := &models.IndexedDocument{
		ID:             "doc-model-maternal-risk",
		IndexName:      "statgate_global",
		ResourceID:     mlModel.ID,
		ResourceType:   "MODEL",
		Title:          mlModel.Name,
		Content:        mlModel.Description + " machine learning, healthcare, clinical prediction, maternal mortality, risk scoring.",
		Domain:         mlModel.Domain,
		Classification: "RESTRICTED",
		Owner:          mlModel.CreatedBy,
		Tags:           []string{"maternal-health", "ml-model", "clinical"},
		Vector:         []float32{0.88, 0.12, 0.34, 0.95, 0.11},
		TenantID:       "default",
		IndexedAt:      now,
	}
	m.indexedDocs[doc2.ID] = doc2

	// 10. Compute Node
	node1 := &models.ComputeNode{
		ID:         "node-hpc-01",
		Hostname:   "hpc-compute-worker-01.internal",
		IPAddress:  "10.0.4.12",
		TotalCPUs:  64,
		AllocCPUs:  12,
		TotalRAMGB: 256.0,
		AllocRAMGB: 48.0,
		TotalGPUs:  4,
		AllocGPUs:  1,
		Status:     "READY",
		TenantID:   "default",
		LastPing:   now,
	}
	m.computeNodes[node1.ID] = node1
}

// ─── Data Catalog & Sources Implementations ─────────────────────────────────

func (m *MemStore) CreateDataset(ctx context.Context, ds *models.Dataset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ds.ID == "" {
		ds.ID = "ds-" + uuid.New().String()[:8]
	}
	if ds.URN == "" {
		ds.URN = fmt.Sprintf("urn:statgate:dataset:%s:%s", ds.Domain, ds.ID)
	}
	ds.CreatedAt = time.Now().UTC()
	ds.UpdatedAt = ds.CreatedAt
	m.datasets[ds.ID] = ds

	// Register Lineage Node
	m.lineageNodes[ds.ID] = &models.LineageNode{
		ID:       ds.ID,
		URN:      ds.URN,
		Type:     "DATASET",
		Name:     ds.Name,
		Domain:   ds.Domain,
		TenantID: ds.TenantID,
	}
	return nil
}

func (m *MemStore) GetDatasetByID(ctx context.Context, id string) (*models.Dataset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ds, exists := m.datasets[id]
	if !exists {
		return nil, errors.New("dataset not found")
	}
	return ds, nil
}

func (m *MemStore) GetDatasetByURN(ctx context.Context, urn string) (*models.Dataset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ds := range m.datasets {
		if ds.URN == urn {
			return ds, nil
		}
	}
	return nil, errors.New("dataset not found by URN")
}

func (m *MemStore) ListDatasets(ctx context.Context, tenantID, domain, classification, workspaceID string, limit, offset int) ([]*models.Dataset, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	filtered := make([]*models.Dataset, 0)
	for _, ds := range m.datasets {
		if tenantID != "" && ds.TenantID != tenantID && ds.TenantID != "default" {
			continue
		}
		if workspaceID != "" && ds.WorkspaceID != workspaceID {
			continue
		}
		if domain != "" && ds.Domain != domain {
			continue
		}
		if classification != "" && string(ds.Classification) != classification {
			continue
		}
		filtered = append(filtered, ds)
	}
	total := int64(len(filtered))
	if offset > len(filtered) {
		return []*models.Dataset{}, total, nil
	}
	end := len(filtered)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return filtered[offset:end], total, nil
}

func (m *MemStore) UpdateDataset(ctx context.Context, ds *models.Dataset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, exists := m.datasets[ds.ID]
	if !exists {
		return errors.New("dataset not found")
	}
	ds.UpdatedAt = time.Now().UTC()
	ds.CreatedAt = existing.CreatedAt
	m.datasets[ds.ID] = ds
	return nil
}

func (m *MemStore) DeleteDataset(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.datasets, id)
	delete(m.lineageNodes, id)
	return nil
}

func (m *MemStore) CreateDataSource(ctx context.Context, src *models.DataSource) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if src.ID == "" {
		src.ID = "src-" + uuid.New().String()[:8]
	}
	src.CreatedAt = time.Now().UTC()
	src.UpdatedAt = src.CreatedAt
	m.dataSources[src.ID] = src
	return nil
}

func (m *MemStore) GetDataSourceByID(ctx context.Context, id string) (*models.DataSource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	src, exists := m.dataSources[id]
	if !exists {
		return nil, errors.New("data source not found")
	}
	return src, nil
}

func (m *MemStore) ListDataSources(ctx context.Context, tenantID, sourceType, workspaceID string) ([]*models.DataSource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.DataSource, 0)
	for _, s := range m.dataSources {
		if tenantID != "" && s.TenantID != tenantID && s.TenantID != "default" {
			continue
		}
		if workspaceID != "" && s.WorkspaceID != workspaceID && s.WorkspaceID != "" {
			continue
		}
		if sourceType != "" && s.SourceType != sourceType {
			continue
		}
		res = append(res, s)
	}
	return res, nil
}

func (m *MemStore) UpdateDataSource(ctx context.Context, src *models.DataSource) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, exists := m.dataSources[src.ID]
	if !exists {
		return errors.New("data source not found")
	}
	src.UpdatedAt = time.Now().UTC()
	src.CreatedAt = existing.CreatedAt
	m.dataSources[src.ID] = src
	return nil
}

// ─── Schema Registry Implementations ────────────────────────────────────────

func (m *MemStore) RegisterSchema(ctx context.Context, schema *models.SchemaDefinition) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if schema.ID == "" {
		schema.ID = fmt.Sprintf("sch-%s-v%d", schema.Subject, schema.Version)
	}
	schema.CreatedAt = time.Now().UTC()
	schema.UpdatedAt = schema.CreatedAt
	m.schemas[schema.ID] = schema
	return nil
}

func (m *MemStore) GetSchemaByID(ctx context.Context, id string) (*models.SchemaDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, exists := m.schemas[id]
	if !exists {
		return nil, errors.New("schema not found")
	}
	return s, nil
}

func (m *MemStore) GetLatestSchemaBySubject(ctx context.Context, subject, tenantID string) (*models.SchemaDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var latest *models.SchemaDefinition
	for _, s := range m.schemas {
		if s.Subject == subject && (tenantID == "" || s.TenantID == tenantID || s.TenantID == "default") {
			if latest == nil || s.Version > latest.Version {
				latest = s
			}
		}
	}
	if latest == nil {
		return nil, errors.New("schema with subject not found")
	}
	return latest, nil
}

func (m *MemStore) ListSchemas(ctx context.Context, tenantID string) ([]*models.SchemaDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.SchemaDefinition, 0)
	for _, s := range m.schemas {
		if tenantID != "" && s.TenantID != tenantID && s.TenantID != "default" {
			continue
		}
		res = append(res, s)
	}
	return res, nil
}

// ─── Data Contracts Implementations ─────────────────────────────────────────

func (m *MemStore) CreateDataContract(ctx context.Context, contract *models.DataContract) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if contract.ID == "" {
		contract.ID = "contract-" + uuid.New().String()[:8]
	}
	contract.CreatedAt = time.Now().UTC()
	contract.UpdatedAt = contract.CreatedAt
	m.contracts[contract.ID] = contract
	return nil
}

func (m *MemStore) GetDataContractByID(ctx context.Context, id string) (*models.DataContract, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, exists := m.contracts[id]
	if !exists {
		return nil, errors.New("data contract not found")
	}
	return c, nil
}

func (m *MemStore) ListDataContracts(ctx context.Context, tenantID, datasetID string) ([]*models.DataContract, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.DataContract, 0)
	for _, c := range m.contracts {
		if tenantID != "" && c.TenantID != tenantID && c.TenantID != "default" {
			continue
		}
		if datasetID != "" && c.DatasetID != datasetID {
			continue
		}
		res = append(res, c)
	}
	return res, nil
}

func (m *MemStore) UpdateDataContract(ctx context.Context, contract *models.DataContract) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, exists := m.contracts[contract.ID]
	if !exists {
		return errors.New("data contract not found")
	}
	contract.UpdatedAt = time.Now().UTC()
	contract.CreatedAt = existing.CreatedAt
	m.contracts[contract.ID] = contract
	return nil
}

// ─── Data Quality & Governance Implementations ──────────────────────────────

func (m *MemStore) CreateQualityRule(ctx context.Context, rule *models.DataQualityRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rule.ID == "" {
		rule.ID = "qr-" + uuid.New().String()[:8]
	}
	rule.CreatedAt = time.Now().UTC()
	rule.UpdatedAt = rule.CreatedAt
	m.qualityRules[rule.ID] = rule
	return nil
}

func (m *MemStore) ListQualityRules(ctx context.Context, tenantID, datasetID string) ([]*models.DataQualityRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.DataQualityRule, 0)
	for _, r := range m.qualityRules {
		if tenantID != "" && r.TenantID != tenantID && r.TenantID != "default" {
			continue
		}
		if datasetID != "" && r.DatasetID != datasetID {
			continue
		}
		res = append(res, r)
	}
	return res, nil
}

func (m *MemStore) SaveQualityReport(ctx context.Context, report *models.DataQualityReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if report.ID == "" {
		report.ID = "qrep-" + uuid.New().String()[:8]
	}
	if report.EvaluatedAt.IsZero() {
		report.EvaluatedAt = time.Now().UTC()
	}
	m.qualityReps[report.ID] = report
	return nil
}

func (m *MemStore) GetLatestQualityReport(ctx context.Context, datasetID string) (*models.DataQualityReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var latest *models.DataQualityReport
	for _, r := range m.qualityReps {
		if r.DatasetID == datasetID {
			if latest == nil || r.EvaluatedAt.After(latest.EvaluatedAt) {
				latest = r
			}
		}
	}
	if latest == nil {
		return nil, errors.New("no quality report found for dataset")
	}
	return latest, nil
}

func (m *MemStore) ListQualityReports(ctx context.Context, tenantID, datasetID string, limit int) ([]*models.DataQualityReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.DataQualityReport, 0)
	for _, r := range m.qualityReps {
		if tenantID != "" && r.TenantID != tenantID && r.TenantID != "default" {
			continue
		}
		if datasetID != "" && r.DatasetID != datasetID {
			continue
		}
		res = append(res, r)
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

// ─── Data Lineage Implementations ───────────────────────────────────────────

func (m *MemStore) RecordLineageNode(ctx context.Context, node *models.LineageNode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lineageNodes[node.ID] = node
	return nil
}

func (m *MemStore) RecordLineageEdge(ctx context.Context, edge *models.LineageEdge) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if edge.ID == "" {
		edge.ID = fmt.Sprintf("edge-%s-%s", edge.SourceNodeID, edge.TargetNodeID)
	}
	if edge.CreatedAt.IsZero() {
		edge.CreatedAt = time.Now().UTC()
	}
	m.lineageEdges[edge.ID] = edge
	return nil
}

func (m *MemStore) GetLineageGraph(ctx context.Context, rootID, tenantID string, depth int) (*models.LineageGraph, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	visitedNodes := make(map[string]bool)
	visitedEdges := make(map[string]bool)
	nodes := make([]models.LineageNode, 0)
	edges := make([]models.LineageEdge, 0)

	queue := []string{rootID}
	visitedNodes[rootID] = true

	if n, ok := m.lineageNodes[rootID]; ok {
		nodes = append(nodes, *n)
	}

	curDepth := 0
	for len(queue) > 0 && curDepth <= depth {
		nextQueue := make([]string, 0)
		for _, curID := range queue {
			for _, edge := range m.lineageEdges {
				if visitedEdges[edge.ID] {
					continue
				}
				if edge.SourceNodeID == curID || edge.TargetNodeID == curID {
					visitedEdges[edge.ID] = true
					edges = append(edges, *edge)

					otherID := edge.TargetNodeID
					if edge.TargetNodeID == curID {
						otherID = edge.SourceNodeID
					}
					if !visitedNodes[otherID] {
						visitedNodes[otherID] = true
						if n, ok := m.lineageNodes[otherID]; ok {
							nodes = append(nodes, *n)
						}
						nextQueue = append(nextQueue, otherID)
					}
				}
			}
		}
		queue = nextQueue
		curDepth++
	}

	return &models.LineageGraph{
		RootID: rootID,
		Nodes:  nodes,
		Edges:  edges,
	}, nil
}

// ─── Data Pipelines & Streaming Implementations ─────────────────────────────

func (m *MemStore) CreatePipeline(ctx context.Context, p *models.DataPipeline) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.ID == "" {
		p.ID = "pipe-" + uuid.New().String()[:8]
	}
	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = p.CreatedAt
	m.pipelines[p.ID] = p

	// Register Lineage Node
	m.lineageNodes[p.ID] = &models.LineageNode{
		ID:       p.ID,
		URN:      fmt.Sprintf("urn:statgate:pipeline:%s", p.ID),
		Type:     "PIPELINE",
		Name:     p.Name,
		Domain:   "dataops",
		TenantID: p.TenantID,
	}
	return nil
}

func (m *MemStore) GetPipelineByID(ctx context.Context, id string) (*models.DataPipeline, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, exists := m.pipelines[id]
	if !exists {
		return nil, errors.New("pipeline not found")
	}
	return p, nil
}

func (m *MemStore) ListPipelines(ctx context.Context, tenantID, status, workspaceID string) ([]*models.DataPipeline, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.DataPipeline, 0)
	for _, p := range m.pipelines {
		if tenantID != "" && p.TenantID != tenantID && p.TenantID != "default" {
			continue
		}
		if workspaceID != "" && p.WorkspaceID != workspaceID {
			continue
		}
		if status != "" && string(p.Status) != status {
			continue
		}
		res = append(res, p)
	}
	return res, nil
}

func (m *MemStore) UpdatePipeline(ctx context.Context, p *models.DataPipeline) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, exists := m.pipelines[p.ID]
	if !exists {
		return errors.New("pipeline not found")
	}
	p.UpdatedAt = time.Now().UTC()
	p.CreatedAt = existing.CreatedAt
	m.pipelines[p.ID] = p
	return nil
}

func (m *MemStore) DeletePipeline(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.pipelines, id)
	delete(m.lineageNodes, id)
	return nil
}

func (m *MemStore) RecordPipelineRun(ctx context.Context, run *models.PipelineRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if run.ID == "" {
		run.ID = "run-" + uuid.New().String()[:8]
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = time.Now().UTC()
	}
	m.pipelineRuns[run.ID] = run

	if p, ok := m.pipelines[run.PipelineID]; ok {
		now := run.StartedAt
		p.LastRunAt = &now
	}
	return nil
}

func (m *MemStore) GetPipelineRunByID(ctx context.Context, id string) (*models.PipelineRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	run, exists := m.pipelineRuns[id]
	if !exists {
		return nil, errors.New("pipeline run not found")
	}
	return run, nil
}

func (m *MemStore) ListPipelineRuns(ctx context.Context, pipelineID, tenantID string, limit int) ([]*models.PipelineRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.PipelineRun, 0)
	for _, r := range m.pipelineRuns {
		if tenantID != "" && r.TenantID != tenantID && r.TenantID != "default" {
			continue
		}
		if pipelineID != "" && r.PipelineID != pipelineID {
			continue
		}
		res = append(res, r)
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (m *MemStore) UpdatePipelineRun(ctx context.Context, run *models.PipelineRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pipelineRuns[run.ID] = run
	return nil
}

func (m *MemStore) CreateStreamingJob(ctx context.Context, job *models.StreamingJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job.ID == "" {
		job.ID = "stream-" + uuid.New().String()[:8]
	}
	job.CreatedAt = time.Now().UTC()
	job.UpdatedAt = job.CreatedAt
	job.LastCheckpoint = job.CreatedAt
	m.streamingJobs[job.ID] = job
	return nil
}

func (m *MemStore) GetStreamingJobByID(ctx context.Context, id string) (*models.StreamingJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, exists := m.streamingJobs[id]
	if !exists {
		return nil, errors.New("streaming job not found")
	}
	return job, nil
}

func (m *MemStore) ListStreamingJobs(ctx context.Context, tenantID, status, workspaceID string) ([]*models.StreamingJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.StreamingJob, 0)
	for _, j := range m.streamingJobs {
		if tenantID != "" && j.TenantID != tenantID && j.TenantID != "default" {
			continue
		}
		if workspaceID != "" && j.WorkspaceID != workspaceID && j.WorkspaceID != "" {
			continue
		}
		if status != "" && j.Status != status {
			continue
		}
		res = append(res, j)
	}
	return res, nil
}

func (m *MemStore) UpdateStreamingJob(ctx context.Context, job *models.StreamingJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, exists := m.streamingJobs[job.ID]
	if !exists {
		return errors.New("streaming job not found")
	}
	job.UpdatedAt = time.Now().UTC()
	job.CreatedAt = existing.CreatedAt
	m.streamingJobs[job.ID] = job
	return nil
}

// ─── Feature Store Implementations ──────────────────────────────────────────

func (m *MemStore) CreateFeatureView(ctx context.Context, fv *models.FeatureView) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fv.ID == "" {
		fv.ID = "fv-" + uuid.New().String()[:8]
	}
	fv.CreatedAt = time.Now().UTC()
	fv.UpdatedAt = fv.CreatedAt
	m.featureViews[fv.ID] = fv
	if _, ok := m.features[fv.ID]; !ok {
		m.features[fv.ID] = make(map[string]*models.FeatureRecord)
	}
	return nil
}

func (m *MemStore) GetFeatureViewByID(ctx context.Context, id string) (*models.FeatureView, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	fv, exists := m.featureViews[id]
	if !exists {
		return nil, errors.New("feature view not found")
	}
	return fv, nil
}

func (m *MemStore) ListFeatureViews(ctx context.Context, tenantID, entityName, workspaceID string) ([]*models.FeatureView, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.FeatureView, 0)
	for _, fv := range m.featureViews {
		if tenantID != "" && fv.TenantID != tenantID && fv.TenantID != "default" {
			continue
		}
		if workspaceID != "" && fv.WorkspaceID != workspaceID && fv.WorkspaceID != "" {
			continue
		}
		if entityName != "" && fv.EntityName != entityName {
			continue
		}
		res = append(res, fv)
	}
	return res, nil
}

func (m *MemStore) SaveFeatureRecords(ctx context.Context, records []models.FeatureRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rec := range records {
		if _, ok := m.features[rec.FeatureViewID]; !ok {
			m.features[rec.FeatureViewID] = make(map[string]*models.FeatureRecord)
		}
		r := rec
		if r.Timestamp.IsZero() {
			r.Timestamp = time.Now().UTC()
		}
		m.features[rec.FeatureViewID][rec.EntityKey] = &r
	}
	return nil
}

func (m *MemStore) GetOnlineFeatures(ctx context.Context, featureViewID, entityKey, tenantID string) (*models.FeatureVector, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	subMap, exists := m.features[featureViewID]
	if !exists {
		return nil, errors.New("feature view has no records")
	}
	rec, exists := subMap[entityKey]
	if !exists {
		return nil, errors.New("entity feature record not found")
	}
	return &models.FeatureVector{
		EntityKey:   entityKey,
		Features:    rec.Values,
		RetrievedAt: time.Now().UTC(),
	}, nil
}

// ─── Scientific Computing Implementations ───────────────────────────────────

func (m *MemStore) CreateNotebookSession(ctx context.Context, nb *models.NotebookSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if nb.ID == "" {
		nb.ID = "nb-" + uuid.New().String()[:8]
	}
	nb.CreatedAt = time.Now().UTC()
	nb.UpdatedAt = nb.CreatedAt
	m.notebooks[nb.ID] = nb
	return nil
}

func (m *MemStore) GetNotebookSessionByID(ctx context.Context, id string) (*models.NotebookSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	nb, exists := m.notebooks[id]
	if !exists {
		return nil, errors.New("notebook session not found")
	}
	return nb, nil
}

func (m *MemStore) ListNotebookSessions(ctx context.Context, tenantID, language string) ([]*models.NotebookSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.NotebookSession, 0)
	for _, nb := range m.notebooks {
		if tenantID != "" && nb.TenantID != tenantID && nb.TenantID != "default" {
			continue
		}
		if language != "" && nb.Language != language {
			continue
		}
		res = append(res, nb)
	}
	return res, nil
}

func (m *MemStore) UpdateNotebookSession(ctx context.Context, nb *models.NotebookSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, exists := m.notebooks[nb.ID]
	if !exists {
		return errors.New("notebook session not found")
	}
	nb.UpdatedAt = time.Now().UTC()
	nb.CreatedAt = existing.CreatedAt
	m.notebooks[nb.ID] = nb
	return nil
}

func (m *MemStore) CreateExperiment(ctx context.Context, exp *models.Experiment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if exp.ID == "" {
		exp.ID = "exp-" + uuid.New().String()[:8]
	}
	exp.CreatedAt = time.Now().UTC()
	exp.UpdatedAt = exp.CreatedAt
	m.experiments[exp.ID] = exp
	return nil
}

func (m *MemStore) GetExperimentByID(ctx context.Context, id string) (*models.Experiment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	exp, exists := m.experiments[id]
	if !exists {
		return nil, errors.New("experiment not found")
	}
	return exp, nil
}

func (m *MemStore) ListExperiments(ctx context.Context, tenantID, domain string) ([]*models.Experiment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.Experiment, 0)
	for _, e := range m.experiments {
		if tenantID != "" && e.TenantID != tenantID && e.TenantID != "default" {
			continue
		}
		if domain != "" && e.Domain != domain {
			continue
		}
		res = append(res, e)
	}
	return res, nil
}

func (m *MemStore) RecordExperimentRun(ctx context.Context, run *models.ExperimentRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if run.ID == "" {
		run.ID = "exprun-" + uuid.New().String()[:8]
	}
	if run.StartTime.IsZero() {
		run.StartTime = time.Now().UTC()
	}
	m.expRuns[run.ID] = run
	return nil
}

func (m *MemStore) GetExperimentRunByID(ctx context.Context, id string) (*models.ExperimentRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	run, exists := m.expRuns[id]
	if !exists {
		return nil, errors.New("experiment run not found")
	}
	return run, nil
}

func (m *MemStore) ListExperimentRuns(ctx context.Context, experimentID, tenantID string) ([]*models.ExperimentRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.ExperimentRun, 0)
	for _, r := range m.expRuns {
		if tenantID != "" && r.TenantID != tenantID && r.TenantID != "default" {
			continue
		}
		if experimentID != "" && r.ExperimentID != experimentID {
			continue
		}
		res = append(res, r)
	}
	return res, nil
}

func (m *MemStore) CreateRegisteredModel(ctx context.Context, rm *models.RegisteredModel) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rm.ID == "" {
		rm.ID = "model-" + uuid.New().String()[:8]
	}
	rm.CreatedAt = time.Now().UTC()
	rm.UpdatedAt = rm.CreatedAt
	m.models[rm.ID] = rm

	// Register Lineage Node
	m.lineageNodes[rm.ID] = &models.LineageNode{
		ID:       rm.ID,
		URN:      fmt.Sprintf("urn:statgate:model:%s", rm.ID),
		Type:     "ML_MODEL",
		Name:     rm.Name,
		Domain:   rm.Domain,
		TenantID: rm.TenantID,
	}
	return nil
}

func (m *MemStore) GetRegisteredModelByID(ctx context.Context, id string) (*models.RegisteredModel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	model, exists := m.models[id]
	if !exists {
		return nil, errors.New("registered model not found")
	}
	return model, nil
}

func (m *MemStore) ListRegisteredModels(ctx context.Context, tenantID, domain string) ([]*models.RegisteredModel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.RegisteredModel, 0)
	for _, mod := range m.models {
		if tenantID != "" && mod.TenantID != tenantID && mod.TenantID != "default" {
			continue
		}
		if domain != "" && mod.Domain != domain {
			continue
		}
		res = append(res, mod)
	}
	return res, nil
}

func (m *MemStore) CreateModelVersion(ctx context.Context, mv *models.ModelVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mv.ID == "" {
		mv.ID = fmt.Sprintf("%s-v%d", mv.ModelID, mv.Version)
	}
	mv.CreatedAt = time.Now().UTC()
	mv.UpdatedAt = mv.CreatedAt
	m.modelVers[mv.ID] = mv

	// Update Model Latest Stage
	if mod, ok := m.models[mv.ModelID]; ok {
		mod.LatestStage = mv.Stage
		mod.UpdatedAt = mv.CreatedAt
	}
	return nil
}

func (m *MemStore) ListModelVersions(ctx context.Context, modelID, tenantID string) ([]*models.ModelVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.ModelVersion, 0)
	for _, v := range m.modelVers {
		if tenantID != "" && v.TenantID != tenantID && v.TenantID != "default" {
			continue
		}
		if modelID != "" && v.ModelID != modelID {
			continue
		}
		res = append(res, v)
	}
	return res, nil
}

func (m *MemStore) UpdateModelStage(ctx context.Context, versionID string, stage models.ModelStage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, exists := m.modelVers[versionID]
	if !exists {
		return errors.New("model version not found")
	}
	v.Stage = stage
	v.UpdatedAt = time.Now().UTC()
	if mod, ok := m.models[v.ModelID]; ok {
		mod.LatestStage = stage
		mod.UpdatedAt = v.UpdatedAt
	}
	return nil
}

func (m *MemStore) RegisterComputeNode(ctx context.Context, node *models.ComputeNode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if node.ID == "" {
		node.ID = "node-" + uuid.New().String()[:8]
	}
	node.LastPing = time.Now().UTC()
	m.computeNodes[node.ID] = node
	return nil
}

func (m *MemStore) ListComputeNodes(ctx context.Context, tenantID string) ([]*models.ComputeNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.ComputeNode, 0)
	for _, n := range m.computeNodes {
		if tenantID != "" && n.TenantID != tenantID && n.TenantID != "default" {
			continue
		}
		res = append(res, n)
	}
	return res, nil
}

func (m *MemStore) CreateComputeJob(ctx context.Context, job *models.ComputeJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job.ID == "" {
		job.ID = "cjob-" + uuid.New().String()[:8]
	}
	job.CreatedAt = time.Now().UTC()
	job.Status = "QUEUED"
	m.computeJobs[job.ID] = job
	return nil
}

func (m *MemStore) GetComputeJobByID(ctx context.Context, id string) (*models.ComputeJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, exists := m.computeJobs[id]
	if !exists {
		return nil, errors.New("compute job not found")
	}
	return job, nil
}

func (m *MemStore) ListComputeJobs(ctx context.Context, tenantID, status string) ([]*models.ComputeJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.ComputeJob, 0)
	for _, j := range m.computeJobs {
		if tenantID != "" && j.TenantID != tenantID && j.TenantID != "default" {
			continue
		}
		if status != "" && j.Status != status {
			continue
		}
		res = append(res, j)
	}
	return res, nil
}

func (m *MemStore) UpdateComputeJob(ctx context.Context, job *models.ComputeJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.computeJobs[job.ID] = job
	return nil
}

// ─── Enterprise Search Implementations ──────────────────────────────────────

func (m *MemStore) CreateSearchIndex(ctx context.Context, idx *models.SearchIndex) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if idx.ID == "" {
		idx.ID = "idx-" + uuid.New().String()[:8]
	}
	idx.CreatedAt = time.Now().UTC()
	idx.LastIndexedAt = idx.CreatedAt
	m.searchIdxs[idx.IndexName] = idx
	return nil
}

func (m *MemStore) GetSearchIndex(ctx context.Context, indexName, tenantID string) (*models.SearchIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	idx, exists := m.searchIdxs[indexName]
	if !exists {
		return nil, errors.New("search index not found")
	}
	return idx, nil
}

func (m *MemStore) IndexDocument(ctx context.Context, doc *models.IndexedDocument) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if doc.ID == "" {
		doc.ID = "doc-" + uuid.New().String()[:8]
	}
	doc.IndexedAt = time.Now().UTC()
	m.indexedDocs[doc.ID] = doc

	if idx, ok := m.searchIdxs[doc.IndexName]; ok {
		idx.DocumentCount = int64(len(m.indexedDocs))
		idx.LastIndexedAt = doc.IndexedAt
	}
	return nil
}

func (m *MemStore) DeleteIndexedDocument(ctx context.Context, indexName, documentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.indexedDocs, documentID)
	return nil
}

func (m *MemStore) GetIndexedDocument(ctx context.Context, indexName, documentID string) (*models.IndexedDocument, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	doc, exists := m.indexedDocs[documentID]
	if !exists {
		return nil, errors.New("indexed document not found")
	}
	return doc, nil
}

func (m *MemStore) Search(ctx context.Context, req *models.HybridSearchRequest) (*models.SearchResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	start := time.Now()

	alpha := req.Alpha
	if alpha <= 0 && len(req.Vector) > 0 {
		alpha = 0.5
	}

	queryTokens := strings.Fields(strings.ToLower(req.Query))
	results := make([]models.SearchResultItem, 0)
	facetDomain := make(map[string]int64)
	facetType := make(map[string]int64)

	for _, doc := range m.indexedDocs {
		if req.TenantID != "" && doc.TenantID != req.TenantID && doc.TenantID != "default" {
			continue
		}
		if req.ResourceType != "" && doc.ResourceType != req.ResourceType {
			continue
		}
		if req.Domain != "" && doc.Domain != req.Domain {
			continue
		}
		if req.Classification != "" && doc.Classification != req.Classification {
			continue
		}

		// Calculate Keyword/BM25 Score
		docText := strings.ToLower(fmt.Sprintf("%s %s %s", doc.Title, doc.Content, strings.Join(doc.Tags, " ")))
		bm25Score := 0.0
		for _, token := range queryTokens {
			if strings.Contains(docText, token) {
				bm25Score += 1.0
			}
		}

		// Calculate Vector Cosine Similarity
		vecScore := 0.0
		if len(req.Vector) > 0 && len(doc.Vector) > 0 {
			vecScore = cosineSimilarity(req.Vector, doc.Vector)
		}

		// Combined Hybrid Score
		finalScore := (1.0-alpha)*bm25Score + alpha*vecScore
		if len(queryTokens) == 0 && len(req.Vector) == 0 {
			finalScore = 1.0 // match all
		}

		if finalScore > 0.0 || (len(queryTokens) == 0 && len(req.Vector) == 0) {
			snippet := doc.Content
			if len(snippet) > 180 {
				snippet = snippet[:180] + "..."
			}

			results = append(results, models.SearchResultItem{
				DocumentID:     doc.ID,
				ResourceID:     doc.ResourceID,
				ResourceType:   doc.ResourceType,
				Title:          doc.Title,
				Snippet:        snippet,
				Domain:         doc.Domain,
				Classification: doc.Classification,
				Score:          finalScore,
				BM25Score:      bm25Score,
				VectorScore:    vecScore,
				Metadata:       doc.Metadata,
				IndexedAt:      doc.IndexedAt,
			})

			facetDomain[doc.Domain]++
			facetType[doc.ResourceType]++
		}
	}

	// Sort results by combined score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	totalHits := int64(len(results))
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := req.Offset
	if offset > len(results) {
		results = []models.SearchResultItem{}
	} else {
		end := len(results)
		if offset+limit < end {
			end = offset + limit
		}
		results = results[offset:end]
	}

	facets := []models.FacetResult{
		{Field: "domain", Counts: facetDomain},
		{Field: "resource_type", Counts: facetType},
	}

	return &models.SearchResponse{
		Query:       req.Query,
		TotalHits:   totalHits,
		ExecutionMs: time.Since(start).Milliseconds(),
		Results:     results,
		Facets:      facets,
	}, nil
}

func (m *MemStore) GetSuggestions(ctx context.Context, prefix, tenantID string, limit int) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 {
		limit = 10
	}
	p := strings.ToLower(prefix)
	res := make([]string, 0)
	for _, doc := range m.indexedDocs {
		if strings.HasPrefix(strings.ToLower(doc.Title), p) {
			res = append(res, doc.Title)
		}
		for _, tag := range doc.Tags {
			if strings.HasPrefix(strings.ToLower(tag), p) {
				res = append(res, tag)
			}
		}
		if len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (m *MemStore) CreateSavedSearch(ctx context.Context, ss *models.SavedSearch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ss.ID == "" {
		ss.ID = "ss-" + uuid.New().String()[:8]
	}
	ss.CreatedAt = time.Now().UTC()
	m.savedSearches[ss.ID] = ss
	return nil
}

func (m *MemStore) ListSavedSearches(ctx context.Context, tenantID, createdBy string) ([]*models.SavedSearch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.SavedSearch, 0)
	for _, ss := range m.savedSearches {
		if tenantID != "" && ss.TenantID != tenantID && ss.TenantID != "default" {
			continue
		}
		if createdBy != "" && ss.CreatedBy != createdBy {
			continue
		}
		res = append(res, ss)
	}
	return res, nil
}

// ─── Compliance & Cross-App Primitives ──────────────────────────────────────

func (m *MemStore) LogAuditEvent(ctx context.Context, entry *models.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry.ID == "" {
		entry.ID = "audit-" + uuid.New().String()[:8]
	}
	if entry.EventTimestamp.IsZero() {
		entry.EventTimestamp = time.Now().UTC()
	}
	m.auditLogs = append(m.auditLogs, entry)
	return nil
}

func (m *MemStore) ListAuditLogs(ctx context.Context, tenantID, resourceType string, limit int) ([]*models.AuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.AuditLog, 0, len(m.auditLogs))
	for i := len(m.auditLogs) - 1; i >= 0; i-- {
		entry := m.auditLogs[i]
		if tenantID != "" && entry.ActorTenantID != tenantID && entry.ActorTenantID != "default" && entry.ActorTenantID != "" {
			continue
		}
		if resourceType != "" && entry.ResourceType != resourceType {
			continue
		}
		res = append(res, entry)
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (m *MemStore) CreateObjectLink(ctx context.Context, link *models.ObjectLink) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if link.CreatedAt.IsZero() {
		link.CreatedAt = time.Now().UTC()
	}
	link.ID = len(m.objectLinks) + 1
	m.objectLinks = append(m.objectLinks, link)
	return nil
}

func (m *MemStore) GetObjectLinks(ctx context.Context, sourceType, sourceID, tenantID string) ([]*models.ObjectLink, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.ObjectLink, 0)
	for _, l := range m.objectLinks {
		if (l.SourceType == sourceType && l.SourceID == sourceID) ||
			(l.TargetType == sourceType && l.TargetID == sourceID) {
			if tenantID != "" && l.TenantID != tenantID && l.TenantID != "default" && l.TenantID != "" {
				continue
			}
			res = append(res, l)
		}
	}
	return res, nil
}

func cosineSimilarity(a, b []float32) float64 {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	if minLen == 0 {
		return 0.0
	}
	var dot, normA, normB float64
	for i := 0; i < minLen; i++ {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
