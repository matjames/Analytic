package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"statchat/pkg/store"
)

func connectionRequestsHandler(w http.ResponseWriter, r *http.Request) {
	requests, err := store.GetConnectionRequests(requestUserID(r), requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load connection requests")
		return
	}
	writeJSON(w, requests)
}

func respondConnectionRequestHandler(w http.ResponseWriter, r *http.Request) {
	action := mux.Vars(r)["action"]
	if action != "accept" && action != "decline" {
		writeError(w, http.StatusBadRequest, "action must be accept or decline")
		return
	}
	accept := action == "accept"
	request, err := store.RespondToConnectionRequest(mux.Vars(r)["id"], requestUserID(r), requestTenantID(r), accept)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "pending connection request not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to respond to connection request")
		return
	}
	writeJSON(w, request)
}
