package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/statchat"
)

var errGovernanceObjectNotFound = errors.New("governance object not found")

type governanceStatChat struct {
	baseURL string
	client  *statchat.Client
}

var governanceChat *governanceStatChat

type governanceDiscussionRequest struct {
	ObjectType string `json:"object_type"`
	ObjectID   string `json:"object_id"`
	Name       string `json:"name"`
}

func initStatChat() {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("STATCHAT_API_URL")), "/")
	internalKey := strings.TrimSpace(os.Getenv("STATGATE_INTERNAL_API_KEY"))
	if baseURL == "" || internalKey == "" {
		log.Printf("StatChat governance discussions disabled: STATCHAT_API_URL and STATGATE_INTERNAL_API_KEY are required")
		return
	}
	governanceChat = &governanceStatChat{
		baseURL: baseURL,
		client:  statchat.NewClient(baseURL, internalKey),
	}
	log.Printf("StatChat governance discussions enabled (base URL: %s)", baseURL)
}

func governanceObjectReference(objectType, objectID string) string {
	return fmt.Sprintf("obj:statgovernance:%s:%s", objectType, objectID)
}

func (s *governanceStatChat) ensureDiscussion(ctx context.Context, userID, tenantID, workspaceID, objectType, objectID, name string) (statchat.Conversation, error) {
	if s == nil || s.client == nil {
		return statchat.Conversation{}, errors.New("StatChat governance discussions are disabled")
	}
	// Keep identity request-scoped. Governance serves many tenants concurrently,
	// so mutating the shared client would risk forwarding another user's headers.
	client := *s.client
	client.ServiceUserID = userID
	client.TenantID = tenantID
	return client.EnsureObjectConversation(ctx, "", workspaceID, statchat.ObjectConversationRequest{
		ObjectRef: governanceObjectReference(objectType, objectID),
		Name:      name,
		Metadata: map[string]any{
			"source":     "statgovernance",
			"objectType": objectType,
			"objectId":   objectID,
		},
	})
}

func normalizeGovernanceObjectType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func governanceObjectTitle(objectType, objectID, tenantID string) (string, error) {
	queries := map[string]string{
		"policy":   `SELECT title FROM statgovernance.policies WHERE id = $1 AND tenant_id = $2`,
		"risk":     `SELECT title FROM statgovernance.risks WHERE id = $1 AND tenant_id = $2`,
		"control":  `SELECT name FROM statgovernance.controls WHERE id = $1 AND tenant_id = $2`,
		"finding":  `SELECT title FROM statgovernance.audit_findings WHERE id = $1 AND tenant_id = $2`,
		"evidence": `SELECT title FROM statgovernance.evidence_records WHERE id = $1 AND tenant_id = $2`,
	}
	query, supported := queries[objectType]
	if !supported {
		return "", fmt.Errorf("unsupported governance object type %q", objectType)
	}

	if DB != nil {
		var title string
		err := DB.QueryRow(query, objectID, tenantID).Scan(&title)
		if errors.Is(err, sql.ErrNoRows) {
			return "", errGovernanceObjectNotFound
		}
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(title), nil
	}

	switch objectType {
	case "policy":
		if value, found := memStore.GetPolicyByID(objectID, tenantID); found {
			return value.Title, nil
		}
	case "risk":
		for _, value := range memStore.GetRisks(tenantID, "", "", "") {
			if value.ID == objectID {
				return value.Title, nil
			}
		}
	case "control":
		for _, value := range memStore.GetControls(tenantID, "", "") {
			if value.ID == objectID {
				return value.Name, nil
			}
		}
	case "finding":
		for _, value := range memStore.GetFindings(tenantID, "", "") {
			if value.ID == objectID {
				return value.Title, nil
			}
		}
	case "evidence":
		for _, value := range memStore.GetEvidence(tenantID, "", "") {
			if value.ID == objectID {
				return value.Title, nil
			}
		}
	}
	return "", errGovernanceObjectNotFound
}

func dbCreateDiscussion(c *gin.Context) {
	var req governanceDiscussionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid discussion request"})
		return
	}
	req.ObjectType = normalizeGovernanceObjectType(req.ObjectType)
	req.ObjectID = strings.TrimSpace(req.ObjectID)
	if req.ObjectType == "" || req.ObjectID == "" || len(req.ObjectType) > 64 || len(req.ObjectID) > 128 || strings.ContainsAny(req.ObjectID, " /\\\t\r\n") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "object_type and object_id are required and must be safe identifiers"})
		return
	}

	userID, _, _, tenantID := getAuthContext(c)
	title, err := governanceObjectTitle(req.ObjectType, req.ObjectID, tenantID)
	if errors.Is(err, errGovernanceObjectNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "governance object not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify governance object"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = title
	}
	if len(name) > 255 {
		name = name[:255]
	}
	workspaceValue, _ := c.Get("workspace_id")
	workspaceID, _ := workspaceValue.(string)

	if governanceChat == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "StatChat discussions are not configured"})
		return
	}
	conversation, err := governanceChat.ensureDiscussion(c.Request.Context(), userID, tenantID, workspaceID, req.ObjectType, req.ObjectID, name)
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
