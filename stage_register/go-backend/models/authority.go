package models

import "time"

// FieldAgency represents an employing organization, partner, or surveying firm.
type FieldAgency struct {
	ID          int64     `json:"id"`
	MflUid      string    `json:"statgate_uid" db:"mfl_uid"`
	Code        string    `json:"agency_code" db:"code"`
	Name        string    `json:"agency_name" db:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updatedAt"`
}

type Authority = FieldAgency

// FieldAgencyListResponse is the response struct for GET /api/v1/field/agencies (excludes code)
type FieldAgencyListResponse struct {
	ID          int64     `json:"id"`
	MflUid      string    `json:"statgate_uid" db:"mfl_uid"`
	Name        string    `json:"agency_name" db:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AuthorityListResponse = FieldAgencyListResponse

// ToAuthorityListResponse converts a FieldAgency to FieldAgencyListResponse
func (a *FieldAgency) ToAuthorityListResponse() FieldAgencyListResponse {
	return FieldAgencyListResponse{
		ID:          a.ID,
		MflUid:      a.MflUid,
		Name:        a.Name,
		Description: a.Description,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

// FieldAgencyMflResponse is the response struct for StatGate API integration
type FieldAgencyMflResponse struct {
	MflUid      string    `json:"statgate_uid" db:"mfl_uid"`
	Code        string    `json:"agency_code" db:"code"`
	Name        string    `json:"agency_name" db:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updatedAt"`
}

type AuthorityMflResponse = FieldAgencyMflResponse

// ToAuthorityMflResponse converts a FieldAgency to FieldAgencyMflResponse
func (a *FieldAgency) ToAuthorityMflResponse() *FieldAgencyMflResponse {
	return &FieldAgencyMflResponse{
		MflUid:      a.MflUid,
		Code:        a.Code,
		Name:        a.Name,
		Description: a.Description,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}
