package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// toFloat64 safely converts an interface{} to float64
// Handles float64, float32, int, int64, int32, and string types
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case string:
		trimmed := strings.TrimSpace(val)
		if trimmed == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	case nil:
		return 0, false
	default:
		return 0, false
	}
}

// toString safely converts an interface{} to string
func toString(v interface{}) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case []byte:
		return string(val), true
	case nil:
		return "", false
	default:
		return fmt.Sprintf("%v", val), true
	}
}

// DataTransformer interface defines how to transform raw SQL results into component-specific data
type DataTransformer interface {
	Transform(tableData TableData, component *Component) (interface{}, error)
}

// TransformerRegistry holds all registered transformers, keyed by component type
type TransformerRegistry map[string]DataTransformer

// NewTransformerRegistry creates and initializes a transformer registry with all built-in transformers
func NewTransformerRegistry() TransformerRegistry {
	return TransformerRegistry{
		"line":     &ChartTransformer{},
		"bar":      &ChartTransformer{},
		"bar_line": &ChartTransformer{},
		"pie":      &ChartTransformer{},
		"pyramid":  &ChartTransformer{},

		"table":      &TableTransformer{},
		"map":        &MapTransformer{},
		"choropleth": &ChoroplethTransformer{},
	}
}

// Global transformer registry (initialized at package load)
var transformerRegistry = NewTransformerRegistry()

// transformData transforms raw SQL query results into the format expected by the frontend components.
// Uses the transformer registry to dispatch to the appropriate transformer based on component type.
func transformData(componentType string, tableData TableData, component *Component) (interface{}, error) {
	log.Printf("Transforming data for component type: %s", componentType)

	transformer, exists := transformerRegistry[componentType]
	if !exists {
		return nil, fmt.Errorf("unknown component type: %s", componentType)
	}

	return transformer.Transform(tableData, component)
}

// ============== Transformer Implementations ==============

// ChartTransformer implements DataTransformer for line and bar charts
type ChartTransformer struct{}

// filterTableDataBySelectColumns filters table data to only include selected columns
// If selectColumns is empty or nil, returns the original data unchanged
// hasGroupBy indicates if the first column is a label (should always be kept for bar/line charts)
func filterTableDataBySelectColumns(tableData TableData, selectColumns []string) TableData {
	if len(selectColumns) == 0 {
		return tableData
	}

	// Build a set of selected columns for fast lookup
	selectedSet := make(map[string]bool)
	for _, col := range selectColumns {
		selectedSet[col] = true
	}

	// Find indices of columns to keep
	keepIndices := []int{}
	newHeaders := []string{}

	for i, header := range tableData.Headers {
		if selectedSet[header] {
			keepIndices = append(keepIndices, i)
			newHeaders = append(newHeaders, header)
		}
	}

	// If no columns matched, return original data
	if len(keepIndices) == 0 {
		return tableData
	}

	// Filter rows to only include selected columns
	newRows := make([][]interface{}, len(tableData.Rows))
	for i, row := range tableData.Rows {
		newRow := make([]interface{}, len(keepIndices))
		for j, idx := range keepIndices {
			if idx < len(row) {
				newRow[j] = row[idx]
			}
		}
		newRows[i] = newRow
	}

	return TableData{
		Headers: newHeaders,
		Rows:    newRows,
	}
}

// filterTableDataWithLabel is like filterTableDataBySelectColumns but always keeps the first column (label)
// Used for bar/line charts where the first column is the groupBy label
func filterTableDataWithLabel(tableData TableData, selectColumns []string) TableData {
	if len(selectColumns) == 0 {
		return tableData
	}

	// Build a set of selected columns for fast lookup
	selectedSet := make(map[string]bool)
	for _, col := range selectColumns {
		selectedSet[col] = true
	}

	// Always keep the first column (label)
	keepIndices := []int{0}
	newHeaders := []string{tableData.Headers[0]}

	for i, header := range tableData.Headers {
		if i == 0 {
			continue // Already added
		}
		if selectedSet[header] {
			keepIndices = append(keepIndices, i)
			newHeaders = append(newHeaders, header)
		}
	}

	// If only the label column matched, return original data
	if len(keepIndices) <= 1 && len(tableData.Headers) > 1 {
		return tableData
	}

	// Filter rows to only include selected columns
	newRows := make([][]interface{}, len(tableData.Rows))
	for i, row := range tableData.Rows {
		newRow := make([]interface{}, len(keepIndices))
		for j, idx := range keepIndices {
			if idx < len(row) {
				newRow[j] = row[idx]
			}
		}
		newRows[i] = newRow
	}

	return TableData{
		Headers: newHeaders,
		Rows:    newRows,
	}
}

