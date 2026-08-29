package main

import "testing"

func TestDefaultRoleCatalogMatchesJWTWhitelist(t *testing.T) {
	roles := defaultRoleCatalog()
	if len(roles) != len(validRoles) {
		t.Fatalf("expected %d roles, got %d", len(validRoles), len(roles))
	}

	seen := map[string]bool{}
	for _, role := range roles {
		if !validRoles[role.Role] {
			t.Fatalf("catalog role %q is not accepted by JWT middleware", role.Role)
		}
		if seen[role.Role] {
			t.Fatalf("duplicate role %q", role.Role)
		}
		if role.Name == "" || role.Description == "" || role.Scope == "" || len(role.Resources) == 0 {
			t.Fatalf("role %q is missing catalog metadata: %#v", role.Role, role)
		}
		seen[role.Role] = true
	}

	for role := range validRoles {
		if !seen[role] {
			t.Fatalf("JWT role %q is missing from permission catalog", role)
		}
	}
}

func TestDefaultPoliciesUseCanonicalRoles(t *testing.T) {
	for _, policy := range defaultPolicies() {
		if _, ok := roleDefinition(policy.Role); !ok {
			t.Fatalf("policy %q references unknown role %q", policy.ID, policy.Role)
		}
		if policy.Resource == "" || len(policy.Actions) == 0 {
			t.Fatalf("policy %q is missing resource/actions: %#v", policy.ID, policy)
		}
	}
}

func TestEvaluatePermissionUsesCanonicalDefaultPolicies(t *testing.T) {
	previousRedis := redisClient
	redisClient = nil
	t.Cleanup(func() { redisClient = previousRedis })

	t.Setenv("STATGATE_ADMIN_ROLE_1", "platform_admin")
	t.Setenv("STATGATE_ADMIN_ROLE_2", "analyst")
	t.Setenv("STATGATE_ADMIN_ROLE_3", "viewer")
	t.Setenv("STATGATE_ADMIN_ROLE_4", "root")

	if !evaluatePermission("1", "user", "delete") {
		t.Fatalf("expected platform admin wildcard permission")
	}
	if !evaluatePermission("2", "analytics", "export") {
		t.Fatalf("expected analyst analytics export permission")
	}
	if evaluatePermission("2", "user", "admin") {
		t.Fatalf("analyst should not administer users")
	}
	if evaluatePermission("3", "analytics", "write") {
		t.Fatalf("viewer should not write analytics")
	}
	if evaluatePermission("4", "analytics", "read") {
		t.Fatalf("unknown roles should not receive default permissions")
	}
}
