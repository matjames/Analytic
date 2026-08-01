package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"
)

// RecordApproval records an approval action
func RecordApproval(requestID int64, stage string, approverID int64, comments *string) error {
	_, err := configs.DB.Exec(
		`INSERT INTO facility_request_approvals 
		 (request_id, stage, action, approver_id, comments)
		 VALUES ($1, $2, 'approved', $3, $4)`,
		requestID, stage, approverID, comments,
	)
	return err
}

// RecordRejection records a rejection action
func RecordRejection(requestID int64, stage string, approverID int64, comments *string) error {
	_, err := configs.DB.Exec(
		`INSERT INTO facility_request_approvals 
		 (request_id, stage, action, approver_id, comments)
		 VALUES ($1, $2, 'rejected', $3, $4)`,
		requestID, stage, approverID, comments,
	)
	return err
}

// UpdateRequestAfterApproval updates request status after approval
func UpdateRequestAfterApproval(request *models.Request, nextStage string, approverID int64) error {
	if nextStage == "completed" {
		// Final approval - update status and stage
		_, err := configs.DB.Exec(
			`UPDATE facility_requests 
			 SET current_status = 'approved',
			     current_stage = 'completed',
			     moh_publisher_id = $1,
			     "updatedAt" = NOW()
			 WHERE id = $2`,
			approverID, request.ID,
		)
		if err != nil {
			return err
		}

		// Apply the changes to the facility
		return ApplyFacilityChanges(request)
	} else if request.CurrentStage == "district_approver" {
		// Move to next stage
		_, err := configs.DB.Exec(
			`UPDATE facility_requests 
			 SET current_stage = $1,
			     district_approver_id = $2,
			     "updatedAt" = NOW()
			 WHERE id = $3`,
			nextStage, approverID, request.ID,
		)
		return err
	} else if request.CurrentStage == "moh_clinical" {
		// Move to next stage
		_, err := configs.DB.Exec(
			`UPDATE facility_requests 
			 SET current_stage = $1,
			     moh_clinical_id = $2,
			     "updatedAt" = NOW()
			 WHERE id = $3`,
			nextStage, approverID, request.ID,
		)
		return err
	}
	return nil
}

// ApplyFacilityChanges applies facility changes based on request type
func ApplyFacilityChanges(request *models.Request) error {
	facilityData := request.FacilityData
	initiatorID := request.InitiatedBy

	if request.RequestType == "new_addition" {
		return CreateFacilityFromRequest(facilityData, initiatorID)
	} else if request.RequestType == "update" && request.FacilityID != nil {
		return UpdateFacilityFromRequest(*request.FacilityID, facilityData, initiatorID)
	} else if request.RequestType == "deactivation" && request.FacilityID != nil {
		return DeactivateFacility(*request.FacilityID, initiatorID)
	}
	return nil
}

