package api

import (
	"encoding/json"
	"net/http"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/gorilla/mux"
)

// ── Knowledge Hub interactions ──

func upvoteKnowledgeIdeaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ideaID := vars["id"]
	votes, err := store.UpvoteKnowledgeIdea(ideaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to upvote idea")
		return
	}
	writeJSON(w, map[string]int{"votes": votes})
}

func followKnowledgeExpertHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	expertID := vars["id"]
	followers, err := store.FollowKnowledgeExpert(expertID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to follow expert")
		return
	}
	writeJSON(w, map[string]int{"followers": followers})
}

func sharePostHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]
	shares, err := store.SharePost(postID)
	if err != nil {
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
	if req.UserID == "" {
		req.UserID = requestUserID(r)
	}

	reaction, err := store.AddReaction(messageID, req.UserID, req.Emoji)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add reaction")
		return
	}
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
	if req.UserID == "" {
		req.UserID = requestUserID(r)
	}

	if err := store.RemoveReaction(messageID, req.UserID, req.Emoji); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove reaction")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Post Interactions (Likes & Comments) ──

func togglePostLikeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]

	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.UserID == "" {
		req.UserID = requestUserID(r)
	}

	liked, err := store.TogglePostLike(postID, req.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle post like")
		return
	}
	writeJSON(w, map[string]interface{}{"liked": liked})
}

func postCommentsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]

	comments, err := store.GetPostComments(postID)
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
		Author string `json:"author"`
		Role   string `json:"role"`
		Org    string `json:"org"`
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	if req.Author == "" {
		req.Author = "StatChat User"
	}

	comment, err := store.AddPostComment(model.PostComment{
		PostID: postID,
		Author: req.Author,
		Role:   req.Role,
		Org:    req.Org,
		Text:   req.Text,
	})
	if err != nil {
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
	if req.UserID == "" {
		req.UserID = requestUserID(r)
	}

	if err := store.MarkMessageRead(messageID, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark message as read")
		return
	}
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

	var req pinMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.MessageID == "" {
		writeError(w, http.StatusBadRequest, "messageId is required")
		return
	}
	if req.PinnedBy == "" {
		req.PinnedBy = requestUserID(r)
	}

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
	var tasks []model.Task
	var err error

	if conversationID != "" {
		tasks, err = store.GetTasksByConversation(conversationID)
	} else {
		tasks, err = store.GetTasks()
	}
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
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = requestUserID(r)
	}

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
	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	task, err := store.UpdateTaskStatus(taskID, req.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task status")
		return
	}
	writeJSON(w, task)
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

	if err := store.MarkNotificationRead(notificationID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark notification as read")
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
