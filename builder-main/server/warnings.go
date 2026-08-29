package main

import (
	"fmt"
	"regexp"
	"strings"
)

// ReportWarning represents a design quality warning (non-blocking)
type ReportWarning struct {
	Message   string `json:"message"`
	Tip       string `json:"tip,omitempty"`        // Actionable explanation shown on click
	Component string `json:"component,omitempty"`  // e.g. "Section 1, Component 2"
	Section   int    `json:"section,omitempty"`
	Index     int    `json:"index,omitempty"`
	Severity  string `json:"severity,omitempty"`   // "low" = red hint, default = orange
}

// ValidateReportWarnings checks a report for common design issues.
// Returns soft warnings — technically valid but likely problematic configs.
func ValidateReportWarnings(report *Report) []ReportWarning {
	var warnings []ReportWarning

	warnings = append(warnings, checkGroupByWithoutYear(report)...)
	warnings = append(warnings, checkNoFilters(report)...)
	warnings = append(warnings, checkTooManyAggregations(report)...)
	warnings = append(warnings, checkRawSQLIgnoresFilters(report)...)
	warnings = append(warnings, checkUndefinedFormulaAlias(report)...)
	warnings = append(warnings, checkMissingSectionDescription(report)...)
	warnings = append(warnings, checkMissingComponentTitle(report)...)
	warnings = append(warnings, checkSingleComponentTwoColumn(report)...)
	warnings = append(warnings, checkTooManyComponents(report)...)
	warnings = append(warnings, checkLowReferenceUsage(report)...)
	warnings = append(warnings, checkReferenceLineOutsideYLimit(report)...)
	warnings = append(warnings, checkCrossTableFilterCompatibility(report)...)
	warnings = append(warnings, checkChoroplethColumnMapping(report)...)
	warnings = append(warnings, checkDeprecatedRawSQLChoropleth(report)...)
	warnings = append(warnings, checkTooManyChoropleths(report)...)

	return warnings
}

// checkGroupByWithoutYear warns when grouping by month/week/quarter without year.
// Grouping by month alone merges January 2023 and January 2024 into one bar.
func checkGroupByWithoutYear(report *Report) []ReportWarning {
	var warnings []ReportWarning

	for si, section := range report.Sections {
		for ci, comp := range section.Components {
			if comp.Query == nil || len(comp.Query.GroupBy) == 0 {
				continue
			}

			hasNoYear := false
			for _, gb := range comp.Query.GroupBy {
				f := strings.ToLower(gb.Format)
				if f == "month_noyear" || f == "week_noyear" || f == "quarter_noyear" {
					hasNoYear = true
				}
			}

			if hasNoYear {
				warnings = append(warnings, ReportWarning{
					Message:   "Grouping by period without year will merge data across years",
					Tip:       "The _noyear format intentionally combines data across years (e.g. all Januaries into one bucket). If this isn't what you want, switch to the standard month/week/quarter format which includes year automatically.",
					Component: fmt.Sprintf("Section %d, Component %d", si+1, ci+1),
					Section:   si,
					Index:     ci,
				})
			}
		}
	}

	return warnings
}

// checkNoFilters warns when no standard filters are enabled.
// Users can't narrow down data by time or location.
func checkNoFilters(report *Report) []ReportWarning {
	standardFilters := map[string]bool{
		"year": true, "month": true, "week": true, "quarter": true,
		"district": true, "region": true, "facility": true,
	}

	for _, f := range report.Filters {
		if standardFilters[f] {
			return nil
		}
	}

	return []ReportWarning{{
		Message: "No filters enabled — users cannot narrow data by time or location",
		Tip:     "Without filters, users see all data at once and cannot drill down by year, district, or facility. Add a filters list to the report YAML (e.g. filters: [year, district]).",
	}}
}

