package spatial

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// AdminBoundary represents a geographic administrative region (district, region, county).
type AdminBoundary struct {
	Code       string  `json:"code"`         // e.g. "UG-102" (district code)
	Name       string  `json:"name"`         // e.g. "Kampala District"
	Level      int     `json:"admin_level"`  // 1=region, 2=district, 3=sub-county
	CentroidX  float64 `json:"centroid_x"`   // Longitude
	CentroidY  float64 `json:"centroid_y"`   // Latitude
	AreaSqKm   float64 `json:"area_sq_km"`
}

// SpatialRecord represents a geo-referenced data observation (e.g. survey submission, sensor reading).
type SpatialRecord struct {
	ID           string                 `json:"id"`
	Longitude    float64                `json:"longitude"`
	Latitude     float64                `json:"latitude"`
	BoundaryCode string                `json:"boundary_code,omitempty"`
	Attributes   map[string]interface{} `json:"attributes"`
	Timestamp    time.Time              `json:"timestamp"`
}

// SpatialIndicatorRequest configures a spatially disaggregated indicator computation.
type SpatialIndicatorRequest struct {
	IndicatorCode  string           `json:"indicator_code"`   // e.g. "SDG_3.1.1"
	IndicatorName  string           `json:"indicator_name"`   // Human label
	MeasureField   string           `json:"measure_field"`    // Field in Attributes to aggregate
	AggregationFn  string           `json:"aggregation_fn"`   // "COUNT", "SUM", "MEAN", "RATE"
	Numerator      string           `json:"numerator,omitempty"`   // For RATE: numerator field
	Denominator    string           `json:"denominator,omitempty"` // For RATE: denominator field
	Boundaries     []AdminBoundary  `json:"boundaries"`
	Records        []SpatialRecord  `json:"records"`
	BaseYear       int              `json:"base_year,omitempty"`
}

// SpatialIndicatorValue holds the computed indicator for one admin boundary.
type SpatialIndicatorValue struct {
	BoundaryCode string  `json:"boundary_code"`
	BoundaryName string  `json:"boundary_name"`
	AdminLevel   int     `json:"admin_level"`
	Value        float64 `json:"value"`
	RecordCount  int     `json:"record_count"`
	CentroidX    float64 `json:"centroid_x"`
	CentroidY    float64 `json:"centroid_y"`
	AreaSqKm     float64 `json:"area_sq_km"`
	Rank         int     `json:"rank"`
}

// SpatialIndicatorResult is the full spatially disaggregated result.
type SpatialIndicatorResult struct {
	IndicatorCode  string                  `json:"indicator_code"`
	IndicatorName  string                  `json:"indicator_name"`
	AggregationFn  string                  `json:"aggregation_fn"`
	NationalValue  float64                 `json:"national_value"`
	BoundaryValues []SpatialIndicatorValue `json:"boundary_values"`
	TotalRecords   int                     `json:"total_records"`
	CoveredBounds  int                     `json:"covered_boundaries"`
	TotalBounds    int                     `json:"total_boundaries"`
	SDMXDimension  string                  `json:"sdmx_spatial_dimension"`
	ComputedAt     time.Time               `json:"computed_at"`
}

// Engine computes spatially disaggregated indicators across administrative boundaries.
type Engine struct{}

// NewEngine creates a new spatial indicator computation engine.
func NewEngine() *Engine {
	return &Engine{}
}

