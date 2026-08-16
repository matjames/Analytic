package science

import (
	"context"
	"fmt"
	"time"

	"statdata-backend/internal/models"
	"statdata-backend/internal/store"
)

// ModelRegistry manages the lifecycle and versioning of machine learning & statistical models
type ModelRegistry struct {
	store store.Store
}

// NewModelRegistry creates a new model registry manager
func NewModelRegistry(s store.Store) *ModelRegistry {
	return &ModelRegistry{store: s}
}

// RegisterNewVersion publishes a new model artifact version
func (mr *ModelRegistry) RegisterNewVersion(ctx context.Context, modelID, artifactURI, createdBy, tenantID string, metrics map[string]float64) (*models.ModelVersion, error) {
	model, err := mr.store.GetRegisteredModelByID(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}

	existingVers, err := mr.store.ListModelVersions(ctx, modelID, tenantID)
	if err != nil {
		return nil, err
	}
	nextVer := len(existingVers) + 1

	mv := &models.ModelVersion{
		ID:             fmt.Sprintf("%s-v%d", model.ID, nextVer),
		ModelID:        model.ID,
		Version:        nextVer,
		Stage:          models.ModelStageStaging,
		ArtifactURI:    artifactURI,
		MetricsSummary: metrics,
		TenantID:       tenantID,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := mr.store.CreateModelVersion(ctx, mv); err != nil {
		return nil, fmt.Errorf("failed to register model version: %w", err)
	}

	return mv, nil
}

// TransitionStage promotes or archives a model version
func (mr *ModelRegistry) TransitionStage(ctx context.Context, versionID string, targetStage models.ModelStage, tenantID string) error {
	return mr.store.UpdateModelStage(ctx, versionID, targetStage)
}