// Transform transforms raw table data into chart format
func (ct *ChartTransformer) Transform(tableData TableData, component *Component) (interface{}, error) {
	if len(tableData.Rows) == 0 {
		// Return chart data with empty arrays (not nil) to prevent null pointer errors on frontend
		return ChartData{
			Labels:   []string{},
			Datasets: []Dataset{},
		}, nil
	}

	// Check if this is a pie chart without groupBy (aggregations only, no grouping)
	hasGroupBy := component.Query != nil && len(component.Query.GroupBy) > 0
	// For raw SQL charts (no structured Query, or Query with only periodLimit/metadata),
	// determine behavior from component type:
	// pie → no groupBy (headers as labels, single row as values)
	// bar/line/bar_line/pyramid → groupBy (column 0 as labels, rest as datasets)
	if component.SQL != "" && !hasGroupBy {
		hasGroupBy = component.Type != "pie"
	}

	// Apply selectColumns filter if specified
	// Use different filter based on whether there's a groupBy (label column)
	if component.Query != nil && len(component.Query.SelectColumns) > 0 {
		if hasGroupBy {
			// For bar/line charts, keep the label column (first column) plus selected columns
			tableData = filterTableDataWithLabel(tableData, component.Query.SelectColumns)
		} else {
			// For pie charts, only keep selected columns (no label column)
			tableData = filterTableDataBySelectColumns(tableData, component.Query.SelectColumns)
		}
	}

	if !hasGroupBy {
		// Pie chart case: headers are the labels, create a single dataset with all values
		labels := tableData.Headers

		// Extract all values from the first (and only) row
		dataPoints := make([]interface{}, len(tableData.Rows[0]))
		for j, val := range tableData.Rows[0] {
			dataPoints[j] = val
		}

		// Build color array from aggregation colors or dataset colors
		colors := make([]string, len(labels))
		defaultPalette := []string{
			"#0f766e", "#e74c3c", "#2ecc71", "#d97706", "#6366f1",
			"#14b8a6", "#334155", "#f59e0b", "#8b5cf6", "#0d9488",
		}
		if component.Query != nil {
			for i := 0; i < len(labels) && i < len(component.Query.Aggregations); i++ {
				if component.Query.Aggregations[i].Color != "" {
					colors[i] = component.Query.Aggregations[i].Color
				} else if i < len(component.Data.Datasets) && component.Data.Datasets[i].Color != "" {
					colors[i] = component.Data.Datasets[i].Color
				} else {
					colors[i] = defaultPalette[i%len(defaultPalette)]
				}
			}
		}
		// Fill any remaining unset colors (for raw SQL charts or extra labels)
		for i := range colors {
			if colors[i] == "" {
				if i < len(component.Data.Datasets) && component.Data.Datasets[i].Color != "" {
					colors[i] = component.Data.Datasets[i].Color
				} else {
					colors[i] = defaultPalette[i%len(defaultPalette)]
				}
			}
		}

		// Create datasets with one color per aggregation (for pie chart color mapping)
		datasets := make([]Dataset, len(labels))
		for i, color := range colors {
			datasets[i] = Dataset{
				Label: labels[i],
				Data:  []interface{}{dataPoints[i]},
				Color: color,
			}
		}

		return ChartData{
			Labels:   labels,
			Datasets: datasets,
		}, nil
	}

	// SeriesBy pivot: flat rows → one dataset per unique series value
	if component.Query != nil && component.Query.SeriesBy != "" {
		return ct.transformWithSeriesPivot(tableData, component)
	}

	// Bar/Line chart case: first column is labels, remaining columns are datasets
	labels := make([]string, len(tableData.Rows))
	for i, row := range tableData.Rows {
		// Accept both string and numeric labels; convert to string
		switch v := row[0].(type) {
		case string:
			labels[i] = v
		case int, int32, int64, float32, float64:
			// Convert numeric types to string
			labels[i] = fmt.Sprintf("%v", v)
		default:
			// Fallback for any other type
			labels[i] = fmt.Sprintf("%v", v)
		}
	}

	numDatasets := len(tableData.Headers) - 1

	// Build a map of column names to their colors/chartTypes from aggregations and datasets
	colorMap := make(map[string]string)
	chartTypeMap := make(map[string]string)

	// First, map aggregation aliases to colors and chartTypes from aggregations
	if component.Query != nil {
		for _, agg := range component.Query.Aggregations {
			if agg.Color != "" {
				colorMap[agg.Alias] = agg.Color
			}
			if agg.ChartType != "" {
				chartTypeMap[agg.Alias] = agg.ChartType
			}
		}

		// Map calculated columns to their colors and chartTypes
		for _, calc := range component.Query.Calculate {
			alias := calc.ResultAlias
			if alias == "" {
				alias = "value"
			}
			if calc.Color != "" {
				colorMap[alias] = calc.Color
			}
			if calc.ChartType != "" {
				chartTypeMap[alias] = calc.ChartType
			}
		}
	}

	// Then, map dataset labels to colors/chartTypes from data.datasets
	for _, ds := range component.Data.Datasets {
		if ds.Color != "" {
			colorMap[ds.Label] = ds.Color
		}
		if ds.ChartType != "" {
			chartTypeMap[ds.Label] = ds.ChartType
		}
	}

	// Default color palette for columns without explicit colors
	defaultPalette := []string{
		"#0f766e", "#e74c3c", "#2ecc71", "#d97706", "#6366f1",
		"#14b8a6", "#334155", "#f59e0b", "#8b5cf6", "#0d9488",
	}

	// Check if colorMap contains entries keyed by x-axis label values rather than column names.
	// This happens when data.datasets labels match category VALUES (e.g. "Access", "Watch") instead
	// of SQL column names (e.g. "Percentage"). When detected, build a per-bar Colors array so each
	// bar gets its configured color even though there is only one metric column.
	allLabelsInColorMap := numDatasets == 1 && len(labels) > 0 && len(colorMap) > 0
	if allLabelsInColorMap {
		for _, lbl := range labels {
			if colorMap[lbl] == "" {
				allLabelsInColorMap = false
				break
			}
		}
	}

	datasets := make([]Dataset, numDatasets)
	for i := 0; i < numDatasets; i++ {
		datasetIndex := i + 1
		datasetLabel := tableData.Headers[datasetIndex]

		dataPoints := make([]interface{}, len(tableData.Rows))
		for j, row := range tableData.Rows {
			dataPoints[j] = row[datasetIndex]
		}

		// Look up color by header name, fall back to default palette
		color := colorMap[datasetLabel]
		if color == "" {
			color = defaultPalette[i%len(defaultPalette)]
		}

		// Look up chartType by header name (for bar_line charts)
		chartType := chartTypeMap[datasetLabel]

		ds := Dataset{
			Label:     datasetLabel,
			Data:      dataPoints,
			Color:     color,
			ChartType: chartType,
		}

		// When colorMap is keyed by x-axis label values, build a per-bar Colors array
		// so each individual bar gets its configured color.
		if allLabelsInColorMap {
			perBarColors := make([]string, len(labels))
			for j, lbl := range labels {
				perBarColors[j] = colorMap[lbl]
			}
			ds.Colors = perBarColors
		}

		datasets[i] = ds
	}

	return ChartData{
		Labels:   labels,
		Datasets: datasets,
	}, nil
}

