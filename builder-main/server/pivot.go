package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// applyPivot transforms a 3-column SQL result (columns, rows, values) into a
// cross-tab pivot table.  Column order in the output mirrors the order in which
// distinct column-axis values are first encountered in the SQL rows, so the
// author controls ordering via the ORDER BY clause of their query.
//
// The function returns a new TableData; the original is not modified.
func applyPivot(data TableData, cfg *PivotConfig) (TableData, error) {
	if cfg == nil {
		return data, nil
	}

	// Locate the three required column indices by name
	colIdx, rowIdx, valIdx := -1, -1, -1
	for i, h := range data.Headers {
		switch h {
		case cfg.Columns:
			colIdx = i
		case cfg.Rows:
			rowIdx = i
		case cfg.Values:
			valIdx = i
		}
	}

	if colIdx < 0 {
		return data, fmt.Errorf("pivot: column axis '%s' not found in SQL result (got: %v)", cfg.Columns, data.Headers)
	}
	if rowIdx < 0 {
		return data, fmt.Errorf("pivot: row axis '%s' not found in SQL result (got: %v)", cfg.Rows, data.Headers)
	}
	if valIdx < 0 {
		return data, fmt.Errorf("pivot: value column '%s' not found in SQL result (got: %v)", cfg.Values, data.Headers)
	}

	// Collect distinct column and row labels preserving SQL encounter order
	var colOrder []string
	colSeen := map[string]bool{}
	var rowOrder []string
	rowSeen := map[string]bool{}

	// cellMap: rowLabel -> colLabel -> raw value (nil = blank)
	cellMap := map[string]map[string]interface{}{}

	for _, row := range data.Rows {
		colVal := cellString(row[colIdx])
		rowVal := cellString(row[rowIdx])
		val := row[valIdx]

		if !colSeen[colVal] {
			colSeen[colVal] = true
			colOrder = append(colOrder, colVal)
		}
		if !rowSeen[rowVal] {
			rowSeen[rowVal] = true
			rowOrder = append(rowOrder, rowVal)
			cellMap[rowVal] = map[string]interface{}{}
		}
		cellMap[rowVal][colVal] = val
	}

	// Build output headers: [row-axis name, col1, col2, ..., extraCol1, ...]
	headers := []string{cfg.Rows}
	headers = append(headers, colOrder...)
	for _, ec := range cfg.ExtraColumns {
		headers = append(headers, ec.Label)
	}

	// Build pivot rows
	var pivotRows [][]interface{}
	for _, rowLabel := range rowOrder {
		row := []interface{}{rowLabel}
		dataVals := make([]float64, len(colOrder)) // float64 slice for formula evaluation

		for ci, col := range colOrder {
			raw := cellMap[rowLabel][col] // nil if missing → blank
			if raw == nil {
				row = append(row, "")
				// dataVals[ci] stays 0
			} else {
				row = append(row, formatPivotCell(raw, cfg.Format))
				if f, ok := toFloat64(raw); ok {
					dataVals[ci] = f
				}
			}
		}

		// Append extra columns
		for _, ec := range cfg.ExtraColumns {
			computed := evalPivotColFormula(ec.Formula, dataVals)
			if computed == nil {
				row = append(row, "")
			} else {
				row = append(row, formatPivotCell(computed, cfg.Format))
			}
		}

		pivotRows = append(pivotRows, row)
	}

	// Build extra rows (aggregate across data rows per column)
	// Capture length so extra rows only sum over the original data rows, not each other.
	dataRowCount := len(pivotRows)
	for _, er := range cfg.ExtraRows {
		row := []interface{}{er.Label}

		// For each data column, collect all numeric values from data rows only
		extraRowDataVals := make([]float64, len(colOrder))
		for ci := range colOrder {
			var colNums []float64
			for ri := 0; ri < dataRowCount; ri++ {
				pr := pivotRows[ri]
				// pr[0] = row label, pr[1..len(colOrder)] = data cols
				raw := pr[ci+1]
				if f, ok := toFloat64(raw); ok {
					colNums = append(colNums, f)
				}
			}
			extraRowDataVals[ci] = evalPivotRowAgg(er.Formula, colNums)
			row = append(row, formatPivotCell(extraRowDataVals[ci], cfg.Format))
		}

		// Extra columns for this extra row
		for _, ec := range cfg.ExtraColumns {
			computed := evalPivotColFormula(ec.Formula, extraRowDataVals)
			if computed == nil {
				row = append(row, "")
			} else {
				row = append(row, formatPivotCell(computed, cfg.Format))
			}
		}

		pivotRows = append(pivotRows, row)
	}

	return TableData{
		Headers: headers,
		Rows:    pivotRows,
	}, nil
}

