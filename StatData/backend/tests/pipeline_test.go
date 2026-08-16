package tests

import (
	"context"
	"testing"

	"statdata-backend/internal/models"
	"statdata-backend/internal/pipeline"
	"statdata-backend/internal/store"
)

func TestPipelineExecutionAndLineage(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemStore()
	qualityEngine := pipeline.NewQualityEngine(memStore)
	lineageTracker := pipeline.NewLineageTracker(memStore)
	engine := pipeline.NewPipelineEngine(memStore, qualityEngine, lineageTracker)

	// 1. Create a test pipeline
	pipe := &models.DataPipeline{
		ID:              "pipe-health-etl",
		Name:            "National Health Indicators ETL",
		PipelineType:    models.PipelineTypeETL,
		Status:          models.PipelineStatusActive,
		SourceDatasetID: "src-pg-primary",
		TargetDatasetID: "ds-census-2026",
		Stages: []models.PipelineStage{
			{ID: "stg-ext", Name: "Extract", StageType: "EXTRACT"},
			{ID: "stg-val", Name: "Validate", StageType: "VALIDATE"},
			{ID: "stg-load", Name: "Load", StageType: "LOAD"},
		},
		TenantID:  "default",
		CreatedBy: "test-user",
	}

	if err := memStore.CreatePipeline(ctx, pipe); err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// 2. Trigger Pipeline Execution
	run, err := engine.TriggerPipeline(ctx, pipe.ID, "MANUAL", "test-user", "default")
	if err != nil {
		t.Fatalf("TriggerPipeline failed: %v", err)
	}

	if run.Status != models.RunStatusSuccess {
		t.Errorf("Expected run status SUCCESS, got %s", run.Status)
	}
	if len(run.StageRuns) != 3 {
		t.Errorf("Expected 3 stage runs, got %d", len(run.StageRuns))
	}
	if run.RecordsRead <= 0 {
		t.Errorf("Expected positive records read, got %d", run.RecordsRead)
	}

	// 3. Verify Lineage Graph
	graph, err := lineageTracker.GetGraph(ctx, pipe.ID, "default", 3)
	if err != nil {
		t.Fatalf("GetGraph failed: %v", err)
	}
	if len(graph.Nodes) == 0 {
		t.Errorf("Expected lineage nodes for pipeline, got 0")
	}
	if len(graph.Edges) == 0 {
		t.Errorf("Expected lineage edges connecting pipeline, got 0")
	}
}

func TestDataQualityEvaluation(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemStore()
	qualityEngine := pipeline.NewQualityEngine(memStore)

	// 1. Create a custom dataset and rule
	ds := &models.Dataset{
		ID:             "ds-survey-test",
		Name:           "Household Survey 2026",
		Domain:         "demographics",
		Classification: models.ClassificationInternal,
		OwnerTeam:      "Surveys",
		TenantID:       "default",
	}
	_ = memStore.CreateDataset(ctx, ds)

	rule := &models.DataQualityRule{
		ID:          "qr-test-notnull",
		DatasetID:   ds.ID,
		RuleName:    "Household ID Not Null",
		RuleType:    "NOT_NULL",
		TargetField: "hh_id",
		Severity:    "ERROR",
		IsEnabled:   true,
		TenantID:    "default",
	}
	_ = memStore.CreateQualityRule(ctx, rule)

	// 2. Evaluate with valid data
	validData := []map[string]interface{}{
		{"hh_id": "HH-001", "size": 4},
		{"hh_id": "HH-002", "size": 2},
	}
	rep1, err := qualityEngine.EvaluateDataset(ctx, ds.ID, "run-1", "default", validData)
	if err != nil {
		t.Fatalf("EvaluateDataset valid data failed: %v", err)
	}
	if rep1.Status != "PASSED" || rep1.QualityScore != 100.0 {
		t.Errorf("Expected PASSED with 100 score, got %s with %.2f", rep1.Status, rep1.QualityScore)
	}

	// 3. Evaluate with invalid data (null value)
	invalidData := []map[string]interface{}{
		{"hh_id": "HH-001", "size": 4},
		{"hh_id": nil, "size": 2},
	}
	rep2, err := qualityEngine.EvaluateDataset(ctx, ds.ID, "run-2", "default", invalidData)
	if err != nil {
		t.Fatalf("EvaluateDataset invalid data failed: %v", err)
	}
	if rep2.Status != "FAILED" || rep2.FailedRules != 1 {
		t.Errorf("Expected FAILED with 1 failed rule, got %s with %d failed", rep2.Status, rep2.FailedRules)
	}
}
