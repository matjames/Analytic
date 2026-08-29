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
	ErrDocumentForbidden = errors.New("document action forbidden")
	ErrDocumentConflict  = errors.New("document version conflict")
)

func ensureDocumentsSchema(ctx context.Context) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS collaboration_documents (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  title TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  created_by TEXT NOT NULL REFERENCES users(id),
  updated_by TEXT NOT NULL REFERENCES users(id),
  version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS collaboration_documents_tenant_updated_idx ON collaboration_documents (tenant_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS collaboration_document_members (
  document_id TEXT NOT NULL REFERENCES collaboration_documents(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('owner','editor','viewer')),
  added_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (document_id, user_id)
);
CREATE INDEX IF NOT EXISTS collaboration_document_members_user_idx ON collaboration_document_members (tenant_id, user_id);

CREATE TABLE IF NOT EXISTS collaboration_document_revisions (
  id BIGSERIAL PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES collaboration_documents(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  edited_by TEXT NOT NULL REFERENCES users(id),
  edited_at TIMESTAMPTZ NOT NULL,
  UNIQUE (document_id, version)
);
CREATE INDEX IF NOT EXISTS collaboration_document_revisions_document_idx ON collaboration_document_revisions (tenant_id, document_id, version DESC);
`)
	return err
}

func CreateCollaborationDocument(document model.CollaborationDocument) (model.CollaborationDocument, error) {
	document.ID = uuid.NewString()
	document.TenantID = normalizedTenantID(document.TenantID)
	document.Title = strings.TrimSpace(document.Title)
	document.Content = strings.TrimSpace(document.Content)
	document.Version = 1
	document.Role = "owner"
	document.CanEdit = true
	document.CreatedAt = time.Now().UTC()
	document.UpdatedAt = document.CreatedAt
	document.UpdatedBy = document.CreatedBy

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return document, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO collaboration_documents (id,tenant_id,title,content,created_by,updated_by,version,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$5,1,$6,$6)`,
		document.ID, document.TenantID, document.Title, document.Content, document.CreatedBy, document.CreatedAt); err != nil {
		return document, err
	}
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO collaboration_document_members (document_id,tenant_id,user_id,role,added_at) VALUES ($1,$2,$3,'owner',$4)`,
		document.ID, document.TenantID, document.CreatedBy, document.CreatedAt); err != nil {
		return document, err
	}
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO collaboration_document_revisions (document_id,tenant_id,version,title,content,edited_by,edited_at) VALUES ($1,$2,1,$3,$4,$5,$6)`,
		document.ID, document.TenantID, document.Title, document.Content, document.CreatedBy, document.CreatedAt); err != nil {
		return document, err
	}
	return document, tx.Commit()
}

func collaborationDocumentRole(documentID, tenantID, userID string) (string, error) {
	var role string
	err := db.QueryRowContext(context.Background(), `SELECT dm.role FROM collaboration_document_members dm JOIN collaboration_documents d ON d.id=dm.document_id AND d.tenant_id=dm.tenant_id WHERE dm.document_id=$1 AND dm.tenant_id=$2 AND dm.user_id=$3`, documentID, normalizedTenantID(tenantID), userID).Scan(&role)
	return role, err
}

func GetCollaborationDocuments(tenantID, userID string) ([]model.CollaborationDocument, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT d.id,d.tenant_id,d.title,d.content,d.created_by,creator.name,d.updated_by,d.version,d.created_at,d.updated_at,dm.role
FROM collaboration_documents d
JOIN collaboration_document_members dm ON dm.document_id=d.id AND dm.tenant_id=d.tenant_id AND dm.user_id=$2
JOIN users creator ON creator.id=d.created_by
WHERE d.tenant_id=$1
ORDER BY d.updated_at DESC`, normalizedTenantID(tenantID), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	documents := []model.CollaborationDocument{}
	for rows.Next() {
		var document model.CollaborationDocument
		if err := rows.Scan(&document.ID, &document.TenantID, &document.Title, &document.Content, &document.CreatedBy, &document.Author, &document.UpdatedBy, &document.Version, &document.CreatedAt, &document.UpdatedAt, &document.Role); err != nil {
			return nil, err
		}
		document.CanEdit = document.Role == "owner" || document.Role == "editor"
		documents = append(documents, document)
	}
	return documents, rows.Err()
}

func GetCollaborationDocument(documentID, tenantID, userID string) (model.CollaborationDocument, error) {
	var document model.CollaborationDocument
	err := db.QueryRowContext(context.Background(), `
SELECT d.id,d.tenant_id,d.title,d.content,d.created_by,creator.name,d.updated_by,d.version,d.created_at,d.updated_at,dm.role
FROM collaboration_documents d
JOIN collaboration_document_members dm ON dm.document_id=d.id AND dm.tenant_id=d.tenant_id AND dm.user_id=$3
JOIN users creator ON creator.id=d.created_by
WHERE d.id=$1 AND d.tenant_id=$2`, documentID, normalizedTenantID(tenantID), userID).Scan(
		&document.ID, &document.TenantID, &document.Title, &document.Content, &document.CreatedBy, &document.Author, &document.UpdatedBy, &document.Version, &document.CreatedAt, &document.UpdatedAt, &document.Role)
	if err != nil {
		return model.CollaborationDocument{}, err
	}
	document.CanEdit = document.Role == "owner" || document.Role == "editor"
	return document, nil
}

func UpdateCollaborationDocument(documentID, tenantID, userID, title, content string, expectedVersion int) (model.CollaborationDocument, error) {
	role, err := collaborationDocumentRole(documentID, tenantID, userID)
	if err != nil {
		return model.CollaborationDocument{}, err
	}
	if role != "owner" && role != "editor" {
		return model.CollaborationDocument{}, ErrDocumentForbidden
	}
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if expectedVersion < 1 {
		return model.CollaborationDocument{}, ErrDocumentConflict
	}
	now := time.Now().UTC()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return model.CollaborationDocument{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(context.Background(), `UPDATE collaboration_documents SET title=$3,content=$4,updated_by=$5,version=version+1,updated_at=$6 WHERE id=$1 AND tenant_id=$2 AND version=$7`, documentID, normalizedTenantID(tenantID), title, content, userID, now, expectedVersion)
	if err != nil {
		return model.CollaborationDocument{}, err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return model.CollaborationDocument{}, ErrDocumentConflict
	}
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO collaboration_document_revisions (document_id,tenant_id,version,title,content,edited_by,edited_at) SELECT id,tenant_id,version,title,content,updated_by,updated_at FROM collaboration_documents WHERE id=$1 AND tenant_id=$2`, documentID, normalizedTenantID(tenantID)); err != nil {
		return model.CollaborationDocument{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.CollaborationDocument{}, err
	}
	return GetCollaborationDocument(documentID, tenantID, userID)
}

