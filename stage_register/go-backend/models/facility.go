package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// FieldAgent represents an enumerator, field worker, or candidate in the StatGate Field System.
// It maps directly onto the underlying database table while exposing StatGate-aligned domain JSON tags.
type FieldAgent struct {
	ID                  int64      `json:"id"`
	Identifier          string     `json:"agent_code" db:"identifier"`
	MflUid              *string    `json:"statgate_uid" db:"mfl_uid"`
	Name                string     `json:"full_name" db:"name"`
	ShortName           *string    `json:"alias" db:"short_name"`
	HistoricalID        *string    `json:"external_id" db:"historical_id"`
	AdminUnitID         *int64     `json:"territory_id" db:"admin_unit_id"`
	Level               *string    `json:"role_tier" db:"level"`
	Ownership           *string    `json:"contract_type" db:"ownership"`
	Authority           *string    `json:"field_agency" db:"authority"`
	Region              *string    `json:"region,omitempty"`
	District            *string    `json:"district,omitempty"`
	Subcounty           *string    `json:"subcounty,omitempty"`
	Status              *string    `json:"recruitment_status" db:"status"`
	Reporting           *bool      `json:"active_field_status" db:"reporting"`
	Licensed            *bool      `json:"vetted_and_cleared" db:"licensed"`
	Address             *string    `json:"primary_location" db:"address"`
	ContactPersonEmail  *string    `json:"email" db:"contact_personemail"`
	ContactPersonMobile *string    `json:"phone_number" db:"contact_personmobile"`
	ContactPersonName   *string    `json:"supervisor_name" db:"contact_personname"`
	ContactPersonTitle  *string    `json:"supervisor_title" db:"contact_persontitle"`
	Longitude           *float64   `json:"longitude"`
	Latitude            *float64   `json:"latitude"`
	OpeningDate         *string    `json:"onboarding_date" db:"opening_date"`
	ClosingDate         *string    `json:"offboarding_date" db:"closing_date"`
	BedCapacity         *int64     `json:"daily_survey_capacity" db:"bed_capacity"`
	Services            Campaigns  `json:"assigned_campaigns" db:"services"`
	UserID              *int64     `json:"user_id" db:"user_id"`
	CreatedAt           time.Time  `json:"createdAt" db:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt" db:"updatedAt"`

	// Related data (from joins)
	AdminUnitMflUid *string `json:"territory_uid,omitempty" db:"admin_unit_mfl_uid"`
	AdminUnitName   *string `json:"territory_name,omitempty" db:"admin_unit_name"`
	LevelName       *string `json:"role_tier_name,omitempty" db:"level_name"`
	OwnershipName   *string `json:"contract_type_name,omitempty" db:"ownership_name"`
	AuthorityName   *string `json:"agency_name,omitempty" db:"authority_name"`

	// Nested objects for compatibility
	National     *NestedObject `json:"national,omitempty"`
	RegionObj    *NestedObject `json:"-"`
	DistrictObj  *NestedObject `json:"-"`
	SubcountyObj *NestedObject `json:"-"`
	Parish       *NestedObject `json:"parish,omitempty"`
	Village      *NestedObject `json:"village,omitempty"`
	Facility     *NestedObject `json:"field_station,omitempty"`
	LevelObj     *NestedObject `json:"-"`
	OwnershipObj *NestedObject `json:"-"`
	AuthorityObj *NestedObject `json:"-"`
}

type FacilityMflResponse struct {
	Identifier          string        `json:"agent_code" db:"identifier"`
	MflUid              *string       `json:"statgate_uid" db:"mfl_uid"`
	Name                string        `json:"full_name" db:"name"`
	ShortName           *string       `json:"alias" db:"short_name"`
	HistoricalID        *string       `json:"external_id" db:"historical_id"`
	AdminUnitID         *int64        `json:"territory_id" db:"admin_unit_id"`
	Level               *string       `json:"role_tier" db:"level"`
	Ownership           *string       `json:"contract_type" db:"ownership"`
	Authority           *string       `json:"field_agency" db:"authority"`
	Region              *string       `json:"region,omitempty"`
	District            *string       `json:"district,omitempty"`
	Subcounty           *string       `json:"subcounty,omitempty"`
	Status              *string       `json:"recruitment_status" db:"status"`
	Reporting           *bool         `json:"active_field_status" db:"reporting"`
	Licensed            *bool         `json:"vetted_and_cleared" db:"licensed"`
	Address             *string       `json:"primary_location" db:"address"`
	ContactPersonEmail  *string       `json:"email" db:"contact_personemail"`
	ContactPersonMobile *string       `json:"phone_number" db:"contact_personmobile"`
	ContactPersonName   *string       `json:"supervisor_name" db:"contact_personname"`
	ContactPersonTitle  *string       `json:"supervisor_title" db:"contact_persontitle"`
	Longitude           *float64      `json:"longitude"`
	Latitude            *float64      `json:"latitude"`
	OpeningDate         *string       `json:"onboarding_date" db:"opening_date"`
	ClosingDate         *string       `json:"offboarding_date" db:"closing_date"`
	BedCapacity         *int64        `json:"daily_survey_capacity" db:"bed_capacity"`
	Services            Campaigns     `json:"assigned_campaigns" db:"services"`
	CreatedAt           time.Time     `json:"createdAt" db:"createdAt"`
	UpdatedAt           time.Time     `json:"updatedAt" db:"updatedAt"`
	AdminUnitMflUid     *string       `json:"territory_uid,omitempty" db:"admin_unit_mfl_uid"`
	AdminUnitName       *string       `json:"territory_name,omitempty" db:"admin_unit_name"`
	LevelName           *string       `json:"role_tier_name,omitempty" db:"level_name"`
	OwnershipName       *string       `json:"contract_type_name,omitempty" db:"ownership_name"`
	AuthorityName       *string       `json:"agency_name,omitempty" db:"authority_name"`
	National            *NestedObject `json:"national,omitempty"`
	Parish              *NestedObject `json:"parish,omitempty"`
	Village             *NestedObject `json:"village,omitempty"`
	Facility            *NestedObject `json:"field_station,omitempty"`
}

