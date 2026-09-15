package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/statchat"
)

var safeDiscussionObjectID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

type statTrustStatChat struct{ client *statchat.Client }

func newStatTrustStatChat(baseURL, internalKey string) *statTrustStatChat {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(internalKey) == "" {
		return &statTrustStatChat{}
	}
	return &statTrustStatChat{client: statchat.NewClient(baseURL, internalKey)}
}

func (s *statTrustStatChat) ensureDiscussion(ctx context.Context, authorization, userID, tenantID, workspaceID, objectType, objectID, name string) (statchat.Conversation, error) {
	if s == nil || s.client == nil {
		return statchat.Conversation{}, errors.New("StatChat integration is not configured")
	}
	client := *s.client
	client.ServiceUserID, client.TenantID = userID, tenantID
	return client.EnsureObjectConversation(ctx, authorization, workspaceID, statchat.ObjectConversationRequest{
		ObjectRef: fmt.Sprintf("obj:stattrust:%s:%s", objectType, objectID), Name: name,
		Metadata: map[string]any{"source": "stattrust", "object_type": objectType, "object_id": objectID},
	})
}

type statTrustDiscussionHandler struct{ chat *statTrustStatChat }

func newStatTrustDiscussionHandler(chat *statTrustStatChat) *statTrustDiscussionHandler {
	return &statTrustDiscussionHandler{chat: chat}
}

func (h *statTrustDiscussionHandler) create(c *gin.Context) {
	if h == nil || h.chat == nil || h.chat.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "StatChat discussions are not configured"})
		return
	}
	tenantID, workspaceID := requestScope(c)
	userID, _ := c.Get("user_id")
	uid := fmt.Sprint(userID)
	if uid == "<nil>" {
		uid = ""
	}
	if uid == "" || tenantID == "" {
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
	name, ok := h.objectName(req.ObjectType, req.ObjectID, tenantID, workspaceID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "object not found"})
		return
	}
	conversation, err := h.chat.ensureDiscussion(c.Request.Context(), c.GetHeader("Authorization"), uid, tenantID, workspaceID, req.ObjectType, req.ObjectID, name)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create StatChat discussion"})
		return
	}
	if conversation.ID == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "StatChat returned no conversation id"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversation_id": conversation.ID, "object_ref": conversation.ObjectRef, "object_type": req.ObjectType, "object_id": req.ObjectID})
}

func (h *statTrustDiscussionHandler) objectName(objectType, objectID, tenantID, workspaceID string) (string, bool) {
	switch objectType {
	case "incident", "security_incident":
		for _, item := range globalStore.ListIncidentsScoped(tenantID, workspaceID) {
			if item.ID == objectID {
				return item.Title, true
			}
		}
	case "ledger", "ledger_block":
		for _, item := range globalStore.ListLedgerScoped(tenantID, workspaceID) {
			if fmt.Sprint(item.Index) == objectID || fmt.Sprint(item.StorageID) == objectID {
				return item.EventType, true
			}
		}
	case "provenance", "artifact_provenance":
		for _, item := range globalStore.ListProvenanceScoped(tenantID, workspaceID) {
			if item.ID == objectID || item.ArtifactID == objectID {
				return item.ArtifactName, true
			}
		}
	case "certificate", "pki_certificate":
		for _, item := range globalStore.ListCertificates(tenantID) {
			if item.ID == objectID {
				return item.CommonName, true
			}
		}
	}
	return "", false
}
