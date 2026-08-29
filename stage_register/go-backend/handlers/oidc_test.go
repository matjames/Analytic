package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestOIDCLoginFailsClosedWhenUnconfigured(t *testing.T) {
	t.Setenv("OIDC_ISSUER_URL", "")
	t.Setenv("OIDC_CLIENT_ID", "")
	t.Setenv("OIDC_CLIENT_SECRET", "")
	t.Setenv("OIDC_REDIRECT_URL", "")

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)

	OIDCLogin(c)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "oidc_not_configured") {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func TestOIDCLoginDiscoversProviderAndRedirects(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			t.Fatalf("unexpected provider path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(oidcDiscovery{
			AuthorizationEndpoint: "https://idp.example.test/authorize",
			TokenEndpoint:         "https://idp.example.test/token",
			UserInfoEndpoint:      "https://idp.example.test/userinfo",
		})
	}))
	defer provider.Close()

	t.Setenv("OIDC_ISSUER_URL", provider.URL)
	t.Setenv("OIDC_CLIENT_ID", "client-123")
	t.Setenv("OIDC_CLIENT_SECRET", "secret-123")
	t.Setenv("OIDC_REDIRECT_URL", "https://registry.example.test/api/auth/oidc/callback")
	t.Setenv("OIDC_SCOPES", "openid email")
	t.Setenv("OIDC_STATE_SECRET", "state-secret-123")

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)

	OIDCLogin(c)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	location := recorder.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if parsed.Host != "idp.example.test" || parsed.Path != "/authorize" {
		t.Fatalf("unexpected redirect location %s", location)
	}
	query := parsed.Query()
	if query.Get("client_id") != "client-123" || query.Get("redirect_uri") != "https://registry.example.test/api/auth/oidc/callback" || query.Get("response_type") != "code" || query.Get("scope") != "openid email" {
		t.Fatalf("unexpected redirect query: %s", parsed.RawQuery)
	}
	if query.Get("state") == "" || query.Get("nonce") == "" {
		t.Fatalf("state and nonce are required in redirect query: %s", parsed.RawQuery)
	}
	if cookie := recorder.Result().Cookies(); len(cookie) == 0 || cookie[0].Name != oidcStateCookie || !cookie[0].HttpOnly {
		t.Fatalf("expected http-only OIDC state cookie, got %#v", cookie)
	}
}

func TestOIDCStateRoundTripRejectsTampering(t *testing.T) {
	t.Setenv("OIDC_STATE_SECRET", "state-secret-123")
	encoded, err := encodeOIDCState(oidcStatePayload{State: "state-a", Nonce: "nonce-a", Iat: time.Now().Unix()})
	if err != nil {
		t.Fatalf("encode state: %v", err)
	}
	decoded, err := decodeOIDCState(encoded)
	if err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if decoded.State != "state-a" || decoded.Nonce != "nonce-a" {
		t.Fatalf("unexpected decoded state: %#v", decoded)
	}
	if _, err := decodeOIDCState(encoded + "tampered"); err == nil {
		t.Fatalf("expected tampered state to fail")
	}
}

func TestOIDCStateRejectsFarFutureIssueTime(t *testing.T) {
	t.Setenv("OIDC_STATE_SECRET", "state-secret-123")
	encoded, err := encodeOIDCState(oidcStatePayload{State: "state-a", Nonce: "nonce-a", Iat: time.Now().Add(5 * time.Minute).Unix()})
	if err != nil {
		t.Fatalf("encode state: %v", err)
	}
	if _, err := decodeOIDCState(encoded); err == nil {
		t.Fatalf("expected future state issue time to fail")
	}
}

func TestMappedOIDCRoleAllowsOnlyConfiguredRoles(t *testing.T) {
	cfg := oidcConfig{DefaultRole: "viewer", AllowedRoles: map[string]bool{"analyst": true}}
	if got := mappedOIDCRole(cfg, "analyst"); got != "analyst" {
		t.Fatalf("expected analyst, got %q", got)
	}
	if got := mappedOIDCRole(cfg, "platform_admin"); got != "viewer" {
		t.Fatalf("expected disallowed privileged role to fall back to viewer, got %q", got)
	}
}

func TestExchangeOIDCCodePostsAuthorizationCodePayload(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		expected := map[string]string{
			"grant_type":    "authorization_code",
			"code":          "code-123",
			"redirect_uri":  "https://registry.example.test/callback",
			"client_id":     "client-123",
			"client_secret": "secret-123",
		}
		for key, want := range expected {
			if got := r.Form.Get(key); got != want {
				t.Fatalf("form %s: expected %q, got %q", key, want, got)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "access-123", "token_type": "Bearer"})
	}))
	defer provider.Close()

	token, err := exchangeOIDCCode(
		oidcConfig{ClientID: "client-123", ClientSecret: "secret-123", RedirectURL: "https://registry.example.test/callback"},
		oidcDiscovery{TokenEndpoint: provider.URL},
		"code-123",
	)
	if err != nil {
		t.Fatalf("exchange code: %v", err)
	}
	if token != "access-123" {
		t.Fatalf("expected access token, got %q", token)
	}
}

func TestFetchOIDCUserInfoUsesBearerToken(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer access-123" {
			t.Fatalf("expected bearer token, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(oidcUserInfo{
			Subject:           "sub-123",
			Email:             "person@example.test",
			PreferredUsername: "person",
			Role:              "analyst",
			TenantID:          "tenant-a",
		})
	}))
	defer provider.Close()

	info, err := fetchOIDCUserInfo(oidcDiscovery{UserInfoEndpoint: provider.URL}, "access-123")
	if err != nil {
		t.Fatalf("fetch userinfo: %v", err)
	}
	if info.Email != "person@example.test" || info.TenantID != "tenant-a" || info.Role != "analyst" {
		t.Fatalf("unexpected userinfo: %#v", info)
	}
}
