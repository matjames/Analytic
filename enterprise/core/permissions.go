package main

import (
	"context"
	"encoding/json"
	"fmt"
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
		{ID: "pol_admin", Name: "Administrator", Description: "Full platform access", Role: "admin", Resource: "*", Actions: []string{"*"}, Priority: 100, CreatedAt: now},
		{ID: "pol_manager", Name: "Manager", Description: "Manage projects and teams", Role: "manager", Resource: "project", Actions: []string{"read", "write", "approve"}, Priority: 80, CreatedAt: now},
		{ID: "pol_researcher", Name: "Researcher", Description: "Research operations", Role: "researcher", Resource: "research", Actions: []string{"read", "write"}, Priority: 60, CreatedAt: now},
		{ID: "pol_analyst", Name: "Analyst", Description: "Analytics and reporting", Role: "analyst", Resource: "analytics", Actions: []string{"read", "export"}, Priority: 60, CreatedAt: now},
		{ID: "pol_viewer", Name: "Viewer", Description: "Read-only access", Role: "viewer", Resource: "*", Actions: []string{"read"}, Priority: 40, CreatedAt: now},
	}
}

func handleCreatePolicy(c *gin.Context) {
	var p PermissionPolicy
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(400, gin.H{"error": "invalid policy"})
		return
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("pol_%d", time.Now().UnixNano())
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
	role := getEnv("STATGATE_ADMIN_ROLE_"+userID, "viewer")
	if role == "admin" {
		return true
	}
	if role == "manager" && (action == "read" || action == "write" || action == "approve") {
		return true
	}
	if role == "researcher" && (action == "read" || action == "write") && (resource == "research" || resource == "dataset") {
		return true
	}
	if role == "analyst" && (action == "read" || action == "export") && (resource == "analytics" || resource == "report") {
		return true
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
	roles := []map[string]interface{}{
		{"role": "admin", "name": "Administrator", "description": "Full access"},
		{"role": "manager", "name": "Manager", "description": "Manage teams and projects"},
		{"role": "researcher", "name": "Researcher", "description": "Research operations"},
		{"role": "analyst", "name": "Analyst", "description": "Analytics and reporting"},
		{"role": "field_worker", "name": "Field Worker", "description": "Field data collection"},
		{"role": "viewer", "name": "Viewer", "description": "Read-only"},
	}
	c.JSON(200, gin.H{"roles": roles})
}

func handleCreateRole(c *gin.Context) {
	var role map[string]interface{}
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(400, gin.H{"error": "invalid role"})
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
