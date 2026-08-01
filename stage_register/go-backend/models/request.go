package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type Request struct {
	ID                  int64          `json:"id"`
	RequestType         string         `json:"request_type" db:"request_type"`
	FacilityID          *int64         `json:"facility_id" db:"facility_id"`
	FacilityMflUid      *string        `json:"facility_mfl_uid" db:"facility_mfl_uid"`
	FacilityData        FacilityData   `json:"facility_data" db:"facility_data"`
	CurrentStatus       string         `json:"current_status" db:"current_status"`
	CurrentStage        string         `json:"current_stage" db:"current_stage"`
	InitiatedBy         int64          `json:"initiated_by" db:"initiated_by"`
	InitiatedByName     *string        `json:"initiated_by_name" db:"initiated_by_name"`
	InitiatedByEmail    *string        `json:"initiated_by_email" db:"initiated_by_email"`
	DistrictApproverID  *int64         `json:"district_approver_id" db:"district_approver_id"`
	MohClinicalID       *int64         `json:"moh_clinical_id" db:"moh_clinical_id"`
	MohPublisherID      *int64         `json:"moh_publisher_id" db:"moh_publisher_id"`
	RejectionReason     *string        `json:"rejection_reason" db:"rejection_reason"`
	CreatedAt           time.Time      `json:"createdAt" db:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt" db:"updatedAt"`
	Approvals           []Approval         `json:"approvals,omitempty"`
	Documents           []RequestDocument  `json:"documents,omitempty"`
}

// FacilityData represents the JSONB facility_data field
type FacilityData map[string]interface{}

// Value implements driver.Valuer for FacilityData
func (f FacilityData) Value() (driver.Value, error) {
	return json.Marshal(f)
}

// Scan implements sql.Scanner for FacilityData
func (f *FacilityData) Scan(value interface{}) error {
	if value == nil {
		*f = FacilityData{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, f)
}

type Approval struct {
	ID           int64     `json:"id"`
	RequestID    int64     `json:"request_id" db:"request_id"`
	Stage        string    `json:"stage"`
	Action       string    `json:"action"`
	ApproverID   int64     `json:"approver_id" db:"approver_id"`
	ApproverName *string   `json:"approver_name" db:"approver_name"`
	ApproverEmail *string  `json:"approver_email" db:"approver_email"`
	Comments     *string   `json:"comments"`
	CreatedAt    time.Time `json:"createdAt" db:"createdAt"`
}

type RequestDocument struct {
	ID              int64     `json:"id"`
	RequestID       int64     `json:"request_id" db:"request_id"`
	Filename        string    `json:"filename"`
	OriginalFilename string   `json:"original_filename" db:"original_filename"`
	FilePath        string    `json:"file_path" db:"file_path"`
	FileSize        *int64    `json:"file_size" db:"file_size"`
	MimeType        *string   `json:"mime_type" db:"mime_type"`
	DocType         *string   `json:"doc_type,omitempty" db:"doc_type"`
	CreatedAt       time.Time `json:"createdAt" db:"createdAt"`
}
