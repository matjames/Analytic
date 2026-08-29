package main

import (
	"encoding/json"
	"strings"
)

// ReportListItem represents a report in the list
type ReportListItem struct {
	ID          string `json:"id" yaml:"id"`
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Keywords    string `json:"keywords,omitempty" yaml:"keywords,omitempty"` // Comma-separated keywords for search
	Category    string `json:"category" yaml:"category"`
	SearchText  string `json:"searchText,omitempty" yaml:"-"` // Concatenated section/component titles for search
}

// FilterDefinition describes how a filter should be rendered and behave
type FilterDefinition struct {
	Type          string   `json:"type"`                    // "select", "multiselect", "checkbox", etc.
	Label         string   `json:"label"`                   // Display label
	APIEndpoint   string   `json:"apiEndpoint"`             // API URL to fetch options
	DependsOn     []string `json:"dependsOn"`               // Which filters this depends on (e.g., ["year"])
	CascadesTo    []string `json:"cascadesTo"`              // Which filters depend on this (e.g., ["month"])
	ParamName     string   `json:"paramName"`               // Query param name for this filter
	ParentParams  []string `json:"parentParams"`            // Which params to pass to API (from parent filters)
	ExclusiveWith []string `json:"exclusiveWith,omitempty"` // Filters that cannot be used together with this one
	DefaultValue  string   `json:"defaultValue,omitempty"`  // Optional default for custom single-select filters
	NoDefault     bool     `json:"noDefault,omitempty"`     // Suppress automatic default selection
	HideNone      bool     `json:"hideNone,omitempty"`      // Remove the blank "None" option from the dropdown
}

// FilterDefinitions maps filter names to their definitions
type FilterDefinitions map[string]FilterDefinition

// TimeColumns specifies which columns contain time/date data for filtering
// Used to extract year, month, week, quarter from columns with any naming convention
type TimeColumns struct {
	Year        string `json:"year,omitempty" yaml:"year,omitempty"`               // Column name to extract year from
	Month       string `json:"month,omitempty" yaml:"month,omitempty"`             // Column name to extract month from (optional)
	MonthFormat string `json:"monthFormat,omitempty" yaml:"monthFormat,omitempty"` // "text" if month column stores names (January, February, ...)
	// MonthBindString: when true, month filter placeholders bind "1".."12" as text for IN/= clauses.
	// Use for TEXT month columns (e.g. some CHT materialized views). Default binds integers.
	MonthBindString bool   `json:"monthBindString,omitempty" yaml:"monthBindString,omitempty"`
	Week            string `json:"week,omitempty" yaml:"week,omitempty"`       // Column name to extract week from (optional)
	Quarter         string `json:"quarter,omitempty" yaml:"quarter,omitempty"` // Column name to extract quarter from (optional)
}

// LocationColumns specifies which columns contain location data for filtering
// Used to configure custom column names for district, region, and facility
type LocationColumns struct {
	District      string `json:"district,omitempty" yaml:"district,omitempty"`           // Column name for district (defaults to "district")
	Region        string `json:"region,omitempty" yaml:"region,omitempty"`               // Column name for region (defaults to "region")
	Facility      string `json:"facility,omitempty" yaml:"facility,omitempty"`           // Column name for facility (defaults to "facility")
	DistrictWhere string `json:"districtWhere,omitempty" yaml:"districtWhere,omitempty"` // Optional extra WHERE clause applied when fetching district filter values
}

// RelatedReport represents a reference to another report (Phase 3)
type RelatedReport struct {
	ID    string `json:"id" yaml:"id"`
	Title string `json:"title" yaml:"title"`
}

// CustomFilterDef defines a custom column-based filter
type CustomFilterDef struct {
	Column       string `json:"column" yaml:"column"`                                 // Column name to filter on
	Table        string `json:"table" yaml:"table"`                                   // Table to query for distinct values
	Label        string `json:"label" yaml:"label"`                                   // Display label
	Type         string `json:"type" yaml:"type"`                                     // "select" or "multiselect"
	DefaultValue string `json:"defaultValue,omitempty" yaml:"defaultValue,omitempty"` // Optional default for single-select filters
	NoDefault    bool   `json:"noDefault,omitempty" yaml:"noDefault,omitempty"`       // Suppress automatic default selection
	HideNone     bool   `json:"hideNone,omitempty" yaml:"hideNone,omitempty"`         // Remove the blank "None" option from the dropdown
}

// FailedComponent describes a component that failed during execution
type FailedComponent struct {
	SectionIndex   int    `json:"sectionIndex"`
	ComponentIndex int    `json:"componentIndex"`
	Title          string `json:"title"`
	Error          string `json:"error"`
}

