package api

import (
	"fmt"
	"strings"
)

func resolveTenantID(tenantID string) string {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return "default"
	}
	return tenantID
}

func routeConversationKey(tenantID, conversationID string) string {
	tenantID = resolveTenantID(tenantID)
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		conversationID = "general"
	}
	return fmt.Sprintf("%s:%s", tenantID, conversationID)
}
