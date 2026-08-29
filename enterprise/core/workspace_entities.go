package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type platformWorkspace struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Role        string `json:"role,omitempty"`
}

type workspaceMember struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	JoinedAt string `json:"joined_at"`
}

var workspaceSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

func workspaceID() string {
	return fmt.Sprintf("ws-%d", time.Now().UTC().UnixNano())
}

func handleListWorkspaces(c *gin.Context) {
	if dbPool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workspace persistence unavailable"})
		return
	}
	userID, tenantID := getContextUserID(c), getContextTenantID(c)
	rows, err := dbPool.Query(`
		SELECT w.id, w.tenant_id, w.name, w.slug, w.description, w.status, w.created_by,
		       w.created_at, w.updated_at, m.role
		FROM platform_workspaces w
		JOIN platform_workspace_members m ON m.workspace_id = w.id
		WHERE w.tenant_id = $1 AND m.user_id = $2 AND m.status = 'active' AND w.status = 'active'
		ORDER BY w.name`, tenantID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list workspaces"})
		return
	}
	defer rows.Close()
	workspaces := make([]platformWorkspace, 0)
	for rows.Next() {
		var w platformWorkspace
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&w.ID, &w.TenantID, &w.Name, &w.Slug, &w.Description, &w.Status, &w.CreatedBy, &createdAt, &updatedAt, &w.Role); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read workspaces"})
			return
		}
		w.CreatedAt, w.UpdatedAt = createdAt.UTC().Format(time.RFC3339), updatedAt.UTC().Format(time.RFC3339)
		workspaces = append(workspaces, w)
	}
	c.JSON(http.StatusOK, gin.H{"workspaces": workspaces})
}