// ComputeSpatialIndicator assigns records to boundaries and aggregates per admin unit.
func (e *Engine) ComputeSpatialIndicator(ctx context.Context, req SpatialIndicatorRequest) (*SpatialIndicatorResult, error) {
	if req.IndicatorCode == "" {
		return nil, fmt.Errorf("indicator_code is required")
	}
	if len(req.Boundaries) == 0 {
		return nil, fmt.Errorf("boundaries list is required")
	}
	if req.AggregationFn == "" {
		req.AggregationFn = "COUNT"
	}

	// Build boundary lookup
	boundaryMap := make(map[string]*AdminBoundary)
	for i := range req.Boundaries {
		boundaryMap[req.Boundaries[i].Code] = &req.Boundaries[i]
	}

	// Assign each record to a boundary
	type cellData struct {
		count     int
		sum       float64
		numerator float64
		denominator float64
	}
	cells := make(map[string]*cellData)

	for _, rec := range req.Records {
		code := rec.BoundaryCode
		if code == "" {
			// Point-in-polygon approximation: assign to nearest boundary centroid
			code = nearestBoundary(rec.Longitude, rec.Latitude, req.Boundaries)
		}
		if _, exists := boundaryMap[code]; !exists {
			continue
		}
		if cells[code] == nil {
			cells[code] = &cellData{}
		}
		cells[code].count++

		if req.MeasureField != "" {
			if v, ok := toFloat(rec.Attributes[req.MeasureField]); ok {
				cells[code].sum += v
			}
		}
		if req.Numerator != "" {
			if v, ok := toFloat(rec.Attributes[req.Numerator]); ok {
				cells[code].numerator += v
			}
		}
		if req.Denominator != "" {
			if v, ok := toFloat(rec.Attributes[req.Denominator]); ok {
				cells[code].denominator += v
			}
		}
	}

	// Compute aggregated values
	values := make([]SpatialIndicatorValue, 0, len(req.Boundaries))
	nationalSum := 0.0
	nationalCount := 0
	nationalNumerator := 0.0
	nationalDenominator := 0.0

	for _, b := range req.Boundaries {
		c := cells[b.Code]
		if c == nil {
			c = &cellData{}
		}

		var val float64
		switch req.AggregationFn {
		case "COUNT":
			val = float64(c.count)
		case "SUM":
			val = c.sum
		case "MEAN":
			if c.count > 0 {
				val = c.sum / float64(c.count)
			}
		case "RATE":
			if c.denominator > 0 {
				val = (c.numerator / c.denominator) * 100000 // per 100,000
			}
		default:
			val = float64(c.count)
		}

		val = math.Round(val*100) / 100
		nationalSum += c.sum
		nationalCount += c.count
		nationalNumerator += c.numerator
		nationalDenominator += c.denominator

		values = append(values, SpatialIndicatorValue{
			BoundaryCode: b.Code,
			BoundaryName: b.Name,
			AdminLevel:   b.Level,
			Value:        val,
			RecordCount:  c.count,
			CentroidX:    b.CentroidX,
			CentroidY:    b.CentroidY,
			AreaSqKm:     b.AreaSqKm,
		})
	}

	// Rank boundaries by value (descending)
	sort.Slice(values, func(i, j int) bool {
		return values[i].Value > values[j].Value
	})
	for i := range values {
		values[i].Rank = i + 1
	}

	// Compute national value
	natVal := 0.0
	switch req.AggregationFn {
	case "COUNT":
		natVal = float64(nationalCount)
	case "SUM":
		natVal = nationalSum
	case "MEAN":
		if nationalCount > 0 {
			natVal = nationalSum / float64(nationalCount)
		}
	case "RATE":
		if nationalDenominator > 0 {
			natVal = (nationalNumerator / nationalDenominator) * 100000
		}
	}
	natVal = math.Round(natVal*100) / 100

	coveredBounds := 0
	for _, v := range values {
		if v.RecordCount > 0 {
			coveredBounds++
		}
	}

	return &SpatialIndicatorResult{
		IndicatorCode:  req.IndicatorCode,
		IndicatorName:  req.IndicatorName,
		AggregationFn:  req.AggregationFn,
		NationalValue:  natVal,
		BoundaryValues: values,
		TotalRecords:   len(req.Records),
		CoveredBounds:  coveredBounds,
		TotalBounds:    len(req.Boundaries),
		SDMXDimension:  fmt.Sprintf("REF_AREA_%s", req.IndicatorCode),
		ComputedAt:     time.Now().UTC(),
	}, nil
}

// nearestBoundary finds the closest admin boundary centroid to a given point.
func nearestBoundary(lon, lat float64, boundaries []AdminBoundary) string {
	minDist := math.MaxFloat64
	closest := ""
	for _, b := range boundaries {
		dx := lon - b.CentroidX
		dy := lat - b.CentroidY
		dist := dx*dx + dy*dy
		if dist < minDist {
			minDist = dist
			closest = b.Code
		}
	}
	return closest
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
