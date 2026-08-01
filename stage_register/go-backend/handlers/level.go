package handlers

import (
	"net/http"
	"strconv"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"

	"github.com/gin-gonic/gin"
)

// ListLevels - GET /level - List all facility levels (public endpoint, no auth required)
func ListLevels(c *gin.Context) {
	rows, err := configs.DB.Query("SELECT id, mfl_uid, code, name, description, \"createdAt\", \"updatedAt\" FROM level ORDER BY id ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var levels []models.LevelListResponse
	for rows.Next() {
		var l models.Level
		var createdAt, updatedAt utils.FlexibleTime
		err := rows.Scan(&l.ID, &l.MflUid, &l.Code, &l.Name, &l.Description, &createdAt, &updatedAt)
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
		levels = append(levels, l.ToLevelListResponse())
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, levels)
}

// ListLevelsMfl - GET /mfl/level - List all facility levels for API integration (excludes id)
func ListLevelsMfl(c *gin.Context) {
	rows, err := configs.DB.Query("SELECT id, mfl_uid, code, name, description, \"createdAt\", \"updatedAt\" FROM level ORDER BY id ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var levels []*models.LevelMflResponse
	for rows.Next() {
		var l models.Level
		var createdAt, updatedAt utils.FlexibleTime
		err := rows.Scan(&l.ID, &l.MflUid, &l.Code, &l.Name, &l.Description, &createdAt, &updatedAt)
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
		levels = append(levels, l.ToLevelMflResponse())
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, levels)
}

// GetLevel - GET /level/:id - Get a single facility level
func GetLevel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var l models.Level
	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		"SELECT id, mfl_uid, code, name, description, \"createdAt\", \"updatedAt\" FROM level WHERE id = $1 LIMIT 1",
		id,
	).Scan(&l.ID, &l.MflUid, &l.Code, &l.Name, &l.Description, &createdAt, &updatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	if createdAt.Valid {
		l.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		l.UpdatedAt = updatedAt.Time
	}

	c.JSON(http.StatusOK, l)
}

// CreateLevel - POST /level - Create a new facility level
func CreateLevel(c *gin.Context) {
	var l models.Level

	if err := c.BindJSON(&l); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, err := utils.EnsureUniqueMflUid(configs.DB, "level")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate unique mfl_uid"})
		return
	}

	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		`INSERT INTO level (mfl_uid, code, name, description, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 RETURNING id, mfl_uid, code, name, description, "createdAt", "updatedAt"`,
		uid, l.Code, l.Name, l.Description,
	).Scan(&l.ID, &l.MflUid, &l.Code, &l.Name, &l.Description, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			l.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			l.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, l)
}

// UpdateLevel - PUT /level/:id - Update a facility level
func UpdateLevel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if level exists
	var existingID int64
	err = configs.DB.QueryRow("SELECT id FROM level WHERE id = $1 LIMIT 1", id).Scan(&existingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	var l models.Level
	if err := c.BindJSON(&l); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update the level
	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		`UPDATE level
		 SET code = $1,
		     name = $2,
		     description = $3,
		     "updatedAt" = NOW()
		 WHERE id = $4
		 RETURNING id, mfl_uid, code, name, description, "createdAt", "updatedAt"`,
		l.Code, l.Name, l.Description, id,
	).Scan(&l.ID, &l.MflUid, &l.Code, &l.Name, &l.Description, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			l.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			l.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, l)
}

// DeleteLevel - DELETE /level/:id - Delete a facility level
func DeleteLevel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if level exists
	var existingID int64
	err = configs.DB.QueryRow("SELECT id FROM level WHERE id = $1 LIMIT 1", id).Scan(&existingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	// Delete the level
	_, err = configs.DB.Exec("DELETE FROM level WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
