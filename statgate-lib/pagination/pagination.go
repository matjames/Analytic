package pagination

import (
	"net/http"
	"strconv"
	"strings"
)

// Params contains universal pagination and ordering parameters.
type Params struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	SortBy   string `json:"sort_by"`
	SortDir  string `json:"sort_dir"` // "ASC" or "DESC"
	Query    string `json:"query,omitempty"`
}

// Result wraps paginated data with enterprise metadata.
type Result struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// Parse extracts pagination parameters from HTTP query with safe defaults.
func Parse(r *http.Request, defaultPageSize, maxPageSize int) Params {
	if defaultPageSize <= 0 {
		defaultPageSize = 20
	}
	if maxPageSize <= 0 {
		maxPageSize = 100
	}

	q := r.URL.Query()

	page := 1
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		page = p
	}

	pageSize := defaultPageSize
	if ps, err := strconv.Atoi(q.Get("page_size")); err == nil && ps > 0 {
		pageSize = ps
	} else if ps, err := strconv.Atoi(q.Get("limit")); err == nil && ps > 0 {
		pageSize = ps
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	sortBy := strings.TrimSpace(q.Get("sort_by"))
	if sortBy == "" {
		sortBy = strings.TrimSpace(q.Get("sort"))
	}
	if sortBy == "" {
		sortBy = "created_at"
	}

	sortDir := strings.ToUpper(strings.TrimSpace(q.Get("sort_dir")))
	if sortDir == "" {
		sortDir = strings.ToUpper(strings.TrimSpace(q.Get("order")))
	}
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "DESC"
	}

	searchQuery := strings.TrimSpace(q.Get("q"))
	if searchQuery == "" {
		searchQuery = strings.TrimSpace(q.Get("search"))
	}

	return Params{
		Page:     page,
		PageSize: pageSize,
		SortBy:   sortBy,
		SortDir:  sortDir,
		Query:    searchQuery,
	}
}

// Offset calculates SQL OFFSET.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// BuildResult wraps items into a Result structure.
func BuildResult(data interface{}, totalItems int64, p Params) Result {
	totalPages := int((totalItems + int64(p.PageSize) - 1) / int64(p.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	return Result{
		Data:       data,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    p.Page < totalPages,
		HasPrev:    p.Page > 1,
	}
}
