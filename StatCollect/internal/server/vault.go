package server

import (
	"fmt"
	"os"

	vault "github.com/hashicorp/vault/api"
)

// LoadKeysFromVault reads api_keys and admin_keys from Vault at given path
func LoadKeysFromVault(addr, token, path string) ([]string, []string, error) {
	cfg := vault.DefaultConfig()
	cfg.Address = addr
	client, err := vault.NewClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	// support AppRole if token not provided
	if token != "" {
		client.SetToken(token)
	} else {
		// try AppRole via env vars if present
		roleID := os.Getenv("VAULT_APPROLE_ROLE_ID")
		secretID := os.Getenv("VAULT_APPROLE_SECRET_ID")
		if roleID != "" && secretID != "" {
			resp, err := client.Logical().Write("auth/approle/login", map[string]interface{}{"role_id": roleID, "secret_id": secretID})
			if err == nil && resp != nil && resp.Auth != nil {
				client.SetToken(resp.Auth.ClientToken)
			}
		}
	}
	// try reading via logical
	sec, err := client.Logical().Read(path)
	if err != nil || sec == nil {
		return nil, nil, fmt.Errorf("vault read failed: %v", err)
	}
	var apiKeys []string
	var adminKeys []string
	if v, ok := sec.Data["api_keys"]; ok {
		if arr, ok := v.([]interface{}); ok {
			for _, e := range arr {
				apiKeys = append(apiKeys, fmt.Sprintf("%v", e))
			}
		}
	}
	if v, ok := sec.Data["admin_keys"]; ok {
		if arr, ok := v.([]interface{}); ok {
			for _, e := range arr {
				adminKeys = append(adminKeys, fmt.Sprintf("%v", e))
			}
		}
	}
	return apiKeys, adminKeys, nil
}

// SaveKeysToVault writes api/admin keys to Vault path (simple write)
func SaveKeysToVault(addr, token, path string, apiKeys, adminKeys []string) error {
	cfg := vault.DefaultConfig()
	cfg.Address = addr
	client, err := vault.NewClient(cfg)
	if err != nil {
		return err
	}
	if token != "" {
		client.SetToken(token)
	} else {
		roleID := os.Getenv("VAULT_APPROLE_ROLE_ID")
		secretID := os.Getenv("VAULT_APPROLE_SECRET_ID")
		if roleID != "" && secretID != "" {
			resp, err := client.Logical().Write("auth/approle/login", map[string]interface{}{"role_id": roleID, "secret_id": secretID})
			if err == nil && resp != nil && resp.Auth != nil {
				client.SetToken(resp.Auth.ClientToken)
			}
		}
	}
	data := map[string]interface{}{
		"api_keys":   apiKeys,
		"admin_keys": adminKeys,
	}
	_, err = client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("vault write failed: %w", err)
	}
	return nil
}
