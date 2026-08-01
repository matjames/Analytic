package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"go-backend/configs"
	"go-backend/models"
)

// BuildRoleBasedWhereClause builds WHERE clause based on user role
func BuildRoleBasedWhereClause(userID int64, role string, districtID *string, params []interface{}, paramCount int) (string, []interface{}, int) {
	whereClause := "WHERE 1=1"

	if role == "public" || role == "district_initiator" {
		whereClause += fmt.Sprintf(" AND fr.initiated_by = $%d", paramCount)
		params = append(params, userID)
		paramCount++
	} else if role == "district_approver" {
		if districtID == nil {
			return "WHERE 1=0", params, paramCount
		}

		whereClause += fmt.Sprintf(` AND (
			(
				fr.current_stage = 'district_approver' 
				AND EXISTS (
					SELECT 1 FROM users u 
					WHERE u.id = fr.initiated_by 
					AND u.district_id = $%d
				)
			)
			OR (
				EXISTS (
					SELECT 1 FROM facility_request_approvals fra 
					WHERE fra.request_id = fr.id 
					AND fra.stage = 'district_approver' 
					AND fra.approver_id = $%d
					AND fra.action = 'approved'
				)
				AND EXISTS (
					SELECT 1 FROM users u 
					WHERE u.id = fr.initiated_by 
					AND u.district_id = $%d
				)
			)
		)`, paramCount, paramCount+1, paramCount)
		params = append(params, *districtID, userID)
		paramCount += 2
	} else if role == "district" {
		// Unified district role: can see its own initiated requests AND district-queue requests
		if districtID == nil {
			return "WHERE 1=0", params, paramCount
		}

		whereClause += fmt.Sprintf(` AND (
			fr.initiated_by = $%d
			OR (
				(
					fr.current_stage = 'district_approver' 
					AND EXISTS (
						SELECT 1 FROM users u 
						WHERE u.id = fr.initiated_by 
						AND u.district_id = $%d
					)
				)
				OR (
					EXISTS (
						SELECT 1 FROM facility_request_approvals fra 
						WHERE fra.request_id = fr.id 
						AND fra.stage = 'district_approver' 
						AND fra.approver_id = $%d
						AND fra.action = 'approved'
					)
					AND EXISTS (
						SELECT 1 FROM users u 
						WHERE u.id = fr.initiated_by 
						AND u.district_id = $%d
					)
				)
			)
		)`, paramCount, paramCount+1, paramCount+2, paramCount+1)
		params = append(params, userID, *districtID, userID)
		paramCount += 3
	} else if role == "moh_clinical" {
		whereClause += fmt.Sprintf(` AND (
			fr.current_stage = 'moh_clinical' 
			OR EXISTS (
				SELECT 1 FROM facility_request_approvals fra 
				WHERE fra.request_id = fr.id 
				AND fra.stage = 'moh_clinical' 
				AND fra.approver_id = $%d
				AND fra.action = 'approved'
			)
		)`, paramCount)
		params = append(params, userID)
		paramCount++
	} else if role == "moh_publisher" {
		whereClause += fmt.Sprintf(` AND (
			fr.current_stage = 'moh_publisher' 
			OR EXISTS (
				SELECT 1 FROM facility_request_approvals fra 
				WHERE fra.request_id = fr.id 
				AND fra.stage = 'moh_publisher' 
				AND fra.approver_id = $%d
				AND fra.action = 'approved'
			)
		)`, paramCount)
		params = append(params, userID)
		paramCount++
	} else if role != "admin" {
		return "WHERE 1=0", params, paramCount
	}

	return whereClause, params, paramCount
}

// ValidateRequestType validates request type
func ValidateRequestType(requestType string) error {
	validTypes := []string{"new_addition", "update", "deactivation"}
	for _, t := range validTypes {
		if t == requestType {
			return nil
		}
	}
	return fmt.Errorf("request_type must be one of: new_addition, update, deactivation")
}

// ValidateInitiatorRole validates user role for initiating requests
func ValidateInitiatorRole(role string) error {
	if role != "public" && role != "district_initiator" && role != "district" {
		return fmt.Errorf("Only users with public, district_initiator or district role can initiate requests")
	}
	return nil
}