// transformWithSeriesPivot pivots flat rows into one dataset per unique series value.
// Input columns: [label, series, value1, value2?, calculated?]
// The last numeric column is used as the dataset value (so calculate takes priority).
// Missing label/series combinations are filled with 0.
func (ct *ChartTransformer) transformWithSeriesPivot(tableData TableData, component *Component) (interface{}, error) {
	// Column layout: [label (0), series (1), agg1 (2), agg2? (3), calc? (4)]
	// Value column = last column (index len(headers)-1)
	if len(tableData.Headers) < 3 {
		return ChartData{Labels: []string{}, Datasets: []Dataset{}}, nil
	}

	valueColIdx := len(tableData.Headers) - 1

	// Collect unique labels and series values in order of appearance
	labelOrder := []string{}
	labelSeen := map[string]bool{}
	seriesOrder := []string{}
	seriesSeen := map[string]bool{}

	// Build pivot map: series → label → value
	pivotMap := map[string]map[string]interface{}{}

	for _, row := range tableData.Rows {
		if len(row) < 3 {
			continue
		}

		labelStr := fmt.Sprintf("%v", row[0])
		seriesStr := fmt.Sprintf("%v", row[1])
		value := row[valueColIdx]

		if !labelSeen[labelStr] {
			labelSeen[labelStr] = true
			labelOrder = append(labelOrder, labelStr)
		}
		if !seriesSeen[seriesStr] {
			seriesSeen[seriesStr] = true
			seriesOrder = append(seriesOrder, seriesStr)
		}

		if pivotMap[seriesStr] == nil {
			pivotMap[seriesStr] = map[string]interface{}{}
		}
		pivotMap[seriesStr][labelStr] = value
	}

	if len(seriesOrder) > 15 {
		log.Printf("[WARN] seriesBy produced %d series — chart may be hard to read. Consider adding a WHERE clause or filter to limit categories.", len(seriesOrder))
	}

	// Default color palette
	defaultPalette := []string{
		"#0f766e", "#e74c3c", "#2ecc71", "#d97706", "#6366f1",
		"#14b8a6", "#334155", "#f59e0b", "#8b5cf6", "#0d9488",
	}

	// Build datasets: one per series value
	datasets := make([]Dataset, len(seriesOrder))
	for i, seriesVal := range seriesOrder {
		dataPoints := make([]interface{}, len(labelOrder))
		for j, label := range labelOrder {
			if val, ok := pivotMap[seriesVal][label]; ok {
				dataPoints[j] = val
			} else {
				dataPoints[j] = 0 // Fill missing combinations
			}
		}

		// Use positional color from data.datasets if available, else default palette
		color := defaultPalette[i%len(defaultPalette)]
		if i < len(component.Data.Datasets) && component.Data.Datasets[i].Color != "" {
			color = component.Data.Datasets[i].Color
		}

		datasets[i] = Dataset{
			Label: seriesVal,
			Data:  dataPoints,
			Color: color,
		}
	}

	return ChartData{
		Labels:   labelOrder,
		Datasets: datasets,
	}, nil
}