// Report represents a full report with sections
type Report struct {
	ID              string            `json:"id" yaml:"id"`
	Title           string            `json:"title" yaml:"title"`
	Description     string            `json:"description,omitempty" yaml:"description,omitempty"`
	Keywords        string            `json:"keywords,omitempty" yaml:"keywords,omitempty"` // Comma-separated keywords for search discoverability
	Author          string            `json:"author,omitempty" yaml:"author,omitempty"`     // Author name for audit trail (not displayed)
	Category        string            `json:"category,omitempty" yaml:"category,omitempty"` // Optional category override
	GeneratedAt     string            `json:"generated_at" yaml:"generated_at"`
	Filters         []string          `json:"filters,omitempty" yaml:"filters,omitempty"`                 // Report-level filters: ["district", "year", "month"]
	CustomFilters   []CustomFilterDef `json:"customFilters,omitempty" yaml:"customFilters,omitempty"`     // Custom column-based filters
	TimeColumns     TimeColumns       `json:"timeColumns,omitempty" yaml:"timeColumns,omitempty"`         // Maps filter types to column names
	LocationColumns LocationColumns   `json:"locationColumns,omitempty" yaml:"locationColumns,omitempty"` // Maps location filter types to column names
	// FilterOptionsTable overrides the table for district/region/facility option lists only (not year/week — those stay on the primary data table).
	// Use when {{district_filter}} binds to hierarchy but getPrimaryTable resolves to another table (e.g. mv_form_meta).
	FilterOptionsTable string            `json:"filterOptionsTable,omitempty" yaml:"filterOptionsTable,omitempty"`
	FilterDefinitions  FilterDefinitions `json:"filterDefinitions,omitempty" yaml:"-"`                     // Populated by backend
	AllEmpty           bool              `json:"allEmpty,omitempty" yaml:"-"`                              // True when all components returned empty data
	PartialFailure     bool              `json:"partialFailure,omitempty" yaml:"-"`                        // True when some components failed during execution
	FailedComponents   []FailedComponent `json:"failedComponents,omitempty" yaml:"-"`                      // Components that failed during execution
	RelatedReports     []RelatedReport   `json:"relatedReports,omitempty" yaml:"relatedReports,omitempty"` // Related reports (Phase 3)
	Cache              *bool             `json:"cache,omitempty" yaml:"cache,omitempty"`                   // false = bypass component cache (always live). Nil = default cached behaviour.
	Datasource         string            `json:"datasource,omitempty" yaml:"datasource,omitempty"`         // Optional: named connection (e.g. "report") routes this component's queries to that datasource
	Sections           []Section         `json:"sections" yaml:"sections"`
}

