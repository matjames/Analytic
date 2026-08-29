package tabulation

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

// AggregationType defines mathematical aggregation functions for crosstabs.
type AggregationType string

const (
	AggCount      AggregationType = "COUNT"
	AggSum        AggregationType = "SUM"
	AggMean       AggregationType = "MEAN"
	AggPercentage AggregationType = "PERCENTAGE"
)

// TabulationRequest represents a dynamic tabulation or crosstab query.
type TabulationRequest struct {
	Title           string                 `json:"title"`
	RowVariable     string                 `json:"row_variable"`
	ColVariable     string                 `json:"col_variable,omitempty"`
	MeasureVariable string                 `json:"measure_variable,omitempty"`
	Aggregation     AggregationType        `json:"aggregation"`
	WeightVariable  string                 `json:"weight_variable,omitempty"`
	DataRecords     []map[string]interface{} `json:"data_records"`
	FilterCriteria  map[string]interface{} `json:"filter_criteria,omitempty"`
}

// CrosstabCell represents a single intersecting matrix cell.
type CrosstabCell struct {
	Count         int     `json:"count"`
	Value         float64 `json:"value"`
	RowPercentage float64 `json:"row_percentage"`
	ColPercentage float64 `json:"col_percentage"`
	TotalPercent  float64 `json:"total_percentage"`
}

// StatisticalTestResult summarizes statistical independence tests (e.g. Chi-Square).
type StatisticalTestResult struct {
	TestName           string  `json:"test_name"`
	StatisticValue     float64 `json:"statistic_value"`
	DegreesOfFreedom   int     `json:"degrees_of_freedom"`
	ApproximatePValue  float64 `json:"approx_p_value"`
	StatisticallyValid bool    `json:"statistically_valid"`
}

// TabulationResult contains computed crosstabs, totals, and statistical test results.
type TabulationResult struct {
	Title            string                           `json:"title"`
	RowVariable      string                           `json:"row_variable"`
	ColVariable      string                           `json:"col_variable,omitempty"`
	RowCategories    []string                         `json:"row_categories"`
	ColCategories    []string                         `json:"col_categories"`
	Matrix           map[string]map[string]CrosstabCell `json:"matrix"`
	RowTotals        map[string]float64               `json:"row_totals"`
	ColTotals        map[string]float64               `json:"col_totals"`
	GrandTotal       float64                          `json:"grand_total"`
	TotalRecordCount int                              `json:"total_record_count"`
	ChiSquareTest    *StatisticalTestResult           `json:"chi_square_test,omitempty"`
	GeneratedAt      time.Time                        `json:"generated_at"`
	SDMXDimension    string                           `json:"sdmx_dimension,omitempty"`
}

// Engine computes dynamic multidimensional crosstabs and official statistics tabulations.
type Engine struct{}

