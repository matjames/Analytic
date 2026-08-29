package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"statchat/pkg/model"
	"statchat/pkg/store"
)

type registryUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	Organisation string `json:"organisation"`
	DistrictID   string `json:"district_id"`
}

func registryAPIURL() string {
	if endpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("STATGATE_REGISTRY_API_URL")), "/"); endpoint != "" {
		return endpoint
	}
	return "http://localhost:9090/api"
}

// registryDirectory is the canonical staff directory for StatChat. The
// Registry authorizes the request with either the shared SSO token or StatChat's
// internal credential. Internal requests carry the authenticated tenant scope.
func registryDirectory(ctx context.Context, authorization, tenantID string) ([]model.User, error) {
	endpoint := registryAPIURL() + "/users"
	internalKey := strings.TrimSpace(os.Getenv("STATGATE_INTERNAL_API_KEY"))
	if internalKey != "" {
		if strings.TrimSpace(tenantID) == "" {
			return nil, fmt.Errorf("registry directory requires tenant scope")
		}
		endpoint = registryAPIURL() + "/internal/users"
	} else if strings.TrimSpace(authorization) == "" {
		return nil, fmt.Errorf("registry directory requires a shared bearer token or internal service credential")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	if internalKey != "" {
		req.Header.Set("X-StatGate-Internal-Key", internalKey)
		req.Header.Set("X-Tenant-ID", strings.TrimSpace(tenantID))
	}

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry directory request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry directory returned %s", response.Status)
	}

	var records []registryUser
	if err := json.NewDecoder(response.Body).Decode(&records); err != nil {
		return nil, fmt.Errorf("decode registry directory: %w", err)
	}

	users := make([]model.User, 0, len(records))
	for _, record := range records {
		name := strings.TrimSpace(strings.Join([]string{record.FirstName, record.LastName}, " "))
		if name == "" {
			name = record.Username
		}
		organizationID := record.Organisation
		if organizationID == "" && record.DistrictID != "" {
			organizationID = "district:" + record.DistrictID
		}
		roles := []string{}
		if record.Role != "" {
			roles = append(roles, record.Role)
		}
		user := model.User{
			ID:             fmt.Sprint(record.ID),
			Name:           name,
			Email:          record.Email,
			OrganizationID: organizationID,
			Roles:          roles,
			Presence:       "offline",
		}
		// Cache only non-sensitive collaboration metadata. Registry remains the
		// account authority and never receives chat data.
		if err := store.UpsertTrustedUser(user); err != nil {
			return nil, fmt.Errorf("cache registry user %s: %w", user.ID, err)
		}
		users = append(users, user)
	}
	return users, nil
}