// checkTooManyAggregations warns when a chart has >8 aggregations.
// Charts with many datasets are unreadable.
func checkTooManyAggregations(report *Report) []ReportWarning {
	var warnings []ReportWarning
	chartTypes := map[string]bool{
		"bar": true, "line": true, "pie": true, "bar_line": true,
	}

	for si, section := range report.Sections {
		for ci, comp := range section.Components {
			if !chartTypes[comp.Type] || comp.Query == nil {
				continue
			}
			if len(comp.Query.Aggregations) > 8 {
				warnings = append(warnings, ReportWarning{
					Message:   fmt.Sprintf("Chart has %d aggregations — more than 8 makes charts hard to read", len(comp.Query.Aggregations)),
					Tip:       "Each aggregation becomes a separate dataset (bar group, line, or pie slice). Beyond 8, colors become hard to distinguish and legends overflow. Split into multiple charts or use a table instead.",
					Component: fmt.Sprintf("Section %d, Component %d", si+1, ci+1),
					Section:   si,
					Index:     ci,
				})
			}
		}
	}

	return warnings
}

// checkRawSQLIgnoresFilters warns when table_advanced SQL doesn't use filter placeholders
// but the report has filters enabled.
func checkRawSQLIgnoresFilters(report *Report) []ReportWarning {
	if len(report.Filters) == 0 {
		return nil
	}

	filterPlaceholderRe := regexp.MustCompile(`\{\{[a-z_]+_filter\}\}`)

	var warnings []ReportWarning
	for si, section := range report.Sections {
		for ci, comp := range section.Components {
			if comp.Type != "table_advanced" || comp.SQL == "" {
				continue
			}
			if !filterPlaceholderRe.MatchString(comp.SQL) {
				warnings = append(warnings, ReportWarning{
					Message:   "Raw SQL has no {{*_filter}} placeholders — component ignores report filters",
					Tip:       "When a table_advanced component uses raw SQL, filters are applied via placeholders like {{district_filter}} in your WHERE clause. Without them, changing filters in the UI has no effect on this component's data.",
					Component: fmt.Sprintf("Section %d, Component %d", si+1, ci+1),
					Section:   si,
					Index:     ci,
				})
			}
		}
	}

	return warnings
}

// formulaIdentifierRe matches word-boundary identifiers (letters/underscores, not numbers)
var formulaIdentifierRe = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)

// sqlKeywords are SQL keywords that can appear in CASE WHEN formulas and should
// not be treated as aggregation alias references.
var sqlKeywords = map[string]bool{
	"CASE": true, "WHEN": true, "THEN": true, "ELSE": true, "END": true,
	"IS": true, "NULL": true, "OR": true, "AND": true, "NOT": true,
	"COALESCE": true, "NULLIF": true, "CAST": true, "AS": true,
	"INTEGER": true, "NUMERIC": true, "FLOAT": true, "TEXT": true,
}

// checkUndefinedFormulaAlias warns when a calculate formula references an alias
// not defined in the component's aggregations. Collapses multiple into one warning.
func checkUndefinedFormulaAlias(report *Report) []ReportWarning {
	var locations []string

	for si, section := range report.Sections {
		for ci, comp := range section.Components {
			if comp.Query == nil || len(comp.Query.Calculate) == 0 {
				continue
			}

			// Build set of defined aliases
			aliases := make(map[string]bool, len(comp.Query.Aggregations))
			for _, agg := range comp.Query.Aggregations {
				if agg.Alias != "" {
					aliases[agg.Alias] = true
				}
			}

			found := false
			for _, calc := range comp.Query.Calculate {
				if calc.Formula == "" || found {
					continue
				}
				identifiers := formulaIdentifierRe.FindAllString(calc.Formula, -1)
				for _, id := range identifiers {
					if sqlKeywords[strings.ToUpper(id)] {
						continue
					}
					if !aliases[id] {
						locations = append(locations, fmt.Sprintf("S%d/C%d: \"%s\"", si+1, ci+1, id))
						found = true
						break
					}
				}
			}
		}
	}

	if len(locations) == 0 {
		return nil
	}

	msg := fmt.Sprintf("Formula references undefined alias in %d component(s) — %s", len(locations), strings.Join(locations, ", "))

	return []ReportWarning{{
		Message: msg,
		Tip:     "The calculate formula uses a name that doesn't match any alias in the aggregations list. Check for typos, or add an aggregation with this alias. Aliases are case-sensitive.",
	}}
}

