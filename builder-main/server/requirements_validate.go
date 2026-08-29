package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── types ────────────────────────────────────────────────────────────────────

// ValidationColumnCheck is one column-level pass/fail result.
type ValidationColumnCheck struct {
	Column          string `json:"column"`                   // as stored in the spec (may be a UID or a column name)
	ResolvedColumn  string `json:"resolvedColumn,omitempty"` // actual MV column name (when Column is a column UID)
	Role            string `json:"role,omitempty"`           // "direct", "numerator", "denominator"
	Found           bool   `json:"found"`
	Source          string `json:"source,omitempty"` // qualified table where column was found
	Message         string `json:"message"`
	DataElementID   string `json:"dataElementId,omitempty"`   // column UID for this data element
	DataElementName string `json:"dataElementName,omitempty"` // human-readable label from DDL alias mapping
}

// IndicatorValidation holds the full result for one indicator.
type IndicatorValidation struct {
	Name        string                  `json:"name"`
	Mode        string                  `json:"mode"` // "direct" or "calculated"
	VisualTitle string                  `json:"visualTitle,omitempty"`
	Section     string                  `json:"section,omitempty"`
	Formula     string                  `json:"formula,omitempty"`
	Passed      bool                    `json:"passed"`
	Checks      []ValidationColumnCheck `json:"checks"`
	Note        string                  `json:"note,omitempty"`
}

// ComponentCoverageCheck records whether one visual title declared in the
// requirements spec is present as a component in the linked report YAML,
// and whether its component type matches the spec's declared visualType.
type ComponentCoverageCheck struct {
	VisualTitle   string `json:"visualTitle"`
	SpecSection   string `json:"specSection,omitempty"` // section name as declared in the spec
	Found         bool   `json:"found"`
	YAMLSection   string `json:"yamlSection,omitempty"`   // section in YAML where the component was matched
	YAMLComponent string `json:"yamlComponent,omitempty"` // exact component title as written in YAML
	Message       string `json:"message"`
	// Type check (only populated when Found is true)
	SpecType    string `json:"specType,omitempty"`    // visualType from the spec
	YAMLType    string `json:"yamlType,omitempty"`    // type field of the matched YAML component
	TypeChecked bool   `json:"typeChecked"`           // false when spec type is empty / "Other"
	TypeMatch   bool   `json:"typeMatch"`             // true when types are compatible
	TypeMessage string `json:"typeMessage,omitempty"` // human-readable type comparison result
}

// FilterCheck records whether a filter required by the spec is set in the
// linked report YAML.
type FilterCheck struct {
	Filter  string `json:"filter"`
	Found   bool   `json:"found"`
	Message string `json:"message"`
}

// ValidationReport is the full response from GET /api/requirements-specs/{id}/validate.
type ValidationReport struct {
	ReportName        string                   `json:"reportName"`
	ReportID          string                   `json:"reportId"`
	DataSources       []string                 `json:"dataSources"`
	TotalChecks       int                      `json:"totalChecks"`
	Passed            int                      `json:"passed"`
	Failed            int                      `json:"failed"`
	Indicators        []IndicatorValidation    `json:"indicators"`
	ReportYAMLFound   bool                     `json:"reportYamlFound"`
	CoverageTotal     int                      `json:"coverageTotal"`
	CoveragePassed    int                      `json:"coveragePassed"`
	CoverageFailed    int                      `json:"coverageFailed"`
	TypeMismatchCount int                      `json:"typeMismatchCount"`
	ComponentCoverage []ComponentCoverageCheck `json:"componentCoverage,omitempty"`
	FilterTotal       int                      `json:"filterTotal"`
	FilterPassed      int                      `json:"filterPassed"`
	FilterFailed      int                      `json:"filterFailed"`
	FilterChecks      []FilterCheck            `json:"filterChecks,omitempty"`
}

// schemaTableRe matches qualified table names like "report.mv_conditions".
// Both schema and table segments must be lowercase snake-case identifiers.
var schemaTableRe = regexp.MustCompile(`\b([a-z_][a-z0-9_]*)\.([a-z_][a-z0-9_]*)\b`)

