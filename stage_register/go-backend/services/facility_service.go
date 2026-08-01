package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sort"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"
)

// FacilityUI represents a facility for UI display
type FacilityUI struct {
	Identifier string  `json:"identifier"`
	Name       string  `json:"name"`
	MflUid     *string `json:"mfl_uid"`
	Level      *string `json:"level"`
	Ownership  *string `json:"ownership"`
	Authority  *string `json:"authority"`
	Subcounty  *string `json:"subcounty"`
	District   *string `json:"district"`
	Region     *string `json:"region"`
}

// ListFacilitiesForUI lists facilities for UI with simple query
// userID, role, and districtID are optional - if provided, filters by user_id for public/initiator roles
// or by district for district_approver/district_initiator roles
func ListFacilitiesForUI(filters map[string]string, page, pageSize int, userID *int64, role *string, districtID *string) (map[string]interface{}, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	offset := (page - 1) * pageSize

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	params := []interface{}{}
	paramCount := 1

	if filters["q"] != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR identifier ILIKE $%d)", paramCount, paramCount)
		params = append(params, "%"+filters["q"]+"%")
		paramCount++
	}

	if filters["level"] != "" {
		whereClause += fmt.Sprintf(" AND level ILIKE $%d", paramCount)
		params = append(params, "%"+filters["level"]+"%")
		paramCount++
	}

	if filters["ownership"] != "" {
		whereClause += fmt.Sprintf(" AND ownership ILIKE $%d", paramCount)
		params = append(params, "%"+filters["ownership"]+"%")
		paramCount++
	}

	if filters["authority"] != "" {
		whereClause += fmt.Sprintf(" AND authority ILIKE $%d", paramCount)
		params = append(params, "%"+filters["authority"]+"%")
		paramCount++
	}

	if filters["region"] != "" {
		whereClause += fmt.Sprintf(" AND region ILIKE $%d", paramCount)
		params = append(params, "%"+filters["region"]+"%")
		paramCount++
	}

	if filters["district"] != "" {
		whereClause += fmt.Sprintf(" AND district ILIKE $%d", paramCount)
		params = append(params, "%"+filters["district"]+"%")
		paramCount++
	}

	if filters["subcounty"] != "" {
		whereClause += fmt.Sprintf(" AND subcounty ILIKE $%d", paramCount)
		params = append(params, "%"+filters["subcounty"]+"%")
		paramCount++
	}

	// Filter by user_id for public roles (district-level roles are handled by district filter below)
	if userID != nil && role != nil {
		if *role == "public" {
			whereClause += fmt.Sprintf(" AND user_id = $%d", paramCount)
			params = append(params, *userID)
			paramCount++
		}
	}

	// Filter by district for district-level roles
	if role != nil && districtID != nil {
		if *role == "district_approver" || *role == "district_initiator" || *role == "district" {
			// Get district name from admin_units table using district_id (mfl_uid)
			var districtName string
			err := configs.DB.QueryRow(
				`SELECT name FROM admin_units WHERE mfl_uid = $1`,
				*districtID,
			).Scan(&districtName)

			if err == nil && districtName != "" {
				whereClause += fmt.Sprintf(" AND district = $%d", paramCount)
				params = append(params, districtName)
				paramCount++
			}
		}
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM public.mfl_details %s", whereClause)
	var total int
	err := configs.DB.QueryRow(countQuery, params...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Data query
	query := fmt.Sprintf(`
		SELECT identifier, "name", mfl_uid, "level", ownership, authority, subcounty, district, region
		FROM public.mfl_details
		%s
		ORDER BY region, district, subcounty
		LIMIT $%d OFFSET $%d
	`, whereClause, paramCount, paramCount+1)

	params = append(params, pageSize, offset)

	rows, err := configs.DB.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facilities []*FacilityUI
	for rows.Next() {
		facility := &FacilityUI{}
		var mflUid, level, ownership, authority, subcounty, district, region sql.NullString

		err := rows.Scan(
			&facility.Identifier,
			&facility.Name,
			&mflUid,
			&level,
			&ownership,
			&authority,
			&subcounty,
			&district,
			&region,
		)

		if err != nil {
			continue
		}

		if mflUid.Valid {
			facility.MflUid = &mflUid.String
		}
		if level.Valid {
			facility.Level = &level.String
		}
		if ownership.Valid {
			facility.Ownership = &ownership.String
		}
		if authority.Valid {
			facility.Authority = &authority.String
		}
		if subcounty.Valid {
			facility.Subcounty = &subcounty.String
		}
		if district.Valid {
			facility.District = &district.String
		}
		if region.Valid {
			facility.Region = &region.String
		}

		facilities = append(facilities, facility)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"rows":     facilities,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	}, nil
}

// ExportFacility represents a facility for export
type ExportFacility struct {
	ID                  int64       `json:"id"`
	Identifier          string      `json:"identifier"`
	Name                string      `json:"name"`
	MflUid              *string     `json:"mfl_uid"`
	ShortName           *string     `json:"short_name"`
	HistoricalID        *string     `json:"historical_id"`
	Region              *string     `json:"region"`
	District            *string     `json:"district"`
	Subcounty           *string     `json:"subcounty"`
	Parish              *string     `json:"parish"`
	Village             *string     `json:"village"`
	Level               *string     `json:"level"`
	Ownership           *string     `json:"ownership"`
	Authority           *string     `json:"authority"`
	Status              *string     `json:"status"`
	Reporting           *bool       `json:"reporting"`
	Licensed            *bool       `json:"licensed"`
	Address             *string     `json:"address"`
	ContactPersonEmail  *string     `json:"contact_personemail"`
	ContactPersonMobile *string     `json:"contact_personmobile"`
	ContactPersonName   *string     `json:"contact_personname"`
	ContactPersonTitle  *string     `json:"contact_persontitle"`
	Longitude           *float64    `json:"longitude"`
	Latitude            *float64    `json:"latitude"`
	OpeningDate         *string     `json:"opening_date"`
	ClosingDate         *string     `json:"closing_date"`
	BedCapacity         *int64      `json:"bed_capacity"`
	Services            interface{} `json:"services"`
	CreatedAt           string      `json:"createdAt"`
	UpdatedAt           string      `json:"updatedAt"`
}

// ExportFacilitiesForUI exports all facilities for UI (no pagination, uses filters)
// userID, role, and districtID are optional - if provided, filters by user_id for public/initiator roles
// or by district for district_approver/district_initiator roles
func ExportFacilitiesForUI(filters map[string]string, userID *int64, role *string, districtID *string) ([]*ExportFacility, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	params := []interface{}{}
	paramCount := 1

	if filters["q"] != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR identifier ILIKE $%d)", paramCount, paramCount)
		params = append(params, "%"+filters["q"]+"%")
		paramCount++
	}

	if filters["level"] != "" {
		whereClause += fmt.Sprintf(" AND level ILIKE $%d", paramCount)
		params = append(params, "%"+filters["level"]+"%")
		paramCount++
	}

	if filters["ownership"] != "" {
		whereClause += fmt.Sprintf(" AND ownership ILIKE $%d", paramCount)
		params = append(params, "%"+filters["ownership"]+"%")
		paramCount++
	}

	if filters["authority"] != "" {
		whereClause += fmt.Sprintf(" AND authority ILIKE $%d", paramCount)
		params = append(params, "%"+filters["authority"]+"%")
		paramCount++
	}

	if filters["region"] != "" {
		whereClause += fmt.Sprintf(" AND region ILIKE $%d", paramCount)
		params = append(params, "%"+filters["region"]+"%")
		paramCount++
	}

	if filters["district"] != "" {
		whereClause += fmt.Sprintf(" AND district ILIKE $%d", paramCount)
		params = append(params, "%"+filters["district"]+"%")
		paramCount++
	}

	if filters["subcounty"] != "" {
		whereClause += fmt.Sprintf(" AND subcounty ILIKE $%d", paramCount)
		params = append(params, "%"+filters["subcounty"]+"%")
		paramCount++
	}

	// Filter by user_id for public roles (district_initiator is handled by district filter below)
	if userID != nil && role != nil {
		if *role == "public" {
			whereClause += fmt.Sprintf(" AND user_id = $%d", paramCount)
			params = append(params, *userID)
			paramCount++
		}
	}

	// Filter by district for district_approver and district_initiator roles
	if role != nil && districtID != nil {
		if *role == "district_approver" || *role == "district_initiator" {
			// Get district name from admin_units table using district_id (mfl_uid)
			var districtName string
			err := configs.DB.QueryRow(
				`SELECT name FROM admin_units WHERE mfl_uid = $1`,
				*districtID,
			).Scan(&districtName)

			if err == nil && districtName != "" {
				whereClause += fmt.Sprintf(" AND district = $%d", paramCount)
				params = append(params, districtName)
				paramCount++
			}
		}
	}

	// Data query - no pagination, returns all matching records
	query := fmt.Sprintf(`
		SELECT id, identifier, "name", mfl_uid, short_name, historical_id, 
			region, district, subcounty, parish, village, 
			"level", ownership, authority, 
			status, reporting, licensed, 
			address, contact_personemail, contact_personmobile, 
			contact_personname, contact_persontitle, 
			longitude, latitude, 
			opening_date, closing_date, 
			bed_capacity, services, 
			"createdAt", "updatedAt"
		FROM public.mfl_details
		%s
		ORDER BY region, district, subcounty
	`, whereClause)

	rows, err := configs.DB.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facilities []*ExportFacility
	for rows.Next() {
		facility := &ExportFacility{}
		var region, district, subcounty, parish, village, level, ownership, authority, mflUid sql.NullString
		var shortName, historicalID, status, address, contactPersonEmail, contactPersonMobile sql.NullString
		var contactPersonName, contactPersonTitle, openingDate, closingDate sql.NullString
		var servicesData []byte
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&facility.ID,
			&facility.Identifier,
			&facility.Name,
			&mflUid,
			&shortName,
			&historicalID,
			&region,
			&district,
			&subcounty,
			&parish,
			&village,
			&level,
			&ownership,
			&authority,
			&status,
			&facility.Reporting,
			&facility.Licensed,
			&address,
			&contactPersonEmail,
			&contactPersonMobile,
			&contactPersonName,
			&contactPersonTitle,
			&facility.Longitude,
			&facility.Latitude,
			&openingDate,
			&closingDate,
			&facility.BedCapacity,
			&servicesData,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			continue
		}

		// Set string fields
		if mflUid.Valid {
			facility.MflUid = &mflUid.String
		}
		if shortName.Valid {
			facility.ShortName = &shortName.String
		}
		if historicalID.Valid {
			facility.HistoricalID = &historicalID.String
		}
		if region.Valid {
			facility.Region = &region.String
		}
		if district.Valid {
			facility.District = &district.String
		}
		if subcounty.Valid {
			facility.Subcounty = &subcounty.String
		}
		if parish.Valid {
			facility.Parish = &parish.String
		}
		if village.Valid {
			facility.Village = &village.String
		}
		if level.Valid {
			facility.Level = &level.String
		}
		if ownership.Valid {
			facility.Ownership = &ownership.String
		}
		if authority.Valid {
			facility.Authority = &authority.String
		}
		if status.Valid {
			facility.Status = &status.String
		}
		if address.Valid {
			facility.Address = &address.String
		}
		if contactPersonEmail.Valid {
			facility.ContactPersonEmail = &contactPersonEmail.String
		}
		if contactPersonMobile.Valid {
			facility.ContactPersonMobile = &contactPersonMobile.String
		}
		if contactPersonName.Valid {
			facility.ContactPersonName = &contactPersonName.String
		}
		if contactPersonTitle.Valid {
			facility.ContactPersonTitle = &contactPersonTitle.String
		}
		if openingDate.Valid {
			facility.OpeningDate = &openingDate.String
		}
		if closingDate.Valid {
			facility.ClosingDate = &closingDate.String
		}

		// Parse services JSON
		if len(servicesData) > 0 {
			var services interface{}
			if err := json.Unmarshal(servicesData, &services); err == nil {
				facility.Services = services
			}
		}

		// Format dates
		if createdAt.Valid {
			facility.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z")
		}
		if updatedAt.Valid {
			facility.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z")
		}

		facilities = append(facilities, facility)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return facilities, nil
}

