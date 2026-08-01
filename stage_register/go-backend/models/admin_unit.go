package models

import "time"

// FieldTerritory represents a geographic zone, region, district, or subcounty sampling unit in StatGate.
type FieldTerritory struct {
	ID          int64       `json:"id"`
	Name        string      `json:"territory_name" db:"name"`
	Code        *string     `json:"code"`
	MflUid      string      `json:"statgate_uid" db:"mfl_uid"`
	ParentID    *int64      `json:"parent_id" db:"parent_id"`
	LevelID     int64       `json:"hierarchy_level_id" db:"level_id"`
	Path        string      `json:"path"`
	CreatedAt   time.Time   `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt" db:"updatedAt"`
	AdminLevel  *AdminLevel `json:"admin_level,omitempty"`
}

type AdminUnit = FieldTerritory

// FieldTerritoryResponse is the response struct for StatGate Field API integration
type FieldTerritoryResponse struct {
	ID        int64           `json:"id"`
	Name      string          `json:"territory_name" db:"name"`
	MflUid    string          `json:"statgate_uid" db:"mfl_uid"`
	Path      string          `json:"path"`
	CreatedAt time.Time       `json:"createdAt" db:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt" db:"updatedAt"`
	Level     *AdminLevelInfo `json:"level,omitempty"`
	Parent    *ParentInfo     `json:"parent,omitempty"`
}

type AdminUnitMflResponse = FieldTerritoryResponse

// AdminLevelInfo is a simplified admin level for StatGate response
type AdminLevelInfo struct {
	ID          int64  `json:"id"`
	MflUid      string `json:"statgate_uid" db:"mfl_uid"`
	Name        string `json:"level_name" db:"name"`
	LevelNumber int    `json:"level_number"`
}

// ParentInfo contains parent territory information
type ParentInfo struct {
	ID     int64  `json:"id"`
	Name   string `json:"territory_name" db:"name"`
	MflUid string `json:"statgate_uid" db:"mfl_uid"`
}

// ToAdminUnitMflResponse converts a FieldTerritory to FieldTerritoryResponse
func (a *FieldTerritory) ToAdminUnitMflResponse(parent *FieldTerritory) *FieldTerritoryResponse {
	resp := &FieldTerritoryResponse{
		ID:        a.ID,
		Name:      a.Name,
		MflUid:    a.MflUid,
		Path:      a.Path,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}

	if a.AdminLevel != nil {
		resp.Level = &AdminLevelInfo{
			ID:          a.AdminLevel.ID,
			MflUid:      a.AdminLevel.MflUid,
			Name:        a.AdminLevel.Name,
			LevelNumber: a.AdminLevel.LevelNumber,
		}
	}

	if parent != nil {
		resp.Parent = &ParentInfo{
			ID:     parent.ID,
			Name:   parent.Name,
			MflUid: parent.MflUid,
		}
	}

	return resp
}
