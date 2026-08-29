package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/gorilla/mux"
)

// ── Knowledge Hub interactions ──

func upvoteKnowledgeIdeaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ideaID := vars["id"]
	votes, err := store.UpvoteKnowledgeIdea(ideaID, requestTenantID(r), requestUserID(r))
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "idea not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to upvote idea")
		return
	}
	writeJSON(w, map[string]int{"votes": votes})
}

func followKnowledgeExpertHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	expertID := vars["id"]
	followers, err := store.FollowKnowledgeExpert(expertID, requestTenantID(r), requestUserID(r))
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "expert not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to follow expert")
		return
	}
	writeJSON(w, map[string]int{"followers": followers})
}

func sharePostHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]
	shares, err := store.SharePost(postID, requestTenantID(r))
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to share post")
		return
	}
	writeJSON(w, map[string]int{"shares": shares})
}

// ── Message Reactions ──

type reactionRequest struct {
	UserID string `json:"userId"`
	Emoji  string `json:"emoji"`
}

func addReactionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	var req reactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Emoji == "" {
		writeError(w, http.StatusBadRequest, "emoji is required")
		return
	}
	message, accessible := requireMessageAccess(w, r, messageID)
	if !accessible {
		return
	}
	req.UserID = requestUserID(r)

	reaction, err := store.AddReaction(messageID, req.UserID, req.Emoji)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add reaction")
		return
	}
	store.BroadcastEnvelope(message.TenantID, message.ConversationID, "message.reaction.added", map[string]interface{}{
		"conversationId": message.ConversationID, "messageId": messageID, "reaction": reaction,
	})
	writeJSON(w, reaction)
}

func removeReactionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	var req reactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Emoji == "" {
		writeError(w, http.StatusBadRequest, "emoji is required")
		return
	}
	message, accessible := requireMessageAccess(w, r, messageID)
	if !accessible {
		return
	}
	req.UserID = requestUserID(r)

	if err := store.RemoveReaction(messageID, req.UserID, req.Emoji); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove reaction")
		return
	}
	store.BroadcastEnvelope(message.TenantID, message.ConversationID, "message.reaction.removed", map[string]interface{}{
		"conversationId": message.ConversationID, "messageId": messageID, "userId": req.UserID, "emoji": req.Emoji,
	})
	w.WriteHeader(http.StatusNoContent)
}

// ── Post Interactions (Likes & Comments) ──

func togglePostLikeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]

	liked, err := store.TogglePostLike(postID, requestUserID(r), requestTenantID(r))
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to toggle post like")
		return
	}
	writeJSON(w, map[string]interface{}{"liked": liked})
}

func postCommentsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]

	comments, err := store.GetPostComments(postID, requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load comments")
		return
	}
	writeJSON(w, comments)
}

func addPostCommentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]

	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	currentUser, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authenticated user is required")
		return
	}
	role := ""
	if len(currentUser.Roles) > 0 {
		role = currentUser.Roles[0]
	}

	comment, err := store.AddPostComment(model.PostComment{
		TenantID: requestTenantID(r),
		AuthorID: currentUser.ID,
		PostID:   postID,
		Author:   currentUser.Name,
		Role:     role,
		Org:      currentUser.OrganizationID,
		Text:     req.Text,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to add comment")
		return
	}
	writeJSON(w, comment)
}

// ── Read Receipts ──

func markMessageReadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	message, accessible := requireMessageAccess(w, r, messageID)
	if !accessible {
		return
	}
	req.UserID = requestUserID(r)

	if err := store.MarkMessageRead(messageID, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark message as read")
		return
	}
	store.BroadcastEnvelope(message.TenantID, message.ConversationID, "message.read", map[string]interface{}{
		"conversationId": message.ConversationID, "messageId": messageID, "userId": req.UserID,
	})
	w.WriteHeader(http.StatusNoContent)
}

// ── Pinned Messages ──

type pinMessageRequest struct {
	MessageID string `json:"messageId"`
	PinnedBy  string `json:"pinnedBy"`
}

func pinnedMessagesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	if !requireConversationAccess(w, r, conversationID) {
		return
	}

	pins, err := store.GetPinnedMessages(conversationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load pinned messages")
		return
	}
	writeJSON(w, pins)
}

func pinMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	if !requireConversationAccess(w, r, conversationID) {
		return
	}

	var req pinMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.MessageID == "" {
		writeError(w, http.StatusBadRequest, "messageId is required")
		return
	}
	message, accessible := requireMessageAccess(w, r, req.MessageID)
	if !accessible {
		return
	}
	if message.ConversationID != conversationID {
		writeError(w, http.StatusBadRequest, "message does not belong to this conversation")
		return
	}
	req.PinnedBy = requestUserID(r)

	pin, err := store.PinMessage(conversationID, req.MessageID, req.PinnedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to pin message")
		return
	}
	writeJSON(w, pin)
}

func unpinMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	if !requireConversationAccess(w, r, conversationID) {
		return
	}

	var req struct {
		MessageID string `json:"messageId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.MessageID == "" {
		writeError(w, http.StatusBadRequest, "messageId is required")
		return
	}

	if err := store.UnpinMessage(conversationID, req.MessageID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to unpin message")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Tasks ──

func tasksHandler(w http.ResponseWriter, r *http.Request) {
	conversationID := r.URL.Query().Get("conversationId")
	if conversationID != "" && !requireConversationAccess(w, r, conversationID) {
		return
	}
	tasks, err := store.GetTasksForUser(requestTenantID(r), requestUserID(r), conversationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load tasks")
		return
	}
	writeJSON(w, tasks)
}

func createTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req model.Task
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if !validateTaskInput(w, &req, false) {
		return
	}
	if req.ConversationID != "" && !requireConversationAccess(w, r, req.ConversationID) {
		return
	}
	req.CreatedBy = requestUserID(r)
	req.TenantID = requestTenantID(r)

	task, err := store.CreateTask(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, task)
}

func updateTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if !isTaskStatus(req.Status) {
		writeError(w, http.StatusBadRequest, "status must be todo, in_progress, blocked, or done")
		return
	}

	task, err := store.GetTaskForUser(taskID, requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	task.Status = req.Status
	task, err = store.UpdateTask(task)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task status")
		return
	}
	writeJSON(w, task)
}

func validateTaskInput(w http.ResponseWriter, task *model.Task, requireStatus bool) bool {
	task.Title = strings.TrimSpace(task.Title)
	if task.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return false
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	if !isTaskPriority(task.Priority) {
		writeError(w, http.StatusBadRequest, "priority must be low, medium, high, or urgent")
		return false
	}
	if requireStatus && !isTaskStatus(task.Status) {
		writeError(w, http.StatusBadRequest, "status must be todo, in_progress, blocked, or done")
		return false
	}
	if task.Status != "" && !isTaskStatus(task.Status) {
		writeError(w, http.StatusBadRequest, "status must be todo, in_progress, blocked, or done")
		return false
	}
	return true
}

func isTaskStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "todo", "in_progress", "blocked", "done":
		return true
	default:
		return false
	}
}

func isTaskPriority(priority string) bool {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "low", "medium", "high", "urgent":
		return true
	default:
		return false
	}
}

// ── Notifications ──

func notificationsHandler(w http.ResponseWriter, r *http.Request) {
	notifications, err := store.GetNotifications(requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load notifications")
		return
	}
	writeJSON(w, notifications)
}

func markNotificationReadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notificationID := vars["id"]

	if err := store.MarkNotificationRead(notificationID, requestUserID(r)); err != nil {
		writeError(w, http.StatusNotFound, "notification not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func markAllNotificationsReadHandler(w http.ResponseWriter, r *http.Request) {
	if err := store.MarkAllNotificationsRead(requestUserID(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark all notifications as read")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Presence ──

func presenceHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID != "" {
		presence, err := store.GetPresence(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load presence")
			return
		}
		writeJSON(w, presence)
		return
	}

	presence, err := store.GetAllPresence()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load presence")
		return
	}
	writeJSON(w, presence)
}

func updatePresenceHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"userId"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}
	if req.UserID == "" {
		req.UserID = requestUserID(r)
	}

	if err := store.UpdatePresence(req.UserID, req.Status); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update presence")
		return
	}
	presence, err := store.GetPresence(req.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load updated presence")
		return
	}
	writeJSON(w, presence)
}