// NewEngine creates a new tabulation engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Tabulate processes data records and generates a complete crosstab with marginals and chi-square.
func (e *Engine) Tabulate(ctx context.Context, req TabulationRequest) (*TabulationResult, error) {
	if req.RowVariable == "" {
		return nil, fmt.Errorf("row_variable is required for tabulation")
	}
	if len(req.DataRecords) == 0 {
		return nil, fmt.Errorf("data_records cannot be empty")
	}
	if req.Aggregation == "" {
		req.Aggregation = AggCount
	}

	rowSet := make(map[string]bool)
	colSet := make(map[string]bool)
	if req.ColVariable == "" {
		req.ColVariable = "Total"
		colSet["Total"] = true
	}

	// 1. Discover unique categories and count/sum values
	cellSums := make(map[string]map[string]float64)
	cellCounts := make(map[string]map[string]int)

	for _, rec := range req.DataRecords {
		rowVal := fmt.Sprintf("%v", rec[req.RowVariable])
		if rowVal == "" || rowVal == "<nil>" {
			rowVal = "Unspecified"
		}
		rowSet[rowVal] = true

		colVal := "Total"
		if req.ColVariable != "Total" {
			colVal = fmt.Sprintf("%v", rec[req.ColVariable])
			if colVal == "" || colVal == "<nil>" {
				colVal = "Unspecified"
			}
		}
		colSet[colVal] = true

		// Determine cell measure
		measure := 1.0
		if req.Aggregation != AggCount && req.MeasureVariable != "" {
			if num, ok := toFloat(rec[req.MeasureVariable]); ok {
				measure = num
			}
		}
		if req.WeightVariable != "" {
			if w, ok := toFloat(rec[req.WeightVariable]); ok && w > 0 {
				measure *= w
			}
		}

		if cellSums[rowVal] == nil {
			cellSums[rowVal] = make(map[string]float64)
			cellCounts[rowVal] = make(map[string]int)
		}
		cellSums[rowVal][colVal] += measure
		cellCounts[rowVal][colVal]++
	}

	// Ordered categories
	rowCategories := make([]string, 0, len(rowSet))
	for r := range rowSet {
		rowCategories = append(rowCategories, r)
	}
	colCategories := make([]string, 0, len(colSet))
	for c := range colSet {
		colCategories = append(colCategories, c)
	}

	// 2. Compute Row Totals, Col Totals, Grand Total
	rowTotals := make(map[string]float64)
	colTotals := make(map[string]float64)
	grandTotal := 0.0

	for _, r := range rowCategories {
		for _, c := range colCategories {
			val := cellSums[r][c]
			if req.Aggregation == AggMean && cellCounts[r][c] > 0 {
				val = val / float64(cellCounts[r][c])
			}
			rowTotals[r] += val
			colTotals[c] += val
			grandTotal += val
		}
	}

	// 3. Build Matrix with Percentages
	matrix := make(map[string]map[string]CrosstabCell)
	for _, r := range rowCategories {
		matrix[r] = make(map[string]CrosstabCell)
		for _, c := range colCategories {
			val := cellSums[r][c]
			cnt := cellCounts[r][c]
			if req.Aggregation == AggMean && cnt > 0 {
				val = val / float64(cnt)
			}

			rowPct := 0.0
			if rowTotals[r] > 0 {
				rowPct = math.Round((val/rowTotals[r])*1000) / 10
			}
			colPct := 0.0
			if colTotals[c] > 0 {
				colPct = math.Round((val/colTotals[c])*1000) / 10
			}
			totPct := 0.0
			if grandTotal > 0 {
				totPct = math.Round((val/grandTotal)*1000) / 10
			}

			matrix[r][c] = CrosstabCell{
				Count:         cnt,
				Value:         math.Round(val*100) / 100,
				RowPercentage: rowPct,
				ColPercentage: colPct,
				TotalPercent:  totPct,
			}
		}
	}

	// 4. Compute Chi-Square Test of Independence (if 2D table >= 2x2)
	var chiTest *StatisticalTestResult
	if len(rowCategories) >= 2 && len(colCategories) >= 2 && req.ColVariable != "Total" && grandTotal > 0 {
		chiStat := 0.0
		for _, r := range rowCategories {
			for _, c := range colCategories {
				observed := cellSums[r][c]
				expected := (rowTotals[r] * colTotals[c]) / grandTotal
				if expected > 0 {
					diff := observed - expected
					chiStat += (diff * diff) / expected
				}
			}
		}
		df := (len(rowCategories) - 1) * (len(colCategories) - 1)
		chiTest = &StatisticalTestResult{
			TestName:           "Pearson Chi-Square Test of Independence",
			StatisticValue:     math.Round(chiStat*1000) / 1000,
			DegreesOfFreedom:   df,
			ApproximatePValue:  approximateChiSquareP(chiStat, df),
			StatisticallyValid: chiStat > 0 && grandTotal >= 30,
		}
	}

	title := req.Title
	if title == "" {
		title = fmt.Sprintf("Tabulation of %s by %s", req.RowVariable, req.ColVariable)
	}

	return &TabulationResult{
		Title:            title,
		RowVariable:      req.RowVariable,
		ColVariable:      req.ColVariable,
		RowCategories:    rowCategories,
		ColCategories:    colCategories,
		Matrix:           matrix,
		RowTotals:        rowTotals,
		ColTotals:        colTotals,
		GrandTotal:       math.Round(grandTotal*100) / 100,
		TotalRecordCount: len(req.DataRecords),
		ChiSquareTest:    chiTest,
		GeneratedAt:      time.Now().UTC(),
		SDMXDimension:    fmt.Sprintf("DIM_%s_%s", req.RowVariable, req.ColVariable),
	}, nil
}

