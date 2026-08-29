package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSessionMetadataHashesIPAndCapturesGeoHints(t *testing.T) {
	t.Setenv("SESSION_METADATA_SECRET", "test-session-secret")
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:4567"
	req.Header.Set("User-Agent", "StatGate Test Browser")
	req.Header.Set("CF-IPCountry", "ug")
	req.Header.Set("X-Geo-Region", "central")
	c.Request = req

	metadata := sessionMetadata(c)
	if metadata.UserAgent != "StatGate Test Browser" {
		t.Fatalf("unexpected user agent %q", metadata.UserAgent)
	}
	if metadata.IPHash == "" || metadata.IPHash == "203.0.113.10" {
		t.Fatalf("expected privacy-preserving IP hash, got %q", metadata.IPHash)
	}
	if metadata.GeoCountry != "UG" {
		t.Fatalf("expected country UG, got %q", metadata.GeoCountry)
	}
	if metadata.GeoRegion != "CENTRAL" {
		t.Fatalf("expected region CENTRAL, got %q", metadata.GeoRegion)
	}
}

func TestGeoHeaderIgnoresUnknownValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("CF-IPCountry", "XX")
	req.Header.Set("X-Geo-Country", "ke")
	c.Request = req

	if got := geoHeader(c, "CF-IPCountry", "X-Geo-Country"); got != "KE" {
		t.Fatalf("expected fallback country KE, got %q", got)
	}
}
