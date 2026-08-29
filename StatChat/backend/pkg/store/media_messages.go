package store

import (
	"context"

	"statchat/pkg/model"
)

func StoreMessageWithAttachments(message model.Message) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO messages (id, conversation_id, channel_id, sender_id, sender, text, created_at, status, tenant_id, delivery_status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, message.ID, message.ConversationID, nullString(message.ChannelID), nullString(message.SenderID), message.Sender, message.Text, message.CreatedAt, message.Status, normalizedTenantID(message.TenantID), message.DeliveryStatus); err != nil {
		return err
	}
	for _, attachment := range message.Attachments {
		if _, err := tx.Exec(`INSERT INTO message_attachments (id, message_id, file_name, file_type, url, created_at) VALUES ($1,$2,$3,$4,$5,$6)`, attachment.ID, message.ID, attachment.FileName, attachment.FileType, attachment.URL, attachment.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}
