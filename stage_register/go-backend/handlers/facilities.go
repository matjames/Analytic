package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"go-backend/models"
	"go-backend/services"

	"github.com/gin-gonic/gin"
)

// ListFacilities lists facilities (for UI - uses simple mfl_details query)
func ListFacilities(c *gin.Context) {
	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}

	page, _ := strconv.Atoi(filters["page"])
	pageSize, _ := strconv.Atoi(filters["pageSize"])

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	// Get user context for filtering
	var userID *int64
	var role *string
	var districtID *string
	if userIDVal, exists := c.Get("user_id"); exists {
		if uid, ok := userIDVal.(int64); ok {
			userID = &uid
		}
	}
	if roleVal, exists := c.Get("user_role"); exists {
		if r, ok := roleVal.(string); ok {
			role = &r
		}
	}
	if districtIDVal, exists := c.Get("user_district_id"); exists {
		if did, ok := districtIDVal.(string); ok && did != "" {
			districtID = &did
		}
	}

	result, err := services.ListFacilitiesForUI(filters, page, pageSize, userID, role, districtID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListFacilitiesMfl lists facilities from mfl_list (for API integration)
func ListFacilitiesMfl(c *gin.Context) {
	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}

	result, err := services.ListFacilities(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert facilities to MFL response format (exclude id and user_id)
	if rows, ok := result["rows"].([]*models.Facility); ok {
		mflRows := make([]*models.FacilityMflResponse, len(rows))
		for i, facility := range rows {
			mflRows[i] = facility.ToFacilityMflResponse()
		}
		result["rows"] = mflRows
	}

	c.JSON(http.StatusOK, result)
}

// ListFacilitiesPublic lists facilities publicly (no auth) - uses simple mfl_details query
func ListFacilitiesPublic(c *gin.Context) {
	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}

	page, _ := strconv.Atoi(filters["page"])
	pageSize, _ := strconv.Atoi(filters["pageSize"])

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	// Public endpoint - no user filtering
	result, err := services.ListFacilitiesForUI(filters, page, pageSize, nil, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetFacilityOwnershipByLevel returns summary of facilities by level and ownership (public).
// Optional query parameters: region, district.
func GetFacilityOwnershipByLevel(c *gin.Context) {
	filters := map[string]string{
		"region":    c.Query("region"),
		"district":  c.Query("district"),
		"subcounty": c.Query("subcounty"),
	}

	stats, err := services.GetFacilityOwnershipByLevel(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetFacilityOwnershipTotals returns overall ownership and total facility counts (public).
// Optional query parameters: region, district.
func GetFacilityOwnershipTotals(c *gin.Context) {
	filters := map[string]string{
		"region":    c.Query("region"),
		"district":  c.Query("district"),
		"subcounty": c.Query("subcounty"),
	}

	totals, err := services.GetFacilityOwnershipTotals(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, totals)
}

// GetFacilityDistributionByOwnership returns distribution of facilities by ownership
func GetFacilityDistributionByOwnership(c *gin.Context) {
	stats, err := services.GetFacilityDistributionByOwnership()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetFacilityDistributionByLevel returns distribution of facilities by level
func GetFacilityDistributionByLevel(c *gin.Context) {
	stats, err := services.GetFacilityDistributionByLevel()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetFacilityDistributionByAuthority returns distribution of facilities by authority
func GetFacilityDistributionByAuthority(c *gin.Context) {
	stats, err := services.GetFacilityDistributionByAuthority()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetFacilityFilterOptions returns hierarchical filter options for public filters.
func GetFacilityFilterOptions(c *gin.Context) {
	options, err := services.GetFacilityFilterOptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, options)
}

// ExportFacilitiesPublic exports all facilities publicly (no auth) - uses filters if provided
func ExportFacilitiesPublic(c *gin.Context) {
	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}

	// Public endpoint - no user filtering
	facilities, err := services.ExportFacilitiesForUI(filters, nil, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, facilities)
}

// ExportFacilities exports all facilities for authenticated users - filters by user if initiator/public
func ExportFacilities(c *gin.Context) {
	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}

	// Get user context for filtering
	var userID *int64
	var role *string
	var districtID *string
	if userIDVal, exists := c.Get("user_id"); exists {
		if uid, ok := userIDVal.(int64); ok {
			userID = &uid
		}
	}
	if roleVal, exists := c.Get("user_role"); exists {
		if r, ok := roleVal.(string); ok {
			role = &r
		}
	}
	if districtIDVal, exists := c.Get("user_district_id"); exists {
		if did, ok := districtIDVal.(string); ok && did != "" {
			districtID = &did
		}
	}

	facilities, err := services.ExportFacilitiesForUI(filters, userID, role, districtID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, facilities)
}

// GetFacility gets a facility by mfl_uid
func GetFacility(c *gin.Context) {
	mflUid := c.Param("id")

	facility, err := services.GetFacility(mflUid)
	if err != nil {
		if strings.Contains(err.Error(), "Not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, facility)
}

// GetFacilityPublic gets a facility publicly (no auth) by mfl_uid
func GetFacilityPublic(c *gin.Context) {
	mflUid := c.Param("id")

	facility, err := services.GetFacility(mflUid)
	if err != nil {
		if strings.Contains(err.Error(), "Not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, facility)
}

// CreateFacility creates a new facility
func CreateFacility(c *gin.Context) {
	var facility models.Facility

	if err := c.ShouldBindJSON(&facility); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := services.CreateFacility(&facility)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// UpdateFacility updates a facility
func UpdateFacility(c *gin.Context) {
	id := c.Param("id")

	var facility models.Facility
	if err := c.ShouldBindJSON(&facility); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := services.UpdateFacility(id, &facility)
	if err != nil {
		if strings.Contains(err.Error(), "Not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteFacility deletes a facility
func DeleteFacility(c *gin.Context) {
	id := c.Param("id")

	err := services.DeleteFacility(id)
	if err != nil {
		if strings.Contains(err.Error(), "Not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// UploadFacilitiesRequest is the JSON body for bulk facility upload
type UploadFacilitiesRequest struct {
	Facilities []services.UploadFacilityRow `json:"facilities"`
}

// UploadFacilities handles POST /facilities/upload - bulk create facilities from Excel/CSV data; assigns unique mfl_uid per row (uses provided id if not yet assigned)
func UploadFacilities(c *gin.Context) {
	var body UploadFacilitiesRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}
	if len(body.Facilities) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "facilities array is required and cannot be empty"})
		return
	}

	result, err := services.UploadFacilities(body.Facilities)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"total":        result.Total,
		"created":      result.Created,
		"failed":       result.Failed,
		"errors":       result.Errors,
		"created_rows": result.CreatedRows,
	})
}
