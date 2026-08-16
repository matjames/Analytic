package store

import (
	"context"
	"strconv"

	"aiengines/pkg/model"
)

type row interface{ Scan(...interface{}) error }

func itoa(n int) string { return strconv.Itoa(n) }

// ─── AI Models (P22) ─────────────────────────────────────────────────────────

func CreateModel(ctx context.Context, m *model.AIModel) error {
	m.ID = newID()
	m.CreatedAt = nowT()
	m.UpdatedAt = m.CreatedAt
	if m.Status == "" {
		m.Status = "draft"
	}
	if m.Version == "" {
		m.Version = "1.0"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO ai_models (id, tenant_id, name, kind, version, config, status, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		m.ID, m.TenantID, m.Name, m.Kind, m.Version, jsonB(m.Config), m.Status, m.CreatedBy, m.CreatedAt, m.UpdatedAt)
	return err
}

func scanModel(row row) (*model.AIModel, error) {
	var m model.AIModel
	var cfg []byte
	if err := row.Scan(&m.ID, &m.TenantID, &m.Name, &m.Kind, &m.Version, &cfg, &m.Status, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	_ = jsonUnmarshal(cfg, &m.Config)
	return &m, nil
}

const modelCols = `id, tenant_id, name, kind, version, config, status, created_by, created_at, updated_at`

func ListModels(ctx context.Context, tenantID string) ([]model.AIModel, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+modelCols+` FROM ai_models WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AIModel
	for rows.Next() {
		m, err := scanModel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func GetModel(ctx context.Context, id string) (*model.AIModel, error) {
	return scanModel(db.QueryRowContext(ctx, `SELECT `+modelCols+` FROM ai_models WHERE id=$1`, id))
}

func DeleteModel(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM ai_models WHERE id=$1`, id)
	return err
}

// ─── Predictions (P22) ───────────────────────────────────────────────────────

func CreatePrediction(ctx context.Context, p *model.Prediction) error {
	p.ID = newID()
	p.CreatedAt = nowT()
	if p.Status == "" {
		p.Status = "in_progress"
	}
	if p.TriggeredBy == "" {
		p.TriggeredBy = "manual"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO predictions (id, tenant_id, model_id, target_type, target_id, fields, prediction, confidence, horizon, status, triggered_by, source_event, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		p.ID, p.TenantID, p.ModelID, p.TargetType, p.TargetID, jsonB(p.Fields), jsonB(p.Prediction), p.Confidence, p.Horizon, p.Status, p.TriggeredBy, p.SourceEvent, p.CreatedBy, p.CreatedAt)
	return err
}

func scanPred(row row) (*model.Prediction, error) {
	var p model.Prediction
	var fields, pred []byte
	if err := row.Scan(&p.ID, &p.TenantID, &p.ModelID, &p.TargetType, &p.TargetID, &fields, &pred, &p.Confidence, &p.Horizon, &p.Status, &p.TriggeredBy, &p.SourceEvent, &p.CreatedBy, &p.CreatedAt); err != nil {
		return nil, err
	}
	_ = jsonUnmarshal(fields, &p.Fields)
	_ = jsonUnmarshal(pred, &p.Prediction)
	return &p, nil
}

const predCols = `id, tenant_id, model_id, target_type, target_id, fields, prediction, confidence, horizon, status, triggered_by, source_event, created_by, created_at`

func ListPredictions(ctx context.Context, tenantID, modelID, status string) ([]model.Prediction, error) {
	query := `SELECT ` + predCols + ` FROM predictions WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	argc := 1
	if modelID != "" {
		argc++
		args = append(args, modelID)
		query += ` AND model_id=$` + itoa(argc)
	}
	if status != "" {
		argc++
		args = append(args, status)
		query += ` AND status=$` + itoa(argc)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Prediction
	for rows.Next() {
		p, err := scanPred(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func GetPrediction(ctx context.Context, id string) (*model.Prediction, error) {
	return scanPred(db.QueryRowContext(ctx, `SELECT `+predCols+` FROM predictions WHERE id=$1`, id))
}

// CompletePrediction marks a prediction completed with a computed result.
func CompletePrediction(ctx context.Context, id, status string, result map[string]interface{}) error {
	_, err := db.ExecContext(ctx, `UPDATE predictions SET status=$2, prediction=$3 WHERE id=$1`,
		id, status, jsonB(result))
	return err
}

// ─── Decision Pipelines (P22) ────────────────────────────────────────────────

func CreatePipeline(ctx context.Context, p *model.DecisionPipeline) error {
	p.ID = newID()
	p.CreatedAt = nowT()
	if p.Status == "" {
		p.Status = "active"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO pipelines (id, tenant_id, name, description, definition, status, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.ID, p.TenantID, p.Name, p.Description, jsonB(p.Definition), p.Status, p.CreatedBy, p.CreatedAt)
	return err
}

func scanPipeline(row row) (*model.DecisionPipeline, error) {
	var p model.DecisionPipeline
	var def []byte
	if err := row.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &def, &p.Status, &p.CreatedBy, &p.CreatedAt); err != nil {
		return nil, err
	}
	_ = jsonUnmarshal(def, &p.Definition)
	return &p, nil
}

const pipelineCols = `id, tenant_id, name, description, definition, status, created_by, created_at`

func ListPipelines(ctx context.Context, tenantID string) ([]model.DecisionPipeline, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+pipelineCols+` FROM pipelines WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DecisionPipeline
	for rows.Next() {
		p, err := scanPipeline(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func GetPipeline(ctx context.Context, id string) (*model.DecisionPipeline, error) {
	return scanPipeline(db.QueryRowContext(ctx, `SELECT `+pipelineCols+` FROM pipelines WHERE id=$1`, id))
}