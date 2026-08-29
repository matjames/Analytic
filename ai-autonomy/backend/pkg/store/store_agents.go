package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"aiengines/pkg/model"
)

// ─── Agents (P31) ────────────────────────────────────────────────────────────

func CreateAgent(ctx context.Context, a *model.Agent) error {
	a.ID = newID()
	a.CreatedAt = nowT()
	a.UpdatedAt = a.CreatedAt
	if a.Status == "" {
		a.Status = "registered"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO agents (id, tenant_id, workspace_id, name, role, persona, capabilities, safety_policy, status, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		a.ID, a.TenantID, a.WorkspaceID, a.Name, a.Role, jsonB(a.Persona), jsonB(a.Capabilities), jsonB(a.SafetyPolicy), a.Status, a.CreatedBy, a.CreatedAt, a.UpdatedAt)
	return err
}

func scanAgent(row row) (*model.Agent, error) {
	var a model.Agent
	var persona, cap, policy []byte
	if err := row.Scan(&a.ID, &a.TenantID, &a.WorkspaceID, &a.Name, &a.Role, &persona, &cap, &policy, &a.Status, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	_ = jsonUnmarshal(persona, &a.Persona)
	_ = jsonUnmarshal(cap, &a.Capabilities)
	_ = jsonUnmarshal(policy, &a.SafetyPolicy)
	return &a, nil
}

const agentCols = `id, tenant_id, name, role, persona, capabilities, safety_policy, status, created_by, COALESCE(workspace_id, ''), created_at, updated_at`

func ListAgents(ctx context.Context, tenantID, status, workspaceID string) ([]model.Agent, error) {
	query := `SELECT ` + agentCols + ` FROM agents WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if workspaceID != "" {
		query += ` AND (workspace_id=$2 OR workspace_id='')`
		args = append(args, workspaceID)
	}
	if status != "" {
		query += fmt.Sprintf(` AND status=$%d`, len(args)+1)
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func GetAgent(ctx context.Context, id string) (*model.Agent, error) {
	return scanAgent(db.QueryRowContext(ctx, `SELECT `+agentCols+` FROM agents WHERE id=$1`, id))
}

// FindActiveAgentByRole returns the most recent active agent with the given role.
func FindActiveAgentByRole(ctx context.Context, tenantID, role string) (*model.Agent, error) {
	return scanAgent(db.QueryRowContext(ctx,
		`SELECT `+agentCols+` FROM agents WHERE tenant_id=$1 AND role=$2 AND status='active' ORDER BY created_at DESC LIMIT 1`,
		tenantID, role))
}

func UpdateAgent(ctx context.Context, a *model.Agent) error {
	a.UpdatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		UPDATE agents SET name=$2, role=$3, persona=$4, capabilities=$5, safety_policy=$6, status=$7, updated_at=$8
		WHERE id=$1`,
		a.ID, a.Name, a.Role, jsonB(a.Persona), jsonB(a.Capabilities), jsonB(a.SafetyPolicy), a.Status, a.UpdatedAt)
	return err
}

// ─── Agent Tasks (P31) ───────────────────────────────────────────────────────

func CreateTask(ctx context.Context, t *model.AgentTask) error {
	t.ID = newID()
	t.CreatedAt = nowT()
	if t.Status == "" {
		t.Status = "queued"
	}
	if t.TriggerType == "" {
		t.TriggerType = "manual"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO agent_tasks (id, tenant_id, agent_id, name, task_type, payload, trigger_type, triggered_by, status, result, error, created_at, started_at, finished_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		t.ID, t.TenantID, t.AgentID, t.Name, t.TaskType, jsonB(t.Payload), t.TriggerType, t.TriggeredBy, t.Status, jsonB(t.Result), t.Error, t.CreatedAt, t.StartedAt, t.FinishedAt)
	return err
}

func scanTask(row row) (*model.AgentTask, error) {
	var t model.AgentTask
	var payload, result []byte
	var start, finish sql.NullTime
	if err := row.Scan(&t.ID, &t.TenantID, &t.AgentID, &t.Name, &t.TaskType, &payload, &t.TriggerType, &t.TriggeredBy, &t.Status, &result, &t.Error, &t.CreatedAt, &start, &finish); err != nil {
		return nil, err
	}
	_ = jsonUnmarshal(payload, &t.Payload)
	_ = jsonUnmarshal(result, &t.Result)
	if start.Valid {
		t.StartedAt = &start.Time
	}
	if finish.Valid {
		t.FinishedAt = &finish.Time
	}
	return &t, nil
}

const taskCols = `id, tenant_id, agent_id, name, task_type, payload, trigger_type, triggered_by, status, result, error, created_at, started_at, finished_at`

func ListTasks(ctx context.Context, agentID, status string) ([]model.AgentTask, error) {
	query := `SELECT ` + taskCols + ` FROM agent_tasks WHERE agent_id=$1`
	args := []interface{}{agentID}
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
	var out []model.AgentTask
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func GetTask(ctx context.Context, id string) (*model.AgentTask, error) {
	return scanTask(db.QueryRowContext(ctx, `SELECT `+taskCols+` FROM agent_tasks WHERE id=$1`, id))
}

// CompleteTask marks a task finished with results (or error).
func CompleteTask(ctx context.Context, id, status string, result map[string]interface{}, errMsg string) error {
	fin := time.Now().UTC()
	_, err := db.ExecContext(ctx, `
		UPDATE agent_tasks SET status=$2, result=$3, error=$4, finished_at=$5 WHERE id=$1`,
		id, status, jsonB(result), sB(errMsg), fin)
	return err
}

// StartTask marks a task running.
func StartTask(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `UPDATE agent_tasks SET status='running', started_at=$2 WHERE id=$1`, id, now)
	return err
}

// ─── Agent Memory (P31) ──────────────────────────────────────────────────────

func CreateMemory(ctx context.Context, m *model.AgentMemory) error {
	m.ID = newID()
	m.CreatedAt = nowT()
	if m.Kind == "" {
		m.Kind = "observation"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO agent_memory (id, tenant_id, agent_id, kind, content, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		m.ID, m.TenantID, m.AgentID, m.Kind, jsonB(m.Content), m.CreatedAt)
	return err
}

const memoryCols = `id, tenant_id, agent_id, kind, content, created_at`

func ListMemory(ctx context.Context, agentID string) ([]model.AgentMemory, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+memoryCols+` FROM agent_memory WHERE agent_id=$1 ORDER BY created_at DESC`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AgentMemory
	for rows.Next() {
		var m model.AgentMemory
		var content []byte
		if err := rows.Scan(&m.ID, &m.TenantID, &m.AgentID, &m.Kind, &content, &m.CreatedAt); err != nil {
			return nil, err
		}
		_ = jsonUnmarshal(content, &m.Content)
		out = append(out, m)
	}
	return out, rows.Err()
}

// ─── Agent Communications Bus (P31) ──────────────────────────────────────────

func CreateMessage(ctx context.Context, m *model.AgentMessage) error {
	m.ID = newID()
	m.CreatedAt = nowT()
	if m.Type == "" {
		m.Type = "inform"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO agent_messages (id, tenant_id, from_agent, to_agent, type, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		m.ID, m.TenantID, m.FromAgent, m.ToAgent, m.Type, jsonB(m.Payload), m.CreatedAt)
	return err
}

func ListMessages(ctx context.Context, tenantID, agentID string) ([]model.AgentMessage, error) {
	query := `SELECT id, tenant_id, from_agent, to_agent, type, payload, created_at FROM agent_messages WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if agentID != "" {
		query += ` AND (from_agent=$2 OR to_agent=$2)`
		args = append(args, agentID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AgentMessage
	for rows.Next() {
		var m model.AgentMessage
		var payload []byte
		if err := rows.Scan(&m.ID, &m.TenantID, &m.FromAgent, &m.ToAgent, &m.Type, &payload, &m.CreatedAt); err != nil {
			return nil, err
		}
		_ = jsonUnmarshal(payload, &m.Payload)
		out = append(out, m)
	}
	return out, rows.Err()
}

// ─── AI Safety & Governance Guardrails (P31) ─────────────────────────────────

func CreateGovernanceAction(ctx context.Context, g *model.GovernanceAction) error {
	g.ID = newID()
	g.CreatedAt = nowT()
	if g.Verdict == "" {
		g.Verdict = "auto_approved"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO governance_actions (id, tenant_id, agent_id, action, decision, risk_score, verdict, reviewed_by, reviewed_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		g.ID, g.TenantID, g.AgentID, g.Action, jsonB(g.Decision), g.RiskScore, g.Verdict, g.ReviewedBy, g.ReviewedAt, g.CreatedAt)
	return err
}

const govCols = `id, tenant_id, agent_id, action, decision, risk_score, verdict, reviewed_by, reviewed_at, created_at`

func ListGovernance(ctx context.Context, tenantID, verdict string) ([]model.GovernanceAction, error) {
	query := `SELECT ` + govCols + ` FROM governance_actions WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if verdict != "" {
		query += ` AND verdict=$2`
		args = append(args, verdict)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.GovernanceAction
	for rows.Next() {
		var g model.GovernanceAction
		var decision []byte
		var review sql.NullTime
		if err := rows.Scan(&g.ID, &g.TenantID, &g.AgentID, &g.Action, &decision, &g.RiskScore, &g.Verdict, &g.ReviewedBy, &review, &g.CreatedAt); err != nil {
			return nil, err
		}
		_ = jsonUnmarshal(decision, &g.Decision)
		if review.Valid {
			g.ReviewedAt = &review.Time
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// ReviewGovernance updates the verdict of a governance action (human-in-the-loop).
func ReviewGovernance(ctx context.Context, id, verdict, reviewer string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE governance_actions SET verdict=$2, reviewed_by=$3, reviewed_at=$4 WHERE id=$1`,
		id, verdict, reviewer, time.Now().UTC())
	return err
}