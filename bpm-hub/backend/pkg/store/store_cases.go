package store

import (
	"context"
	"database/sql"

	"bpmhub/pkg/model"
)

// ─── Cases & Case Items (P48) ────────────────────────────────────────────────

func CreateCase(ctx context.Context, c *model.CaseInstance) error {
	c.ID = newID()
	c.CreatedAt = nowT()
	c.UpdatedAt = c.CreatedAt
	if c.Status == "" {
		c.Status = "open"
	}
	if c.Priority == "" {
		c.Priority = "medium"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO case_instances (id, tenant_id, key, name, status, priority, owner, due_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		c.ID, c.TenantID, c.Key, c.Name, c.Status, c.Priority, c.Owner, c.DueAt, c.CreatedAt, c.UpdatedAt)
	return err
}

const caseCols = `id, tenant_id, key, name, status, priority, owner, due_at, created_at, updated_at`

func scanCase(row row) (*model.CaseInstance, error) {
	var c model.CaseInstance
	var due sql.NullTime
	if err := row.Scan(&c.ID, &c.TenantID, &c.Key, &c.Name, &c.Status, &c.Priority, &c.Owner, &due, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	if due.Valid {
		c.DueAt = &due.Time
	}
	return &c, nil
}

func ListCases(ctx context.Context, tenantID, status string) ([]model.CaseInstance, error) {
	query := `SELECT ` + caseCols + ` FROM case_instances WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if status != "" {
		query += ` AND status=$2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.CaseInstance
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func GetCase(ctx context.Context, id string) (*model.CaseInstance, error) {
	return scanCase(db.QueryRowContext(ctx, `SELECT `+caseCols+` FROM case_instances WHERE id=$1`, id))
}

func UpdateCaseStatus(ctx context.Context, id, status string) error {
	_, err := db.ExecContext(ctx, `UPDATE case_instances SET status=$2, updated_at=$3 WHERE id=$1`, id, status, nowT())
	return err
}

func DeleteCase(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM case_instances WHERE id=$1`, id)
	return err
}

func AddCaseItem(ctx context.Context, ci *model.CaseItem) error {
	ci.ID = newID()
	ci.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO case_items (id, tenant_id, case_id, item_type, content, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		ci.ID, ci.TenantID, ci.CaseID, ci.ItemType, jsonB(ci.Content), ci.CreatedAt)
	return err
}

func ListCaseItems(ctx context.Context, caseID string) ([]model.CaseItem, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, case_id, item_type, content, created_at FROM case_items
		WHERE case_id=$1 ORDER BY created_at DESC`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.CaseItem
	for rows.Next() {
		var ci model.CaseItem
		var content []byte
		if err := rows.Scan(&ci.ID, &ci.TenantID, &ci.CaseID, &ci.ItemType, &content, &ci.CreatedAt); err != nil {
			return nil, err
		}
		jsonUnmarshal(content, &ci.Content)
		out = append(out, ci)
	}
	return out, rows.Err()
}

// ─── Work Items / Task management (P48) ─────────────────────────────────────

func CreateWorkItem(ctx context.Context, w *model.WorkItem) error {
	w.ID = newID()
	w.CreatedAt = nowT()
	w.UpdatedAt = w.CreatedAt
	if w.Status == "" {
		w.Status = "todo"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO work_items (id, tenant_id, instance_id, node_id, name, assignee, status, payload, due_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		w.ID, w.TenantID, w.InstanceID, w.NodeID, w.Name, w.Assignee, w.Status, jsonB(w.Payload), w.DueAt, w.CreatedAt, w.UpdatedAt)
	return err
}

const wiCols = `id, tenant_id, instance_id, node_id, name, assignee, status, payload, due_at, created_at, updated_at`

func scanWorkItem(row row) (*model.WorkItem, error) {
	var w model.WorkItem
	var payload []byte
	var due sql.NullTime
	if err := row.Scan(&w.ID, &w.TenantID, &w.InstanceID, &w.NodeID, &w.Name, &w.Assignee, &w.Status, &payload, &due, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return nil, err
	}
	jsonUnmarshal(payload, &w.Payload)
	if due.Valid {
		w.DueAt = &due.Time
	}
	return &w, nil
}

func ListWorkItems(ctx context.Context, tenantID, assignee, status string) ([]model.WorkItem, error) {
	query := `SELECT ` + wiCols + ` FROM work_items WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	argc := 1
	if assignee != "" {
		argc++
		args = append(args, assignee)
		query += ` AND assignee=$` + itoa(argc)
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
	var out []model.WorkItem
	for rows.Next() {
		w, err := scanWorkItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

func GetWorkItem(ctx context.Context, id string) (*model.WorkItem, error) {
	return scanWorkItem(db.QueryRowContext(ctx, `SELECT `+wiCols+` FROM work_items WHERE id=$1`, id))
}

func UpdateWorkItem(ctx context.Context, w *model.WorkItem) error {
	w.UpdatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		UPDATE work_items SET name=$2, assignee=$3, status=$4, payload=$5, due_at=$6, updated_at=$7
		WHERE id=$1`,
		w.ID, w.Name, w.Assignee, w.Status, jsonB(w.Payload), w.DueAt, w.UpdatedAt)
	return err
}

// ListWorkItemsByInstance returns tasks belonging to a process instance.
func ListWorkItemsByInstance(ctx context.Context, instanceID string) ([]model.WorkItem, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+wiCols+` FROM work_items WHERE instance_id=$1 ORDER BY created_at`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.WorkItem
	for rows.Next() {
		w, err := scanWorkItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

// ─── Automation service (P48) ────────────────────────────────────────────────

func CreateAutomationRule(ctx context.Context, r *model.AutomationRule) error {
	r.ID = newID()
	r.CreatedAt = nowT()
	r.Enabled = true
	_, err := db.ExecContext(ctx, `
		INSERT INTO automation_rules (id, tenant_id, name, trigger_event, condition, actions, enabled, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		r.ID, r.TenantID, r.Name, r.TriggerEvent, jsonB(r.Condition), jsonB(r.Actions), r.Enabled, r.CreatedAt)
	return err
}

const ruleCols = `id, tenant_id, name, trigger_event, condition, actions, enabled, created_at`

func ListAutomationRules(ctx context.Context, tenantID string) ([]model.AutomationRule, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+ruleCols+` FROM automation_rules WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AutomationRule
	for rows.Next() {
		var r model.AutomationRule
		var cond, act []byte
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.TriggerEvent, &cond, &act, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		jsonUnmarshal(cond, &r.Condition)
		jsonUnmarshal(act, &r.Actions)
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetAutomationRuleEnabled toggles a rule.
func SetAutomationRuleEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := db.ExecContext(ctx, `UPDATE automation_rules SET enabled=$2 WHERE id=$1`, id, enabled)
	return err
}

// FindAutomationRulesForEvent returns enabled rules matching an event type.
func FindAutomationRulesForEvent(ctx context.Context, tenantID, eventType string) ([]model.AutomationRule, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+ruleCols+` FROM automation_rules WHERE tenant_id=$1 AND trigger_event=$2 AND enabled=true`, tenantID, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AutomationRule
	for rows.Next() {
		var r model.AutomationRule
		var cond, act []byte
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.TriggerEvent, &cond, &act, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		jsonUnmarshal(cond, &r.Condition)
		jsonUnmarshal(act, &r.Actions)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ─── Process mining / operational analytics (P48) ─────────────────────────────

func MiningSummary(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	var total, completed, running int
	var avgCycle sql.NullFloat64
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM process_instances WHERE tenant_id=$1`, tenantID).Scan(&total)
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM process_instances WHERE tenant_id=$1 AND status='completed'`, tenantID).Scan(&completed)
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM process_instances WHERE tenant_id=$1 AND status IN ('created','running')`, tenantID).Scan(&running)
	_ = db.QueryRowContext(ctx, `SELECT avg(EXTRACT(EPOCH FROM (ended_at - started_at))) FROM process_instances WHERE tenant_id=$1 AND ended_at IS NOT NULL`, tenantID).Scan(&avgCycle)
	avg := float64(0)
	if avgCycle.Valid {
		avg = avgCycle.Float64
	}
	return map[string]interface{}{
		"total_instances":     total,
		"completed_instances": completed,
		"running_instances":   running,
		"avg_cycle_seconds":   int(avg),
	}, nil
}

// ─── Object linkage (cross-app contract) ─────────────────────────────────────

func CreateObjectLink(ctx context.Context, l *model.ObjectLink) error {
	l.ID = newID()
	l.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO object_links (id, tenant_id, source_type, source_id, target_type, target_id, relationship, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		l.ID, l.TenantID, l.SourceType, l.SourceID, l.TargetType, l.TargetID, l.Relationship, l.CreatedAt)
	return err
}

func ListObjectLinks(ctx context.Context, objectType, objectID string) ([]model.ObjectLink, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, source_type, source_id, target_type, target_id, relationship, created_at
		FROM object_links
		WHERE (source_type=$1 AND source_id=$2) OR (target_type=$1 AND target_id=$2)
		ORDER BY created_at DESC`, objectType, objectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ObjectLink
	for rows.Next() {
		var l model.ObjectLink
		if err := rows.Scan(&l.ID, &l.TenantID, &l.SourceType, &l.SourceID, &l.TargetType, &l.TargetID, &l.Relationship, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}