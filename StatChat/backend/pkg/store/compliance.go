package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"statchat/pkg/model"

	"github.com/google/uuid"
)

func GetRetentionPolicy(tenantID string) (model.RetentionPolicy, error) {
	policy := model.RetentionPolicy{TenantID: normalizedTenantID(tenantID), RetentionDays: 365}
	err := db.QueryRowContext(context.Background(), `SELECT tenant_id, retention_days, enabled, updated_by, created_at, updated_at FROM retention_policies WHERE tenant_id = $1`, policy.TenantID).Scan(&policy.TenantID, &policy.RetentionDays, &policy.Enabled, &policy.UpdatedBy, &policy.CreatedAt, &policy.UpdatedAt)
	if err == sql.ErrNoRows {
		return policy, nil
	}
	return policy, err
}

func UpsertRetentionPolicy(tenantID, actorID string, retentionDays int, enabled bool) (model.RetentionPolicy, error) {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return model.RetentionPolicy{}, err
	}
	defer tx.Rollback()
	var policy model.RetentionPolicy
	err = tx.QueryRowContext(context.Background(), `
INSERT INTO retention_policies (tenant_id, retention_days, enabled, updated_by)
VALUES ($1,$2,$3,$4)
ON CONFLICT (tenant_id) DO UPDATE SET retention_days = EXCLUDED.retention_days, enabled = EXCLUDED.enabled, updated_by = EXCLUDED.updated_by, updated_at = NOW()
RETURNING tenant_id, retention_days, enabled, updated_by, created_at, updated_at`, normalizedTenantID(tenantID), retentionDays, enabled, actorID).Scan(&policy.TenantID, &policy.RetentionDays, &policy.Enabled, &policy.UpdatedBy, &policy.CreatedAt, &policy.UpdatedAt)
	if err != nil {
		return policy, err
	}
	if err := createComplianceAuditEventTx(context.Background(), tx, policy.TenantID, actorID, "retention.policy.updated", "tenant", policy.TenantID, map[string]any{"retentionDays": retentionDays, "enabled": enabled}); err != nil {
		return policy, err
	}
	return policy, tx.Commit()
}

func CreateLegalHold(hold model.LegalHold) (model.LegalHold, error) {
	hold.ID = uuid.NewString()
	hold.TenantID = normalizedTenantID(hold.TenantID)
	hold.Status = "active"
	hold.CreatedAt = time.Now().UTC()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return hold, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO legal_holds (id, tenant_id, conversation_id, name, reason, status, created_by, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, hold.ID, hold.TenantID, nullString(hold.ConversationID), hold.Name, hold.Reason, hold.Status, hold.CreatedBy, hold.CreatedAt); err != nil {
		return hold, err
	}
	if err = createComplianceAuditEventTx(context.Background(), tx, hold.TenantID, hold.CreatedBy, "legal_hold.created", "legal_hold", hold.ID, map[string]any{"conversationId": hold.ConversationID, "name": hold.Name, "reason": hold.Reason}); err != nil {
		return hold, err
	}
	return hold, tx.Commit()
}

