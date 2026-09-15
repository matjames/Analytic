package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"learningcrm/pkg/store"

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
	client.ServiceUserID, client.TenantID = userID, tenantID
	return client.EnsureObjectConversation(ctx, authorization, workspaceID, statchat.ObjectConversationRequest{
		ObjectRef: fmt.Sprintf("obj:learning-crm:%s:%s", objectType, objectID), Name: name,
		Metadata: map[string]any{"source": "learning-crm", "object_type": objectType, "object_id": objectID},
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
	case "course":
		items, err := store.ListCourses(ctx, tenantID, "", "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"course", item.ID, item.Title}, nil
			}
		}
	case "lead":
		items, err := store.ListLeads(ctx, tenantID, "", workspaceID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"lead", item.ID, item.Name}, nil
			}
		}
	case "account":
		items, err := store.ListAccounts(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"account", item.ID, item.Name}, nil
			}
		}
	case "opportunity":
		items, err := store.ListOpportunities(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"opportunity", item.ID, item.Name}, nil
			}
		}
	case "partner":
		items, err := store.ListPartners(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"partner", item.ID, item.Name}, nil
			}
		}
	case "service_request":
		items, err := store.ListServiceRequests(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"service_request", item.ID, item.Title}, nil
			}
		}
	case "invoice":
		items, err := store.ListInvoices(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"invoice", item.ID, item.Number}, nil
			}
		}
	case "stakeholder":
		items, err := store.ListStakeholders(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"stakeholder", item.ID, item.Name}, nil
			}
		}
	case "engagement":
		items, err := store.ListEngagements(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"engagement", item.ID, item.Title}, nil
			}
		}
	case "stewardship_action":
		items, err := store.ListStewardshipActions(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"stewardship_action", item.ID, item.Action}, nil
			}
		}
	}
	return discussionObject{}, errors.New("object not found")
}
