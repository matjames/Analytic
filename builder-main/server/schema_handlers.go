package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// GetAvailableSchemasHandler returns list of all schemas with tables/MVs
func GetAvailableSchemasHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	schemas, err := GetAvailableSchemas(DB)
	if err != nil {
		log.Printf("GetAvailableSchemas error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to get schemas"))
		return
	}

	json.NewEncoder(w).Encode(schemas)
}

// GetSchemaTablesHandler returns list of available tables, optionally filtered by schema
func GetSchemaTablesHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get optional schema filter from query params
	schema := r.URL.Query().Get("schema")

	tables, err := GetTablesBySchema(schema, DB)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to get tables"))
		return
	}

	json.NewEncoder(w).Encode(tables)
}

// GetTableColumnsHandler returns columns for a specific table
func GetTableColumnsHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract schema and table from URL path: /api/schema/table/{schema}/{table}/columns
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid path"))
		return
	}

	// Dispatch /preview suffix to GetTablePreviewHandler
	if len(parts) >= 7 && parts[6] == "preview" {
		GetTablePreviewHandler(w, r)
		return
	}

	schema := parts[4]
	table := parts[5]

	// Remove "/columns" suffix if present
	table = strings.TrimSuffix(table, "/columns")

	columns, err := GetTableColumns(schema, table, DB)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to get columns"))
		return
	}

	response := TableColumnsResponse{
		Table:   schema + "." + table,
		Columns: columns,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTablePreviewHandler returns sample data from a table
func GetTablePreviewHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract schema and table from URL path: /api/schema/table/{schema}/{table}/preview
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid path"))
		return
	}

	schema := parts[4]
	table := parts[5]

	// Remove "/preview" suffix if present
	table = strings.TrimSuffix(table, "/preview")

	// Get limit from query parameter
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	headers, rows, err := GetTablePreview(schema, table, limit, DB)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to get preview"))
		return
	}

	response := TablePreviewResponse{
		Headers:  headers,
		Rows:     rows,
		RowCount: len(rows),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ValidateQueryHandler validates a structured query
func ValidateQueryHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "method not allowed"))
		return
	}

	// Limit request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req QueryValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid request"))
		return
	}

	// Build query
	query, err := BuildSQLWithSquirrel(Query{
		Table:        req.Table,
		Where:        req.Where,
		Aggregations: req.Aggregations,
		GroupBy:      req.GroupBy,
		OrderBy:      req.OrderBy,
		Calculate:    req.Calculate,
	}, "table", req.Table, []string{})

	result := QueryValidationResult{
		Valid:        true,
		GeneratedSQL: query,
	}

	if err != nil {
		result.Valid = false
		result.Errors = []string{err.Error()}
	} else {
		// Add warnings for TEXT-stored numeric columns
		columns, _ := GetTableColumns("report", req.Table)
		for _, col := range columns {
			if col.IsNumeric && col.DataType == "text" {
				result.Warnings = append(result.Warnings,
					"Column '"+col.Name+"' is TEXT; will be wrapped with COALESCE/CAST for safe conversion")
			}
		}
		// Add warning for count_distinct — column will be cast to TEXT
		for _, agg := range req.Aggregations {
			if strings.ToUpper(agg.Function) == "COUNT_DISTINCT" {
				result.Warnings = append(result.Warnings,
					"COUNT DISTINCT on '"+agg.Column+"' will cast the column to TEXT before counting (UUIDs and numeric types are supported)")
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// PreviewQueryHandler executes a query and returns sample data
func PreviewQueryHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "method not allowed"))
		return
	}

	// Limit request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req QueryPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid request"))
		return
	}

	result := ExecuteQueryPreview(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// PreviewAdvancedSQLHandler executes raw SQL with safety checks
func PreviewAdvancedSQLHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "method not allowed"))
		return
	}

	// Limit request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req AdvancedSQLPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid request"))
		return
	}

	result := ExecuteAdvancedSQLPreview(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// CategoryInfo represents a top-level category with descendant folder paths.
type CategoryInfo struct {
	Name          string   `json:"name"`
	Subcategories []string `json:"subcategories"`
}

// GetBuilderCategoriesHandler returns the category structure from configs folder
func GetBuilderCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Map to track top-level categories and every descendant folder path under
	// each category. The builder can publish into nested paths by sending the
	// selected descendant path as the subcategory.
	categoryMap := make(map[string]map[string]bool)

	// Locate the configs directory — in tests the CWD is the package dir, so fall back to parent.
	configsPath := "configs"
	if _, err := os.Stat(configsPath); os.IsNotExist(err) {
		configsPath = "../configs"
	}

	// Walk the configs directory
	err := filepath.Walk(configsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Only process directories
		if !info.IsDir() {
			return nil
		}

		// Get relative path from configs
		relPath, err := filepath.Rel(configsPath, path)
		if err != nil || relPath == "." {
			return nil
		}

		parts := strings.Split(relPath, string(filepath.Separator))

		topLevel := parts[0]
		if categoryMap[topLevel] == nil {
			categoryMap[topLevel] = make(map[string]bool)
		}
		if len(parts) > 1 {
			categoryMap[topLevel][strings.Join(parts[1:], "/")] = true
		}

		return nil
	})

	if err != nil {
		log.Printf("Error walking configs directory: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to read categories"))
		return
	}

	// Convert map to sorted slice
	categories := []CategoryInfo{}
	var categoryNames []string
	for name := range categoryMap {
		categoryNames = append(categoryNames, name)
	}
	sort.Strings(categoryNames)

	for _, name := range categoryNames {
		var subcategories []string
		for sub := range categoryMap[name] {
			subcategories = append(subcategories, sub)
		}
		sort.Strings(subcategories)
		categories = append(categories, CategoryInfo{
			Name:          name,
			Subcategories: subcategories,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"categories": categories,
	})
}