// Section represents a section within a report
type Section struct {
	ID          string      `json:"id" yaml:"id"`
	Title       string      `json:"title" yaml:"title"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Layout      string      `json:"layout" yaml:"layout"` // "single", "two-column", "three-column", or "four-column"
	Components  []Component `json:"components" yaml:"components"`
}

// FacetConfig defines faceting configuration for maps/choropleths
type FacetConfig struct {
	PeriodType string `json:"periodType" yaml:"periodType"` // month, quarter, year
	Count      int    `json:"count" yaml:"count"`           // 1-4 facets allowed
}

// FacetInstance represents a single facet with its label, value, and data
type FacetInstance struct {
	Label string      `json:"label"`
	Value string      `json:"value"`
	Data  interface{} `json:"data"`
}

// SplitColumnGroup defines a grouped table header where a parent spans child columns.
type SplitColumnGroup struct {
	Parent   string   `json:"parent,omitempty" yaml:"parent,omitempty"`
	Children []string `json:"children,omitempty" yaml:"children,omitempty"`
}

// Component represents a single visual or text element
type Component struct {
	Type               string                `json:"type" yaml:"type"` // "kpi", "text", "line", "bar", "table", "table_advanced", "map"
	Title              string                `json:"title,omitempty" yaml:"title,omitempty"`
	Description        string                `json:"description,omitempty" yaml:"description,omitempty"`
	Content            string                `json:"content,omitempty" yaml:"content,omitempty"`             // For text type
	Tone               string                `json:"tone,omitempty" yaml:"tone,omitempty"`                   // For kpi: accent | good | alert | muted (default omitted)
	Unit               string                `json:"unit,omitempty" yaml:"unit,omitempty"`                   // For kpi: literal suffix appended to value (e.g. "%")
	SQL                string                `json:"sql,omitempty" yaml:"sql,omitempty"`                     // For table_advanced and choropleth: raw SQL
	FilterTable        string                `json:"filterTable,omitempty" yaml:"filterTable,omitempty"`     // For table_advanced: table to apply filters to
	ColumnMapping      *ColumnMapping        `json:"columnMapping,omitempty" yaml:"columnMapping,omitempty"` // For choropleth with SQL: maps columns to district/value
	Data               Data                  `json:"data,omitempty" yaml:"data,omitempty"`
	Query              *Query                `json:"query,omitempty" yaml:"query,omitempty"`                           // Structured query
	Facet              *FacetConfig          `json:"facet,omitempty" yaml:"facet,omitempty"`                           // Faceting config for maps
	Stacked            bool                  `json:"stacked,omitempty" yaml:"stacked,omitempty"`                       // Bar chart: stack datasets
	Horizontal         bool                  `json:"horizontal,omitempty" yaml:"horizontal,omitempty"`                 // Bar chart: horizontal orientation
	WrapLabels         bool                  `json:"wrapLabels,omitempty" yaml:"wrapLabels,omitempty"`                 // Bar chart: wrap long category-axis labels onto multiple lines
	ShowValues         bool                  `json:"showValues,omitempty" yaml:"showValues,omitempty"`                 // Bar chart: show value labels on bars
	ShowValuesWithAxis bool                  `json:"showValuesWithAxis,omitempty" yaml:"showValuesWithAxis,omitempty"` // Bar chart: show value labels while keeping Y-axis visible
	ShowAxisTitles     *bool                 `json:"showAxisTitles,omitempty" yaml:"showAxisTitles,omitempty"`         // Bar/bar_line chart: show axis titles by default; set false to hide titles
	HideAxes           bool                  `json:"hideAxes,omitempty" yaml:"hideAxes,omitempty"`                     // Legacy alias for older configs
	Comparison         string                `json:"comparison,omitempty" yaml:"comparison,omitempty"`                 // For text: "previous_month" or "previous_year"
	LowerIsBetter      bool                  `json:"lowerIsBetter,omitempty" yaml:"lowerIsBetter,omitempty"`           // Invert comparison colors (decrease=green)
	Target             *TargetConfig         `json:"target,omitempty" yaml:"target,omitempty"`                         // For text: static target value comparison
	ReferenceLines     []ReferenceLineConfig `json:"referenceLines,omitempty" yaml:"referenceLines,omitempty"`         // For bar/line: horizontal reference lines
	TrendLine          bool                  `json:"trendLine,omitempty" yaml:"trendLine,omitempty"`                   // For line: show linear regression trend line
	TrendLineColor     string                `json:"trendLineColor,omitempty" yaml:"trendLineColor,omitempty"`         // Custom color for trend line
	Pyramid            bool                  `json:"pyramid,omitempty" yaml:"pyramid,omitempty"`                       // Population pyramid: negate first dataset for mirrored bar
	AxisLabels         *AxisLabelsConfig     `json:"axisLabels,omitempty" yaml:"axisLabels,omitempty"`                 // For bar/line: axis title labels
	AxisTooltips       map[string]string     `json:"axisTooltips,omitempty" yaml:"axisTooltips,omitempty"`             // For bar/line: optional tooltip text keyed by axis (x, y)
	AxisFormulas       map[string]string     `json:"axisFormulas,omitempty" yaml:"axisFormulas,omitempty"`             // For bar/line: optional formula text keyed by axis (x, y)
	YLimit             *YLimitConfig         `json:"yLimit,omitempty" yaml:"yLimit,omitempty"`                         // For bar/line/bar_line: fix value-axis range; data is clipped to ceiling

	ValueType            string             `json:"valueType,omitempty" yaml:"valueType,omitempty"`                       // For choropleth: "categorical" for string values instead of numeric
	Heatmap              *HeatmapConfig     `json:"heatmap,omitempty" yaml:"heatmap,omitempty"`                           // For tables: conditional cell coloring
	HeaderTooltips       map[string]string  `json:"headerTooltips,omitempty" yaml:"headerTooltips,omitempty"`             // For tables: optional tooltip text keyed by column header
	HeaderFormulas       map[string]string  `json:"headerFormulas,omitempty" yaml:"headerFormulas,omitempty"`             // For tables: optional formula text keyed by column header
	TableHeaderWrap      bool               `json:"tableHeaderWrap,omitempty" yaml:"tableHeaderWrap,omitempty"`           // For tables: allow wrapping long header labels
	TableHeaderStyle     string             `json:"tableHeaderStyle,omitempty" yaml:"tableHeaderStyle,omitempty"`         // For tables: "default", "enhanced-compact", or "enhanced-readable"
	SplitChildHeaderMode string             `json:"splitChildHeaderMode,omitempty" yaml:"splitChildHeaderMode,omitempty"` // For split columns: "symbols" (default) or "labels"
	SplitColumns         []SplitColumnGroup `json:"splitColumns,omitempty" yaml:"splitColumns,omitempty"`                 // For tables: grouped two-row headers
	TableCellBoundaries  bool               `json:"tableCellBoundaries,omitempty" yaml:"tableCellBoundaries,omitempty"`   // For tables: show cell boundary lines using table grid styling
	TableVerticalLines   bool               `json:"tableVerticalLines,omitempty" yaml:"tableVerticalLines,omitempty"`     // For tables: add vertical separators between columns
	PageSize             int                `json:"pageSize,omitempty" yaml:"pageSize,omitempty"`                         // For tables: rows per page (default: 20)
	FirstRowIsHeader     bool               `json:"firstRowIsHeader,omitempty" yaml:"firstRowIsHeader,omitempty"`         // For table_advanced: use first SQL row values as column headers (legacy; use Pivot instead)
	Pivot                *PivotConfig       `json:"pivot,omitempty" yaml:"pivot,omitempty"`                               // For table_advanced: server-side pivot transformation
	InfoboxBody          string             `json:"infoboxBody,omitempty" yaml:"infoboxBody,omitempty"`                   // For infobox: body text/HTML
	AccentColor          string             `json:"accentColor,omitempty" yaml:"accentColor,omitempty"`                   // For infobox/kpi: left border and title colour (hex, e.g. "#0f766e")
	ThresholdColumn      string             `json:"thresholdColumn,omitempty" yaml:"thresholdColumn,omitempty"`           // For infobox: SQL result column to evaluate against Thresholds (default: first column)
	Thresholds           []ThresholdRule    `json:"thresholds,omitempty" yaml:"thresholds,omitempty"`                     // For infobox: ordered list of threshold rules; first match wins
	ValueTemplate        string             `json:"valueTemplate,omitempty" yaml:"valueTemplate,omitempty"`               // For kpi: value display pattern with {{alias}} placeholders (default: "{{value}}")
	Error                string             `json:"error,omitempty" yaml:"-"`                                             // Set when component execution failed; not persisted to YAML
}

// ThresholdRule defines a single condition + accent color for infobox threshold coloring.
// Conditions use standard comparison operators against a numeric value:
//
//	">= 30"  — value is greater than or equal to 30
//	"> 0"    — value is above zero
//	"<= 15"  — value is at or below 15
//	"== 0"   — value equals zero exactly
type ThresholdRule struct {
	Condition string `json:"condition" yaml:"condition"` // Comparison expression, e.g. ">= 30" or "== 0"
	Color     string `json:"color" yaml:"color"`         // Hex accent color to apply when condition is met, e.g. "#922B21"
}

// AxisLabelsConfig defines axis title labels for charts
type AxisLabelsConfig struct {
	X  string `json:"x,omitempty" yaml:"x,omitempty"`   // X-axis title
	Y  string `json:"y,omitempty" yaml:"y,omitempty"`   // Left Y-axis title
	Y1 string `json:"y1,omitempty" yaml:"y1,omitempty"` // Right Y-axis title
}

// YLimitConfig fixes the value-axis range so charts stay visually consistent
// regardless of the underlying data. Bars and line points beyond Max are
// clipped at the ceiling. For bar_line, Y1 sets the right axis limits.
type YLimitConfig struct {
	Min *float64          `json:"min,omitempty" yaml:"min,omitempty"`
	Max *float64          `json:"max,omitempty" yaml:"max,omitempty"`
	Y1  *YLimitAxisConfig `json:"y1,omitempty" yaml:"y1,omitempty"`
}

// YLimitAxisConfig is the per-axis form used inside YLimitConfig.Y1.
type YLimitAxisConfig struct {
	Min *float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max *float64 `json:"max,omitempty" yaml:"max,omitempty"`
}

// LegendConfig defines chart legend display options
type LegendConfig struct {
	Show     bool   `json:"show" yaml:"show"`         // Whether to display the legend
	Position string `json:"position" yaml:"position"` // Position: top, bottom, left, right
}

// Data holds the flexible data structure from YAML
type Data struct {
	// Chart data fields (no omitempty - always include for consistent frontend handling)
	Labels   []string  `json:"labels" yaml:"labels,omitempty"`
	Datasets []Dataset `json:"datasets" yaml:"datasets,omitempty"`

	// Chart legend configuration
	Legend *LegendConfig `json:"legend,omitempty" yaml:"legend,omitempty"`

	// Table data fields
	Headers []string        `json:"headers,omitempty" yaml:"headers,omitempty"`
	Rows    [][]interface{} `json:"rows,omitempty" yaml:"rows,omitempty"`

	// Map data fields
	Center   []float64          `json:"center,omitempty" yaml:"center,omitempty"`
	Zoom     int                `json:"zoom,omitempty" yaml:"zoom,omitempty"`
	Markers  []Marker           `json:"markers,omitempty" yaml:"markers,omitempty"`
	Unmapped []UnmappedFacility `json:"unmapped,omitempty" yaml:"-"`
	// Optional CSV lookup used by facility maps. Expected columns:
	// district,facility,latitude,longitude.
	FacilityCoordinatesPath string `json:"facility_coordinates_path,omitempty" yaml:"facility_coordinates_path,omitempty"`
	ValueLabel              string `json:"value_label,omitempty" yaml:"value_label,omitempty"`

	// Choropleth data fields
	GeoJSONPath    string                 `json:"geojson_path,omitempty" yaml:"geojson_path,omitempty"`
	DistrictValues map[string]interface{} `json:"district_values,omitempty" yaml:"district_values,omitempty"`
	ColorScheme    []string               `json:"color_scheme,omitempty" yaml:"color_scheme,omitempty"`
	BinLabels      []string               `json:"bin_labels,omitempty" yaml:"bin_labels,omitempty"`
	BinEdges       []float64              `json:"bin_edges,omitempty" yaml:"bin_edges,omitempty"`
	LegendTitle    string                 `json:"legend_title,omitempty" yaml:"legend_title,omitempty"`

	// Faceted map fields
	Facets []FacetInstance `json:"facets,omitempty" yaml:"facets,omitempty"`

	// Comparison data for text components
	Comparison       *ComparisonData       `json:"comparison,omitempty" yaml:"-"`
	TargetComparison *TargetComparisonData `json:"targetComparison,omitempty" yaml:"-"`

	// Reference lines for bar/line charts (passed through from component config)
	ReferenceLines []ReferenceLineConfig `json:"referenceLines,omitempty" yaml:"-"`

	// Trend line for line charts (passed through from component config)
	TrendLine      bool   `json:"trendLine,omitempty" yaml:"-"`
	TrendLineColor string `json:"trendLineColor,omitempty" yaml:"-"`

	// Axis labels for bar/line charts (passed through from component config)
	AxisLabels *AxisLabelsConfig `json:"axisLabels,omitempty" yaml:"-"`

	// Y-axis limit for bar/line/bar_line charts (passed through from component config)
	YLimit *YLimitConfig `json:"yLimit,omitempty" yaml:"-"`

	// Per-column formatting hints derived from the component's Query (agg/calc roundTo).
	// Keyed by column/dataset name (aggregation alias or calculate resultAlias).
	// Frontend formatters use this to render exactly the author's requested precision.
	ColumnFormats map[string]ColumnFormat `json:"columnFormats,omitempty" yaml:"-"`
}

// ColumnFormat carries display-level formatting for a single result column.
type ColumnFormat struct {
	Decimals *int `json:"decimals,omitempty"`
}

// ============== Structured Query Models ==============

// Query represents a structured database query (replaces raw SQL)
type Query struct {
	Table         string           `json:"table,omitempty" yaml:"table,omitempty"`
	Where         string           `json:"where,omitempty" yaml:"where,omitempty"` // Custom WHERE condition (e.g., "age_group = 'under_5'")
	Aggregations  []Aggregation    `json:"aggregations,omitempty" yaml:"aggregations,omitempty"`
	GroupBy       []GroupBy        `json:"groupBy,omitempty" yaml:"groupBy,omitempty"`
	OrderBy       []OrderBy        `json:"orderBy,omitempty" yaml:"orderBy,omitempty"`
	Calculate     CalculateList    `json:"calculate,omitempty" yaml:"calculate,omitempty"`         // Post-aggregation calculations (supports single object or array)
	SelectColumns []string         `json:"selectColumns,omitempty" yaml:"selectColumns,omitempty"` // Which columns to display in chart (filters in transformer)
	Transpose     *TransposeConfig `json:"transpose,omitempty" yaml:"transpose,omitempty"`         // Transpose: metric names become rows, periods become columns
	SeriesBy      string           `json:"seriesBy,omitempty" yaml:"seriesBy,omitempty"`           // Column to pivot into separate datasets (one line/bar per unique value)
	Limit         int              `json:"limit,omitempty" yaml:"limit,omitempty"`
	PeriodLimit   int              `json:"periodLimit,omitempty" yaml:"periodLimit,omitempty"` // Auto-expand period range to show N periods ending at selected period
}

// TransposeConfig defines settings for transposing table data
// Transpose swaps rows and columns: metric names become rows, first column values become column headers
type TransposeConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`                       // Whether transpose is enabled
	RowLabel string `json:"rowLabel,omitempty" yaml:"rowLabel,omitempty"` // Label for first column (default: "Indicator")
}