// CreateFacilityFromRequest creates a new facility from request data
func CreateFacilityFromRequest(facilityData models.FacilityData, userID int64) error {
	// Get the Facility level_id from admin_level
	var facilityLevelID int64
	err := configs.DB.QueryRow(
		`SELECT id FROM admin_level 
		 WHERE name ILIKE 'facility' 
		 OR level_number = (SELECT MAX(level_number) FROM admin_level) 
		 ORDER BY level_number DESC LIMIT 1`,
	).Scan(&facilityLevelID)

	if err == sql.ErrNoRows {
		return fmt.Errorf("Facility level not found in admin_level table")
	}
	if err != nil {
		return err
	}

	// Get parent admin_unit details
	var parentID *int64
	var parentPath string = "/"
	adminUnitID, ok := facilityData["admin_unit_id"].(float64)
	if ok {
		parentIDInt := int64(adminUnitID)
		parentID = &parentIDInt

		var parentMflUid sql.NullString
		err = configs.DB.QueryRow(
			"SELECT id, mfl_uid, path FROM admin_units WHERE id = $1",
			*parentID,
		).Scan(&parentID, &parentMflUid, &parentPath)
		if err == nil && parentPath == "" {
			parentPath = "/"
		}
	}

	// Generate unique mfl_uid for facility admin_unit
	uid, err := utils.EnsureUniqueMflUid(configs.DB, "admin_units")
	if err != nil {
		return fmt.Errorf("Failed to generate unique mfl_uid for facility")
	}

	// Compute path (no trailing slash)
	parentTrimmed := strings.TrimSuffix(parentPath, "/")
	if parentTrimmed == "" {
		parentTrimmed = "/"
	}
	facilityPath := parentTrimmed + "/" + uid

	// Create admin_unit entry for the facility
	var facilityAdminUnitID int64
		err = configs.DB.QueryRow(
			`INSERT INTO admin_units (name, code, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		 RETURNING id`,
		getString(facilityData, "name"), nil, uid, facilityLevelID, parentID, facilityPath,
		).Scan(&facilityAdminUnitID)

	if err != nil {
		return err
	}

	// Generate facility identifier
	identifier := utils.GenerateFacilityIdentifier()

	// Create facility entry
	servicesJSON, _ := json.Marshal(facilityData["services"])

	_, err = configs.DB.Exec(
		`INSERT INTO facilities (
			identifier, mfl_uid, name, short_name, historical_id, admin_unit_id, level, ownership, authority,
			status, reporting, licensed, address, contact_personemail, contact_personmobile,
			contact_personname, contact_persontitle, longitude, latitude, opening_date,
			closing_date, bed_capacity, services, user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)`,
		identifier, uid,
		getString(facilityData, "name"),
		getStringPtr(facilityData, "short_name"),
		getStringPtr(facilityData, "historical_id"),
		facilityAdminUnitID,
		getStringPtr(facilityData, "level"),
		getStringPtr(facilityData, "ownership"),
		getStringPtr(facilityData, "authority"),
		"Functional",
		true,
		getBool(facilityData, "licensed"),
		getStringPtr(facilityData, "address"),
		getStringPtr(facilityData, "contact_personemail"),
		getStringPtr(facilityData, "contact_personmobile"),
		getStringPtr(facilityData, "contact_personname"),
		getStringPtr(facilityData, "contact_persontitle"),
		getFloat64Ptr(facilityData, "longitude"),
		getFloat64Ptr(facilityData, "latitude"),
		getStringPtr(facilityData, "opening_date"),
		getStringPtr(facilityData, "closing_date"),
		getInt64Ptr(facilityData, "bed_capacity"),
		servicesJSON,
		userID,
	)

	return err
}

// UpdateFacilityFromRequest updates an existing facility from request data
func UpdateFacilityFromRequest(facilityID int64, facilityData models.FacilityData, userID int64) error {
	servicesJSON, _ := json.Marshal(facilityData["services"])

	_, err := configs.DB.Exec(
		`UPDATE facilities SET
			name = COALESCE($1, name),
			short_name = COALESCE($2, short_name),
			historical_id = COALESCE($3, historical_id),
			admin_unit_id = COALESCE($4, admin_unit_id),
			level = COALESCE($5, level),
			ownership = COALESCE($6, ownership),
			authority = COALESCE($7, authority),
			status = COALESCE($8, status),
			reporting = COALESCE($9, reporting),
			licensed = COALESCE($10, licensed),
			address = COALESCE($11, address),
			contact_personemail = COALESCE($12, contact_personemail),
			contact_personmobile = COALESCE($13, contact_personmobile),
			contact_personname = COALESCE($14, contact_personname),
			contact_persontitle = COALESCE($15, contact_persontitle),
			longitude = COALESCE($16, longitude),
			latitude = COALESCE($17, latitude),
			opening_date = COALESCE($18, opening_date),
			closing_date = COALESCE($19, closing_date),
			bed_capacity = COALESCE($20, bed_capacity),
			services = COALESCE($21, services),
			user_id = COALESCE($22, user_id),
			"updatedAt" = NOW()
		WHERE id = $23`,
		getStringPtr(facilityData, "name"),
		getStringPtr(facilityData, "short_name"),
		getStringPtr(facilityData, "historical_id"),
		getInt64Ptr(facilityData, "admin_unit_id"),
		getStringPtr(facilityData, "level"),
		getStringPtr(facilityData, "ownership"),
		getStringPtr(facilityData, "authority"),
		getStringPtr(facilityData, "status"),
		getBoolPtr(facilityData, "reporting"),
		getBoolPtr(facilityData, "licensed"),
		getStringPtr(facilityData, "address"),
		getStringPtr(facilityData, "contact_personemail"),
		getStringPtr(facilityData, "contact_personmobile"),
		getStringPtr(facilityData, "contact_personname"),
		getStringPtr(facilityData, "contact_persontitle"),
		getFloat64Ptr(facilityData, "longitude"),
		getFloat64Ptr(facilityData, "latitude"),
		getStringPtr(facilityData, "opening_date"),
		getStringPtr(facilityData, "closing_date"),
		getInt64Ptr(facilityData, "bed_capacity"),
		servicesJSON,
		userID,
		facilityID,
	)

	return err
}