func ListLegalHolds(tenantID string) ([]model.LegalHold, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, tenant_id, COALESCE(conversation_id,''), name, reason, status, created_by, created_at, COALESCE(released_by,''), released_at FROM legal_holds WHERE tenant_id = $1 ORDER BY created_at DESC`, normalizedTenantID(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	holds := []model.LegalHold{}
	for rows.Next() {
		var hold model.LegalHold
		var releasedAt sql.NullTime
		if err := rows.Scan(&hold.ID, &hold.TenantID, &hold.ConversationID, &hold.Name, &hold.Reason, &hold.Status, &hold.CreatedBy, &hold.CreatedAt, &hold.ReleasedBy, &releasedAt); err != nil {
			return nil, err
		}
		if releasedAt.Valid {
			hold.ReleasedAt = releasedAt.Time
		}
		holds = append(holds, hold)
	}
	return holds, rows.Err()
}

func ReleaseLegalHold(tenantID, holdID, actorID string) (bool, error) {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(context.Background(), `UPDATE legal_holds SET status = 'released', released_by = $3, released_at = NOW() WHERE id = $1 AND tenant_id = $2 AND status = 'active'`, holdID, normalizedTenantID(tenantID), actorID)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return false, err
	}
	if err := createComplianceAuditEventTx(context.Background(), tx, tenantID, actorID, "legal_hold.released", "legal_hold", holdID, nil); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func CreateComplianceAuditEvent(tenantID, actorID, action, targetType, targetID string, details map[string]any) error {
	return createComplianceAuditEventTx(context.Background(), db, tenantID, actorID, action, targetType, targetID, details)
}

type complianceExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func createComplianceAuditEventTx(ctx context.Context, execer complianceExecer, tenantID, actorID, action, targetType, targetID string, details map[string]any) error {
	if details == nil {
		details = map[string]any{}
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = execer.ExecContext(ctx, `INSERT INTO compliance_audit_events (id, tenant_id, actor_id, action, target_type, target_id, details, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, uuid.NewString(), normalizedTenantID(tenantID), actorID, action, targetType, targetID, encoded, time.Now().UTC())
	return err
}

func ConversationBelongsToTenant(conversationID, tenantID string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(context.Background(), `SELECT EXISTS (SELECT 1 FROM conversations WHERE id = $1 AND tenant_id = $2)`, conversationID, normalizedTenantID(tenantID)).Scan(&exists)
	return exists, err
}