func DeleteCollaborationDocument(documentID, tenantID, userID string) error {
	role, err := collaborationDocumentRole(documentID, tenantID, userID)
	if err != nil {
		return err
	}
	if role != "owner" {
		return ErrDocumentForbidden
	}
	result, err := db.ExecContext(context.Background(), `DELETE FROM collaboration_documents WHERE id=$1 AND tenant_id=$2`, documentID, normalizedTenantID(tenantID))
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func GetCollaborationDocumentMembers(documentID, tenantID, userID string) ([]model.CollaborationDocumentMember, error) {
	if _, err := collaborationDocumentRole(documentID, tenantID, userID); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(context.Background(), `SELECT dm.document_id,dm.user_id,u.name,dm.role,u.organization_id,dm.added_at FROM collaboration_document_members dm JOIN users u ON u.id=dm.user_id WHERE dm.document_id=$1 AND dm.tenant_id=$2 ORDER BY CASE dm.role WHEN 'owner' THEN 0 WHEN 'editor' THEN 1 ELSE 2 END,u.name`, documentID, normalizedTenantID(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := []model.CollaborationDocumentMember{}
	for rows.Next() {
		var member model.CollaborationDocumentMember
		if err := rows.Scan(&member.DocumentID, &member.UserID, &member.Name, &member.Role, &member.Org, &member.AddedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func AddCollaborationDocumentMember(documentID, tenantID, actorID, targetUserID, role string) (model.CollaborationDocumentMember, error) {
	actorRole, err := collaborationDocumentRole(documentID, tenantID, actorID)
	if err != nil {
		return model.CollaborationDocumentMember{}, err
	}
	if actorRole != "owner" {
		return model.CollaborationDocumentMember{}, ErrDocumentForbidden
	}
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "editor" && role != "viewer" {
		return model.CollaborationDocumentMember{}, ErrDocumentForbidden
	}
	tenantID = normalizedTenantID(tenantID)
	var member model.CollaborationDocumentMember
	err = db.QueryRowContext(context.Background(), `SELECT id,name,organization_id FROM users WHERE id=$1 AND organization_id=$2`, targetUserID, tenantID).Scan(&member.UserID, &member.Name, &member.Org)
	if err != nil {
		return model.CollaborationDocumentMember{}, err
	}
	member.DocumentID = documentID
	member.Role = role
	member.AddedAt = time.Now().UTC()
	_, err = db.ExecContext(context.Background(), `INSERT INTO collaboration_document_members (document_id,tenant_id,user_id,role,added_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (document_id,user_id) DO UPDATE SET role=EXCLUDED.role WHERE collaboration_document_members.role <> 'owner'`, documentID, tenantID, targetUserID, role, member.AddedAt)
	return member, err
}

func RemoveCollaborationDocumentMember(documentID, tenantID, actorID, targetUserID string) error {
	actorRole, err := collaborationDocumentRole(documentID, tenantID, actorID)
	if err != nil {
		return err
	}
	if actorRole != "owner" {
		return ErrDocumentForbidden
	}
	result, err := db.ExecContext(context.Background(), `DELETE FROM collaboration_document_members WHERE document_id=$1 AND tenant_id=$2 AND user_id=$3 AND role <> 'owner'`, documentID, normalizedTenantID(tenantID), targetUserID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func GetCollaborationDocumentRevisions(documentID, tenantID, userID string) ([]model.CollaborationDocumentRevision, error) {
	if _, err := collaborationDocumentRole(documentID, tenantID, userID); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(context.Background(), `SELECT id,document_id,version,title,content,edited_by,edited_at FROM collaboration_document_revisions WHERE document_id=$1 AND tenant_id=$2 ORDER BY version DESC`, documentID, normalizedTenantID(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	revisions := []model.CollaborationDocumentRevision{}
	for rows.Next() {
		var revision model.CollaborationDocumentRevision
		if err := rows.Scan(&revision.ID, &revision.DocumentID, &revision.Version, &revision.Title, &revision.Content, &revision.EditedBy, &revision.EditedAt); err != nil {
			return nil, err
		}
		revisions = append(revisions, revision)
	}
	return revisions, rows.Err()
}
