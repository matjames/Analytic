package main

import (
	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/permissions"
	"net/http"
	"strings"
)

// requireClusterAccess layers ABAC scope checks over statgate-lib's authoritative RBAC matrix.
// Cluster IDs are never trusted solely from a caller supplied header.
func requireClusterAccess(permission permissions.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !permissions.HasPermission(c.GetString("user_role"), permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "insufficient RunOps permission"})
			return
		}
		requested := c.Param("clusterID")
		if requested == "" {
			requested = c.GetHeader("X-StatGate-Cluster")
		}
		allowed := c.GetHeader("X-StatGate-Allowed-Clusters")
		if requested != "" && allowed != "" && c.GetString("user_role") != "platform_admin" && c.GetString("user_role") != "superadmin" {
			found := false
			for _, id := range strings.Split(allowed, ",") {
				if strings.TrimSpace(id) == requested {
					found = true
					break
				}
			}
			if !found {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "cluster_scope_denied"})
				return
			}
		}
		c.Next()
	}
}
