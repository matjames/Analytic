package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"statchat/pkg/model"
)

var (
	ErrWhiteboardForbidden = errors.New("whiteboard action forbidden")
	ErrWhiteboardConflict  = errors.New("whiteboard version conflict")
)

func ensureWhiteboardsSchema(ctx context.Context) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS collaboration_whiteboards (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  title TEXT NOT NULL,
  data TEXT NOT NULL DEFAULT '[]',
  created_by TEXT NOT NULL REFERENCES users(id),
  updated_by TEXT NOT NULL REFERENCES users(id),
  version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS collaboration_whiteboards_tenant_updated_idx ON collaboration_whiteboards (tenant_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS collaboration_whiteboard_members (
  whiteboard_id TEXT NOT NULL REFERENCES collaboration_whiteboards(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('owner','editor','viewer')),
  added_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (whiteboard_id, user_id)
);
CREATE INDEX IF NOT EXISTS collaboration_whiteboard_members_user_idx ON collaboration_whiteboard_members (tenant_id, user_id);

CREATE TABLE IF NOT EXISTS collaboration_whiteboard_revisions (
  id BIGSERIAL PRIMARY KEY,
  whiteboard_id TEXT NOT NULL REFERENCES collaboration_whiteboards(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  title TEXT NOT NULL,
  data TEXT NOT NULL,
  edited_by TEXT NOT NULL REFERENCES users(id),
  edited_at TIMESTAMPTZ NOT NULL,
  UNIQUE (whiteboard_id, version)
);
CREATE INDEX IF NOT EXISTS collaboration_whiteboard_revisions_board_idx ON collaboration_whiteboard_revisions (tenant_id, whiteboard_id, version DESC);
`)
	return err
}

func CreateCollaborationWhiteboard(board model.CollaborationWhiteboard) (model.CollaborationWhiteboard, error) {
	board.ID = uuid.NewString()
	board.TenantID = normalizedTenantID(board.TenantID)
	board.Title = strings.TrimSpace(board.Title)
	board.Data = strings.TrimSpace(board.Data)
	if board.Data == "" {
		board.Data = "[]"
	}
	board.Version = 1
	board.Role = "owner"
	board.CanEdit = true
	board.CreatedAt = time.Now().UTC()
	board.UpdatedAt = board.CreatedAt
	board.UpdatedBy = board.CreatedBy
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return board, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO collaboration_whiteboards (id,tenant_id,title,data,created_by,updated_by,version,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$5,1,$6,$6)`, board.ID, board.TenantID, board.Title, board.Data, board.CreatedBy, board.CreatedAt); err != nil {
		return board, err
	}
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO collaboration_whiteboard_members (whiteboard_id,tenant_id,user_id,role,added_at) VALUES ($1,$2,$3,'owner',$4)`, board.ID, board.TenantID, board.CreatedBy, board.CreatedAt); err != nil {
		return board, err
	}
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO collaboration_whiteboard_revisions (whiteboard_id,tenant_id,version,title,data,edited_by,edited_at) VALUES ($1,$2,1,$3,$4,$5,$6)`, board.ID, board.TenantID, board.Title, board.Data, board.CreatedBy, board.CreatedAt); err != nil {
		return board, err
	}
	return board, tx.Commit()
}

func collaborationWhiteboardRole(boardID, tenantID, userID string) (string, error) {
	var role string
	err := db.QueryRowContext(context.Background(), `SELECT wm.role FROM collaboration_whiteboard_members wm JOIN collaboration_whiteboards b ON b.id=wm.whiteboard_id AND b.tenant_id=wm.tenant_id WHERE wm.whiteboard_id=$1 AND wm.tenant_id=$2 AND wm.user_id=$3`, boardID, normalizedTenantID(tenantID), userID).Scan(&role)
	return role, err
}

