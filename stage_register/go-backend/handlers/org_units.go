package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"go-backend/services"

	"github.com/gin-gonic/gin"
)

func ListOrgUnits(c *gin.Context) {
	// Parse numeric parameters
	level, _ := strconv.Atoi(c.Query("level"))
	minLevel, _ := strconv.Atoi(c.Query("minLevel"))
	maxLevel, _ := strconv.Atoi(c.Query("maxLevel"))
	parentId, _ := strconv.ParseInt(c.Query("parentId"), 10, 64)
	nationalId, _ := strconv.ParseInt(c.Query("nationalId"), 10, 64)
	regionId, _ := strconv.ParseInt(c.Query("regionId"), 10, 64)
	districtIdParam := c.Query("districtId") // accepts mfl_uid (e.g. REJuxCmTwXG) or numeric id
	var districtId int64
	var districtMflUid string
	if districtIdParam != "" {
		if parsed, err := strconv.ParseInt(districtIdParam, 10, 64); err == nil {
			districtId = parsed
		} else {
			districtMflUid = districtIdParam
		}
	}
	districtName := c.Query("districtName")
	if districtName == "" {
		districtName = c.Query("district") // alias for district name
	}
	mflUid := c.Query("mflUid")
	if mflUid == "" {
		mflUid = c.Query("mfluid") // alias used by some clients
	}
	dlgId, _ := strconv.ParseInt(c.Query("dlgId"), 10, 64)
	subcountyId, _ := strconv.ParseInt(c.Query("subcountyId"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	
	// Parse boolean parameters
	var reporting *bool
	var licensed *bool
	if c.Query("reporting") != "" {
		val := c.Query("reporting") == "true"
		reporting = &val
	}
	if c.Query("licensed") != "" {
		val := c.Query("licensed") == "true"
		licensed = &val
	}

	// Build query parameters
	queryParams := services.OrgUnitsQueryParams{
		Level:             level,
		MinLevel:          minLevel,
		MaxLevel:          maxLevel,
		ParentID:          parentId,
		NationalID:        nationalId,
		RegionID:          regionId,
		RegionName:        c.Query("regionName"),
		DistrictID:        districtId,
		DistrictMflUid:    districtMflUid,
		DistrictName:      districtName,
		DLGID:             dlgId,
		SubcountyID:       subcountyId,
		SubcountyName:     c.Query("subcountyName"),
		Ownership:         c.Query("ownership"),
		Authority:         c.Query("authority"),
		FacilityLevel:     c.Query("facility_level"), // facility level type (e.g. HC III); level=6 is hierarchy level
		Status:            c.Query("status"),
		Reporting:         reporting,
		Licensed:          licensed,
		Search:            c.Query("search"),
		Name:              c.Query("name"),
		MflUid:            mflUid,
		Paging:            c.DefaultQuery("paging", "true") == "true",
		Page:              page,
		PageSize:          pageSize,
		IncludeChildren:   c.Query("includeChildren") == "true",
		IncludeFacilities: c.Query("includeFacilities") == "true",
		UpdatedSince:      c.Query("updatedSince"),
	}

	// Call service
	response, err := services.ListOrgUnits(queryParams)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Return response based on paging parameter
	if queryParams.Paging {
		c.JSON(http.StatusOK, response)
	} else {
		// Without paging, return just the org units array
		c.JSON(http.StatusOK, gin.H{
			"orgunits": response.OrgUnits,
			"total":    response.Total,
		})
	}
}

// GetOrgUnit handles GET /api/orgunits/:id - Get single organization unit
// Query parameters:
//   - includeChildren: include children (true/false)
//   - includeParent: include parent (true/false)
//   - fields: comma-separated field names
func GetOrgUnit(c *gin.Context) {
	id := c.Param("id")

	includeChildren := c.Query("includeChildren") == "true"
	includeParent := c.Query("includeParent") == "true"

	orgUnit, err := services.GetOrgUnit(id, includeChildren, includeParent)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organization unit not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orgUnit)
}

// ListOrgUnitsByLevel handles GET /api/orgunits/level/:level - Get org units by level
func ListOrgUnitsByLevel(c *gin.Context) {
	level, err := strconv.Atoi(c.Param("level"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid level parameter"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	response, err := services.ListOrgUnitsByLevel(level, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetOrgUnitsTree handles GET /api/orgunits/tree - Get organization units tree
// Query parameters:
//   - rootId: root organization unit ID (optional)
func GetOrgUnitsTree(c *gin.Context) {
	rootID := c.Query("rootId")

	tree, err := services.ListOrgUnitsTree(rootID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orgunits": tree,
	})
}

// ListOrgUnitsWithChildren handles GET /api/orgunits/:id/children
// Returns immediate children of an organization unit by ID
func ListOrgUnitsWithChildren(c *gin.Context) {
	id := c.Param("id")
	
	// First get the org unit to find its admin_unit_id
	orgUnit, err := services.GetOrgUnit(id, false, false)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organization unit not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get children using parent ID filter
	queryParams := services.OrgUnitsQueryParams{
		ParentID: orgUnit.ID,
		Paging:   false,
	}

	response, err := services.ListOrgUnits(queryParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orgunits": response.OrgUnits,
		"total":    response.Total,
	})
}

// GetDistrictFacilities handles GET /api/orgunits/district/:id/facilities
// Returns all facilities under a specific district
func GetDistrictFacilities(c *gin.Context) {
	districtId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid district ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	
	queryParams := services.OrgUnitsQueryParams{
		DistrictID: districtId,
		Level:      6, // Facilities are level 6
		Page:       page,
		PageSize:   pageSize,
		Paging:     c.DefaultQuery("paging", "true") == "true",
	}

	response, err := services.ListOrgUnits(queryParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if queryParams.Paging {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusOK, gin.H{
			"orgunits": response.OrgUnits,
			"total":    response.Total,
		})
	}
}

// GetSubcountyFacilities handles GET /api/orgunits/subcounty/:id/facilities
// Returns all facilities under a specific subcounty
func GetSubcountyFacilities(c *gin.Context) {
	subcountyId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subcounty ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	
	queryParams := services.OrgUnitsQueryParams{
		SubcountyID: subcountyId,
		Level:       6, // Facilities are level 6
		Page:        page,
		PageSize:    pageSize,
		Paging:      c.DefaultQuery("paging", "true") == "true",
	}

	response, err := services.ListOrgUnits(queryParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if queryParams.Paging {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusOK, gin.H{
			"orgunits": response.OrgUnits,
			"total":    response.Total,
		})
	}
}
