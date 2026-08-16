package store

import (
	"context"
	"database/sql"
	"time"

	"learningcrm/pkg/model"
)

// ─── Stakeholders (P35) ──────────────────────────────────────────────────────

func CreateStakeholder(ctx context.Context, s *model.Stakeholder) error {
	s.ID = newID()
	s.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO stakeholders (id, tenant_id, name, kind, tier, engagement_level, contact_email, notes, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		s.ID, s.TenantID, s.Name, s.Kind, s.Tier, s.EngagementLevel, s.ContactEmail, s.Notes, s.CreatedAt)
	return err
}

const stakeholderCols = `id, tenant_id, name, kind, tier, engagement_level, contact_email, notes, created_at`

func ListStakeholders(ctx context.Context, tenantID string) ([]model.Stakeholder, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+stakeholderCols+` FROM stakeholders WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Stakeholder
	for rows.Next() {
		var s model.Stakeholder
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Kind, &s.Tier, &s.EngagementLevel, &s.ContactEmail, &s.Notes, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func GetStakeholder(ctx context.Context, id string) (*model.Stakeholder, error) {
	var s model.Stakeholder
	err := db.QueryRowContext(ctx, `SELECT `+stakeholderCols+` FROM stakeholders WHERE id=$1`, id).
		Scan(&s.ID, &s.TenantID, &s.Name, &s.Kind, &s.Tier, &s.EngagementLevel, &s.ContactEmail, &s.Notes, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func DeleteStakeholder(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM stakeholders WHERE id=$1`, id)
	return err
}

// ─── Engagements (P35) ─────────────────────────────────────────────────────────

func CreateEngagement(ctx context.Context, e *model.Engagement) error {
	e.ID = newID()
	e.CreatedAt = nowT()
	if e.HappenedAt.IsZero() {
		e.HappenedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO engagements (id, tenant_id, stakeholder_id, kind, title, happened_at, notes, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		e.ID, e.TenantID, e.StakeholderID, e.Kind, e.Title, e.HappenedAt, e.Notes, e.CreatedAt)
	return err
}

func ListEngagements(ctx context.Context, tenantID string) ([]model.Engagement, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, stakeholder_id, kind, title, happened_at, notes, created_at
		FROM engagements WHERE tenant_id=$1 ORDER BY happened_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Engagement
	for rows.Next() {
		var e model.Engagement
		if err := rows.Scan(&e.ID, &e.TenantID, &e.StakeholderID, &e.Kind, &e.Title, &e.HappenedAt, &e.Notes, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ─── Stewardship Actions (P35) ────────────────────────────────────────────────

func CreateStewardshipAction(ctx context.Context, a *model.StewardshipAction) error {
	a.ID = newID()
	a.CreatedAt = nowT()
	a.UpdatedAt = a.CreatedAt
	if a.Status == "" {
		a.Status = "open"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO stewardship_actions (id, tenant_id, action, entity_type, entity_id, owner, status, due_at, updated_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		a.ID, a.TenantID, a.Action, a.EntityType, a.EntityID, a.Owner, a.Status, a.DueAt, a.UpdatedAt, a.CreatedAt)
	return err
}

const stewardCols = `id, tenant_id, action, entity_type, entity_id, owner, status, due_at, updated_at, created_at`

func GetStewardshipAction(ctx context.Context, id string) (*model.StewardshipAction, error) {
	var a model.StewardshipAction
	var due sql.NullTime
	err := db.QueryRowContext(ctx, `SELECT `+stewardCols+` FROM stewardship_actions WHERE id=$1`, id).
		Scan(&a.ID, &a.TenantID, &a.Action, &a.EntityType, &a.EntityID, &a.Owner, &a.Status, &due, &a.UpdatedAt, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if due.Valid {
		a.DueAt = &due.Time
	}
	return &a, nil
}

func ListStewardshipActions(ctx context.Context, tenantID, status string) ([]model.StewardshipAction, error) {
	query := `SELECT ` + stewardCols + ` FROM stewardship_actions WHERE tenant_id=$1`
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
	var out []model.StewardshipAction
	for rows.Next() {
		var a model.StewardshipAction
		var due sql.NullTime
		if err := rows.Scan(&a.ID, &a.TenantID, &a.Action, &a.EntityType, &a.EntityID, &a.Owner, &a.Status, &due, &a.UpdatedAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		if due.Valid {
			a.DueAt = &due.Time
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func UpdateStewardshipAction(ctx context.Context, a *model.StewardshipAction) error {
	a.UpdatedAt = time.Now().UTC()
	_, err := db.ExecContext(ctx, `
		UPDATE stewardship_actions SET action=$2, entity_type=$3, entity_id=$4, owner=$5, status=$6, due_at=$7, updated_at=$8
		WHERE id=$1`,
		a.ID, a.Action, a.EntityType, a.EntityID, a.Owner, a.Status, a.DueAt, a.UpdatedAt)
	return err
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