// ── handler ──────────────────────────────────────────────────────────────────

// handleValidateRequirementsSpec validates that all indicator columns referenced
// in the requirements spec payload exist in the declared materialized views.
// Route: GET /api/requirements-specs/{id}/validate
func handleValidateRequirementsSpec(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid requirements spec ID")
		return
	}

	ctx := r.Context()
	entry, err := GetRequirementsSpecByID(ctx, id)
	if err != nil {
		logErrorCtx(ctx, "Requirements validate: fetch id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch requirements spec")
		return
	}
	if entry == nil {
		writeJSONError(w, http.StatusNotFound, "requirements spec not found")
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(entry.PayloadJSON), &payload); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to parse payload JSON")
		return
	}

	// 1. Extract qualified table names from dataSources text.
	dataReq := asMap(payload["dataRequirements"])
	dsText := readString(dataReq, "dataSources")
	tables := extractSchemaTableNames(dsText)

	// 2. Query information_schema.columns for all declared tables.
	columnMap, err := loadColumnsForTables(ctx, tables)
	if err != nil {
		logErrorCtx(ctx, "Requirements validate: column query id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to query information schema")
		return
	}

	// 3. Build a flat lookup: column_name → first qualifying table.
	//    Ties are broken by first appearance in the tables slice so the order
	//    from dataSources text is preserved.
	colToTable := make(map[string]string, 128)
	for _, qt := range tables {
		for _, col := range columnMap[qt] {
			if _, exists := colToTable[col]; !exists {
				colToTable[col] = qt
			}
		}
	}

	// 3b. Also populate colToTable from DDL-parsed columns for declared tables.
	//     This allows validation to work even when the MV is not yet in the live DB.
	ddlCache := GetDDLSchemaCache()
	for _, qt := range tables {
		if info, ok := ddlCache[qt]; ok {
			for col := range info.Columns {
				if _, exists := colToTable[col]; !exists {
					colToTable[col] = qt + " (DDL)"
				}
			}
		}
	}

	// Build the column resolver used by all indicator extraction functions.
	resolver := &columnResolver{
		colToTable:     colToTable,
		ddlCache:       ddlCache,
		declaredTables: tables,
	}

	// 4. Extract and validate indicators — try the structured Python-generated
	//    format first (specifications[].indicators[]), then fall back to the
	//    older SQL-seed format (indicatorRequirements[]).
	indicators := extractNewFormatIndicators(payload, resolver)
	if len(indicators) == 0 {
		indicators = extractOldFormatIndicators(payload, resolver)
	}

	// 5. Tally results.
	totalChecks, passed, failed := 0, 0, 0
	for i := range indicators {
		ind := &indicators[i]
		if len(ind.Checks) == 0 {
			// Derived/info-only indicators count as one passing entry.
			totalChecks++
			passed++
			ind.Passed = true
			continue
		}
		allPassed := true
		for _, ch := range ind.Checks {
			totalChecks++
			if ch.Found {
				passed++
			} else {
				failed++
				allPassed = false
			}
		}
		ind.Passed = allPassed
	}

	result := ValidationReport{
		ReportName:  entry.ReportName,
		ReportID:    entry.ReportID,
		DataSources: tables,
		TotalChecks: totalChecks,
		Passed:      passed,
		Failed:      failed,
		Indicators:  indicators,
	}

	// 6. Component coverage + filter checks: requires a linked report YAML.
	if reportYAML := loadReportYAMLForCoverage(entry.ReportID); reportYAML != nil {
		result.ReportYAMLFound = true
		result.ComponentCoverage = checkComponentCoverage(payload, reportYAML)
		for _, cc := range result.ComponentCoverage {
			result.CoverageTotal++
			if cc.Found {
				result.CoveragePassed++
			} else {
				result.CoverageFailed++
			}
			if cc.TypeChecked && !cc.TypeMatch {
				result.TypeMismatchCount++
			}
		}

		// 7. Filter checks: verify spec-required filters are set in the YAML.
		result.FilterChecks = checkFilterCoverage(payload, reportYAML)
		for _, fc := range result.FilterChecks {
			result.FilterTotal++
			if fc.Found {
				result.FilterPassed++
			} else {
				result.FilterFailed++
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// ── extraction helpers ────────────────────────────────────────────────────────

// loadReportYAMLForCoverage loads the report YAML identified by reportID from
// the configs directory. Returns nil when reportID is empty, not valid, or the
// file cannot be read. Path traversal is prevented by requiring the resolved
// absolute path to remain within the configs directory.
//
// Resolution order:
//  1. Exact path: configs/{reportID}.yaml  (e.g. "Programs/report-weekly")
//  2. Filename fallback: recursively scan configs/ for a file named
//     {basename(reportID)}.yaml  — covers specs that store only the short
//     filename without the directory prefix (e.g. "report-weekly").
func loadReportYAMLForCoverage(reportID string) *Report {
	if reportID == "" {
		return nil
	}
	if err := ValidateReportID(reportID); err != nil {
		return nil
	}

	absBase, err := filepath.Abs("configs")
	if err != nil {
		return nil
	}

	// Helper: parse YAML bytes into a Report, returning nil on failure.
	parseReport := func(data []byte) *Report {
		var r Report
		if err := yaml.Unmarshal(data, &r); err != nil {
			return nil
		}
		return &r
	}

	// 1. Exact path lookup.
	sanitized := strings.ReplaceAll(reportID, "/", string(filepath.Separator))
	exactPath := filepath.Join("configs", sanitized+".yaml")
	absExact, err := filepath.Abs(exactPath)
	if err == nil && strings.HasPrefix(absExact, absBase+string(filepath.Separator)) {
		if data, err := os.ReadFile(exactPath); err == nil {
			return parseReport(data)
		}
	}

	// 2. Filename fallback: search configs/ recursively for {basename}.yaml.
	//    Only the last path segment is used so partial paths like
	//    "report-weekly" also resolve correctly.
	baseName := filepath.Base(sanitized) + ".yaml"
	var found string
	_ = filepath.WalkDir("configs", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || found != "" {
			return nil
		}
		if d.Name() == baseName {
			abs, absErr := filepath.Abs(path)
			if absErr == nil && strings.HasPrefix(abs, absBase+string(filepath.Separator)) {
				found = path
			}
		}
		return nil
	})
	if found != "" {
		if data, err := os.ReadFile(found); err == nil {
			return parseReport(data)
		}
	}
	return nil
}

// specToYAMLTypes maps a spec visualType value (lowercase) to the set of
// acceptable YAML component type strings. An empty slice means "any type is
// valid" (used for "other"). Types not present in the map are treated as
// unrecognised and skipped from type checking.
var specToYAMLTypes = map[string][]string{
	"infobox":                  {"infobox"},
	"bar":                      {"bar"},
	"bar chart":                {"bar"},
	"bar chart (stacked)":      {"bar"},
	"line":                     {"line"},
	"line chart":               {"line"},
	"bar-line":                 {"bar_line"},
	"bar-line chart":           {"bar_line"},
	"bar + line chart":         {"bar_line"},
	"table":                    {"table", "table_advanced"},
	"table (pivot)":            {"table", "table_advanced"},
	"table (advanced)":         {"table", "table_advanced"},
	"table (split columns)":    {"table", "table_advanced"},
	"pie":                      {"pie"},
	"pie chart":                {"pie"},
	"map":                      {"choropleth", "map"},
	"choropleth map":           {"choropleth", "map"},
	"choropleth map (faceted)": {"choropleth", "map"},
	"other":                    {}, // any YAML type is acceptable
}

// checkTypeCompat returns whether specType is compatible with yamlType and a
// human-readable explanation. typeChecked is false when the spec type is empty
// or "other", meaning no assertion was made.
func checkTypeCompat(specType, yamlType string) (typeChecked, typeMatch bool, msg string) {
	specLower := strings.ToLower(strings.TrimSpace(specType))
	yamlLower := strings.ToLower(strings.TrimSpace(yamlType))

	if specLower == "" || specLower == "other" {
		return false, true, ""
	}
	allowed, known := specToYAMLTypes[specLower]
	if !known {
		return false, true, fmt.Sprintf("spec type %q not recognised; type check skipped", specType)
	}
	if len(allowed) == 0 {
		// "other" bucket — everything passes
		return false, true, ""
	}
	for _, a := range allowed {
		if yamlLower == a {
			return true, true, fmt.Sprintf("%s → %s", specType, yamlType)
		}
	}
	return true, false, fmt.Sprintf("spec declares %q but YAML component type is %q", specType, yamlType)
}

// checkComponentCoverage verifies that every visual title declared in the spec
// (visualizationRequirements.specifications[].visualTitle) has a matching
// component title in the linked report YAML, and that the component type is
// compatible with the spec's declared visualType. Title matching is
// case-insensitive exact match.
func checkComponentCoverage(payload map[string]interface{}, report *Report) []ComponentCoverageCheck {
	vizReq := asMap(payload["visualizationRequirements"])
	specsRaw, _ := vizReq["specifications"].([]interface{})
	if len(specsRaw) == 0 {
		return nil
	}

	// Build flat map: lower(componentTitle) → (sectionTitle, exact title, type).
	type compLoc struct{ section, title, compType string }
	yamlComps := make(map[string]compLoc, 32)
	for _, sec := range report.Sections {
		for _, comp := range sec.Components {
			key := strings.ToLower(strings.TrimSpace(comp.Title))
			if key != "" {
				if _, exists := yamlComps[key]; !exists {
					yamlComps[key] = compLoc{section: sec.Title, title: comp.Title, compType: comp.Type}
				}
			}
		}
	}

	seen := make(map[string]bool)
	var checks []ComponentCoverageCheck

	for _, specRaw := range specsRaw {
		spec := asMap(specRaw)
		visualTitle := strings.TrimSpace(readString(spec, "visualTitle"))
		specSection := strings.TrimSpace(readString(spec, "section"))
		specType := strings.TrimSpace(readString(spec, "visualType"))
		if visualTitle == "" || seen[visualTitle] {
			continue
		}
		seen[visualTitle] = true

		if loc, ok := yamlComps[strings.ToLower(visualTitle)]; ok {
			typeChecked, typeMatch, typeMsg := checkTypeCompat(specType, loc.compType)
			checks = append(checks, ComponentCoverageCheck{
				VisualTitle:   visualTitle,
				SpecSection:   specSection,
				Found:         true,
				YAMLSection:   loc.section,
				YAMLComponent: loc.title,
				Message:       fmt.Sprintf("Found in YAML section %q", loc.section),
				SpecType:      specType,
				YAMLType:      loc.compType,
				TypeChecked:   typeChecked,
				TypeMatch:     typeMatch,
				TypeMessage:   typeMsg,
			})
		} else {
			checks = append(checks, ComponentCoverageCheck{
				VisualTitle: visualTitle,
				SpecSection: specSection,
				Found:       false,
				Message:     fmt.Sprintf("%q not found as a component title in the report YAML", visualTitle),
				SpecType:    specType,
			})
		}
	}
	return checks
}

// knownFilterAliases maps legacy spec filter values (lower-cased) to canonical
// YAML filter names. Values that already equal the YAML name are also included
// so the lookup always works for the current form values.
var knownFilterAliases = map[string]string{
	"year":             "year",
	"month":            "month",
	"quarter":          "quarter",
	"week":             "week",
	"district":         "district",
	"region":           "region",
	"facility":         "facility",
	"reporting period": "year", // legacy label — maps to the time filter
}

// checkFilterCoverage verifies that every filter declared in the spec's
// reportFilters.selectedFilters is present in the linked report YAML.
func checkFilterCoverage(payload map[string]interface{}, report *Report) []FilterCheck {
	rf := asMap(payload["reportFilters"])
	raw, _ := rf["selectedFilters"].([]interface{})
	if len(raw) == 0 {
		return nil
	}

	// Build a set of YAML filter names (lower-cased for case-insensitive comparison).
	yamlFilters := make(map[string]bool, len(report.Filters))
	for _, f := range report.Filters {
		yamlFilters[strings.ToLower(f)] = true
	}
	// Also include custom filter column names.
	for _, cf := range report.CustomFilters {
		yamlFilters[strings.ToLower(cf.Column)] = true
	}

	var checks []FilterCheck
	seen := make(map[string]bool)

	for _, v := range raw {
		specVal := strings.TrimSpace(fmt.Sprintf("%v", v))
		if specVal == "" {
			continue
		}
		canonical, ok := knownFilterAliases[strings.ToLower(specVal)]
		if !ok {
			canonical = strings.ToLower(specVal) // fall back to the value itself
		}
		if canonical == "" || seen[canonical] {
			continue
		}
		seen[canonical] = true

		if yamlFilters[canonical] {
			checks = append(checks, FilterCheck{
				Filter:  canonical,
				Found:   true,
				Message: fmt.Sprintf("filter %q is set in the YAML", canonical),
			})
		} else {
			checks = append(checks, FilterCheck{
				Filter:  canonical,
				Found:   false,
				Message: fmt.Sprintf("filter %q is required by the spec but not set in the YAML", canonical),
			})
		}
	}
	return checks
}

// extractSchemaTableNames finds unique "schema.table" references in text.
func extractSchemaTableNames(text string) []string {
	lower := strings.ToLower(text)
	matches := schemaTableRe.FindAllStringSubmatch(lower, -1)
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		qt := m[1] + "." + m[2]
		if !seen[qt] {
			seen[qt] = true
			result = append(result, qt)
		}
	}
	return result
}

// loadColumnsForTables queries information_schema.columns for the given
// qualified table names and returns a map of "schema.table" → []column_name.
func loadColumnsForTables(ctx context.Context, tables []string) (map[string][]string, error) {
	result := make(map[string][]string, len(tables))
	if DB == nil || len(tables) == 0 {
		return result, nil
	}

	args := make([]interface{}, 0, len(tables)*2)
	placeholders := make([]string, 0, len(tables))
	for _, qt := range tables {
		parts := strings.SplitN(qt, ".", 2)
		if len(parts) != 2 {
			continue
		}
		idx := len(args)
		placeholders = append(placeholders, fmt.Sprintf("($%d,$%d)", idx+1, idx+2))
		args = append(args, parts[0], parts[1])
	}
	if len(placeholders) == 0 {
		return result, nil
	}

	query := fmt.Sprintf(`
		SELECT table_schema, table_name, column_name
		  FROM information_schema.columns
		 WHERE (table_schema, table_name) IN (%s)
		 ORDER BY table_schema, table_name, ordinal_position`,
		strings.Join(placeholders, ","),
	)

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table, col string
		if err := rows.Scan(&schema, &table, &col); err != nil {
			return nil, err
		}
		qt := schema + "." + table
		result[qt] = append(result[qt], col)
	}
	return result, rows.Err()
}

