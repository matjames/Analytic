package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"knowledgeportal/pkg/model"
	"knowledgeportal/pkg/store"

	"github.com/gorilla/mux"
)

var reviewStatuses = map[string]bool{
	"submitted":         true,
	"approved":          true,
	"changes_requested": true,
	"rejected":          true,
	"withdrawn":         true,
}

func GetContentReviewHandler(w http.ResponseWriter, r *http.Request) {
	contentID := mux.Vars(r)["id"]
	if _, err := store.GetContentItemScoped(r.Context(), actorTenant(r), actorWorkspace(r), contentID); err != nil {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	review, err := store.GetContentReview(r.Context(), actorTenant(r), actorWorkspace(r), contentID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "content review not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, review)
}

func ListContentReviewsHandler(w http.ResponseWriter, r *http.Request) {
	if !canModerate(r) {
		writeError(w, http.StatusForbidden, "editorial moderation role required")
		return
	}
	reviews, err := store.ListContentReviews(r.Context(), actorTenant(r), actorWorkspace(r), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

func SubmitContentReviewHandler(w http.ResponseWriter, r *http.Request) {
	contentID := mux.Vars(r)["id"]
	content, err := store.GetContentItemScoped(r.Context(), actorTenant(r), actorWorkspace(r), contentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	var body struct {
		Note   string `json:"note"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	status := strings.TrimSpace(body.Status)
	if status == "" {
		status = "submitted"
	}
	if status != "submitted" && status != "withdrawn" {
		writeError(w, http.StatusBadRequest, "submission status must be submitted or withdrawn")
		return
	}
	now := time.Now().UTC()
	review := &model.ContentReview{
		TenantID: actorTenant(r), WorkspaceID: actorWorkspace(r), ContentID: contentID,
		Status: status, Note: strings.TrimSpace(body.Note), SubmittedBy: actorID(r), SubmittedAt: now,
		CreatedAt: now,
	}
	if existing, getErr := store.GetContentReview(r.Context(), review.TenantID, review.WorkspaceID, contentID); getErr == nil {
		review.ID, review.CreatedAt = existing.ID, existing.CreatedAt
	} else if getErr != sql.ErrNoRows {
		writeError(w, http.StatusInternalServerError, getErr.Error())
		return
	}
	if err := store.SaveContentReview(r.Context(), review); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if status == "submitted" && content.Status == "draft" {
		content.Status = "in_review"
		content.UpdatedBy = actorID(r)
		if err := store.UpdateContentItem(r.Context(), content); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	recordAudit(r, "content.review.submitted", "content", contentID, map[string]interface{}{"status": status})
	writeJSON(w, http.StatusOK, review)
}

func DecideContentReviewHandler(w http.ResponseWriter, r *http.Request) {
	if !canModerate(r) {
		writeError(w, http.StatusForbidden, "editorial moderation role required")
		return
	}
	contentID := mux.Vars(r)["id"]
	content, err := store.GetContentItemScoped(r.Context(), actorTenant(r), actorWorkspace(r), contentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	review, err := store.GetContentReview(r.Context(), actorTenant(r), actorWorkspace(r), contentID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "content review not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var body struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	status := strings.TrimSpace(body.Status)
	if status != "approved" && status != "changes_requested" && status != "rejected" {
		writeError(w, http.StatusBadRequest, "decision status must be approved, changes_requested, or rejected")
		return
	}
	review.Status, review.Note, review.ReviewerID = status, strings.TrimSpace(body.Note), actorID(r)
	now := time.Now().UTC()
	review.ReviewedAt = &now
	if err := store.SaveContentReview(r.Context(), review); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	content.UpdatedBy = actorID(r)
	switch status {
	case "approved":
		content.Status = "published"
		if content.PublishedAt == nil {
			content.PublishedAt = &now
		}
	case "rejected":
		if content.Status == "published" {
			content.Status = "archived"
		} else {
			content.Status = "draft"
		}
	case "changes_requested":
		if content.Status != "published" {
			content.Status = "draft"
		}
	}
	if err := store.UpdateContentItem(r.Context(), content); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "content.review.decided", "content", contentID, map[string]interface{}{"status": status})
	writeJSON(w, http.StatusOK, review)
}
