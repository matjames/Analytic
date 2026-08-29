package utils

import (
	"testing"
	"time"

	"go-backend/models"
)

func TestRefreshTokenRoundTrip(t *testing.T) {
	t.Setenv("STATGATE_REGISTRY_JWT_SECRET", "test-refresh-secret-0123456789")

	user := models.User{ID: 42}
	refreshToken, err := SignRefreshToken(user)
	if err != nil {
		t.Fatalf("SignRefreshToken() error = %v", err)
	}

	got, err := RefreshSubject(refreshToken)
	if err != nil {
		t.Fatalf("RefreshSubject() error = %v", err)
	}
	if got != user.ID {
		t.Fatalf("RefreshSubject() = %d, want %d", got, user.ID)
	}
}

func TestVerifyTOTPRFCVector(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if !VerifyTOTP(secret, "287082", time.Unix(59, 0)) {
		t.Fatal("VerifyTOTP() rejected the RFC 6238 SHA-1 vector")
	}
}

func TestRefreshSubjectRejectsAccessToken(t *testing.T) {
	t.Setenv("STATGATE_REGISTRY_JWT_SECRET", "test-refresh-secret-0123456789")

	accessToken, err := SignUserToken(models.User{ID: 42})
	if err != nil {
		t.Fatalf("SignUserToken() error = %v", err)
	}
	if _, err := RefreshSubject(accessToken); err == nil {
		t.Fatal("RefreshSubject() accepted an access token")
	}
}

func TestRefreshSubjectRequiresConfiguredSecret(t *testing.T) {
	t.Setenv("STATGATE_REGISTRY_JWT_SECRET", "")
	t.Setenv("JWT_SECRET", "")

	if _, err := RefreshSubject("not-a-token"); err == nil {
		t.Fatal("RefreshSubject() succeeded without a signing secret")
	}
}