// checkColumn builds a ValidationColumnCheck for one column name.
// It tries three resolution strategies in order:
//  1. Direct column name match against information_schema (or DDL-seeded colToTable).
//  2. Column UID lookup in the DDL schema cache for declared data-source tables.
//  3. Direct column name lookup in the DDL cache (covers DDL-only columns).
func checkColumn(col, role string, colToTable map[string]string) ValidationColumnCheck {
	col = strings.TrimSpace(col)
	if source, ok := colToTable[col]; ok {
		return ValidationColumnCheck{
			Column:  col,
			Role:    role,
			Found:   true,
			Source:  source,
			Message: "Found in " + source,
		}
	}
	return ValidationColumnCheck{
		Column:  col,
		Role:    role,
		Found:   false,
		Message: "Column '" + col + "' not found in any declared data source",
	}
}

// columnResolver wraps the column lookup sources and provides the canonical
// check function used by all indicator extraction helpers.
type columnResolver struct {
	colToTable     map[string]string     // column_name → qualified table (info_schema + DDL)
	ddlCache       map[string]*MVDDLInfo // full DDL schema cache
	declaredTables []string              // only check these tables for UID resolution
}

// lookupDEInfo fills DataElementID and DataElementName on a check result by
// searching the DDL cache for the given MV column name or UID.
func (cr *columnResolver) lookupDEInfo(result *ValidationColumnCheck, mvCol, uid string) {
	for _, qt := range cr.declaredTables {
		info, ok := cr.ddlCache[qt]
		if !ok {
			continue
		}
		// If we already have the UID, just look up the description.
		if uid != "" {
			result.DataElementID = uid
			if desc, ok := info.UIDToDescription[uid]; ok && desc != "" {
				result.DataElementName = desc
				return
			}
			continue
		}
		// Derive UID from MV column name.
		if u, ok := info.ColumnToUID[mvCol]; ok {
			result.DataElementID = u
			if desc, ok := info.UIDToDescription[u]; ok {
				result.DataElementName = desc
			}
			return
		}
	}
}

