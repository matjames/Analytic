package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"

	"github.com/gin-gonic/gin"
)

// formatAdminUnit formats admin unit with admin_level info
func formatAdminUnit(unit models.AdminUnit) models.AdminUnit {
	return unit
}

// computePath computes the materialized path for an admin unit (no trailing slash)
func computePath(mflUid string, parentID *int64) (string, error) {
	if parentID == nil {
		return "/" + mflUid, nil
	}

	var parentPath string
	err := configs.DB.QueryRow("SELECT path FROM admin_units WHERE id = $1 LIMIT 1", *parentID).Scan(&parentPath)
	if err == sql.ErrNoRows {
		return "/" + mflUid, nil
	}
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSuffix(parentPath, "/")
	if trimmed == "" {
		return "/" + mflUid, nil
	}
	return trimmed + "/" + mflUid, nil
}

// ListAdminUnits - GET /admin-units - List all admin units (public endpoint)
func ListAdminUnits(c *gin.Context) {
	level := c.Query("level")
	mfluid := c.Query("mfluid")
	q := c.Query("q")

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	paramCount := 1

	if level != "" {
		levelValue, _ := strconv.ParseInt(level, 10, 64)
		whereClause += " AND (al.level_number = $" + strconv.Itoa(paramCount) + " OR al.id = $" + strconv.Itoa(paramCount) + ")"
		args = append(args, levelValue)
		paramCount++
	}
	if mfluid != "" {
		whereClause += " AND au.mfl_uid = $" + strconv.Itoa(paramCount)
		args = append(args, mfluid)
		paramCount++
	}
	if q != "" {
		whereClause += " AND au.name ILIKE $" + strconv.Itoa(paramCount)
		args = append(args, "%"+q+"%")
		paramCount++
	}

	query := `
		SELECT au.id, au.name, au.code, au.mfl_uid, au.parent_id, au.level_id, au.path, au."createdAt", au."updatedAt",
		       al.id as "admin_level.id",
		       al.name as "admin_level.name",
		       al.level_number as "admin_level.level_number",
		       al.mfl_uid as "admin_level.mfl_uid"
		FROM admin_units au
		JOIN admin_level al ON au.level_id = al.id
		` + whereClause + `
		ORDER BY au.id ASC
	`

	rows, err := configs.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var units []models.AdminUnit
	for rows.Next() {
		var unit models.AdminUnit
		var adminLevelID sql.NullInt64
		var adminLevelName sql.NullString
		var adminLevelNumber sql.NullInt64
		var adminLevelMflUid sql.NullString
		var createdAt, updatedAt utils.FlexibleTime

		err := rows.Scan(
			&unit.ID, &unit.Name, &unit.Code, &unit.MflUid, &unit.ParentID, &unit.LevelID, &unit.Path,
			&createdAt, &updatedAt,
			&adminLevelID, &adminLevelName, &adminLevelNumber, &adminLevelMflUid,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			unit.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			unit.UpdatedAt = updatedAt.Time
		}

		if adminLevelID.Valid {
			unit.AdminLevel = &models.AdminLevel{
				ID:          adminLevelID.Int64,
				Name:        adminLevelName.String,
				LevelNumber: int(adminLevelNumber.Int64),
				MflUid:      adminLevelMflUid.String,
			}
		}

		units = append(units, unit)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, units)
}

