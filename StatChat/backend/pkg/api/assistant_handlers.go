package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"statchat/pkg/model"
	"statchat/pkg/store"
)

type assistantRequest struct {
	ConversationID string `json:"conversationId"`
	Mode           string `json:"mode"`
}

func assistantHandler(w http.ResponseWriter, r *http.Request) {
	var req assistantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.ConversationID = strings.TrimSpace(req.ConversationID)
	req.Mode = strings.ToLower(strings.TrimSpace(req.Mode))
	if req.ConversationID == "" || !isAssistantMode(req.Mode) {
		writeError(w, http.StatusBadRequest, "conversationId and a valid mode are required")
		return
	}
	if !requireConversationAccess(w, r, req.ConversationID) {
		return
	}
	messages, err := store.GetMessagesForTenant(req.ConversationID, requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load conversation context")
		return
	}
	response := buildAssistantResponse(req.Mode, messages)
	writeJSON(w, map[string]string{"mode": req.Mode, "response": response})
}

func isAssistantMode(mode string) bool {
	switch mode {
	case "summary", "actions", "draft":
		return true
	default:
		return false
	}
}

func buildAssistantResponse(mode string, messages []model.Message) string {
	visible := make([]model.Message, 0, len(messages))
	for _, message := range messages {
		if strings.TrimSpace(message.Text) != "" && message.Status != "deleted" {
			visible = append(visible, message)
		}
	}
	if len(visible) == 0 {
		return "There are no visible messages to analyse yet."
	}
	if len(visible) > 20 {
		visible = visible[len(visible)-20:]
	}

	switch mode {
	case "actions":
		actions := make([]string, 0)
		for _, message := range visible {
			lower := strings.ToLower(message.Text)
			if strings.Contains(lower, "please") || strings.Contains(lower, "need to") || strings.Contains(lower, "must") || strings.Contains(lower, "todo") || strings.Contains(lower, "will ") {
				actions = append(actions, fmt.Sprintf("- %s: %s", message.Sender, truncateAssistantText(message.Text, 180)))
			}
		}
		if len(actions) == 0 {
			return "No explicit action items were found in the recent conversation."
		}
		return "Action items from the recent conversation:\n" + strings.Join(actions, "\n")
	case "draft":
		latest := visible[len(visible)-1]
		return fmt.Sprintf("Thanks, %s. I have noted this: %s I will confirm the next steps shortly.", firstName(latest.Sender), truncateAssistantText(latest.Text, 160))
	default:
		start := 0
		if len(visible) > 6 {
			start = len(visible) - 6
		}
		lines := make([]string, 0, len(visible)-start)
		for _, message := range visible[start:] {
			lines = append(lines, fmt.Sprintf("- %s: %s", message.Sender, truncateAssistantText(message.Text, 160)))
		}
		return "Recent conversation summary:\n" + strings.Join(lines, "\n")
	}
}

func truncateAssistantText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= limit {
		return value
	}
	return value[:limit-3] + "..."
}

func firstName(value string) string {
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return "there"
	}
	return parts[0]
}
