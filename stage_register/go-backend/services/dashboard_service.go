package services

import (
	"go-backend/configs"
)

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalFacilities  int64 `json:"total_facilities"`
	ActiveFacilities int64 `json:"active_facilities"`
	PendingReview    int64 `json:"pending_review"`
	RegisteredUsers  int64 `json:"registered_users"`
}

// GetDashboardStats retrieves dashboard statistics
func GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Get total facilities count
	err := configs.DB.QueryRow("SELECT COUNT(*) FROM facilities").Scan(&stats.TotalFacilities)
	if err != nil {
		return nil, err
	}

	// Get active facilities count (status is not 'Non-Functional' or is NULL)
	err = configs.DB.QueryRow(
		"SELECT COUNT(*) FROM facilities WHERE status IS NULL OR status != 'Non-Functional'",
	).Scan(&stats.ActiveFacilities)
	if err != nil {
		return nil, err
	}

	// Get pending review count (requests with status 'pending')
	err = configs.DB.QueryRow(
		"SELECT COUNT(*) FROM facility_requests WHERE current_status = 'pending'",
	).Scan(&stats.PendingReview)
	if err != nil {
		return nil, err
	}

	// Get registered users count
	err = configs.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.RegisteredUsers)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

