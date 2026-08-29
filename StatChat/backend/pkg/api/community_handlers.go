package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"statchat/pkg/model"
	"statchat/pkg/store"
)

func communitiesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		communities, err := store.GetCommunities(requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load communities")
			return
		}
		writeJSON(w, communities)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Visibility  string `json:"visibility"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Visibility = strings.ToLower(strings.TrimSpace(req.Visibility))
	if req.Name == "" || len(req.Name) > 120 {
		writeError(w, http.StatusBadRequest, "community name is required")
		return
	}
	if req.Visibility == "" {
		req.Visibility = "public"
	}
	if req.Visibility != "public" && req.Visibility != "private" {
		writeError(w, http.StatusBadRequest, "visibility must be public or private")
		return
	}

	created, err := store.CreateCommunity(model.Community{
		TenantID:    requestTenantID(r),
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
		CreatedBy:   requestUserID(r),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create community")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, created)
}

func joinCommunityHandler(w http.ResponseWriter, r *http.Request) {
	community, err := store.JoinCommunity(mux.Vars(r)["id"], requestTenantID(r), requestUserID(r))
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to join community")
		return
	}
	writeJSON(w, community)
}

func leaveCommunityHandler(w http.ResponseWriter, r *http.Request) {
	if err := store.LeaveCommunity(mux.Vars(r)["id"], requestTenantID(r), requestUserID(r)); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusConflict, "community owners cannot leave without transferring ownership")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to leave community")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func communityMembersHandler(w http.ResponseWriter, r *http.Request) {
	communityID := mux.Vars(r)["id"]
	if r.Method == http.MethodGet {
		members, err := store.GetCommunityMembers(communityID, requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeJSON(w, members)
		return
	}

	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.UserID) == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}
	member, err := store.AddCommunityMember(communityID, requestTenantID(r), requestUserID(r), strings.TrimSpace(req.UserID))
	if err != nil {
		if err == store.ErrCommunityForbidden {
			writeError(w, http.StatusForbidden, "only community owners can add tenant members")
			return
		}
		writeError(w, http.StatusNotFound, "target user not found in this tenant")
		return
	}
	writeJSON(w, member)
}

func removeCommunityMemberHandler(w http.ResponseWriter, r *http.Request) {
	if err := store.RemoveCommunityMember(mux.Vars(r)["id"], requestTenantID(r), requestUserID(r), mux.Vars(r)["userId"]); err != nil {
		if err == store.ErrCommunityForbidden {
			writeError(w, http.StatusForbidden, "only community owners can remove members")
			return
		}
		if err == sql.ErrNoRows {
			writeError(w, http.StatusConflict, "member cannot be removed")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to remove community member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func transferCommunityOwnerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.UserID) == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}
	if err := store.TransferCommunityOwnership(mux.Vars(r)["id"], requestTenantID(r), requestUserID(r), strings.TrimSpace(req.UserID)); err != nil {
		if err == store.ErrCommunityForbidden {
			writeError(w, http.StatusForbidden, "only community owners can transfer ownership")
			return
		}
		if err == sql.ErrNoRows {
			writeError(w, http.StatusConflict, "ownership can only transfer from owner to another member")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to transfer community ownership")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func communityTopicsHandler(w http.ResponseWriter, r *http.Request) {
	communityID := mux.Vars(r)["id"]
	if r.Method == http.MethodGet {
		topics, err := store.GetCommunityTopics(communityID, requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeJSON(w, topics)
		return
	}

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)
	if req.Title == "" || req.Body == "" || len(req.Title) > 180 {
		writeError(w, http.StatusBadRequest, "title and body are required")
		return
	}
	currentUser, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authenticated user is required")
		return
	}
	role := ""
	if len(currentUser.Roles) > 0 {
		role = currentUser.Roles[0]
	}
	topic, err := store.CreateCommunityTopic(model.CommunityTopic{
		TenantID:    requestTenantID(r),
		CommunityID: communityID,
		Title:       req.Title,
		Body:        req.Body,
		AuthorID:    currentUser.ID,
		Author:      currentUser.Name,
		Role:        role,
		Org:         currentUser.OrganizationID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "community not found")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, topic)
}

func deleteCommunityTopicHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := store.DeleteCommunityTopic(vars["communityId"], vars["topicId"], requestTenantID(r), requestUserID(r)); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusForbidden, "only the topic author or community owner can delete this topic")
			return
		}
		writeError(w, http.StatusNotFound, "topic not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func communityRepliesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	communityID, topicID := vars["communityId"], vars["topicId"]
	if r.Method == http.MethodGet {
		replies, err := store.GetCommunityReplies(communityID, topicID, requestTenantID(r), requestUserID(r))
		if err != nil {
			writeError(w, http.StatusNotFound, "topic not found")
			return
		}
		writeJSON(w, replies)
		return
	}

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "reply body is required")
		return
	}
	currentUser, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authenticated user is required")
		return
	}
	role := ""
	if len(currentUser.Roles) > 0 {
		role = currentUser.Roles[0]
	}
	reply, err := store.CreateCommunityReply(model.CommunityReply{
		TenantID:    requestTenantID(r),
		CommunityID: communityID,
		TopicID:     topicID,
		AuthorID:    currentUser.ID,
		Author:      currentUser.Name,
		Role:        role,
		Org:         currentUser.OrganizationID,
		Body:        req.Body,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "topic not found")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, reply)
}

func deleteCommunityReplyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := store.DeleteCommunityReply(vars["communityId"], vars["topicId"], vars["replyId"], requestTenantID(r), requestUserID(r)); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusForbidden, "only the reply author or community owner can delete this reply")
			return
		}
		writeError(w, http.StatusNotFound, "reply not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
