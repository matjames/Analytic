package science

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"statdata-backend/internal/models"
	"statdata-backend/internal/store"
)

// ExperimentTracker provides metric, parameter, and artifact tracking for scientific runs
type ExperimentTracker struct {
	store store.Store
}

// NewExperimentTracker creates an experiment tracker
func NewExperimentTracker(s store.Store) *ExperimentTracker {
	return &ExperimentTracker{store: s}
}

// StartRun begins a new trackable trial
func (et *ExperimentTracker) StartRun(ctx context.Context, expID, runName, createdBy, tenantID string, params map[string]interface{}) (*models.ExperimentRun, error) {
	_, err := et.store.GetExperimentByID(ctx, expID)
	if err != nil {
		return nil, fmt.Errorf("experiment not found: %w", err)
	}

	runID := "exprun-" + uuid.New().String()[:8]
	run := &models.ExperimentRun{
		ID:           runID,
		ExperimentID: expID,
		RunName:      runName,
		Status:       "RUNNING",
		Parameters:   params,
		Metrics:      make(map[string]float64),
		Tags:         make(map[string]string),
		Artifacts:    make([]models.ModelArtifact, 0),
		StartTime:    time.Now().UTC(),
		TenantID:     tenantID,
		CreatedBy:    createdBy,
	}

	if err := et.store.RecordExperimentRun(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to start run: %w", err)
	}
	return run, nil
}

// LogMetrics updates evaluation metrics for an active run
func (et *ExperimentTracker) LogMetrics(ctx context.Context, runID string, metrics map[string]float64) (*models.ExperimentRun, error) {
	run, err := et.store.GetExperimentRunByID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("experiment run not found: %w", err)
	}

	if run.Metrics == nil {
		run.Metrics = make(map[string]float64)
	}
	for k, v := range metrics {
		run.Metrics[k] = v
	}

	_ = et.store.RecordExperimentRun(ctx, run)
	return run, nil
}

// CompleteRun finishes a trial and calculates execution duration
func (et *ExperimentTracker) CompleteRun(ctx context.Context, runID string, artifacts []models.ModelArtifact) (*models.ExperimentRun, error) {
	run, err := et.store.GetExperimentRunByID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("experiment run not found: %w", err)
	}

	now := time.Now().UTC()
	run.Status = "COMPLETED"
	run.EndTime = &now
	run.DurationMs = now.Sub(run.StartTime).Milliseconds()
	if artifacts != nil {
		run.Artifacts = artifacts
	}

	_ = et.store.RecordExperimentRun(ctx, run)
	return run, nil
}
