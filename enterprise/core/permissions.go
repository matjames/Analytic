package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type PermissionPolicy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Role        string                 `json:"role"`
	Resource    string                 `json:"resource"`
	Actions     []string               `json:"actions"`
	Conditions  map[string]interface{} `json:"conditions,omitempty"`
	Priority    int                    `json:"priority"`
	CreatedAt   string                 `json:"created_at"`
}

type RoleDefinition struct {
	Role        string   `json:"role"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Scope       string   `json:"scope"`
	Resources   []string `json:"resources"`
}

type Grant struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Resource    string                 `json:"resource"`
	ResourceID  string                 `json:"resource_id,omitempty"`
	Permission  string                 `json:"permission"`
	FromDate    string                 `json:"from_date,omitempty"`
	ToDate      string                 `json:"to_date,omitempty"`
	DelegatedBy string                 `json:"delegated_by,omitempty"`
	Status      string                 `json:"status"`
	CreatedAt   string                 `json:"created_at"`
	Scopes      map[string]interface{} `json:"scopes,omitempty"`
}

func defaultRoleCatalog() []RoleDefinition {
	return []RoleDefinition{
		{Role: "viewer", Name: "Viewer", Description: "Read-only access to permitted records and dashboards", Scope: "tenant", Resources: []string{"dashboard", "document", "facility", "project", "research", "analytics", "report", "search"}},
		{Role: "analyst", Name: "Analyst", Description: "Analytics, reporting, exports, and dataset review", Scope: "tenant", Resources: []string{"analytics", "dataset", "report", "dashboard", "search"}},
		{Role: "editor", Name: "Editor", Description: "Create and update operational records without approval authority", Scope: "tenant", Resources: []string{"document", "facility", "project", "research", "dataset"}},
		{Role: "operator", Name: "Operator", Description: "Operate workflows, queues, field processes, and routine service actions", Scope: "tenant", Resources: []string{"workflow", "facility", "request", "field", "notification"}},
		{Role: "manager", Name: "Manager", Description: "Manage teams, projects, operational workflows, and approvals", Scope: "tenant", Resources: []string{"project", "programme", "workflow", "request", "team", "dashboard"}},
		{Role: "agent", Name: "Field Agent", Description: "Submit and update assigned field data collection work", Scope: "district", Resources: []string{"field", "survey", "facility", "request"}},
		{Role: "district", Name: "District User", Description: "District-scoped access to local facilities, requests, and reports", Scope: "district", Resources: []string{"facility", "request", "report", "dashboard"}},
		{Role: "district_admin", Name: "District Administrator", Description: "Administer district-scoped users, facilities, and operational requests", Scope: "district", Resources: []string{"facility", "request", "user", "workspace", "dashboard"}},
		{Role: "tenant_admin", Name: "Tenant Administrator", Description: "Administer users, invitations, branding, workspaces, and tenant configuration", Scope: "tenant", Resources: []string{"user", "invitation", "workspace", "branding", "audit", "permission"}},
		{Role: "governance_officer", Name: "Governance Officer", Description: "Review governance records, controls, risks, audit evidence, and policy workflows", Scope: "tenant", Resources: []string{"governance", "audit", "risk", "control", "policy"}},
		{Role: "property_officer", Name: "Property Officer", Description: "Manage property, assets, facilities, and related operational records", Scope: "tenant", Resources: []string{"asset", "property", "facility", "procurement"}},
		{Role: "admin", Name: "Administrator", Description: "Platform administration with cross-tenant operational access", Scope: "platform", Resources: []string{"*"}},
		{Role: "superadmin", Name: "Super Administrator", Description: "Highest platform administration role for break-glass and ownership operations", Scope: "platform", Resources: []string{"*"}},
		{Role: "platform_admin", Name: "Platform Administrator", Description: "Platform-wide administration across tenants, services, roles, and policies", Scope: "platform", Resources: []string{"*"}},
	}
}

func roleDefinition(role string) (RoleDefinition, bool) {
	role = strings.ToLower(strings.TrimSpace(role))
	for _, definition := range defaultRoleCatalog() {
		if definition.Role == role {
			return definition, true
		}
	}
	return RoleDefinition{}, false
}

func actionAllowed(actions []string, action string) bool {
	action = strings.ToLower(strings.TrimSpace(action))
	for _, allowed := range actions {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if allowed == "*" || allowed == action {
			return true
		}
	}
	return false
}

func resourceAllowed(policyResource, resource string) bool {
	policyResource = strings.ToLower(strings.TrimSpace(policyResource))
	resource = strings.ToLower(strings.TrimSpace(resource))
	return policyResource == "*" || policyResource == resource
}

func handleListPolicies(c *gin.Context) {
	policies := fetchPolicies()
	c.JSON(200, gin.H{"policies": policies, "count": len(policies)})
}

func fetchPolicies() []PermissionPolicy {
	if redisClient == nil {
		return defaultPolicies()
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, "statgate:policies", 0, -1).Result()
	policies := make([]PermissionPolicy, 0, len(raw))
	for _, item := range raw {
		var p PermissionPolicy
		if err := json.Unmarshal([]byte(item), &p); err == nil {
			policies = append(policies, p)
		}
	}
	if len(policies) == 0 {
		return defaultPolicies()
	}
	return policies
}

func defaultPolicies() []PermissionPolicy {
	now := time.Now().UTC().Format(time.RFC3339)
	return []PermissionPolicy{
		{ID: "pol_platform_admin", Name: "Platform administrator", Description: "Platform-wide service, tenant, role, and policy administration", Role: "platform_admin", Resource: "*", Actions: []string{"*"}, Priority: 100, CreatedAt: now},
		{ID: "pol_superadmin", Name: "Super administrator", Description: "Break-glass platform-wide administration", Role: "superadmin", Resource: "*", Actions: []string{"*"}, Priority: 100, CreatedAt: now},
		{ID: "pol_admin", Name: "Administrator", Description: "Cross-tenant operational administration", Role: "admin", Resource: "*", Actions: []string{"*"}, Priority: 95, CreatedAt: now},
		{ID: "pol_tenant_admin", Name: "Tenant administrator", Description: "Tenant-scoped user, invitation, workspace, branding, audit, and permission administration", Role: "tenant_admin", Resource: "*", Actions: []string{"read", "write", "approve", "admin"}, Priority: 90, CreatedAt: now},
		{ID: "pol_district_admin", Name: "District administrator", Description: "District-scoped facility, request, user, and workspace administration", Role: "district_admin", Resource: "*", Actions: []string{"read", "write", "approve"}, Priority: 80, CreatedAt: now},
		{ID: "pol_governance_officer", Name: "Governance officer", Description: "Governance, audit, risk, control, and policy review", Role: "governance_officer", Resource: "governance", Actions: []string{"read", "write", "approve", "export"}, Priority: 75, CreatedAt: now},
		{ID: "pol_property_officer", Name: "Property officer", Description: "Property, asset, procurement, and facility operations", Role: "property_officer", Resource: "asset", Actions: []string{"read", "write", "approve", "export"}, Priority: 70, CreatedAt: now},
		{ID: "pol_manager", Name: "Manager", Description: "Project, programme, workflow, request, and team management", Role: "manager", Resource: "*", Actions: []string{"read", "write", "approve"}, Priority: 65, CreatedAt: now},
		{ID: "pol_operator", Name: "Operator", Description: "Workflow, request, field, and notification operations", Role: "operator", Resource: "workflow", Actions: []string{"read", "write"}, Priority: 60, CreatedAt: now},
		{ID: "pol_editor", Name: "Editor", Description: "Tenant-scoped content and operational record editing", Role: "editor", Resource: "*", Actions: []string{"read", "write"}, Priority: 55, CreatedAt: now},
		{ID: "pol_analyst", Name: "Analyst", Description: "Analytics, datasets, reports, dashboard, and search exports", Role: "analyst", Resource: "analytics", Actions: []string{"read", "export"}, Priority: 50, CreatedAt: now},
		{ID: "pol_agent", Name: "Field agent", Description: "Assigned field collection and request submission", Role: "agent", Resource: "field", Actions: []string{"read", "write"}, Priority: 45, CreatedAt: now},
		{ID: "pol_district", Name: "District user", Description: "District-scoped facility, request, report, and dashboard access", Role: "district", Resource: "*", Actions: []string{"read"}, Priority: 40, CreatedAt: now},
		{ID: "pol_viewer", Name: "Viewer", Description: "Read-only access to permitted tenant records and dashboards", Role: "viewer", Resource: "*", Actions: []string{"read"}, Priority: 30, CreatedAt: now},
	}
}

func handleCreatePolicy(c *gin.Context) {
	if !requireRole(c, "admin", "superadmin", "platform_admin", "tenant_admin") {
		return
	}
	var p PermissionPolicy
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(400, gin.H{"error": "invalid policy"})
		return
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("pol_%d", time.Now().UnixNano())
	}
	p.Role = strings.ToLower(strings.TrimSpace(p.Role))
	p.Resource = strings.ToLower(strings.TrimSpace(p.Resource))
	if _, ok := roleDefinition(p.Role); !ok {
		c.JSON(400, gin.H{"error": "unknown role"})
		return
	}
	if p.Resource == "" || len(p.Actions) == 0 {
		c.JSON(400, gin.H{"error": "resource and actions are required"})
		return
	}
	if p.CreatedAt == "" {
		p.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if redisClient != nil {
		data, _ := json.Marshal(p)
		redisClient.RPush(context.Background(), "statgate:policies", string(data))
	}
	c.JSON(201, p)
}

func handleCheckPermission(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	resource := c.Query("resource")
	action := c.Query("action")
	if resource == "" || action == "" {
		c.JSON(400, gin.H{"error": "resource and action required"})
		return
	}
	allowed := evaluatePermission(userID, resource, action)
	c.JSON(200, gin.H{
		"user_id": userID, "resource": resource, "action": action, "allowed": allowed, "policy": "default",
	})
}

func evaluatePermission(userID, resource, action string) bool {
	if userID == "" {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(getEnv("STATGATE_ADMIN_ROLE_"+userID, "viewer")))
	if _, ok := roleDefinition(role); ok {
		for _, p := range fetchPolicies() {
			if strings.ToLower(strings.TrimSpace(p.Role)) == role && resourceAllowed(p.Resource, resource) && actionAllowed(p.Actions, action) {
				return true
			}
		}
	}
	grants := fetchGrants(userID)
	for _, g := range grants {
		if g.Status != "active" {
			continue
		}
		if g.Resource == resource || g.Resource == "*" {
			if g.Permission == action || g.Permission == "admin" {
				return true
			}
		}
	}
	// A delegated resource grant may extend a viewer's default read access.
	// Evaluate it before applying the default-role fallback.
	if role == "" || role == "viewer" {
		return action == "read"
	}
	return false
}

func handleListRoles(c *gin.Context) {
	roles := defaultRoleCatalog()
	c.JSON(200, gin.H{"roles": roles})
}

func handleCreateRole(c *gin.Context) {
	if !requireRole(c, "admin", "superadmin", "platform_admin", "tenant_admin") {
		return
	}
	var role RoleDefinition
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(400, gin.H{"error": "invalid role"})
		return
	}
	role.Role = strings.ToLower(strings.TrimSpace(role.Role))
	if _, ok := roleDefinition(role.Role); !ok {
		c.JSON(400, gin.H{"error": "unknown role"})
		return
	}
	if redisClient != nil {
		data, _ := json.Marshal(role)
		redisClient.RPush(context.Background(), "statgate:roles", string(data))
	}
	c.JSON(201, role)
}

func handleListGrants(c *gin.Context) {
	userID := c.Query("user_id")
	grants := fetchGrants(userID)
	c.JSON(200, gin.H{"grants": grants, "count": len(grants)})
}

func fetchGrants(userID string) []Grant {
	if redisClient == nil {
		return []Grant{}
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, "statgate:grants", 0, -1).Result()
	grants := make([]Grant, 0, len(raw))
	for _, item := range raw {
		var g Grant
		if err := json.Unmarshal([]byte(item), &g); err == nil {
			if userID == "" || g.UserID == userID {
				if g.ToDate != "" {
					if expiry, err := time.Parse(time.RFC3339, g.ToDate); err == nil {
						if time.Now().After(expiry) {
							g.Status = "expired"
						}
					}
				}
				grants = append(grants, g)
			}
		}
	}
	return grants
}

func handleCreateGrant(c *gin.Context) {
	var g Grant
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(400, gin.H{"error": "invalid grant"})
		return
	}
	if g.ID == "" {
		g.ID = fmt.Sprintf("grant_%d", time.Now().UnixNano())
	}
	if g.Status == "" {
		g.Status = "active"
	}
	if g.CreatedAt == "" {
		g.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if redisClient != nil {
		data, _ := json.Marshal(g)
		redisClient.RPush(context.Background(), "statgate:grants", string(data))
	}
	recordAudit("permission.grant", "enterprise", g.UserID, map[string]interface{}{
		"grant_id": g.ID, "resource": g.Resource, "permission": g.Permission,
	})
	c.JSON(201, g)
}

func handleRevokeGrant(c *gin.Context) {
	id := c.Param("id")
	grants := fetchGrants("")
	for _, g := range grants {
		if g.ID == id {
			g.Status = "revoked"
			persistGrant(id, g)
			c.JSON(200, gin.H{"status": "revoked", "id": id})
			return
		}
	}
	c.JSON(404, gin.H{"error": "grant not found"})
}

func handleDelegateGrant(c *gin.Context) {
	id := c.Param("id")
	delegateTo := c.GetHeader("X-User-ID")
	if delegateTo == "" {
		delegateTo = c.Query("delegate_to")
	}
	if delegateTo == "" {
		c.JSON(400, gin.H{"error": "delegate_to required"})
		return
	}
	grants := fetchGrants("")
	for _, g := range grants {
		if g.ID == id {
			newGrant := g
			newGrant.ID = fmt.Sprintf("grant_%d", time.Now().UnixNano())
			newGrant.UserID = delegateTo
			newGrant.DelegatedBy = g.UserID
			newGrant.CreatedAt = time.Now().UTC().Format(time.RFC3339)
			if redisClient != nil {
				data, _ := json.Marshal(newGrant)
				redisClient.RPush(context.Background(), "statgate:grants", string(data))
			}
			c.JSON(201, newGrant)
			return
		}
	}
	c.JSON(404, gin.H{"error": "grant not found"})
}

func persistGrant(id string, updated Grant) {
	if redisClient == nil {
		return
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, "statgate:grants", 0, -1).Result()
	redisClient.Del(ctx, "statgate:grants")
	for _, item := range raw {
		var g Grant
		if err := json.Unmarshal([]byte(item), &g); err == nil {
			if g.ID == id {
				g = updated
			}
			data, _ := json.Marshal(g)
			redisClient.RPush(ctx, "statgate:grants", string(data))
		}
	}
}