// ListFacilities lists facilities for API integration (uses mfl_list view)
func ListFacilities(filters map[string]string) (map[string]interface{}, error) {
	page, _ := strconv.Atoi(filters["page"])
	pageSize, _ := strconv.Atoi(filters["pageSize"])

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	offset := (page - 1) * pageSize

	params := []interface{}{}
	paramCount := 1

	whereClause := buildMflListFilters(filters, &params, &paramCount)

	query := fmt.Sprintf(`
		SELECT 
			identifier, name, mfl_uid, short_name, historical_id,
			national_uid, national_name,
			region_uid, region_name,
			district_uid, district_name,
			subcounty_uid, subcounty_name,
			parish_uid, parish_name,
			village_uid, village_name,
			facility_uid, facility_name,
			level AS level_mfl_uid, level_name,
			ownership AS ownership_mfl_uid, ownership_name,
			authority AS authority_mfl_uid, authority_name,
			status, reporting, licensed,
			address, contact_personemail, contact_personmobile, contact_personname, contact_persontitle,
			longitude, latitude,
			opening_date, closing_date,
			bed_capacity, services,
			"createdAt", "updatedAt"
		FROM mfl_list
		%s
		ORDER BY identifier ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, paramCount, paramCount+1)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM mfl_list %s", whereClause)

	params = append(params, pageSize, offset)

	// Count
	var total int
	err := configs.DB.QueryRow(countQuery, params[:len(params)-2]...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Data
	rows, err := configs.DB.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facilities []*models.Facility
	for rows.Next() {
		facility := &models.Facility{}
		err := scanMflListRow(rows, facility)
		if err != nil {
			continue
		}
		facilities = append(facilities, facility)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"rows":     facilities,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	}, nil
}

// GetFacility gets a facility by mfl_uid from mfl_details view
func GetFacility(mflUid string) (*models.Facility, error) {
	var facility models.Facility
	var region, district, subcounty, parish, village, level, ownership, authority sql.NullString

	err := configs.DB.QueryRow(`
		SELECT id, identifier, "name", mfl_uid, short_name, historical_id, user_id, 
			region, district, subcounty, parish, village, 
			"level", ownership, authority, 
			status, reporting, licensed, 
			address, contact_personemail, contact_personmobile, 
			contact_personname, contact_persontitle, 
			longitude, latitude, 
			opening_date, closing_date, 
			bed_capacity, services, 
			"createdAt", "updatedAt"
		FROM public.mfl_details
		WHERE mfl_uid = $1
		LIMIT 1
	`, mflUid).Scan(
		&facility.ID, &facility.Identifier, &facility.Name, &facility.MflUid,
		&facility.ShortName, &facility.HistoricalID, &facility.UserID,
		&region, &district, &subcounty, &parish, &village,
		&level, &ownership, &authority,
		&facility.Status, &facility.Reporting, &facility.Licensed,
		&facility.Address, &facility.ContactPersonEmail, &facility.ContactPersonMobile,
		&facility.ContactPersonName, &facility.ContactPersonTitle,
		&facility.Longitude, &facility.Latitude,
		&facility.OpeningDate, &facility.ClosingDate,
		&facility.BedCapacity, &facility.Services,
		&facility.CreatedAt, &facility.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Not found")
	}
	if err != nil {
		return nil, err
	}

	// Set string fields from NullString values
	if region.Valid {
		facility.Region = &region.String
	}
	if district.Valid {
		facility.District = &district.String
	}
	if subcounty.Valid {
		facility.Subcounty = &subcounty.String
	}
	if parish.Valid {
		parishName := parish.String
		facility.Parish = &models.NestedObject{Name: &parishName}
	}
	if village.Valid {
		villageName := village.String
		facility.Village = &models.NestedObject{Name: &villageName}
	}
	if level.Valid {
		levelStr := level.String
		facility.Level = &levelStr
		facility.LevelName = &levelStr
	}
	if ownership.Valid {
		ownershipStr := ownership.String
		facility.Ownership = &ownershipStr
		facility.OwnershipName = &ownershipStr
	}
	if authority.Valid {
		authorityStr := authority.String
		facility.Authority = &authorityStr
		facility.AuthorityName = &authorityStr
	}

	return &facility, nil
}

// GetFacilityOwnershipByLevel returns aggregated counts of facilities by level and ownership.
// Optional filters can be provided for region and district which will be applied to the query.
func GetFacilityOwnershipByLevel(filters map[string]string) ([]*models.FacilityLevelOwnershipStat, error) {
	// Build WHERE clause with optional filters
	whereClause := `WHERE status = 'Functional'`
	args := []interface{}{}
	paramCount := 1

	if filters != nil {
		if region, ok := filters["region"]; ok && region != "" {
			whereClause += fmt.Sprintf(" AND region ILIKE $%d", paramCount)
			args = append(args, "%"+region+"%")
			paramCount++
		}
		if district, ok := filters["district"]; ok && district != "" {
			whereClause += fmt.Sprintf(" AND district ILIKE $%d", paramCount)
			args = append(args, "%"+district+"%")
			paramCount++
		}
	}

	query := fmt.Sprintf(`
		SELECT
			"level",
			COUNT(*) FILTER (WHERE ownership = 'GOV') AS government,
			COUNT(*) FILTER (WHERE ownership = 'PFP') AS pfp,
			COUNT(*) FILTER (WHERE ownership = 'PNFP') AS pnfp,
			COUNT(*) AS total
		FROM public.mfl_details
		%s
		GROUP BY "level"
		ORDER BY "level";
	`, whereClause)

	rows, err := configs.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*models.FacilityLevelOwnershipStat
	for rows.Next() {
		stat := &models.FacilityLevelOwnershipStat{}
		if err := rows.Scan(&stat.Level, &stat.Government, &stat.Pfp, &stat.Pnfp, &stat.Total); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetFacilityOwnershipTotals returns overall ownership and total facility counts (for cards).
// Optional filters can be provided for region, district, and subcounty which will be applied to the query.
func GetFacilityOwnershipTotals(filters map[string]string) (*models.FacilityOwnershipTotals, error) {
	// Build WHERE clause with optional filters
	whereClause := `WHERE status = 'Functional'`
	args := []interface{}{}
	paramCount := 1

	if filters != nil {
		if region, ok := filters["region"]; ok && region != "" {
			whereClause += fmt.Sprintf(" AND region ILIKE $%d", paramCount)
			args = append(args, "%"+region+"%")
			paramCount++
		}
		if district, ok := filters["district"]; ok && district != "" {
			whereClause += fmt.Sprintf(" AND district ILIKE $%d", paramCount)
			args = append(args, "%"+district+"%")
			paramCount++
		}
		if subcounty, ok := filters["subcounty"]; ok && subcounty != "" {
			whereClause += fmt.Sprintf(" AND subcounty ILIKE $%d", paramCount)
			args = append(args, "%"+subcounty+"%")
			paramCount++
		}
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) FILTER (WHERE ownership = 'GOV')  AS government,
			COUNT(*) FILTER (WHERE ownership = 'PFP')  AS pfp,
			COUNT(*) FILTER (WHERE ownership = 'PNFP') AS pnfp,
			COUNT(*) AS total
		FROM public.mfl_details
		%s;
	`, whereClause)

	row := configs.DB.QueryRow(query, args...)

	totals := &models.FacilityOwnershipTotals{}
	if err := row.Scan(&totals.Government, &totals.Pfp, &totals.Pnfp, &totals.Total); err != nil {
		return nil, err
	}

	return totals, nil
}

