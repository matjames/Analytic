package store

import (
	"context"
	"database/sql"
	"time"

	"statchat/pkg/model"
)

type MessageSearchFilters struct {
	Query          string
	ConversationID string
	Sender         string
	DateFrom       *time.Time
	DateTo         *time.Time
	HasAttachment  *bool
	SavedOnly      bool
}

func SaveMessage(userID, messageID string) (time.Time, error) {
	savedAt := time.Now().UTC()
	err := db.QueryRowContext(context.Background(), `
INSERT INTO saved_messages (user_id, message_id, created_at) VALUES ($1,$2,$3)
ON CONFLICT (user_id, message_id) DO UPDATE SET created_at = saved_messages.created_at
RETURNING created_at`, userID, messageID, savedAt).Scan(&savedAt)
	return savedAt, err
}

func UnsaveMessage(userID, messageID string) error {
	_, err := db.ExecContext(context.Background(), `DELETE FROM saved_messages WHERE user_id = $1 AND message_id = $2`, userID, messageID)
	return err
}

func GetSavedMessages(userID, tenantID string, filters MessageSearchFilters) ([]model.Message, error) {
	return searchMessages(userID, normalizedTenantID(tenantID), filters, true)
}

func SearchMessages(userID, tenantID string, filters MessageSearchFilters) ([]model.Message, error) {
	return searchMessages(userID, normalizedTenantID(tenantID), filters, filters.SavedOnly)
}

func searchMessages(userID, tenantID string, filters MessageSearchFilters, savedOnly bool) ([]model.Message, error) {
	query := `
SELECT m.id, m.conversation_id, m.channel_id, COALESCE(m.sender_id, ''), m.sender, m.text,
       m.created_at, m.updated_at, m.deleted_at, m.parent_message_id, m.thread_root_id,
       m.status, m.tenant_id, m.delivery_status, COALESCE(m.forwarded_from_message_id, ''),
       COALESCE(m.forwarded_from_sender, ''), sm.created_at
FROM messages m
JOIN conversations c ON c.id = m.conversation_id
LEFT JOIN saved_messages sm ON sm.message_id = m.id AND sm.user_id = $2
WHERE m.status != 'deleted'
  AND m.tenant_id = $3
  AND (c.tenant_id = $3 OR c.tenant_id = 'default')
  AND (c.member_ids::jsonb @> to_jsonb($2::text) OR c.type = 'channel')
  AND ($1 = '' OR m.text ILIKE '%' || $1 || '%' OR m.sender ILIKE '%' || $1 || '%')
  AND ($4 = '' OR m.conversation_id = $4)
  AND ($5 = '' OR m.sender_id = $5 OR m.sender ILIKE '%' || $5 || '%')
  AND ($6::timestamptz IS NULL OR m.created_at >= $6)
  AND ($7::timestamptz IS NULL OR m.created_at < $7)
  AND ($8::boolean IS NULL OR EXISTS (SELECT 1 FROM message_attachments ma WHERE ma.message_id = m.id) = $8)
  AND (NOT $9 OR sm.message_id IS NOT NULL)
ORDER BY COALESCE(sm.created_at, m.created_at) DESC
LIMIT 100`
	rows, err := db.QueryContext(context.Background(), query, filters.Query, userID, tenantID, filters.ConversationID, filters.Sender, nullableTime(filters.DateFrom), nullableTime(filters.DateTo), nullableBool(filters.HasAttachment), savedOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []model.Message{}
	for rows.Next() {
		message, err := scanSearchedMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSearchedMessage(row rowScanner) (model.Message, error) {
	var message model.Message
	var channelID, parentID, threadRootID sql.NullString
	var updatedAt, deletedAt, savedAt sql.NullTime
	err := row.Scan(&message.ID, &message.ConversationID, &channelID, &message.SenderID, &message.Sender, &message.Text, &message.CreatedAt, &updatedAt, &deletedAt, &parentID, &threadRootID, &message.Status, &message.TenantID, &message.DeliveryStatus, &message.ForwardedFromMessageID, &message.ForwardedFromSender, &savedAt)
	if err != nil {
		return message, err
	}
	message.ChannelID = channelID.String
	message.ParentMessageID = parentID.String
	message.ThreadRootID = threadRootID.String
	if updatedAt.Valid {
		message.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		message.DeletedAt = deletedAt.Time
	}
	if savedAt.Valid {
		message.SavedAt = savedAt.Time
	}
	return message, nil
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}
