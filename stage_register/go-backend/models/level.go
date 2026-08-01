package models

import "time"

// AgentRoleTier represents qualification & role tier levels in StatGate (e.g. Field Surveyor L1, Team Lead L2, Regional Lead L3).
type AgentRoleTier struct {
	ID          int64     `json:"id"`
	MflUid      string    `json:"statgate_uid" db:"mfl_uid"`
	Code        string    `json:"role_code" db:"code"`
	Name        string    `json:"tier_name" db:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updatedAt"`
}

type Level = AgentRoleTier

// RoleTierResponse is the response struct for GET /api/v1/field/tiers (excludes code)
type RoleTierResponse struct {
	ID          int64     `json:"id"`
	MflUid      string    `json:"statgate_uid" db:"mfl_uid"`
	Name        string    `json:"tier_name" db:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type LevelListResponse = RoleTierResponse

// ToLevelListResponse converts an AgentRoleTier to RoleTierResponse
func (l *AgentRoleTier) ToLevelListResponse() RoleTierResponse {
	return RoleTierResponse{
		ID:          l.ID,
		MflUid:      l.MflUid,
		Name:        l.Name,
		Description: l.Description,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}

// RoleTierMflResponse is the response struct for StatGate API integration
type RoleTierMflResponse struct {
	MflUid      string    `json:"statgate_uid" db:"mfl_uid"`
	Code        string    `json:"role_code" db:"code"`
	Name        string    `json:"tier_name" db:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updatedAt"`
}

type LevelMflResponse = RoleTierMflResponse

// ToLevelMflResponse converts an AgentRoleTier to RoleTierMflResponse
func (l *AgentRoleTier) ToLevelMflResponse() *RoleTierMflResponse {
	return &RoleTierMflResponse{
		MflUid:      l.MflUid,
		Code:        l.Code,
		Name:        l.Name,
		Description: l.Description,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}
