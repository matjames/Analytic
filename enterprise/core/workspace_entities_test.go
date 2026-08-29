package main

import "testing"

func TestCanManageWorkspaceAllowsWorkspaceOwnersAndAdmins(t *testing.T) {
	for _, role := range []string{"owner", "admin"} {
		if !canManageWorkspace(role, "viewer") {
			t.Fatalf("expected workspace role %q to manage workspace", role)
		}
	}
}

func TestCanManageWorkspaceAllowsPlatformAdministrators(t *testing.T) {
	for _, role := range []string{"platform_admin", "superadmin"} {
		if !canManageWorkspace("member", role) {
			t.Fatalf("expected platform role %q to manage workspace", role)
		}
		if !canManageWorkspace("", role) {
			t.Fatalf("expected platform role %q to manage workspace without membership", role)
		}
	}
}

func TestCanManageWorkspaceRejectsRegularMembersAndTenantAdmins(t *testing.T) {
	cases := []struct {
		memberRole   string
		platformRole string
	}{
		{memberRole: "member", platformRole: "viewer"},
		{memberRole: "member", platformRole: "tenant_admin"},
		{memberRole: "", platformRole: "admin"},
		{memberRole: "", platformRole: "tenant_admin"},
	}
	for _, tc := range cases {
		if canManageWorkspace(tc.memberRole, tc.platformRole) {
			t.Fatalf("expected member=%q platform=%q to be rejected", tc.memberRole, tc.platformRole)
		}
	}
}
