package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"knowledgeportal/pkg/store"

	"github.com/matjames/statgate-lib/statchat"
	"github.com/matjames/statgate-lib/tenant"
)

var safeDiscussionObjectID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

type StatChatIntegration struct{ client *statchat.Client }

func NewStatChatIntegration(baseURL, internalKey string) *StatChatIntegration {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(internalKey) == "" {
		return &StatChatIntegration{}
	}
	return &StatChatIntegration{client: statchat.NewClient(baseURL, internalKey)}
}

func (s *StatChatIntegration) ensureDiscussion(ctx context.Context, authorization, userID, tenantID, workspaceID, objectType, objectID, name string) (statchat.Conversation, error) {
	if s == nil || s.client == nil {
		return statchat.Conversation{}, errors.New("StatChat integration is not configured")
	}
	client := *s.client
	client.ServiceUserID, client.TenantID = userID, tenantID
	return client.EnsureObjectConversation(ctx, authorization, workspaceID, statchat.ObjectConversationRequest{
		ObjectRef: fmt.Sprintf("obj:knowledge-portal:%s:%s", objectType, objectID), Name: name,
		Metadata: map[string]any{"source": "knowledge-portal", "object_type": objectType, "object_id": objectID},
	})
}

type DiscussionHandler struct{ chat *StatChatIntegration }

func NewDiscussionHandler(chat *StatChatIntegration) *DiscussionHandler {
	return &DiscussionHandler{chat: chat}
}

type discussionRequest struct {
	ObjectType string `json:"object_type"`
	ObjectID   string `json:"object_id"`
}
type discussionObject struct{ ObjectType, ObjectID, Name string }

func (h *DiscussionHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.chat == nil || h.chat.client == nil {
		writeError(w, http.StatusServiceUnavailable, "StatChat discussions are not configured")
		return
	}
	userID, tenantID := actorID(r), actorTenant(r)
	if userID == "" || tenantID == "" {
		writeError(w, http.StatusUnauthorized, "authenticated user and tenant are required")
		return
	}
	var req discussionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	req.ObjectType, req.ObjectID = strings.ToLower(strings.TrimSpace(req.ObjectType)), strings.TrimSpace(req.ObjectID)
	if !safeDiscussionObjectID.MatchString(req.ObjectID) {
		writeError(w, http.StatusBadRequest, "object_id is invalid")
		return
	}
	object, err := resolveDiscussionObject(r, req.ObjectType, req.ObjectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "object not found")
		return
	}
	conversation, err := h.chat.ensureDiscussion(r.Context(), r.Header.Get("Authorization"), userID, tenantID, tenant.WorkspaceIDFromRequest(r), object.ObjectType, object.ObjectID, object.Name)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to create StatChat discussion")
		return
	}
	if conversation.ID == "" {
		writeError(w, http.StatusBadGateway, "StatChat returned no conversation id")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"conversation_id": conversation.ID, "object_ref": conversation.ObjectRef, "object_type": object.ObjectType, "object_id": object.ObjectID})
}

func resolveDiscussionObject(r *http.Request, objectType, objectID string) (discussionObject, error) {
	ctx, tenantID := r.Context(), actorTenant(r)
	switch objectType {
	case "content", "content_item":
		items, err := store.ListContentItems(ctx, tenantID, "", "", "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"content_item", item.ID, item.Title}, nil
			}
		}
	case "dataset", "public_dataset":
		items, err := store.ListDatasets(ctx, tenantID, "", "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"public_dataset", item.ID, item.Title}, nil
			}
		}
	case "repository", "repository_item":
		items, err := store.ListRepositoryItems(ctx, tenantID, "", "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"repository_item", item.ID, item.Title}, nil
			}
		}
	}
	return discussionObject{}, errors.New("object not found")
}
