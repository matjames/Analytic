package pipeline

import (
	"context"
	"fmt"
	"time"

	"statdata-backend/internal/models"
	"statdata-backend/internal/store"
)

// LineageTracker manages lineage graphs and tracks data transformations
type LineageTracker struct {
	store store.Store
}

// NewLineageTracker creates a new lineage tracker
func NewLineageTracker(s store.Store) *LineageTracker {
	return &LineageTracker{store: s}
}

// RecordPipelineExecutionLineage connects source datasets, pipelines, and target datasets in the lineage graph
func (lt *LineageTracker) RecordPipelineExecutionLineage(ctx context.Context, p *models.DataPipeline, runID string) error {
	// 1. Ensure Pipeline Node is registered
	pipeNode := &models.LineageNode{
		ID:       p.ID,
		URN:      fmt.Sprintf("urn:statgate:pipeline:%s", p.ID),
		Type:     "PIPELINE",
		Name:     p.Name,
		Domain:   "dataops",
		TenantID: p.TenantID,
		Metadata: map[string]interface{}{
			"pipeline_type": p.PipelineType,
			"last_run_id":   runID,
		},
	}
	_ = lt.store.RecordLineageNode(ctx, pipeNode)

	// 2. Connect Source -> Pipeline
	if p.SourceDatasetID != "" {
		sourceEdge := &models.LineageEdge{
			ID:             fmt.Sprintf("edge-%s-%s", p.SourceDatasetID, p.ID),
			SourceNodeID:   p.SourceDatasetID,
			TargetNodeID:   p.ID,
			RelationType:   "READS_FROM",
			PipelineID:     p.ID,
			Transformation: string(p.PipelineType),
			TenantID:       p.TenantID,
			CreatedAt:      time.Now().UTC(),
		}
		_ = lt.store.RecordLineageEdge(ctx, sourceEdge)
	}

	// 3. Connect Pipeline -> Target
	if p.TargetDatasetID != "" {
		targetEdge := &models.LineageEdge{
			ID:             fmt.Sprintf("edge-%s-%s", p.ID, p.TargetDatasetID),
			SourceNodeID:   p.ID,
			TargetNodeID:   p.TargetDatasetID,
			RelationType:   "TRANSFORMS_TO",
			PipelineID:     p.ID,
			Transformation: string(p.PipelineType),
			TenantID:       p.TenantID,
			CreatedAt:      time.Now().UTC(),
		}
		_ = lt.store.RecordLineageEdge(ctx, targetEdge)
	}

	return nil
}

// GetGraph retrieves complete lineage graph for a given entity
func (lt *LineageTracker) GetGraph(ctx context.Context, rootID, tenantID string, depth int) (*models.LineageGraph, error) {
	if depth <= 0 {
		depth = 3
	}
	return lt.store.GetLineageGraph(ctx, rootID, tenantID, depth)
}
