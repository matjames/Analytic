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

	// 1. Register the source dataset with four real rows.
	src := &models.Dataset{
		ID:             "ds-src-real",
		Name:           "District Register (source)",
		Domain:         "demographics",
		Classification: models.ClassificationInternal,
		OwnerTeam:      "Surveys",
		TenantID:       "default",
	}
	if err := memStore.CreateDataset(ctx, src); err != nil {
		t.Fatalf("Failed to create source dataset: %v", err)
	}
	sourceRows := []map[string]interface{}{
		{"district": "North", "population": 52000, "note": "x"},
		{"district": "South", "population": 31000, "note": "y"},
		{"district": "East", "population": 8400, "note": "z"},
		{"district": "West", "population": 47000, "note": "w"},
	}
	if _, err := memStore.AppendDatasetRecords(ctx, src.ID, "default", sourceRows); err != nil {
		t.Fatalf("Failed to store source rows: %v", err)
	}

	// 2. Register the target dataset.
	tgt := &models.Dataset{
		ID:             "ds-tgt-real",
		Name:           "Filtered Regions (target)",
		Domain:         "demographics",
		Classification: models.ClassificationInternal,
		OwnerTeam:      "Data Platform",
		TenantID:       "default",
	}
	if err := memStore.CreateDataset(ctx, tgt); err != nil {
		t.Fatalf("Failed to create target dataset: %v", err)
	}

	// 3. Register a real quality rule on the source dataset.
	rule := &models.DataQualityRule{
		ID:          "qr-real-notnull",
		DatasetID:   src.ID,
		RuleName:    "District Not Null",
		RuleType:    "NOT_NULL",
		TargetField: "district",
		Severity:    "ERROR",
		IsEnabled:   true,
		TenantID:    "default",
	}
	if err := memStore.CreateQualityRule(ctx, rule); err != nil {
		t.Fatalf("Failed to create quality rule: %v", err)
	}

	// 4. Create the pipeline: extract, filter, rename, validate, load.
	pipe := &models.DataPipeline{
		ID:              "pipe-real-etl",
		Name:            "District Register ETL",
		PipelineType:    models.PipelineTypeETL,
		Status:          models.PipelineStatusActive,
		SourceDatasetID: src.ID,
		TargetDatasetID: tgt.ID,
		Stages: []models.PipelineStage{
			{ID: "stg-ext", Name: "Extract", StageType: "EXTRACT"},
			{ID: "stg-val", Name: "Validate", StageType: "VALIDATE"},
			{ID: "stg-xf", Name: "Filter and Rename", StageType: "TRANSFORM", Config: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{"type": "filter", "field": "population", "op": "gte", "value": 10000},
					map[string]interface{}{"type": "rename", "from": "district", "to": "region"},
					map[string]interface{}{"type": "remove_fields", "fields": []interface{}{"note"}},
				},
			}},
			{ID: "stg-load", Name: "Load", StageType: "LOAD"},
		},
		TenantID:  "default",
		CreatedBy: "test-user",
	}
	if err := memStore.CreatePipeline(ctx, pipe); err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// 5. Trigger real pipeline execution.
	run, err := engine.TriggerPipeline(ctx, pipe.ID, "MANUAL", "test-user", "default")
	if err != nil {
		t.Fatalf("TriggerPipeline failed: %v", err)
	}

	if run.Status != models.RunStatusSuccess {
		t.Fatalf("Expected run status SUCCESS, got %s (%s)", run.Status, run.ErrorMessage)
	}
	if len(run.StageRuns) != 4 {
		t.Fatalf("Expected 4 stage runs, got %d", len(run.StageRuns))
	}
	if run.RecordsRead != 4 {
		t.Errorf("Expected 4 records read from the source dataset, got %d", run.RecordsRead)
	}
	if run.RecordsWritten != 3 {
		t.Errorf("Expected 3 records written to the target dataset, got %d", run.RecordsWritten)
	}
	if run.RecordsRejected != 1 {
		t.Errorf("Expected 1 rejected record (East below population threshold), got %d", run.RecordsRejected)
	}

	// 6. The target dataset must physically hold the transformed rows.
	targetRows, err := memStore.ListDatasetRecords(ctx, tgt.ID, 0)
	if err != nil {
		t.Fatalf("ListDatasetRecords failed: %v", err)
	}
	if len(targetRows) != 3 {
		t.Fatalf("Expected 3 stored target rows, got %d", len(targetRows))
	}
	for _, row := range targetRows {
		if _, ok := row["region"]; !ok {
			t.Errorf("Expected transformed row to carry 'region', got %v", row)
		}
		if _, ok := row["district"]; ok {
			t.Errorf("Expected 'district' to be renamed away, got %v", row)
		}
		if _, ok := row["note"]; ok {
			t.Errorf("Expected 'note' to be removed, got %v", row)
		}
	}

	// 7. Catalog counters must reflect real storage.
	stored, err := memStore.GetDatasetByID(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("GetDatasetByID failed: %v", err)
	}
	if stored.RowCount != 3 {
		t.Errorf("Expected target row_count 3, got %d", stored.RowCount)
	}
	if stored.SizeBytes <= 0 {
		t.Errorf("Expected positive target size_bytes, got %d", stored.SizeBytes)
	}

	// 8. Verify Lineage Graph.
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

