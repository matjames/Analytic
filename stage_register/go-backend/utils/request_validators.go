package utils

import "fmt"

// ValidateRequestType validates request type
func ValidateRequestType(requestType string) error {
	validTypes := []string{"new_addition", "update", "deactivation"}
	for _, valid := range validTypes {
		if requestType == valid {
			return nil
		}
	}
	return fmt.Errorf("request_type must be one of: new_addition, update, deactivation")
}

// ValidateRequestCreation validates request creation data
func ValidateRequestCreation(requestType string, facilityID *int64, facilityData interface{}, files []interface{}) error {
	if err := ValidateRequestType(requestType); err != nil {
		return err
	}

	// For update and deactivation, facility_id is required
	if (requestType == "update" || requestType == "deactivation") && facilityID == nil {
		return fmt.Errorf("facility_id is required for update and deactivation requests")
	}

	// For new_addition and update, facility_data is required
	if (requestType == "new_addition" || requestType == "update") && facilityData == nil {
		return fmt.Errorf("facility_data is required for new_addition and update requests")
	}

	// For update and deactivation, supporting documents are mandatory
	if (requestType == "update" || requestType == "deactivation") && (files == nil || len(files) == 0) {
		return fmt.Errorf("Supporting documents are required for update and deactivation requests")
	}

	return nil
}

// ValidateInitiatorRole validates user role for initiating requests
func ValidateInitiatorRole(userRole string) error {
	if userRole != "public" && userRole != "district_initiator" {
		return fmt.Errorf("Only users with public or district_initiator role can initiate requests")
	}
	return nil
}

// ValidateRejection validates rejection data
func ValidateRejection(rejectionReason string) error {
	if rejectionReason == "" {
		return fmt.Errorf("rejection_reason is required")
	}
	return nil
}

