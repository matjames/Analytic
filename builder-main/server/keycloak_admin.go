// Package main: Keycloak Admin API client (server-side only).
// Uses KEYCLOAK_ADMIN_ID and KEYCLOAK_ADMIN_SECRET to obtain tokens
// and proxy Admin API requests. Never expose admin secret to the frontend.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	adminToken           string
	adminTokenExp        time.Time
	adminTokenMux        sync.RWMutex
	adminTokenTTL        = 50 * time.Second // refresh before expiry (Keycloak often uses 60s)
	keycloakAdminGetFunc = keycloakAdminGet
	getClientUUIDFunc    = getClientUUID
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// getAdminToken returns a valid Keycloak admin access token (client credentials).
// Uses KEYCLOAK_URL, KEYCLOAK_REALM, KEYCLOAK_ADMIN_ID, KEYCLOAK_ADMIN_SECRET.
func getAdminToken() (string, error) {
	if keycloakURL == "" || keycloakRealm == "" || keycloakAdminID == "" || keycloakAdminSecret == "" {
		return "", fmt.Errorf("keycloak admin not configured: set KEYCLOAK_URL, KEYCLOAK_REALM, KEYCLOAK_ADMIN_ID, KEYCLOAK_ADMIN_SECRET")
	}

	adminTokenMux.RLock()
	if adminToken != "" && time.Now().Before(adminTokenExp) {
		tok := adminToken
		adminTokenMux.RUnlock()
		return tok, nil
	}
	adminTokenMux.RUnlock()

	adminTokenMux.Lock()
	defer adminTokenMux.Unlock()

	// Double-check after acquiring write lock
	if adminToken != "" && time.Now().Before(adminTokenExp) {
		return adminToken, nil
	}

	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", keycloakURL, keycloakRealm)
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", keycloakAdminID)
	form.Set("client_secret", keycloakAdminSecret)

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("keycloak token failed status=%d body=%s", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("empty access_token in response")
	}

	expiresIn := tr.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 60
	}
	adminToken = tr.AccessToken
	adminTokenExp = time.Now().Add(time.Duration(expiresIn) * time.Second)

	return adminToken, nil
}

// keycloakAdminRequest performs an authenticated request to Keycloak Admin API.
// path is the path under /admin/realms/{realm}, e.g. "/users" or "/users/abc-123".
func keycloakAdminRequest(method, path string, body []byte) (*http.Response, error) {
	token, err := getAdminToken()
	if err != nil {
		return nil, err
	}

	base := fmt.Sprintf("%s/admin/realms/%s", keycloakURL, keycloakRealm)
	path = strings.TrimPrefix(path, "/")
	reqURL := base + "/" + path

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("admin request: %w", err)
	}
	return resp, nil
}

// keycloakAdminGet returns response body and caller must close body (or drain and close).
func keycloakAdminGet(path string) (*http.Response, error) {
	return keycloakAdminRequest(http.MethodGet, path, nil)
}

// keycloakAdminPost sends JSON body and returns response.
func keycloakAdminPost(path string, body interface{}) (*http.Response, error) {
	var buf []byte
	if body != nil {
		var err error
		buf, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	return keycloakAdminRequest(http.MethodPost, path, buf)
}

// keycloakAdminPut sends JSON body and returns response.
func keycloakAdminPut(path string, body interface{}) (*http.Response, error) {
	var buf []byte
	if body != nil {
		var err error
		buf, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	return keycloakAdminRequest(http.MethodPut, path, buf)
}

// keycloakAdminDelete performs DELETE and returns response.
func keycloakAdminDelete(path string) (*http.Response, error) {
	return keycloakAdminRequest(http.MethodDelete, path, nil)
}

// getAppClientID returns the application client ID (KEYCLOAK_CLIENT_ID or dwhlanding).
func getAppClientID() string {
	if keycloakRolesClient != "" {
		return keycloakRolesClient
	}
	if keycloakClientID != "" {
		return keycloakClientID
	}
	return "dwhlanding"
}

func decodeClientUUIDFromBody(bodyBytes []byte, wantedClientID string) (string, error) {
	var clients []map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &clients); err != nil {
		return "", err
	}
	if len(clients) == 0 {
		return "", nil
	}

	trimmedWanted := strings.TrimSpace(wantedClientID)

	// Prefer an exact match on either Keycloak's logical clientId or internal id.
	for _, client := range clients {
		clientID, _ := client["clientId"].(string)
		internalID, _ := client["id"].(string)
		if strings.TrimSpace(clientID) == trimmedWanted || strings.TrimSpace(internalID) == trimmedWanted {
			if internalID != "" {
				return internalID, nil
			}
		}
	}

	// Fall back to the first result when the exact-filter endpoint already narrowed it down.
	if id, _ := clients[0]["id"].(string); id != "" {
		return id, nil
	}
	return "", nil
}

// getClientUUID resolves a client ID (e.g. "dwhlanding") to Keycloak's internal UUID.
func getClientUUID(clientID string) (string, error) {
	clientID = strings.TrimSpace(clientID)
	resp, err := keycloakAdminGet("clients?clientId=" + url.QueryEscape(clientID))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read body once so we can both log and decode it
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("[Keycloak] getClientUUID: failed to read response body for clientId=%s: %v", clientID, readErr)
		return "", fmt.Errorf("clients list returned %d (read body error)", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Keycloak] getClientUUID: non-200 status for clientId=%s status=%d body=%s", clientID, resp.StatusCode, string(bodyBytes))
		return "", fmt.Errorf("clients list returned %d", resp.StatusCode)
	}

	id, err := decodeClientUUIDFromBody(bodyBytes, clientID)
	if err != nil {
		log.Printf("[Keycloak] getClientUUID: failed to decode JSON for clientId=%s body=%s error=%v", clientID, string(bodyBytes), err)
		return "", err
	}
	if id != "" {
		return id, nil
	}

	// Some Keycloak versions/plugins are more reliable with search than exact clientId filtering.
	searchResp, err := keycloakAdminGet("clients?search=" + url.QueryEscape(clientID))
	if err != nil {
		log.Printf("[Keycloak] getClientUUID: search fallback failed for clientId=%s error=%v", clientID, err)
		return "", fmt.Errorf("client %q not found", clientID)
	}
	defer searchResp.Body.Close()

	searchBody, readErr := io.ReadAll(searchResp.Body)
	if readErr != nil {
		log.Printf("[Keycloak] getClientUUID: failed to read search response body for clientId=%s: %v", clientID, readErr)
		return "", fmt.Errorf("client %q not found", clientID)
	}
	if searchResp.StatusCode != http.StatusOK {
		log.Printf("[Keycloak] getClientUUID: search fallback non-200 for clientId=%s status=%d body=%s", clientID, searchResp.StatusCode, string(searchBody))
		return "", fmt.Errorf("client %q not found", clientID)
	}

	id, err = decodeClientUUIDFromBody(searchBody, clientID)
	if err != nil {
		log.Printf("[Keycloak] getClientUUID: failed to decode search response for clientId=%s body=%s error=%v", clientID, string(searchBody), err)
		return "", err
	}
	if id == "" {
		log.Printf("[Keycloak] getClientUUID: no clients returned for clientId=%s exactBody=%s searchBody=%s", clientID, string(bodyBytes), string(searchBody))
		return "", fmt.Errorf("client %q not found", clientID)
	}
	return id, nil
}
