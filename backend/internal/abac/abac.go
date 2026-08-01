package abac

import (
	"errors"
	"strings"
)

type UserAttributes struct {
	UserID     string   `json:"user_id"`
	TenantID   string   `json:"tenant_id"`
	Role       string   `json:"role"`       // admin, analyst, viewer
	Clearance  int      `json:"clearance"`  // clearance level e.g. 1-5
	Clearances []string `json:"clearances"` // specific tags
}

type Policy struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	AllowedRoles []string `json:"allowed_roles"`
	MinClearance int      `json:"min_clearance"`
	Resource     string   `json:"resource"`
	Action       string   `json:"action"` // read, write, execute
}

type Engine struct {
	policies []Policy
}

func NewEngine() *Engine {
	// Seed with default tenant policies
	return &Engine{
		policies: []Policy{
			{
				ID:           "default-read",
				TenantID:     "*",
				AllowedRoles: []string{"admin", "analyst", "viewer"},
				MinClearance: 1,
				Resource:     "telemetry",
				Action:       "read",
			},
			{
				ID:           "default-ingest",
				TenantID:     "*",
				AllowedRoles: []string{"admin", "analyst"},
				MinClearance: 2,
				Resource:     "telemetry",
				Action:       "write",
			},
			{
				ID:           "default-notebook",
				TenantID:     "*",
				AllowedRoles: []string{"admin", "analyst"},
				MinClearance: 2,
				Resource:     "notebook",
				Action:       "execute",
			},
		},
	}
}

func (e *Engine) Evaluate(user UserAttributes, resource string, action string, targetTenant string) error {
	if user.TenantID == "" {
		return errors.New("abac denied: missing tenant context")
	}

	// Strict tenant isolation rule
	if user.TenantID != targetTenant && user.Role != "superadmin" {
		return errors.New("abac denied: cross-tenant access prohibited")
	}

	for _, pol := range e.policies {
		if (pol.TenantID == "*" || pol.TenantID == user.TenantID) &&
			(pol.Resource == "*" || pol.Resource == resource) &&
			(pol.Action == "*" || pol.Action == action) {

			roleOk := false
			for _, r := range pol.AllowedRoles {
				if strings.EqualFold(r, user.Role) {
					roleOk = true
					break
				}
			}

			if roleOk && user.Clearance >= pol.MinClearance {
				return nil
			}
		}
	}

	return errors.New("abac denied: insufficient attribute permissions")
}

func (e *Engine) AddPolicy(p Policy) {
	e.policies = append(e.policies, p)
}

func (e *Engine) GetPolicies() []Policy {
	return e.policies
}
