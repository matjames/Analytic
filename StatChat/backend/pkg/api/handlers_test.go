package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"statchat/pkg/model"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddlewareRejectsMissingTokenWhenEnabled(t *testing.T) {
	t.Setenv("STATCHAT_AUTH_REQUIRED", "true")
	t.Setenv("STATCHAT_JWT_SECRET", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	authMiddleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAuthMiddlewareAcceptsValidToken(t *testing.T) {
	t.Setenv("STATCHAT_AUTH_REQUIRED", "true")
	t.Setenv("STATCHAT_JWT_SECRET", "test-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	encoded, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+encoded)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	authMiddleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestInternalObjectConversationIdentityIsNarrowlyScoped(t *testing.T) {
	t.Setenv("STATGATE_INTERNAL_API_KEY", "internal-key")
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/conversations/object", nil)
	req.Header.Set("X-Internal-API-Key", "internal-key")
	req.Header.Set("X-StatGate-User-ID", "enterprise-workflow")
	req.Header.Set("X-Tenant-ID", "tenant-1")
	identity, _, ok := internalObjectConversationIdentity(req)
	if !ok || identity.ID != "enterprise-workflow" || identity.OrganizationID != "tenant-1" {
		t.Fatalf("expected trusted service identity, got %+v ok=%v", identity, ok)
	}
	req = httptest.NewRequest(http.MethodPost, "/v1/chat/messages", nil)
	req.Header.Set("X-Internal-API-Key", "internal-key")
	req.Header.Set("X-StatGate-User-ID", "enterprise-workflow")
	req.Header.Set("X-Tenant-ID", "tenant-1")
	if _, _, ok := internalObjectConversationIdentity(req); ok {
		t.Fatal("expected internal identity to be denied outside object-conversation creation")
	}
	if _, _, ok := internalServiceIdentity(req); !ok {
		t.Fatal("expected trusted service identity for message delivery")
	}
	req = httptest.NewRequest(http.MethodGet, "/v1/chat/messages", nil)
	req.Header.Set("X-Internal-API-Key", "internal-key")
	req.Header.Set("X-StatGate-User-ID", "enterprise-workflow")
	req.Header.Set("X-Tenant-ID", "tenant-1")
	if _, _, ok := internalServiceIdentity(req); ok {
		t.Fatal("expected service identity to be denied for non-message methods")
	}
}

func TestAuthMiddlewareAcceptsWebSocketQueryToken(t *testing.T) {
	t.Setenv("STATCHAT_AUTH_REQUIRED", "true")
	t.Setenv("STATCHAT_JWT_SECRET", "test-secret")
	// WebSocket tickets in the query string MUST be short-lived (<= 6 min).
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user-001", "exp": time.Now().Add(2 * time.Minute).Unix()})
	encoded, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ws?ticket="+encoded, nil)
	rr := httptest.NewRecorder()
	authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestAuthMiddlewareAcceptsWebSocketSubprotocolToken(t *testing.T) {
	t.Setenv("STATCHAT_AUTH_REQUIRED", "true")
	t.Setenv("STATCHAT_JWT_SECRET", "test-secret")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-001",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	encoded, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "Bearer."+encoded)
	rr := httptest.NewRecorder()
	authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := requestUserID(r); got != "user-001" {
			t.Fatalf("expected authenticated websocket user, got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestAuthMiddlewareRejectsLongLivedWebSocketQueryToken(t *testing.T) {
	t.Setenv("STATCHAT_AUTH_REQUIRED", "true")
	t.Setenv("STATCHAT_JWT_SECRET", "test-secret")
	// A long-lived JWT in the WebSocket query string must be rejected
	// (no long-lived credentials in URLs - SG-SEC-2026-08).
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user-001", "exp": time.Now().Add(6 * time.Hour).Unix()})
	encoded, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ws?ticket="+encoded, nil)
	rr := httptest.NewRecorder()
	authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d for long-lived ws query token, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestReadinessHandlerRejectsTrafficWhenDatabaseIsUnavailable(t *testing.T) {
	rr := httptest.NewRecorder()
	readinessHandler(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
}

func TestAuthMiddlewarePopulatesRequestUserIDFromTokenSubject(t *testing.T) {
	t.Setenv("STATCHAT_AUTH_REQUIRED", "true")
	t.Setenv("STATCHAT_JWT_SECRET", "test-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-456",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	encoded, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+encoded)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := requestUserID(r); got != "user-456" {
			t.Fatalf("expected request user id %q, got %q", "user-456", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	authMiddleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestHealthHandlerReportsReadinessFlag(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	healthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("expected valid json body, got %v", err)
	}
	if _, ok := body["ready"]; !ok {
		t.Fatalf("expected readiness field in health response, got %v", body)
	}
}

func TestAuthMiddlewareSkipsOptionsPreflight(t *testing.T) {
	t.Setenv("STATCHAT_AUTH_REQUIRED", "true")
	t.Setenv("STATCHAT_JWT_SECRET", "test-secret")

	req := httptest.NewRequest(http.MethodOptions, "/secure", nil)
	req.Header.Set("Origin", "http://localhost")
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	authMiddleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestBuildMessageFromPayloadUsesDefaultConversation(t *testing.T) {
	message, err := buildMessageFromPayload(map[string]interface{}{
		"sender": "Ada",
		"text":   "hello",
	}, model.User{ID: "user-001", Name: "Ada"})
	if err != nil {
		t.Fatalf("expected payload to build, got %v", err)
	}
	if message.ConversationID != "general" {
		t.Fatalf("expected default conversation, got %s", message.ConversationID)
	}
	if message.Status != "active" {
		t.Fatalf("expected active status, got %s", message.Status)
	}
}

func TestValidObjectRef(t *testing.T) {
	for _, value := range []string{
		"obj:pms:project:project-42",
		"obj:rms:research:study:2026-04",
	} {
		if !validObjectRef(value) {
			t.Fatalf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"", "project-42", "obj:pms::42", "obj:pms:project:"} {
		if validObjectRef(value) {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}