// GetReportSourceHandler returns the raw YAML content of a report for editing
func GetReportSourceHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract report ID from URL path: /api/report/source/{id}
	path := r.URL.Path
	prefix := basePath + "/api/report/source/"

	if !strings.HasPrefix(path, prefix) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid path"))
		return
	}

	reportID := strings.TrimPrefix(path, prefix)

	// Validate report ID
	if err := ValidateReportID(reportID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid report ID"))
		return
	}

	// Construct file path
	sanitizedPath := strings.ReplaceAll(reportID, "/", string(filepath.Separator))
	filePath := filepath.Join("configs", sanitizedPath+".yaml")

	// Security check: ensure path is within configs
	absConfigPath, _ := filepath.Abs("configs")
	absFilePath, _ := filepath.Abs(filePath)
	if !strings.HasPrefix(absFilePath, absConfigPath+string(filepath.Separator)) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid report path"))
		return
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "report not found"))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to read report"))
		}
		return
	}

	// Get the category path from the report ID
	categoryPath := ""
	parts := strings.Split(reportID, "/")
	if len(parts) > 1 {
		categoryPath = strings.Join(parts[:len(parts)-1], "/")
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"yaml":     string(data),
		"path":     categoryPath,
		"filename": parts[len(parts)-1],
	})
}

// PublishReportRequest represents the request to publish a report
type PublishReportRequest struct {
	YAML        string `json:"yaml"`
	Category    string `json:"category"`
	Subcategory string `json:"subcategory"`
	Filename    string `json:"filename"`
	Overwrite   bool   `json:"overwrite"`
}

