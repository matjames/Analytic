package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"statchat/pkg/store"

	"github.com/gorilla/mux"
)

// ── Global Search ──

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		result, err := store.GlobalSearch(query, requestUserID(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to run search")
			return
		}
		writeJSON(w, result)
		return
	}

	result, err := store.GlobalSearch(query, requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to run search")
		return
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

	var req muteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to mute when no body is given.
		req.Muted = true
	}

	if err := store.SetMuteConversation(requestUserID(r), conversationID, req.Muted); err != nil {
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

	if err := store.ClearConversationMessages(conversationID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to clear conversation")
		return
	}
	store.BroadcastEnvelope("", conversationID, "conversation.cleared", map[string]string{
		"conversationId": conversationID,
	})
	w.WriteHeader(http.StatusNoContent)
}
