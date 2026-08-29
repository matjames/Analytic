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

func collaborationDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		documents, err := store.GetCollaborationDocuments(requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load documents")
			return
		}
		writeJSON(w, documents)
		return
	}
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" || len(req.Title) > 180 {
		writeError(w, http.StatusBadRequest, "document title is required")
		return
	}
	if len(req.Content) > 250000 {
		writeError(w, http.StatusBadRequest, "document content is too large")
		return
	}
	document, err := store.CreateCollaborationDocument(model.CollaborationDocument{
		TenantID:  requestTenantID(r),
		Title:     req.Title,
		Content:   req.Content,
		CreatedBy: requestUserID(r),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create document")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, document)
}

func collaborationDocumentHandler(w http.ResponseWriter, r *http.Request) {
	documentID := mux.Vars(r)["id"]
	if r.Method == http.MethodGet {
		document, err := store.GetCollaborationDocument(documentID, requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		writeJSON(w, document)
		return
	}
	if r.Method == http.MethodDelete {
		err := store.DeleteCollaborationDocument(documentID, requestTenantID(r), requestUserID(r))
		if errors.Is(err, store.ErrDocumentForbidden) {
			writeError(w, http.StatusForbidden, "only the document owner can delete it")
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete document")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var req struct {
		Title           string `json:"title"`
		Content         string `json:"content"`
		ExpectedVersion int    `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" || len(req.Title) > 180 || len(req.Content) > 250000 || req.ExpectedVersion < 1 {
		writeError(w, http.StatusBadRequest, "title, content size, and current version are required")
		return
	}
	document, err := store.UpdateCollaborationDocument(documentID, requestTenantID(r), requestUserID(r), req.Title, req.Content, req.ExpectedVersion)
	if errors.Is(err, store.ErrDocumentForbidden) {
		writeError(w, http.StatusForbidden, "viewer access cannot edit this document")
		return
	}
	if errors.Is(err, store.ErrDocumentConflict) {
		writeError(w, http.StatusConflict, "document changed; reload before saving")
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update document")
		return
	}
	writeJSON(w, document)
}

func collaborationDocumentMembersHandler(w http.ResponseWriter, r *http.Request) {
	documentID := mux.Vars(r)["id"]
	if r.Method == http.MethodGet {
		members, err := store.GetCollaborationDocumentMembers(documentID, requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusNotFound, "document not found")
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
	member, err := store.AddCollaborationDocumentMember(documentID, requestTenantID(r), requestUserID(r), strings.TrimSpace(req.UserID), req.Role)
	if errors.Is(err, store.ErrDocumentForbidden) {
		writeError(w, http.StatusForbidden, "only document owners can manage members")
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "document or target user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add document member")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, member)
}

func removeCollaborationDocumentMemberHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := store.RemoveCollaborationDocumentMember(vars["id"], requestTenantID(r), requestUserID(r), vars["userId"])
	if errors.Is(err, store.ErrDocumentForbidden) {
		writeError(w, http.StatusForbidden, "only document owners can manage members")
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusConflict, "owner cannot be removed or member was not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove document member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func collaborationDocumentRevisionsHandler(w http.ResponseWriter, r *http.Request) {
	revisions, err := store.GetCollaborationDocumentRevisions(mux.Vars(r)["id"], requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	writeJSON(w, revisions)
}
