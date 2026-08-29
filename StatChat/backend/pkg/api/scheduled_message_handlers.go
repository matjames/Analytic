package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/gorilla/mux"
)

var errInvalidScheduledTime = errors.New("scheduled time must be between one second and one year in the future")

func scheduledMessagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		conversationID := strings.TrimSpace(r.URL.Query().Get("conversationId"))
		if conversationID != "" && !requireConversationAccess(w, r, conversationID) {
			return
		}
		items, err := store.ListScheduledMessages(requestTenantID(r), requestUserID(r), conversationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load scheduled messages")
			return
		}
		writeJSON(w, items)
		return
	}
	var req struct {
		ConversationID string `json:"conversationId"`
		Text           string `json:"text"`
		ScheduledFor   string `json:"scheduledFor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.ConversationID = strings.TrimSpace(req.ConversationID)
	req.Text = strings.TrimSpace(req.Text)
	scheduledFor, err := validateScheduledFor(req.ScheduledFor, time.Now().UTC())
	if req.ConversationID == "" || req.Text == "" || err != nil {
		writeError(w, http.StatusBadRequest, "conversationId, text, and a future RFC3339 scheduledFor are required")
		return
	}
	if !requireConversationAccess(w, r, req.ConversationID) {
		return
	}
	item, err := store.CreateScheduledMessage(model.ScheduledMessage{TenantID: requestTenantID(r), ConversationID: req.ConversationID, SenderID: requestUserID(r), Sender: requestUserName(r), Text: req.Text, ScheduledFor: scheduledFor})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to schedule message")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, item)
}

func cancelScheduledMessageHandler(w http.ResponseWriter, r *http.Request) {
	cancelled, err := store.CancelScheduledMessage(strings.TrimSpace(mux.Vars(r)["id"]), requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to cancel scheduled message")
		return
	}
	if !cancelled {
		writeError(w, http.StatusNotFound, "pending scheduled message not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateScheduledFor(value string, now time.Time) (time.Time, error) {
	scheduledFor, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil || !scheduledFor.After(now.Add(time.Second)) || scheduledFor.After(now.AddDate(1, 0, 0)) {
		return time.Time{}, errInvalidScheduledTime
	}
	return scheduledFor.UTC(), nil
}
