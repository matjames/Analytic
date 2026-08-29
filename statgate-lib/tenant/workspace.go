package tenant

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// ContextWorkspaceID is the gin context key under which the selected
	// workspace is stored. It matches the key already used by PMS, RMS,
	// StatChat, StatOps and the Registry shell.
	ContextWorkspaceID = "workspace_id"

	workspaceHeader   = "X-Workspace-ID"
	maxWorkspaceIDLen = 128

	// DefaultEnterpriseAPIURL is used when STATGATE_ENTERPRISE_API_URL is unset.
	DefaultEnterpriseAPIURL = "http://localhost:8096/api"
)

// errNotMember signals Enterprise Core rejected the membership check.
var errNotMember = fmt.Errorf("not a workspace member")

// ValidWorkspaceID reports whether wsID is shape-valid: at most 128 characters
// of [A-Za-z0-9_-] with the first character a letter or underscore.
func ValidWorkspaceID(wsID string) bool {
	if wsID == "" || len(wsID) > maxWorkspaceIDLen {
		return false
	}
	for i, r := range wsID {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_', r == '-':
			// allowed anywhere
		case r >= '0' && r <= '9' && i > 0:
			// digits allowed, but not as the first character
		default:
			return false
		}
	}
	return true
}

// GinWorkspaceContext reads the optional X-Workspace-ID header, validates its
// shape, and stores it in the gin context under ContextWorkspaceID. An absent
// header simply means "no workspace selected" and never aborts the request.
func GinWorkspaceContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := strings.TrimSpace(c.GetHeader(workspaceHeader))
		if wsID != "" && !ValidWorkspaceID(wsID) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_workspace_context",
				"message": "X-Workspace-ID header is malformed",
			})
			return
		}
		c.Set(ContextWorkspaceID, wsID)
		c.Next()
	}
}

// WorkspaceIDFromGin returns the workspace selected on the request, or "".
func WorkspaceIDFromGin(c *gin.Context) string {
	if v, ok := c.Get(ContextWorkspaceID); ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// GinWorkspaceMembership returns middleware that verifies — when a workspace
// is selected AND the request carries an Authorization header — that the
// authenticated principal is a member of that workspace, by asking Enterprise
// Core (GET {base}/workspaces/{id} with the caller's Authorization forwarded).
//
// Responses on failure:
//   - 403 workspace_membership_required when Enterprise Core says 403/404
//   - 503 workspace_membership_unavailable when Enterprise Core is unreachable
//
// Requests without a workspace selection or without credentials pass through;
// route-level authentication remains responsible for rejecting those.
// enterpriseBaseURL falls back to STATGATE_ENTERPRISE_API_URL, then to
// DefaultEnterpriseAPIURL. A nil client uses a 2-second-timeout client.
func GinWorkspaceMembership(enterpriseBaseURL string, client *http.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := WorkspaceIDFromGin(c)
		authHeader := c.GetHeader("Authorization")
		if wsID == "" || authHeader == "" {
			c.Next()
			return
		}
		if err := verifyWorkspaceMembership(c.Request.Context(), enterpriseBaseURL, client, wsID, authHeader); err != nil {
			if err == errNotMember {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "workspace_membership_required",
					"message": "authenticated principal is not a member of the selected workspace",
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":   "workspace_membership_unavailable",
				"message": "workspace membership could not be verified",
			})
			return
		}
		c.Next()
	}
}

// VerifyWorkspaceMembership performs the Enterprise Core membership check and
// is exported for services that need to enforce membership inside handlers
// (for example on WebSocket upgrades or device pairing flows).
func VerifyWorkspaceMembership(ctx context.Context, enterpriseBaseURL string, client *http.Client, wsID, authHeader string) error {
	return verifyWorkspaceMembership(ctx, enterpriseBaseURL, client, wsID, authHeader)
}

func verifyWorkspaceMembership(ctx context.Context, enterpriseBaseURL string, client *http.Client, wsID, authHeader string) error {
	base := strings.TrimRight(strings.TrimSpace(enterpriseBaseURL), "/")
	if base == "" {
		base = strings.TrimRight(os.Getenv("STATGATE_ENTERPRISE_API_URL"), "/")
	}
	if base == "" {
		base = DefaultEnterpriseAPIURL
	}
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/workspaces/"+wsID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return errNotMember
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("enterprise core returned %d", resp.StatusCode)
	}
	return nil
}