// TableTransformer implements DataTransformer for table components
type TableTransformer struct{}

// Transform returns the table data, optionally filtered by selectColumns and pivoted
func (tt *TableTransformer) Transform(tableData TableData, component *Component) (interface{}, error) {
	// Ensure non-nil slices so JSON serializes as [] not null
	if tableData.Headers == nil {
		tableData.Headers = []string{}
	}
	if tableData.Rows == nil {
		tableData.Rows = [][]interface{}{}
	}

	// Apply selectColumns filter if specified
	if component.Query != nil && len(component.Query.SelectColumns) > 0 {
		hasGroupBy := len(component.Query.GroupBy) > 0
		if hasGroupBy {
			tableData = filterTableDataWithLabel(tableData, component.Query.SelectColumns)
		} else {
			tableData = filterTableDataBySelectColumns(tableData, component.Query.SelectColumns)
		}
	}

	// Apply transpose transformation if enabled
	if component.Query != nil && component.Query.Transpose != nil && component.Query.Transpose.Enabled {
		rowLabel := component.Query.Transpose.RowLabel
		if rowLabel == "" {
			rowLabel = "Indicator" // Default label
		}

		// If no groupBy, all columns are metrics - need to add a synthetic first column
		// Otherwise the first metric gets used as "period" values instead of being a row
		dataToTranspose := tableData
		if len(component.Query.GroupBy) == 0 && len(tableData.Rows) > 0 {
			// Prepend a "Total" column since there's no period/groupBy dimension
			newHeaders := append([]string{"Period"}, tableData.Headers...)
			newRows := make([][]interface{}, len(tableData.Rows))
			for i, row := range tableData.Rows {
				newRows[i] = append([]interface{}{"Total"}, row...)
			}
			dataToTranspose = TableData{Headers: newHeaders, Rows: newRows}
		}

		transposedData, err := transposeTableData(dataToTranspose, rowLabel)
		if err != nil {
			log.Printf("Transpose transformation failed: %v, returning original data", err)
			return tableData, nil
		}
		return transposedData, nil
	}

	return tableData, nil
}