// ValidateRequestCreation validates request creation data
func ValidateRequestCreation(requestType string, facilityID *int64, facilityData interface{}, files []map[string]interface{}) error {
	if err := ValidateRequestType(requestType); err != nil {
		return err
	}

	if (requestType == "update" || requestType == "deactivation") && facilityID == nil {
		return fmt.Errorf("facility_id is required for update and deactivation requests")
	}

	if (requestType == "new_addition" || requestType == "update") && facilityData == nil {
		return fmt.Errorf("facility_data is required for new_addition and update requests")
	}

	if (requestType == "update" || requestType == "deactivation") && len(files) == 0 {
		return fmt.Errorf("Supporting documents are required for update and deactivation requests")
	}

	return nil
}

// CreateRequest creates a new facility request
func CreateRequest(userID int64, role string, requestType string, facilityID *int64, facilityData interface{}, files []map[string]interface{}) (*models.Request, error) {
	// Validate user role
	if err := ValidateInitiatorRole(role); err != nil {
		return nil, err
	}

	// Get user's district
	var districtID sql.NullString
	err := configs.DB.QueryRow("SELECT district_id FROM users WHERE id = $1", userID).Scan(&districtID)
	if err != nil || !districtID.Valid {
		return nil, fmt.Errorf("User district not found. Please contact administrator.")
	}

	// Validate request data
	if err := ValidateRequestCreation(requestType, facilityID, facilityData, files); err != nil {
		return nil, err
	}

	// Parse facility data
	var facilityDataJSON []byte
	if facilityData != nil {
		facilityDataJSON, err = json.Marshal(facilityData)
		if err != nil {
			return nil, fmt.Errorf("Invalid facility_data: %v", err)
		}
	}

	// Verify facility exists for update/deactivation
	if requestType == "update" || requestType == "deactivation" {
		var exists int64
		err = configs.DB.QueryRow("SELECT id FROM facilities WHERE id = $1", *facilityID).Scan(&exists)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("Facility not found")
		}
		if err != nil {
			return nil, err
		}
	}

	// Begin transaction
	tx, err := configs.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create the request
	var facilityIDValue interface{}
	if requestType == "new_addition" {
		facilityIDValue = nil
	} else {
		facilityIDValue = *facilityID
	}

	var req models.Request
	err = tx.QueryRow(
		`INSERT INTO facility_requests 
		 (request_type, facility_id, facility_data, initiated_by, current_status, current_stage)
		 VALUES ($1, $2, $3, $4, 'pending', 'district_approver')
		 RETURNING id, request_type, facility_id, facility_data, current_status, current_stage, 
		           initiated_by, district_approver_id, moh_clinical_id, moh_publisher_id, 
		           rejection_reason, "createdAt", "updatedAt"`,
		requestType, facilityIDValue, facilityDataJSON, userID,
	).Scan(
		&req.ID, &req.RequestType, &req.FacilityID, &req.FacilityData,
		&req.CurrentStatus, &req.CurrentStage, &req.InitiatedBy,
		&req.DistrictApproverID, &req.MohClinicalID, &req.MohPublisherID,
		&req.RejectionReason, &req.CreatedAt, &req.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Handle uploaded documents
	if files != nil && len(files) > 0 {
		for _, file := range files {
			fileSize, _ := file["file_size"].(int64)
			_, err = tx.Exec(
				`INSERT INTO facility_request_documents 
				 (request_id, filename, original_filename, file_path, file_size, mime_type, doc_type)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				req.ID,
				file["filename"],
				file["original_filename"],
				file["file_path"],
				fileSize,
				file["mime_type"],
				file["doc_type"],
			)
			if err != nil {
				return nil, err
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Fetch documents
	docRows, err := configs.DB.Query(
		"SELECT id, request_id, filename, original_filename, file_path, file_size, mime_type, doc_type, \"createdAt\" FROM facility_request_documents WHERE request_id = $1",
		req.ID,
	)
	if err == nil {
		defer docRows.Close()
		for docRows.Next() {
			var doc models.RequestDocument
			docRows.Scan(&doc.ID, &doc.RequestID, &doc.Filename, &doc.OriginalFilename, &doc.FilePath, &doc.FileSize, &doc.MimeType, &doc.DocType, &doc.CreatedAt)
			req.Documents = append(req.Documents, doc)
		}
	}

	return &req, nil
}

// ListRequests lists requests with pagination and filtering
func ListRequests(userID int64, role string, districtID *string, filters map[string]string) (map[string]interface{}, error) {
	status := filters["status"]
	stage := filters["stage"]
	requestType := filters["request_type"]
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

	// Build role-based filtering
	whereClause, params, paramCount := BuildRoleBasedWhereClause(userID, role, districtID, params, paramCount)

	// Additional filters
	if status != "" {
		whereClause += fmt.Sprintf(" AND fr.current_status = $%d", paramCount)
		params = append(params, status)
		paramCount++
	}
	if stage != "" {
		whereClause += fmt.Sprintf(" AND fr.current_stage = $%d", paramCount)
		params = append(params, stage)
		paramCount++
	}
	if requestType != "" {
		whereClause += fmt.Sprintf(" AND fr.request_type = $%d", paramCount)
		params = append(params, requestType)
		paramCount++
	}

	// Count query
	var total int
	countQuery := "SELECT COUNT(*) FROM facility_requests fr " + whereClause
	err := configs.DB.QueryRow(countQuery, params...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Data query
	params = append(params, pageSize, offset)
	dataQuery := fmt.Sprintf(`
		SELECT 
			fr.id, fr.request_type, fr.facility_id, fr.facility_data, fr.current_status, fr.current_stage,
			fr.initiated_by, fr.district_approver_id, fr.moh_clinical_id, fr.moh_publisher_id,
			fr.rejection_reason, fr."createdAt", fr."updatedAt",
			u1.username as initiated_by_name,
			u1.email as initiated_by_email,
			f.mfl_uid as facility_mfl_uid
		FROM facility_requests fr
		LEFT JOIN users u1 ON fr.initiated_by = u1.id
		LEFT JOIN facilities f ON fr.facility_id = f.id
		%s
		ORDER BY fr."createdAt" DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, paramCount, paramCount+1)

	rows, err := configs.DB.Query(dataQuery, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*models.Request
	for rows.Next() {
		var req models.Request
		var facilityMflUid sql.NullString
		err := rows.Scan(
			&req.ID, &req.RequestType, &req.FacilityID, &req.FacilityData,
			&req.CurrentStatus, &req.CurrentStage, &req.InitiatedBy,
			&req.DistrictApproverID, &req.MohClinicalID, &req.MohPublisherID,
			&req.RejectionReason, &req.CreatedAt, &req.UpdatedAt,
			&req.InitiatedByName, &req.InitiatedByEmail,
			&facilityMflUid,
		)
		if facilityMflUid.Valid {
			req.FacilityMflUid = &facilityMflUid.String
		}
		if err != nil {
			return nil, err
		}
		requests = append(requests, &req)
	}

	return map[string]interface{}{
		"rows":     requests,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	}, nil
}

// GetRequestStats gets aggregated request statistics
func GetRequestStats(userID int64, role string, districtID *string) (map[string]int, error) {
	params := []interface{}{}
	paramCount := 1

	whereClause, params, paramCount := BuildRoleBasedWhereClause(userID, role, districtID, params, paramCount)

	statsQuery := fmt.Sprintf(`
		SELECT
			COUNT(*)::int AS total,
			SUM(CASE WHEN fr.current_status = 'pending' THEN 1 ELSE 0 END)::int AS pending,
			SUM(CASE WHEN fr.current_status = 'approved' THEN 1 ELSE 0 END)::int AS approved,
			SUM(CASE WHEN fr.current_status = 'rejected' THEN 1 ELSE 0 END)::int AS rejected,
			SUM(CASE WHEN fr.current_status = 'cancelled' THEN 1 ELSE 0 END)::int AS cancelled
		FROM facility_requests fr
		%s
	`, whereClause)

	var stats struct {
		Total    int
		Pending  int
		Approved int
		Rejected int
		Cancelled int
	}

	err := configs.DB.QueryRow(statsQuery, params...).Scan(
		&stats.Total, &stats.Pending, &stats.Approved, &stats.Rejected, &stats.Cancelled,
	)
	if err != nil {
		return nil, err
	}

	return map[string]int{
		"total":    stats.Total,
		"pending":  stats.Pending,
		"approved": stats.Approved,
		"rejected": stats.Rejected,
		"cancelled": stats.Cancelled,
	}, nil
}

// GetRequestById gets request by ID
func GetRequestById(requestID int64, userID int64, role string) (*models.Request, error) {
	var req models.Request
	var facilityMflUid sql.NullString
	err := configs.DB.QueryRow(
		`SELECT 
			fr.id, fr.request_type, fr.facility_id, fr.facility_data, fr.current_status, fr.current_stage,
			fr.initiated_by, fr.district_approver_id, fr.moh_clinical_id, fr.moh_publisher_id,
			fr.rejection_reason, fr."createdAt", fr."updatedAt",
			u1.username as initiated_by_name,
			u1.email as initiated_by_email,
			f.mfl_uid as facility_mfl_uid
		 FROM facility_requests fr
		 LEFT JOIN users u1 ON fr.initiated_by = u1.id
		 LEFT JOIN facilities f ON fr.facility_id = f.id
		 WHERE fr.id = $1`,
		requestID,
	).Scan(
		&req.ID, &req.RequestType, &req.FacilityID, &req.FacilityData,
		&req.CurrentStatus, &req.CurrentStage, &req.InitiatedBy,
		&req.DistrictApproverID, &req.MohClinicalID, &req.MohPublisherID,
		&req.RejectionReason, &req.CreatedAt, &req.UpdatedAt,
		&req.InitiatedByName, &req.InitiatedByEmail,
		&facilityMflUid,
	)
	if facilityMflUid.Valid {
		req.FacilityMflUid = &facilityMflUid.String
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Request not found")
	}
	if err != nil {
		return nil, err
	}

	// Check permissions
	if role == "public" && req.InitiatedBy != userID {
		return nil, fmt.Errorf("You can only view your own requests")
	}

	// Get approval history
	approvalRows, err := configs.DB.Query(
		`SELECT 
			fra.id, fra.request_id, fra.stage, fra.action, fra.approver_id, fra.comments, fra."createdAt",
			u.username as approver_name,
			u.email as approver_email
		 FROM facility_request_approvals fra
		 LEFT JOIN users u ON fra.approver_id = u.id
		 WHERE fra.request_id = $1
		 ORDER BY fra."createdAt" ASC`,
		requestID,
	)
	if err == nil {
		defer approvalRows.Close()
		for approvalRows.Next() {
			var approval models.Approval
			approvalRows.Scan(
				&approval.ID, &approval.RequestID, &approval.Stage, &approval.Action,
				&approval.ApproverID, &approval.Comments, &approval.CreatedAt,
				&approval.ApproverName, &approval.ApproverEmail,
			)
			req.Approvals = append(req.Approvals, approval)
		}
	}

	// Get documents
	docRows, err := configs.DB.Query(
		"SELECT id, request_id, filename, original_filename, file_path, file_size, mime_type, doc_type, \"createdAt\" FROM facility_request_documents WHERE request_id = $1 ORDER BY \"createdAt\" ASC",
		requestID,
	)
	if err == nil {
		defer docRows.Close()
		for docRows.Next() {
			var doc models.RequestDocument
			docRows.Scan(&doc.ID, &doc.RequestID, &doc.Filename, &doc.OriginalFilename, &doc.FilePath, &doc.FileSize, &doc.MimeType, &doc.DocType, &doc.CreatedAt)
			req.Documents = append(req.Documents, doc)
		}
	}

	return &req, nil
}

// GetFacilitiesForSelection gets facilities for selection
func GetFacilitiesForSelection(userID int64, role string, districtID *string, searchQuery string) ([]map[string]interface{}, error) {
	whereClause := "WHERE 1=1"
	params := []interface{}{}
	paramCount := 1

	if role == "public" {
		whereClause += fmt.Sprintf(" AND f.user_id = $%d", paramCount)
		params = append(params, userID)
		paramCount++
	} else if role == "district_initiator" && districtID != nil {
		whereClause += fmt.Sprintf(` AND f.admin_unit_id IN (
			SELECT au.id
			FROM admin_units au
			WHERE au.path LIKE (
				(SELECT path FROM admin_units WHERE mfl_uid = $%d) || '%%'
			)
		)`, paramCount)
		params = append(params, *districtID)
		paramCount++
	}

	if searchQuery != "" {
		whereClause += fmt.Sprintf(" AND (f.name ILIKE $%d OR f.identifier ILIKE $%d OR f.historical_id ILIKE $%d)", paramCount, paramCount, paramCount)
		params = append(params, "%"+searchQuery+"%")
		paramCount++
	}

	query := fmt.Sprintf(`
		SELECT f.id, f.identifier, f.name, f.short_name, f.status, f.mfl_uid, f.historical_id
		FROM facilities f
		%s
		ORDER BY f.name
		LIMIT 100
	`, whereClause)

	rows, err := configs.DB.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facilities []map[string]interface{}
	for rows.Next() {
		var id int64
		var identifier, name sql.NullString
		var shortName, status, mflUid, historicalID sql.NullString

		err := rows.Scan(&id, &identifier, &name, &shortName, &status, &mflUid, &historicalID)
		if err != nil {
			continue
		}

		facility := map[string]interface{}{
			"id": id,
		}
		if identifier.Valid {
			facility["identifier"] = identifier.String
		}
		if name.Valid {
			facility["name"] = name.String
		}
		if shortName.Valid {
			facility["short_name"] = shortName.String
		}
		if status.Valid {
			facility["status"] = status.String
		}
		if mflUid.Valid {
			facility["mfl_uid"] = mflUid.String
		}
		if historicalID.Valid {
			facility["historical_id"] = historicalID.String
		}

		facilities = append(facilities, facility)
	}

	return facilities, nil
}

// GetDistrictInfo gets user's district and subcounties
func GetDistrictInfo(userID int64, districtID *string) (map[string]interface{}, error) {
	if districtID == nil {
		return nil, fmt.Errorf("User district not found")
	}

	var district struct {
		ID       int64
		MflUid   string
		Name     string
		Path     string
	}

	err := configs.DB.QueryRow(
		`SELECT au.id, au.mfl_uid, au.name, au.path
		 FROM admin_units au
		 WHERE au.mfl_uid = $1`,
		*districtID,
	).Scan(&district.ID, &district.MflUid, &district.Name, &district.Path)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("District admin unit not found")
	}
	if err != nil {
		return nil, err
	}

	if district.Path == "" {
		return nil, fmt.Errorf("District path not found")
	}

	// Get subcounty level_id
	var subcountyLevelID int64
	err = configs.DB.QueryRow(
		`SELECT id FROM admin_level WHERE name ILIKE '%%subcounty%%' OR name ILIKE '%%sub-county%%' LIMIT 1`,
	).Scan(&subcountyLevelID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Subcounty level not found in admin_level table")
	}
	if err != nil {
		return nil, err
	}

	// Get subcounties
	subcountyRows, err := configs.DB.Query(
		`SELECT id, name, mfl_uid 
		 FROM admin_units 
		 WHERE path LIKE $1 || '%%' 
		   AND level_id = $2
		   AND path != $1
		 ORDER BY name`,
		district.Path, subcountyLevelID,
	)
	if err != nil {
		return nil, err
	}
	defer subcountyRows.Close()

	var subcounties []map[string]interface{}
	for subcountyRows.Next() {
		var id int64
		var name, mflUid string
		subcountyRows.Scan(&id, &name, &mflUid)
		subcounties = append(subcounties, map[string]interface{}{
			"id":      id,
			"name":    name,
			"mfl_uid": mflUid,
		})
	}

	return map[string]interface{}{
		"district": map[string]interface{}{
			"id":      district.ID,
			"mfl_uid": district.MflUid,
			"name":    district.Name,
		},
		"subcounties": subcounties,
	}, nil
}

// GetDocumentForDownload gets document for download
func GetDocumentForDownload(requestID, docID int64, userID int64, role string) (*models.RequestDocument, error) {
	// Verify request exists and user has access
	var initiatedBy int64
	err := configs.DB.QueryRow("SELECT initiated_by FROM facility_requests WHERE id = $1", requestID).Scan(&initiatedBy)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Request not found")
	}
	if err != nil {
		return nil, err
	}

	if role == "public" && initiatedBy != userID {
		return nil, fmt.Errorf("Access denied")
	}

	// Get document
	var doc models.RequestDocument
	err = configs.DB.QueryRow(
		"SELECT id, request_id, filename, original_filename, file_path, file_size, mime_type, doc_type, \"createdAt\" FROM facility_request_documents WHERE id = $1 AND request_id = $2",
		docID, requestID,
	).Scan(&doc.ID, &doc.RequestID, &doc.Filename, &doc.OriginalFilename, &doc.FilePath, &doc.FileSize, &doc.MimeType, &doc.DocType, &doc.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Document not found")
	}
	if err != nil {
		return nil, err
	}

	return &doc, nil
}
