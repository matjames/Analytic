package services

import (
	"database/sql"
	"fmt"
	"strings"

	"go-backend/configs"
)

// OrgUnit represents a DHIS2-like organization unit response
type OrgUnit struct {
	ID                int64               `json:"id"`
	MflUid            string              `json:"mfl_uid"`
	Identifier        *string             `json:"identifier,omitempty"`
	Name              string              `json:"name"`
	ShortName         *string             `json:"shortName,omitempty"`
	Level             int                 `json:"level"`
	Path              string              `json:"path"`
	Parent            *OrgUnitRef         `json:"parent,omitempty"`
	Children          []*OrgUnitRef       `json:"children,omitempty"`
	OrganisationUnits []*OrgUnitRef       `json:"organisationUnits,omitempty"`
	OpeningDate       *string             `json:"opening_date,omitempty"`
	ClosedDate        *string             `json:"closing_date,omitempty"`
	Geometry          *Geometry           `json:"geometry,omitempty"`
	LastUpdated       string              `json:"lastUpdated"`
	Created           string              `json:"created"`
	AttributeValues   []map[string]string `json:"attributeValues,omitempty"`
	// Facility-specific fields (only for level 6; omit when nil so non-facility levels don't show them)
	FacilityMflUid      *string       `json:"facility_mfl_uid,omitempty"`
	HistoricalID        *string       `json:"historical_id,omitempty"`
	AdminUnitID         *int64        `json:"admin_unit_id,omitempty"`
	Status              *string       `json:"status,omitempty"`
	Reporting           *bool         `json:"reporting,omitempty"`
	Longitude           *float64      `json:"longitude,omitempty"`
	Latitude            *float64      `json:"latitude,omitempty"`
	FacilityLevel       *LookupObject `json:"facility_level,omitempty"`
	Authority           *LookupObject `json:"authority,omitempty"`
	Ownership           *LookupObject `json:"ownership,omitempty"`
	Address             *string       `json:"address,omitempty"`
	ContactPersonEmail  *string       `json:"contact_personemail,omitempty"`
	ContactPersonMobile *string       `json:"contact_personmobile,omitempty"`
	ContactPersonName   *string       `json:"contact_personname,omitempty"`
	ContactPersonTitle  *string       `json:"contact_persontitle,omitempty"`
	BedCapacity         *int          `json:"bed_capacity,omitempty"`
	Services            *string       `json:"services,omitempty"`
	// Hierarchy (admin units in the path - national, region, district, etc.)
	National        *OrgUnitRef `json:"national,omitempty"`
	Region          *OrgUnitRef `json:"region,omitempty"`
	District        *OrgUnitRef `json:"district,omitempty"`
	DLGMunicipality *OrgUnitRef `json:"dlg_municipality,omitempty"`
	Subcounty       *OrgUnitRef `json:"subcounty,omitempty"`
	Parish          *OrgUnitRef `json:"parish,omitempty"`
	Village         *OrgUnitRef `json:"village,omitempty"`
}

// LookupObject represents a lookup reference (level, authority, ownership)
type LookupObject struct {
	MflUid string `json:"mfl_uid"`
	Name   string `json:"name"`
}

// OrgUnitRef represents a reference to an organization unit
type OrgUnitRef struct {
	ID     int64  `json:"id"`
	MflUid string `json:"mfl_uid"`
	Name   string `json:"name,omitempty"`
}

// Geometry represents geographic coordinates
type Geometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// OrgUnitsResponse represents the DHIS2-like paginated response
type OrgUnitsResponse struct {
	Pager    *Pager     `json:"pager,omitempty"`
	OrgUnits []*OrgUnit `json:"orgunits"`
	Total    int        `json:"total,omitempty"`
	Page     int        `json:"page,omitempty"`
	PageSize int        `json:"pageSize,omitempty"`
}

// Pager represents pagination information
type Pager struct {
	Page      int `json:"page"`
	PageSize  int `json:"pageSize"`
	PageCount int `json:"pageCount"`
	Total     int `json:"total"`
}

// OrgUnitsQueryParams holds query parameters for organization units
type OrgUnitsQueryParams struct {
	Level             int
	MinLevel          int
	MaxLevel          int
	ParentID          int64
	NationalID        int64
	RegionID          int64
	RegionName        string
	DistrictID        int64
	DistrictMflUid    string // filter by district mfl_uid (e.g. from districtId query param)
	DistrictName      string
	DLGID             int64
	SubcountyID       int64
	SubcountyName     string
	Name              string
	MflUid            string
	Search            string
	Status            string
	Ownership         string
	Authority         string
	FacilityLevel     string // facility level type filter (level_mfl_uid or level_name, e.g. "HC III")
	Reporting         *bool
	Licensed          *bool
	Paging            bool
	Page              int
	PageSize          int
	IncludeChildren   bool
	IncludeParent     bool
	IncludeFacilities bool
	RootOnly          bool
	UpdatedSince      string // filter by lastUpdated >= date (YYYY-MM-DD or ISO format)
}