func ListComplianceAuditEvents(tenantID string, limit int) ([]model.ComplianceAuditEvent, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := db.QueryContext(context.Background(), `SELECT id, tenant_id, actor_id, action, target_type, target_id, details, created_at FROM compliance_audit_events WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`, normalizedTenantID(tenantID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []model.ComplianceAuditEvent{}
	for rows.Next() {
		var event model.ComplianceAuditEvent
		var details []byte
		if err := rows.Scan(&event.ID, &event.TenantID, &event.ActorID, &event.Action, &event.TargetType, &event.TargetID, &details, &event.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(details, &event.Details)
		events = append(events, event)
	}
	return events, rows.Err()
}

func EnforceRetention(ctx context.Context, tenantID, actorID string) (int64, error) {
	policy, err := GetRetentionPolicy(tenantID)
	if err != nil || !policy.Enabled {
		return 0, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var locked bool
	if err := tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock(hashtext($1))`, "statchat-retention:"+policy.TenantID).Scan(&locked); err != nil || !locked {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
CREATE TEMP TABLE retention_candidates ON COMMIT DROP AS
SELECT m.id FROM messages m
WHERE m.tenant_id = $1 AND m.created_at < NOW() - make_interval(days => $2)
  AND NOT EXISTS (
    SELECT 1 FROM legal_holds lh
    WHERE lh.tenant_id = $1 AND lh.status = 'active'
      AND (lh.conversation_id IS NULL OR lh.conversation_id = m.conversation_id)
  )`, policy.TenantID, policy.RetentionDays); err != nil {
		return 0, err
	}
	var count int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM retention_candidates`).Scan(&count); err != nil {
		return 0, err
	}
	attachmentRows, err := tx.QueryContext(ctx, `SELECT url FROM message_attachments WHERE message_id IN (SELECT id FROM retention_candidates)`)
	if err != nil {
		return 0, err
	}
	var uploadURLs []string
	for attachmentRows.Next() {
		var url string
		if err := attachmentRows.Scan(&url); err != nil {
			attachmentRows.Close()
			return 0, err
		}
		uploadURLs = append(uploadURLs, url)
	}
	if err := attachmentRows.Close(); err != nil {
		return 0, err
	}
	statements := []string{
		`UPDATE scheduled_messages SET message_id = NULL WHERE message_id IN (SELECT id FROM retention_candidates)`,
		`DELETE FROM message_reactions WHERE message_id IN (SELECT id FROM retention_candidates)`,
		`DELETE FROM message_attachments WHERE message_id IN (SELECT id FROM retention_candidates)`,
		`DELETE FROM pinned_messages WHERE message_id IN (SELECT id FROM retention_candidates)`,
		`DELETE FROM read_receipts WHERE message_id IN (SELECT id FROM retention_candidates)`,
		`DELETE FROM saved_messages WHERE message_id IN (SELECT id FROM retention_candidates)`,
		`DELETE FROM message_mentions WHERE message_id IN (SELECT id FROM retention_candidates)`,
		`DELETE FROM message_locations WHERE message_id IN (SELECT id FROM retention_candidates)`,
		`DELETE FROM messages WHERE id IN (SELECT id FROM retention_candidates)`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return 0, err
		}
	}
	if err := createComplianceAuditEventTx(ctx, tx, policy.TenantID, actorID, "retention.enforced", "tenant", policy.TenantID, map[string]any{"deletedMessages": count, "retentionDays": policy.RetentionDays}); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	removeRetainedUploads(uploadURLs)
	return count, nil
}

func removeRetainedUploads(urls []string) {
	root := strings.TrimSpace(os.Getenv("STATCHAT_UPLOAD_DIR"))
	if root == "" {
		root = "./uploads"
	}
	for _, url := range urls {
		if !strings.HasPrefix(url, "/uploads/") {
			continue
		}
		name := filepath.Base(strings.TrimPrefix(url, "/uploads/"))
		if name != "." && name != "" {
			_ = os.Remove(filepath.Join(root, name))
		}
	}
}

func StartRetentionWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			if err := enforceAllRetentionPolicies(ctx); err != nil && ctx.Err() == nil {
				log.Printf("retention enforcement failed: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func enforceAllRetentionPolicies(ctx context.Context) error {
	rows, err := db.QueryContext(ctx, `SELECT tenant_id FROM retention_policies WHERE enabled = true`)
	if err != nil {
		return err
	}
	var tenants []string
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err != nil {
			rows.Close()
			return err
		}
		tenants = append(tenants, tenantID)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, tenantID := range tenants {
		if _, err := EnforceRetention(ctx, tenantID, "system:retention-worker"); err != nil {
			return err
		}
	}
	return nil
}

func ExportConversationMessages(conversationID, tenantID string, includeDeleted bool) ([]model.Message, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, conversation_id, channel_id, COALESCE(sender_id,''), sender, text, created_at, updated_at, deleted_at, parent_message_id, thread_root_id, status, tenant_id, delivery_status, COALESCE(forwarded_from_message_id,''), COALESCE(forwarded_from_sender,'') FROM messages WHERE conversation_id = $1 AND tenant_id = $2 AND ($3 OR status != 'deleted') ORDER BY created_at`, conversationID, normalizedTenantID(tenantID), includeDeleted)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []model.Message{}
	for rows.Next() {
		var message model.Message
		var channelID, parentID, threadRootID sql.NullString
		var updatedAt, deletedAt sql.NullTime
		if err := rows.Scan(&message.ID, &message.ConversationID, &channelID, &message.SenderID, &message.Sender, &message.Text, &message.CreatedAt, &updatedAt, &deletedAt, &parentID, &threadRootID, &message.Status, &message.TenantID, &message.DeliveryStatus, &message.ForwardedFromMessageID, &message.ForwardedFromSender); err != nil {
			return nil, err
		}
		message.ChannelID, message.ParentMessageID, message.ThreadRootID = channelID.String, parentID.String, threadRootID.String
		if updatedAt.Valid {
			message.UpdatedAt = updatedAt.Time
		}
		if deletedAt.Valid {
			message.DeletedAt = deletedAt.Time
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for index := range messages {
		messages[index].Attachments, err = GetMessageAttachments(messages[index].ID)
		if err != nil {
			return nil, err
		}
		messages[index].Location, err = GetMessageLocation(messages[index].ID)
		if err != nil {
			return nil, err
		}
		messages[index].MentionUserIDs, messages[index].MentionAll, err = GetMessageMentions(messages[index].ID)
		if err != nil {
			return nil, err
		}
		messages[index].Reactions, err = GetMessageReactions(messages[index].ID)
		if err != nil {
			return nil, err
		}
		messages[index].ReadBy, err = GetMessageReadBy(messages[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}
