package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/statchat"
	"statiot-backend/internal/store"
)

var errFieldObjectNotFound = errors.New("field object not found")

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

func fieldObjectReference(objectType, objectID string) string {
	return fmt.Sprintf("obj:statiot:%s:%s", objectType, objectID)
}

func (s *StatChatIntegration) ensureDiscussion(ctx context.Context, userID, tenantID, workspaceID, objectType, objectID, name string) (statchat.Conversation, error) {
	if s == nil || s.client == nil {
		return statchat.Conversation{}, errors.New("StatChat integration is not configured")
	}
	client := *s.client
	client.ServiceUserID = userID
	client.TenantID = tenantID
	return client.EnsureObjectConversation(ctx, "", workspaceID, statchat.ObjectConversationRequest{
		ObjectRef: fieldObjectReference(objectType, objectID),
		Name:      name,
		Metadata: map[string]any{
			"source":     "statiot",
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

type fieldDiscussionRequest struct {
	ObjectType string `json:"object_type"`
	ObjectID   string `json:"object_id"`
}

func (h *DiscussionHandler) Create(c *gin.Context) {
	var req fieldDiscussionRequest
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

	tenantID := strings.TrimSpace(c.GetString("tenant_id"))
	userID := strings.TrimSpace(c.GetString("user_id"))
	if tenantID == "" || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated tenant identity is required"})
		return
	}
	if h.chat == nil || h.chat.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "StatChat discussions are not configured"})
		return
	}
	workspaceID := getWorkspaceID(c)
	name, err := h.objectName(c, req.ObjectType, req.ObjectID, tenantID, workspaceID)
	if errors.Is(err, errFieldObjectNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "field object not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify field object"})
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

func (h *DiscussionHandler) objectName(c *gin.Context, objectType, objectID, tenantID, workspaceID string) (string, error) {
	ctx := c.Request.Context()
	switch objectType {
	case "gateway":
		items, err := h.store.ListGateways(ctx, tenantID, workspaceID)
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.Name, nil
			}
		}
	case "device":
		items, err := h.store.ListDevices(ctx, tenantID, workspaceID)
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.Name, nil
			}
		}
	case "alert":
		items, err := h.store.ListAlerts(ctx, tenantID, "", workspaceID)
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return fmt.Sprintf("%s alert: %s", item.Severity, item.Message), nil
			}
		}
	case "worker":
		items, err := h.store.ListFieldWorkers(ctx, tenantID, workspaceID)
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.FullName, nil
			}
		}
	case "form":
		items, err := h.store.ListFormDefinitions(ctx, tenantID)
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID == objectID {
				return item.Title, nil
			}
		}
	default:
		return "", fmt.Errorf("unsupported field object type %q", objectType)
	}
	return "", errFieldObjectNotFound
}
