package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"aiengines/pkg/model"
)

// ─── Digital Twins (P22) ─────────────────────────────────────────────────────

func CreateTwin(ctx context.Context, t *model.DigitalTwin) error {
	t.ID = newID()
	t.CreatedAt = nowT()
	t.UpdatedAt = t.CreatedAt
	if t.Status == "" {
		t.Status = "active"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO digital_twins (id, tenant_id, name, description, entity_type, entity_id, parameters, state, status, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		t.ID, t.TenantID, t.Name, t.Description, t.EntityType, t.EntityID, jsonB(t.Parameters), jsonB(t.State), t.Status, t.CreatedBy, t.CreatedAt, t.UpdatedAt)
	return err
}

func scanTwin(row interface{ Scan(...interface{}) error }) (*model.DigitalTwin, error) {
	var t model.DigitalTwin
	var params, state []byte
	if err := row.Scan(&t.ID, &t.TenantID, &t.Name, &t.Description, &t.EntityType, &t.EntityID, &params, &state, &t.Status, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	_ = jsonUnmarshal(params, &t.Parameters)
	_ = jsonUnmarshal(state, &t.State)
	return &t, nil
}

const twinCols = `id, tenant_id, name, description, entity_type, entity_id, parameters, state, status, created_by, created_at, updated_at`

func ListTwins(ctx context.Context, tenantID, q string) ([]model.DigitalTwin, error) {
	query := `SELECT ` + twinCols + ` FROM digital_twins WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if q != "" {
		query += ` AND (lower(name) LIKE $2 OR lower(description) LIKE $2)`
		args = append(args, "%"+lower(q)+"%")
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DigitalTwin
	for rows.Next() {
		t, err := scanTwin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func GetTwin(ctx context.Context, id string) (*model.DigitalTwin, error) {
	return scanTwin(db.QueryRowContext(ctx, `SELECT `+twinCols+` FROM digital_twins WHERE id=$1`, id))
}

// TwinExists reports whether a twin already tracks the given source entity.
func TwinExists(ctx context.Context, tenantID, entityType, entityID string) (bool, error) {
	var one int
	err := db.QueryRowContext(ctx,
		`SELECT 1 FROM digital_twins WHERE tenant_id=$1 AND entity_type=$2 AND entity_id=$3 LIMIT 1`,
		tenantID, entityType, entityID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return one == 1, nil
}

func UpdateTwin(ctx context.Context, t *model.DigitalTwin) error {
	t.UpdatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		UPDATE digital_twins SET name=$2, description=$3, entity_type=$4, entity_id=$5, parameters=$6, state=$7, status=$8, updated_at=$9
		WHERE id=$1`,
		t.ID, t.Name, t.Description, t.EntityType, t.EntityID, jsonB(t.Parameters), jsonB(t.State), t.Status, t.UpdatedAt)
	return err
}

func DeleteTwin(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM digital_twins WHERE id=$1`, id)
	return err
}

// ─── Simulations (P22) ───────────────────────────────────────────────────────

func CreateSimulation(ctx context.Context, s *model.Simulation) error {
	s.ID = newID()
	s.CreatedAt = nowT()
	if s.Status == "" {
		s.Status = "queued"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO simulations (id, tenant_id, twin_id, name, scenario, input, results, status, error, started_at, finished_at, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		s.ID, s.TenantID, s.TwinID, s.Name, jsonB(s.Scenario), jsonB(s.Input), jsonB(s.Results), s.Status, s.Error, s.StartedAt, s.FinishedAt, s.CreatedBy, s.CreatedAt)
	return err
}

func scanSim(row interface{ Scan(...interface{}) error }) (*model.Simulation, error) {
	var s model.Simulation
	var scenario, input, results []byte
	var start, finish sql.NullTime
	if err := row.Scan(&s.ID, &s.TenantID, &s.TwinID, &s.Name, &scenario, &input, &results, &s.Status, &s.Error, &start, &finish, &s.CreatedBy, &s.CreatedAt); err != nil {
		return nil, err
	}
	_ = jsonUnmarshal(scenario, &s.Scenario)
	_ = jsonUnmarshal(input, &s.Input)
	_ = jsonUnmarshal(results, &s.Results)
	if start.Valid {
		s.StartedAt = &start.Time
	}
	if finish.Valid {
		s.FinishedAt = &finish.Time
	}
	return &s, nil
}

const simCols = `id, tenant_id, twin_id, name, scenario, input, results, status, error, started_at, finished_at, created_by, created_at`

func ListSimulations(ctx context.Context, twinID string) ([]model.Simulation, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+simCols+` FROM simulations WHERE twin_id=$1 ORDER BY created_at DESC`, twinID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Simulation
	for rows.Next() {
		s, err := scanSim(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func GetSimulation(ctx context.Context, id string) (*model.Simulation, error) {
	return scanSim(db.QueryRowContext(ctx, `SELECT `+simCols+` FROM simulations WHERE id=$1`, id))
}

// CompleteSimulation marks a simulation finished with results (or error).
func CompleteSimulation(ctx context.Context, id, status string, results map[string]interface{}, errMsg string) error {
	fin := nowT()
	_, err := db.ExecContext(ctx, `
		UPDATE simulations SET status=$2, results=$3, error=$4, finished_at=$5 WHERE id=$1`,
		id, status, jsonB(results), sB(errMsg), fin)
	return err
}

func sB(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func lower(s string) string {
	return strings.ToLower(s)
}

func jsonUnmarshal(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}