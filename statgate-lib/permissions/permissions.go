package permissions

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Permission string

const (
	PermRead           Permission = "read"
	PermWrite          Permission = "write"
	PermAdmin          Permission = "admin"
	PermPublish        Permission = "publish"
	PermApprove        Permission = "approve"
	PermAudit          Permission = "audit"
	PermExecute        Permission = "execute"
	PermSpatialAccess  Permission = "spatial:access"
	PermWorkflowAction Permission = "workflow:action"
)

// RolePermissionMatrix defines authoritative role capabilities across StatGate.
var RolePermissionMatrix = map[string][]Permission{
	"viewer":             {PermRead},
	"analyst":            {PermRead, PermSpatialAccess},
	"editor":             {PermRead, PermWrite, PermSpatialAccess},
	"operator":           {PermRead, PermWrite, PermExecute, PermSpatialAccess, PermWorkflowAction},
	"manager":            {PermRead, PermWrite, PermExecute, PermApprove, PermSpatialAccess, PermWorkflowAction},
	"agent":              {PermRead, PermWrite},
	"district":           {PermRead, PermWrite, PermSpatialAccess},
	"district_admin":     {PermRead, PermWrite, PermApprove, PermSpatialAccess, PermAdmin},
	"tenant_admin":       {PermRead, PermWrite, PermPublish, PermApprove, PermAudit, PermAdmin, PermExecute, PermSpatialAccess, PermWorkflowAction},
	"governance_officer": {PermRead, PermAudit, PermApprove, PermWorkflowAction},
	"property_officer":   {PermRead, PermWrite, PermSpatialAccess},
	"admin":              {PermRead, PermWrite, PermPublish, PermApprove, PermAudit, PermAdmin, PermExecute, PermSpatialAccess, PermWorkflowAction},
	"superadmin":         {PermRead, PermWrite, PermPublish, PermApprove, PermAudit, PermAdmin, PermExecute, PermSpatialAccess, PermWorkflowAction},
	"platform_admin":     {PermRead, PermWrite, PermPublish, PermApprove, PermAudit, PermAdmin, PermExecute, PermSpatialAccess, PermWorkflowAction},
}

// HasPermission checks if a given role has the required permission.
func HasPermission(role string, required Permission) bool {
	normRole := strings.ToLower(strings.TrimSpace(role))
	perms, exists := RolePermissionMatrix[normRole]
	if !exists {
		return false
	}
	for _, p := range perms {
		if p == required || p == PermAdmin {
			return true
		}
	}
	return false
}

// RequirePermission Gin middleware enforces permission checks.
func RequirePermission(required Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "User role context missing",
			})
			return
		}

		role, _ := roleVal.(string)
		if !HasPermission(role, required) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Role does not possess required permission: " + string(required),
			})
			return
		}

		c.Next()
	}
}