// transposeTableData swaps rows and columns in table data
// Input: rows with [period, metric1, metric2, metric3, ...]
// Output: rows with [rowLabel, period1, period2, period3, ...] where each metric becomes a row
// Example:
//
//	Input:  period    | Indicator A | Indicator B
//	        Jan 2024  | 150     | 80
//	        Feb 2024  | 180     | 95
//
//	Output: Indicator | Jan 2024 | Feb 2024
//	        Indicator A   | 150      | 180
//	        Indicator B | 80       | 95
func transposeTableData(tableData TableData, rowLabel string) (TableData, error) {
	if len(tableData.Headers) < 2 || len(tableData.Rows) == 0 {
		return tableData, nil
	}

	// First column values become column headers
	// Other column names become row headers (first column)
	numOriginalRows := len(tableData.Rows)
	numOriginalCols := len(tableData.Headers)

	// New headers: [rowLabel, row0_col0_value, row1_col0_value, ...]
	newHeaders := make([]string, numOriginalRows+1)
	newHeaders[0] = rowLabel
	for i, row := range tableData.Rows {
		if len(row) > 0 {
			val, _ := toString(row[0])
			newHeaders[i+1] = val
		}
	}

	// New rows: one row per original column (excluding first column)
	newRows := make([][]interface{}, numOriginalCols-1)
	for colIdx := 1; colIdx < numOriginalCols; colIdx++ {
		newRow := make([]interface{}, numOriginalRows+1)
		newRow[0] = tableData.Headers[colIdx] // Column name becomes row label

		for rowIdx, row := range tableData.Rows {
			if colIdx < len(row) {
				newRow[rowIdx+1] = row[colIdx]
			} else {
				newRow[rowIdx+1] = nil
			}
		}
		newRows[colIdx-1] = newRow
	}

	return TableData{
		Headers: newHeaders,
		Rows:    newRows,
	}, nil
}

// MapTransformer implements DataTransformer for map components
type MapTransformer struct{}

// Transform transforms raw table data into map marker format
func (mt *MapTransformer) Transform(tableData TableData, component *Component) (interface{}, error) {
	// Facility maps return district, facility and value from SQL, then enrich
	// those rows with verified coordinates maintained in a local CSV registry.
	if component.Data.FacilityCoordinatesPath != "" {
		coords, err := loadFacilityCoordinates(component.Data.FacilityCoordinatesPath)
		if err != nil {
			return MapData{}, err
		}
		markers := make([]Marker, 0, len(tableData.Rows))
		unmapped := make([]UnmappedFacility, 0)
		for _, row := range tableData.Rows {
			if len(row) < 3 {
				continue
			}
			district, okDistrict := toString(row[0])
			facility, okFacility := toString(row[1])
			if !okDistrict || !okFacility {
				continue
			}
			coord, ok := coords[strings.ToLower(strings.TrimSpace(district))+"|"+strings.ToLower(strings.TrimSpace(facility))]
			if !ok {
				unmapped = append(unmapped, UnmappedFacility{District: district, Label: facility, Value: row[2]})
				continue
			}
			markers = append(markers, Marker{Lat: coord[0], Lng: coord[1], Label: facility, Value: row[2]})
		}
		return MapData{Center: component.Data.Center, Zoom: component.Data.Zoom, Markers: markers, Unmapped: unmapped, ValueLabel: component.Data.ValueLabel}, nil
	}

	// Initialize markers as empty slice (not nil) to prevent null pointer errors on frontend
	markers := make([]Marker, 0, len(tableData.Rows))
	unmapped := make([]UnmappedFacility, 0)

	for i, row := range tableData.Rows {
		if len(row) < 4 {
			return MapData{}, fmt.Errorf("map data must have at least 4 columns: lat, lng, label, value")
		}

		// Five-column map rows use: lat, lng, district, label, value. This
		// permits a LEFT JOIN to the coordinate registry while retaining totals
		// for facilities whose coordinates are still unavailable.
		if len(row) >= 5 {
			district, _ := toString(row[2])
			label, labelOK := toString(row[3])
			lat, latOK := toFloat64(row[0])
			lng, lngOK := toFloat64(row[1])
			if !labelOK {
				continue
			}
			if !latOK || !lngOK {
				unmapped = append(unmapped, UnmappedFacility{District: district, Label: label, Value: row[4]})
				continue
			}
			markers = append(markers, Marker{Lat: lat, Lng: lng, Label: label, Value: row[4]})
			continue
		}

		lat, ok := toFloat64(row[0])
		if !ok {
			return MapData{}, fmt.Errorf("row %d: latitude must be numeric, got %T", i, row[0])
		}

		lng, ok := toFloat64(row[1])
		if !ok {
			return MapData{}, fmt.Errorf("row %d: longitude must be numeric, got %T", i, row[1])
		}

		label, ok := toString(row[2])
		if !ok {
			return MapData{}, fmt.Errorf("row %d: label must be a string, got %T", i, row[2])
		}

		markers = append(markers, Marker{
			Lat:   lat,
			Lng:   lng,
			Label: label,
			Value: row[3],
		})
	}

	// Preserve center and zoom from the component's YAML configuration.
	return MapData{
		Center:     component.Data.Center,
		Zoom:       component.Data.Zoom,
		Markers:    markers,
		Unmapped:   unmapped,
		ValueLabel: component.Data.ValueLabel,
	}, nil
}