// Aggregation defines a column aggregation (SUM, COUNT, AVG, MAX, MIN, VARIANCE, STDDEV)
type Aggregation struct {
	Column    string `json:"column" yaml:"column"`                           // Can be expression: "col1 + col2"
	Function  string `json:"function" yaml:"function"`                       // sum, count, avg, max, min, variance, stddev
	Alias     string `json:"alias" yaml:"alias"`                             // Result column name
	Color     string `json:"color,omitempty" yaml:"color,omitempty"`         // Color for chart display (optional)
	ChartType string `json:"chartType,omitempty" yaml:"chartType,omitempty"` // Chart type for bar_line combo (bar or line)
	RoundTo   *int   `json:"roundTo,omitempty" yaml:"roundTo,omitempty"`     // Decimal places for result; ignored for COUNT/COUNT_DISTINCT
}

// GroupBy defines grouping and formatting
type GroupBy struct {
	Field     string `json:"field" yaml:"field"`                             // Column name
	Format    string `json:"format,omitempty" yaml:"format,omitempty"`       // month, year, date (for date columns)
	Alias     string `json:"alias,omitempty" yaml:"alias,omitempty"`         // Custom label for grouping column (default: "label")
	YearField string `json:"yearField,omitempty" yaml:"yearField,omitempty"` // Column name for year (default: "year")
}

