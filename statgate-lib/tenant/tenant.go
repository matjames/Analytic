package tenant

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type tenantKey string

const ContextTenantID tenantKey = "statgate_tenant_id"

var (
	ErrMissingTenant     = errors.New("missing tenant identifier")
	ErrTenantMismatch    = errors.New("cross-tenant access forbidden: tenant header does not match authenticated token")
	ErrInvalidTenantSlug = errors.New("tenant identifier contains invalid characters")
)

// ValidateTenantSlug checks if a tenant string is safe and valid.
func ValidateTenantSlug(tenant string) error {
	t := strings.TrimSpace(tenant)
	if t == "" {
		return ErrMissingTenant
	}
	if len(t) < 2 || len(t) > 64 {
		return errors.New("tenant ID must be between 2 and 64 characters")
	}
	for _, ch := range t {
		if !(ch >= 'a' && ch <= 'z') && !(ch >= 'A' && ch <= 'Z') && !(ch >= '0' && ch <= '9') && ch != '-' && ch != '_' {
			return ErrInvalidTenantSlug
		}
	}
	return nil
}

// GinTenantIsolation ensures the request's explicit X-Tenant-ID matches the token's authenticated tenant_id.
func GinTenantIsolation() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenTenant, _ := c.Get("tenant_id")
		tokenTenantStr, _ := tokenTenant.(string)

		headerTenant := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))

		// If header is provided, it must strictly match token tenant
		if headerTenant != "" && tokenTenantStr != "" {
			if !strings.EqualFold(headerTenant, tokenTenantStr) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "cross_tenant_violation",
					"message": "Tenant in X-Tenant-ID header does not match token tenant context",
				})
				return
			}
		}

		effectiveTenant := tokenTenantStr
		if effectiveTenant == "" {
			effectiveTenant = headerTenant
		}

		if effectiveTenant == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "missing_tenant",
				"message": "A valid tenant context is required for enterprise operations",
			})
			return
		}

		if err := ValidateTenantSlug(effectiveTenant); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_tenant",
				"message": err.Error(),
			})
			return
		}

		c.Set("tenant_id", effectiveTenant)
		c.Next()
	}
}

// EnforceContext injects tenant ID into standard Go context.
func EnforceContext(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, ContextTenantID, tenantID)
}

// FromContext extracts tenant ID from context.
func FromContext(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(ContextTenantID).(string)
	return val, ok
}