// ListAdminUnitsMfl - GET /mfl/adminunits - List all admin units for API integration (excludes id)
func ListAdminUnitsMfl(c *gin.Context) {
	level := c.Query("level")
	mfluid := c.Query("mfluid")
	q := c.Query("q")

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	paramCount := 1

	if level != "" {
		levelValue, _ := strconv.ParseInt(level, 10, 64)
		whereClause += " AND (al.level_number = $" + strconv.Itoa(paramCount) + " OR al.id = $" + strconv.Itoa(paramCount) + ")"
		args = append(args, levelValue)
		paramCount++
	}
	if mfluid != "" {
		whereClause += " AND au.mfl_uid = $" + strconv.Itoa(paramCount)
		args = append(args, mfluid)
		paramCount++
	}
	if q != "" {
		whereClause += " AND au.name ILIKE $" + strconv.Itoa(paramCount)
		args = append(args, "%"+q+"%")
		paramCount++
	}

	query := `
		SELECT au.id, au.name, au.mfl_uid, au.parent_id, au.level_id, au.path, au."createdAt", au."updatedAt",
		       al.id as "admin_level.id",
		       al.name as "admin_level.name",
		       al.level_number as "admin_level.level_number",
		       al.mfl_uid as "admin_level.mfl_uid",
		       parent.id as "parent.id",
		       parent.name as "parent.name",
		       parent.mfl_uid as "parent.mfl_uid"
		FROM admin_units au
		JOIN admin_level al ON au.level_id = al.id
		LEFT JOIN admin_units parent ON au.parent_id = parent.id
		` + whereClause + `
		ORDER BY au.id ASC
	`

	rows, err := configs.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var units []*models.AdminUnitMflResponse
	for rows.Next() {
		var unit models.AdminUnit
		var adminLevelID sql.NullInt64
		var adminLevelName sql.NullString
		var adminLevelNumber sql.NullInt64
		var adminLevelMflUid sql.NullString
		var parentID sql.NullInt64
		var parentName sql.NullString
		var parentMflUid sql.NullString
		var createdAt, updatedAt utils.FlexibleTime

		err := rows.Scan(
			&unit.ID, &unit.Name, &unit.MflUid, &unit.ParentID, &unit.LevelID, &unit.Path,
			&createdAt, &updatedAt,
			&adminLevelID, &adminLevelName, &adminLevelNumber, &adminLevelMflUid,
			&parentID, &parentName, &parentMflUid,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			unit.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			unit.UpdatedAt = updatedAt.Time
		}

		if adminLevelID.Valid {
			unit.AdminLevel = &models.AdminLevel{
				ID:          adminLevelID.Int64,
				Name:        adminLevelName.String,
				LevelNumber: int(adminLevelNumber.Int64),
				MflUid:      adminLevelMflUid.String,
			}
		}

		var parent *models.AdminUnit
		if parentID.Valid {
			parent = &models.AdminUnit{
				ID:     parentID.Int64,
				Name:   parentName.String,
				MflUid: parentMflUid.String,
			}
		}

		units = append(units, unit.ToAdminUnitMflResponse(parent))
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, units)
}

// ListDistrictsPublic - GET /adminunits/districts/public - List all districts (level 3) - public endpoint
func ListDistrictsPublic(c *gin.Context) {
	query := `
		SELECT au.id, au.name, au.code, au.mfl_uid, au.parent_id, au.level_id, au.path, au."createdAt", au."updatedAt",
		       al.id as "admin_level.id",
		       al.name as "admin_level.name",
		       al.level_number as "admin_level.level_number",
		       al.mfl_uid as "admin_level.mfl_uid"
		FROM admin_units au
		JOIN admin_level al ON au.level_id = al.id
		WHERE al.level_number = 3
		ORDER BY au.name ASC
	`

	rows, err := configs.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var units []models.AdminUnit
	for rows.Next() {
		var unit models.AdminUnit
		var adminLevelID sql.NullInt64
		var adminLevelName sql.NullString
		var adminLevelNumber sql.NullInt64
		var adminLevelMflUid sql.NullString
		var createdAt, updatedAt utils.FlexibleTime

		err := rows.Scan(
			&unit.ID, &unit.Name, &unit.Code, &unit.MflUid, &unit.ParentID, &unit.LevelID, &unit.Path,
			&createdAt, &updatedAt,
			&adminLevelID, &adminLevelName, &adminLevelNumber, &adminLevelMflUid,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			unit.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			unit.UpdatedAt = updatedAt.Time
		}

		if adminLevelID.Valid {
			unit.AdminLevel = &models.AdminLevel{
				ID:          adminLevelID.Int64,
				Name:        adminLevelName.String,
				LevelNumber: int(adminLevelNumber.Int64),
				MflUid:      adminLevelMflUid.String,
			}
		}

		units = append(units, unit)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, units)
}

