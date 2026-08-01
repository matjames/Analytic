package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"

	"github.com/gin-gonic/gin"
)

// ListAdminLevels - GET /admin-level - List all admin levels (requires auth)
func ListAdminLevels(c *gin.Context) {
	rows, err := configs.DB.Query("SELECT id, mfl_uid, name, level_number, \"createdAt\", \"updatedAt\" FROM admin_level ORDER BY level_number ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var levels []models.AdminLevel
	for rows.Next() {
		var l models.AdminLevel
		var createdAt, updatedAt utils.FlexibleTime
		err := rows.Scan(&l.ID, &l.MflUid, &l.Name, &l.LevelNumber, &createdAt, &updatedAt)
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

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, levels)
}

// ListAdminLevelsPublic - GET /adminlevel/public - List all admin levels (public, no auth required)
func ListAdminLevelsPublic(c *gin.Context) {
	rows, err := configs.DB.Query("SELECT id, mfl_uid, name, level_number, \"createdAt\", \"updatedAt\" FROM admin_level ORDER BY level_number ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var levels []models.AdminLevel
	for rows.Next() {
		var l models.AdminLevel
		var createdAt, updatedAt utils.FlexibleTime
		err := rows.Scan(&l.ID, &l.MflUid, &l.Name, &l.LevelNumber, &createdAt, &updatedAt)
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

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, levels)
}

// CreateAdminLevel - POST /admin-level - Create a new admin level (requires auth)
func CreateAdminLevel(c *gin.Context) {
	var level models.AdminLevel

	if err := c.BindJSON(&level); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get max level_number
	var maxLevel sql.NullInt64
	err := configs.DB.QueryRow("SELECT MAX(level_number) FROM admin_level").Scan(&maxLevel)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	nextLevelNumber := 1
	if maxLevel.Valid {
		nextLevelNumber = int(maxLevel.Int64) + 1
	}

	// Generate unique mfl_uid
	uid, err := utils.EnsureUniqueMflUid(configs.DB, "admin_level")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate unique mfl_uid"})
		return
	}

	// Insert into database
	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		`INSERT INTO admin_level (name, mfl_uid, level_number, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, NOW(), NOW())
		 RETURNING id, mfl_uid, name, level_number, "createdAt", "updatedAt"`,
		level.Name, uid, nextLevelNumber,
	).Scan(&level.ID, &level.MflUid, &level.Name, &level.LevelNumber, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			level.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			level.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, level)
}

// UpdateAdminLevel - PUT /admin-level/:id - Update an admin level (requires auth)
func UpdateAdminLevel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if level exists
	var existingID int64
	err = configs.DB.QueryRow("SELECT id FROM admin_level WHERE id = $1 LIMIT 1", id).Scan(&existingID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var level models.AdminLevel
	if err := c.BindJSON(&level); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update the level
	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		`UPDATE admin_level
		 SET name = $1,
		     "updatedAt" = NOW()
		 WHERE id = $2
		 RETURNING id, mfl_uid, name, level_number, "createdAt", "updatedAt"`,
		level.Name, id,
	).Scan(&level.ID, &level.MflUid, &level.Name, &level.LevelNumber, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			level.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			level.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, level)
}

// DeleteAdminLevel - DELETE /admin-level/:id - Delete an admin level (requires auth)
func DeleteAdminLevel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if level is used by admin_units
	var count int
	err = configs.DB.QueryRow("SELECT COUNT(*) FROM admin_units WHERE level_id = $1", id).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Level has units; move/delete units first."})
		return
	}

	// Get the level_number to be deleted
	var levelNumber int
	err = configs.DB.QueryRow("SELECT level_number FROM admin_level WHERE id = $1 LIMIT 1", id).Scan(&levelNumber)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delete the level
	_, err = configs.DB.Exec("DELETE FROM admin_level WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Close gap: shift down levels above the removed one
	_, err = configs.DB.Exec("UPDATE admin_level SET level_number = level_number - 1 WHERE level_number > $1", levelNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ReorderAdminLevels - POST /admin-level/reorder - Reorder levels by array of IDs (requires auth)
func ReorderAdminLevels(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids array required"})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids array required"})
		return
	}

	// Verify all IDs exist
	placeholders := ""
	args := make([]interface{}, len(req.IDs))
	for i, id := range req.IDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "$" + strconv.Itoa(i+1)
		args[i] = id
	}

	rows, err := configs.DB.Query(
		`SELECT id FROM admin_level WHERE id IN (`+placeholders+`)`,
		args...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var foundIDs []int64
	for rows.Next() {
		var foundID int64
		rows.Scan(&foundID)
		foundIDs = append(foundIDs, foundID)
	}

	if len(foundIDs) != len(req.IDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Some level IDs not found"})
		return
	}

	// Get the maximum level_number to use as offset
	var maxLevel sql.NullInt64
	err = configs.DB.QueryRow("SELECT MAX(level_number) FROM admin_level").Scan(&maxLevel)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	maxLevelNumber := 0
	if maxLevel.Valid {
		maxLevelNumber = int(maxLevel.Int64)
	}
	offset := maxLevelNumber + 1000

	// First, set all level_numbers to offset values to avoid unique constraint conflicts
	for i, id := range req.IDs {
		_, err = configs.DB.Exec(
			"UPDATE admin_level SET level_number = $1 WHERE id = $2",
			offset+i, id,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Then assign the correct sequential numbers
	for i, id := range req.IDs {
		_, err = configs.DB.Exec(
			"UPDATE admin_level SET level_number = $1 WHERE id = $2",
			i+1, id,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Return all levels ordered by level_number
	rows, err = configs.DB.Query("SELECT id, mfl_uid, name, level_number, \"createdAt\", \"updatedAt\" FROM admin_level ORDER BY level_number ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var levels []models.AdminLevel
	for rows.Next() {
		var l models.AdminLevel
		var createdAt, updatedAt utils.FlexibleTime
		err := rows.Scan(&l.ID, &l.MflUid, &l.Name, &l.LevelNumber, &createdAt, &updatedAt)
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

	c.JSON(http.StatusOK, levels)
}