// FacilityFilterHierarchyRow represents a single hierarchy combination for filters.
type FacilityFilterHierarchyRow struct {
	Region     string `json:"region"`
	District   string `json:"district"`
	DLG        string `json:"dlg"`
	Subcounty  string `json:"subcounty"`
}

// FacilityFilterOptions bundles all filter options and hierarchy for the public landing page.
type FacilityFilterOptions struct {
	Regions     []string                     `json:"regions"`
	Districts   []string                     `json:"districts"`
	DLGs        []string                     `json:"dlgs"`
	Subcounties []string                     `json:"subcounties"`
	Hierarchy   []FacilityFilterHierarchyRow `json:"hierarchy"`
}

// GetFacilityFilterOptions returns hierarchical filter options (region → district → DLG/municipality → subcounty)
// for facilities, based on the orgunits view (level 6 facilities).
func GetFacilityFilterOptions() (*FacilityFilterOptions, error) {
	rows, err := configs.DB.Query(`
		SELECT DISTINCT
			region_name,
			district_city_name,
			dlg_municipality_name,
			subcounty_division_name
		FROM orgunits
		WHERE level = 6
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	regionsSet := make(map[string]struct{})
	districtsSet := make(map[string]struct{})
	dlgsSet := make(map[string]struct{})
	subcountiesSet := make(map[string]struct{})

	var hierarchy []FacilityFilterHierarchyRow

	for rows.Next() {
		var region, district, dlg, subcounty sql.NullString
		if err := rows.Scan(&region, &district, &dlg, &subcounty); err != nil {
			return nil, err
		}

		row := FacilityFilterHierarchyRow{
			Region:    region.String,
			District:  district.String,
			DLG:       dlg.String,
			Subcounty: subcounty.String,
		}

		// Skip rows that don't have at least region and district
		if row.Region == "" || row.District == "" {
			continue
		}

		hierarchy = append(hierarchy, row)

		regionsSet[row.Region] = struct{}{}
		districtsSet[row.District] = struct{}{}
		if row.DLG != "" {
			dlgsSet[row.DLG] = struct{}{}
		}
		if row.Subcounty != "" {
			subcountiesSet[row.Subcounty] = struct{}{}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	regions := make([]string, 0, len(regionsSet))
	for r := range regionsSet {
		regions = append(regions, r)
	}
	sort.Strings(regions)

	districts := make([]string, 0, len(districtsSet))
	for d := range districtsSet {
		districts = append(districts, d)
	}
	sort.Strings(districts)

	dlgs := make([]string, 0, len(dlgsSet))
	for d := range dlgsSet {
		dlgs = append(dlgs, d)
	}
	sort.Strings(dlgs)

	subcounties := make([]string, 0, len(subcountiesSet))
	for s := range subcountiesSet {
		subcounties = append(subcounties, s)
	}
	sort.Strings(subcounties)

	return &FacilityFilterOptions{
		Regions:     regions,
		Districts:   districts,
		DLGs:        dlgs,
		Subcounties: subcounties,
		Hierarchy:   hierarchy,
	}, nil
}

// GetFacilityDistributionByOwnership returns distribution of facilities by ownership
func GetFacilityDistributionByOwnership() ([]*models.FacilityDistributionStat, error) {
	rows, err := configs.DB.Query(`
		SELECT 
			COALESCE(ownership, 'Unknown') as name,
			COUNT(*) as count
		FROM public.mfl_details
		GROUP BY ownership
		ORDER BY count DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*models.FacilityDistributionStat
	for rows.Next() {
		stat := &models.FacilityDistributionStat{}
		if err := rows.Scan(&stat.Name, &stat.Count); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetFacilityDistributionByLevel returns distribution of facilities by level
func GetFacilityDistributionByLevel() ([]*models.FacilityDistributionStat, error) {
	rows, err := configs.DB.Query(`
		SELECT 
			COALESCE("level", 'Unknown') as name,
			COUNT(*) as count
		FROM public.mfl_details
		GROUP BY "level"
		ORDER BY count DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*models.FacilityDistributionStat
	for rows.Next() {
		stat := &models.FacilityDistributionStat{}
		if err := rows.Scan(&stat.Name, &stat.Count); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetFacilityDistributionByAuthority returns distribution of facilities by authority
func GetFacilityDistributionByAuthority() ([]*models.FacilityDistributionStat, error) {
	rows, err := configs.DB.Query(`
		SELECT 
			COALESCE(authority, 'Unknown') as name,
			COUNT(*) as count
		FROM public.mfl_details
		GROUP BY authority
		ORDER BY count DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*models.FacilityDistributionStat
	for rows.Next() {
		stat := &models.FacilityDistributionStat{}
		if err := rows.Scan(&stat.Name, &stat.Count); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

// CreateFacility creates a new facility
func CreateFacility(facility *models.Facility) (*models.Facility, error) {
	tx, err := configs.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get the Facility level_id from admin_level
	var facilityLevelID int64
	err = tx.QueryRow(
		`SELECT id FROM admin_level 
		 WHERE name ILIKE 'facility' 
		 OR level_number = (SELECT MAX(level_number) FROM admin_level) 
		 ORDER BY level_number DESC LIMIT 1`,
	).Scan(&facilityLevelID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Facility level not found in admin_level table")
	}
	if err != nil {
		return nil, err
	}

	// Get parent admin_unit details (the selected admin unit)
	var parentID *int64
	var parentPath string = "/"
	if facility.AdminUnitID != nil {
		parentIDInt := *facility.AdminUnitID
		parentID = &parentIDInt

		var parentMflUid sql.NullString
		err = tx.QueryRow(
			"SELECT id, mfl_uid, path FROM admin_units WHERE id = $1",
			*parentID,
		).Scan(&parentID, &parentMflUid, &parentPath)
		if err == nil && parentPath == "" {
			parentPath = "/"
		}
		if err != nil {
			return nil, fmt.Errorf("Parent admin unit not found: %v", err)
		}
	}

	// Generate unique mfl_uid for facility admin_unit
	uid, err := utils.EnsureUniqueMflUid(configs.DB, "admin_units")
	if err != nil {
		return nil, fmt.Errorf("Failed to generate unique mfl_uid for facility")
	}

	// Compute path (no trailing slash)
	parentTrimmed := strings.TrimSuffix(parentPath, "/")
	if parentTrimmed == "" {
		parentTrimmed = "/"
	}
	facilityPath := parentTrimmed + "/" + uid

	// Use the provided name for admin_units table
	identifier := utils.GenerateFacilityIdentifier()
	facilityName := facility.Name
	if facilityName == "" {
		// Fallback to short_name if name is not provided
		if facility.ShortName != nil && *facility.ShortName != "" {
			facilityName = *facility.ShortName
		} else {
			facilityName = identifier
		}
	}

	// Create admin_unit entry for the facility
	var facilityAdminUnitID int64
	err = tx.QueryRow(
		`INSERT INTO admin_units (name, code, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		 RETURNING id`,
		facilityName, nil, uid, facilityLevelID, parentID, facilityPath,
	).Scan(&facilityAdminUnitID)

	if err != nil {
		return nil, err
	}

	servicesJSON, _ := json.Marshal(facility.Services)

	// Create facility entry - use the newly created facility admin_unit ID
	var facilityID int64
	err = tx.QueryRow(
		`INSERT INTO facilities (
			name, identifier, short_name, historical_id, admin_unit_id, mfl_uid,
			level, ownership, authority,
			status, reporting, licensed, address,
			contact_personemail, contact_personmobile, contact_personname, contact_persontitle,
			longitude, latitude, opening_date, closing_date, bed_capacity, services
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		RETURNING id`,
		facilityName, identifier, facility.ShortName, facility.HistoricalID,
		&facilityAdminUnitID, uid, // Use the newly created facility admin_unit ID
		facility.Level, facility.Ownership, facility.Authority,
		facility.Status, facility.Reporting, facility.Licensed, facility.Address,
		facility.ContactPersonEmail, facility.ContactPersonMobile,
		facility.ContactPersonName, facility.ContactPersonTitle,
		facility.Longitude, facility.Latitude,
		facility.OpeningDate, facility.ClosingDate,
		facility.BedCapacity, servicesJSON,
	).Scan(&facilityID)

	if err != nil {
		return nil, err
	}

	var mflUid string
	err = tx.QueryRow("SELECT mfl_uid FROM facilities WHERE id = $1 LIMIT 1", facilityID).Scan(&mflUid)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Fetch with related data from mfl_details view
	return GetFacility(mflUid)
}

// UploadFacilityRow represents one row from the facilities upload (Excel/CSV)
type UploadFacilityRow struct {
	Name                string `json:"name"`
	OrganisationUnitID  string `json:"organisation_unit_id"`
}

// CreatedFacilityRow is one successfully created facility returned from upload
type CreatedFacilityRow struct {
	Identifier string `json:"identifier"`
	MflUid     string `json:"mfl_uid"`
	Name       string `json:"name"`
}

// UploadFacilitiesResult is the result of a bulk facility upload
type UploadFacilitiesResult struct {
	Total       int                   `json:"total"`
	Created     int                   `json:"created"`
	Failed      int                   `json:"failed"`
	Errors      []UploadFacilityError `json:"errors,omitempty"`
	CreatedRows []CreatedFacilityRow  `json:"created_rows,omitempty"`
}

// UploadFacilityError describes a single row failure
type UploadFacilityError struct {
	Row   int    `json:"row"`
	Name  string `json:"name,omitempty"`
	Error string `json:"error"`
}

// UploadFacilities processes bulk facility upload: for each row assigns a unique mfl_uid (uses provided if not yet assigned) and creates admin_unit + facility
func UploadFacilities(rows []UploadFacilityRow) (*UploadFacilitiesResult, error) {
	result := &UploadFacilitiesResult{
		Total:       len(rows),
		Errors:      []UploadFacilityError{},
		CreatedRows: []CreatedFacilityRow{},
	}

	var facilityLevelID int64
	err := configs.DB.QueryRow(
		`SELECT id FROM admin_level 
		 WHERE name ILIKE 'facility' 
		 OR level_number = (SELECT MAX(level_number) FROM admin_level) 
		 ORDER BY level_number DESC LIMIT 1`,
	).Scan(&facilityLevelID)
	if err == sql.ErrNoRows {
		return result, fmt.Errorf("facility level not found in admin_level table")
	}
	if err != nil {
		return result, err
	}

	parentID := (*int64)(nil)
	parentPath := "/"

	for i, row := range rows {
		rowNum := i + 2 // 1-based, +1 for header
		name := strings.TrimSpace(row.Name)
		if name == "" {
			result.Failed++
			result.Errors = append(result.Errors, UploadFacilityError{Row: rowNum, Name: row.Name, Error: "name is required"})
			continue
		}

		mflUid, err := utils.ResolveUniqueMflUid(configs.DB, strings.TrimSpace(row.OrganisationUnitID))
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, UploadFacilityError{Row: rowNum, Name: name, Error: err.Error()})
			continue
		}

		parentTrimmed := strings.TrimSuffix(parentPath, "/")
		if parentTrimmed == "" {
			parentTrimmed = "/"
		}
		facilityPath := parentTrimmed + "/" + mflUid

		var facilityAdminUnitID int64
		err = configs.DB.QueryRow(
			`INSERT INTO admin_units (name, code, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
			 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			 RETURNING id`,
			name, nil, mflUid, facilityLevelID, parentID, facilityPath,
		).Scan(&facilityAdminUnitID)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, UploadFacilityError{Row: rowNum, Name: name, Error: err.Error()})
			continue
		}

		identifier := utils.GenerateFacilityIdentifier()
		_, err = configs.DB.Exec(
			`INSERT INTO facilities (
				identifier, mfl_uid, name, short_name, historical_id, admin_unit_id, level, ownership, authority,
				status, reporting, licensed, address, contact_personemail, contact_personmobile,
				contact_personname, contact_persontitle, longitude, latitude, opening_date,
				closing_date, bed_capacity, services, user_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)`,
			identifier, mflUid, name, nil, nil, facilityAdminUnitID, nil, nil, nil,
			"Functional", true, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, UploadFacilityError{Row: rowNum, Name: name, Error: err.Error()})
			// best-effort: remove the admin_unit we just created to avoid orphan
			_, _ = configs.DB.Exec("DELETE FROM admin_units WHERE id = $1", facilityAdminUnitID)
			continue
		}

		result.Created++
		result.CreatedRows = append(result.CreatedRows, CreatedFacilityRow{
			Identifier: identifier,
			MflUid:     mflUid,
			Name:       name,
		})
	}

	return result, nil
}

// UpdateFacility updates a facility
func UpdateFacility(id string, facility *models.Facility) (*models.Facility, error) {
	// Resolve facility ID
	facilityID, err := resolveFacilityID(id)
	if err != nil {
		return nil, fmt.Errorf("Not found")
	}

	tx, err := configs.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	updateFields := []string{}
	params := []interface{}{}
	paramCount := 1

	addField := func(field string, value interface{}) {
		if value != nil {
			updateFields = append(updateFields, fmt.Sprintf("%s = $%d", field, paramCount))
			params = append(params, value)
			paramCount++
		}
	}

	addField("name", facility.Name)
	addField("short_name", facility.ShortName)
	addField("status", facility.Status)
	addField("reporting", facility.Reporting)
	addField("licensed", facility.Licensed)
	addField("address", facility.Address)
	addField("contact_personemail", facility.ContactPersonEmail)
	addField("contact_personmobile", facility.ContactPersonMobile)
	addField("contact_personname", facility.ContactPersonName)
	addField("contact_persontitle", facility.ContactPersonTitle)
	addField("longitude", facility.Longitude)
	addField("latitude", facility.Latitude)
	addField("opening_date", facility.OpeningDate)
	addField("closing_date", facility.ClosingDate)
	addField("bed_capacity", facility.BedCapacity)
	addField("level", facility.Level)
	addField("ownership", facility.Ownership)
	addField("authority", facility.Authority)

	if facility.Services != nil {
		servicesJSON, _ := json.Marshal(facility.Services)
		addField("services", servicesJSON)
	}

	if facility.AdminUnitID != nil {
		addField("admin_unit_id", *facility.AdminUnitID)
		var mflUid sql.NullString
		err = tx.QueryRow("SELECT mfl_uid FROM admin_units WHERE id = $1 LIMIT 1", *facility.AdminUnitID).Scan(&mflUid)
		if err == nil && mflUid.Valid {
			addField("mfl_uid", mflUid.String)
		}
	}

	if len(updateFields) > 0 {
		updateFields = append(updateFields, "\"updatedAt\" = NOW()")
		params = append(params, facilityID)
		_, err = tx.Exec(
			fmt.Sprintf("UPDATE facilities SET %s WHERE id = $%d", strings.Join(updateFields, ", "), paramCount),
			params...,
		)
		if err != nil {
			return nil, err
		}
	}

	var mflUid string
	err = tx.QueryRow("SELECT mfl_uid FROM facilities WHERE id = $1 LIMIT 1", facilityID).Scan(&mflUid)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Fetch with related data from mfl_details view
	return GetFacility(mflUid)
}

// DeleteFacility deletes a facility
func DeleteFacility(id string) error {
	facilityID, err := resolveFacilityID(id)
	if err != nil {
		return fmt.Errorf("Not found")
	}

	tx, err := configs.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM facilities WHERE id = $1", facilityID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Helper functions
func resolveFacilityID(id string) (int64, error) {
	if idInt, err := strconv.ParseInt(id, 10, 64); err == nil {
		var exists int64
		err = configs.DB.QueryRow("SELECT id FROM facilities WHERE id = $1 LIMIT 1", idInt).Scan(&exists)
		if err == nil {
			return idInt, nil
		}
	}

	var facilityID int64
	err := configs.DB.QueryRow(
		"SELECT id FROM facilities WHERE identifier = $1 OR mfl_uid = $2 LIMIT 1",
		id, id,
	).Scan(&facilityID)

	return facilityID, err
}

func buildMflListFilters(filters map[string]string, params *[]interface{}, paramCount *int) string {
	whereClause := "WHERE 1=1"

	addFilter := func(field, value string, operator string) {
		if value != "" {
			whereClause += fmt.Sprintf(" AND %s %s $%d", field, operator, *paramCount)
			if operator == "ILIKE" {
				*params = append(*params, "%"+value+"%")
			} else {
				*params = append(*params, value)
			}
			*paramCount++
		}
	}

	addFilter("national_name", filters["national"], "ILIKE")
	addFilter("region_name", filters["region"], "ILIKE")
	addFilter("district_name", filters["district"], "ILIKE")
	addFilter("subcounty_name", filters["subcounty"], "ILIKE")
	addFilter("parish_name", filters["parish"], "ILIKE")
	addFilter("village_name", filters["village"], "ILIKE")
	addFilter("facility_name", filters["facility"], "ILIKE")
	addFilter("level_name", filters["level"], "ILIKE")
	addFilter("ownership_name", filters["ownership"], "ILIKE")
	addFilter("authority_name", filters["authority"], "ILIKE")
	addFilter("status", filters["status"], "=")

	if filters["q"] != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR historical_id ILIKE $%d OR identifier ILIKE $%d)", *paramCount, *paramCount, *paramCount)
		*params = append(*params, "%"+filters["q"]+"%")
		*paramCount++
	}

	if filters["last_updated"] != "" || filters["updatedSince"] != "" {
		dateFilter := filters["last_updated"]
		if dateFilter == "" {
			dateFilter = filters["updatedSince"]
		}
		whereClause += fmt.Sprintf(" AND \"updatedAt\"::date >= $%d::date", *paramCount)
		*params = append(*params, dateFilter)
		*paramCount++
	}

	return whereClause
}

