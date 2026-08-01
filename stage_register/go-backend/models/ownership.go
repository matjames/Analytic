package models

import "time"

// ContractType represents employment/contract classification (Direct Hire, Contractor, Volunteer, Consultant).
type ContractType struct {
	ID          int64     `json:"id"`
	MflUid      string    `json:"statgate_uid" db:"mfl_uid"`
	Code        string    `json:"contract_code" db:"code"`
	Name        string    `json:"contract_type_name" db:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updatedAt"`
}

type Ownership = ContractType

// ContractTypeListResponse is the response struct for GET /api/v1/field/contracts (excludes code)
type ContractTypeListResponse struct {
	ID          int64     `json:"id"`
	MflUid      string    `json:"statgate_uid" db:"mfl_uid"`
	Name        string    `json:"contract_type_name" db:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type OwnershipListResponse = ContractTypeListResponse

// ToOwnershipListResponse converts a ContractType to ContractTypeListResponse
func (o *ContractType) ToOwnershipListResponse() ContractTypeListResponse {
	return ContractTypeListResponse{
		ID:          o.ID,
		MflUid:      o.MflUid,
		Name:        o.Name,
		Description: o.Description,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}
}

// ContractTypeMflResponse is the response struct for StatGate API integration
type ContractTypeMflResponse struct {
	MflUid      string    `json:"statgate_uid" db:"mfl_uid"`
	Code        string    `json:"contract_code" db:"code"`
	Name        string    `json:"contract_type_name" db:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updatedAt"`
}

type OwnershipMflResponse = ContractTypeMflResponse

// ToOwnershipMflResponse converts a ContractType to ContractTypeMflResponse
func (o *ContractType) ToOwnershipMflResponse() *ContractTypeMflResponse {
	return &ContractTypeMflResponse{
		MflUid:      o.MflUid,
		Code:        o.Code,
		Name:        o.Name,
		Description: o.Description,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}
}