// toFloat converts diverse numeric types to float64.
func toFloat(val interface{}) (float64, bool) {
	if val == nil {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint:
		return float64(v), true
	default:
		return 0, false
	}
}

// approximateChiSquareP calculates a simple upper-tail p-value approximation.
func approximateChiSquareP(chi float64, df int) float64 {
	if chi <= 0 || df <= 0 {
		return 1.0
	}
	// Normal approximation for p-value estimation
	z := (math.Pow(chi/float64(df), 1.0/3.0) - (1.0 - 2.0/(9.0*float64(df)))) / math.Sqrt(2.0/(9.0*float64(df)))
	p := 0.5 * math.Erfc(z/math.Sqrt(2))
	if p < 0.0001 {
		return 0.0001
	}
	return math.Round(p*10000) / 10000
}
// ExportFormat identifies an official-statistics serialization target.
type ExportFormat string

const (
	// ExportFormatSDMX produces SDMX-JSON 1.0 (the NSO interchange standard).
	ExportFormatSDMX ExportFormat = "sdmx"
	// ExportFormatCSV produces a plain-text CSV crosstab.
	ExportFormatCSV ExportFormat = "csv"
	// ExportFormatXLSX is reserved for spreadsheet export (requires a separate
	// spreadsheet dependency and is not bundled to keep the build hermetic).
	ExportFormatXLSX ExportFormat = "xlsx"
)

// sdmxObs is a single SDMX observation within a series.
type sdmxObs struct {
	Obs float64 `json:"0"`
}

// sdmxSeries is the series key → observation map used by SDMX-JSON 1.0.
type sdmxSeries struct {
	Obs []sdmxObs `json:"observations"`
}

// sdmxDimensionValue describes one categorical value of an SDMX dimension.
type sdmxDimensionValue struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// sdmxDimension describes one SDMX dimension with its allowed categories.
type sdmxDimension struct {
	ID     string                `json:"id"`
	Name   string                `json:"name"`
	Values []sdmxDimensionValue  `json:"values"`
}

// sdmxDataSet is a single data-set block in an SDMX-JSON payload.
type sdmxDataSet struct {
	Action string                `json:"action"`
	Series map[string]sdmxSeries `json:"series"`
}

// sdmxStructure carries the dimensional metadata of an SDMX-JSON message.
type sdmxStructure struct {
	Dimensions struct {
		DataSet []sdmxDimension `json:"dataSet"`
		Series  []sdmxDimension `json:"series"`
	} `json:"dimensions"`
}

// sdmxMessage is an SDMX-JSON 1.0 envelope for a tabulation result.
type sdmxMessage struct {
	Header    struct {
		ID        string    `json:"id"`
		Test      bool      `json:"test"`
		Prepared  time.Time `json:"prepared"`
		DataSetID string    `json:"dataSetID"`
	} `json:"header"`
	DataSets   []sdmxDataSet  `json:"dataSets"`
	Structure  sdmxStructure  `json:"structure"`
}

// Export serialises a computed tabulation into the requested format. It returns
// the serialized payload together with the MIME content type so callers can set
// the correct HTTP response headers.
//
// Supported formats: "sdmx" (SDMX-JSON 1.0) and "csv". The "xlsx" format is not
// bundled (requires an optional spreadsheet dependency).
func (e *Engine) Export(result *TabulationResult, format ExportFormat) ([]byte, string, error) {
	switch format {
	case ExportFormatSDMX:
		b, err := e.exportSDMX(result)
		return b, "application/json", err
	case ExportFormatCSV:
		b, err := e.exportCSV(result)
		return b, "text/csv", err
	default:
		return nil, "", fmt.Errorf("unsupported export format: %q", format)
	}
}