func scanWhiteboard(scanner interface{ Scan(...any) error }) (model.CollaborationWhiteboard, error) {
	var board model.CollaborationWhiteboard
	err := scanner.Scan(&board.ID, &board.TenantID, &board.Title, &board.Data, &board.CreatedBy, &board.Author, &board.UpdatedBy, &board.Version, &board.CreatedAt, &board.UpdatedAt, &board.Role)
	board.CanEdit = board.Role == "owner" || board.Role == "editor"
	return board, err
}

func GetCollaborationWhiteboards(tenantID, userID string) ([]model.CollaborationWhiteboard, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT b.id,b.tenant_id,b.title,b.data,b.created_by,creator.name,b.updated_by,b.version,b.created_at,b.updated_at,wm.role FROM collaboration_whiteboards b JOIN collaboration_whiteboard_members wm ON wm.whiteboard_id=b.id AND wm.tenant_id=b.tenant_id AND wm.user_id=$2 JOIN users creator ON creator.id=b.created_by WHERE b.tenant_id=$1 ORDER BY b.updated_at DESC`, normalizedTenantID(tenantID), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	boards := []model.CollaborationWhiteboard{}
	for rows.Next() {
		board, err := scanWhiteboard(rows)
		if err != nil {
			return nil, err
		}
		boards = append(boards, board)
	}
	return boards, rows.Err()
}

func GetCollaborationWhiteboard(boardID, tenantID, userID string) (model.CollaborationWhiteboard, error) {
	return scanWhiteboard(db.QueryRowContext(context.Background(), `SELECT b.id,b.tenant_id,b.title,b.data,b.created_by,creator.name,b.updated_by,b.version,b.created_at,b.updated_at,wm.role FROM collaboration_whiteboards b JOIN collaboration_whiteboard_members wm ON wm.whiteboard_id=b.id AND wm.tenant_id=b.tenant_id AND wm.user_id=$3 JOIN users creator ON creator.id=b.created_by WHERE b.id=$1 AND b.tenant_id=$2`, boardID, normalizedTenantID(tenantID), userID))
}

func UpdateCollaborationWhiteboard(boardID, tenantID, userID, title, data string, expectedVersion int) (model.CollaborationWhiteboard, error) {
	role, err := collaborationWhiteboardRole(boardID, tenantID, userID)
	if err != nil {
		return model.CollaborationWhiteboard{}, err
	}
	if role != "owner" && role != "editor" {
		return model.CollaborationWhiteboard{}, ErrWhiteboardForbidden
	}
	if expectedVersion < 1 {
		return model.CollaborationWhiteboard{}, ErrWhiteboardConflict
	}
	now := time.Now().UTC()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return model.CollaborationWhiteboard{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(context.Background(), `UPDATE collaboration_whiteboards SET title=$3,data=$4,updated_by=$5,version=version+1,updated_at=$6 WHERE id=$1 AND tenant_id=$2 AND version=$7`, boardID, normalizedTenantID(tenantID), strings.TrimSpace(title), strings.TrimSpace(data), userID, now, expectedVersion)
	if err != nil {
		return model.CollaborationWhiteboard{}, err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return model.CollaborationWhiteboard{}, ErrWhiteboardConflict
	}
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO collaboration_whiteboard_revisions (whiteboard_id,tenant_id,version,title,data,edited_by,edited_at) SELECT id,tenant_id,version,title,data,updated_by,updated_at FROM collaboration_whiteboards WHERE id=$1 AND tenant_id=$2`, boardID, normalizedTenantID(tenantID)); err != nil {
		return model.CollaborationWhiteboard{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.CollaborationWhiteboard{}, err
	}
	return GetCollaborationWhiteboard(boardID, tenantID, userID)
}