// GetFormat returns the format, inferring from field name when format is empty.
// This handles YAMLs generated by the builder which may omit the format field.
// e.g. field:"month" with no format → "month", field:"quarter" → "quarter"
func (gb GroupBy) GetFormat() string {
	if gb.Format != "" {
		return gb.Format
	}
	switch strings.ToLower(gb.Field) {
	case "month":
		return "month"
	case "week", "period":
		return "week"
	case "quarter":
		return "quarter"
	default:
		return ""
	}
}

// IsDateField returns true if the field name indicates a date/timestamp column.
// Checks for "date" substring and common timestamp column names (reported, created_at, etc.)
func (gb GroupBy) IsDateField() bool {
	lower := strings.ToLower(gb.Field)
	if strings.Contains(lower, "date") {
		return true
	}
	switch lower {
	case "reported", "created_at", "updated_at", "submitted", "registered":
		return true
	}
	return false
}

// GetYearField returns the year column name, defaulting to "year" when not set.
// If no explicit YearField and the GroupBy field itself IS a year column
// (format="year" or name contains "year"), use it directly so capitalized
// or differently-named year columns (e.g. "Year") resolve correctly.
func (gb GroupBy) GetYearField() string {
	if gb.YearField != "" {
		return gb.YearField
	}
	lower := strings.ToLower(gb.Field)
	if strings.EqualFold(gb.GetFormat(), "year") || strings.Contains(lower, "year") {
		return gb.Field
	}
	return "year"
}

// Calculate defines post-aggregation calculation
type Calculate struct {
	Formula     string      `json:"formula" yaml:"formula"`                             // "(positive / tested * 100)"
	RoundTo     *int        `json:"roundTo,omitempty" yaml:"roundTo,omitempty"`         // Decimal places
	WhenZero    interface{} `json:"whenZero,omitempty" yaml:"whenZero,omitempty"`       // Fallback value
	ResultAlias string      `json:"resultAlias,omitempty" yaml:"resultAlias,omitempty"` // Custom name for calculated column (default: "value")
	Color       string      `json:"color,omitempty" yaml:"color,omitempty"`             // Color for chart display
	ChartType   string      `json:"chartType,omitempty" yaml:"chartType,omitempty"`     // Chart type for bar_line combo (bar or line)
	Alias       string      `json:"alias,omitempty" yaml:"alias,omitempty"`             // Backward compat alias for resultAlias
}

// CalculateList is a custom type that can unmarshal from either a single Calculate object or an array
type CalculateList []Calculate

// UnmarshalYAML implements custom YAML unmarshaling for backward compatibility
// Accepts both: calculate: {formula: ...} AND calculate: [{formula: ...}]
func (c *CalculateList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as array first
	var arr []Calculate
	if err := unmarshal(&arr); err == nil {
		*c = arr
		return nil
	}

	// Try to unmarshal as single object
	var single Calculate
	var lastErr error
	if lastErr = unmarshal(&single); lastErr == nil {
		*c = []Calculate{single}
		return nil
	}

	logWarn("CalculateList: failed to unmarshal calculate field: %v", lastErr)
	return lastErr
}

