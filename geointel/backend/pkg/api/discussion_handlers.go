package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"geointel/pkg/store"

	"github.com/matjames/statgate-lib/statchat"
)

var safeObjectID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

// StatChatIntegration owns the service-to-service client. The request
// identity is applied to a short-lived client copy for every call.
type StatChatIntegration struct {
	client *statchat.Client
}

func NewStatChatIntegration(baseURL, internalKey string) *StatChatIntegration {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(internalKey) == "" {
		return &StatChatIntegration{}
	}
	return &StatChatIntegration{client: statchat.NewClient(baseURL, internalKey)}
}

func (s *StatChatIntegration) ensureDiscussion(ctx context.Context, authorization, userID, tenantID, workspaceID string, objectType, objectID, name string) (statchat.Conversation, error) {
	if s == nil || s.client == nil {
		return statchat.Conversation{}, errors.New("StatChat integration is not configured")
	}
	client := *s.client
	client.ServiceUserID = userID
	client.TenantID = tenantID
	return client.EnsureObjectConversation(ctx, authorization, workspaceID, statchat.ObjectConversationRequest{
		ObjectRef: fmt.Sprintf("obj:geointel:%s:%s", objectType, objectID),
		Name:      name,
		Metadata: map[string]any{
			"source":      "geointel",
			"object_type": objectType,
			"object_id":   objectID,
		},
	})
}

type DiscussionHandler struct {
	chat *StatChatIntegration
}

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

// Create ensures a tenant-scoped StatChat conversation for a GeoIntel object.
// Object existence is checked here instead of trusting a caller-supplied name.
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
	if !safeObjectID.MatchString(req.ObjectID) {
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
	writeJSON(w, http.StatusOK, map[string]string{
		"conversation_id": conversation.ID,
		"object_ref":      conversation.ObjectRef,
		"object_type":     object.ObjectType,
		"object_id":       object.ObjectID,
	})
}

func resolveDiscussionObject(r *http.Request, objectType, objectID string) (discussionObject, error) {
	ctx := r.Context()
	tenantID := actorTenant(r)
	workspaceID := actorWorkspace(r)
	switch objectType {
	case "layer", "geo_layer":
		items, err := store.ListLayers(ctx, tenantID, "", workspaceID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"geo_layer", item.ID, item.Name}, nil
			}
		}
	case "raster":
		items, err := store.ListRasters(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"raster", item.ID, item.Name}, nil
			}
		}
	case "scene":
		items, err := store.ListScenes(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"scene", item.ID, item.Name}, nil
			}
		}
	case "drone":
		items, err := store.ListDrones(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"drone", item.ID, item.Name}, nil
			}
		}
	case "flight_plan":
		items, err := store.ListFlightPlans(ctx, tenantID)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"flight_plan", item.ID, item.Name}, nil
			}
		}
	case "flight":
		items, err := store.ListFlights(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID {
				return discussionObject{"flight", item.ID, item.ID}, nil
			}
		}
	}
	return discussionObject{}, errors.New("object not found")
}
