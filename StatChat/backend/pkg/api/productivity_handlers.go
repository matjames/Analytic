package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/gorilla/mux"
)

func createChannelHandler(w http.ResponseWriter, r *http.Request) {
	var channel model.Channel
	if err := json.NewDecoder(r.Body).Decode(&channel); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	channel.Name = strings.TrimSpace(channel.Name)
	if channel.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if channel.Visibility != "" && channel.Visibility != "public" && channel.Visibility != "private" {
		writeError(w, http.StatusBadRequest, "visibility must be public or private")
		return
	}
	channel.TenantID = requestTenantID(r)
	created, err := store.CreateChannel(channel, requestUserID(r))
	if err != nil {
		writeError(w, http.StatusConflict, "channel name already exists")
		return
	}
	store.BroadcastEnvelope(created.TenantID, created.ID, "channel.created", created)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, created)
}

func channelDetailHandler(w http.ResponseWriter, r *http.Request) {
	channel, err := store.GetChannelForUser(mux.Vars(r)["id"], requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}
	if channel.Visibility == "private" && !channel.Joined {
		writeError(w, http.StatusForbidden, "this channel is private")
		return
	}
	writeJSON(w, channel)
}

func joinChannelHandler(w http.ResponseWriter, r *http.Request) {
	channel, err := store.GetChannelForUser(mux.Vars(r)["id"], requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}
	if channel.Visibility == "private" && channel.CreatedBy != requestUserID(r) && !channel.Joined {
		writeError(w, http.StatusForbidden, "an invitation is required for this private channel")
		return
	}
	if err := store.AddConversationMember(channel.ID, requestUserID(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to join channel")
		return
	}
	store.BroadcastEnvelope(channel.TenantID, channel.ID, "channel.join", map[string]string{"channelId": channel.ID, "userId": requestUserID(r)})
	w.WriteHeader(http.StatusNoContent)
}

func leaveChannelHandler(w http.ResponseWriter, r *http.Request) {
	channel, err := store.GetChannelForUser(mux.Vars(r)["id"], requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}
	if channel.CreatedBy == requestUserID(r) {
		writeError(w, http.StatusConflict, "the channel owner cannot leave")
		return
	}
	if err := store.RemoveConversationMember(channel.ID, requestUserID(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to leave channel")
		return
	}
	store.BroadcastEnvelope(channel.TenantID, channel.ID, "channel.leave", map[string]string{"channelId": channel.ID, "userId": requestUserID(r)})
	w.WriteHeader(http.StatusNoContent)
}

func archiveChannelHandler(w http.ResponseWriter, r *http.Request) {
	channelID := mux.Vars(r)["id"]
	channel, err := store.GetChannelForUser(channelID, requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}
	if channel.CreatedBy != requestUserID(r) {
		writeError(w, http.StatusForbidden, "only the channel owner can archive it")
		return
	}
	if err := store.ArchiveChannel(channelID, requestTenantID(r), requestUserID(r)); err != nil {
		writeError(w, http.StatusConflict, "channel is already archived or unavailable")
		return
	}
	store.BroadcastEnvelope(channel.TenantID, channel.ID, "channel.archived", channel)
	w.WriteHeader(http.StatusNoContent)
}

func conversationMembersHandler(w http.ResponseWriter, r *http.Request) {
	conversationID := mux.Vars(r)["id"]
	access, err := store.GetConversationMemberAccess(conversationID, requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	isMember := containsString(access.MemberIDs, requestUserID(r))
	canManage := canManageConversationMembers(r, access)
	if !isMember && !canManage {
		writeError(w, http.StatusForbidden, "conversation membership is not available")
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, map[string]any{
			"memberIds": access.MemberIDs, "ownerId": access.OwnerID, "canManage": canManage,
		})
		return
	}
	if access.Type == model.ConversationTypeDirect {
		writeError(w, http.StatusConflict, "direct conversation membership cannot be changed")
		return
	}
	if !canManage {
		writeError(w, http.StatusForbidden, "conversation owner or administrator role required")
		return
	}
	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.UserID) == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	directory, err := registryDirectory(r.Context(), strings.TrimSpace(r.Header.Get("Authorization")), requestTenantID(r))
	if err != nil || !directoryContainsUser(directory, req.UserID) {
		writeError(w, http.StatusBadRequest, "user is not available in this tenant")
		return
	}
	if err := store.AddConversationMember(conversationID, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add member")
		return
	}
	store.BroadcastEnvelope(requestTenantID(r), conversationID, "conversation.member.added", map[string]string{"conversationId": conversationID, "userId": req.UserID})
	w.WriteHeader(http.StatusNoContent)
}

func directoryContainsUser(users []model.User, userID string) bool {
	for _, user := range users {
		if user.ID == userID {
			return true
		}
	}
	return false
}

func removeConversationMemberHandler(w http.ResponseWriter, r *http.Request) {
	conversationID := mux.Vars(r)["id"]
	access, err := store.GetConversationMemberAccess(conversationID, requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if access.Type == model.ConversationTypeDirect {
		writeError(w, http.StatusConflict, "direct conversation membership cannot be changed")
		return
	}
	if !canManageConversationMembers(r, access) {
		writeError(w, http.StatusForbidden, "conversation owner or administrator role required")
		return
	}
	userID := strings.TrimSpace(mux.Vars(r)["userId"])
	if userID == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}
	if userID == access.OwnerID {
		writeError(w, http.StatusConflict, "conversation owner cannot be removed")
		return
	}
	if !containsString(access.MemberIDs, userID) {
		writeError(w, http.StatusNotFound, "conversation member not found")
		return
	}
	if err := store.RemoveConversationMember(conversationID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}
	store.BroadcastEnvelope(requestTenantID(r), conversationID, "conversation.member.removed", map[string]string{"conversationId": conversationID, "userId": userID})
	w.WriteHeader(http.StatusNoContent)
}

func canManageConversationMembers(r *http.Request, access store.ConversationMemberAccess) bool {
	if access.ActorRole == "owner" || access.ActorRole == "admin" {
		return true
	}
	identity, ok := r.Context().Value(requestIdentityKey).(model.User)
	if !ok {
		return false
	}
	for _, role := range identity.Roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "admin", "superadmin", "tenant_admin", "platform_admin":
			return true
		}
	}
	return false
}