// UnmarshalJSON implements custom JSON unmarshaling for the UI builder
// Accepts both: "calculate": {...} AND "calculate": [{...}]
func (c *CalculateList) UnmarshalJSON(data []byte) error {
	// Handle null
	if string(data) == "null" {
		*c = nil
		return nil
	}

	// Try to unmarshal as array first
	var arr []Calculate
	if err := json.Unmarshal(data, &arr); err == nil {
		*c = arr
		return nil
	}

	// Try to unmarshal as single object
	var single Calculate
	var lastErr error
	if lastErr = json.Unmarshal(data, &single); lastErr == nil {
		*c = []Calculate{single}
		return nil
	}

	logWarn("CalculateList: failed to unmarshal calculate field: %v", lastErr)
	return lastErr
}

// OrderBy defines custom ordering for query results
type OrderBy struct {
	Field     string `json:"field" yaml:"field"`                             // Column or alias
	Direction string `json:"direction,omitempty" yaml:"direction,omitempty"` // ASC or DESC (default: ASC)
}

// ReportsList wraps the array of reports for API response
type ReportsList struct {
	Reports  []ReportListItem `json:"reports" yaml:"reports"`
	Sections []string         `json:"sections" yaml:"sections"` // All section paths including empty ones (e.g., "Programs", "Views/Analytics")
}

// TableData represents data for a table component
type TableData struct {
	Headers []string        `json:"headers"`
	Rows    [][]interface{} `json:"rows"`
}

// ChartData represents data for chart components
type ChartData struct {
	Labels   []string  `json:"labels"`
	Datasets []Dataset `json:"datasets"`
}

// Dataset represents a single dataset in a chart
type Dataset struct {
	Label     string        `json:"label" yaml:"label"`
	Data      []interface{} `json:"data" yaml:"data,omitempty"`
	Color     string        `json:"color,omitempty" yaml:"color,omitempty"`
	Colors    []string      `json:"colors,omitempty" yaml:"colors,omitempty"` // per-bar color array (overrides Color when set)
	ChartType string        `json:"chartType,omitempty" yaml:"chartType,omitempty"`
}

// MapData represents data for a map component
type MapData struct {
	Center     []float64          `json:"center"`
	Zoom       int                `json:"zoom"`
	Markers    []Marker           `json:"markers"`
	Unmapped   []UnmappedFacility `json:"unmapped,omitempty"`
	ValueLabel string             `json:"value_label,omitempty"`
}

