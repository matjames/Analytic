package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go-backend/configs"
	"go-backend/services"
)

// CreateRequest creates a new facility request
func CreateRequest(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("user_role")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	if role == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	userIDInt := userID.(int64)
	roleStr := role.(string)

	// Parse form data
	requestType := c.PostForm("request_type")
	facilityIDStr := c.PostForm("facility_id")
	facilityMflUid := c.PostForm("facility_mfl_uid")
	var facilityID *int64
	
	// Support both mfl_uid (preferred) and facility_id (for backward compatibility)
	if facilityMflUid != "" {
		// Look up facility by mfl_uid
		var id int64
		err := configs.DB.QueryRow("SELECT id FROM facilities WHERE mfl_uid = $1", facilityMflUid).Scan(&id)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Facility not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lookup facility"})
			return
		}
		facilityID = &id
	} else if facilityIDStr != "" {
		// Backward compatibility: accept facility_id directly
		id, err := strconv.ParseInt(facilityIDStr, 10, 64)
		if err == nil {
			facilityID = &id
		}
	}

	facilityDataStr := c.PostForm("facility_data")
	var facilityData interface{}
	if facilityDataStr != "" {
		// Parse JSON string from form data
		var parsedData map[string]interface{}
		if err := json.Unmarshal([]byte(facilityDataStr), &parsedData); err == nil {
			facilityData = parsedData
		} else {
			facilityData = facilityDataStr
		}
	}

	// Handle file uploads
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
		return
	}

	files := form.File["documents"]
	docTypes := form.Value["document_types"]
	var fileData []map[string]interface{}

	uploadDir := filepath.Join(".", "uploads", "requests")
	os.MkdirAll(uploadDir, os.ModePerm)

	for idx, file := range files {
		// Generate unique filename
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("%d-%d%s", file.Size, len(file.Filename), ext)
		filePath := filepath.Join(uploadDir, filename)

		// Save file
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		fileSize := file.Size
		var docType string
		if len(docTypes) > idx {
			docType = docTypes[idx]
		}
		fileData = append(fileData, map[string]interface{}{
			"filename":         filename,
			"original_filename": file.Filename,
			"file_path":        filePath,
			"file_size":        fileSize,
			"mime_type":        file.Header.Get("Content-Type"),
			"doc_type":         docType,
		})
	}

	req, err := services.CreateRequest(userIDInt, roleStr, requestType, facilityID, facilityData, fileData)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "permission") || strings.Contains(err.Error(), "role") {
			statusCode = http.StatusForbidden
		} else if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "must be") {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, req)
}

