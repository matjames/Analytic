package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/auth"
)

// registryAuthMiddleware validates Registry JWTs using the shared statgate-lib
// auth validator (Fail-closed zero-default, issuer/audience, canonical role
// whitelist, required tenant context, mock/demo rejection). It also enforces
// tenant-header isolation (X-Tenant-ID must match the token).
func registryAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow public health/ready/live/metrics probes without auth
		path := c.Request.URL.Path
		if path == "/health" || path == "/ready" || path == "/metrics" || path == "/live" {
			c.Next()
			return
		}

		validator, err := auth.NewValidator("", "", "")
		if err != nil {
			// Fail closed (SG-SEC-2026-08): missing secret is misconfiguration.
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":   "service_misconfigured",
				"message": "STATGATE_REGISTRY_JWT_SECRET is not configured",
			})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
			return
		}

		uCtx, err := validator.ValidateToken(strings.TrimSpace(parts[1]))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Tenant header isolation: an explicit X-Tenant-ID must match the token.
		headerTenant := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
		if headerTenant != "" && uCtx.TenantID != "" && !strings.EqualFold(headerTenant, uCtx.TenantID) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "cross_tenant_violation", "message": "Header tenant mismatch with token"})
			return
		}

		c.Set("user_id", uCtx.UserID)
		c.Set("user_role", uCtx.Role)
		c.Set("tenant_id", uCtx.TenantID)
		c.Set("org_id", uCtx.OrgID)
		c.Set("email", uCtx.Email)
		c.Set("user_context", uCtx)

		c.Next()
	}
}
