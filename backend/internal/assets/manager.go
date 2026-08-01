package assets

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Asset struct {
	ID                string                 `json:"id"`
	AssetType         string                 `json:"asset_type"` // "notebook", "dashboard", "widget", "metric"
	ContentDefinition map[string]interface{} `json:"content_definition"`
	OwnerID           string                 `json:"owner_id"` // Tenant ID
	VersionTag        string                 `json:"version_tag"`
	LastModified      string                 `json:"last_modified"`
}

type AssetHistory struct {
	HistoryID         int                    `json:"history_id"`
	AssetID           string                 `json:"asset_id"`
	AssetType         string                 `json:"asset_type"`
	ContentDefinition map[string]interface{} `json:"content_definition"`
	OwnerID           string                 `json:"owner_id"`
	VersionTag        string                 `json:"version_tag"`
	CreatedAt         string                 `json:"created_at"`
}

type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) (*Manager, error) {
	m := &Manager{db: db}
	if err := m.initSchema(); err != nil {
		// Log warning if DB connection is unavailable during initial boot
		fmt.Printf("[AssetManager] Schema init note: %v\n", err)
	}
	return m, nil
}

func (m *Manager) initSchema() error {
	if m.db == nil {
		return nil
	}

	query := `
	CREATE TABLE IF NOT EXISTS analytical_assets (
		id VARCHAR(128) PRIMARY KEY,
		asset_type VARCHAR(64) NOT NULL,
		content_definition JSONB NOT NULL,
		owner_id VARCHAR(128) NOT NULL,
		version_tag VARCHAR(32) NOT NULL DEFAULT '1.0.0',
		last_modified TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS asset_history (
		history_id SERIAL PRIMARY KEY,
		asset_id VARCHAR(128) NOT NULL,
		asset_type VARCHAR(64) NOT NULL,
		content_definition JSONB NOT NULL,
		owner_id VARCHAR(128) NOT NULL,
		version_tag VARCHAR(32) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *Manager) SaveAsset(asset Asset) error {
	if m.db == nil {
		return fmt.Errorf("database unavailable")
	}

	contentBytes, err := json.Marshal(asset.ContentDefinition)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if asset.VersionTag == "" {
		asset.VersionTag = "1.0.0"
	}

	// 1. Upsert into analytical_assets
	upsertQuery := `
	INSERT INTO analytical_assets (id, asset_type, content_definition, owner_id, version_tag, last_modified)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT (id) DO UPDATE SET
		asset_type = EXCLUDED.asset_type,
		content_definition = EXCLUDED.content_definition,
		owner_id = EXCLUDED.owner_id,
		version_tag = EXCLUDED.version_tag,
		last_modified = EXCLUDED.last_modified;
	`
	_, err = m.db.Exec(upsertQuery, asset.ID, asset.AssetType, string(contentBytes), asset.OwnerID, asset.VersionTag, now)
	if err != nil {
		return err
	}

	// 2. Record immutable audit snapshot in asset_history
	historyQuery := `
	INSERT INTO asset_history (asset_id, asset_type, content_definition, owner_id, version_tag, created_at)
	VALUES ($1, $2, $3, $4, $5, $6);
	`
	_, err = m.db.Exec(historyQuery, asset.ID, asset.AssetType, string(contentBytes), asset.OwnerID, asset.VersionTag, now)
	return err
}

func (m *Manager) GetAsset(id string) (*Asset, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	query := `SELECT id, asset_type, content_definition, owner_id, version_tag, last_modified FROM analytical_assets WHERE id = $1;`
	row := m.db.QueryRow(query, id)

	var a Asset
	var contentRaw string
	var lastMod time.Time

	if err := row.Scan(&a.ID, &a.AssetType, &contentRaw, &a.OwnerID, &a.VersionTag, &lastMod); err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(contentRaw), &a.ContentDefinition)
	a.LastModified = lastMod.Format(time.RFC3339)
	return &a, nil
}

func (m *Manager) ListAssets(ownerID string, assetType string) ([]Asset, error) {
	if m.db == nil {
		return []Asset{}, nil
	}

	query := `SELECT id, asset_type, content_definition, owner_id, version_tag, last_modified FROM analytical_assets WHERE owner_id = $1`
	args := []interface{}{ownerID}

	if assetType != "" {
		query += ` AND asset_type = $2`
		args = append(args, assetType)
	}
	query += ` ORDER BY last_modified DESC;`

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return []Asset{}, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		var contentRaw string
		var lastMod time.Time
		if err := rows.Scan(&a.ID, &a.AssetType, &contentRaw, &a.OwnerID, &a.VersionTag, &lastMod); err == nil {
			json.Unmarshal([]byte(contentRaw), &a.ContentDefinition)
			a.LastModified = lastMod.Format(time.RFC3339)
			assets = append(assets, a)
		}
	}
	return assets, nil
}

func (m *Manager) GetAssetHistory(assetID string) ([]AssetHistory, error) {
	if m.db == nil {
		return []AssetHistory{}, nil
	}

	query := `SELECT history_id, asset_id, asset_type, content_definition, owner_id, version_tag, created_at FROM asset_history WHERE asset_id = $1 ORDER BY history_id DESC;`
	rows, err := m.db.Query(query, assetID)
	if err != nil {
		return []AssetHistory{}, err
	}
	defer rows.Close()

	var history []AssetHistory
	for rows.Next() {
		var h AssetHistory
		var contentRaw string
		var createdAt time.Time
		if err := rows.Scan(&h.HistoryID, &h.AssetID, &h.AssetType, &contentRaw, &h.OwnerID, &h.VersionTag, &createdAt); err == nil {
			json.Unmarshal([]byte(contentRaw), &h.ContentDefinition)
			h.CreatedAt = createdAt.Format(time.RFC3339)
			history = append(history, h)
		}
	}
	return history, nil
}
