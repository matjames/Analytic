package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"statchat/pkg/model"

	"github.com/google/uuid"
)

func CreateChannel(channel model.Channel, creatorID string) (model.Channel, error) {
	channel.ID = strings.TrimSpace(channel.ID)
	if channel.ID == "" {
		channel.ID = "channel-" + uuid.NewString()
	}
	channel.TenantID = normalizedTenantID(channel.TenantID)
	channel.Name = strings.TrimSpace(channel.Name)
	channel.Visibility = strings.ToLower(strings.TrimSpace(channel.Visibility))
	if channel.Visibility == "" {
		channel.Visibility = "public"
	}
	channel.CreatedBy = creatorID
	channel.CreatedAt = time.Now().UTC()
	members, _ := json.Marshal([]string{creatorID})
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return channel, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO channels (id, tenant_id, name, description, visibility, created_by, created_at, archived) VALUES ($1,$2,$3,$4,$5,$6,$7,FALSE)`,
		channel.ID, channel.TenantID, channel.Name, nullString(channel.Description), channel.Visibility, creatorID, channel.CreatedAt)
	if err != nil {
		return channel, err
	}
	_, err = tx.Exec(`INSERT INTO conversations (id, tenant_id, name, type, channel_id, member_ids, category) VALUES ($1,$2,$3,'channel',$1,$4,'channel')`,
		channel.ID, channel.TenantID, channel.Name, members)
	if err != nil {
		return channel, err
	}
	_, err = tx.Exec(`INSERT INTO conversation_roles (conversation_id, user_id, role) VALUES ($1,$2,'owner')`, channel.ID, creatorID)
	if err != nil {
		return channel, err
	}
	if err := tx.Commit(); err != nil {
		return channel, err
	}
	channel.MemberCount = 1
	channel.Joined = true
	return channel, nil
}

func GetChannelsForUser(tenantID string, userID string) ([]model.Channel, error) {
	tenantID = normalizedTenantID(tenantID)
	rows, err := db.QueryContext(context.Background(), `
SELECT ch.id, ch.tenant_id, ch.name, COALESCE(ch.description,''), ch.visibility,
       COALESCE(ch.created_by,''), ch.created_at,
       jsonb_array_length(COALESCE(c.member_ids, '[]'::jsonb)),
       COALESCE(c.member_ids, '[]'::jsonb) @> to_jsonb($2::text), ch.archived
FROM channels ch
JOIN conversations c ON c.channel_id = ch.id
WHERE (ch.tenant_id = $1 OR ch.tenant_id = 'default') AND NOT ch.archived
ORDER BY lower(ch.name)`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	channels := []model.Channel{}
	for rows.Next() {
		var channel model.Channel
		if err := rows.Scan(&channel.ID, &channel.TenantID, &channel.Name, &channel.Description, &channel.Visibility, &channel.CreatedBy, &channel.CreatedAt, &channel.MemberCount, &channel.Joined, &channel.Archived); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, rows.Err()
}

func ArchiveChannel(channelID, tenantID, userID string) error {
	result, err := db.ExecContext(context.Background(), `UPDATE channels SET archived = TRUE WHERE id = $1 AND tenant_id = $2 AND created_by = $3 AND archived = FALSE`, channelID, normalizedTenantID(tenantID), userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func GetChannelForUser(channelID string, tenantID string, userID string) (model.Channel, error) {
	channels, err := GetChannelsForUser(tenantID, userID)
	if err != nil {
		return model.Channel{}, err
	}
	for _, channel := range channels {
		if channel.ID == channelID {
			return channel, nil
		}
	}
	return model.Channel{}, sql.ErrNoRows
}

func AddConversationMember(conversationID string, userID string) error {
	members, err := GetConversationMembers(conversationID)
	if err != nil {
		return err
	}
	for _, member := range members {
		if member == userID {
			return nil
		}
	}
	members = append(members, userID)
	encoded, err := json.Marshal(members)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(context.Background(), `UPDATE conversations SET member_ids = $1 WHERE id = $2`, encoded, conversationID)
	return err
}

type ConversationMemberAccess struct {
	ConversationID string
	Type           model.ConversationType
	MemberIDs      []string
	OwnerID        string
	ActorRole      string
}

func backfillConversationOwners(ctx context.Context) error {
	_, err := db.ExecContext(ctx, `
INSERT INTO conversation_roles (conversation_id, user_id, role)
SELECT c.id, c.member_ids->>0, 'owner'
FROM conversations c
WHERE c.type <> 'direct'
  AND jsonb_typeof(c.member_ids) = 'array'
  AND jsonb_array_length(c.member_ids) > 0
  AND NOT EXISTS (
    SELECT 1 FROM conversation_roles role
    WHERE role.conversation_id = c.id AND role.role = 'owner'
  )
ON CONFLICT (conversation_id, user_id) DO UPDATE SET role = 'owner'`)
	return err
}

func EnsureConversationOwner(conversationID, userID string) error {
	conversationID = strings.TrimSpace(conversationID)
	userID = strings.TrimSpace(userID)
	if conversationID == "" || userID == "" {
		return errors.New("conversation and owner are required")
	}
	_, err := db.ExecContext(context.Background(), `
INSERT INTO conversation_roles (conversation_id, user_id, role)
SELECT $1, $2, 'owner'
WHERE EXISTS (SELECT 1 FROM conversations WHERE id = $1 AND type <> 'direct')
  AND NOT EXISTS (
    SELECT 1 FROM conversation_roles WHERE conversation_id = $1 AND role = 'owner'
  )
ON CONFLICT (conversation_id, user_id) DO UPDATE SET role = 'owner'`, conversationID, userID)
	return err
}

func GetConversationMemberAccess(conversationID, tenantID, actorID string) (ConversationMemberAccess, error) {
	access := ConversationMemberAccess{ConversationID: conversationID}
	var conversationType string
	var encodedMembers []byte
	err := db.QueryRowContext(context.Background(), `
SELECT c.type, c.member_ids,
       COALESCE((SELECT user_id FROM conversation_roles WHERE conversation_id = c.id AND role = 'owner'), ''),
       COALESCE((SELECT role FROM conversation_roles WHERE conversation_id = c.id AND user_id = $3), '')
FROM conversations c
WHERE c.id = $1 AND c.tenant_id = $2`, conversationID, normalizedTenantID(tenantID), actorID).Scan(
		&conversationType, &encodedMembers, &access.OwnerID, &access.ActorRole,
	)
	if err != nil {
		return access, err
	}
	access.Type = model.ConversationType(conversationType)
	if err := json.Unmarshal(encodedMembers, &access.MemberIDs); err != nil {
		return access, err
	}
	return access, nil
}

func RemoveConversationMember(conversationID string, userID string) error {
	members, err := GetConversationMembers(conversationID)
	if err != nil {
		return err
	}
	next := make([]string, 0, len(members))
	for _, member := range members {
		if member != userID {
			next = append(next, member)
		}
	}
	encoded, err := json.Marshal(next)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(context.Background(), `UPDATE conversations SET member_ids = $1 WHERE id = $2`, encoded, conversationID)
	return err
}

func GetTaskForUser(taskID string, tenantID string, userID string) (model.Task, error) {
	var task model.Task
	err := db.QueryRowContext(context.Background(), `
SELECT id, tenant_id, title, COALESCE(description,''), COALESCE(assignee,''), priority,
       COALESCE(due_date,''), status, COALESCE(conversation_id,''), created_by,
       created_at, COALESCE(updated_at, created_at)
FROM tasks WHERE id = $1 AND tenant_id = $2
  AND (created_by = $3 OR assignee = $3 OR conversation_id IN (
    SELECT id FROM conversations WHERE type = 'channel' OR member_ids @> to_jsonb($3::text)
  ))`, taskID, normalizedTenantID(tenantID), userID).Scan(
		&task.ID, &task.TenantID, &task.Title, &task.Description, &task.Assignee, &task.Priority,
		&task.DueDate, &task.Status, &task.ConversationID, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt)
	return task, err
}

func GetTasksForUser(tenantID string, userID string, conversationID string) ([]model.Task, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT id, tenant_id, title, COALESCE(description,''), COALESCE(assignee,''), priority,
       COALESCE(due_date,''), status, COALESCE(conversation_id,''), created_by,
       created_at, COALESCE(updated_at, created_at)
FROM tasks WHERE tenant_id = $1 AND ($3 = '' OR conversation_id = $3)
  AND (created_by = $2 OR assignee = $2 OR conversation_id IN (
    SELECT id FROM conversations WHERE type = 'channel' OR member_ids @> to_jsonb($2::text)
  ))
ORDER BY created_at DESC`, normalizedTenantID(tenantID), userID, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := []model.Task{}
	for rows.Next() {
		var task model.Task
		if err := rows.Scan(&task.ID, &task.TenantID, &task.Title, &task.Description, &task.Assignee, &task.Priority, &task.DueDate, &task.Status, &task.ConversationID, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func UpdateTask(task model.Task) (model.Task, error) {
	task.UpdatedAt = time.Now().UTC()
	result, err := db.ExecContext(context.Background(), `UPDATE tasks SET title=$1, description=$2, assignee=$3, priority=$4, due_date=$5, status=$6, conversation_id=$7, updated_at=$8 WHERE id=$9 AND tenant_id=$10`,
		task.Title, nullString(task.Description), nullString(task.Assignee), task.Priority, nullString(task.DueDate), task.Status, nullString(task.ConversationID), task.UpdatedAt, task.ID, task.TenantID)
	if err != nil {
		return model.Task{}, err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return model.Task{}, sql.ErrNoRows
	}
	return task, nil
}

func DeleteTask(taskID string, tenantID string, userID string) error {
	result, err := db.ExecContext(context.Background(), `DELETE FROM tasks WHERE id=$1 AND tenant_id=$2 AND created_by=$3`, taskID, normalizedTenantID(tenantID), userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func CreateCalendarEvent(event model.CalendarEvent) (model.CalendarEvent, error) {
	event.ID = uuid.NewString()
	event.TenantID = normalizedTenantID(event.TenantID)
	event.CreatedAt = time.Now().UTC()
	event.UpdatedAt = event.CreatedAt
	attendees, err := json.Marshal(event.AttendeeIDs)
	if err != nil {
		return event, err
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO calendar_events (id,tenant_id,title,description,location,start_at,end_at,all_day,conversation_id,attendee_ids,created_by,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		event.ID, event.TenantID, event.Title, nullString(event.Description), nullString(event.Location), event.StartAt, event.EndAt, event.AllDay, nullString(event.ConversationID), attendees, event.CreatedBy, event.CreatedAt, event.UpdatedAt)
	return event, err
}

func GetCalendarEvents(tenantID string, userID string, from time.Time, to time.Time) ([]model.CalendarEvent, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT id,tenant_id,title,COALESCE(description,''),COALESCE(location,''),start_at,end_at,all_day,
       COALESCE(conversation_id,''),attendee_ids,created_by,created_at,updated_at
FROM calendar_events
WHERE tenant_id=$1 AND (created_by=$2 OR attendee_ids @> to_jsonb($2::text))
  AND ($3::timestamptz IS NULL OR end_at >= $3) AND ($4::timestamptz IS NULL OR start_at <= $4)
ORDER BY start_at`, normalizedTenantID(tenantID), userID, nullTime(from), nullTime(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []model.CalendarEvent{}
	for rows.Next() {
		var event model.CalendarEvent
		var attendees []byte
		if err := rows.Scan(&event.ID, &event.TenantID, &event.Title, &event.Description, &event.Location, &event.StartAt, &event.EndAt, &event.AllDay, &event.ConversationID, &attendees, &event.CreatedBy, &event.CreatedAt, &event.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(attendees, &event.AttendeeIDs); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func GetCalendarEvent(eventID string, tenantID string, userID string) (model.CalendarEvent, error) {
	events, err := GetCalendarEvents(tenantID, userID, time.Time{}, time.Time{})
	if err != nil {
		return model.CalendarEvent{}, err
	}
	for _, event := range events {
		if event.ID == eventID {
			return event, nil
		}
	}
	return model.CalendarEvent{}, sql.ErrNoRows
}

func UpdateCalendarEvent(event model.CalendarEvent) (model.CalendarEvent, error) {
	event.UpdatedAt = time.Now().UTC()
	attendees, err := json.Marshal(event.AttendeeIDs)
	if err != nil {
		return event, err
	}
	result, err := db.ExecContext(context.Background(), `UPDATE calendar_events SET title=$1,description=$2,location=$3,start_at=$4,end_at=$5,all_day=$6,conversation_id=$7,attendee_ids=$8,updated_at=$9 WHERE id=$10 AND tenant_id=$11 AND created_by=$12`,
		event.Title, nullString(event.Description), nullString(event.Location), event.StartAt, event.EndAt, event.AllDay, nullString(event.ConversationID), attendees, event.UpdatedAt, event.ID, event.TenantID, event.CreatedBy)
	if err != nil {
		return event, err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return event, sql.ErrNoRows
	}
	return event, nil
}

func DeleteCalendarEvent(eventID string, tenantID string, userID string) error {
	result, err := db.ExecContext(context.Background(), `DELETE FROM calendar_events WHERE id=$1 AND tenant_id=$2 AND created_by=$3`, eventID, normalizedTenantID(tenantID), userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

var ErrInvalidTimeRange = errors.New("end time must be after start time")
