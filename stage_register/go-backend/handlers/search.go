package handlers

import (
	"net/http"
	"strings"

	"go-backend/configs"

	"github.com/gin-gonic/gin"
)

// SearchResult represents a unified search result from the Registry
type SearchResult struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Meta        string `json:"meta,omitempty"`
}

// SearchRegistry searches across facilities, users, and admin units
func SearchRegistry(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	searchTerm := "%" + strings.ToLower(query) + "%"
	var results []SearchResult

	// Search facilities
	rows, err := configs.DB.Query(`
		SELECT id::TEXT, name, COALESCE(identifier, ''), COALESCE(district, '')
		FROM mfl_details
		WHERE LOWER(name) LIKE $1 OR LOWER(COALESCE(identifier, '')) LIKE $1 OR LOWER(COALESCE(district, '')) LIKE $1
		LIMIT 10
	`, searchTerm)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var identifier, district string
			if err := rows.Scan(&r.ID, &r.Title, &identifier, &district); err != nil {
				continue
			}
			r.Type = "facility"
			r.Description = district
			r.Meta = identifier
			results = append(results, r)
		}
	}

	// Search users (staff)
	rows, err = configs.DB.Query(`
		SELECT id::TEXT, COALESCE(first_name, '') || ' ' || COALESCE(last_name, ''), COALESCE(email, ''), COALESCE(role, '')
		FROM users
		WHERE LOWER(COALESCE(first_name, '')) LIKE $1 OR LOWER(COALESCE(last_name, '')) LIKE $1 OR LOWER(email) LIKE $1 OR LOWER(username) LIKE $1
		LIMIT 10
	`, searchTerm)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var email, role string
			if err := rows.Scan(&r.ID, &r.Title, &email, &role); err != nil {
				continue
			}
			r.Type = "staff"
			r.Description = email
			r.Meta = role
			results = append(results, r)
		}
	}

	// Search admin units (organizations/territories)
	rows, err = configs.DB.Query(`
		SELECT id::TEXT, name, COALESCE(code, ''), COALESCE(path, '')
		FROM admin_units
		WHERE LOWER(name) LIKE $1 OR LOWER(COALESCE(code, '')) LIKE $1 OR LOWER(COALESCE(path, '')) LIKE $1
		LIMIT 10
	`, searchTerm)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var code, path string
			if err := rows.Scan(&r.ID, &r.Title, &code, &path); err != nil {
				continue
			}
			r.Type = "organization"
			r.Description = path
			r.Meta = code
			results = append(results, r)
		}
	}

	// Search authority (agencies)
	rows, err = configs.DB.Query(`
		SELECT id::TEXT, name, COALESCE(code, ''), COALESCE(description, '')
		FROM authority
		WHERE LOWER(name) LIKE $1 OR LOWER(COALESCE(code, '')) LIKE $1
		LIMIT 5
	`, searchTerm)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var code, desc string
			if err := rows.Scan(&r.ID, &r.Title, &code, &desc); err != nil {
				continue
			}
			r.Type = "organization"
			r.Description = desc
			r.Meta = code
			results = append(results, r)
		}
	}

	if results == nil {
		results = []SearchResult{}
	}
	c.JSON(http.StatusOK, results)
}