// DeactivateFacility deactivates a facility
func DeactivateFacility(facilityID int64, userID int64) error {
	_, err := configs.DB.Exec(
		`UPDATE facilities SET status = 'Non-Functional', user_id = COALESCE($2, user_id), "updatedAt" = NOW() WHERE id = $1`,
		facilityID, userID,
	)
	return err
}

// UpdateRequestAfterRejection updates request status after rejection
func UpdateRequestAfterRejection(request *models.Request, approverID int64, rejectionReason string) error {
	if request.CurrentStage == "district_approver" {
		_, err := configs.DB.Exec(
			`UPDATE facility_requests 
			 SET current_status = 'rejected',
			     district_approver_id = $1,
			     rejection_reason = $2,
			     "updatedAt" = NOW()
			 WHERE id = $3`,
			approverID, rejectionReason, request.ID,
		)
		return err
	} else if request.CurrentStage == "moh_clinical" {
		_, err := configs.DB.Exec(
			`UPDATE facility_requests 
			 SET current_status = 'rejected',
			     moh_clinical_id = $1,
			     rejection_reason = $2,
			     "updatedAt" = NOW()
			 WHERE id = $3`,
			approverID, rejectionReason, request.ID,
		)
		return err
	} else if request.CurrentStage == "moh_publisher" {
		_, err := configs.DB.Exec(
			`UPDATE facility_requests 
			 SET current_status = 'rejected',
			     moh_publisher_id = $1,
			     rejection_reason = $2,
			     "updatedAt" = NOW()
			 WHERE id = $3`,
			approverID, rejectionReason, request.ID,
		)
		return err
	}
	return nil
}

// ApproveRequest approves a request
func ApproveRequest(requestID int64, userID int64, role string, comments *string) (*models.Request, error) {
	// Begin transaction
	tx, err := configs.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get request
	var req models.Request
	err = tx.QueryRow(
		"SELECT id, request_type, facility_id, facility_data, current_status, current_stage, initiated_by FROM facility_requests WHERE id = $1",
		requestID,
	).Scan(&req.ID, &req.RequestType, &req.FacilityID, &req.FacilityData, &req.CurrentStatus, &req.CurrentStage, &req.InitiatedBy)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Request not found")
	}
	if err != nil {
		return nil, err
	}

	// Check if request is in a valid state
	if req.CurrentStatus != "pending" {
		return nil, fmt.Errorf("Request is already %s", req.CurrentStatus)
	}

	// Check if user can approve at this stage
	if !utils.CanApproveStage(role, req.CurrentStage) {
		return nil, fmt.Errorf("You do not have permission to approve requests at %s stage", req.CurrentStage)
	}

	// Record approval
	if _, err := tx.Exec(
		`INSERT INTO facility_request_approvals 
		 (request_id, stage, action, approver_id, comments)
		 VALUES ($1, $2, 'approved', $3, $4)`,
		requestID, req.CurrentStage, userID, comments,
	); err != nil {
		return nil, err
	}

	// Update request based on stage
	nextStage := utils.GetNextStage(req.CurrentStage)
	if err := updateRequestAfterApprovalTx(tx, &req, nextStage, userID); err != nil {
		return nil, err
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Fetch updated request with approvals
	return GetRequestWithApprovals(requestID)
}