// check resolves a single dataElement value (column name or column UID) and
// returns a ValidationColumnCheck with Found=true when the value can be
// mapped to an actual MV column in one of the declared data sources.
func (cr *columnResolver) check(col, role string) ValidationColumnCheck {
	col = strings.TrimSpace(col)
	if col == "" {
		return ValidationColumnCheck{Role: role, Found: false, Message: "empty column reference"}
	}

	// Strategy 1: direct column-name match (covers info_schema and DDL columns).
	if source, ok := cr.colToTable[col]; ok {
		result := ValidationColumnCheck{
			Column:  col,
			Role:    role,
			Found:   true,
			Source:  source,
			Message: "Column found in " + source,
		}
		cr.lookupDEInfo(&result, col, "")
		return result
	}

	// Strategy 2: column UID → column name resolution via DDL alias mapping.
	// Only check declared data-source tables so unrelated views don't cause
	// false positives.
	for _, qt := range cr.declaredTables {
		info, ok := cr.ddlCache[qt]
		if !ok {
			continue
		}
		if cols, ok := info.UIDToColumns[col]; ok && len(cols) > 0 {
			resolved := cols[0]
			result := ValidationColumnCheck{
				Column:         col,
				ResolvedColumn: resolved,
				Role:           role,
				Found:          true,
				Source:         qt,
				Message:        "UID → " + resolved + " in " + qt,
			}
			// UID is already known (col itself is the UID).
			cr.lookupDEInfo(&result, resolved, col)
			return result
		}
	}

	return ValidationColumnCheck{
		Column:  col,
		Role:    role,
		Found:   false,
		Message: "'" + col + "' not found as column or UID in declared data sources",
	}
}

