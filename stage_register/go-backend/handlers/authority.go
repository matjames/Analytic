package handlers

import (
	"net/http"
	"strconv"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"

	"github.com/gin-gonic/gin"
)

// ListAuthorities - GET /authority - List all authority types (public endpoint, no auth required)
func ListAuthorities(c *gin.Context) {
	rows, err := configs.DB.Query("SELECT id, mfl_uid, code, name, description, \"createdAt\", \"updatedAt\" FROM authority ORDER BY id ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var authorities []models.AuthorityListResponse
	for rows.Next() {
		var a models.Authority
		var createdAt, updatedAt utils.FlexibleTime
		err := rows.Scan(&a.ID, &a.MflUid, &a.Code, &a.Name, &a.Description, &createdAt, &updatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			a.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			a.UpdatedAt = updatedAt.Time
		}
		authorities = append(authorities, a.ToAuthorityListResponse())
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, authorities)
}

// ListAuthoritiesMfl - GET /mfl/authority - List all authority types for API integration (excludes id)
func ListAuthoritiesMfl(c *gin.Context) {
	rows, err := configs.DB.Query("SELECT id, mfl_uid, code, name, description, \"createdAt\", \"updatedAt\" FROM authority ORDER BY id ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var authorities []*models.AuthorityMflResponse
	for rows.Next() {
		var a models.Authority
		var createdAt, updatedAt utils.FlexibleTime
		err := rows.Scan(&a.ID, &a.MflUid, &a.Code, &a.Name, &a.Description, &createdAt, &updatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if createdAt.Valid {
			a.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			a.UpdatedAt = updatedAt.Time
		}
		authorities = append(authorities, a.ToAuthorityMflResponse())
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, authorities)
}

// GetAuthority - GET /authority/:id - Get a single authority type
func GetAuthority(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var a models.Authority
	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		"SELECT id, mfl_uid, code, name, description, \"createdAt\", \"updatedAt\" FROM authority WHERE id = $1 LIMIT 1",
		id,
	).Scan(&a.ID, &a.MflUid, &a.Code, &a.Name, &a.Description, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			a.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			a.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	c.JSON(http.StatusOK, a)
}

// CreateAuthority - POST /authority - Create a new authority type
func CreateAuthority(c *gin.Context) {
	var a models.Authority

	if err := c.BindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid := utils.GenerateUid()

	var createdAt, updatedAt utils.FlexibleTime
	err := configs.DB.QueryRow(
		`INSERT INTO authority (mfl_uid, code, name, description, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 RETURNING id, mfl_uid, code, name, description, "createdAt", "updatedAt"`,
		uid, a.Code, a.Name, a.Description,
	).Scan(&a.ID, &a.MflUid, &a.Code, &a.Name, &a.Description, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			a.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			a.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, a)
}

// UpdateAuthority - PUT /authority/:id - Update an authority type
func UpdateAuthority(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if authority exists
	var existingID int64
	err = configs.DB.QueryRow("SELECT id FROM authority WHERE id = $1 LIMIT 1", id).Scan(&existingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	var a models.Authority
	if err := c.BindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update the authority
	var createdAt, updatedAt utils.FlexibleTime
	err = configs.DB.QueryRow(
		`UPDATE authority
		 SET code = $1,
		     name = $2,
		     description = $3,
		     "updatedAt" = NOW()
		 WHERE id = $4
		 RETURNING id, mfl_uid, code, name, description, "createdAt", "updatedAt"`,
		a.Code, a.Name, a.Description, id,
	).Scan(&a.ID, &a.MflUid, &a.Code, &a.Name, &a.Description, &createdAt, &updatedAt)
	if err == nil {
		if createdAt.Valid {
			a.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			a.UpdatedAt = updatedAt.Time
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, a)
}

// DeleteAuthority - DELETE /authority/:id - Delete an authority type
func DeleteAuthority(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Check if authority exists
	var existingID int64
	err = configs.DB.QueryRow("SELECT id FROM authority WHERE id = $1 LIMIT 1", id).Scan(&existingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	// Delete the authority
	_, err = configs.DB.Exec("DELETE FROM authority WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
