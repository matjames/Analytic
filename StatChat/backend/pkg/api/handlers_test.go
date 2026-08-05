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