func loadFacilityCoordinates(path string) (map[string][2]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open facility coordinates %q: %w", path, err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read facility coordinates %q: %w", path, err)
	}
	result := make(map[string][2]float64)
	for i, row := range rows {
		if i == 0 || len(row) < 4 {
			continue
		}
		lat, latErr := strconv.ParseFloat(row[2], 64)
		lng, lngErr := strconv.ParseFloat(row[3], 64)
		if latErr != nil || lngErr != nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(row[0])) + "|" + strings.ToLower(strings.TrimSpace(row[1]))
		result[key] = [2]float64{lat, lng}
	}
	return result, nil
}

// ChoroplethTransformer implements DataTransformer for choropleth map components
type ChoroplethTransformer struct{}

// Transform transforms raw table data into district-to-value mapping format
// Supports explicit column mapping for custom SQL queries
func (cet *ChoroplethTransformer) Transform(tableData TableData, component *Component) (interface{}, error) {
	// Initialize district values as empty map (not nil) to prevent null pointer errors on frontend
	districtValues := make(map[string]interface{})

	if len(tableData.Rows) == 0 {
		return ChoroplethData{
			GeoJSONPath:    component.Data.GeoJSONPath,
			DistrictValues: districtValues,
			ColorScheme:    component.Data.ColorScheme,
			BinLabels:      component.Data.BinLabels,
			BinEdges:       component.Data.BinEdges,
			LegendTitle:    component.Data.LegendTitle,
		}, nil
	}

	// Apply selectColumns filter if specified (keeps first column + selected)
	// This allows choropleth maps with aggregations + calculate to display the calculated value
	if component.Query != nil && len(component.Query.SelectColumns) > 0 {
		tableData = filterTableDataWithLabel(tableData, component.Query.SelectColumns)
	}

	// Determine column indices for district and value
	// Default: column 0 is district, column 1 is value
	districtIdx := 0
	valueIdx := 1

	// If column mapping is provided, find indices by column name
	if component.ColumnMapping != nil {
		foundDistrict := false
		foundValue := false

		for i, header := range tableData.Headers {
			if header == component.ColumnMapping.District {
				districtIdx = i
				foundDistrict = true
			}
			if header == component.ColumnMapping.Value {
				valueIdx = i
				foundValue = true
			}
		}

		// Log warnings if mapped columns weren't found
		if !foundDistrict {
			log.Printf("Warning: district column %q not found in headers %v, using column 0", component.ColumnMapping.District, tableData.Headers)
			districtIdx = 0
		}
		if !foundValue {
			log.Printf("Warning: value column %q not found in headers %v, using column 1", component.ColumnMapping.Value, tableData.Headers)
			valueIdx = 1
		}
	}

	// Validate we have enough columns
	minColsNeeded := districtIdx
	if valueIdx > minColsNeeded {
		minColsNeeded = valueIdx
	}

	for i, row := range tableData.Rows {
		if len(row) <= minColsNeeded {
			return ChoroplethData{}, fmt.Errorf("choropleth data row %d has %d columns, need at least %d", i, len(row), minColsNeeded+1)
		}

		district, ok := toString(row[districtIdx])
		if !ok {
			return ChoroplethData{}, fmt.Errorf("row %d: district must be a string, got %T", i, row[districtIdx])
		}

		districtValues[district] = row[valueIdx]
	}

	return ChoroplethData{
		GeoJSONPath:    component.Data.GeoJSONPath,
		DistrictValues: districtValues,
		ColorScheme:    component.Data.ColorScheme,
		BinLabels:      component.Data.BinLabels,
		BinEdges:       component.Data.BinEdges,
		LegendTitle:    component.Data.LegendTitle,
	}, nil
}