// ── new format (Python-generated specs) ──────────────────────────────────────

// extractNewFormatIndicators processes the structured format produced by the
// Python insertion scripts:
//
//	visualizationRequirements.specifications[].indicators[].{
//	    indicatorLabel, mode, dataElements, numerator, denominator
//	}
//
// where each dataElement has a "dataElement" field holding the column name.
func extractNewFormatIndicators(payload map[string]interface{}, resolver *columnResolver) []IndicatorValidation {
	vizReq := asMap(payload["visualizationRequirements"])
	specsRaw, _ := vizReq["specifications"].([]interface{})

	var result []IndicatorValidation
	seen := make(map[string]bool)

	for _, specRaw := range specsRaw {
		spec := asMap(specRaw)
		visualTitle := readString(spec, "visualTitle")
		section := readString(spec, "section")
		indsRaw, _ := spec["indicators"].([]interface{})

		for _, indRaw := range indsRaw {
			ind := asMap(indRaw)
			label := readString(ind, "indicatorLabel")
			mode := readString(ind, "mode")
			if label == "" || mode == "" {
				continue
			}
			if seen[label] {
				continue // report each unique indicator once
			}
			seen[label] = true

			var checks []ValidationColumnCheck

			switch mode {
			case "direct":
				for _, col := range deColumns(ind, "dataElements") {
					checks = append(checks, resolver.check(col, "direct"))
				}
			case "calculated":
				for _, col := range deColumns(ind, "numerator") {
					checks = append(checks, resolver.check(col, "numerator"))
				}
				for _, col := range deColumns(ind, "denominator") {
					checks = append(checks, resolver.check(col, "denominator"))
				}
			}

			// Deduplicate by column name within this indicator.
			checks = deduplicateChecks(checks)

			if len(checks) == 0 {
				continue
			}

			result = append(result, IndicatorValidation{
				Name:        label,
				Mode:        mode,
				VisualTitle: visualTitle,
				Section:     section,
				Checks:      checks,
			})
		}
	}
	return result
}

