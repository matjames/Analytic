package utils

import "go-backend/models"

// CanApproveStage checks if user has permission for a stage
func CanApproveStage(userRole, stage string) bool {
	roleStageMap := map[string]string{
		"district_approver": "district_approver",
		"district":          "district_approver", // unified district role can approve at district stage
		"moh_clinical":      "moh_clinical",
		"moh_publisher":     "moh_publisher",
	}
	return roleStageMap[userRole] == stage
}

// GetNextStage returns the next stage in the workflow
func GetNextStage(currentStage string) string {
	stages := []string{"district_approver", "moh_clinical", "moh_publisher", "completed"}
	for i, stage := range stages {
		if stage == currentStage && i < len(stages)-1 {
			return stages[i+1]
		}
	}
	return "completed"
}

// FormatRequest formats request response with approvals
func FormatRequest(req *models.Request, approvals []models.Approval) *models.Request {
	if req == nil {
		return nil
	}
	req.Approvals = approvals
	return req
}

// FormatDocument formats document for response
func FormatDocument(doc models.RequestDocument) map[string]interface{} {
	return map[string]interface{}{
		"id":               doc.ID,
		"filename":         doc.Filename,
		"original_filename": doc.OriginalFilename,
		"file_size":        doc.FileSize,
		"mime_type":        doc.MimeType,
		"createdAt":        doc.CreatedAt,
	}
}
