package tests

import (
	"context"
	"testing"

	"statdata-backend/internal/models"
	"statdata-backend/internal/science"
	"statdata-backend/internal/store"
)

func TestScienceNotebookAndExperimentTracking(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemStore()

	notebookRunner := science.NewNotebookRunner(memStore)
	experimentTracker := science.NewExperimentTracker(memStore)
	modelRegistry := science.NewModelRegistry(memStore)
	clusterOrchestrator := science.NewClusterOrchestrator(memStore)

	// 1. Notebook Workspace & Cell Execution
	nb := &models.NotebookSession{
		ID:       "nb-stat-analysis",
		Title:    "Mortality Trends Analysis",
		Language: "PYTHON",
		TenantID: "default",
	}
	_ = memStore.CreateNotebookSession(ctx, nb)

	cell, err := notebookRunner.ExecuteCell(ctx, nb.ID, "", "import pandas as pd\ndf = pd.read_csv('data.csv')\ndf.head()", "PYTHON", "default")
	if err != nil {
		t.Fatalf("ExecuteCell failed: %v", err)
	}
	if cell.Status != "SUCCESS" {
		t.Errorf("Expected cell status SUCCESS, got %s", cell.Status)
	}

	// 2. Experiment Tracking
	exp := &models.Experiment{
		ID:       "exp-deep-learning",
		Name:     "Transformer Forecasting",
		Domain:   "STATISTICS",
		TenantID: "default",
	}
	_ = memStore.CreateExperiment(ctx, exp)

	run, err := experimentTracker.StartRun(ctx, exp.ID, "Trial-01", "scientist-1", "default", map[string]interface{}{"lr": 0.001, "epochs": 50})
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	updatedRun, err := experimentTracker.LogMetrics(ctx, run.ID, map[string]float64{"val_loss": 0.042, "r2_score": 0.965})
	if err != nil {
		t.Fatalf("LogMetrics failed: %v", err)
	}
	if updatedRun.Metrics["r2_score"] != 0.965 {
		t.Errorf("Expected r2_score 0.965, got %f", updatedRun.Metrics["r2_score"])
	}

	completedRun, err := experimentTracker.CompleteRun(ctx, run.ID, []models.ModelArtifact{
		{Name: "model.pt", ArtifactURI: "s3://artifacts/model.pt", SizeBytes: 102400},
	})
	if err != nil {
		t.Fatalf("CompleteRun failed: %v", err)
	}
	if completedRun.Status != "COMPLETED" {
		t.Errorf("Expected run status COMPLETED, got %s", completedRun.Status)
	}

	// 3. Model Registry Versioning & Stage Promotion
	mlModel := &models.RegisteredModel{
		ID:        "model-forecaster",
		Name:      "Forecasting Model",
		Domain:    "STATISTICS",
		Framework: "PYTORCH",
		TenantID:  "default",
	}
	_ = memStore.CreateRegisteredModel(ctx, mlModel)

	ver, err := modelRegistry.RegisterNewVersion(ctx, mlModel.ID, "s3://artifacts/model.pt", "scientist-1", "default", map[string]float64{"r2": 0.965})
	if err != nil {
		t.Fatalf("RegisterNewVersion failed: %v", err)
	}
	if ver.Version != 1 || ver.Stage != models.ModelStageStaging {
		t.Errorf("Expected version 1 in STAGING, got v%d in %s", ver.Version, ver.Stage)
	}

	err = modelRegistry.TransitionStage(ctx, ver.ID, models.ModelStageProduction, "default")
	if err != nil {
		t.Fatalf("TransitionStage failed: %v", err)
	}

	m, _ := memStore.GetRegisteredModelByID(ctx, mlModel.ID)
	if m.LatestStage != models.ModelStageProduction {
		t.Errorf("Expected model latest stage PRODUCTION, got %s", m.LatestStage)
	}

	// 4. Compute Cluster Scheduling
	job := &models.ComputeJob{
		ID:           "cjob-monte-carlo",
		Name:         "Monte Carlo 100k Iterations",
		JobType:      "MONTE_CARLO",
		RequiredCPUs: 4,
		RequiredRAMGB: 16.0,
		TenantID:     "default",
	}
	scheduledJob, err := clusterOrchestrator.SubmitJob(ctx, job)
	if err != nil {
		t.Fatalf("SubmitJob failed: %v", err)
	}
	if scheduledJob.Status != "RUNNING" && scheduledJob.Status != "QUEUED" {
		t.Errorf("Expected scheduled job status RUNNING or QUEUED, got %s", scheduledJob.Status)
	}
}
