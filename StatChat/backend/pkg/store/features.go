package store

import (
	"context"
	"database/sql"
	"time"

	"statchat/pkg/model"

	"github.com/google/uuid"
)

// ── Message Reactions ──

func AddReaction(messageID string, userID string, emoji string) (model.MessageReaction, error) {
	reaction := model.MessageReaction{
		ID:        uuid.NewString(),
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: time.Now().UTC(),
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO message_reactions (id, message_id, user_id, emoji, created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (message_id, user_id, emoji) DO NOTHING`,
		reaction.ID, messageID, userID, emoji, reaction.CreatedAt)
	if err != nil {
		return reaction, err
	}
	// Fetch the user name for the reaction
	var userName string
	err = db.QueryRowContext(context.Background(), `SELECT name FROM users WHERE id = $1`, userID).Scan(&userName)
	if err == nil {
		reaction.UserName = userName
	}
	return reaction, nil
}

func RemoveReaction(messageID string, userID string, emoji string) error {
	_, err := db.ExecContext(context.Background(), `DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`, messageID, userID, emoji)
	return err
}

func GetMessageReactions(messageID string) ([]model.MessageReaction, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT r.id, r.message_id, r.user_id, u.name, r.emoji, r.created_at
FROM message_reactions r
JOIN users u ON u.id = r.user_id
WHERE r.message_id = $1
ORDER BY r.created_at`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reactions := []model.MessageReaction{}
	for rows.Next() {
		var r model.MessageReaction
		if err := rows.Scan(&r.ID, &r.MessageID, &r.UserID, &r.UserName, &r.Emoji, &r.CreatedAt); err != nil {
			return nil, err
		}
		reactions = append(reactions, r)
	}
	return reactions, rows.Err()
}

// ── Pinned Messages ──

func PinMessage(conversationID string, messageID string, pinnedBy string) (model.PinnedMessage, error) {
	pin := model.PinnedMessage{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		MessageID:      messageID,
		PinnedBy:       pinnedBy,
		PinnedAt:       time.Now().UTC(),
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO pinned_messages (id, conversation_id, message_id, pinned_by, pinned_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`,
		pin.ID, conversationID, messageID, pinnedBy, pin.PinnedAt)
	return pin, err
}

func UnpinMessage(conversationID string, messageID string) error {
	_, err := db.ExecContext(context.Background(), `DELETE FROM pinned_messages WHERE conversation_id = $1 AND message_id = $2`, conversationID, messageID)
	return err
}

func GetPinnedMessages(conversationID string) ([]model.PinnedMessage, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, conversation_id, message_id, pinned_by, pinned_at FROM pinned_messages WHERE conversation_id = $1 ORDER BY pinned_at DESC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pins := []model.PinnedMessage{}
	for rows.Next() {
		var p model.PinnedMessage
		if err := rows.Scan(&p.ID, &p.ConversationID, &p.MessageID, &p.PinnedBy, &p.PinnedAt); err != nil {
			return nil, err
		}
		pins = append(pins, p)
	}
	return pins, rows.Err()
}

func IsMessagePinned(conversationID string, messageID string) (bool, error) {
	var count int
	err := db.QueryRowContext(context.Background(), `SELECT COUNT(1) FROM pinned_messages WHERE conversation_id = $1 AND message_id = $2`, conversationID, messageID).Scan(&count)
	return count > 0, err
}

// ── Read Receipts ──

func MarkMessageRead(messageID string, userID string) error {
	_, err := db.ExecContext(context.Background(), `INSERT INTO read_receipts (id, message_id, user_id, read_at) VALUES ($1,$2,$3,$4) ON CONFLICT (message_id, user_id) DO UPDATE SET read_at = EXCLUDED.read_at`,
		uuid.NewString(), messageID, userID, time.Now().UTC())
	return err
}

func GetMessageReadBy(messageID string) ([]string, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT user_id FROM read_receipts WHERE message_id = $1`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userIDs := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
	}
	return userIDs, rows.Err()
}

// ── Tasks ──

func CreateTask(task model.Task) (model.Task, error) {
	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	if task.Status == "" {
		task.Status = "todo"
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO tasks (id, title, description, assignee, priority, due_date, status, conversation_id, created_by, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		task.ID, task.Title, nullString(task.Description), nullString(task.Assignee), task.Priority, nullString(task.DueDate), task.Status, nullString(task.ConversationID), task.CreatedBy, task.CreatedAt, nullTime(task.UpdatedAt))
	return task, err
}

func GetTasks() ([]model.Task, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, title, COALESCE(description,''), COALESCE(assignee,''), priority, COALESCE(due_date,''), status, COALESCE(conversation_id,''), created_by, created_at, COALESCE(updated_at, created_at) FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Assignee, &t.Priority, &t.DueDate, &t.Status, &t.ConversationID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func GetTasksByConversation(conversationID string) ([]model.Task, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, title, COALESCE(description,''), COALESCE(assignee,''), priority, COALESCE(due_date,''), status, COALESCE(conversation_id,''), created_by, created_at, COALESCE(updated_at, created_at) FROM tasks WHERE conversation_id = $1 ORDER BY created_at DESC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Assignee, &t.Priority, &t.DueDate, &t.Status, &t.ConversationID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func UpdateTaskStatus(taskID string, status string) (model.Task, error) {
	_, err := db.ExecContext(context.Background(), `UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3`, status, time.Now().UTC(), taskID)
	if err != nil {
		return model.Task{}, err
	}
	return GetTaskByID(taskID)
}

func GetTaskByID(taskID string) (model.Task, error) {
	var t model.Task
	err := db.QueryRowContext(context.Background(), `SELECT id, title, COALESCE(description,''), COALESCE(assignee,''), priority, COALESCE(due_date,''), status, COALESCE(conversation_id,''), created_by, created_at, COALESCE(updated_at, created_at) FROM tasks WHERE id = $1`, taskID).Scan(
		&t.ID, &t.Title, &t.Description, &t.Assignee, &t.Priority, &t.DueDate, &t.Status, &t.ConversationID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

// ── Notifications ──

func CreateNotification(notification model.Notification) (model.Notification, error) {
	if notification.ID == "" {
		notification.ID = uuid.NewString()
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO notifications (id, user_id, type, title, body, link, read, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		notification.ID, notification.UserID, notification.Type, notification.Title, notification.Body, nullString(notification.Link), notification.Read, notification.CreatedAt)
	return notification, err
}

func GetNotifications(userID string) ([]model.Notification, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, user_id, type, title, body, COALESCE(link,''), read, created_at FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := []model.Notification{}
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Link, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func MarkNotificationRead(notificationID string) error {
	_, err := db.ExecContext(context.Background(), `UPDATE notifications SET read = true WHERE id = $1`, notificationID)
	return err
}

func MarkAllNotificationsRead(userID string) error {
	_, err := db.ExecContext(context.Background(), `UPDATE notifications SET read = true WHERE user_id = $1`, userID)
	return err
}

func GetUnreadNotificationCount(userID string) (int, error) {
	var count int
	err := db.QueryRowContext(context.Background(), `SELECT COUNT(1) FROM notifications WHERE user_id = $1 AND read = false`, userID).Scan(&count)
	return count, err
}

// ── Presence ──

func UpdatePresence(userID string, status string) error {
	_, err := db.ExecContext(context.Background(), `INSERT INTO user_presence (user_id, status, updated_at) VALUES ($1,$2,$3) ON CONFLICT (user_id) DO UPDATE SET status = EXCLUDED.status, updated_at = EXCLUDED.updated_at`,
		userID, status, time.Now().UTC())
	if err != nil {
		return err
	}
	// Also update the users table presence column
	_, err = db.ExecContext(context.Background(), `UPDATE users SET presence = $1 WHERE id = $2`, status, userID)
	return err
}

func GetPresence(userID string) (model.Presence, error) {
	var p model.Presence
	err := db.QueryRowContext(context.Background(), `SELECT user_id, status, updated_at FROM user_presence WHERE user_id = $1`, userID).Scan(&p.UserID, &p.Status, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.Presence{UserID: userID, Status: "offline", UpdatedAt: time.Now().UTC()}, nil
	}
	return p, err
}

func GetAllPresence() ([]model.Presence, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT user_id, status, updated_at FROM user_presence ORDER BY user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	presence := []model.Presence{}
	for rows.Next() {
		var p model.Presence
		if err := rows.Scan(&p.UserID, &p.Status, &p.UpdatedAt); err != nil {
			return nil, err
		}
		presence = append(presence, p)
	}
	return presence, rows.Err()
}