func (f *FieldAgent) ToFacilityMflResponse() *FacilityMflResponse {
	return &FacilityMflResponse{
		Identifier:          f.Identifier,
		MflUid:              f.MflUid,
		Name:                f.Name,
		ShortName:           f.ShortName,
		HistoricalID:        f.HistoricalID,
		AdminUnitID:         f.AdminUnitID,
		Level:               f.Level,
		Ownership:           f.Ownership,
		Authority:           f.Authority,
		Region:              f.Region,
		District:            f.District,
		Subcounty:           f.Subcounty,
		Status:              f.Status,
		Reporting:           f.Reporting,
		Licensed:            f.Licensed,
		Address:             f.Address,
		ContactPersonEmail:  f.ContactPersonEmail,
		ContactPersonMobile: f.ContactPersonMobile,
		ContactPersonName:   f.ContactPersonName,
		ContactPersonTitle:  f.ContactPersonTitle,
		Longitude:           f.Longitude,
		Latitude:            f.Latitude,
		OpeningDate:         f.OpeningDate,
		ClosingDate:         f.ClosingDate,
		BedCapacity:         f.BedCapacity,
		Services:            f.Services,
		CreatedAt:           f.CreatedAt,
		UpdatedAt:           f.UpdatedAt,
		AdminUnitMflUid:     f.AdminUnitMflUid,
		AdminUnitName:       f.AdminUnitName,
		LevelName:           f.LevelName,
		OwnershipName:       f.OwnershipName,
		AuthorityName:       f.AuthorityName,
		National:            f.National,
		Parish:              f.Parish,
		Village:             f.Village,
		Facility:            f.Facility,
	}
}

// Facility is an alias for FieldAgent to preserve backward compatibility during refactoring
type Facility = FieldAgent

// AgentTierContractStat represents aggregated counts by role tier and contract type
type AgentTierContractStat struct {
	RoleTier   string `json:"role_tier"`
	DirectHire int64  `json:"direct_hire"`
	Contractor int64  `json:"contractor"`
	Volunteer  int64  `json:"volunteer"`
	Total      int64  `json:"total"`

	Level      string `json:"level"`
	Government int64  `json:"government"`
	Pfp        int64  `json:"pfp"`
	Pnfp       int64  `json:"pnfp"`
}

// FacilityLevelOwnershipStat is maintained for internal handler compatibility
type FacilityLevelOwnershipStat struct {
	Level      string `json:"level"`
	Government int64  `json:"government"`
	Pfp        int64  `json:"pfp"`
	Pnfp       int64  `json:"pnfp"`
	Total      int64  `json:"total"`
}

// AgentContractTotals represents overall contract totals (for cards)
type AgentContractTotals struct {
	DirectHire int64 `json:"direct_hire"`
	Contractor int64 `json:"contractor"`
	Volunteer  int64 `json:"volunteer"`
	Total      int64 `json:"total"`

	Government int64 `json:"government"`
	Pfp        int64 `json:"pfp"`
	Pnfp       int64 `json:"pnfp"`
}

type FacilityOwnershipTotals struct {
	Government int64 `json:"government"`
	Pfp        int64 `json:"pfp"`
	Pnfp       int64 `json:"pnfp"`
	Total      int64 `json:"total"`
}

// AgentDistributionStat represents a single distribution statistic
type AgentDistributionStat struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type FacilityDistributionStat = AgentDistributionStat

type NestedObject struct {
	MflUid *string `json:"statgate_uid,omitempty" db:"mfl_uid"`
	Name   *string `json:"name,omitempty"`
}

type Campaigns []interface{}

// Value implements driver.Valuer for Campaigns
func (c Campaigns) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner for Campaigns
func (c *Campaigns) Scan(value interface{}) error {
	if value == nil {
		*c = Campaigns{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			return nil
		}
		bytes = []byte(str)
	}
	return json.Unmarshal(bytes, c)
}

type Services = Campaigns