// ListAdminUnitsPaged - GET /admin-units/paged - List with pagination (requires auth)
func ListAdminUnitsPaged(c *gin.Context) {
	level := c.Query("level")
	mfluid := c.Query("mfluid")
	q := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

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

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	paramCount := 1

	if level != "" {
		levelValue, _ := strconv.ParseInt(level, 10, 64)
		whereClause += " AND (al.level_number = $" + strconv.Itoa(paramCount) + " OR al.id = $" + strconv.Itoa(paramCount) + ")"
		args = append(args, levelValue)
		paramCount++
	}
	if mfluid != "" {
		whereClause += " AND au.mfl_uid = $" + strconv.Itoa(paramCount)
		args = append(args, mfluid)
		paramCount++
	}
	if q != "" {
		whereClause += " AND au.name ILIKE $" + strconv.Itoa(paramCount)
		args = append(args, "%"+q+"%")
		paramCount++
	}

	// Count query
	var total int
	countQuery := "SELECT COUNT(*) FROM admin_units au JOIN admin_level al ON au.level_id = al.id " + whereClause
	err := configs.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Data query
	args = append(args, pageSize, offset)
	dataQuery := `
		SELECT au.id, au.name, au.code, au.mfl_uid, au.parent_id, au.level_id, au.path, au."createdAt", au."updatedAt",
		       al.id as "admin_level.id",
		       al.name as "admin_level.name",
		       al.level_number as "admin_level.level_number",
		       al.mfl_uid as "admin_level.mfl_uid"
		FROM admin_units au
		JOIN admin_level al ON au.level_id = al.id
		` + whereClause + `
		ORDER BY au.id ASC
		LIMIT $` + strconv.Itoa(paramCount) + ` OFFSET $` + strconv.Itoa(paramCount+1)

	rows, err := configs.DB.Query(dataQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var units []models.AdminUnit
	for rows.Next() {
		var unit models.AdminUnit
		var adminLevelID sql.NullInt64
		var adminLevelName sql.NullString
		var adminLevelNumber sql.NullInt64
		var adminLevelMflUid sql.NullString
		var createdAt, updatedAt utils.FlexibleTime

		err := rows.Scan(
			&unit.ID, &unit.Name, &unit.Code, &unit.MflUid, &unit.ParentID, &unit.LevelID, &unit.Path,
			&createdAt, &updatedAt,
			&adminLevelID, &adminLevelName, &adminLevelNumber, &adminLevelMflUid,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			unit.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			unit.UpdatedAt = updatedAt.Time
		}

		if adminLevelID.Valid {
			unit.AdminLevel = &models.AdminLevel{
				ID:          adminLevelID.Int64,
				Name:        adminLevelName.String,
				LevelNumber: int(adminLevelNumber.Int64),
				MflUid:      adminLevelMflUid.String,
			}
		}

		units = append(units, unit)
	}

	c.JSON(http.StatusOK, gin.H{
		"rows":     units,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CreateAdminUnit - POST /admin-units - Create admin unit (requires auth)
func CreateAdminUnit(c *gin.Context) {
	var unit models.AdminUnit

	if err := c.BindJSON(&unit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate unique mfl_uid
	uid, err := utils.EnsureUniqueMflUid(configs.DB, "admin_units")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate unique mfl_uid"})
		return
	}

	// Compute path
	path, err := computePath(uid, unit.ParentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If parent_id is provided, verify it exists
	if unit.ParentID != nil {
		var parentExists int64
		err = configs.DB.QueryRow("SELECT id FROM admin_units WHERE id = $1 LIMIT 1", *unit.ParentID).Scan(&parentExists)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parent unit not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Insert into database
	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		`INSERT INTO admin_units (name, code, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		 RETURNING id, name, code, mfl_uid, parent_id, level_id, path, "createdAt", "updatedAt"`,
		unit.Name, unit.Code, uid, unit.LevelID, unit.ParentID, path,
	).Scan(&unit.ID, &unit.Name, &unit.Code, &unit.MflUid, &unit.ParentID, &unit.LevelID, &unit.Path, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			unit.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			unit.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, unit)
}

// UpdateAdminUnit - PUT /admin-units/:id - Update admin unit (requires auth)
func UpdateAdminUnit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if unit exists
	var existingID int64
	err = configs.DB.QueryRow("SELECT id FROM admin_units WHERE id = $1 LIMIT 1", id).Scan(&existingID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var unit models.AdminUnit
	if err := c.BindJSON(&unit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update query
	var fields []string
	var values []interface{}
	paramCount := 1

	if unit.Name != "" {
		fields = append(fields, "name = $"+strconv.Itoa(paramCount))
		values = append(values, unit.Name)
		paramCount++
	}
	if unit.Code != nil {
		fields = append(fields, "code = $"+strconv.Itoa(paramCount))
		values = append(values, *unit.Code)
		paramCount++
	}
	if unit.LevelID > 0 {
		fields = append(fields, "level_id = $"+strconv.Itoa(paramCount))
		values = append(values, unit.LevelID)
		paramCount++
	}
	if unit.ParentID != nil {
		fields = append(fields, "parent_id = $"+strconv.Itoa(paramCount))
		if *unit.ParentID == 0 {
			values = append(values, nil)
		} else {
			values = append(values, *unit.ParentID)
		}
		paramCount++
	}

	if len(fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	fields = append(fields, `"updatedAt" = NOW()`)
	values = append(values, id)

	query := "UPDATE admin_units SET " + strings.Join(fields, ", ") + " WHERE id = $" + strconv.Itoa(paramCount)

	_, err = configs.DB.Exec(query, values...)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch updated unit with admin_level
	var updatedUnit models.AdminUnit
	var adminLevelID sql.NullInt64
	var adminLevelName sql.NullString
	var adminLevelNumber sql.NullInt64
	var adminLevelMflUid sql.NullString
	var createdAt, updatedAt utils.FlexibleTime

	err = configs.DB.QueryRow(
		`SELECT au.id, au.name, au.code, au.mfl_uid, au.parent_id, au.level_id, au.path, au."createdAt", au."updatedAt",
		        al.id as "admin_level.id",
		        al.name as "admin_level.name",
		        al.level_number as "admin_level.level_number",
		        al.mfl_uid as "admin_level.mfl_uid"
		 FROM admin_units au
		 JOIN admin_level al ON au.level_id = al.id
		 WHERE au.id = $1
		 LIMIT 1`,
		id,
	).Scan(
		&updatedUnit.ID, &updatedUnit.Name, &updatedUnit.Code, &updatedUnit.MflUid, &updatedUnit.ParentID,
		&updatedUnit.LevelID, &updatedUnit.Path, &createdAt, &updatedAt,
		&adminLevelID, &adminLevelName, &adminLevelNumber, &adminLevelMflUid,
	)
	if err == nil {
		if createdAt.Valid {
			updatedUnit.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			updatedUnit.UpdatedAt = updatedAt.Time
		}
	}

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if adminLevelID.Valid {
		updatedUnit.AdminLevel = &models.AdminLevel{
			ID:          adminLevelID.Int64,
			Name:        adminLevelName.String,
			LevelNumber: int(adminLevelNumber.Int64),
			MflUid:      adminLevelMflUid.String,
		}
	}

	c.JSON(http.StatusOK, updatedUnit)
}

// DeleteAdminUnit - DELETE /admin-units/:id - Delete admin unit (requires auth)
func DeleteAdminUnit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if unit exists
	var unit models.AdminUnit
	err = configs.DB.QueryRow("SELECT id, path FROM admin_units WHERE id = $1 LIMIT 1", id).Scan(&unit.ID, &unit.Path)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cascade := c.Query("cascade")
	if cascade == "true" {
		if unit.Path != "" {
			_, err = configs.DB.Exec("DELETE FROM admin_units WHERE path LIKE $1", unit.Path+"%")
		} else {
			_, err = configs.DB.Exec("DELETE FROM admin_units WHERE parent_id = $1", id)
			if err == nil {
				_, err = configs.DB.Exec("DELETE FROM admin_units WHERE id = $1", id)
			}
		}
	} else {
		_, err = configs.DB.Exec("DELETE FROM admin_units WHERE id = $1", id)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// MoveAdminUnit - POST /admin-units/:id/move - Move unit to new parent (requires auth)
func MoveAdminUnit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		NewParentID *int64 `json:"newParentId"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get unit
	var unit models.AdminUnit
	err = configs.DB.QueryRow("SELECT id, mfl_uid FROM admin_units WHERE id = $1 LIMIT 1", id).Scan(&unit.ID, &unit.MflUid)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update parent_id
	_, err = configs.DB.Exec("UPDATE admin_units SET parent_id = $1 WHERE id = $2", req.NewParentID, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get new parent path
	var parentPath string
	if req.NewParentID != nil {
		err = configs.DB.QueryRow("SELECT path FROM admin_units WHERE id = $1 LIMIT 1", *req.NewParentID).Scan(&parentPath)
		if err != nil {
			parentPath = "/"
		}
	} else {
		parentPath = "/"
	}

	// Update path for this unit (no trailing slash)
	parentTrimmed := strings.TrimSuffix(parentPath, "/")
	if parentTrimmed == "" {
		parentTrimmed = "/"
	}
	newPath := parentTrimmed + "/" + unit.MflUid
	_, err = configs.DB.Exec("UPDATE admin_units SET path = $1 WHERE id = $2", newPath, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update paths for all descendants recursively (no trailing slash)
	var updateDescendants func(int64, string) error
	updateDescendants = func(nodeID int64, parentPath string) error {
		rows, err := configs.DB.Query("SELECT id, mfl_uid FROM admin_units WHERE parent_id = $1", nodeID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var childID int64
			var childMflUid string
			if err := rows.Scan(&childID, &childMflUid); err != nil {
				continue
			}

			childPath := parentPath + "/" + childMflUid
			_, err = configs.DB.Exec("UPDATE admin_units SET path = $1 WHERE id = $2", childPath, childID)
			if err != nil {
				return err
			}

			if err := updateDescendants(childID, childPath); err != nil {
				return err
			}
		}
		return nil
	}

	if err := updateDescendants(id, newPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetAdminUnitDescendants - GET /admin-units/:uid/descendants - Get descendants (requires auth)
func GetAdminUnitDescendants(c *gin.Context) {
	uid := c.Param("uid")
	levelID := c.Query("levelId")
	includeSelf := c.Query("includeSelf")

	// Get unit
	var unit models.AdminUnit
	err := configs.DB.QueryRow("SELECT id, mfl_uid, path FROM admin_units WHERE mfl_uid = $1 LIMIT 1", uid).Scan(&unit.ID, &unit.MflUid, &unit.Path)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if unit.Path == "" {
		c.JSON(http.StatusOK, []models.AdminUnit{})
		return
	}

	query := `
		SELECT au.id, au.name, au.code, au.mfl_uid, au.parent_id, au.level_id, au.path, au."createdAt", au."updatedAt",
		       al.id as "admin_level.id",
		       al.name as "admin_level.name",
		       al.level_number as "admin_level.level_number",
		       al.mfl_uid as "admin_level.mfl_uid"
		FROM admin_units au
		JOIN admin_level al ON au.level_id = al.id
		WHERE au.path LIKE $1
	`
	args := []interface{}{unit.Path + "%"}
	paramCount := 2

	if includeSelf != "true" {
		query += " AND au.mfl_uid != $" + strconv.Itoa(paramCount)
		args = append(args, unit.MflUid)
		paramCount++
	}
	if levelID != "" {
		levelIDInt, _ := strconv.ParseInt(levelID, 10, 64)
		query += " AND au.level_id = $" + strconv.Itoa(paramCount)
		args = append(args, levelIDInt)
		paramCount++
	}

	query += " ORDER BY au.level_id ASC, au.name ASC"

	rows, err := configs.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var units []models.AdminUnit
	for rows.Next() {
		var u models.AdminUnit
		var adminLevelID sql.NullInt64
		var adminLevelName sql.NullString
		var adminLevelNumber sql.NullInt64
		var adminLevelMflUid sql.NullString
		var createdAt, updatedAt utils.FlexibleTime

		err := rows.Scan(
			&u.ID, &u.Name, &u.Code, &u.MflUid, &u.ParentID, &u.LevelID, &u.Path,
			&createdAt, &updatedAt,
			&adminLevelID, &adminLevelName, &adminLevelNumber, &adminLevelMflUid,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			u.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			u.UpdatedAt = updatedAt.Time
		}

		if adminLevelID.Valid {
			u.AdminLevel = &models.AdminLevel{
				ID:          adminLevelID.Int64,
				Name:        adminLevelName.String,
				LevelNumber: int(adminLevelNumber.Int64),
				MflUid:      adminLevelMflUid.String,
			}
		}

		units = append(units, u)
	}

	c.JSON(http.StatusOK, units)
}

// GetAdminUnitAncestors - GET /admin-units/:uid/ancestors - Get ancestors (requires auth)
func GetAdminUnitAncestors(c *gin.Context) {
	uid := c.Param("uid")
	includeSelf := c.Query("includeSelf")

	// Get unit
	var unit models.AdminUnit
	err := configs.DB.QueryRow("SELECT id, mfl_uid, path FROM admin_units WHERE mfl_uid = $1 LIMIT 1", uid).Scan(&unit.ID, &unit.MflUid, &unit.Path)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if unit.Path == "" || unit.Path == "/"+unit.MflUid+"/" {
		if includeSelf == "true" {
			c.JSON(http.StatusOK, []models.AdminUnit{unit})
		} else {
			c.JSON(http.StatusOK, []models.AdminUnit{})
		}
		return
	}

	// Extract UIDs from path
	pathParts := strings.Split(strings.Trim(unit.Path, "/"), "/")
	var ancestorUids []string
	for _, part := range pathParts {
		if part != "" {
			if includeSelf == "true" || part != unit.MflUid {
				ancestorUids = append(ancestorUids, part)
			}
		}
	}

	if len(ancestorUids) == 0 {
		c.JSON(http.StatusOK, []models.AdminUnit{})
		return
	}

	// Build placeholders
	placeholders := ""
	args := make([]interface{}, len(ancestorUids))
	for i, u := range ancestorUids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "$" + strconv.Itoa(i+1)
		args[i] = u
	}

	query := `
		SELECT au.id, au.name, au.code, au.mfl_uid, au.parent_id, au.level_id, au.path, au."createdAt", au."updatedAt",
		       al.id as "admin_level.id",
		       al.name as "admin_level.name",
		       al.level_number as "admin_level.level_number",
		       al.mfl_uid as "admin_level.mfl_uid"
		FROM admin_units au
		JOIN admin_level al ON au.level_id = al.id
		WHERE au.mfl_uid IN (` + placeholders + `)
		ORDER BY al.level_number ASC
	`

	rows, err := configs.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	unitMap := make(map[string]models.AdminUnit)

	for rows.Next() {
		var u models.AdminUnit
		var adminLevelID sql.NullInt64
		var adminLevelName sql.NullString
		var adminLevelNumber sql.NullInt64
		var adminLevelMflUid sql.NullString
		var createdAt, updatedAt utils.FlexibleTime

		err := rows.Scan(
			&u.ID, &u.Name, &u.Code, &u.MflUid, &u.ParentID, &u.LevelID, &u.Path,
			&createdAt, &updatedAt,
			&adminLevelID, &adminLevelName, &adminLevelNumber, &adminLevelMflUid,
		)
		if err != nil {
			continue
		}
		if createdAt.Valid {
			u.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			u.UpdatedAt = updatedAt.Time
		}

		if adminLevelID.Valid {
			u.AdminLevel = &models.AdminLevel{
				ID:          adminLevelID.Int64,
				Name:        adminLevelName.String,
				LevelNumber: int(adminLevelNumber.Int64),
				MflUid:      adminLevelMflUid.String,
			}
		}

		unitMap[u.MflUid] = u
	}

	// Sort by original order
	var sortedUnits []models.AdminUnit
	for _, uid := range ancestorUids {
		if u, ok := unitMap[uid]; ok {
			sortedUnits = append(sortedUnits, u)
		}
	}

	c.JSON(http.StatusOK, sortedUnits)
}

// GetAdminUnitsTree - GET /admin-units/tree - Get tree structure (requires auth)
func GetAdminUnitsTree(c *gin.Context) {
	// Get all levels
	levelRows, err := configs.DB.Query("SELECT id, mfl_uid, name, level_number, \"createdAt\", \"updatedAt\" FROM admin_level ORDER BY level_number ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer levelRows.Close()

	var levels []models.AdminLevel
	for levelRows.Next() {
		var l models.AdminLevel
		var createdAt, updatedAt utils.FlexibleTime
		err := levelRows.Scan(&l.ID, &l.MflUid, &l.Name, &l.LevelNumber, &createdAt, &updatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			l.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			l.UpdatedAt = updatedAt.Time
		}
		levels = append(levels, l)
	}

	// Get all units
	unitRows, err := configs.DB.Query("SELECT id, name, code, mfl_uid, parent_id, level_id, path, \"createdAt\", \"updatedAt\" FROM admin_units ORDER BY id ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer unitRows.Close()

	var units []models.AdminUnit
	for unitRows.Next() {
		var u models.AdminUnit
		var createdAt, updatedAt utils.FlexibleTime
		err := unitRows.Scan(&u.ID, &u.Name, &u.Code, &u.MflUid, &u.ParentID, &u.LevelID, &u.Path, &createdAt, &updatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			u.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			u.UpdatedAt = updatedAt.Time
		}
		units = append(units, u)
	}

	// Helper to normalize path
	normalizePath := func(path string) string {
		if path == "" {
			return "/"
		}
		normalized := path
		if !strings.HasPrefix(normalized, "/") {
			normalized = "/" + normalized
		}
		if !strings.HasSuffix(normalized, "/") {
			normalized = normalized + "/"
		}
		return normalized
	}

	// Group units by parent path
	byParentPath := make(map[string][]models.AdminUnit)
	for _, u := range units {
		if u.Path == "" {
			byParentPath["/"] = append(byParentPath["/"], u)
			continue
		}

		normalizedPath := normalizePath(u.Path)
		pathParts := strings.Split(strings.Trim(normalizedPath, "/"), "/")
		if len(pathParts) == 1 || pathParts[0] == "" {
			byParentPath["/"] = append(byParentPath["/"], u)
		} else {
			parentPath := "/" + strings.Join(pathParts[:len(pathParts)-1], "/") + "/"
			byParentPath[parentPath] = append(byParentPath[parentPath], u)
		}
	}

	// Build tree recursively
	var build func(string) []map[string]interface{}
	build = func(parentPath string) []map[string]interface{} {
		children := byParentPath[parentPath]
		if len(children) == 0 {
			return []map[string]interface{}{}
		}

		result := make([]map[string]interface{}, 0, len(children))
		for _, u := range children {
			normalizedUnitPath := normalizePath(u.Path)
			if u.Path == "" {
				normalizedUnitPath = normalizePath("/" + u.MflUid + "/")
			}

			result = append(result, map[string]interface{}{
				"id":       u.ID,
				"name":     u.Name,
				"code":     u.Code,
				"mfl_uid":  u.MflUid,
				"parentId": u.ParentID,
				"levelId":  u.LevelID,
				"path":     normalizedUnitPath,
				"children": build(normalizedUnitPath),
			})
		}
		return result
	}

	c.JSON(http.StatusOK, gin.H{
		"levels": levels,
		"tree":   build("/"),
	})
}

// GetAdminUnitsTreePublic - GET /adminunits/tree/public - Get tree structure (public, no auth required)
func GetAdminUnitsTreePublic(c *gin.Context) {
	// Get all levels
	levelRows, err := configs.DB.Query("SELECT id, mfl_uid, name, level_number, \"createdAt\", \"updatedAt\" FROM admin_level ORDER BY level_number ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer levelRows.Close()

	var levels []models.AdminLevel
	for levelRows.Next() {
		var l models.AdminLevel
		var createdAt, updatedAt utils.FlexibleTime
		err := levelRows.Scan(&l.ID, &l.MflUid, &l.Name, &l.LevelNumber, &createdAt, &updatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			l.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			l.UpdatedAt = updatedAt.Time
		}
		levels = append(levels, l)
	}

	// Get all units
	unitRows, err := configs.DB.Query("SELECT id, name, code, mfl_uid, parent_id, level_id, path, \"createdAt\", \"updatedAt\" FROM admin_units ORDER BY id ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer unitRows.Close()

	var units []models.AdminUnit
	for unitRows.Next() {
		var u models.AdminUnit
		var createdAt, updatedAt utils.FlexibleTime
		err := unitRows.Scan(&u.ID, &u.Name, &u.Code, &u.MflUid, &u.ParentID, &u.LevelID, &u.Path, &createdAt, &updatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			u.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			u.UpdatedAt = updatedAt.Time
		}
		units = append(units, u)
	}

	// Helper to normalize path
	normalizePath := func(path string) string {
		if path == "" {
			return "/"
		}
		normalized := path
		if !strings.HasPrefix(normalized, "/") {
			normalized = "/" + normalized
		}
		if !strings.HasSuffix(normalized, "/") {
			normalized = normalized + "/"
		}
		return normalized
	}

	// Group units by parent path
	byParentPath := make(map[string][]models.AdminUnit)
	for _, u := range units {
		if u.Path == "" {
			byParentPath["/"] = append(byParentPath["/"], u)
			continue
		}

		normalizedPath := normalizePath(u.Path)
		pathParts := strings.Split(strings.Trim(normalizedPath, "/"), "/")
		if len(pathParts) == 1 || pathParts[0] == "" {
			byParentPath["/"] = append(byParentPath["/"], u)
		} else {
			parentPath := "/" + strings.Join(pathParts[:len(pathParts)-1], "/") + "/"
			byParentPath[parentPath] = append(byParentPath[parentPath], u)
		}
	}

	// Build tree recursively
	var build func(string) []map[string]interface{}
	build = func(parentPath string) []map[string]interface{} {
		children := byParentPath[parentPath]
		if len(children) == 0 {
			return []map[string]interface{}{}
		}

		result := make([]map[string]interface{}, 0, len(children))
		for _, u := range children {
			normalizedUnitPath := normalizePath(u.Path)
			if u.Path == "" {
				normalizedUnitPath = normalizePath("/" + u.MflUid + "/")
			}

			result = append(result, map[string]interface{}{
				"id":       u.ID,
				"name":     u.Name,
				"code":     u.Code,
				"mfl_uid":  u.MflUid,
				"parentId": u.ParentID,
				"levelId":  u.LevelID,
				"path":     normalizedUnitPath,
				"children": build(normalizedUnitPath),
			})
		}
		return result
	}

	c.JSON(http.StatusOK, gin.H{
		"levels": levels,
		"tree":   build("/"),
	})
}

// GetAdminUnitsPublic - GET /admin-units/public - Public API endpoint (requires auth)
func GetAdminUnitsPublic(c *gin.Context) {
	level := c.Query("level")
	parentUid := c.Query("parentUid")
	mfluid := c.Query("mfluid")
	q := c.Query("q")
	updatedSince := c.Query("updatedSince")
	path := c.Query("path")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "100"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 500 {
		pageSize = 500
	}

	offset := (page - 1) * pageSize

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	paramCount := 1

	if level != "" {
		levelValue, _ := strconv.ParseInt(level, 10, 64)
		whereClause += " AND (al.level_number = $" + strconv.Itoa(paramCount) + " OR al.id = $" + strconv.Itoa(paramCount) + ")"
		args = append(args, levelValue)
		paramCount++
	}
	if parentUid != "" {
		parentID, _ := strconv.ParseInt(parentUid, 10, 64)
		whereClause += " AND au.parent_id = $" + strconv.Itoa(paramCount)
		args = append(args, parentID)
		paramCount++
	}
	if mfluid != "" {
		whereClause += " AND au.mfl_uid = $" + strconv.Itoa(paramCount)
		args = append(args, mfluid)
		paramCount++
	}
	if q != "" {
		whereClause += " AND au.name ILIKE $" + strconv.Itoa(paramCount)
		args = append(args, "%"+q+"%")
		paramCount++
	}
	if updatedSince != "" {
		whereClause += " AND au.\"updatedAt\" >= $" + strconv.Itoa(paramCount)
		args = append(args, updatedSince)
		paramCount++
	}
	if path != "" {
		prefix := path
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}
		if !strings.HasSuffix(prefix, "/") {
			prefix = prefix + "/"
		}
		whereClause += " AND au.path LIKE $" + strconv.Itoa(paramCount)
		args = append(args, prefix+"%")
		paramCount++
	}

	// Count query
	var total int
	countQuery := "SELECT COUNT(*) FROM admin_units au JOIN admin_level al ON au.level_id = al.id " + whereClause
	err := configs.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Data query - only specific fields
	args = append(args, pageSize, offset)
	dataQuery := `
		SELECT au.name, au.code, au.mfl_uid, au.parent_id as parent_mfl_uid, au.path,
		       au."createdAt", au."updatedAt",
		       al.name as level_name, al.level_number
		FROM admin_units au
		JOIN admin_level al ON au.level_id = al.id
		` + whereClause + `
		ORDER BY au.id ASC
		LIMIT $` + strconv.Itoa(paramCount) + ` OFFSET $` + strconv.Itoa(paramCount+1)

	rows, err := configs.DB.Query(dataQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type PublicUnit struct {
		Name         string  `json:"name"`
		Code         *string `json:"code"`
		MflUid       string  `json:"mfl_uid"`
		ParentMflUid *int64  `json:"parent_mfl_uid"`
		Path         string  `json:"path"`
		CreatedAt    string  `json:"createdAt"`
		UpdatedAt    string  `json:"updatedAt"`
		LevelName    string  `json:"level_name"`
		LevelNumber  int     `json:"level_number"`
	}

	var data []PublicUnit
	for rows.Next() {
		var u PublicUnit
		var createdAt, updatedAt string
		err := rows.Scan(
			&u.Name, &u.Code, &u.MflUid, &u.ParentMflUid, &u.Path,
			&createdAt, &updatedAt,
			&u.LevelName, &u.LevelNumber,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		u.CreatedAt = createdAt
		u.UpdatedAt = updatedAt
		data = append(data, u)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     data,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
