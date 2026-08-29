package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"statchat/pkg/model"
	"statchat/pkg/store"
)

func pollsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		polls, err := store.GetPolls(requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load polls")
			return
		}
		writeJSON(w, polls)
		return
	}
	var poll model.Poll
	if err := json.NewDecoder(r.Body).Decode(&poll); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	poll.Question = strings.TrimSpace(poll.Question)
	if poll.Question == "" || len(poll.Options) < 2 || len(poll.Options) > 10 {
		writeError(w, http.StatusBadRequest, "question and 2 to 10 options are required")
		return
	}
	seen := map[string]bool{}
	for _, option := range poll.Options {
		label := strings.TrimSpace(option.Label)
		if label == "" || seen[strings.ToLower(label)] {
			writeError(w, http.StatusBadRequest, "poll options must be unique and non-empty")
			return
		}
		seen[strings.ToLower(label)] = true
	}
	poll.TenantID, poll.CreatedBy = requestTenantID(r), requestUserID(r)
	created, err := store.CreatePoll(poll)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create poll")
		return
	}
	store.BroadcastEnvelope(created.TenantID, created.ID, "poll.created", created)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, created)
}

func votePollHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OptionID string `json:"optionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.OptionID) == "" {
		writeError(w, http.StatusBadRequest, "optionId is required")
		return
	}
	if err := store.VotePoll(mux.Vars(r)["id"], req.OptionID, requestTenantID(r), requestUserID(r)); err != nil {
		writeError(w, http.StatusNotFound, "poll or option not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
