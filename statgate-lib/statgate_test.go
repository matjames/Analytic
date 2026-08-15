package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/errors"
	"github.com/matjames/statgate-lib/events"
	"github.com/matjames/statgate-lib/health"
	"github.com/matjames/statgate-lib/pagination"
	"github.com/matjames/statgate-lib/permissions"
	"github.com/matjames/statgate-lib/tenant"
)

func TestAuthValidation(t *testing.T) {
	secret := "test-statgate-secret-key-32-chars-long-min!"
	validator, err := auth.NewValidator(secret, "statgate-registry", "statgate")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":    "usr-12345",
		"email":     "admin@statgate.gov",
		"tenant_id": "uganda-national",
		"role":      "admin",
		"iss":       "statgate-registry",
		"aud":       "statgate",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	uCtx, err := validator.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("Failed to validate valid token: %v", err)
	}
	if uCtx.UserID != "usr-12345" || uCtx.TenantID != "uganda-national" || uCtx.Role != "admin" {
		t.Errorf("Unexpected user context: %+v", uCtx)
	}

	// Test invalid role rejection
	badRoleToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":    "usr-hacker",
		"tenant_id": "uganda-national",
		"role":      "super_hacker_bypass",
		"iss":       "statgate-registry",
		"aud":       "statgate",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	badStr, _ := badRoleToken.SignedString([]byte(secret))
	if _, err := validator.ValidateToken(badStr); err == nil {
		t.Errorf("Expected invalid role to fail validation")
	}
}

func TestTenantValidation(t *testing.T) {
	if err := tenant.ValidateTenantSlug("uganda-national"); err != nil {
		t.Errorf("Expected valid slug, got %v", err)
	}
	if err := tenant.ValidateTenantSlug("bad/slug/injection"); err == nil {
		t.Errorf("Expected invalid characters to fail validation")
	}
}

func TestPermissions(t *testing.T) {
	if !permissions.HasPermission("admin", permissions.PermWrite) {
		t.Errorf("Admin should have write permission")
	}
	if !permissions.HasPermission("editor", permissions.PermWrite) {
		t.Errorf("Editor should have write permission")
	}
	if permissions.HasPermission("viewer", permissions.PermWrite) {
		t.Errorf("Viewer should NOT have write permission")
	}
}

func TestEventBusDeduplication(t *testing.T) {
	bus, err := events.NewEventBus(events.Config{Source: "test"})
	if err != nil {
		t.Fatalf("Failed to create event bus: %v", err)
	}

	evtID := "evt-unique-001"
	if bus.IsDuplicate(evtID) {
		t.Errorf("First occurrence should not be duplicate")
	}
	if !bus.IsDuplicate(evtID) {
		t.Errorf("Second occurrence must be detected as duplicate")
	}
}

func TestPaginationParsing(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/items?page=2&page_size=10&sort=name&order=ASC", nil)
	params := pagination.Parse(req, 20, 100)

	if params.Page != 2 || params.PageSize != 10 || params.SortBy != "name" || params.SortDir != "ASC" {
		t.Errorf("Unexpected pagination params: %+v", params)
	}
	if params.Offset() != 10 {
		t.Errorf("Expected offset 10, got %d", params.Offset())
	}
}

func TestErrorResponse(t *testing.T) {
	appErr := errors.NotFound("Resource not found")
	if appErr.HTTPStatus != http.StatusNotFound || appErr.Code != errors.CodeNotFound {
		t.Errorf("Unexpected error properties: %+v", appErr)
	}
}

func TestHealthChecker(t *testing.T) {
	checker := health.NewChecker("test-service", nil)
	h := checker.CheckHealth(context.Background())
	if h.Status != health.StatusHealthy {
		t.Errorf("Expected healthy status, got %s", h.Status)
	}
}
