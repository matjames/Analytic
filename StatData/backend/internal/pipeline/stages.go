package pipeline

import (
	"context"
	"fmt"
	"time"

	"statdata-backend/internal/models"
)

// runExtract reads the real stored rows of the source dataset.
func (pe *PipelineEngine) runExtract(ctx context.Context, stage models.PipelineStage, p *models.DataPipeline) ([]map[string]interface{}, map[string]interface{}, error) {
	if p.SourceDatasetID == "" {
		return nil, nil, fmt.Errorf("pipeline has no source_dataset_id configured")
	}
	if _, err := pe.store.GetDatasetByID(ctx, p.SourceDatasetID); err != nil {
		return nil, nil, fmt.Errorf("source dataset %s not found: %w", p.SourceDatasetID, err)
	}
	limit := 0
	if v, ok := stage.Config["max_records"].(float64); ok && v > 0 {
		limit = int(v)
	}
	records, err := pe.store.ListDatasetRecords(ctx, p.SourceDatasetID, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read source dataset %s: %w", p.SourceDatasetID, err)
	}
	return records, map[string]interface{}{
		"source_dataset_id": p.SourceDatasetID,
		"records_extracted": len(records),
	}, nil
}

// runTransform applies the configured per-record operations in order.
func (pe *PipelineEngine) runTransform(ctx context.Context, stage models.PipelineStage, records []map[string]interface{}) ([]map[string]interface{}, map[string]interface{}, error) {
	ops, err := decodeOperations(stage.Config)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]interface{}, 0, len(records))
	for i, rec := range records {
		keep, transformed, err := applyTransformOperations(rec, ops)
		if err != nil {
			return nil, nil, fmt.Errorf("record %d: %w", i, err)
		}
		if keep {
			out = append(out, transformed)
		}
	}
	return out, map[string]interface{}{
		"operations":  len(ops),
		"records_in":  len(records),
		"records_out": len(out),
	}, nil
}

// runEnrich adds or overwrites a field with a constant value on every record.
func (pe *PipelineEngine) runEnrich(ctx context.Context, stage models.PipelineStage, records []map[string]interface{}) ([]map[string]interface{}, map[string]interface{}, error) {
	field, _ := stage.Config["field"].(string)
	if field == "" {
		return nil, nil, fmt.Errorf("enrich stage requires a 'field' in its config")
	}
	value, hasValue := stage.Config["value"]
	if !hasValue {
		return nil, nil, fmt.Errorf("enrich stage requires a 'value' in its config")
	}
	out := make([]map[string]interface{}, 0, len(records))
	for _, rec := range records {
		clone := cloneRecord(rec)
		clone[field] = value
		out = append(out, clone)
	}
	return out, map[string]interface{}{"enriched_field": field, "records_out": len(out)}, nil
}

// runAnonymize replaces the listed fields with their SHA-256 hash.
func (pe *PipelineEngine) runAnonymize(ctx context.Context, stage models.PipelineStage, records []map[string]interface{}) ([]map[string]interface{}, map[string]interface{}, error) {
	fields, err := decodeStringList(stage.Config, "fields")
	if err != nil {
		return nil, nil, fmt.Errorf("anonymize stage: %w", err)
	}
	out := make([]map[string]interface{}, 0, len(records))
	for _, rec := range records {
		clone := cloneRecord(rec)
		for _, f := range fields {
			if v, ok := clone[f]; ok && v != nil {
				clone[f] = hashValue(v)
			}
		}
		out = append(out, clone)
	}
	return out, map[string]interface{}{"anonymized_fields": fields, "records_out": len(out)}, nil
}

// runValidate evaluates the dataset's real quality rules against the
// in-flight records, then re-checks any retained records that existed before
// this run began (append-mode loads must not launder already-stored dirty
// data through validation). The stage fails when the report is FAILED.
func (pe *PipelineEngine) runValidate(ctx context.Context, stage models.PipelineStage, p *models.DataPipeline, records []map[string]interface{}, runID, tenantID string) ([]map[string]interface{}, map[string]interface{}, error) {
	validateID := p.SourceDatasetID
	if validateID == "" {
		validateID = p.TargetDatasetID
	}
	if validateID == "" {
		return nil, nil, fmt.Errorf("validate stage requires a source or target dataset")
	}
	report, err := pe.quality.EvaluateDataset(ctx, validateID, runID, tenantID, records)
	if err != nil {
		return nil, nil, fmt.Errorf("quality evaluation failed: %w", err)
	}
	summary := map[string]interface{}{
		"dataset_id":     validateID,
		"quality_status": report.Status,
		"quality_score":  report.QualityScore,
		"rules_total":    report.TotalRules,
		"rules_failed":   report.FailedRules,
	}
	if report.Status == "FAILED" {
		return nil, summary, fmt.Errorf("quality validation failed: %d of %d rules violated", report.FailedRules, report.TotalRules)
	}
	return records, summary, nil
}

// runLoad writes the in-flight records into the target dataset's real
// storage, supports APPEND (default) and REPLACE modes, and syncs the
// dataset catalog counters with what is actually stored.
func (pe *PipelineEngine) runLoad(ctx context.Context, stage models.PipelineStage, p *models.DataPipeline, records []map[string]interface{}) ([]map[string]interface{}, map[string]interface{}, error) {
	if p.TargetDatasetID == "" {
		return nil, nil, fmt.Errorf("pipeline has no target_dataset_id configured")
	}
	ds, err := pe.store.GetDatasetByID(ctx, p.TargetDatasetID)
	if err != nil {
		return nil, nil, fmt.Errorf("target dataset %s not found: %w", p.TargetDatasetID, err)
	}

	mode := "append"
	if v, ok := stage.Config["mode"].(string); ok && v == "replace" {
		mode = "replace"
		deleted, err := pe.store.DeleteDatasetRecords(ctx, p.TargetDatasetID)
		if err != nil {
			return nil, nil, fmt.Errorf("replace-mode cleanup failed: %w", err)
		}
		ds.Metadata = map[string]interface{}{"replace_deleted_rows": deleted}
	}

	bytesWritten, err := pe.store.AppendDatasetRecords(ctx, p.TargetDatasetID, p.TenantID, records)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load records: %w", err)
	}

	count, err := pe.store.CountDatasetRecords(ctx, p.TargetDatasetID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify loaded row count: %w", err)
	}
	ds.RowCount = count
	ds.SizeBytes += bytesWritten
	ds.UpdatedAt = time.Now().UTC()
	if err := pe.store.UpdateDataset(ctx, ds); err != nil {
		return nil, nil, fmt.Errorf("failed to update dataset catalog entry: %w", err)
	}

	return records, map[string]interface{}{
		"target_dataset_id": p.TargetDatasetID,
		"mode":              mode,
		"records_loaded":    len(records),
		"bytes_written":     bytesWritten,
		"total_rows":        count,
	}, nil
}