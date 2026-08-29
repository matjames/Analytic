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

func wellnessPostCommentsHandler(w http.ResponseWriter, r *http.Request) {
	comments, err := store.GetWellnessComments(mux.Vars(r)["id"], requestTenantID(r))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "wellness post not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load wellness comments")
		return
	}
	writeJSON(w, comments)
}

func addWellnessCommentHandler(w http.ResponseWriter, r *http.Request) {
	var req model.WellnessComment
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		writeError(w, http.StatusBadRequest, "comment text is required")
		return
	}
	currentUser, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authenticated user is required")
		return
	}
	req.PostID = mux.Vars(r)["id"]
	req.AuthorID = currentUser.ID
	req.Author = currentUser.Name
	req.Org = currentUser.OrganizationID
	if len(currentUser.Roles) > 0 {
		req.Role = currentUser.Roles[0]
	}
	comment, err := store.AddWellnessComment(req, requestTenantID(r))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "wellness post not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add wellness comment")
		return
	}
	writeJSON(w, comment)
}

func toggleWellnessLikeHandler(w http.ResponseWriter, r *http.Request) {
	liked, likes, err := store.ToggleWellnessLike(mux.Vars(r)["id"], requestUserID(r), requestTenantID(r))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "wellness post not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update wellness like")
		return
	}
	writeJSON(w, map[string]any{"liked": liked, "likes": likes})
}

func toggleWellnessBookmarkHandler(w http.ResponseWriter, r *http.Request) {
	bookmarked, bookmarks, err := store.ToggleWellnessBookmark(mux.Vars(r)["id"], requestUserID(r), requestTenantID(r))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "wellness post not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update wellness bookmark")
		return
	}
	writeJSON(w, map[string]any{"bookmarked": bookmarked, "bookmarks": bookmarks})
}

func shareWellnessPostHandler(w http.ResponseWriter, r *http.Request) {
	shares, err := store.ShareWellnessPost(mux.Vars(r)["id"], requestUserID(r), requestTenantID(r))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "wellness post not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to share wellness post")
		return
	}
	writeJSON(w, map[string]any{"shares": shares})
}