// updateRequestAfterApprovalTx updates request status after approval (transaction version)
func updateRequestAfterApprovalTx(tx *sql.Tx, request *models.Request, nextStage string, approverID int64) error {
	if nextStage == "completed" {
		// Final approval - update status and stage
		_, err := tx.Exec(
			`UPDATE facility_requests 
			 SET current_status = 'approved',
			     current_stage = 'completed',
			     moh_publisher_id = $1,
			     "updatedAt" = NOW()
			 WHERE id = $2`,
			approverID, request.ID,
		)
		if err != nil {
			return err
		}

		// Apply the changes to the facility
		return applyFacilityChangesTx(tx, request)
	} else if request.CurrentStage == "district_approver" {
		_, err := tx.Exec(
			`UPDATE facility_requests 
			 SET current_stage = $1,
			     district_approver_id = $2,
			     "updatedAt" = NOW()
			 WHERE id = $3`,
			nextStage, approverID, request.ID,
		)
		return err
	} else if request.CurrentStage == "moh_clinical" {
		_, err := tx.Exec(
			`UPDATE facility_requests 
			 SET current_stage = $1,
			     moh_clinical_id = $2,
			     "updatedAt" = NOW()
			 WHERE id = $3`,
			nextStage, approverID, request.ID,
		)
		return err
	}
	return nil
}

// applyFacilityChangesTx applies facility changes based on request type (transaction version)
func applyFacilityChangesTx(tx *sql.Tx, request *models.Request) error {
	facilityData := request.FacilityData
	initiatorID := request.InitiatedBy

	if request.RequestType == "new_addition" {
		return createFacilityFromRequestTx(tx, facilityData, initiatorID)
	} else if request.RequestType == "update" && request.FacilityID != nil {
		return updateFacilityFromRequestTx(tx, *request.FacilityID, facilityData, initiatorID)
	} else if request.RequestType == "deactivation" && request.FacilityID != nil {
		return deactivateFacilityTx(tx, *request.FacilityID, initiatorID)
	}
	return nil
}

// createFacilityFromRequestTx creates a new facility from request data (transaction version)
func createFacilityFromRequestTx(tx *sql.Tx, facilityData models.FacilityData, userID int64) error {
	// Get the Facility level_id from admin_level
	var facilityLevelID int64
	err := tx.QueryRow(
		`SELECT id FROM admin_level 
		 WHERE name ILIKE 'facility' 
		 OR level_number = (SELECT MAX(level_number) FROM admin_level) 
		 ORDER BY level_number DESC LIMIT 1`,
	).Scan(&facilityLevelID)

	if err == sql.ErrNoRows {
		return fmt.Errorf("Facility level not found in admin_level table")
	}
	if err != nil {
		return err
	}

	// Get parent admin_unit details
	var parentID *int64
	var parentPath string = "/"
	adminUnitID, ok := facilityData["admin_unit_id"].(float64)
	if ok {
		parentIDInt := int64(adminUnitID)
		parentID = &parentIDInt

		var parentMflUid sql.NullString
		err = tx.QueryRow(
			"SELECT id, mfl_uid, path FROM admin_units WHERE id = $1",
			*parentID,
		).Scan(&parentID, &parentMflUid, &parentPath)
		if err == nil && parentPath == "" {
			parentPath = "/"
		}
	}

	// Generate unique mfl_uid for facility admin_unit
	uid, err := utils.EnsureUniqueMflUid(configs.DB, "admin_units")
	if err != nil {
		return fmt.Errorf("Failed to generate unique mfl_uid for facility")
	}

	// Compute path (no trailing slash)
	parentTrimmed := strings.TrimSuffix(parentPath, "/")
	if parentTrimmed == "" {
		parentTrimmed = "/"
	}
	facilityPath := parentTrimmed + "/" + uid

	// Create admin_unit entry for the facility
	var facilityAdminUnitID int64
	err = tx.QueryRow(
		`INSERT INTO admin_units (name, code, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		 RETURNING id`,
		getString(facilityData, "name"), nil, uid, facilityLevelID, parentID, facilityPath,
	).Scan(&facilityAdminUnitID)

	if err != nil {
		return err
	}

	// Generate facility identifier
	identifier := utils.GenerateFacilityIdentifier()

	// Create facility entry
	servicesJSON, _ := json.Marshal(facilityData["services"])

	_, err = tx.Exec(
		`INSERT INTO facilities (
			identifier, mfl_uid, name, short_name, historical_id, admin_unit_id, level, ownership, authority,
			status, reporting, licensed, address, contact_personemail, contact_personmobile,
			contact_personname, contact_persontitle, longitude, latitude, opening_date,
			closing_date, bed_capacity, services, user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)`,
		identifier, uid,
		getString(facilityData, "name"),
		getStringPtr(facilityData, "short_name"),
		getStringPtr(facilityData, "historical_id"),
		facilityAdminUnitID,
		getStringPtr(facilityData, "level"),
		getStringPtr(facilityData, "ownership"),
		getStringPtr(facilityData, "authority"),
		"Functional",
		true,
		getBool(facilityData, "licensed"),
		getStringPtr(facilityData, "address"),
		getStringPtr(facilityData, "contact_personemail"),
		getStringPtr(facilityData, "contact_personmobile"),
		getStringPtr(facilityData, "contact_personname"),
		getStringPtr(facilityData, "contact_persontitle"),
		getFloat64Ptr(facilityData, "longitude"),
		getFloat64Ptr(facilityData, "latitude"),
		getStringPtr(facilityData, "opening_date"),
		getStringPtr(facilityData, "closing_date"),
		getInt64Ptr(facilityData, "bed_capacity"),
		servicesJSON,
		userID,
	)

	return err
}

