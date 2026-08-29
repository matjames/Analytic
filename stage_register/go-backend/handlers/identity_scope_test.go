package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func scopedTestContext(role, tenant, district string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_role", role)
	if tenant != "" {
		c.Set("tenant_id", tenant)
	}
	if district != "" {
		c.Set("district_id", district)
	}
	return c
}

func TestScopedOrganisationForWritePinsTenantAdminsToAuthenticatedTenant(t *testing.T) {
	requested := "other-org"
	c := scopedTestContext("tenant_admin", "tenant-a", "district-a")

	org, ok := scopedOrganisationForWrite(c, &requested)
	if !ok {
		t.Fatalf("expected tenant admin scope to be valid")
	}
	if org == nil || *org != "tenant-a" {
		t.Fatalf("expected authenticated tenant, got %#v", org)
	}
}

func TestScopedOrganisationForWriteAllowsPlatformAdminRequest(t *testing.T) {
	requested := " other-org "
	c := scopedTestContext("platform_admin", "tenant-a", "")

	org, ok := scopedOrganisationForWrite(c, &requested)
	if !ok {
		t.Fatalf("expected platform admin scope to be valid")
	}
	if org == nil || *org != "other-org" {
		t.Fatalf("expected requested organisation, got %#v", org)
	}
}

func TestScopedOrganisationForWriteRejectsTenantAdminWithoutTenant(t *testing.T) {
	requested := "other-org"
	c := scopedTestContext("tenant_admin", "", "")

	if _, ok := scopedOrganisationForWrite(c, &requested); ok {
		t.Fatalf("expected tenant admin without tenant context to be rejected")
	}
}

func TestAppendAuthenticatedUserScopeIncludesDistrictFallback(t *testing.T) {
	c := scopedTestContext("tenant_admin", "tenant-a", "district-a")

	where, values, idx, ok := appendAuthenticatedUserScope(c, []string{"id = $1"}, []interface{}{int64(42)}, 2)
	if !ok {
		t.Fatalf("expected scope to be valid")
	}
	if idx != 4 {
		t.Fatalf("expected next placeholder index 4, got %d", idx)
	}
	if len(where) != 2 || where[1] != "(organisation = $2 OR district_id = $3)" {
		t.Fatalf("unexpected where clause: %#v", where)
	}
	if len(values) != 3 || values[1] != "tenant-a" || values[2] != "district-a" {
		t.Fatalf("unexpected values: %#v", values)
	}
}

func TestAppendAuthenticatedOrganisationScopeOmitsDistrictForInvitationTables(t *testing.T) {
	c := scopedTestContext("tenant_admin", "tenant-a", "district-a")

	where, values, idx, ok := appendAuthenticatedOrganisationScope(c, "organisation", nil, nil, 1)
	if !ok {
		t.Fatalf("expected scope to be valid")
	}
	if idx != 2 {
		t.Fatalf("expected next placeholder index 2, got %d", idx)
	}
	if len(where) != 1 || where[0] != "organisation = $1" {
		t.Fatalf("unexpected where clause: %#v", where)
	}
	if len(values) != 1 || values[0] != "tenant-a" {
		t.Fatalf("unexpected values: %#v", values)
	}
}

func TestTenantAdminCannotManagePlatformRoles(t *testing.T) {
	role := "governance_officer"
	c := scopedTestContext("tenant_admin", "tenant-a", "")

	if canAuthenticatedAdminManageRole(c, &role) {
		t.Fatalf("expected tenant admin to be blocked from privileged role")
	}
}

func TestPlatformAdminCanManagePrivilegedRoles(t *testing.T) {
	role := "governance_officer"
	c := scopedTestContext("platform_admin", "", "")

	if !canAuthenticatedAdminManageRole(c, &role) {
		t.Fatalf("expected platform admin to be allowed to manage privileged role")
	}
}

func TestRequireDirectoryAccessAllowsInternalService(t *testing.T) {
	c := scopedTestContext("", "", "")
	c.Set("internal_service", true)

	if !requireDirectoryAccess(c) {
		t.Fatalf("expected internal service directory access")
	}
}

func TestAppendAuthenticatedUserScopePinsInternalDirectoryToRequestedTenant(t *testing.T) {
	c := scopedTestContext("", "", "")
	c.Set("internal_service", true)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/internal/users", nil)
	c.Request.Header.Set("X-Tenant-ID", "tenant-a")

	where, values, idx, ok := appendAuthenticatedUserScope(c, nil, nil, 1)
	if !ok {
		t.Fatalf("expected internal service scope to be valid")
	}
	if idx != 2 || len(where) != 1 || where[0] != "organisation = $1" {
		t.Fatalf("unexpected scoped query: where=%#v idx=%d", where, idx)
	}
	if len(values) != 1 || values[0] != "tenant-a" {
		t.Fatalf("unexpected scoped values: %#v", values)
	}
}

func TestListUsersRejectsNonAdminBeforeDatabaseAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("user_role", "viewer")

	ListUsers(c)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
}