func buildMflDetailsFilters(filters map[string]string, params *[]interface{}, paramCount *int) string {
	whereClause := "WHERE 1=1"

	addFilter := func(field, value string, operator string) {
		if value != "" {
			whereClause += fmt.Sprintf(" AND %s %s $%d", field, operator, *paramCount)
			if operator == "ILIKE" {
				*params = append(*params, "%"+value+"%")
			} else {
				*params = append(*params, value)
			}
			*paramCount++
		}
	}

	addFilter("region", filters["region"], "ILIKE")
	addFilter("district", filters["district"], "ILIKE")
	addFilter("subcounty", filters["subcounty"], "ILIKE")
	addFilter("parish", filters["parish"], "ILIKE")
	addFilter("village", filters["village"], "ILIKE")
	addFilter("level", filters["level"], "ILIKE")
	addFilter("ownership", filters["ownership"], "ILIKE")
	addFilter("authority", filters["authority"], "ILIKE")
	addFilter("status", filters["status"], "=")

	if filters["reporting"] == "true" || filters["reporting"] == "false" {
		whereClause += fmt.Sprintf(" AND reporting = $%d", *paramCount)
		*params = append(*params, filters["reporting"] == "true")
		*paramCount++
	}

	if filters["q"] != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR historical_id ILIKE $%d OR identifier ILIKE $%d)", *paramCount, *paramCount, *paramCount)
		*params = append(*params, "%"+filters["q"]+"%")
		*paramCount++
	}

	if filters["last_updated"] != "" || filters["updatedSince"] != "" {
		dateFilter := filters["last_updated"]
		if dateFilter == "" {
			dateFilter = filters["updatedSince"]
		}
		whereClause += fmt.Sprintf(" AND \"updatedAt\"::date >= $%d::date", *paramCount)
		*params = append(*params, dateFilter)
		*paramCount++
	}

	return whereClause
}

