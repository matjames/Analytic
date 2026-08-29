package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"statchat/pkg/model"
	"statchat/pkg/store"
)

func collaborationWhiteboardsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		boards, err := store.GetCollaborationWhiteboards(requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load whiteboards")
			return
		}
		writeJSON(w, boards)
		return
	}
	var req struct {
		Title string `json:"title"`
		Data  string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.Title, req.Data = strings.TrimSpace(req.Title), strings.TrimSpace(req.Data)
	if req.Title == "" || len(req.Title) > 180 {
		writeError(w, http.StatusBadRequest, "whiteboard title is required")
		return
	}
	if req.Data == "" {
		req.Data = "[]"
	}
	if len(req.Data) > 250000 || !json.Valid([]byte(req.Data)) {
		writeError(w, http.StatusBadRequest, "whiteboard data is invalid or too large")
		return
	}
	board, err := store.CreateCollaborationWhiteboard(model.CollaborationWhiteboard{TenantID: requestTenantID(r), Title: req.Title, Data: req.Data, CreatedBy: requestUserID(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create whiteboard")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, board)
}

func collaborationWhiteboardHandler(w http.ResponseWriter, r *http.Request) {
	boardID := mux.Vars(r)["id"]
	if r.Method == http.MethodGet {
		board, err := store.GetCollaborationWhiteboard(boardID, requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusNotFound, "whiteboard not found")
			return
		}
		writeJSON(w, board)
		return
	}
	if r.Method == http.MethodDelete {
		err := store.DeleteCollaborationWhiteboard(boardID, requestTenantID(r), requestUserID(r))
		if errors.Is(err, store.ErrWhiteboardForbidden) {
			writeError(w, http.StatusForbidden, "only the whiteboard owner can delete it")
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "whiteboard not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete whiteboard")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var req struct {
		Title           string `json:"title"`
		Data            string `json:"data"`
		ExpectedVersion int    `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.Title, req.Data = strings.TrimSpace(req.Title), strings.TrimSpace(req.Data)
	if req.Title == "" || len(req.Title) > 180 || req.ExpectedVersion < 1 {
		writeError(w, http.StatusBadRequest, "title and current version are required")
		return
	}
	if len(req.Data) > 250000 || !json.Valid([]byte(req.Data)) {
		writeError(w, http.StatusBadRequest, "whiteboard data is invalid or too large")
		return
	}
	board, err := store.UpdateCollaborationWhiteboard(boardID, requestTenantID(r), requestUserID(r), req.Title, req.Data, req.ExpectedVersion)
	if errors.Is(err, store.ErrWhiteboardForbidden) {
		writeError(w, http.StatusForbidden, "viewer access cannot edit this whiteboard")
		return
	}
	if errors.Is(err, store.ErrWhiteboardConflict) {
		writeError(w, http.StatusConflict, "whiteboard changed; reload before saving")
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "whiteboard not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update whiteboard")
		return
	}
	writeJSON(w, board)
}

func collaborationWhiteboardMembersHandler(w http.ResponseWriter, r *http.Request) {
	boardID := mux.Vars(r)["id"]
	if r.Method == http.MethodGet {
		members, err := store.GetCollaborationWhiteboardMembers(boardID, requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusNotFound, "whiteboard not found")
			return
		}
		writeJSON(w, members)
		return
	}
	var req struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.UserID) == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}
	member, err := store.AddCollaborationWhiteboardMember(boardID, requestTenantID(r), requestUserID(r), strings.TrimSpace(req.UserID), req.Role)
	if errors.Is(err, store.ErrWhiteboardForbidden) {
		writeError(w, http.StatusForbidden, "only whiteboard owners can manage members")
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "whiteboard or target user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add whiteboard member")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, member)
}

func removeCollaborationWhiteboardMemberHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := store.RemoveCollaborationWhiteboardMember(vars["id"], requestTenantID(r), requestUserID(r), vars["userId"])
	if errors.Is(err, store.ErrWhiteboardForbidden) {
		writeError(w, http.StatusForbidden, "only whiteboard owners can manage members")
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusConflict, "owner cannot be removed or member was not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove whiteboard member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func collaborationWhiteboardRevisionsHandler(w http.ResponseWriter, r *http.Request) {
	revisions, err := store.GetCollaborationWhiteboardRevisions(mux.Vars(r)["id"], requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "whiteboard not found")
		return
	}
	writeJSON(w, revisions)
}