// exportSDMX serialises a tabulation result as an SDMX-JSON 1.0 message whose
// dimensions describe the row (dataSet) and column (series) categories and
// whose observations are the matrix cell values.
func (e *Engine) exportSDMX(result *TabulationResult) ([]byte, error) {
	rowDim := sdmxDimensionValue{ID: "Total", Name: "Total"}
	dimValues := make([]sdmxDimensionValue, 0, len(result.RowCategories))
	for _, cat := range result.RowCategories {
		dimValues = append(dimValues, sdmxDimensionValue{ID: cat, Name: cat})
	}

	colDim := make([]sdmxDimensionValue, 0, len(result.ColCategories))
	for _, cat := range result.ColCategories {
		colDim = append(colDim, sdmxDimensionValue{ID: cat, Name: cat})
	}

	series := make(map[string]sdmxSeries, len(result.RowCategories)*len(result.ColCategories))
	for _, rowCat := range result.RowCategories {
		for _, colCat := range result.ColCategories {
			cell := result.Matrix[rowCat][colCat]
			key := rowCat + "." + colCat
			series[key] = sdmxSeries{
				Obs: []sdmxObs{{Obs: cell.Value}},
			}
		}
		// Row marginal as a series entry.
		series[rowCat+".TOTAL"] = sdmxSeries{
			Obs: []sdmxObs{{Obs: result.RowTotals[rowCat]}},
		}
	}

	msg := sdmxMessage{}
	msg.Header.ID = fmt.Sprintf("tab_%d", time.Now().UnixNano())
	msg.Header.DataSetID = fmt.Sprintf("DS_%s", result.RowVariable)
	msg.Header.Prepared = time.Now().UTC()
	msg.Structure.Dimensions.DataSet = []sdmxDimension{{
		ID:     "ROWS",
		Name:   result.RowVariable,
		Values: append(dimValues, rowDim),
	}}
	msg.Structure.Dimensions.Series = []sdmxDimension{{
		ID:     "COLS",
		Name:   result.ColVariable,
		Values: colDim,
	}}
	msg.DataSets = []sdmxDataSet{{
		Action: "Information",
		Series: series,
	}}

	return json.MarshalIndent(msg, "", "  ")
}

// exportCSV serialises the tabulation matrix as a CSV file with marginals.
func (e *Engine) exportCSV(result *TabulationResult) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	header := []string{result.RowVariable}
	for _, colCat := range result.ColCategories {
		header = append(header, colCat)
	}
	header = append(header, "Total")
	if err := w.Write(header); err != nil {
		return nil, err
	}

	sortedRows := make([]string, len(result.RowCategories))
	copy(sortedRows, result.RowCategories)
	sort.Strings(sortedRows)

	for _, rowCat := range sortedRows {
		line := []string{rowCat}
		for _, colCat := range result.ColCategories {
			cell := result.Matrix[rowCat][colCat]
			line = append(line, fmt.Sprintf("%v", cell.Value))
		}
		line = append(line, fmt.Sprintf("%v", result.RowTotals[rowCat]))
		if err := w.Write(line); err != nil {
			return nil, err
		}
	}

	// Column marginals row.
	sortedCols := make([]string, len(result.ColCategories))
	copy(sortedCols, result.ColCategories)
	sort.Strings(sortedCols)
	footer := []string{"Total"}
	for _, colCat := range sortedCols {
		footer = append(footer, fmt.Sprintf("%v", result.ColTotals[colCat]))
	}
	footer = append(footer, fmt.Sprintf("%v", result.GrandTotal))
	if err := w.Write(footer); err != nil {
		return nil, err
	}

	w.Flush()
	return buf.Bytes(), nil
}
