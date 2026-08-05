package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"statchat/pkg/model"

	"github.com/google/uuid"
)

// ── Global Search ──

type SearchResult struct {
	Users         []model.User         `json:"users"`
	Conversations []model.Conversation `json:"conversations"`
	Messages      []model.Message      `json:"messages"`
	Channels      []model.Channel      `json:"channels"`
}

func GlobalSearch(query string, userID string) (SearchResult, error) {
	result := SearchResult{
		Users:         []model.User{},
		Conversations: []model.Conversation{},
		Messages:      []model.Message{},
		Channels:      []model.Channel{},
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return result, nil
	}

	like := "%" + query + "%"

	// Users
	userRows, err := db.QueryContext(context.Background(), `
SELECT id, name, email, organization_id, roles, avatar_url, about, presence
FROM users WHERE name ILIKE $1 OR email ILIKE $1 OR organization_id ILIKE $1 ORDER BY name LIMIT 20`, like)
	if err != nil {
		return result, err
	}
	for userRows.Next() {
		var u model.User
		var rolesJSON []byte
		var avatarURL sql.NullString
		var about sql.NullString
		var presence sql.NullString
		if err := userRows.Scan(&u.ID, &u.Name, &u.Email, &u.OrganizationID, &rolesJSON, &avatarURL, &about, &presence); err != nil {
			userRows.Close()
			return result, err
		}
		json.Unmarshal(rolesJSON, &u.Roles)
		u.AvatarURL = avatarURL.String
		u.About = about.String
		u.Presence = presence.String
		result.Users = append(result.Users, u)
	}
	userRows.Close()

	// Channels
	channelRows, err := db.QueryContext(context.Background(), `
SELECT id, name FROM channels WHERE name ILIKE $1 ORDER BY name LIMIT 20`, like)
	if err != nil {
		return result, err
	}
	for channelRows.Next() {
		var c model.Channel
		if err := channelRows.Scan(&c.ID, &c.Name); err != nil {
			channelRows.Close()
			return result, err
		}
		result.Channels = append(result.Channels, c)
	}
	channelRows.Close()

	// Messages (only ones in conversations the user belongs to)
	messageRows, err := db.QueryContext(context.Background(), `
SELECT m.id, m.conversation_id, m.channel_id, m.sender, m.text, m.created_at, m.updated_at, m.deleted_at, m.parent_message_id, m.thread_root_id, m.status, m.tenant_id, m.delivery_status
FROM messages m
JOIN conversations c ON c.id = m.conversation_id
WHERE m.status != 'deleted' AND m.text ILIKE $1
  AND (c.member_ids::jsonb @> to_jsonb($2::text) OR c.type = 'channel')
ORDER BY m.created_at DESC LIMIT 30`, like, userID)
	if err != nil {
		return result, err
	}
	for messageRows.Next() {
		var msg model.Message
		var channelID sql.NullString
		var updatedAt sql.NullTime
		var deletedAt sql.NullTime
		var parentID sql.NullString
		var threadRootID sql.NullString
		if err := messageRows.Scan(&msg.ID, &msg.ConversationID, &channelID, &msg.Sender, &msg.Text, &msg.CreatedAt, &updatedAt, &deletedAt, &parentID, &threadRootID, &msg.Status, &msg.TenantID, &msg.DeliveryStatus); err != nil {
			messageRows.Close()
			return result, err
		}
		msg.ChannelID = channelID.String
		if updatedAt.Valid {
			msg.UpdatedAt = updatedAt.Time
		}
		if deletedAt.Valid {
			msg.DeletedAt = deletedAt.Time
		}
		msg.ParentMessageID = parentID.String
		msg.ThreadRootID = threadRootID.String
		result.Messages = append(result.Messages, msg)
	}
	messageRows.Close()

	// Conversations the user belongs to
	convRows, err := db.QueryContext(context.Background(), `
SELECT id, name, type, channel_id, member_ids, COALESCE(category,'')
FROM conversations WHERE name ILIKE $1
  AND (member_ids::jsonb @> to_jsonb($2::text) OR type = 'channel')
ORDER BY name LIMIT 20`, like, userID)
	if err != nil {
		return result, err
	}
	for convRows.Next() {
		var c model.Conversation
		var memberIDsJSON []byte
		var channelID sql.NullString
		var convType string
		var category string
		if err := convRows.Scan(&c.ID, &c.Name, &convType, &channelID, &memberIDsJSON, &category); err != nil {
			convRows.Close()
			return result, err
		}
		c.Type = model.ConversationType(convType)
		c.ChannelID = channelID.String
		c.Category = category
		json.Unmarshal(memberIDsJSON, &c.MemberIDs)
		result.Conversations = append(result.Conversations, c)
	}
	convRows.Close()

	return result, nil
}

// ── Favourite Conversations ──