func scanMflListRow(rows *sql.Rows, facility *models.Facility) error {
	var nationalUid, nationalName, regionUid, regionName, districtUid, districtName,
		subcountyUid, subcountyName, parishUid, parishName, villageUid, villageName,
		facilityUid, facilityName, levelUid, levelName, ownershipUid, ownershipName,
		authorityUid, authorityName sql.NullString

	err := rows.Scan(
		&facility.Identifier, &facility.Name, &facility.MflUid,
		&facility.ShortName, &facility.HistoricalID,
		&nationalUid, &nationalName, &regionUid, &regionName,
		&districtUid, &districtName, &subcountyUid, &subcountyName,
		&parishUid, &parishName, &villageUid, &villageName,
		&facilityUid, &facilityName,
		&levelUid, &levelName,
		&ownershipUid, &ownershipName,
		&authorityUid, &authorityName,
		&facility.Status, &facility.Reporting, &facility.Licensed,
		&facility.Address, &facility.ContactPersonEmail, &facility.ContactPersonMobile,
		&facility.ContactPersonName, &facility.ContactPersonTitle,
		&facility.Longitude, &facility.Latitude,
		&facility.OpeningDate, &facility.ClosingDate,
		&facility.BedCapacity, &facility.Services,
		&facility.CreatedAt, &facility.UpdatedAt,
	)

	if err != nil {
		return err
	}

	if nationalUid.Valid || nationalName.Valid {
		nationalUidStr := nationalUid.String
		nationalNameStr := nationalName.String
		facility.National = &models.NestedObject{MflUid: &nationalUidStr, Name: &nationalNameStr}
	}
	if regionUid.Valid || regionName.Valid {
		regionUidStr := regionUid.String
		regionNameStr := regionName.String
		facility.RegionObj = &models.NestedObject{MflUid: &regionUidStr, Name: &regionNameStr}
	}
	if districtUid.Valid || districtName.Valid {
		districtUidStr := districtUid.String
		districtNameStr := districtName.String
		facility.DistrictObj = &models.NestedObject{MflUid: &districtUidStr, Name: &districtNameStr}
	}
	if subcountyUid.Valid || subcountyName.Valid {
		subcountyUidStr := subcountyUid.String
		subcountyNameStr := subcountyName.String
		facility.SubcountyObj = &models.NestedObject{MflUid: &subcountyUidStr, Name: &subcountyNameStr}
	}
	if parishUid.Valid || parishName.Valid {
		parishUidStr := parishUid.String
		parishNameStr := parishName.String
		facility.Parish = &models.NestedObject{MflUid: &parishUidStr, Name: &parishNameStr}
	}
	if villageUid.Valid || villageName.Valid {
		villageUidStr := villageUid.String
		villageNameStr := villageName.String
		facility.Village = &models.NestedObject{MflUid: &villageUidStr, Name: &villageNameStr}
	}
	if facilityUid.Valid || facilityName.Valid {
		facilityUidStr := facilityUid.String
		facilityNameStr := facilityName.String
		facility.Facility = &models.NestedObject{MflUid: &facilityUidStr, Name: &facilityNameStr}
	}
	if levelUid.Valid || levelName.Valid {
		levelUidStr := levelUid.String
		levelNameStr := levelName.String
		facility.LevelObj = &models.NestedObject{MflUid: &levelUidStr, Name: &levelNameStr}
	}
	if ownershipUid.Valid || ownershipName.Valid {
		ownershipUidStr := ownershipUid.String
		ownershipNameStr := ownershipName.String
		facility.OwnershipObj = &models.NestedObject{MflUid: &ownershipUidStr, Name: &ownershipNameStr}
	}
	if authorityUid.Valid || authorityName.Valid {
		authorityUidStr := authorityUid.String
		authorityNameStr := authorityName.String
		facility.AuthorityObj = &models.NestedObject{MflUid: &authorityUidStr, Name: &authorityNameStr}
	}

	return nil
}

