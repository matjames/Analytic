package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/statchat"
	"statdata-backend/internal/store"
)

var errDataObjectNotFound = errors.New("data object not found")

type StatChatIntegration struct {
	client *statchat.Client
}

func NewStatChatIntegration(baseURL, internalKey string) *StatChatIntegration {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	internalKey = strings.TrimSpace(internalKey)
	if baseURL == "" || internalKey == "" {
		return &StatChatIntegration{}
	}
	return &StatChatIntegration{client: statchat.NewClient(baseURL, internalKey)}
}

func dataObjectReference(objectType, objectID string) string {
	return fmt.Sprintf("obj:statdata:%s:%s", objectType, objectID)
}

func (s *StatChatIntegration) ensureDiscussion(ctx context.Context, userID, tenantID, workspaceID, objectType, objectID, name string) (statchat.Conversation, error) {
	if s == nil || s.client == nil {
		return statchat.Conversation{}, errors.New("StatChat integration is not configured")
	}
	client := *s.client
	client.ServiceUserID = userID
	client.TenantID = tenantID
	return client.EnsureObjectConversation(ctx, "", workspaceID, statchat.ObjectConversationRequest{
		ObjectRef: dataObjectReference(objectType, objectID),
		Name:      name,
		Metadata: map[string]any{
			"source":     "statdata",
			"objectType": objectType,
			"objectId":   objectID,
		},
	})
}

type DiscussionHandler struct {
	store store.Store
	chat  *StatChatIntegration
}

func NewDiscussionHandler(st store.Store, chat *StatChatIntegration) *DiscussionHandler {
	return &DiscussionHandler{store: st, chat: chat}
}

type dataDiscussionRequest struct {
	ObjectType string `json:"object_type"`
	ObjectID   string `json:"object_id"`
}

func (h *DiscussionHandler) Create(c *gin.Context) {
	var req dataDiscussionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid discussion request"})
		return
	}
	req.ObjectType = strings.ToLower(strings.TrimSpace(req.ObjectType))
	req.ObjectID = strings.TrimSpace(req.ObjectID)
	if req.ObjectID == "" || len(req.ObjectID) > 128 || strings.ContainsAny(req.ObjectID, " /\\\t\r\n") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "object_type and object_id must be safe identifiers"})
		return
	}
	userID := strings.TrimSpace(c.GetString("user_id"))
	tenantID := strings.TrimSpace(c.GetString("tenant_id"))
	if userID == "" || tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated tenant identity is required"})
		return
	}
	if h.chat == nil || h.chat.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "StatChat discussions are not configured"})
		return
	}
	workspaceID := getWorkspaceID(c)
	name, err := h.objectName(c.Request.Context(), req.ObjectType, req.ObjectID, tenantID, workspaceID)
	if errors.Is(err, errDataObjectNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "data object not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify data object"})
		return
	}
	conversation, err := h.chat.ensureDiscussion(c.Request.Context(), userID, tenantID, workspaceID, req.ObjectType, req.ObjectID, name)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create StatChat discussion"})
		return
	}
	if strings.TrimSpace(conversation.ID) == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "StatChat returned no conversation id"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"conversation_id": conversation.ID,
		"name":            conversation.Name,
		"object_ref":      conversation.ObjectRef,
		"object_type":     req.ObjectType,
		"object_id":       req.ObjectID,
	})
}

func (h *DiscussionHandler) objectName(ctx context.Context, objectType, objectID, tenantID, workspaceID string) (string, error) {
	switch objectType {
	case "dataset":
		items, _, err := h.store.ListDatasets(ctx, tenantID, "", "", workspaceID, 1000, 0)
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.Name, nil
			}
		}
	case "pipeline":
		items, err := h.store.ListPipelines(ctx, tenantID, "", workspaceID)
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.Name, nil
			}
		}
	case "data_source":
		items, err := h.store.ListDataSources(ctx, tenantID, "", workspaceID)
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.Name, nil
			}
		}
	case "feature_view":
		items, err := h.store.ListFeatureViews(ctx, tenantID, "", workspaceID)
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.Name, nil
			}
		}
	case "notebook":
		items, err := h.store.ListNotebookSessions(ctx, tenantID, "")
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.Title, nil
			}
		}
	case "experiment":
		items, err := h.store.ListExperiments(ctx, tenantID, "")
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.Name, nil
			}
		}
	case "model":
		items, err := h.store.ListRegisteredModels(ctx, tenantID, "")
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.Name, nil
			}
		}
	default:
		return "", fmt.Errorf("unsupported data object type %q", objectType)
	}
	return "", errDataObjectNotFound
}
