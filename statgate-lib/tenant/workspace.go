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

// ── net/http / gorilla-mux middleware variants ─────────────────────────────

type workspaceContextKey string

// ContextWorkspaceIDKey is the request-context key used by the net/http
// middleware variants below.
const ContextWorkspaceIDKey workspaceContextKey = "statgate_workspace_id"

func writeInvalidWorkspace(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(`{"error":"invalid_workspace_context","message":"X-Workspace-ID header is malformed"}`))
}

// WorkspaceContext is the net/http (gorilla/mux compatible) equivalent of
// GinWorkspaceContext: it validates the optional X-Workspace-ID header and
// stores it in the request context. Abort with 400 on a malformed header.
func WorkspaceContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsID := strings.TrimSpace(r.Header.Get(workspaceHeader))
		if wsID != "" && !ValidWorkspaceID(wsID) {
			writeInvalidWorkspace(w)
			return
		}
		ctx := context.WithValue(r.Context(), ContextWorkspaceIDKey, wsID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WorkspaceIDFromRequest returns the workspace selected on the request, or "".
func WorkspaceIDFromRequest(r *http.Request) string {
	if v, ok := r.Context().Value(ContextWorkspaceIDKey).(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// WorkspaceMembership is the net/http equivalent of GinWorkspaceMembership.
// When a workspace is selected AND the request carries an Authorization
// header, membership is verified against Enterprise Core; otherwise the
// request passes through untouched.
func WorkspaceMembership(enterpriseBaseURL string, client *http.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wsID := WorkspaceIDFromRequest(r)
			authHeader := r.Header.Get("Authorization")
			if wsID == "" || authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}
			if err := verifyWorkspaceMembership(r.Context(), enterpriseBaseURL, client, wsID, authHeader); err != nil {
				if err == errNotMember {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"error":"workspace_membership_required","message":"authenticated principal is not a member of the selected workspace"}`))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error":"workspace_membership_unavailable","message":"workspace membership could not be verified"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
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