package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"statchat/pkg/store"

	"github.com/gorilla/mux"
)

func saveMessageHandler(w http.ResponseWriter, r *http.Request) {
	message, ok := requireMessageAccess(w, r, mux.Vars(r)["id"])
	if !ok {
		return
	}
	savedAt, err := store.SaveMessage(requestUserID(r), message.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save message")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]any{"messageId": message.ID, "savedAt": savedAt})
}

func unsaveMessageHandler(w http.ResponseWriter, r *http.Request) {
	messageID := strings.TrimSpace(mux.Vars(r)["id"])
	if messageID == "" {
		writeError(w, http.StatusBadRequest, "message id is required")
		return
	}
	if err := store.UnsaveMessage(requestUserID(r), messageID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove saved message")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func savedMessagesHandler(w http.ResponseWriter, r *http.Request) {
	filters, _, err := parseMessageSearchFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if filters.ConversationID != "" && !requireConversationAccess(w, r, filters.ConversationID) {
		return
	}
	messages, err := store.GetSavedMessages(requestUserID(r), requestTenantID(r), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load saved messages")
		return
	}
	writeJSON(w, messages)
}

func parseMessageSearchFilters(r *http.Request) (store.MessageSearchFilters, bool, error) {
	query := r.URL.Query()
	filters := store.MessageSearchFilters{
		Query:          strings.TrimSpace(query.Get("q")),
		ConversationID: strings.TrimSpace(query.Get("conversationId")),
		Sender:         strings.TrimSpace(query.Get("sender")),
	}
	if len([]rune(filters.Query)) > 200 || len([]rune(filters.Sender)) > 120 {
		return filters, false, &searchValidationError{"search values are too long"}
	}
	active := filters.ConversationID != "" || filters.Sender != ""
	if value := strings.TrimSpace(query.Get("from")); value != "" {
		parsed, err := parseSearchDate(value, false)
		if err != nil {
			return filters, false, err
		}
		filters.DateFrom = &parsed
		active = true
	}
	if value := strings.TrimSpace(query.Get("to")); value != "" {
		parsed, err := parseSearchDate(value, true)
		if err != nil {
			return filters, false, err
		}
		filters.DateTo = &parsed
		active = true
	}
	if value := strings.TrimSpace(query.Get("hasAttachment")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return filters, false, &searchValidationError{"hasAttachment must be true or false"}
		}
		filters.HasAttachment = &parsed
		active = true
	}
	if value := strings.TrimSpace(query.Get("savedOnly")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return filters, false, &searchValidationError{"savedOnly must be true or false"}
		}
		filters.SavedOnly = parsed
		active = active || parsed
	}
	if filters.DateFrom != nil && filters.DateTo != nil && !filters.DateFrom.Before(*filters.DateTo) {
		return filters, false, &searchValidationError{"from must be before to"}
	}
	return filters, active, nil
}

type searchValidationError struct{ message string }

func (e *searchValidationError) Error() string { return e.message }

func parseSearchDate(value string, endOfDay bool) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, &searchValidationError{"dates must use YYYY-MM-DD or RFC3339"}
	}
	if endOfDay {
		parsed = parsed.AddDate(0, 0, 1)
	}
	return parsed.UTC(), nil
}