// identifierForAPI returns identifier with first 6 characters removed for API response
func identifierForAPI(s string) string {
	if len(s) <= 6 {
		return s
	}
	return s[6:]
}

// Helper function to set facility fields
func setFacilityFields(orgUnit *OrgUnit,
	facilityMflUid, historicalID sql.NullString,
	adminUnitID sql.NullInt64,
	status sql.NullString,
	reporting sql.NullBool,
	longitude, latitude sql.NullFloat64,
	openingDate, closingDate sql.NullString,
	facilityLevelMflUid, facilityLevelName sql.NullString,
	authorityMflUid, authorityName sql.NullString,
	ownershipMflUid, ownershipName sql.NullString,
	address, contactEmail, contactMobile, contactName, contactTitle, services sql.NullString,
	bedCapacity sql.NullInt64) {

	if facilityMflUid.Valid {
		orgUnit.FacilityMflUid = &facilityMflUid.String
	}
	if historicalID.Valid {
		orgUnit.HistoricalID = &historicalID.String
	}
	if adminUnitID.Valid {
		orgUnit.AdminUnitID = &adminUnitID.Int64
	}
	if status.Valid {
		orgUnit.Status = &status.String
	}
	if reporting.Valid {
		orgUnit.Reporting = &reporting.Bool
	}
	if longitude.Valid {
		orgUnit.Longitude = &longitude.Float64
	}
	if latitude.Valid {
		orgUnit.Latitude = &latitude.Float64
	}
	if openingDate.Valid {
		orgUnit.OpeningDate = &openingDate.String
	}
	if closingDate.Valid {
		orgUnit.ClosedDate = &closingDate.String
	}
	if facilityLevelMflUid.Valid && facilityLevelName.Valid {
		orgUnit.FacilityLevel = &LookupObject{
			MflUid: facilityLevelMflUid.String,
			Name:   facilityLevelName.String,
		}
	}
	if authorityMflUid.Valid && authorityName.Valid {
		orgUnit.Authority = &LookupObject{
			MflUid: authorityMflUid.String,
			Name:   authorityName.String,
		}
	}
	if ownershipMflUid.Valid && ownershipName.Valid {
		orgUnit.Ownership = &LookupObject{
			MflUid: ownershipMflUid.String,
			Name:   ownershipName.String,
		}
	}
	if address.Valid {
		orgUnit.Address = &address.String
	}
	if contactEmail.Valid {
		orgUnit.ContactPersonEmail = &contactEmail.String
	}
	if contactMobile.Valid {
		orgUnit.ContactPersonMobile = &contactMobile.String
	}
	if contactName.Valid {
		orgUnit.ContactPersonName = &contactName.String
	}
	if contactTitle.Valid {
		orgUnit.ContactPersonTitle = &contactTitle.String
	}
	if bedCapacity.Valid {
		bedCapInt := int(bedCapacity.Int64)
		orgUnit.BedCapacity = &bedCapInt
	}
	if services.Valid {
		orgUnit.Services = &services.String
	}
}

