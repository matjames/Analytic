package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func currentRegistryRole(c *gin.Context) string {
	roleValue, _ := c.Get("user_role")
	role, _ := roleValue.(string)
	return strings.ToLower(strings.TrimSpace(role))
}

func isPlatformAdmin(c *gin.Context) bool {
	switch currentRegistryRole(c) {
	case "admin", "superadmin", "platform_admin":
		return true
	default:
		return false
	}
}

func isInternalService(c *gin.Context) bool {
	value, _ := c.Get("internal_service")
	internal, _ := value.(bool)
	return internal
}

func requireDirectoryAccess(c *gin.Context) bool {
	if isInternalService(c) {
		return true
	}
	return requireInvitationAdmin(c)
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func authenticatedOrganisation(c *gin.Context) (string, bool) {
	org := strings.TrimSpace(c.GetString("tenant_id"))
	if org == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "authenticated organisation context is required"})
		return "", false
	}
	return org, true
}

func authenticatedDistrict(c *gin.Context) *string {
	district := strings.TrimSpace(c.GetString("district_id"))
	if district == "" {
		return nil
	}
	return &district
}

func scopedOrganisationForWrite(c *gin.Context, requested *string) (*string, bool) {
	if isPlatformAdmin(c) {
		return trimOptionalString(requested), true
	}
	org, ok := authenticatedOrganisation(c)
	if !ok {
		return nil, false
	}
	return &org, true
}

func scopedDistrictForWrite(c *gin.Context, requested *string) *string {
	if isPlatformAdmin(c) {
		return trimOptionalString(requested)
	}
	return authenticatedDistrict(c)
}

func canAuthenticatedAdminManageRole(c *gin.Context, role *string) bool {
	if role == nil || strings.TrimSpace(*role) == "" || isPlatformAdmin(c) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(*role)) {
	case "viewer", "analyst", "editor", "operator", "manager", "agent", "district", "district_admin", "tenant_admin":
		return true
	default:
		return false
	}
}

func roleAllowedForAuthenticatedAdmin(c *gin.Context, role *string) bool {
	if canAuthenticatedAdminManageRole(c, role) {
		return true
	}
	switch currentRegistryRole(c) {
	case "tenant_admin":
		c.JSON(http.StatusForbidden, gin.H{"error": "role is not permitted for tenant administration"})
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "role is not permitted for administration"})
	}
	return false
}

func appendAuthenticatedUserScope(c *gin.Context, where []string, values []interface{}, idx int) ([]string, []interface{}, int, bool) {
	// Internal services (e.g. the StatChat directory synchronizer) are authorized
	// by STATGATE_INTERNAL_API_KEY — a server-to-server credential. They must be
	// able to read the authoritative (unscoped) Registry directory without a
	// browser session/tenant context (SG-SEC-2026-08 keeps the Registry as the
	// identity authority).
	if isPlatformAdmin(c) {
		return where, values, idx, true
	}
	if isInternalService(c) {
		// Internal callers may request a tenant-scoped directory without needing
		// an end-user admin token. The internal-service middleware authenticates
		// this header before it can influence the query.
		if org := strings.TrimSpace(c.GetHeader("X-Tenant-ID")); org != "" {
			where = append(where, "organisation = $"+strconv.Itoa(idx))
			values = append(values, org)
			return where, values, idx + 1, true
		}
		return where, values, idx, true
	}

	org, ok := authenticatedOrganisation(c)
	if !ok {
		return where, values, idx, false
	}
	district := authenticatedDistrict(c)
	if district == nil {
		where = append(where, "organisation = $"+strconv.Itoa(idx))
		values = append(values, org)
		return where, values, idx + 1, true
	}

	where = append(where, "(organisation = $"+strconv.Itoa(idx)+" OR district_id = $"+strconv.Itoa(idx+1)+")")
	values = append(values, org, *district)
	return where, values, idx + 2, true
}

func appendAuthenticatedOrganisationScope(c *gin.Context, column string, where []string, values []interface{}, idx int) ([]string, []interface{}, int, bool) {
	if isPlatformAdmin(c) {
		return where, values, idx, true
	}
	org, ok := authenticatedOrganisation(c)
	if !ok {
		return where, values, idx, false
	}
	where = append(where, column+" = $"+strconv.Itoa(idx))
	values = append(values, org)
	return where, values, idx + 1, true
}

func combineWhere(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(conditions, " AND ")
}