// checkLowReferenceUsage warns when fewer than 20% of bar/line/text components
// have reference lines (charts) or targets (KPIs). These visual benchmarks help
// users interpret whether values are good or bad.
func checkLowReferenceUsage(report *Report) []ReportWarning {
	chartTypes := map[string]bool{"bar": true, "line": true, "bar_line": true}

	var total, withRef int
	for _, section := range report.Sections {
		for _, comp := range section.Components {
			if chartTypes[comp.Type] {
				total++
				if len(comp.ReferenceLines) > 0 {
					withRef++
				}
			} else if comp.Type == "text" {
				total++
				if comp.Target != nil {
					withRef++
				}
			}
		}
	}

	if total < 3 {
		return nil // too few components to judge
	}

	threshold := total / 5 // 20%
	if threshold < 1 {
		threshold = 1
	}

	if withRef < threshold {
		return []ReportWarning{{
			Message:  fmt.Sprintf("Only %d/%d charts and KPIs have reference lines or targets — consider adding benchmarks so users can judge values", withRef, total),
			Tip:      "Reference lines (e.g. a target of 15,000) draw a horizontal line on bar/line charts. Targets on KPI text cards show whether a value meets its goal. These help users quickly judge if numbers are good or bad.",
			Severity: "low",
		}}
	}

	return nil
}

// checkReferenceLineOutsideYLimit warns when a reference line falls outside the
// configured yLimit range. Such lines render off-canvas and are invisible.
// For bar_line, lines are assumed to live on the left (y) axis.
func checkReferenceLineOutsideYLimit(report *Report) []ReportWarning {
	var warnings []ReportWarning

	for si, section := range report.Sections {
		for ci, comp := range section.Components {
			if comp.YLimit == nil || len(comp.ReferenceLines) == 0 {
				continue
			}
			min, max := comp.YLimit.Min, comp.YLimit.Max
			if min == nil && max == nil {
				continue
			}
			for _, rl := range comp.ReferenceLines {
				outside := (max != nil && rl.Value > *max) || (min != nil && rl.Value < *min)
				if !outside {
					continue
				}
				label := rl.Label
				if label == "" {
					label = fmt.Sprintf("%g", rl.Value)
				}
				warnings = append(warnings, ReportWarning{
					Message:   fmt.Sprintf("Reference line %q (%g) falls outside yLimit — it will be hidden", label, rl.Value),
					Tip:       "The reference line value is above yLimit.max or below yLimit.min, so it renders off-canvas. Either widen yLimit or move the reference line into range.",
					Component: fmt.Sprintf("Section %d, Component %d", si+1, ci+1),
					Section:   si,
					Index:     ci,
				})
			}
		}
	}

	return warnings
}

// checkMissingSectionDescription warns when sections have no description.
// Collapses multiple missing descriptions into a single warning.
func checkMissingSectionDescription(report *Report) []ReportWarning {
	var missing []string

	for si, section := range report.Sections {
		if strings.TrimSpace(section.Description) == "" {
			missing = append(missing, fmt.Sprintf("%d", si+1))
		}
	}

	if len(missing) == 0 {
		return nil
	}

	msg := "Section has no description — add context so users understand the data"
	if len(missing) > 1 {
		msg += fmt.Sprintf(" (Section %s)", strings.Join(missing, ", "))
	} else {
		msg += fmt.Sprintf(" (Section %s)", missing[0])
	}

	return []ReportWarning{{
		Message: msg,
		Tip:     "Section descriptions appear as gray text below section titles. They give users context about what the data shows, where it comes from, or how to interpret it. Add a description field to each section in the YAML.",
	}}
}

