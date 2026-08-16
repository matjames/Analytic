package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// ═══════════════════════════════════════════════════════════════════
// PHASE 11 — MOBILE ODK COLLECT INTEGRATION & AUTO-CONFIG
// ═══════════════════════════════════════════════════════════════════

type ODKProjectConfig struct {
	ProjectName       string `json:"general.project_name"`
	ServerURL         string `json:"general.server_url"`
	FormListPath      string `json:"general.formlist_url"`
	SubmissionPath    string `json:"general.submission_url"`
	AutoSend          string `json:"general.autosend"`
	AutoDelete        bool   `json:"general.delete_after_send"`
	Navigation        string `json:"general.navigation"`
	HighResolution    bool   `json:"general.high_resolution"`
	AdminPasswordHash string `json:"admin.admin_pw,omitempty"`
}

func handleODKProjectSettings(c *gin.Context) {
	serverHost := strings.TrimSpace(os.Getenv("STATGATE_PUBLIC_URL"))
	if serverHost == "" {
		serverHost = "http://localhost:8080"
	}

	projectName := c.DefaultQuery("project_name", "StatGate Official Statistics")
	username := c.DefaultQuery("username", "enumerator")

	config := ODKProjectConfig{
		ProjectName:    fmt.Sprintf("%s (%s)", projectName, username),
		ServerURL:      serverHost,
		FormListPath:   serverHost + "/formList",
		SubmissionPath: serverHost + "/submission",
		AutoSend:       "wifi_and_cellular",
		AutoDelete:     false,
		Navigation:     "swipe_and_buttons",
		HighResolution: true,
	}

	c.JSON(http.StatusOK, gin.H{
		"settings": config,
		"qr_payload": base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`{"general":{"server_url":"%s","username":"%s"}}`, serverHost, username))),
		"instructions": "Scan this QR configuration in ODK Collect / collect-master to auto-provision server endpoints.",
		"status": "ready",
	})
}