// updateFacilityFromRequestTx updates an existing facility from request data (transaction version)
func updateFacilityFromRequestTx(tx *sql.Tx, facilityID int64, facilityData models.FacilityData, userID int64) error {
	servicesJSON, _ := json.Marshal(facilityData["services"])

	_, err := tx.Exec(
		`UPDATE facilities SET
			name = COALESCE($1, name),
			short_name = COALESCE($2, short_name),
			historical_id = COALESCE($3, historical_id),
			admin_unit_id = COALESCE($4, admin_unit_id),
			level = COALESCE($5, level),
			ownership = COALESCE($6, ownership),
			authority = COALESCE($7, authority),
			status = COALESCE($8, status),
			reporting = COALESCE($9, reporting),
			licensed = COALESCE($10, licensed),
			address = COALESCE($11, address),
			contact_personemail = COALESCE($12, contact_personemail),
			contact_personmobile = COALESCE($13, contact_personmobile),
			contact_personname = COALESCE($14, contact_personname),
			contact_persontitle = COALESCE($15, contact_persontitle),
			longitude = COALESCE($16, longitude),
			latitude = COALESCE($17, latitude),
			opening_date = COALESCE($18, opening_date),
			closing_date = COALESCE($19, closing_date),
			bed_capacity = COALESCE($20, bed_capacity),
			services = COALESCE($21, services),
			user_id = COALESCE($22, user_id),
			"updatedAt" = NOW()
		WHERE id = $23`,
		getStringPtr(facilityData, "name"),
		getStringPtr(facilityData, "short_name"),
		getStringPtr(facilityData, "historical_id"),
		getInt64Ptr(facilityData, "admin_unit_id"),
		getStringPtr(facilityData, "level"),
		getStringPtr(facilityData, "ownership"),
		getStringPtr(facilityData, "authority"),
		getStringPtr(facilityData, "status"),
		getBoolPtr(facilityData, "reporting"),
		getBoolPtr(facilityData, "licensed"),
		getStringPtr(facilityData, "address"),
		getStringPtr(facilityData, "contact_personemail"),
		getStringPtr(facilityData, "contact_personmobile"),
		getStringPtr(facilityData, "contact_personname"),
		getStringPtr(facilityData, "contact_persontitle"),
		getFloat64Ptr(facilityData, "longitude"),
		getFloat64Ptr(facilityData, "latitude"),
		getStringPtr(facilityData, "opening_date"),
		getStringPtr(facilityData, "closing_date"),
		getInt64Ptr(facilityData, "bed_capacity"),
		servicesJSON,
		userID,
		facilityID,
	)

	return err
}

// deactivateFacilityTx deactivates a facility (transaction version)
func deactivateFacilityTx(tx *sql.Tx, facilityID int64, userID int64) error {
	_, err := tx.Exec(
		`UPDATE facilities SET status = 'Non-Functional', user_id = COALESCE($2, user_id), "updatedAt" = NOW() WHERE id = $1`,
		facilityID, userID,
	)
	return err
}