func TestPipelineFailsOnMissingSource(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemStore()
	engine := pipeline.NewPipelineEngine(memStore, pipeline.NewQualityEngine(memStore), pipeline.NewLineageTracker(memStore))

	pipe := &models.DataPipeline{
		ID:              "pipe-missing-source",
		Name:            "Ghost Read",
		PipelineType:    models.PipelineTypeETL,
		Status:          models.PipelineStatusActive,
		SourceDatasetID: "ds-does-not-exist",
		TargetDatasetID: "ds-also-missing",
		Stages: []models.PipelineStage{
			{ID: "stg-ext", Name: "Extract", StageType: "EXTRACT"},
			{ID: "stg-load", Name: "Load", StageType: "LOAD"},
		},
		TenantID:  "default",
		CreatedBy: "test-user",
	}
	if err := memStore.CreatePipeline(ctx, pipe); err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	run, err := engine.TriggerPipeline(ctx, pipe.ID, "MANUAL", "test-user", "default")
	if err != nil {
		t.Fatalf("TriggerPipeline returned infrastructure error: %v", err)
	}
	if run.Status != models.RunStatusFailed {
		t.Fatalf("Expected FAILED status for a missing source dataset, got %s", run.Status)
	}
	if run.ErrorMessage == "" {
		t.Errorf("Expected a real error message on the failed run")
	}
	if len(run.StageRuns) != 1 || run.StageRuns[0].Status != models.RunStatusFailed {
		t.Errorf("Expected exactly one FAILED extract stage run, got %+v", run.StageRuns)
	}
}

func TestPipelineFailsOnQualityViolation(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemStore()
	engine := pipeline.NewPipelineEngine(memStore, pipeline.NewQualityEngine(memStore), pipeline.NewLineageTracker(memStore))

	src := &models.Dataset{ID: "ds-qviol-src", Name: "Dirty Register", Domain: "demographics", Classification: models.ClassificationInternal, OwnerTeam: "Surveys", TenantID: "default"}
	tgt := &models.Dataset{ID: "ds-qviol-tgt", Name: "Clean Register", Domain: "demographics", Classification: models.ClassificationInternal, OwnerTeam: "Data Platform", TenantID: "default"}
	_ = memStore.CreateDataset(ctx, src)
	_ = memStore.CreateDataset(ctx, tgt)
	_, _ = memStore.AppendDatasetRecords(ctx, src.ID, "default", []map[string]interface{}{
		{"district": "North", "population": 100},
		{"district": nil, "population": 200},
	})
	_ = memStore.CreateQualityRule(ctx, &models.DataQualityRule{
		ID: "qr-qviol", DatasetID: src.ID, RuleName: "District Not Null",
		RuleType: "NOT_NULL", TargetField: "district", Severity: "ERROR",
		IsEnabled: true, TenantID: "default",
	})

	pipe := &models.DataPipeline{
		ID: "pipe-qviol", Name: "Dirty ETL", PipelineType: models.PipelineTypeETL,
		Status: models.PipelineStatusActive, SourceDatasetID: src.ID, TargetDatasetID: tgt.ID,
		Stages: []models.PipelineStage{
			{ID: "stg-ext", Name: "Extract", StageType: "EXTRACT"},
			{ID: "stg-val", Name: "Validate", StageType: "VALIDATE"},
			{ID: "stg-load", Name: "Load", StageType: "LOAD"},
		},
		TenantID: "default", CreatedBy: "test-user",
	}
	_ = memStore.CreatePipeline(ctx, pipe)

	run, err := engine.TriggerPipeline(ctx, pipe.ID, "MANUAL", "test-user", "default")
	if err != nil {
		t.Fatalf("TriggerPipeline returned infrastructure error: %v", err)
	}
	if run.Status != models.RunStatusFailed {
		t.Fatalf("Expected FAILED status on quality violation, got %s", run.Status)
	}
	if len(run.StageRuns) != 2 || run.StageRuns[1].Status != models.RunStatusFailed {
		t.Errorf("Expected the VALIDATE stage to fail, got %+v", run.StageRuns)
	}
	// The load stage must never have executed: the target stays empty.
	count, _ := memStore.CountDatasetRecords(ctx, tgt.ID)
	if count != 0 {
		t.Errorf("Expected 0 target rows after a failed validation, got %d", count)
	}
}
