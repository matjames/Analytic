package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"statchat/pkg/store"

	"github.com/gorilla/mux"
)

// ── Global Search ──

func searchHandler(w http.ResponseWriter, r *http.Request) {
	filters, filtersActive, err := parseMessageSearchFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	query := filters.Query
	result, err := store.GlobalSearch(query, requestUserID(r), requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to run search")
		return
	}
	if filters.ConversationID != "" && !requireConversationAccess(w, r, filters.ConversationID) {
		return
	}
	if query != "" || filtersActive {
		result.Messages, err = store.SearchMessages(requestUserID(r), requestTenantID(r), filters)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to search messages")
			return
		}
	}
	if query != "" {
		if users, directoryErr := registryDirectory(r.Context(), strings.TrimSpace(r.Header.Get("Authorization")), requestTenantID(r)); directoryErr == nil {
			needle := strings.ToLower(query)
			result.Users = result.Users[:0]
			for _, user := range users {
				if strings.Contains(strings.ToLower(user.Name), needle) || strings.Contains(strings.ToLower(user.Email), needle) {
					result.Users = append(result.Users, user)
				}
			}
		}
	}
	writeJSON(w, result)
}

// ── Favourite Conversations ──

func toggleFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "conversation id is required")
		return
	}
	if !requireConversationAccess(w, r, conversationID) {
		return
	}

	favourite, err := store.ToggleFavourite(requestUserID(r), conversationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle favourite")
		return
	}
	writeJSON(w, map[string]interface{}{"conversationId": conversationID, "favourite": favourite})
}

func favouriteStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "conversation id is required")
		return
	}
	if !requireConversationAccess(w, r, conversationID) {
		return
	}

	favourite, err := store.IsFavourite(requestUserID(r), conversationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load favourite status")
		return
	}
	writeJSON(w, map[string]interface{}{"conversationId": conversationID, "favourite": favourite})
}

func favouriteIdsHandler(w http.ResponseWriter, r *http.Request) {
	ids, err := store.GetFavouriteIds(requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load favourites")
		return
	}
	writeJSON(w, ids)
}

// ── Mute Conversation ──

type muteRequest struct {
	Muted bool `json:"muted"`
}

func muteConversationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "conversation id is required")
		return
	}
	if !requireConversationAccess(w, r, conversationID) {
		return
	}

	var req muteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to mute when no body is given.
		req.Muted = true
	}

	if err := store.SetMuteConversation(requestUserID(r), conversationID, req.Muted); err != nil {
		log.Printf("failed to update mute state for %s: %v", requestUserID(r), err)
		writeError(w, http.StatusInternalServerError, "failed to update mute state")
		return
	}
	writeJSON(w, map[string]interface{}{"conversationId": conversationID, "muted": req.Muted})
}

func unmuteConversationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "conversation id is required")
		return
	}
	if !requireConversationAccess(w, r, conversationID) {
		return
	}

	if err := store.SetMuteConversation(requestUserID(r), conversationID, false); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to unmute conversation")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mutedIdsHandler(w http.ResponseWriter, r *http.Request) {
	ids, err := store.GetMutedConversationIds(requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load muted conversations")
		return
	}
	writeJSON(w, ids)
}

// ── Clear Chat ──

func clearConversationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "conversation id is required")
		return
	}
	if !requireConversationAccess(w, r, conversationID) {
		return
	}

	if err := store.ClearConversationMessages(conversationID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to clear conversation")
		return
	}
	store.BroadcastEnvelope("", conversationID, "conversation.cleared", map[string]string{
		"conversationId": conversationID,
	})
	w.WriteHeader(http.StatusNoContent)
}