func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := mux.Vars(r)["id"]
	existing, err := store.GetTaskForUser(taskID, requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	var update model.Task
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if strings.TrimSpace(update.Title) != "" {
		existing.Title = strings.TrimSpace(update.Title)
	}
	existing.Description = update.Description
	existing.Assignee = update.Assignee
	if update.Priority != "" {
		existing.Priority = update.Priority
	}
	existing.DueDate = update.DueDate
	if update.Status != "" {
		existing.Status = update.Status
	}
	if !validateTaskInput(w, &existing, true) {
		return
	}
	if update.ConversationID != "" {
		if !requireConversationAccess(w, r, update.ConversationID) {
			return
		}
		existing.ConversationID = update.ConversationID
	}
	updated, err := store.UpdateTask(existing)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	writeJSON(w, updated)
}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	if err := store.DeleteTask(mux.Vars(r)["id"], requestTenantID(r), requestUserID(r)); err != nil {
		writeError(w, http.StatusNotFound, "task not found or you are not its creator")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func calendarEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		from, ok := optionalRFC3339(w, r.URL.Query().Get("from"))
		if !ok {
			return
		}
		to, ok := optionalRFC3339(w, r.URL.Query().Get("to"))
		if !ok {
			return
		}
		if !from.IsZero() && !to.IsZero() && to.Before(from) {
			writeError(w, http.StatusBadRequest, "to must be after from")
			return
		}
		events, err := store.GetCalendarEvents(requestTenantID(r), requestUserID(r), from, to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load calendar events")
			return
		}
		writeJSON(w, events)
		return
	}
	var event model.CalendarEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if !validateCalendarEvent(w, r, &event) {
		return
	}
	created, err := store.CreateCalendarEvent(event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create calendar event")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, created)
}

func calendarEventHandler(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["id"]
	existing, err := store.GetCalendarEvent(eventID, requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "calendar event not found")
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, existing)
		return
	}
	if existing.CreatedBy != requestUserID(r) {
		writeError(w, http.StatusForbidden, "only the event creator can modify it")
		return
	}
	if r.Method == http.MethodDelete {
		if err := store.DeleteCalendarEvent(eventID, requestTenantID(r), requestUserID(r)); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete calendar event")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var update model.CalendarEvent
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	update.ID = existing.ID
	update.TenantID = existing.TenantID
	update.CreatedBy = existing.CreatedBy
	update.CreatedAt = existing.CreatedAt
	if !validateCalendarEvent(w, r, &update) {
		return
	}
	updated, err := store.UpdateCalendarEvent(update)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update calendar event")
		return
	}
	writeJSON(w, updated)
}

func validateCalendarEvent(w http.ResponseWriter, r *http.Request, event *model.CalendarEvent) bool {
	event.Title = strings.TrimSpace(event.Title)
	if event.Title == "" || event.StartAt.IsZero() || event.EndAt.IsZero() {
		writeError(w, http.StatusBadRequest, "title, startAt and endAt are required")
		return false
	}
	if !event.EndAt.After(event.StartAt) {
		writeError(w, http.StatusBadRequest, "endAt must be after startAt")
		return false
	}
	if event.ConversationID != "" && !requireConversationAccess(w, r, event.ConversationID) {
		return false
	}
	event.TenantID = requestTenantID(r)
	event.CreatedBy = requestUserID(r)
	if !containsString(event.AttendeeIDs, requestUserID(r)) {
		event.AttendeeIDs = append(event.AttendeeIDs, requestUserID(r))
	}
	return true
}

func optionalRFC3339(w http.ResponseWriter, value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "date filters must use RFC3339")
		return time.Time{}, false
	}
	return parsed, true
}
