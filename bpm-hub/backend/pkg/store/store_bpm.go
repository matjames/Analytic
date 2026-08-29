package store

import (
	"context"
	"database/sql"

	"bpmhub/pkg/model"
)

// ─── Process Definitions (P48) ───────────────────────────────────────────────

func CreateProcessDefinition(ctx context.Context, p *model.ProcessDefinition) error {
	p.ID = newID()
	p.CreatedAt = nowT()
	p.UpdatedAt = p.CreatedAt
	if p.Version == 0 {
		p.Version = 1
	}
	if p.Status == "" {
		p.Status = "draft"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO process_definitions (id, tenant_id, workspace_id, name, key, version, description, start_node, nodes, transitions, status, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		p.ID, p.TenantID, p.WorkspaceID, p.Name, p.Key, p.Version, p.Description, p.StartNode, jsonB(p.Nodes), jsonB(p.Transitions), p.Status, p.CreatedBy, p.CreatedAt, p.UpdatedAt)
	return err
}

func scanProcessDef(row row) (*model.ProcessDefinition, error) {
	var p model.ProcessDefinition
	var nodes, trans []byte
	if err := row.Scan(&p.ID, &p.TenantID, &p.WorkspaceID, &p.Name, &p.Key, &p.Version, &p.Description, &p.StartNode, &nodes, &trans, &p.Status, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	jsonUnmarshal(nodes, &p.Nodes)
	jsonUnmarshal(trans, &p.Transitions)
	return &p, nil
}

const pdCols = `id, tenant_id, name, key, version, description, start_node, nodes, transitions, status, created_by, COALESCE(workspace_id, ''), created_at, updated_at`

func ListProcessDefinitions(ctx context.Context, tenantID, workspaceID string) ([]model.ProcessDefinition, error) {
	query := `SELECT ` + pdCols + ` FROM process_definitions WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if workspaceID != "" {
		query += ` AND (workspace_id=$2 OR workspace_id='')`
		args = append(args, workspaceID)
	}
	rows, err := db.QueryContext(ctx, query+` ORDER BY key, version DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ProcessDefinition
	for rows.Next() {
		p, err := scanProcessDef(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func GetProcessDefinition(ctx context.Context, id string) (*model.ProcessDefinition, error) {
	return scanProcessDef(db.QueryRowContext(ctx, `SELECT `+pdCols+` FROM process_definitions WHERE id=$1`, id))
}

func UpdateProcessDefinitionStatus(ctx context.Context, id, status string) error {
	_, err := db.ExecContext(ctx, `UPDATE process_definitions SET status=$2, updated_at=$3 WHERE id=$1`, id, status, nowT())
	return err
}

// ─── Process Instances (P48) ─────────────────────────────────────────────────

func CreateInstance(ctx context.Context, in *model.ProcessInstance) error {
	in.ID = newID()
	in.CreatedAt = nowT()
	in.UpdatedAt = in.CreatedAt
	if in.Status == "" {
		in.Status = "created"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO process_instances (id, tenant_id, workspace_id, definition_id, status, current_node, context, started_at, ended_at, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		in.ID, in.TenantID, in.WorkspaceID, in.DefinitionID, in.Status, in.CurrentNode, jsonB(in.Context), in.StartedAt, in.EndedAt, in.CreatedBy, in.CreatedAt, in.UpdatedAt)
	return err
}

func scanInstance(row row) (*model.ProcessInstance, error) {
	var in model.ProcessInstance
	var context []byte
	var start, end sql.NullTime
	if err := row.Scan(&in.ID, &in.TenantID, &in.WorkspaceID, &in.DefinitionID, &in.Status, &in.CurrentNode, &context, &start, &end, &in.CreatedBy, &in.CreatedAt, &in.UpdatedAt); err != nil {
		return nil, err
	}
	jsonUnmarshal(context, &in.Context)
	if start.Valid {
		in.StartedAt = &start.Time
	}
	if end.Valid {
		in.EndedAt = &end.Time
	}
	return &in, nil
}

const instCols = `id, tenant_id, definition_id, status, current_node, context, started_at, ended_at, created_by, COALESCE(workspace_id, ''), created_at, updated_at`

func ListInstances(ctx context.Context, tenantID, definitionID, status, workspaceID string) ([]model.ProcessInstance, error) {
	query := `SELECT ` + instCols + ` FROM process_instances WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	argc := 1
	if workspaceID != "" {
		argc++
		args = append(args, workspaceID)
		query += ` AND (workspace_id=$` + itoa(argc) + ` OR workspace_id='')`
	}
	if definitionID != "" {
		argc++
		args = append(args, definitionID)
		query += ` AND definition_id=$` + itoa(argc)
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
	var out []model.ProcessInstance
	for rows.Next() {
		in, err := scanInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *in)
	}
	return out, rows.Err()
}

func GetInstance(ctx context.Context, id string) (*model.ProcessInstance, error) {
	return scanInstance(db.QueryRowContext(ctx, `SELECT `+instCols+` FROM process_instances WHERE id=$1`, id))
}

// AdvanceInstance sets the instance to a node and marks it running.
func AdvanceInstance(ctx context.Context, id, node string) error {
	_, err := db.ExecContext(ctx, `UPDATE process_instances SET current_node=$2, status='running', started_at=COALESCE(started_at, $3), updated_at=$3 WHERE id=$1`,
		id, node, nowT())
	return err
}

// CompleteInstance marks an instance finished.
func CompleteInstance(ctx context.Context, id, status string) error {
	now := nowT()
	_, err := db.ExecContext(ctx, `UPDATE process_instances SET status=$2, ended_at=$3, updated_at=$3 WHERE id=$1`, id, status, now)
	return err
}

// ─── Activity log (process mining source, P48) ───────────────────────────────

func LogActivity(ctx context.Context, a *model.ActivityLog) error {
	a.ID = newID()
	if a.At.IsZero() {
		a.At = nowT()
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO activity_logs (id, tenant_id, instance_id, case_id, action, node_id, actor, meta, at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.ID, a.TenantID, a.InstanceID, a.CaseID, a.Action, a.NodeID, a.Actor, jsonB(a.Meta), a.At)
	return err
}

func ListActivity(ctx context.Context, tenantID, instanceID string) ([]model.ActivityLog, error) {
	query := `SELECT id, tenant_id, instance_id, case_id, action, node_id, actor, meta, at FROM activity_logs WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if instanceID != "" {
		query += ` AND instance_id=$2`
		args = append(args, instanceID)
	}
	query += ` ORDER BY at ASC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ActivityLog
	for rows.Next() {
		var a model.ActivityLog
		var meta []byte
		if err := rows.Scan(&a.ID, &a.TenantID, &a.InstanceID, &a.CaseID, &a.Action, &a.NodeID, &a.Actor, &meta, &a.At); err != nil {
			return nil, err
		}
		jsonUnmarshal(meta, &a.Meta)
		out = append(out, a)
	}
	return out, rows.Err()
}