func ensureFavouriteSchema(ctx context.Context) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS favourite_conversations (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  conversation_id TEXT NOT NULL REFERENCES conversations(id),
  created_at TIMESTAMPTZ NOT NULL,
  UNIQUE (user_id, conversation_id)
);
CREATE TABLE IF NOT EXISTS conversation_mutes (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  conversation_id TEXT NOT NULL REFERENCES conversations(id),
  muted BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  UNIQUE (user_id, conversation_id)
);
ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS wallpaper TEXT;`)
	return err
}

func ToggleFavourite(userID string, conversationID string) (bool, error) {
	var count int
	err := db.QueryRowContext(context.Background(), `SELECT COUNT(1) FROM favourite_conversations WHERE user_id = $1 AND conversation_id = $2`, userID, conversationID).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		_, err = db.ExecContext(context.Background(), `DELETE FROM favourite_conversations WHERE user_id = $1 AND conversation_id = $2`, userID, conversationID)
		return false, err
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO favourite_conversations (id, user_id, conversation_id, created_at) VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
		uuid.NewString(), userID, conversationID, time.Now().UTC())
	return true, err
}

func GetFavouriteIds(userID string) (map[string]bool, error) {
	result := map[string]bool{}
	rows, err := db.QueryContext(context.Background(), `SELECT conversation_id FROM favourite_conversations WHERE user_id = $1`, userID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return result, err
		}
		result[id] = true
	}
	return result, rows.Err()
}

func IsFavourite(userID string, conversationID string) (bool, error) {
	var count int
	err := db.QueryRowContext(context.Background(), `SELECT COUNT(1) FROM favourite_conversations WHERE user_id = $1 AND conversation_id = $2`, userID, conversationID).Scan(&count)
	return count > 0, err
}

// ── Conversation Mutes ──

func SetMuteConversation(userID string, conversationID string, muted bool) error {
	_, err := db.ExecContext(context.Background(), `INSERT INTO conversation_mutes (id, user_id, conversation_id, muted, created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (user_id, conversation_id) DO UPDATE SET muted = EXCLUDED.muted`,
		uuid.NewString(), userID, conversationID, muted, time.Now().UTC())
	return err
}

func GetMutedConversationIds(userID string) (map[string]bool, error) {
	result := map[string]bool{}
	rows, err := db.QueryContext(context.Background(), `SELECT conversation_id FROM conversation_mutes WHERE user_id = $1 AND muted = true`, userID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return result, err
		}
		result[id] = true
	}
	return result, rows.Err()
}

// ── Conversation Member helpers ──

func GetConversationMembers(conversationID string) ([]string, error) {
	var memberIDsJSON []byte
	err := db.QueryRowContext(context.Background(), `SELECT member_ids FROM conversations WHERE id = $1`, conversationID).Scan(&memberIDsJSON)
	if err != nil {
		return nil, err
	}
	var ids []string
	if err := json.Unmarshal(memberIDsJSON, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// ── Clear Conversation Messages ──

func ClearConversationMessages(conversationID string) error {
	_, err := db.ExecContext(context.Background(), `UPDATE messages SET status = 'deleted', deleted_at = $1 WHERE conversation_id = $2 AND status != 'deleted'`,
		time.Now().UTC(), conversationID)
	return err
}

// ── Real-time envelope broadcast ──

func BroadcastEnvelope(tenantID string, conversationID string, event string, payload interface{}) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	roomKey := conversationRouteKey(tenantID, conversationID)
	envelope := model.GatewayEnvelope{
		Event:    event,
		TenantID: tenantID,
		Payload:  payload,
	}
	for client := range clients {
		if client.conversation != roomKey {
			continue
		}
		if err := client.conn.WriteJSON(envelope); err != nil {
			client.conn.Close()
			delete(clients, client)
		}
	}
}

// ── Notify conversation members about a new message ──

func NotifyConversationMembers(conversationID string, senderUserID string, messageText string, link string) error {
	memberIDs, err := GetConversationMembers(conversationID)
	if err != nil {
		// If members can't be resolved, fall back to notifying nothing.
		return nil
	}
	// Determine conversation name for the notification title.
	var convName string
	_ = db.QueryRowContext(context.Background(), `SELECT name FROM conversations WHERE id = $1`, conversationID).Scan(&convName)
	if convName == "" {
		convName = conversationID
	}

	preview := messageText
	if len(preview) > 120 {
		preview = preview[:120] + "…"
	}

	for _, memberID := range memberIDs {
		if memberID == senderUserID {
			continue
		}
		_, err := CreateNotification(model.Notification{
			UserID: memberID,
			Type:   "message",
			Title:  "New message",
			Body:   preview,
			Link:   link,
		})
		if err != nil {
			// Continue notifying other members even if one fails.
			continue
		}
	}
	return nil
}
