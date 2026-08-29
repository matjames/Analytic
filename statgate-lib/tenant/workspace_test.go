package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newWorkspaceRouter(t *testing.T, enterpriseURL string, client *http.Client, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinWorkspaceContext())
	r.Use(GinWorkspaceMembership(enterpriseURL, client))
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"workspace": WorkspaceIDFromGin(c)})
	})
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestWorkspaceContextAbsent(t *testing.T) {
	w := newWorkspaceRouter(t, "", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != `{"workspace":""}` {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestWorkspaceContextInvalidShape(t *testing.T) {
	cases := map[string]string{
		"too long":       "-" + string(make([]byte, 200)),
		"slash":          "team/one",
		"leading digit":  "1team",
		"space":          "team one",
		"newline":        "team\none",
	}
	for name, ws := range cases {
		w := newWorkspaceRouter(t, "", nil, map[string]string{"X-Workspace-ID": ws})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", name, w.Code)
		}
	}
}

func TestWorkspaceContextValidShapes(t *testing.T) {
	for _, ws := range []string{"ws1", "tenant-alpha", "Team_One", "a", "workspace-42"} {
		w := newWorkspaceRouter(t, "", nil, map[string]string{"X-Workspace-ID": ws})
		if w.Code != http.StatusOK {
			t.Fatalf("%q: expected 200, got %d", ws, w.Code)
		}
	}
}

func TestWorkspaceMembershipMember(t *testing.T) {
	var gotAuth string
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := newWorkspaceRouter(t, srv.URL, nil, map[string]string{
		"X-Workspace-ID": "ws1",
		"Authorization":  "Bearer tok",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("authorization not forwarded: %q", gotAuth)
	}
	if gotPath != "/workspaces/ws1" {
		t.Fatalf("unexpected enterprise path: %s", gotPath)
	}
	if w.Body.String() != `{"workspace":"ws1"}` {
		t.Fatalf("workspace not in context: %s", w.Body.String())
	}
}

func TestWorkspaceMembershipNonMember(t *testing.T) {
	for _, code := range []int{http.StatusForbidden, http.StatusNotFound} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
		}))
		w := newWorkspaceRouter(t, srv.URL, nil, map[string]string{
			"X-Workspace-ID": "ws1",
			"Authorization":  "Bearer tok",
		})
		if w.Code != http.StatusForbidden {
			t.Fatalf("enterprise %d: expected 403, got %d", code, w.Code)
		}
		srv.Close()
	}
}

func TestWorkspaceMembershipEnterpriseDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	w := newWorkspaceRouter(t, srv.URL, nil, map[string]string{
		"X-Workspace-ID": "ws1",
		"Authorization":  "Bearer tok",
	})
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
	srv.Close()
}

func TestWorkspaceMembershipSkippedWithoutAuth(t *testing.T) {
	w := newWorkspaceRouter(t, "http://127.0.0.1:1", nil, map[string]string{"X-Workspace-ID": "ws1"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected pass-through without Authorization, got %d", w.Code)
	}
}

func TestWorkspaceMembershipSkippedWithoutWorkspace(t *testing.T) {
	w := newWorkspaceRouter(t, "http://127.0.0.1:1", nil, map[string]string{"Authorization": "Bearer tok"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected pass-through without workspace, got %d", w.Code)
	}
}

func TestVerifyWorkspaceMembershipTimeout(t *testing.T) {
	hang := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer hang.Close()
	client := &http.Client{Timeout: 50 * time.Millisecond}
	if err := VerifyWorkspaceMembership(context.Background(), hang.URL, client, "ws1", "Bearer tok"); err == nil {
		t.Fatal("expected timeout error")
	}
}

// ── net/http middleware tests ──

func httpProbe(t *testing.T, enterpriseURL string, client *http.Client, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	h := WorkspaceContext(WorkspaceMembership(enterpriseURL, client)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"workspace":"` + WorkspaceIDFromRequest(r) + `"}`))
	})))
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestHTTPWorkspaceContextAndMembership(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()

	// member: passes, workspace in context
	w := httpProbe(t, srv.URL, nil, map[string]string{"X-Workspace-ID": "ws1", "Authorization": "Bearer tok"})
	if w.Code != http.StatusOK || w.Body.String() != `{"workspace":"ws1"}` {
		t.Fatalf("member: got %d %s", w.Code, w.Body.String())
	}

	// non-member: 403
	deny := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer deny.Close()
	w = httpProbe(t, deny.URL, nil, map[string]string{"X-Workspace-ID": "ws1", "Authorization": "Bearer tok"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-member: expected 403, got %d", w.Code)
	}

	// invalid shape: 400
	w = httpProbe(t, "", nil, map[string]string{"X-Workspace-ID": "bad/slash"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid: expected 400, got %d", w.Code)
	}

	// no workspace: pass-through
	w = httpProbe(t, srv.URL, nil, nil)
	if w.Code != http.StatusOK || w.Body.String() != `{"workspace":""}` {
		t.Fatalf("pass-through: got %d %s", w.Code, w.Body.String())
	}
}

func TestValidWorkspaceID(t *testing.T) {
	if !ValidWorkspaceID("tenant-alpha_1") {
		t.Fatal("expected valid")
	}
	if ValidWorkspaceID("") {
		t.Fatal("empty must be invalid")
	}
	long := ""
	for i := 0; i < 129; i++ {
		long += "a"
	}
	if ValidWorkspaceID(long) {
		t.Fatal("expected too-long invalid")
	}
}