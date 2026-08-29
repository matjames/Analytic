package store

import (
	"context"
	"database/sql"
	"strings"

	"statchat/pkg/model"
)

func StoreMessageWithMentions(message model.Message) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO messages (id, conversation_id, channel_id, sender_id, sender, text, created_at, updated_at, deleted_at, parent_message_id, thread_root_id, status, tenant_id, delivery_status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, message.ID, message.ConversationID, nullString(message.ChannelID), nullString(message.SenderID), message.Sender, message.Text, message.CreatedAt, nullTime(message.UpdatedAt), nullTime(message.DeletedAt), nullString(message.ParentMessageID), nullString(message.ThreadRootID), message.Status, normalizedTenantID(message.TenantID), message.DeliveryStatus); err != nil {
		return err
	}
	for _, userID := range message.MentionUserIDs {
		if _, err := tx.Exec(`INSERT INTO message_mentions (message_id, user_id, created_at) VALUES ($1,$2,$3)`, message.ID, userID, message.CreatedAt); err != nil {
			return err
		}
	}
	if message.MentionAll {
		if _, err := tx.Exec(`INSERT INTO message_mentions (message_id, user_id, created_at) VALUES ($1,'*',$2)`, message.ID, message.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func GetMessageMentions(messageID string) ([]string, bool, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT user_id FROM message_mentions WHERE message_id = $1 ORDER BY user_id`, messageID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	mentions := []string{}
	mentionAll := false
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, false, err
		}
		if userID == "*" {
			mentionAll = true
			continue
		}
		mentions = append(mentions, userID)
	}
	return mentions, mentionAll, rows.Err()
}

func IsConversationMuted(userID, conversationID string) (bool, error) {
	var muted bool
	err := db.QueryRowContext(context.Background(), `SELECT muted FROM conversation_mutes WHERE user_id = $1 AND conversation_id = $2`, userID, conversationID).Scan(&muted)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return muted, err
}

func NormalizeMentionTargets(requested, members []string, senderID string, mentionAll bool) ([]string, error) {
	memberSet := make(map[string]bool, len(members))
	for _, memberID := range members {
		memberSet[strings.TrimSpace(memberID)] = true
	}
	targets := requested
	if mentionAll {
		targets = members
	}
	seen := map[string]bool{}
	normalized := make([]string, 0, len(targets))
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" || target == senderID || seen[target] {
			continue
		}
		if !memberSet[target] {
			return nil, sql.ErrNoRows
		}
		seen[target] = true
		normalized = append(normalized, target)
	}
	if len(normalized) > 500 {
		return nil, sql.ErrNoRows
	}
	return normalized, nil
}