// Marker represents a single marker on a map
type Marker struct {
	Lat   float64     `json:"lat"`
	Lng   float64     `json:"lng"`
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// UnmappedFacility retains totals for facilities that do not yet have
// verified coordinates, preventing them from disappearing silently.
type UnmappedFacility struct {
	District string      `json:"district"`
	Label    string      `json:"label"`
	Value    interface{} `json:"value"`
}

// ChoroplethData represents data for a choropleth map component
type ChoroplethData struct {
	GeoJSONPath    string                 `json:"geojson_path"`
	DistrictValues map[string]interface{} `json:"district_values"`
	ColorScheme    []string               `json:"color_scheme"`
	BinLabels      []string               `json:"bin_labels,omitempty"`
	BinEdges       []float64              `json:"bin_edges,omitempty"`
	LegendTitle    string                 `json:"legend_title,omitempty"`
}

// ColumnMapping defines explicit column mappings for choropleth SQL queries
// Used when the user writes custom SQL and needs to specify which columns
// contain the district name and value
type ColumnMapping struct {
	District string `json:"district" yaml:"district"` // Column name containing district/location names
	Value    string `json:"value" yaml:"value"`       // Column name containing the numeric value
}

// FilterOptions represents available filter values
type FilterOptions struct {
	Districts []string `json:"districts"`
	Years     []int    `json:"years"`
	Months    []int    `json:"months"`
}

// ReportFilters represents user-selected filter values from query parameters
type ReportFilters struct {
	Districts []string // empty slice means "All" (multi-select)
	Year      string   // empty string means "All"
	Month     string   // empty string means "All"
	Week      string   // empty string means "All" (format: "2024W15")
	Quarter   string   // empty string means "All" (1-4)
	// SuppressTimeDefaults disables server-side smart time defaults when no
	// time filters are provided (used by explicit "Clear" action in UI).
	SuppressTimeDefaults bool

	// Track which time filters were explicitly provided by user (vs auto-defaulted)
	// Used by previous_period comparison to determine user intent
	YearProvided    bool
	WeekProvided    bool
	MonthProvided   bool
	QuarterProvided bool

	// Multi-select location filters (mutually exclusive - only one should be used at a time)
	Regions    []string // empty slice means "All"
	Facilities []string // empty slice means "All"

	// YearMonths maps years to their valid months for period expansion
	// Used to avoid cross-product when filtering across year boundaries
	// Example: {"2023": [10,11,12], "2024": [1,2]} for Oct 2023 - Feb 2024
	YearMonths map[string][]int

	// YearQuarters maps years to their valid quarters for period expansion
	// Used to avoid cross-product when filtering across year boundaries
	// Example: {"2023": [3,4], "2024": [1,2]} for Q3 2023 - Q2 2024
	YearQuarters map[string][]int

	// CustomValues holds filter values for custom column filters
	// Key: column name, Value: filter value(s) - comma-separated for multiselect
	CustomValues map[string]string
}

// ComparisonData holds period-over-period comparison data for text components
type ComparisonData struct {
	PreviousValue    float64 `json:"previousValue"`
	Change           float64 `json:"change"`
	PercentageChange float64 `json:"percentageChange"`
	Direction        string  `json:"direction"` // "up", "down", "unchanged"
}

// TargetConfig defines a static target value for comparison in text components
type TargetConfig struct {
	Value float64 `json:"value" yaml:"value"`                     // Target value to compare against
	Label string  `json:"label,omitempty" yaml:"label,omitempty"` // Optional label (defaults to "Target")
}

// TargetComparisonData holds target comparison data for text components
type TargetComparisonData struct {
	TargetValue      float64 `json:"targetValue"`
	TargetLabel      string  `json:"targetLabel"`
	Change           float64 `json:"change"`
	PercentageChange float64 `json:"percentageChange"`
	Direction        string  `json:"direction"` // "up" (above target), "down" (below target), "unchanged"
}

// HeatmapConfig defines conditional cell coloring for table components
type HeatmapConfig struct {
	Scale   string          `json:"scale,omitempty" yaml:"scale,omitempty"`     // "green-red", "red-green", or "blue"
	Rules   []HeatmapRule   `json:"rules,omitempty" yaml:"rules,omitempty"`     // Per-column threshold rules (mutually exclusive with Scale)
	Columns []HeatmapColumn `json:"columns,omitempty" yaml:"columns,omitempty"` // Per-column scale overrides; when set, only listed columns get heatmap
}

// HeatmapRule defines a per-column threshold rule for discrete coloring
type HeatmapRule struct {
	Column     string    `json:"column" yaml:"column"`
	Thresholds []float64 `json:"thresholds" yaml:"thresholds"`
	Colors     string    `json:"colors" yaml:"colors"` // "red-yellow-green" or "green-yellow-red"
}

// HeatmapColumn restricts heatmap coloring to a specific column and optionally overrides the scale direction
type HeatmapColumn struct {
	Name  string `json:"name" yaml:"name"`
	Scale string `json:"scale,omitempty" yaml:"scale,omitempty"` // overrides HeatmapConfig.Scale for this column
}

// PivotExtraItem defines an extra computed column or row appended to a pivot table
type PivotExtraItem struct {
	Label   string `json:"label,omitempty" yaml:"label,omitempty"`     // Header / row label shown in the table
	Formula string `json:"formula,omitempty" yaml:"formula,omitempty"` // "sum", "avg", or expression using [-N] positional refs
}

// PivotConfig defines a server-side pivot transformation for table_advanced components.
// The SQL must return exactly the three named columns (columns, rows, values) already
// aggregated — the server pivots them into a cross-tab at runtime.
type PivotConfig struct {
	Columns      string           `json:"columns,omitempty" yaml:"columns,omitempty"`           // SQL column whose distinct values become table column headers
	Rows         string           `json:"rows,omitempty" yaml:"rows,omitempty"`                 // SQL column whose distinct values become row labels
	Values       string           `json:"values,omitempty" yaml:"values,omitempty"`             // SQL column whose values fill the cells
	Format       string           `json:"format,omitempty" yaml:"format,omitempty"`             // Cell formatting: "auto" (default), "integer", "decimal:N", "percent:N"
	ExtraColumns []PivotExtraItem `json:"extraColumns,omitempty" yaml:"extraColumns,omitempty"` // Computed columns appended after pivot columns
	ExtraRows    []PivotExtraItem `json:"extraRows,omitempty" yaml:"extraRows,omitempty"`       // Computed rows appended after data rows
}

// ReferenceLineConfig defines a horizontal reference line for bar/line charts
type ReferenceLineConfig struct {
	Value float64 `json:"value" yaml:"value"`                     // Y-axis value for the line
	Label string  `json:"label" yaml:"label"`                     // Label text (displayed on left edge)
	Color string  `json:"color,omitempty" yaml:"color,omitempty"` // Line color (default: green for first, gray for others)
	Style string  `json:"style,omitempty" yaml:"style,omitempty"` // Line style: "dashed" (default) or "dotted"
	Axis  string  `json:"axis,omitempty" yaml:"axis,omitempty"`   // Y-axis to use: "y" (left, default) or "y1" (right, for bar_line charts)
}

// ComponentResult holds the execution result for a single component processed in a goroutine
type ComponentResult struct {
	SectionIdx       int                   `json:"sectionIdx"`
	ComponentIdx     int                   `json:"componentIdx"`
	Data             Data                  `json:"data"`
	Content          string                `json:"content,omitempty"`
	AccentColor      string                `json:"accentColor,omitempty"` // Overrides component.AccentColor when a threshold rule matches
	Comparison       *ComparisonData       `json:"comparison,omitempty"`
	TargetComparison *TargetComparisonData `json:"targetComparison,omitempty"`
	Error            error                 `json:"-"`
}

// ============== Schema Introspection Models ==============

// TableInfo represents a database table
type TableInfo struct {
	Schema   string `json:"schema"`
	Table    string `json:"table"`
	RowCount int64  `json:"rowCount,omitempty"`
}

// ColumnInfo represents a database column with metadata
type ColumnInfo struct {
	Name         string `json:"name"`
	DataType     string `json:"dataType"`
	IsNullable   bool   `json:"isNullable"`
	DefaultValue string `json:"defaultValue,omitempty"`
	MaxLength    int    `json:"maxLength,omitempty"`
	IsNumeric    bool   `json:"isNumeric"` // For TEXT-stored numeric columns
}

// TableColumnsResponse wraps column list for API response
type TableColumnsResponse struct {
	Table   string       `json:"table"`
	Columns []ColumnInfo `json:"columns"`
}

// TablePreviewResponse wraps sample data for API response
type TablePreviewResponse struct {
	Headers   []string        `json:"headers"`
	Rows      [][]interface{} `json:"rows"`
	RowCount  int             `json:"rowCount"`
	TotalRows int64           `json:"totalRows,omitempty"`
}

// ============== Query Validation Models ==============

// QueryValidationRequest represents a structured query to validate
type QueryValidationRequest struct {
	Table        string        `json:"table"`
	Where        string        `json:"where,omitempty"`
	Aggregations []Aggregation `json:"aggregations"`
	GroupBy      []GroupBy     `json:"groupBy,omitempty"`
	OrderBy      []OrderBy     `json:"orderBy,omitempty"`
	Calculate    CalculateList `json:"calculate,omitempty"`
}

// QueryValidationResult returns validation result and generated SQL
type QueryValidationResult struct {
	Valid        bool     `json:"valid"`
	GeneratedSQL string   `json:"generatedSQL"`
	Errors       []string `json:"errors"`
	Warnings     []string `json:"warnings"`
}

// QueryPreviewRequest represents a query to preview with sample data
type QueryPreviewRequest struct {
	Table        string        `json:"table"`
	Datasource   string        `json:"datasource,omitempty"` // Optional named .env connection to preview against
	Where        string        `json:"where,omitempty"`
	Aggregations []Aggregation `json:"aggregations"`
	GroupBy      []GroupBy     `json:"groupBy,omitempty"`
	OrderBy      []OrderBy     `json:"orderBy,omitempty"`
	Calculate    CalculateList `json:"calculate,omitempty"`
	Limit        int           `json:"limit,omitempty"` // Default 10
}

// QueryPreviewResult returns sample data from query execution
type QueryPreviewResult struct {
	Headers        []string        `json:"headers"`
	Rows           [][]interface{} `json:"rows"`
	RowCount       int             `json:"rowCount"`
	ExecutionTime  string          `json:"executionTime"`
	AppliedFilters string          `json:"appliedFilters,omitempty"` // Shows which filters were applied
	Error          string          `json:"error,omitempty"`
}

// ============== YAML Generation & Parsing Models ==============

// YAMLGenerateRequest represents a complete report to generate YAML from
type YAMLGenerateRequest struct {
	ID              string            `json:"id"`
	Title           string            `json:"title"`
	Description     string            `json:"description,omitempty"`
	Keywords        string            `json:"keywords,omitempty"` // Comma-separated keywords for search
	Category        string            `json:"category,omitempty"`
	Datasource      string            `json:"datasource,omitempty"` // Optional: named .env connection to read from
	Filters         []string          `json:"filters,omitempty"`
	CustomFilters   []CustomFilterDef `json:"customFilters,omitempty"`
	TimeColumns     TimeColumns       `json:"timeColumns,omitempty"`
	LocationColumns LocationColumns   `json:"locationColumns,omitempty"`
	Sections        []Section         `json:"sections"`
}

// YAMLGenerateResponse returns generated YAML and metadata
type YAMLGenerateResponse struct {
	YAML         string              `json:"yaml"`
	Filename     string              `json:"filename"`
	SavePath     string              `json:"savePath"`
	SQLProcessed []SQLProcessingInfo `json:"sqlProcessed,omitempty"` // Info about SQL processing for table_advanced components
	Warnings     []ReportWarning     `json:"warnings,omitempty"`     // Design quality warnings
}

// SQLProcessingInfo provides feedback about SQL processing during YAML generation
type SQLProcessingInfo struct {
	ComponentIndex int      `json:"componentIndex"`         // Which component was processed
	DetectedTable  string   `json:"detectedTable"`          // Table detected from FROM clause
	Replacements   []string `json:"replacements,omitempty"` // Hardcoded values that were replaced
	FiltersApplied []string `json:"filtersApplied"`         // Which filters were injected
	Warnings       []string `json:"warnings,omitempty"`     // Any warnings (e.g., columns not found)
}

// YAMLParseRequest represents YAML string to parse
type YAMLParseRequest struct {
	YAML string `json:"yaml"`
}

// YAMLParseResult returns parsed report structure
type YAMLParseResult struct {
	Report Report   `json:"report"`
	Errors []string `json:"errors"`
	Valid  bool     `json:"valid"`
}

// ============== Advanced SQL Models ==============

// AdvancedSQLPreviewRequest represents a raw SQL query to preview
type AdvancedSQLPreviewRequest struct {
	SQL        string            `json:"sql"`
	Datasource string            `json:"datasource,omitempty"` // Optional named .env connection to preview against
	Filters    map[string]string `json:"filters,omitempty"`    // {{district}}, {{year}}, etc.
	Limit      int               `json:"limit,omitempty"`      // Default 100, max 1000
	Pivot      *PivotConfig      `json:"pivot,omitempty"`      // Optional: apply pivot transformation to preview result
}

// AdvancedSQLPreviewResult returns results from raw SQL execution
type AdvancedSQLPreviewResult struct {
	Headers       []string        `json:"headers"`
	Rows          [][]interface{} `json:"rows"`
	RowCount      int             `json:"rowCount"`
	ExecutionTime string          `json:"executionTime"`
	Error         string          `json:"error,omitempty"`
}