func scanMflDetailsRow(rows *sql.Rows, facility *models.Facility) error {
	var region, district, subcounty, parish, village, level, ownership, authority sql.NullString

	err := rows.Scan(
		&facility.ID, &facility.Identifier, &facility.MflUid, &facility.Name,
		&facility.ShortName, &facility.HistoricalID, &facility.UserID,
		&region, &district, &subcounty, &parish, &village,
		&level, &ownership, &authority,
		&facility.Status, &facility.Reporting, &facility.Licensed,
		&facility.Address, &facility.ContactPersonEmail, &facility.ContactPersonMobile,
		&facility.ContactPersonName, &facility.ContactPersonTitle,
		&facility.Longitude, &facility.Latitude,
		&facility.OpeningDate, &facility.ClosingDate,
		&facility.BedCapacity, &facility.Services,
		&facility.CreatedAt, &facility.UpdatedAt,
	)

	if err != nil {
		return err
	}

	if region.Valid {
		regionName := region.String
		facility.Region = &regionName
	}
	if district.Valid {
		districtName := district.String
		facility.District = &districtName
	}
	if subcounty.Valid {
		subcountyName := subcounty.String
		facility.Subcounty = &subcountyName
	}
	if parish.Valid {
		parishName := parish.String
		facility.Parish = &models.NestedObject{Name: &parishName}
	}
	if village.Valid {
		villageName := village.String
		facility.Village = &models.NestedObject{Name: &villageName}
	}
	if level.Valid {
		levelStr := level.String
		facility.Level = &levelStr
		facility.LevelName = &levelStr
	}
	if ownership.Valid {
		ownershipStr := ownership.String
		facility.Ownership = &ownershipStr
		facility.OwnershipName = &ownershipStr
	}
	if authority.Valid {
		authorityStr := authority.String
		facility.Authority = &authorityStr
		facility.AuthorityName = &authorityStr
	}

	return nil
}
