package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"statchat/pkg/model"

	"github.com/google/uuid"
)

func CreateScheduledMessage(item model.ScheduledMessage) (model.ScheduledMessage, error) {
	item.ID = uuid.NewString()
	item.TenantID = normalizedTenantID(item.TenantID)
	item.ScheduledFor = item.ScheduledFor.UTC()
	item.Status = "pending"
	item.CreatedAt = time.Now().UTC()
	item.UpdatedAt = item.CreatedAt
	_, err := db.ExecContext(context.Background(), `INSERT INTO scheduled_messages (id, tenant_id, conversation_id, sender_id, sender, text, scheduled_for, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		item.ID, item.TenantID, item.ConversationID, item.SenderID, item.Sender, item.Text, item.ScheduledFor, item.Status, item.CreatedAt, item.UpdatedAt)
	return item, err
}

func ListScheduledMessages(tenantID, senderID, conversationID string) ([]model.ScheduledMessage, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, tenant_id, conversation_id, sender_id, sender, text, scheduled_for, status, COALESCE(message_id, ''), created_at, updated_at FROM scheduled_messages WHERE tenant_id = $1 AND sender_id = $2 AND status = 'pending' AND ($3 = '' OR conversation_id = $3) ORDER BY scheduled_for`, normalizedTenantID(tenantID), senderID, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.ScheduledMessage{}
	for rows.Next() {
		var item model.ScheduledMessage
		if err := rows.Scan(&item.ID, &item.TenantID, &item.ConversationID, &item.SenderID, &item.Sender, &item.Text, &item.ScheduledFor, &item.Status, &item.MessageID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func CancelScheduledMessage(id, tenantID, senderID string) (bool, error) {
	result, err := db.ExecContext(context.Background(), `UPDATE scheduled_messages SET status = 'cancelled', updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND sender_id = $3 AND status = 'pending'`, id, normalizedTenantID(tenantID), senderID)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func StartScheduledMessageWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			if err := deliverDueScheduledMessages(ctx); err != nil && ctx.Err() == nil {
				log.Printf("scheduled message delivery failed: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func deliverDueScheduledMessages(ctx context.Context) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Membership can change after scheduling. Cancel due items whose sender no
	// longer has access instead of publishing stale-authority content.
	if _, err := tx.ExecContext(ctx, `
UPDATE scheduled_messages sm
SET status = 'cancelled', updated_at = NOW()
WHERE sm.status = 'pending' AND sm.scheduled_for <= NOW()
  AND NOT EXISTS (
    SELECT 1 FROM conversations c
    WHERE c.id = sm.conversation_id
      AND (c.tenant_id = sm.tenant_id OR c.tenant_id = 'default')
      AND (c.type = 'channel' OR c.member_ids::jsonb @> to_jsonb(sm.sender_id))
  )`); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, tenant_id, conversation_id, sender_id, sender, text, scheduled_for, created_at FROM scheduled_messages WHERE status = 'pending' AND scheduled_for <= NOW() ORDER BY scheduled_for FOR UPDATE SKIP LOCKED LIMIT 50`)
	if err != nil {
		return err
	}
	items := []model.ScheduledMessage{}
	for rows.Next() {
		var item model.ScheduledMessage
		if err := rows.Scan(&item.ID, &item.TenantID, &item.ConversationID, &item.SenderID, &item.Sender, &item.Text, &item.ScheduledFor, &item.CreatedAt); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	messages := make([]model.Message, 0, len(items))
	for _, item := range items {
		message := model.Message{ID: uuid.NewString(), TenantID: item.TenantID, ConversationID: item.ConversationID, SenderID: item.SenderID, Sender: item.Sender, Text: item.Text, CreatedAt: time.Now().UTC(), Status: "active", DeliveryStatus: "sent"}
		if _, err := tx.ExecContext(ctx, `INSERT INTO messages (id, conversation_id, sender_id, sender, text, created_at, status, tenant_id, delivery_status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, message.ID, message.ConversationID, message.SenderID, message.Sender, message.Text, message.CreatedAt, message.Status, message.TenantID, message.DeliveryStatus); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE scheduled_messages SET status = 'sent', message_id = $1, updated_at = NOW() WHERE id = $2`, message.ID, item.ID); err != nil {
			return err
		}
		messages = append(messages, message)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, message := range messages {
		BroadcastMessage(message)
		_ = NotifyConversationMembers(message.ConversationID, message.SenderID, message.Sender, message.Text, fmt.Sprintf("/chat/%s", message.ConversationID), nil)
	}
	return nil
}