// deColumns extracts "dataElement" column values from an indicator's array key.
func deColumns(ind map[string]interface{}, key string) []string {
	raw, _ := ind[key].([]interface{})
	cols := make([]string, 0, len(raw))
	for _, r := range raw {
		de := asMap(r)
		if col := readString(de, "dataElement"); col != "" {
			cols = append(cols, col)
		}
	}
	return cols
}

// deduplicateChecks removes duplicate column entries keeping first occurrence.
func deduplicateChecks(checks []ValidationColumnCheck) []ValidationColumnCheck {
	seen := make(map[string]bool, len(checks))
	out := checks[:0]
	for _, ch := range checks {
		if !seen[ch.Column] {
			seen[ch.Column] = true
			out = append(out, ch)
		}
	}
	return out
}

// ── old format (SQL-seed specs) ───────────────────────────────────────────────

// extractOldFormatIndicators processes the older format where indicators are
// stored at the top-level:
//
//	indicatorRequirements[].{name, source, formula}
//
// The "source" field may be one of:
//   - "schema.table.column_name"     → direct column check
//   - "Derived"                      → calculated, no column check
//   - "Derived using schema.table"   → calculated with declared source
//   - "schema.table WHERE ..."       → filter on table, no specific column
func extractOldFormatIndicators(payload map[string]interface{}, resolver *columnResolver) []IndicatorValidation {
	indsRaw, _ := payload["indicatorRequirements"].([]interface{})
	var result []IndicatorValidation

	for _, indRaw := range indsRaw {
		ind := asMap(indRaw)
		name := readString(ind, "name")
		source := readString(ind, "source")
		formula := readString(ind, "formula")
		if name == "" {
			continue
		}

		mode := "direct"
		var note string
		if formula != "" || strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "derived") {
			mode = "calculated"
		}

		parsed := parseOldSource(source)
		var checks []ValidationColumnCheck

		switch {
		case parsed.column != "":
			checks = append(checks, resolver.check(parsed.column, mode))
		case parsed.isDerived:
			if formula != "" {
				note = "Calculated: " + formula
			} else {
				note = "Derived indicator — no direct column check required"
			}
		case parsed.schema != "" && parsed.table != "":
			note = fmt.Sprintf("References %s.%s; no specific column declared", parsed.schema, parsed.table)
		default:
			note = "Source: " + source
		}

		result = append(result, IndicatorValidation{
			Name:    name,
			Mode:    mode,
			Formula: formula,
			Checks:  checks,
			Note:    note,
		})
	}
	return result
}

type parsedOldSource struct {
	schema    string
	table     string
	column    string
	isDerived bool
}

// parseOldSource deconstructs old-format source strings.
func parseOldSource(source string) parsedOldSource {
	s := strings.TrimSpace(source)
	lower := strings.ToLower(s)

	if strings.HasPrefix(lower, "derived") {
		return parsedOldSource{isDerived: true}
	}

	// Drop trailing " WHERE ..." clause before splitting.
	if idx := strings.Index(strings.ToUpper(s), " WHERE "); idx >= 0 {
		s = s[:idx]
	}

	parts := strings.Split(strings.TrimSpace(s), ".")
	switch len(parts) {
	case 3:
		return parsedOldSource{schema: parts[0], table: parts[1], column: parts[2]}
	case 2:
		return parsedOldSource{schema: parts[0], table: parts[1]}
	}
	return parsedOldSource{}
}
