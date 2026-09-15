package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/statchat"
	"statfederation-backend/internal/models"
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
		ObjectRef: fmt.Sprintf("obj:statfederation:%s:%s", objectType, objectID), Name: name,
		Metadata: map[string]any{"source": "statfederation", "object_type": objectType, "object_id": objectID},
	})
}

type DiscussionHandler struct {
	chat  *StatChatIntegration
	store interface {
		ListNodes(context.Context, string, string, string) ([]*models.FederatedNode, error)
		ListDSAs(context.Context, string, string) ([]*models.DataSharingAgreement, error)
		ListIndicators(context.Context, string, string) ([]*models.NationalIndicator, error)
		ListQueries(context.Context, string, int) ([]*models.DistributedQueryRecord, error)
		ListVocabularies(context.Context, string, string) ([]*models.MetadataVocabulary, error)
		ListTreaties(context.Context, string, string) ([]*models.DiplomaticTreaty, error)
		ListReports(context.Context, string, string) ([]*models.InternationalReport, error)
		ListComplianceLogs(context.Context, string, int) ([]*models.ComplianceAuditLog, error)
	}
}

func NewDiscussionHandler(s interface {
	ListNodes(context.Context, string, string, string) ([]*models.FederatedNode, error)
	ListDSAs(context.Context, string, string) ([]*models.DataSharingAgreement, error)
	ListIndicators(context.Context, string, string) ([]*models.NationalIndicator, error)
	ListQueries(context.Context, string, int) ([]*models.DistributedQueryRecord, error)
	ListVocabularies(context.Context, string, string) ([]*models.MetadataVocabulary, error)
	ListTreaties(context.Context, string, string) ([]*models.DiplomaticTreaty, error)
	ListReports(context.Context, string, string) ([]*models.InternationalReport, error)
	ListComplianceLogs(context.Context, string, int) ([]*models.ComplianceAuditLog, error)
}, chat *StatChatIntegration) *DiscussionHandler {
	return &DiscussionHandler{store: s, chat: chat}
}

type discussionObject struct{ ObjectType, ObjectID, Name, WorkspaceID string }

func (h *DiscussionHandler) Create(c *gin.Context) {
	if h == nil || h.chat == nil || h.chat.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "StatChat discussions are not configured"})
		return
	}
	userID, _ := c.Get("user_id")
	tenantID, _ := c.Get("tenant_id")
	uid, tid := fmt.Sprint(userID), fmt.Sprint(tenantID)
	if uid == "<nil>" {
		uid = ""
	}
	if tid == "<nil>" {
		tid = ""
	}
	if uid == "" || tid == "" || tid == "<nil>" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user and tenant are required"})
		return
	}
	var req struct {
		ObjectType string `json:"object_type"`
		ObjectID   string `json:"object_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body: " + err.Error()})
		return
	}
	req.ObjectType, req.ObjectID = strings.ToLower(strings.TrimSpace(req.ObjectType)), strings.TrimSpace(req.ObjectID)
	if !safeDiscussionObjectID.MatchString(req.ObjectID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "object_id is invalid"})
		return
	}
	object, err := h.resolve(c.Request.Context(), tid, c.GetString("workspace_id"), req.ObjectType, req.ObjectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "object not found"})
		return
	}
	conversation, err := h.chat.ensureDiscussion(c.Request.Context(), c.GetHeader("Authorization"), uid, tid, c.GetString("workspace_id"), object.ObjectType, object.ObjectID, object.Name)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create StatChat discussion"})
		return
	}
	if conversation.ID == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "StatChat returned no conversation id"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversation_id": conversation.ID, "object_ref": conversation.ObjectRef, "object_type": object.ObjectType, "object_id": object.ObjectID})
}

func (h *DiscussionHandler) resolve(ctx context.Context, tenantID, workspaceID, objectType, objectID string) (discussionObject, error) {
	workspaceMatch := func(value string) bool { return workspaceID == "" || value == "" || value == workspaceID }
	switch objectType {
	case "node", "federated_node":
		items, err := h.store.ListNodes(ctx, tenantID, "", "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"federated_node", item.ID, item.Name, item.WorkspaceID}, nil
			}
		}
	case "dsa", "data_sharing_agreement":
		items, err := h.store.ListDSAs(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"data_sharing_agreement", item.ID, item.Title, item.WorkspaceID}, nil
			}
		}
	case "indicator", "national_indicator":
		items, err := h.store.ListIndicators(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"national_indicator", item.ID, item.Title, item.WorkspaceID}, nil
			}
		}
	case "query", "distributed_query":
		items, err := h.store.ListQueries(ctx, tenantID, 500)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"distributed_query", item.ID, item.QueryName, item.WorkspaceID}, nil
			}
		}
	case "vocabulary":
		items, err := h.store.ListVocabularies(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"vocabulary", item.ID, item.VocabularyName, item.WorkspaceID}, nil
			}
		}
	case "treaty":
		items, err := h.store.ListTreaties(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"treaty", item.ID, item.Title, item.WorkspaceID}, nil
			}
		}
	case "report", "international_report":
		items, err := h.store.ListReports(ctx, tenantID, "")
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"international_report", item.ID, item.ReportTitle, item.WorkspaceID}, nil
			}
		}
	case "compliance_log":
		items, err := h.store.ListComplianceLogs(ctx, tenantID, 500)
		if err != nil {
			return discussionObject{}, err
		}
		for _, item := range items {
			if item.ID == objectID && workspaceMatch(item.WorkspaceID) {
				return discussionObject{"compliance_log", item.ID, item.Action, item.WorkspaceID}, nil
			}
		}
	}
	return discussionObject{}, errors.New("object not found")
}