// cellString converts an interface cell value to a string for use as a map key.
func cellString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// formatPivotCell applies a format string to a cell value.
// format: "" or "auto" → as-is; "integer" → rounded int; "decimal:N" → N decimal places;
// "percent:N" → N decimal places with % suffix.
func formatPivotCell(v interface{}, format string) interface{} {
	if v == nil {
		return ""
	}
	if format == "" || format == "auto" {
		return v
	}

	f, ok := toFloat64(v)
	if !ok {
		return v // non-numeric: return as-is
	}

	switch {
	case format == "integer":
		return strconv.FormatInt(int64(math.Round(f)), 10)
	case strings.HasPrefix(format, "decimal:"):
		n, err := strconv.Atoi(strings.TrimPrefix(format, "decimal:"))
		if err != nil || n < 0 {
			return v
		}
		return strconv.FormatFloat(f, 'f', n, 64)
	case strings.HasPrefix(format, "percent:"):
		n, err := strconv.Atoi(strings.TrimPrefix(format, "percent:"))
		if err != nil || n < 0 {
			return v
		}
		return strconv.FormatFloat(f, 'f', n, 64) + "%"
	}
	return v
}

// evalPivotColFormula evaluates a per-row extra-column formula.
// vals contains the numeric values of the data columns in order.
// Returns nil on error / blank result.
func evalPivotColFormula(formula string, vals []float64) interface{} {
	formula = strings.TrimSpace(formula)
	switch formula {
	case "sum":
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s
	case "avg":
		if len(vals) == 0 {
			return nil
		}
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s / float64(len(vals))
	default:
		// Replace [-N] references with actual float values
		result, err := evalPositionalExpr(formula, vals)
		if err != nil {
			return nil
		}
		return result
	}
}

// evalPivotRowAgg evaluates an extra-row aggregation formula against a slice of
// column values collected from all data rows.
func evalPivotRowAgg(formula string, vals []float64) float64 {
	formula = strings.TrimSpace(formula)
	switch formula {
	case "avg":
		if len(vals) == 0 {
			return 0
		}
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s / float64(len(vals))
	default: // "sum" and anything else → sum
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s
	}
}

// evalPositionalExpr evaluates an arithmetic expression that may contain [-N]
// positional references into vals, where [-1] is the last element, [-2] the
// second-to-last, etc.  Supports +, -, *, / and parentheses.
func evalPositionalExpr(expr string, vals []float64) (float64, error) {
	// Replace [-N] tokens with their numeric values
	re := regexp.MustCompile(`\[-(\d+)\]`)
	resolved := re.ReplaceAllStringFunc(expr, func(m string) string {
		sub := re.FindStringSubmatch(m)
		n, err := strconv.Atoi(sub[1])
		if err != nil || n <= 0 || n > len(vals) {
			return "0"
		}
		idx := len(vals) - n
		return strconv.FormatFloat(vals[idx], 'f', -1, 64)
	})

	return evalArithExpr(resolved)
}

// evalArithExpr evaluates a simple arithmetic expression string supporting
// +, -, *, / and parentheses using recursive-descent parsing.
func evalArithExpr(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)
	p := &arithParser{input: expr, pos: 0}
	val, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	if p.pos < len(p.input) {
		// leftover characters
		return 0, fmt.Errorf("unexpected character at pos %d: %s", p.pos, p.input[p.pos:])
	}
	return val, nil
}

type arithParser struct {
	input string
	pos   int
}

func (p *arithParser) skipWS() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t') {
		p.pos++
	}
}

func (p *arithParser) parseExpr() (float64, error) {
	return p.parseAddSub()
}

func (p *arithParser) parseAddSub() (float64, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return 0, err
	}
	for {
		p.skipWS()
		if p.pos >= len(p.input) {
			break
		}
		op := p.input[p.pos]
		if op != '+' && op != '-' {
			break
		}
		p.pos++
		right, err := p.parseMulDiv()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

func (p *arithParser) parseMulDiv() (float64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	for {
		p.skipWS()
		if p.pos >= len(p.input) {
			break
		}
		op := p.input[p.pos]
		if op != '*' && op != '/' {
			break
		}
		p.pos++
		right, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		if op == '*' {
			left *= right
		} else {
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
		}
	}
	return left, nil
}

func (p *arithParser) parseUnary() (float64, error) {
	p.skipWS()
	if p.pos < len(p.input) && p.input[p.pos] == '-' {
		p.pos++
		v, err := p.parsePrimary()
		return -v, err
	}
	return p.parsePrimary()
}

func (p *arithParser) parsePrimary() (float64, error) {
	p.skipWS()
	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("unexpected end of expression")
	}
	if p.input[p.pos] == '(' {
		p.pos++ // consume '('
		val, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skipWS()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++ // consume ')'
		return val, nil
	}
	// Parse number (integer or float, optional leading sign already handled)
	start := p.pos
	if p.pos < len(p.input) && (p.input[p.pos] == '+') {
		p.pos++
	}
	for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9' || p.input[p.pos] == '.') {
		p.pos++
	}
	if p.pos == start {
		return 0, fmt.Errorf("expected number at pos %d: %s", p.pos, p.input[p.pos:])
	}
	return strconv.ParseFloat(strings.TrimSpace(p.input[start:p.pos]), 64)
}