// RejectRequest rejects a request
func RejectRequest(requestID int64, userID int64, role string, rejectionReason string, comments *string) (*models.Request, error) {
	// Begin transaction
	tx, err := configs.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get request
	var req models.Request
	err = tx.QueryRow(
		"SELECT id, request_type, facility_id, facility_data, current_status, current_stage, initiated_by FROM facility_requests WHERE id = $1",
		requestID,
	).Scan(&req.ID, &req.RequestType, &req.FacilityID, &req.FacilityData, &req.CurrentStatus, &req.CurrentStage, &req.InitiatedBy)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Request not found")
	}
	if err != nil {
		return nil, err
	}

	// Check if request is in a valid state
	if req.CurrentStatus != "pending" {
		return nil, fmt.Errorf("Request is already %s", req.CurrentStatus)
	}

	// Check if user can reject at this stage
	if !utils.CanApproveStage(role, req.CurrentStage) {
		return nil, fmt.Errorf("You do not have permission to reject requests at %s stage", req.CurrentStage)
	}

	// Record rejection
	if _, err := tx.Exec(
		`INSERT INTO facility_request_approvals 
		 (request_id, stage, action, approver_id, comments)
		 VALUES ($1, $2, 'rejected', $3, $4)`,
		requestID, req.CurrentStage, userID, comments,
	); err != nil {
		return nil, err
	}

	// Update request status
	if err := updateRequestAfterRejectionTx(tx, &req, userID, rejectionReason); err != nil {
		return nil, err
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Fetch updated request with approvals
	return GetRequestWithApprovals(requestID)
}

// updateRequestAfterRejectionTx updates request status after rejection (transaction version)
func updateRequestAfterRejectionTx(tx *sql.Tx, request *models.Request, approverID int64, rejectionReason string) error {
	if request.CurrentStage == "district_approver" {
		_, err := tx.Exec(
			`UPDATE facility_requests 
			 SET current_status = 'rejected',
			     district_approver_id = $1,
			     rejection_reason = $2,
			     "updatedAt" = NOW()
			 WHERE id = $3`,
			approverID, rejectionReason, request.ID,
		)
		return err
	} else if request.CurrentStage == "moh_clinical" {
		_, err := tx.Exec(
			`UPDATE facility_requests 
			 SET current_status = 'rejected',
			     moh_clinical_id = $1,
			     rejection_reason = $2,
			     "updatedAt" = NOW()
			 WHERE id = $3`,
			approverID, rejectionReason, request.ID,
		)
		return err
	} else if request.CurrentStage == "moh_publisher" {
		_, err := tx.Exec(
			`UPDATE facility_requests 
			 SET current_status = 'rejected',
			     moh_publisher_id = $1,
			     rejection_reason = $2,
			     "updatedAt" = NOW()
			 WHERE id = $3`,
			approverID, rejectionReason, request.ID,
		)
		return err
	}
	return nil
}

// GetRequestWithApprovals gets request with approvals
func GetRequestWithApprovals(requestID int64) (*models.Request, error) {
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

	if err != nil {
		return nil, err
	}

	// Get approvals
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

	return &req, nil
}

// Helper functions to extract values from FacilityData map
func getString(data models.FacilityData, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getStringPtr(data models.FacilityData, key string) *string {
	if val, ok := data[key].(string); ok && val != "" {
		return &val
	}
	return nil
}

func getBool(data models.FacilityData, key string) bool {
	if val, ok := data[key].(bool); ok {
		return val
	}
	return false
}

func getBoolPtr(data models.FacilityData, key string) *bool {
	if val, ok := data[key].(bool); ok {
		return &val
	}
	return nil
}

func getInt64Ptr(data models.FacilityData, key string) *int64 {
	if val, ok := data[key].(float64); ok {
		intVal := int64(val)
		return &intVal
	}
	return nil
}

func getFloat64Ptr(data models.FacilityData, key string) *float64 {
	if val, ok := data[key].(float64); ok {
		return &val
	}
	return nil
}
