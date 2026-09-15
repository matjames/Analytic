package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"statspatial/pkg/store"

	"github.com/matjames/statgate-lib/auth"
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
		ObjectRef: fmt.Sprintf("obj:statspatial:%s:%s", objectType, objectID), Name: name,
		Metadata: map[string]any{"source": "statspatial", "object_type": objectType, "object_id": objectID},
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
	userID := ""
	if u, ok := auth.GetUserFromContext(r.Context()); ok && u != nil {
		userID = u.UserID
	}
	tenantID := ""
	if value, ok := r.Context().Value(auth.ContextKeyTenant).(string); ok {
		tenantID = value
	}
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
	object, err := resolveDiscussionObject(r.Context(), tenantID, tenant.WorkspaceIDFromRequest(r), req.ObjectType, req.ObjectID)
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

func resolveDiscussionObject(ctx context.Context, tenantID, workspaceID, objectType, objectID string) (discussionObject, error) {
	workspaceMatch := func(value string) bool { return workspaceID == "" || value == "" || value == workspaceID }
	switch objectType {
	case "layer", "geo_layer":
		items, err := store.ListGeoLayers(ctx)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && item.TenantID == tenantID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"geo_layer", item.ID, item.Name}, nil
			}
		}
	case "feature", "geo_feature":
		layers, err := store.ListGeoLayers(ctx)
		if err != nil {
			return discussionObject{}, err
		}
		for _, layer := range layers {
			if layer.TenantID != tenantID || !workspaceMatch(layer.WorkspaceID) {
				continue
			}
			items, err := store.ListGeoFeatures(ctx, layer.ID, "")
			if err != nil {
				return discussionObject{}, err
			}
			for _, item := range items {
				if item.ID == objectID && item.TenantID == tenantID && workspaceMatch(item.WorkspaceID) {
					return discussionObject{"geo_feature", item.ID, item.Name}, nil
				}
			}
		}
	case "node", "federated_node":
		items, err := store.ListFederatedNodes(ctx, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && item.TenantID == tenantID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"federated_node", item.ID, item.Name}, nil
			}
		}
	case "agreement", "data_sharing_agreement":
		items, err := store.ListAgreements(ctx)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && item.TenantID == tenantID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"data_sharing_agreement", item.ID, item.Title}, nil
			}
		}
	case "dataset", "federated_dataset":
		nodes, err := store.ListFederatedNodes(ctx, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, node := range nodes {
			if node.TenantID != tenantID || !workspaceMatch(node.WorkspaceID) {
				continue
			}
			items, err := store.ListFederatedDatasets(ctx, node.ID)
			if err != nil {
				return discussionObject{}, err
			}
			for _, item := range items {
				if item.ID == objectID && item.TenantID == tenantID && workspaceMatch(item.WorkspaceID) {
					return discussionObject{"federated_dataset", item.ID, item.Name}, nil
				}
			}
		}
	}
	return discussionObject{}, errors.New("object not found")
}