// ListRequests lists requests with pagination and filtering
func ListRequests(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("user_role")
	districtID, _ := c.Get("user_district_id")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	if role == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	userIDInt := userID.(int64)
	roleStr := role.(string)
	var districtIDStr *string
	if districtID != nil {
		if d, ok := districtID.(string); ok {
			districtIDStr = &d
		} else if d, ok := districtID.(int64); ok {
			// Backward compatibility: convert int64 to string
			ds := strconv.FormatInt(d, 10)
			districtIDStr = &ds
		}
	}

	filters := make(map[string]string)
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if stage := c.Query("stage"); stage != "" {
		filters["stage"] = stage
	}
	if requestType := c.Query("request_type"); requestType != "" {
		filters["request_type"] = requestType
	}
	if page := c.Query("page"); page != "" {
		filters["page"] = page
	}
	if pageSize := c.Query("pageSize"); pageSize != "" {
		filters["pageSize"] = pageSize
	}

	result, err := services.ListRequests(userIDInt, roleStr, districtIDStr, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetRequestStats gets aggregated request statistics
func GetRequestStats(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("user_role")
	districtID, _ := c.Get("user_district_id")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	if role == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	userIDInt := userID.(int64)
	roleStr := role.(string)
	var districtIDStr *string
	if districtID != nil {
		if d, ok := districtID.(string); ok {
			districtIDStr = &d
		} else if d, ok := districtID.(int64); ok {
			// Backward compatibility: convert int64 to string
			ds := strconv.FormatInt(d, 10)
			districtIDStr = &ds
		}
	}

	stats, err := services.GetRequestStats(userIDInt, roleStr, districtIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetFacilitiesForSelection gets facilities for selection
func GetFacilitiesForSelection(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("user_role")
	districtID, _ := c.Get("user_district_id")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	if role == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	userIDInt := userID.(int64)
	roleStr := role.(string)
	var districtIDStr *string
	if districtID != nil {
		if d, ok := districtID.(string); ok {
			districtIDStr = &d
		} else if d, ok := districtID.(int64); ok {
			// Backward compatibility: convert int64 to string
			ds := strconv.FormatInt(d, 10)
			districtIDStr = &ds
		}
	}

	searchQuery := c.Query("q")

	facilities, err := services.GetFacilitiesForSelection(userIDInt, roleStr, districtIDStr, searchQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, facilities)
}

// GetDistrictInfo gets user's district and subcounties
func GetDistrictInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	districtID, _ := c.Get("user_district_id")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var districtIDStr *string
	if districtID != nil {
		if d, ok := districtID.(string); ok {
			districtIDStr = &d
		} else if d, ok := districtID.(int64); ok {
			// Backward compatibility: convert int64 to string
			ds := strconv.FormatInt(d, 10)
			districtIDStr = &ds
		}
	} else {
		// Try to get from user
		userIDInt := userID.(int64)
		var d sql.NullString
		err := configs.DB.QueryRow("SELECT district_id FROM users WHERE id = $1", userIDInt).Scan(&d)
		if err == nil && d.Valid {
			districtIDStr = new(string)
			*districtIDStr = d.String
		}
	}

	if districtIDStr == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User district not found"})
		return
	}

	districtInfo, err := services.GetDistrictInfo(userID.(int64), districtIDStr)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, districtInfo)
}

// GetRequestById gets request by ID
func GetRequestById(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("user_role")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	if role == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	userIDInt := userID.(int64)
	roleStr := role.(string)

	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	req, err := services.GetRequestById(requestID, userIDInt, roleStr)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "permission") || strings.Contains(err.Error(), "own") {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, req)
}

// DownloadRequestDocument downloads a request document
func DownloadRequestDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("user_role")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	if role == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	userIDInt := userID.(int64)
	roleStr := role.(string)

	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	docID, err := strconv.ParseInt(c.Param("docId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	doc, err := services.GetDocumentForDownload(requestID, docID, userIDInt, roleStr)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "Access denied") {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	// Check if file exists
	if _, err := os.Stat(doc.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found on server"})
		return
	}

	c.FileAttachment(doc.FilePath, doc.OriginalFilename)
}

// ApproveRequest approves a request
func ApproveRequest(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("user_role")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	if role == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	userIDInt := userID.(int64)
	roleStr := role.(string)

	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var body struct {
		Comments *string `json:"comments"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		// Comments are optional
	}

	req, err := services.ApproveRequest(requestID, userIDInt, roleStr, body.Comments)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "permission") {
			statusCode = http.StatusForbidden
		} else if strings.Contains(err.Error(), "already") {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, req)
}

// RejectRequest rejects a request
func RejectRequest(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("user_role")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	if role == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	userIDInt := userID.(int64)
	roleStr := role.(string)

	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var body struct {
		Comments        *string `json:"comments"`
		RejectionReason string  `json:"rejection_reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if body.RejectionReason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rejection_reason is required"})
		return
	}

	req, err := services.RejectRequest(requestID, userIDInt, roleStr, body.RejectionReason, body.Comments)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "permission") {
			statusCode = http.StatusForbidden
		} else if strings.Contains(err.Error(), "already") || strings.Contains(err.Error(), "required") {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, req)
}

