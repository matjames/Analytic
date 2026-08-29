package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

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
		if workspaceID := strings.TrimSpace(c.GetHeader("X-Workspace-ID")); workspaceID != "" {
			if len(workspaceID) > 128 || strings.ContainsAny(workspaceID, " /\\\t\r\n") {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_workspace_context"})
				return
			}
			if !workspaceMember(c, workspaceID, authHeader) {
				return
			}
			c.Set("workspace_id", workspaceID)
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

func workspaceIDContext(c *gin.Context) string {
	if value, exists := c.Get("workspace_id"); exists {
		if workspaceID, ok := value.(string); ok {
			return strings.TrimSpace(workspaceID)
		}
	}
	return ""
}

func workspaceResearchMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		workspaceID := workspaceIDContext(c)
		if workspaceID == "" || !strings.HasPrefix(c.Request.URL.Path, "/api/research/") {
			c.Next()
			return
		}
		var exists bool
		if err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM rms.research_projects WHERE id=$1 AND workspace_id=$2)`, c.Param("id"), workspaceID).Scan(&exists); err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "workspace scope unavailable"})
			return
		}
		if !exists {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "research study not found in workspace"})
			return
		}
		c.Next()
	}
}

func workspaceResearchBodyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		workspaceID := workspaceIDContext(c)
		if workspaceID == "" || c.Request.Method == http.MethodGet || !strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
			c.Next()
			return
		}
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		var payload map[string]interface{}
		if json.Unmarshal(body, &payload) == nil {
			if raw, ok := payload["research_id"]; ok {
				if !workspaceResearchReference(c, fmt.Sprint(raw), workspaceID) {
					return
				}
			} else if raw, ok := payload["researchId"]; ok {
				if !workspaceResearchReference(c, fmt.Sprint(raw), workspaceID) {
					return
				}
			}
		}
		c.Next()
	}
}

func workspaceResearchChildIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		workspaceID := workspaceIDContext(c)
		path := c.Request.URL.Path
		if workspaceID == "" || (c.Request.Method != http.MethodPut && c.Request.Method != http.MethodDelete) || strings.HasPrefix(path, "/api/research/") {
			c.Next()
			return
		}
		prefixes := []string{"/api/members/", "/api/proposals/", "/api/ethics/", "/api/grants/", "/api/literature/", "/api/datasets/", "/api/publications/", "/api/tasks/", "/api/risks/", "/api/issues/", "/api/documents/", "/api/surveys/", "/api/reports/", "/api/calendar/"}
		matched := false
		for _, prefix := range prefixes {
			matched = matched || strings.HasPrefix(path, prefix)
		}
		if !matched {
			c.Next()
			return
		}
		childID := c.Param("id")
		tables := []string{"research_members", "proposals", "ethics_applications", "grants", "literature", "datasets", "publications", "tasks", "risks", "issues", "documents", "surveys", "reports", "calendar_events", "chat_messages", "audit_logs"}
		for _, table := range tables {
			var exists bool
			query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM rms.%s child JOIN rms.research_projects project ON project.id=child.research_id WHERE child.id=$1 AND project.workspace_id=$2)`, table)
			if err := DB.QueryRow(query, childID, workspaceID).Scan(&exists); err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "workspace scope unavailable"})
				return
			}
			if exists {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "resource not found in workspace"})
	}
}

func workspaceResearchReference(c *gin.Context, researchID, workspaceID string) bool {
	var exists bool
	if err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM rms.research_projects WHERE id=$1 AND workspace_id=$2)`, researchID, workspaceID).Scan(&exists); err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "workspace scope unavailable"})
		return false
	}
	if !exists {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "research study not found in workspace"})
		return false
	}
	return true
}

func workspaceMember(c *gin.Context, workspaceID, authHeader string) bool {
	base := strings.TrimRight(os.Getenv("STATGATE_ENTERPRISE_API_URL"), "/")
	if base == "" {
		base = "http://localhost:8096/api"
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, fmt.Sprintf("%s/workspaces/%s", base, workspaceID), nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "workspace membership unavailable"})
		return false
	}
	req.Header.Set("Authorization", authHeader)
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "workspace membership unavailable"})
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "workspace membership required"})
		return false
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "workspace membership unavailable"})
		return false
	}
	return true
}