func handleCreateWorkspace(c *gin.Context) {
	if dbPool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workspace persistence unavailable"})
		return
	}
	if !requireRole(c, "admin", "superadmin", "platform_admin", "tenant_admin") {
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Slug        string `json:"slug" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and slug are required"})
		return
	}
	req.Name, req.Slug = strings.TrimSpace(req.Name), strings.ToLower(strings.TrimSpace(req.Slug))
	if req.Name == "" || !workspaceSlugPattern.MatchString(req.Slug) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug must contain 2-63 lowercase letters, numbers, or hyphens"})
		return
	}
	id, tenantID, userID := workspaceID(), getContextTenantID(c), getContextUserID(c)
	var w platformWorkspace
	var createdAt, updatedAt time.Time
	err := dbPool.QueryRow(`
		INSERT INTO platform_workspaces(id, tenant_id, name, slug, description, created_by)
		VALUES($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, name, slug, description, status, created_by, created_at, updated_at`,
		id, tenantID, req.Name, req.Slug, strings.TrimSpace(req.Description), userID,
	).Scan(&w.ID, &w.TenantID, &w.Name, &w.Slug, &w.Description, &w.Status, &w.CreatedBy, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "workspace slug already exists or could not be created"})
		return
	}
	if _, err := dbPool.Exec(`INSERT INTO platform_workspace_members(workspace_id, user_id, role) VALUES($1, $2, 'owner')`, id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "workspace owner could not be created"})
		return
	}
	w.CreatedAt, w.UpdatedAt, w.Role = createdAt.UTC().Format(time.RFC3339), updatedAt.UTC().Format(time.RFC3339), "owner"
	c.JSON(http.StatusCreated, gin.H{"workspace": w})
}

func workspaceMemberRole(workspaceID, userID, tenantID string) (string, error) {
	var role string
	err := dbPool.QueryRow(`
		SELECT m.role FROM platform_workspace_members m
		JOIN platform_workspaces w ON w.id = m.workspace_id
		WHERE m.workspace_id = $1 AND m.user_id = $2 AND m.status = 'active' AND w.tenant_id = $3 AND w.status = 'active'`,
		workspaceID, userID, tenantID).Scan(&role)
	return role, err
}

func canManageWorkspace(memberRole, platformRole string) bool {
	switch memberRole {
	case "owner", "admin":
		return true
	}
	switch platformRole {
	case "platform_admin", "superadmin":
		return true
	default:
		return false
	}
}

func requireWorkspaceManager(c *gin.Context, workspaceID string) bool {
	if dbPool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workspace persistence unavailable"})
		return false
	}
	role, err := workspaceMemberRole(workspaceID, getContextUserID(c), getContextTenantID(c))
	if err != nil || !canManageWorkspace(role, getContextRole(c)) {
		if canManageWorkspace("", getContextRole(c)) {
			return true
		}
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "workspace owner or admin access required",
		})
		return false
	}
	return true
}

func handleGetWorkspace(c *gin.Context) {
	if dbPool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workspace persistence unavailable"})
		return
	}
	workspaceID, userID, tenantID := c.Param("id"), getContextUserID(c), getContextTenantID(c)
	role, err := workspaceMemberRole(workspaceID, userID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}
	var w platformWorkspace
	var createdAt, updatedAt time.Time
	err = dbPool.QueryRow(`SELECT id, tenant_id, name, slug, description, status, created_by, created_at, updated_at FROM platform_workspaces WHERE id = $1 AND tenant_id = $2 AND status = 'active'`, workspaceID, tenantID).
		Scan(&w.ID, &w.TenantID, &w.Name, &w.Slug, &w.Description, &w.Status, &w.CreatedBy, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}
	w.CreatedAt, w.UpdatedAt, w.Role = createdAt.UTC().Format(time.RFC3339), updatedAt.UTC().Format(time.RFC3339), role
	c.JSON(http.StatusOK, gin.H{"workspace": w})
}

func handleListWorkspaceMembers(c *gin.Context) {
	workspaceID, tenantID := c.Param("id"), getContextTenantID(c)
	if dbPool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workspace persistence unavailable"})
		return
	}
	if _, err := workspaceMemberRole(workspaceID, getContextUserID(c), tenantID); err != nil && !requireRole(c, "platform_admin", "superadmin") {
		return
	}
	rows, err := dbPool.Query(`SELECT m.user_id, m.role, m.status, m.joined_at FROM platform_workspace_members m JOIN platform_workspaces w ON w.id=m.workspace_id WHERE m.workspace_id=$1 AND w.tenant_id=$2 ORDER BY m.joined_at`, workspaceID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list workspace members"})
		return
	}
	defer rows.Close()
	members := make([]workspaceMember, 0)
	for rows.Next() {
		var member workspaceMember
		var joinedAt time.Time
		if err := rows.Scan(&member.UserID, &member.Role, &member.Status, &joinedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read workspace members"})
			return
		}
		member.JoinedAt = joinedAt.UTC().Format(time.RFC3339)
		members = append(members, member)
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

func handleAddWorkspaceMember(c *gin.Context) {
	if !requireWorkspaceManager(c, c.Param("id")) {
		return
	}
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Role   string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.UserID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}
	if req.Role != "member" && req.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be member or admin"})
		return
	}
	_, err := dbPool.Exec(`INSERT INTO platform_workspace_members(workspace_id, user_id, role) SELECT w.id, $2, $3 FROM platform_workspaces w WHERE w.id=$1 AND w.tenant_id=$4 ON CONFLICT(workspace_id, user_id) DO UPDATE SET role = EXCLUDED.role, status = 'active'`, c.Param("id"), strings.TrimSpace(req.UserID), req.Role, getContextTenantID(c))
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "member could not be added"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "member_added", "workspace_id": c.Param("id"), "user_id": strings.TrimSpace(req.UserID), "role": req.Role})
}

func handleRemoveWorkspaceMember(c *gin.Context) {
	if !requireWorkspaceManager(c, c.Param("id")) {
		return
	}
	result, err := dbPool.Exec(`UPDATE platform_workspace_members m SET status = 'removed' FROM platform_workspaces w WHERE m.workspace_id = $1 AND m.user_id = $2 AND m.role <> 'owner' AND w.id=m.workspace_id AND w.tenant_id=$3`, c.Param("id"), c.Param("user"), getContextTenantID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "member could not be removed"})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found or owner cannot be removed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "member_removed"})
}
