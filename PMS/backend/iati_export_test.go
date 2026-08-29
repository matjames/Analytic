package main

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIATIExportEmptyDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/iati/activities.xml", dbExportIATIActivities)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/iati/activities.xml", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/xml; charset=utf-8" {
		t.Errorf("Expected Content-Type application/xml, got %s", contentType)
	}

	var parsed IATIActivities
	err := xml.Unmarshal(w.Body.Bytes(), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse IATI XML: %v", err)
	}

	if parsed.Version != "2.03" {
		t.Errorf("Expected IATI version 2.03, got %s", parsed.Version)
	}
}