// checkMissingComponentTitle warns when components have no title.
// Collapses multiple into a single warning listing affected locations.
func checkMissingComponentTitle(report *Report) []ReportWarning {
	var missing []string

	for si, section := range report.Sections {
		for ci, comp := range section.Components {
			if comp.Type == "text" {
				continue // KPI cards use content as the visual label
			}
			if strings.TrimSpace(comp.Title) == "" {
				missing = append(missing, fmt.Sprintf("S%d/C%d", si+1, ci+1))
			}
		}
	}

	if len(missing) == 0 {
		return nil
	}

	msg := fmt.Sprintf("%d component(s) have no title — users won't know what the data shows (%s)", len(missing), strings.Join(missing, ", "))

	return []ReportWarning{{
		Message: msg,
		Tip:     "Every chart, table, and KPI card should have a clear title so users understand the data at a glance. Add a title field to each component in the YAML.",
	}}
}

// checkSingleComponentTwoColumn warns when a two-column section has only one component.
// This wastes half the screen — switch to single layout or add a second component.
func checkSingleComponentTwoColumn(report *Report) []ReportWarning {
	var sections []string

	for si, section := range report.Sections {
		if section.Layout == "two-column" && len(section.Components) == 1 {
			sections = append(sections, fmt.Sprintf("%d", si+1))
		}
	}

	if len(sections) == 0 {
		return nil
	}

	msg := "Two-column layout with only 1 component wastes half the row"
	if len(sections) > 1 {
		msg += fmt.Sprintf(" (Section %s)", strings.Join(sections, ", "))
	} else {
		msg += fmt.Sprintf(" (Section %s)", sections[0])
	}

	return []ReportWarning{{
		Message: msg,
		Tip:     "Two-column layout splits the row into two equal halves. With only one component, the second half is empty. Either add a second component or switch the section layout to single.",
	}}
}

// checkTooManyComponents warns when a section has more than 4 components.
// Overloaded sections are hard to scan — split into focused sub-sections.
func checkTooManyComponents(report *Report) []ReportWarning {
	var sections []string

	for si, section := range report.Sections {
		if len(section.Components) > 6 {
			sections = append(sections, fmt.Sprintf("%d (%d)", si+1, len(section.Components)))
		}
	}

	if len(sections) == 0 {
		return nil
	}

	msg := fmt.Sprintf("Section has too many components — more than 4 makes sections hard to scan (Section %s)", strings.Join(sections, ", "))

	return []ReportWarning{{
		Message:  msg,
		Tip:      "Large sections with many components overwhelm users. Consider splitting into focused sub-sections of 3-4 components each, grouped by theme or metric type.",
		Severity: "low",
	}}
}

// checkCrossTableFilterCompatibility warns when components query different tables
// but the report uses shared filter column mappings (timeColumns/locationColumns).
// A single timeColumns.year = "period_year" may not exist in every table.
func checkCrossTableFilterCompatibility(report *Report) []ReportWarning {
	if len(report.Filters) == 0 {
		return nil
	}

	tables := make(map[string]bool)
	for _, section := range report.Sections {
		for _, comp := range section.Components {
			if comp.Query != nil && comp.Query.Table != "" {
				tables[comp.Query.Table] = true
			}
		}
	}

	if len(tables) <= 1 {
		return nil
	}

	// Check if any standard time/location filters are enabled
	hasStandardFilter := false
	for _, f := range report.Filters {
		switch f {
		case "year", "month", "week", "quarter", "district", "region", "facility":
			hasStandardFilter = true
		}
	}
	if !hasStandardFilter {
		return nil
	}

	tableList := make([]string, 0, len(tables))
	for t := range tables {
		tableList = append(tableList, t)
	}

	return []ReportWarning{{
		Message: fmt.Sprintf("Components query %d different tables (%s) but share filter column mappings — verify timeColumns/locationColumns exist in all tables", len(tableList), strings.Join(tableList, ", ")),
		Tip:     "The report defines one set of timeColumns/locationColumns (e.g. year: period_year), but components pull from different tables. If a table uses 'report_year' instead of 'period_year', the filter will fail silently. Verify that the mapped column names exist in every table used.",
	}}
}

