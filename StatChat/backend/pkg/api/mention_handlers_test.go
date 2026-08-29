package api

import (
	"testing"

	"statchat/pkg/model"
	"statchat/pkg/store"
)

func TestCanUseMentionAll(t *testing.T) {
	if canUseMentionAll(model.User{Roles: []string{"viewer"}}, store.ConversationMemberAccess{Type: model.ConversationTypeDirect, ActorRole: "owner"}) {
		t.Fatal("direct conversations must not allow mention all")
	}
	if !canUseMentionAll(model.User{}, store.ConversationMemberAccess{Type: model.ConversationTypeGroup, ActorRole: "owner"}) {
		t.Fatal("conversation owners should be allowed")
	}
	if canUseMentionAll(model.User{Roles: []string{"viewer"}}, store.ConversationMemberAccess{Type: model.ConversationTypeGroup}) {
		t.Fatal("ordinary members should not be allowed")
	}
	if !canUseMentionAll(model.User{Roles: []string{"tenant_admin"}}, store.ConversationMemberAccess{Type: model.ConversationTypeChannel}) {
		t.Fatal("tenant administrators should be allowed")
	}
}
