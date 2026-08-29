package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"statchat/pkg/model"
	"statchat/pkg/store"
)

func memberAuthorizationRequest(userID string, roles ...string) *http.Request {
	req := httptest.NewRequest("GET", "/v1/chat/conversations/group-1/members", nil)
	identity := model.User{ID: userID, OrganizationID: "tenant-1", Roles: roles}
	ctx := context.WithValue(req.Context(), requestUserIDKey, userID)
	ctx = context.WithValue(ctx, requestIdentityKey, identity)
	return req.WithContext(ctx)
}

func TestConversationOwnerCanManageMembers(t *testing.T) {
	access := store.ConversationMemberAccess{ActorRole: "owner"}
	if !canManageConversationMembers(memberAuthorizationRequest("owner-1", "viewer"), access) {
		t.Fatal("expected conversation owner to manage members")
	}
}

func TestOrdinaryConversationMemberCannotManageMembers(t *testing.T) {
	access := store.ConversationMemberAccess{}
	if canManageConversationMembers(memberAuthorizationRequest("member-1", "viewer"), access) {
		t.Fatal("expected ordinary member to be denied")
	}
}

func TestCanonicalAdministratorsCanManageMembers(t *testing.T) {
	for _, role := range []string{"admin", "superadmin", "tenant_admin", "platform_admin"} {
		t.Run(role, func(t *testing.T) {
			if !canManageConversationMembers(memberAuthorizationRequest("admin-1", role), store.ConversationMemberAccess{}) {
				t.Fatalf("expected %s to manage members", role)
			}
		})
	}
}