// checkChoroplethColumnMapping warns on SQL choropleth misconfigurations:
// - SQL set but no columnMapping (silent fallback to columns 0/1)
// - SQL set but no filterTable when report has filters
func checkChoroplethColumnMapping(report *Report) []ReportWarning {
	var warnings []ReportWarning

	hasFilters := len(report.Filters) > 0

	for si, section := range report.Sections {
		for ci, comp := range section.Components {
			if comp.Type != "choropleth" || comp.SQL == "" {
				continue
			}

			loc := fmt.Sprintf("Section %d, Component %d", si+1, ci+1)

			if comp.ColumnMapping == nil {
				warnings = append(warnings, ReportWarning{
					Message:   "SQL choropleth has no columnMapping — system falls back to columns 0/1, define explicit district/value mapping",
					Tip:       "Without columnMapping, the system blindly uses the first column as district and the second as value. Add columnMapping: {district: your_district_col, value: your_value_col} to be explicit and avoid surprises if column order changes.",
					Component: loc,
					Section:   si,
					Index:     ci,
				})
			}

			if hasFilters && comp.FilterTable == "" {
				warnings = append(warnings, ReportWarning{
					Message:   "SQL choropleth has no filterTable — filters cannot bind without a table context",
					Tip:       "Raw SQL choropleths need filterTable set to the main table name so the system knows which table to validate filter columns against. Without it, filter placeholders like {{district_filter}} cannot be resolved.",
					Component: loc,
					Section:   si,
					Index:     ci,
				})
			}
		}
	}

	return warnings
}

// checkDeprecatedRawSQLChoropleth warns when a choropleth uses raw SQL instead of a structured query.
// Raw SQL choropleths are deprecated and will be removed in a future version.
func checkDeprecatedRawSQLChoropleth(report *Report) []ReportWarning {
	var locations []string

	for si, section := range report.Sections {
		for ci, comp := range section.Components {
			if comp.Type == "choropleth" && comp.SQL != "" {
				locations = append(locations, fmt.Sprintf("Section %d, Component %d", si+1, ci+1))
			}
		}
	}

	if len(locations) == 0 {
		return nil
	}

	msg := fmt.Sprintf("Raw SQL choropleth is deprecated and will be removed — migrate to structured query (%s)", strings.Join(locations, "; "))

	return []ReportWarning{{
		Message: msg,
		Tip:     "Choropleth components using raw SQL (the sql: field) are deprecated. Convert to a structured query using query: with table, aggregations, and groupBy fields instead. This gives you automatic filter binding, safer SQL generation, and better validation. See other choropleth components in the codebase for examples.",
	}}
}

// checkTooManyChoropleths warns when a report has more than 6 choropleth/map components.
// Each map renders 3 GeoJSON layers (~161 features each), so many maps strain the browser.
func checkTooManyChoropleths(report *Report) []ReportWarning {
	count := 0
	for _, section := range report.Sections {
		for _, comp := range section.Components {
			if comp.Type == "choropleth" || comp.Type == "map" {
				count++
			}
		}
	}

	if count <= 6 {
		return nil
	}

	return []ReportWarning{{
		Message:  fmt.Sprintf("Report has %d map components — reports with more than 6 maps may load slowly on some devices. Consider other visuals.", count),
		Tip:      "Each choropleth map renders multiple GeoJSON layers with hundreds of features. Too many maps on one page increases memory usage and slows rendering, especially on tablets and low-end devices. Consider using tables, bar charts, or splitting into sub-reports.",
		Severity: "low",
	}}
}