func DeleteCollaborationWhiteboard(boardID, tenantID, userID string) error {
	role, err := collaborationWhiteboardRole(boardID, tenantID, userID)
	if err != nil {
		return err
	}
	if role != "owner" {
		return ErrWhiteboardForbidden
	}
	result, err := db.ExecContext(context.Background(), `DELETE FROM collaboration_whiteboards WHERE id=$1 AND tenant_id=$2`, boardID, normalizedTenantID(tenantID))
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func GetCollaborationWhiteboardMembers(boardID, tenantID, userID string) ([]model.CollaborationWhiteboardMember, error) {
	if _, err := collaborationWhiteboardRole(boardID, tenantID, userID); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(context.Background(), `SELECT wm.whiteboard_id,wm.user_id,u.name,wm.role,u.organization_id,wm.added_at FROM collaboration_whiteboard_members wm JOIN users u ON u.id=wm.user_id WHERE wm.whiteboard_id=$1 AND wm.tenant_id=$2 ORDER BY CASE wm.role WHEN 'owner' THEN 0 WHEN 'editor' THEN 1 ELSE 2 END,u.name`, boardID, normalizedTenantID(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := []model.CollaborationWhiteboardMember{}
	for rows.Next() {
		var member model.CollaborationWhiteboardMember
		if err := rows.Scan(&member.WhiteboardID, &member.UserID, &member.Name, &member.Role, &member.Org, &member.AddedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func AddCollaborationWhiteboardMember(boardID, tenantID, actorID, targetUserID, role string) (model.CollaborationWhiteboardMember, error) {
	actorRole, err := collaborationWhiteboardRole(boardID, tenantID, actorID)
	if err != nil {
		return model.CollaborationWhiteboardMember{}, err
	}
	if actorRole != "owner" {
		return model.CollaborationWhiteboardMember{}, ErrWhiteboardForbidden
	}
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "editor" && role != "viewer" {
		return model.CollaborationWhiteboardMember{}, ErrWhiteboardForbidden
	}
	tenantID = normalizedTenantID(tenantID)
	var member model.CollaborationWhiteboardMember
	if err := db.QueryRowContext(context.Background(), `SELECT id,name,organization_id FROM users WHERE id=$1 AND organization_id=$2`, targetUserID, tenantID).Scan(&member.UserID, &member.Name, &member.Org); err != nil {
		return model.CollaborationWhiteboardMember{}, err
	}
	member.WhiteboardID = boardID
	member.Role = role
	member.AddedAt = time.Now().UTC()
	_, err = db.ExecContext(context.Background(), `INSERT INTO collaboration_whiteboard_members (whiteboard_id,tenant_id,user_id,role,added_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (whiteboard_id,user_id) DO UPDATE SET role=EXCLUDED.role WHERE collaboration_whiteboard_members.role <> 'owner'`, boardID, tenantID, targetUserID, role, member.AddedAt)
	return member, err
}

func RemoveCollaborationWhiteboardMember(boardID, tenantID, actorID, targetUserID string) error {
	actorRole, err := collaborationWhiteboardRole(boardID, tenantID, actorID)
	if err != nil {
		return err
	}
	if actorRole != "owner" {
		return ErrWhiteboardForbidden
	}
	result, err := db.ExecContext(context.Background(), `DELETE FROM collaboration_whiteboard_members WHERE whiteboard_id=$1 AND tenant_id=$2 AND user_id=$3 AND role <> 'owner'`, boardID, normalizedTenantID(tenantID), targetUserID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func GetCollaborationWhiteboardRevisions(boardID, tenantID, userID string) ([]model.CollaborationWhiteboardRevision, error) {
	if _, err := collaborationWhiteboardRole(boardID, tenantID, userID); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(context.Background(), `SELECT id,whiteboard_id,version,title,data,edited_by,edited_at FROM collaboration_whiteboard_revisions WHERE whiteboard_id=$1 AND tenant_id=$2 ORDER BY version DESC`, boardID, normalizedTenantID(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	revisions := []model.CollaborationWhiteboardRevision{}
	for rows.Next() {
		var revision model.CollaborationWhiteboardRevision
		if err := rows.Scan(&revision.ID, &revision.WhiteboardID, &revision.Version, &revision.Title, &revision.Data, &revision.EditedBy, &revision.EditedAt); err != nil {
			return nil, err
		}
		revisions = append(revisions, revision)
	}
	return revisions, rows.Err()
}