// PublishReportHandler saves a YAML report to the configs directory
func PublishReportHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "method not allowed"))
		return
	}

	// Block publishing in production mode - changes must go through GitHub
	if authMode == "on" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "publishing disabled in production mode"))
		return
	}

	// Limit request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to read request"))
		return
	}

	var req PublishReportRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid request"))
		return
	}

	// Validate required fields
	if req.YAML == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "yaml content is required"))
		return
	}
	if req.Category == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "category is required"))
		return
	}
	if req.Filename == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "filename is required"))
		return
	}

	// Validate filename (no path separators, no extension)
	if strings.ContainsAny(req.Filename, "/\\") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "filename cannot contain path separators"))
		return
	}
	// Remove .yaml extension if provided
	req.Filename = strings.TrimSuffix(req.Filename, ".yaml")

	// Validate category and subcategory names
	validNamePattern := ValidateReportID
	if err := validNamePattern(req.Category); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid category name"))
		return
	}
	if req.Subcategory != "" {
		if err := validNamePattern(req.Subcategory); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid subcategory name"))
			return
		}
	}

	// Build the target path
	var targetDir string
	if req.Subcategory != "" {
		targetDir = filepath.Join("configs", req.Category, req.Subcategory)
	} else {
		targetDir = filepath.Join("configs", req.Category)
	}
	targetFile := filepath.Join(targetDir, req.Filename+".yaml")

	// Security check: ensure path is within configs
	absConfigPath, _ := filepath.Abs("configs")
	absTargetPath, _ := filepath.Abs(targetFile)
	if !strings.HasPrefix(absTargetPath, absConfigPath+string(filepath.Separator)) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid target path"))
		return
	}

	validateOnly := r.URL.Query().Get("validateOnly") == "true"

	// Check if file exists
	fileExists := false
	if _, err := os.Stat(targetFile); err == nil {
		fileExists = true
		if !validateOnly && !req.Overwrite {
			// File exists and overwrite not requested
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "exists",
				"message": "File already exists. Set overwrite to true to replace it.",
				"path":    targetFile,
			})
			return
		}
	}

	// Parse and validate YAML before saving
	var report Report
	if err := yaml.Unmarshal([]byte(req.YAML), &report); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"errors":  []PublishError{{Message: "Invalid YAML: " + err.Error()}},
		})
		return
	}

	result := validateReportForPublish(&report)

	// Validate-only mode: return validation results without writing file
	if validateOnly {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":      result.Valid,
			"errors":     result.Errors,
			"warnings":   result.Warnings,
			"fileExists": fileExists,
			"path":       targetFile,
		})
		return
	}

	if !result.Valid {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  false,
			"errors":   result.Errors,
			"warnings": result.Warnings,
		})
		return
	}

	// Ensure directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Printf("Error creating directory %s: %v", targetDir, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to create directory"))
		return
	}

	// Merge: preserve manually-edited SQL from the on-disk file for component
	// types that have no SQL editor in the builder (pie, bar, line, etc.).
	// Without this, a builder save with stale in-memory state silently
	// overwrites SQL that was edited directly in the YAML file.
	if fileExists {
		if existingData, readErr := os.ReadFile(targetFile); readErr == nil {
			var existingReport Report
			if yaml.Unmarshal(existingData, &existingReport) == nil {
				mergeProtectedSQL(&report, &existingReport)
				if mergedBytes, marshalErr := yaml.Marshal(&report); marshalErr == nil {
					req.YAML = string(mergedBytes)
				}
			}
		}
	}

	// Write the file
	if err := os.WriteFile(targetFile, []byte(req.YAML), 0644); err != nil {
		log.Printf("Error writing file %s: %v", targetFile, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to write file"))
		return
	}

	log.Printf("Published report to %s", targetFile)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"path":     targetFile,
		"warnings": result.Warnings,
	})
}

// mergeProtectedSQL preserves manually-edited SQL in the on-disk file for
// component types that have no SQL editor in the builder (pie, bar, line,
// choropleth-query, etc.). For table_advanced and choropleth, the builder
// provides a SQL editor, so the incoming SQL is treated as authoritative.
//
// Matching is by section ID (stable) + component index (positional within section).
// If the existing file has different, non-empty SQL for a protected component,
// the incoming component's SQL is replaced with the file's SQL.
func mergeProtectedSQL(incoming, existing *Report) {
	// Build lookup: sectionID -> componentIndex -> existing SQL
	existingSQLs := make(map[string]map[int]string)
	for _, sec := range existing.Sections {
		compMap := make(map[int]string)
		for i, comp := range sec.Components {
			if comp.SQL != "" {
				compMap[i] = comp.SQL
			}
		}
		if len(compMap) > 0 {
			existingSQLs[sec.ID] = compMap
		}
	}

	for si := range incoming.Sections {
		secID := incoming.Sections[si].ID
		compMap, ok := existingSQLs[secID]
		if !ok {
			continue
		}
		for ci := range incoming.Sections[si].Components {
			existingSQL, ok := compMap[ci]
			if !ok || existingSQL == "" {
				continue
			}
			comp := &incoming.Sections[si].Components[ci]
			// table_advanced and choropleth have a SQL editor in the builder;
			// trust the incoming SQL for those types.
			if comp.Type == "table_advanced" || comp.Type == "choropleth" {
				continue
			}
			// For all other types (pie, bar, line, etc.) preserve the file's SQL.
			if comp.SQL != existingSQL {
				comp.SQL = existingSQL
			}
		}
	}
}
