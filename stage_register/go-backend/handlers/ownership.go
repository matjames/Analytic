package handlers

import (
	"net/http"
	"strconv"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"

	"github.com/gin-gonic/gin"
)

// ListOwnerships - GET /ownership - List all ownership types (public endpoint, no auth required)
func ListOwnerships(c *gin.Context) {
	rows, err := configs.DB.Query("SELECT id, mfl_uid, code, name, description, \"createdAt\", \"updatedAt\" FROM ownership ORDER BY id ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var ownerships []models.OwnershipListResponse
	for rows.Next() {
		var o models.Ownership
		var createdAt, updatedAt utils.FlexibleTime
		err := rows.Scan(&o.ID, &o.MflUid, &o.Code, &o.Name, &o.Description, &createdAt, &updatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			o.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			o.UpdatedAt = updatedAt.Time
		}
		ownerships = append(ownerships, o.ToOwnershipListResponse())
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ownerships)
}

// ListOwnershipsMfl - GET /mfl/ownership - List all ownership types for API integration (excludes id)
func ListOwnershipsMfl(c *gin.Context) {
	rows, err := configs.DB.Query("SELECT id, mfl_uid, code, name, description, \"createdAt\", \"updatedAt\" FROM ownership ORDER BY id ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var ownerships []*models.OwnershipMflResponse
	for rows.Next() {
		var o models.Ownership
		var createdAt, updatedAt utils.FlexibleTime
		err := rows.Scan(&o.ID, &o.MflUid, &o.Code, &o.Name, &o.Description, &createdAt, &updatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			o.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			o.UpdatedAt = updatedAt.Time
		}
		ownerships = append(ownerships, o.ToOwnershipMflResponse())
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ownerships)
}

// GetOwnership - GET /ownership/:id - Get a single ownership type
func GetOwnership(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var o models.Ownership
	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		"SELECT id, mfl_uid, code, name, description, \"createdAt\", \"updatedAt\" FROM ownership WHERE id = $1 LIMIT 1",
		id,
	).Scan(&o.ID, &o.MflUid, &o.Code, &o.Name, &o.Description, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			o.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			o.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	c.JSON(http.StatusOK, o)
}

// CreateOwnership - POST /ownership - Create a new ownership type
func CreateOwnership(c *gin.Context) {
	var o models.Ownership

	if err := c.BindJSON(&o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, err := utils.EnsureUniqueMflUid(configs.DB, "ownership")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate unique mfl_uid"})
		return
	}

	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		`INSERT INTO ownership (mfl_uid, code, name, description, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 RETURNING id, mfl_uid, code, name, description, "createdAt", "updatedAt"`,
		uid, o.Code, o.Name, o.Description,
	).Scan(&o.ID, &o.MflUid, &o.Code, &o.Name, &o.Description, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			o.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			o.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, o)
}

// UpdateOwnership - PUT /ownership/:id - Update an ownership type
func UpdateOwnership(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if ownership exists
	var existingID int64
	err = configs.DB.QueryRow("SELECT id FROM ownership WHERE id = $1 LIMIT 1", id).Scan(&existingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	var o models.Ownership
	if err := c.BindJSON(&o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update the ownership
	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		`UPDATE ownership
		 SET code = $1,
		     name = $2,
		     description = $3,
		     "updatedAt" = NOW()
		 WHERE id = $4
		 RETURNING id, mfl_uid, code, name, description, "createdAt", "updatedAt"`,
		o.Code, o.Name, o.Description, id,
	).Scan(&o.ID, &o.MflUid, &o.Code, &o.Name, &o.Description, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			o.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			o.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, o)
}

// DeleteOwnership - DELETE /ownership/:id - Delete an ownership type
func DeleteOwnership(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if ownership exists
	var existingID int64
	err = configs.DB.QueryRow("SELECT id FROM ownership WHERE id = $1 LIMIT 1", id).Scan(&existingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	// Delete the ownership
	_, err = configs.DB.Exec("DELETE FROM ownership WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

