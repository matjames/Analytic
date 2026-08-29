package api

import (
	"fmt"
	"strings"

	"statchat/pkg/model"
	"statchat/pkg/store"
)

func canUseMentionAll(identity model.User, access store.ConversationMemberAccess) bool {
	if access.Type == model.ConversationTypeDirect {
		return false
	}
	if access.ActorRole == "owner" || access.ActorRole == "admin" {
		return true
	}
	for _, role := range identity.Roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "admin", "superadmin", "tenant_admin", "platform_admin":
			return true
		}
	}
	return false
}

func resolveMessageMentions(conversationID, senderID, tenantID string, requested []string, mentionAll bool, identity model.User) ([]string, error) {
	if len(requested) > 20 {
		return nil, fmt.Errorf("at most 20 individual mentions are allowed")
	}
	members, err := store.GetConversationMembers(conversationID)
	if err != nil {
		return nil, err
	}
	if mentionAll {
		access, err := store.GetConversationMemberAccess(conversationID, tenantID, senderID)
		if err != nil || !canUseMentionAll(identity, access) {
			return nil, fmt.Errorf("mentioning everyone requires conversation management permission")
		}
	}
	targets, err := store.NormalizeMentionTargets(requested, members, senderID, mentionAll)
	if err != nil {
		return nil, fmt.Errorf("mention targets must be conversation members")
	}
	return targets, nil
}