// ListOrgUnits lists organization units with query parameters
func ListOrgUnits(queryParams OrgUnitsQueryParams) (*OrgUnitsResponse, error) {
	// Set defaults
	if queryParams.Page < 1 {
		queryParams.Page = 1
	}
	if queryParams.PageSize < 1 {
		queryParams.PageSize = 50
	}
	if queryParams.PageSize > 500 {
		queryParams.PageSize = 500
	}

	offset := (queryParams.Page - 1) * queryParams.PageSize

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	params := []interface{}{}
	paramCount := 1

	if queryParams.Level > 0 {
		whereClause += fmt.Sprintf(" AND o.level = $%d", paramCount)
		params = append(params, queryParams.Level)
		paramCount++
	}

	if queryParams.MinLevel > 0 {
		whereClause += fmt.Sprintf(" AND o.level >= $%d", paramCount)
		params = append(params, queryParams.MinLevel)
		paramCount++
	}

	if queryParams.MaxLevel > 0 {
		whereClause += fmt.Sprintf(" AND o.level <= $%d", paramCount)
		params = append(params, queryParams.MaxLevel)
		paramCount++
	}

	if queryParams.ParentID > 0 {
		whereClause += fmt.Sprintf(" AND o.parent_id = $%d", paramCount)
		params = append(params, queryParams.ParentID)
		paramCount++
	}

	if queryParams.NationalID > 0 {
		whereClause += fmt.Sprintf(" AND o.national_id = $%d", paramCount)
		params = append(params, queryParams.NationalID)
		paramCount++
	}

	if queryParams.RegionID > 0 {
		whereClause += fmt.Sprintf(" AND o.region_id = $%d", paramCount)
		params = append(params, queryParams.RegionID)
		paramCount++
	}

	if queryParams.RegionName != "" {
		whereClause += fmt.Sprintf(" AND o.region_name ILIKE $%d", paramCount)
		params = append(params, "%"+queryParams.RegionName+"%")
		paramCount++
	}

	if queryParams.DistrictID > 0 {
		whereClause += fmt.Sprintf(" AND o.district_city_id = $%d", paramCount)
		params = append(params, queryParams.DistrictID)
		paramCount++
	}

	if queryParams.DistrictMflUid != "" {
		whereClause += fmt.Sprintf(" AND o.district_city_mfl_uid = $%d", paramCount)
		params = append(params, queryParams.DistrictMflUid)
		paramCount++
	}

	if queryParams.DistrictName != "" {
		whereClause += fmt.Sprintf(" AND o.district_city_name ILIKE $%d", paramCount)
		params = append(params, "%"+queryParams.DistrictName+"%")
		paramCount++
	}

	if queryParams.DLGID > 0 {
		whereClause += fmt.Sprintf(" AND o.dlg_municipality_id = $%d", paramCount)
		params = append(params, queryParams.DLGID)
		paramCount++
	}

	if queryParams.SubcountyID > 0 {
		whereClause += fmt.Sprintf(" AND o.subcounty_division_id = $%d", paramCount)
		params = append(params, queryParams.SubcountyID)
		paramCount++
	}

	if queryParams.SubcountyName != "" {
		whereClause += fmt.Sprintf(" AND o.subcounty_division_name ILIKE $%d", paramCount)
		params = append(params, "%"+queryParams.SubcountyName+"%")
		paramCount++
	}

	if queryParams.Name != "" {
		whereClause += fmt.Sprintf(" AND o.name = $%d", paramCount)
		params = append(params, queryParams.Name)
		paramCount++
	}

	if queryParams.Search != "" {
		whereClause += fmt.Sprintf(" AND o.name ILIKE $%d", paramCount)
		params = append(params, "%"+queryParams.Search+"%")
		paramCount++
	}

	if queryParams.MflUid != "" {
		whereClause += fmt.Sprintf(" AND o.mfl_uid = $%d", paramCount)
		params = append(params, queryParams.MflUid)
		paramCount++
	}

	if queryParams.Status != "" {
		whereClause += fmt.Sprintf(" AND o.status = $%d", paramCount)
		params = append(params, queryParams.Status)
		paramCount++
	}

	if queryParams.Ownership != "" {
		// Match by mfl_uid, name (ILIKE), or code (via subquery) so "GOV", "Government", or mfl_uid work
		whereClause += fmt.Sprintf(" AND (o.ownership_mfl_uid = $%d OR o.ownership_name ILIKE $%d OR EXISTS (SELECT 1 FROM ownership ow WHERE ow.mfl_uid = o.ownership AND ow.code ILIKE $%d))", paramCount, paramCount+1, paramCount+2)
		params = append(params, queryParams.Ownership, "%"+queryParams.Ownership+"%", "%"+queryParams.Ownership+"%")
		paramCount += 3
	}

	if queryParams.Authority != "" {
		// Match by mfl_uid, name (ILIKE), or code (via subquery)
		whereClause += fmt.Sprintf(" AND (o.authority_mfl_uid = $%d OR o.authority_name ILIKE $%d OR EXISTS (SELECT 1 FROM authority av WHERE av.mfl_uid = o.authority AND av.code ILIKE $%d))", paramCount, paramCount+1, paramCount+2)
		params = append(params, queryParams.Authority, "%"+queryParams.Authority+"%", "%"+queryParams.Authority+"%")
		paramCount += 3
	}

	if queryParams.FacilityLevel != "" {
		// Filter by facility level type (e.g. HC III): match level_mfl_uid or level_name
		whereClause += fmt.Sprintf(" AND (o.level_mfl_uid = $%d OR o.level_name ILIKE $%d)", paramCount, paramCount+1)
		params = append(params, queryParams.FacilityLevel, "%"+queryParams.FacilityLevel+"%")
		paramCount += 2
	}

	if queryParams.Reporting != nil {
		whereClause += fmt.Sprintf(" AND o.reporting = $%d", paramCount)
		params = append(params, *queryParams.Reporting)
		paramCount++
	}

	if queryParams.Licensed != nil {
		whereClause += fmt.Sprintf(" AND o.licensed = $%d", paramCount)
		params = append(params, *queryParams.Licensed)
		paramCount++
	}

	if queryParams.RootOnly {
		whereClause += " AND o.parent_id IS NULL"
	}

	if queryParams.UpdatedSince != "" {
		whereClause += fmt.Sprintf(" AND o.\"updatedAt\" >= $%d", paramCount)
		params = append(params, queryParams.UpdatedSince)
		paramCount++
	}

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM orgunits o
		%s
	`, whereClause)

	var total int
	err := configs.DB.QueryRow(countQuery, params...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Prepare field selection (include hierarchy columns from view)
	selectFields := `
		o.admin_unit_id, o.mfl_uid, o.identifier, o.name, o.short_name,
		o.level, au.path, o.opening_date, o.closing_date, 
		o."createdAt", o."updatedAt",
		o.historical_id, o.status, o.reporting, o.licensed, 
		o.longitude, o.latitude,
		o.level_mfl_uid, o.level_name,
		o.authority_mfl_uid, o.authority_name,
		o.ownership_mfl_uid, o.ownership_name,
		o.address, o.contact_personemail, o.contact_personmobile, 
		o.contact_personname, o.contact_persontitle, o.bed_capacity, o.services,
		o.national_id, o.national_name, o.national_mfl_uid,
		o.region_id, o.region_name, o.region_mfl_uid,
		o.district_city_id, o.district_city_name, o.district_city_mfl_uid,
		o.dlg_municipality_id, o.dlg_municipality_name, o.dlg_municipality_mfl_uid,
		o.subcounty_division_id, o.subcounty_division_name, o.subcounty_division_mfl_uid,
		o.parish_id, o.parish_name, o.parish_mfl_uid,
		o.village_id, o.village_name, o.village_mfl_uid,
		p.admin_unit_id as parent_admin_id, p.mfl_uid as parent_mfl_uid, p.name as parent_name
	`

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT %s
		FROM orgunits o
		LEFT JOIN admin_units au ON o.admin_unit_id = au.id
		LEFT JOIN orgunits p ON o.parent_id = p.admin_unit_id
		%s
		ORDER BY o.level ASC, o.name ASC
	`, selectFields, whereClause)

	// Add pagination if enabled
	if queryParams.Paging {
		dataQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramCount, paramCount+1)
		params = append(params, queryParams.PageSize, offset)
	}

	rows, err := configs.DB.Query(dataQuery, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgUnits []*OrgUnit
	for rows.Next() {
		var id sql.NullInt64
		var mflUid, name, path, created, updated string
		var identifier, shortName sql.NullString
		var level int
		var openingDate, closingDate sql.NullString
		var historicalID, status sql.NullString
		var reporting, licensed sql.NullBool
		var longitude, latitude sql.NullFloat64
		var facilityLevelMflUid, facilityLevelName sql.NullString
		var authorityMflUid, authorityName sql.NullString
		var ownershipMflUid, ownershipName sql.NullString
		var address, contactEmail, contactMobile, contactName, contactTitle, services sql.NullString
		var bedCapacity sql.NullInt64
		var nationalID, regionID, districtID, dlgID, subcountyID, parishID, villageID sql.NullInt64
		var nationalName, nationalMflUid, regionName, regionMflUid, districtName, districtMflUid sql.NullString
		var dlgName, dlgMflUid, subcountyName, subcountyMflUid, parishName, parishMflUid, villageName, villageMflUid sql.NullString
		var parentAdminID sql.NullInt64
		var parentMflUid, parentName sql.NullString

		err := rows.Scan(
			&id, &mflUid, &identifier, &name, &shortName,
			&level, &path, &openingDate, &closingDate,
			&created, &updated,
			&historicalID, &status, &reporting, &licensed,
			&longitude, &latitude,
			&facilityLevelMflUid, &facilityLevelName,
			&authorityMflUid, &authorityName,
			&ownershipMflUid, &ownershipName,
			&address, &contactEmail, &contactMobile, &contactName, &contactTitle, &bedCapacity, &services,
			&nationalID, &nationalName, &nationalMflUid,
			&regionID, &regionName, &regionMflUid,
			&districtID, &districtName, &districtMflUid,
			&dlgID, &dlgName, &dlgMflUid,
			&subcountyID, &subcountyName, &subcountyMflUid,
			&parishID, &parishName, &parishMflUid,
			&villageID, &villageName, &villageMflUid,
			&parentAdminID, &parentMflUid, &parentName,
		)
		if err != nil {
			continue
		}

		orgUnit := &OrgUnit{
			ID:          id.Int64,
			MflUid:      mflUid,
			Name:        name,
			Level:       level,
			Path:        strings.TrimSuffix(path, "/"),
			Created:     created,
			LastUpdated: updated,
		}

		if identifier.Valid {
			s := identifierForAPI(identifier.String)
			orgUnit.Identifier = &s
		}
		if shortName.Valid {
			orgUnit.ShortName = &shortName.String
		}
		if openingDate.Valid {
			orgUnit.OpeningDate = &openingDate.String
		}
		if closingDate.Valid {
			orgUnit.ClosedDate = &closingDate.String
		}

		// Add parent for any level when parent exists (same as facilities)
		if parentAdminID.Valid && parentMflUid.Valid {
			orgUnit.Parent = &OrgUnitRef{
				ID:     parentAdminID.Int64,
				MflUid: parentMflUid.String,
				Name:   parentName.String,
			}
		}

		// Add hierarchy objects (omit the one matching current level; it's already in name)
		if level != 1 && level != 2 && level != 3 && nationalID.Valid && nationalMflUid.Valid {
			orgUnit.National = &OrgUnitRef{ID: nationalID.Int64, MflUid: nationalMflUid.String, Name: nationalName.String}
		}
		if level != 2 && level != 3 && regionID.Valid && regionMflUid.Valid {
			orgUnit.Region = &OrgUnitRef{ID: regionID.Int64, MflUid: regionMflUid.String, Name: regionName.String}
		}
		if level != 3 && districtID.Valid && districtMflUid.Valid {
			orgUnit.District = &OrgUnitRef{ID: districtID.Int64, MflUid: districtMflUid.String, Name: districtName.String}
		}
		if level != 4 && dlgID.Valid && dlgMflUid.Valid {
			orgUnit.DLGMunicipality = &OrgUnitRef{ID: dlgID.Int64, MflUid: dlgMflUid.String, Name: dlgName.String}
		}
		if level != 5 && subcountyID.Valid && subcountyMflUid.Valid {
			orgUnit.Subcounty = &OrgUnitRef{ID: subcountyID.Int64, MflUid: subcountyMflUid.String, Name: subcountyName.String}
		}
		// For level 6 (facilities) hierarchy stops at subcounty (5). Parish and village only for level 7+.
		if level != 6 {
			if level != 7 && parishID.Valid && parishMflUid.Valid {
				orgUnit.Parish = &OrgUnitRef{ID: parishID.Int64, MflUid: parishMflUid.String, Name: parishName.String}
			}
			if level != 8 && villageID.Valid && villageMflUid.Valid {
				orgUnit.Village = &OrgUnitRef{ID: villageID.Int64, MflUid: villageMflUid.String, Name: villageName.String}
			}
		}

		// Add facility-specific fields only for level 6 (facilities)
		if level == 6 {
			if historicalID.Valid {
				orgUnit.HistoricalID = &historicalID.String
			}
			if status.Valid {
				orgUnit.Status = &status.String
			}
			if reporting.Valid {
				orgUnit.Reporting = &reporting.Bool
			}
			if longitude.Valid {
				orgUnit.Longitude = &longitude.Float64
			}
			if latitude.Valid {
				orgUnit.Latitude = &latitude.Float64
			}
			if facilityLevelMflUid.Valid && facilityLevelName.Valid {
				orgUnit.FacilityLevel = &LookupObject{
					MflUid: facilityLevelMflUid.String,
					Name:   facilityLevelName.String,
				}
			}
			if authorityMflUid.Valid && authorityName.Valid {
				orgUnit.Authority = &LookupObject{
					MflUid: authorityMflUid.String,
					Name:   authorityName.String,
				}
			}
			if ownershipMflUid.Valid && ownershipName.Valid {
				orgUnit.Ownership = &LookupObject{
					MflUid: ownershipMflUid.String,
					Name:   ownershipName.String,
				}
			}
			if address.Valid {
				orgUnit.Address = &address.String
			}
			if contactEmail.Valid {
				orgUnit.ContactPersonEmail = &contactEmail.String
			}
			if contactMobile.Valid {
				orgUnit.ContactPersonMobile = &contactMobile.String
			}
			if contactName.Valid {
				orgUnit.ContactPersonName = &contactName.String
			}
			if contactTitle.Valid {
				orgUnit.ContactPersonTitle = &contactTitle.String
			}
			if bedCapacity.Valid {
				bedCapInt := int(bedCapacity.Int64)
				orgUnit.BedCapacity = &bedCapInt
			}
			if services.Valid {
				orgUnit.Services = &services.String
			}
		}

		// Add children if requested
		if queryParams.IncludeChildren && id.Valid {
			children, _ := getOrgUnitChildren(id.Int64)
			orgUnit.Children = children
			orgUnit.OrganisationUnits = children
		}

		orgUnits = append(orgUnits, orgUnit)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Build response
	response := &OrgUnitsResponse{
		OrgUnits: orgUnits,
		Total:    total,
		Page:     queryParams.Page,
		PageSize: queryParams.PageSize,
	}

	// Add pager if paging is enabled
	if queryParams.Paging {
		pageCount := (total + queryParams.PageSize - 1) / queryParams.PageSize
		response.Pager = &Pager{
			Page:      queryParams.Page,
			PageSize:  queryParams.PageSize,
			PageCount: pageCount,
			Total:     total,
		}
	}

	return response, nil
}

// GetOrgUnit gets a single organization unit by ID (mfl_uid)
func GetOrgUnit(id string, includeChildren, includeParent bool) (*OrgUnit, error) {
	query := `
		SELECT 
			o.admin_unit_id, o.mfl_uid, o.identifier, o.name, au.path, o."createdAt", o."updatedAt",
			o.level,
			p.admin_unit_id as parent_id, p.mfl_uid as parent_mfl_uid, p.name as parent_name,
			o.mfl_uid as facility_mfl_uid, o.historical_id, o.admin_unit_id, 
			o.status, o.reporting, o.longitude, o.latitude, o.opening_date, o.closing_date,
			o.level_mfl_uid, o.level_name,
			o.authority_mfl_uid, o.authority_name,
			o.ownership_mfl_uid, o.ownership_name,
			o.address, o.contact_personemail, o.contact_personmobile,
			o.contact_personname, o.contact_persontitle, o.bed_capacity, o.services,
			o.national_id, o.national_name, o.national_mfl_uid,
			o.region_id, o.region_name, o.region_mfl_uid,
			o.district_city_id, o.district_city_name, o.district_city_mfl_uid,
			o.dlg_municipality_id, o.dlg_municipality_name, o.dlg_municipality_mfl_uid,
			o.subcounty_division_id, o.subcounty_division_name, o.subcounty_division_mfl_uid,
			o.parish_id, o.parish_name, o.parish_mfl_uid,
			o.village_id, o.village_name, o.village_mfl_uid
		FROM orgunits o
		LEFT JOIN admin_units au ON o.admin_unit_id = au.id
		LEFT JOIN orgunits p ON o.parent_id = p.admin_unit_id
		WHERE o.mfl_uid = $1
		LIMIT 1
	`

	var dbID int64
	var mflUid, name, path string
	var identifier sql.NullString
	var createdAt, updatedAt string
	var levelNumber int
	var parentID sql.NullInt64
	var parentMflUid, parentName sql.NullString
	// Facility-specific fields
	var facilityMflUid, historicalID sql.NullString
	var adminUnitID sql.NullInt64
	var status sql.NullString
	var reporting sql.NullBool
	var longitude, latitude sql.NullFloat64
	var openingDate, closingDate sql.NullString
	var facilityLevelMflUid, facilityLevelName sql.NullString
	var authorityMflUid, authorityName sql.NullString
	var ownershipMflUid, ownershipName sql.NullString
	var address, contactEmail, contactMobile, contactName, contactTitle, services sql.NullString
	var bedCapacity sql.NullInt64
	var nationalID, regionID, districtID, dlgID, subcountyID, parishID, villageID sql.NullInt64
	var nationalName, nationalMflUid, regionName, regionMflUid, districtName, districtMflUid sql.NullString
	var dlgName, dlgMflUid, subcountyName, subcountyMflUid, parishName, parishMflUid, villageName, villageMflUid sql.NullString

	err := configs.DB.QueryRow(query, id).Scan(
		&dbID, &mflUid, &identifier, &name, &path, &createdAt, &updatedAt,
		&levelNumber,
		&parentID, &parentMflUid, &parentName,
		&facilityMflUid, &historicalID, &adminUnitID,
		&status, &reporting, &longitude, &latitude, &openingDate,
		&facilityLevelMflUid, &facilityLevelName,
		&authorityMflUid, &authorityName,
		&ownershipMflUid, &ownershipName,
		&address, &contactEmail, &contactMobile, &contactName, &contactTitle, &bedCapacity, &services,
		&nationalID, &nationalName, &nationalMflUid,
		&regionID, &regionName, &regionMflUid,
		&districtID, &districtName, &districtMflUid,
		&dlgID, &dlgName, &dlgMflUid,
		&subcountyID, &subcountyName, &subcountyMflUid,
		&parishID, &parishName, &parishMflUid,
		&villageID, &villageName, &villageMflUid,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization unit not found")
	}
	if err != nil {
		return nil, err
	}

	orgUnit := &OrgUnit{
		ID:          dbID,
		MflUid:      mflUid,
		Name:        name,
		Level:       levelNumber,
		Path:        strings.TrimSuffix(path, "/"),
		Created:     createdAt,
		LastUpdated: updatedAt,
	}

	if identifier.Valid {
		s := identifierForAPI(identifier.String)
		orgUnit.Identifier = &s
	}

	// Add parent for any level when parent exists (same as facilities)
	if parentID.Valid && parentMflUid.Valid {
		orgUnit.Parent = &OrgUnitRef{
			ID:     parentID.Int64,
			MflUid: parentMflUid.String,
			Name:   parentName.String,
		}
	}

	// Add hierarchy objects (omit the one matching current level; it's already in name)
	if levelNumber != 1 && levelNumber != 2 && levelNumber != 3 && nationalID.Valid && nationalMflUid.Valid {
		orgUnit.National = &OrgUnitRef{ID: nationalID.Int64, MflUid: nationalMflUid.String, Name: nationalName.String}
	}
	if levelNumber != 2 && levelNumber != 3 && regionID.Valid && regionMflUid.Valid {
		orgUnit.Region = &OrgUnitRef{ID: regionID.Int64, MflUid: regionMflUid.String, Name: regionName.String}
	}
	if levelNumber != 3 && districtID.Valid && districtMflUid.Valid {
		orgUnit.District = &OrgUnitRef{ID: districtID.Int64, MflUid: districtMflUid.String, Name: districtName.String}
	}
	if levelNumber != 4 && dlgID.Valid && dlgMflUid.Valid {
		orgUnit.DLGMunicipality = &OrgUnitRef{ID: dlgID.Int64, MflUid: dlgMflUid.String, Name: dlgName.String}
	}
	if levelNumber != 5 && subcountyID.Valid && subcountyMflUid.Valid {
		orgUnit.Subcounty = &OrgUnitRef{ID: subcountyID.Int64, MflUid: subcountyMflUid.String, Name: subcountyName.String}
	}
	// For level 6 (facilities) hierarchy stops at subcounty (5). Parish and village only for level 7+.
	if levelNumber != 6 {
		if levelNumber != 7 && parishID.Valid && parishMflUid.Valid {
			orgUnit.Parish = &OrgUnitRef{ID: parishID.Int64, MflUid: parishMflUid.String, Name: parishName.String}
		}
		if levelNumber != 8 && villageID.Valid && villageMflUid.Valid {
			orgUnit.Village = &OrgUnitRef{ID: villageID.Int64, MflUid: villageMflUid.String, Name: villageName.String}
		}
	}

	// Add facility-specific fields (only for level 6)
	if levelNumber == 6 {
		setFacilityFields(orgUnit, facilityMflUid, historicalID, adminUnitID,
			status, reporting, longitude, latitude, openingDate, closingDate,
			facilityLevelMflUid, facilityLevelName,
			authorityMflUid, authorityName,
			ownershipMflUid, ownershipName,
			address, contactEmail, contactMobile, contactName, contactTitle, services, bedCapacity)
	}

	// Add children if requested
	if includeChildren {
		children, _ := getOrgUnitChildren(dbID)
		orgUnit.Children = children
		orgUnit.OrganisationUnits = children
	}

	return orgUnit, nil
}

// getOrgUnitChildren gets immediate children of an organization unit
func getOrgUnitChildren(parentID int64) ([]*OrgUnitRef, error) {
	query := `
		SELECT au.id, au.mfl_uid, au.name
		FROM admin_units au
		WHERE au.parent_id = $1
		ORDER BY au.name ASC
	`

	rows, err := configs.DB.Query(query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var children []*OrgUnitRef
	for rows.Next() {
		var id int64
		var mflUid, name string
		if err := rows.Scan(&id, &mflUid, &name); err != nil {
			continue
		}
		children = append(children, &OrgUnitRef{
			ID:     id,
			MflUid: mflUid,
			Name:   name,
		})
	}

	return children, nil
}

// ListOrgUnitsByLevel lists organization units by level
func ListOrgUnitsByLevel(level int, page, pageSize int) (*OrgUnitsResponse, error) {
	return ListOrgUnits(OrgUnitsQueryParams{
		Level:    level,
		Paging:   true,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListOrgUnitsTree gets the organization units tree structure
func ListOrgUnitsTree(rootID string) ([]*OrgUnit, error) {
	whereClause := ""
	params := []interface{}{}

	if rootID != "" {
		whereClause = "WHERE o.mfl_uid = $1"
		params = append(params, rootID)
	} else {
		whereClause = "WHERE o.parent_id IS NULL"
	}

	query := fmt.Sprintf(`
		SELECT 
			o.admin_unit_id, o.mfl_uid, o.identifier, o.name, au.path, o."createdAt", o."updatedAt",
			o.level
		FROM orgunits o
		LEFT JOIN admin_units au ON o.admin_unit_id = au.id
		%s
		ORDER BY o.name ASC
	`, whereClause)

	rows, err := configs.DB.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgUnits []*OrgUnit
	for rows.Next() {
		var id int64
		var mflUid, name, path string
		var identifier sql.NullString
		var createdAt, updatedAt string
		var levelNumber int

		err := rows.Scan(
			&id, &mflUid, &identifier, &name, &path, &createdAt, &updatedAt,
			&levelNumber,
		)
		if err != nil {
			continue
		}

		orgUnit := &OrgUnit{
			ID:          id,
			MflUid:      mflUid,
			Name:        name,
			Level:       levelNumber,
			Path:        strings.TrimSuffix(path, "/"),
			Created:     createdAt,
			LastUpdated: updatedAt,
		}

		if identifier.Valid {
			s := identifierForAPI(identifier.String)
			orgUnit.Identifier = &s
		}

		// Recursively get children
		children := buildOrgUnitTree(id)
		if len(children) > 0 {
			orgUnit.Children = make([]*OrgUnitRef, len(children))
			for i, child := range children {
				orgUnit.Children[i] = &OrgUnitRef{
					ID:     child.ID,
					MflUid: child.MflUid,
					Name:   child.Name,
				}
			}
			orgUnit.OrganisationUnits = orgUnit.Children
		}

		orgUnits = append(orgUnits, orgUnit)
	}

	return orgUnits, nil
}

// buildOrgUnitTree recursively builds organization unit tree
func buildOrgUnitTree(parentID int64) []*OrgUnit {
	query := `
		SELECT 
			o.admin_unit_id, o.mfl_uid, o.identifier, o.name, au.path, o."createdAt", o."updatedAt",
			o.level
		FROM orgunits o
		LEFT JOIN admin_units au ON o.admin_unit_id = au.id
		WHERE o.parent_id = $1
		ORDER BY o.name ASC
	`

	rows, err := configs.DB.Query(query, parentID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var orgUnits []*OrgUnit
	for rows.Next() {
		var id int64
		var mflUid, name, path string
		var identifier sql.NullString
		var createdAt, updatedAt string
		var levelNumber int

		err := rows.Scan(
			&id, &mflUid, &identifier, &name, &path, &createdAt, &updatedAt,
			&levelNumber,
		)
		if err != nil {
			continue
		}

		orgUnit := &OrgUnit{
			ID:          id,
			MflUid:      mflUid,
			Name:        name,
			Level:       levelNumber,
			Path:        strings.TrimSuffix(path, "/"),
			Created:     createdAt,
			LastUpdated: updatedAt,
		}

		if identifier.Valid {
			s := identifierForAPI(identifier.String)
			orgUnit.Identifier = &s
		}

		// Recursively get children
		children := buildOrgUnitTree(id)
		if len(children) > 0 {
			orgUnit.Children = make([]*OrgUnitRef, len(children))
			for i, child := range children {
				orgUnit.Children[i] = &OrgUnitRef{
					ID:     child.ID,
					MflUid: child.MflUid,
					Name:   child.Name,
				}
			}
			orgUnit.OrganisationUnits = orgUnit.Children
		}

		orgUnits = append(orgUnits, orgUnit)
	}

	return orgUnits
}
