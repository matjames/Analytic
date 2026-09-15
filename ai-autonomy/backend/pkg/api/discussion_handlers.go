package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"aiengines/pkg/store"

	"github.com/matjames/statgate-lib/statchat"
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
	client.ServiceUserID = userID
	client.TenantID = tenantID
	return client.EnsureObjectConversation(ctx, authorization, workspaceID, statchat.ObjectConversationRequest{
		ObjectRef: fmt.Sprintf("obj:ai-autonomy:%s:%s", objectType, objectID),
		Name:      name,
		Metadata:  map[string]any{"source": "ai-autonomy", "object_type": objectType, "object_id": objectID},
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

type discussionObject struct {
	ObjectType string
	ObjectID   string
	Name       string
}

func (h *DiscussionHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.chat == nil || h.chat.client == nil {
		writeError(w, http.StatusServiceUnavailable, "StatChat discussions are not configured")
		return
	}
	userID := actorID(r)
	tenantID := actorTenant(r)
	if userID == "" || tenantID == "" {
		writeError(w, http.StatusUnauthorized, "authenticated user and tenant are required")
		return
	}
	var req discussionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	req.ObjectType = strings.ToLower(strings.TrimSpace(req.ObjectType))
	req.ObjectID = strings.TrimSpace(req.ObjectID)
	if !safeDiscussionObjectID.MatchString(req.ObjectID) {
		writeError(w, http.StatusBadRequest, "object_id is invalid")
		return
	}
	object, err := resolveDiscussionObject(r, req.ObjectType, req.ObjectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "object not found")
		return
	}
	conversation, err := h.chat.ensureDiscussion(r.Context(), r.Header.Get("Authorization"), userID, tenantID, actorWorkspace(r), object.ObjectType, object.ObjectID, object.Name)
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
	ctx, tenantID, workspaceID := r.Context(), actorTenant(r), actorWorkspace(r)
	switch objectType {
	case "twin", "digital_twin":
		items, err := store.ListTwins(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"digital_twin", item.ID, item.Name}, nil
			}
		}
	case "model", "ai_model":
		items, err := store.ListModels(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"ai_model", item.ID, item.Name}, nil
			}
		}
	case "pipeline", "decision_pipeline":
		items, err := store.ListPipelines(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"decision_pipeline", item.ID, item.Name}, nil
			}
		}
	case "agent":
		items, err := store.ListAgents(ctx, tenantID, "", workspaceID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"agent", item.ID, item.Name}, nil
			}
		}
	case "graph_node":
		items, err := store.ListGraphNodes(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"graph_node", item.ID, item.Label}, nil
			}
		}
	}
	return discussionObject{}, errors.New("object not found")
}
