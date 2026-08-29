package spatial

import (
	"context"
	"testing"
)

func TestComputeSpatialIndicatorCount(t *testing.T) {
	engine := NewEngine()

	boundaries := []AdminBoundary{
		{Code: "UG-101", Name: "Kampala District", Level: 2, CentroidX: 32.58, CentroidY: 0.32, AreaSqKm: 189},
		{Code: "UG-201", Name: "Wakiso District", Level: 2, CentroidX: 32.45, CentroidY: 0.40, AreaSqKm: 1907},
		{Code: "UG-301", Name: "Jinja District", Level: 2, CentroidX: 33.19, CentroidY: 0.44, AreaSqKm: 750},
	}

	records := []SpatialRecord{
		{ID: "r1", BoundaryCode: "UG-101", Attributes: map[string]interface{}{"cases": 5.0}},
		{ID: "r2", BoundaryCode: "UG-101", Attributes: map[string]interface{}{"cases": 3.0}},
		{ID: "r3", BoundaryCode: "UG-201", Attributes: map[string]interface{}{"cases": 7.0}},
		{ID: "r4", BoundaryCode: "UG-301", Attributes: map[string]interface{}{"cases": 2.0}},
		{ID: "r5", BoundaryCode: "UG-301", Attributes: map[string]interface{}{"cases": 4.0}},
	}

	req := SpatialIndicatorRequest{
		IndicatorCode: "HEALTH_CASES",
		IndicatorName: "Health Facility Cases",
		AggregationFn: "COUNT",
		Boundaries:    boundaries,
		Records:       records,
	}

	result, err := engine.ComputeSpatialIndicator(context.Background(), req)
	if err != nil {
		t.Fatalf("ComputeSpatialIndicator failed: %v", err)
	}

	if result.TotalRecords != 5 {
		t.Errorf("Expected 5 total records, got %d", result.TotalRecords)
	}
	if result.CoveredBounds != 3 {
		t.Errorf("Expected 3 covered boundaries, got %d", result.CoveredBounds)
	}
	if result.NationalValue != 5.0 {
		t.Errorf("Expected national count of 5, got %.1f", result.NationalValue)
	}

	// First ranked boundary should have 2 records (Kampala or Jinja)
	found := false
	for _, v := range result.BoundaryValues {
		if v.BoundaryCode == "UG-101" && v.Value == 2 {
			found = true
		}
	}
	if !found {
		t.Error("Expected Kampala District to have 2 records")
	}
}

func TestComputeSpatialIndicatorRate(t *testing.T) {
	engine := NewEngine()

	boundaries := []AdminBoundary{
		{Code: "D1", Name: "District Alpha", Level: 2, CentroidX: 32.0, CentroidY: 0.0, AreaSqKm: 500},
		{Code: "D2", Name: "District Beta", Level: 2, CentroidX: 33.0, CentroidY: 1.0, AreaSqKm: 800},
	}

	records := []SpatialRecord{
		{ID: "r1", BoundaryCode: "D1", Attributes: map[string]interface{}{"deaths": 15.0, "population": 50000.0}},
		{ID: "r2", BoundaryCode: "D1", Attributes: map[string]interface{}{"deaths": 10.0, "population": 50000.0}},
		{ID: "r3", BoundaryCode: "D2", Attributes: map[string]interface{}{"deaths": 5.0, "population": 100000.0}},
	}

	req := SpatialIndicatorRequest{
		IndicatorCode: "SDG_3.1.1",
		IndicatorName: "Maternal Mortality Ratio",
		AggregationFn: "RATE",
		Numerator:     "deaths",
		Denominator:   "population",
		Boundaries:    boundaries,
		Records:       records,
	}

	result, err := engine.ComputeSpatialIndicator(context.Background(), req)
	if err != nil {
		t.Fatalf("ComputeSpatialIndicator RATE failed: %v", err)
	}

	// District Alpha: 25 deaths / 100000 pop = 25 per 100,000
	for _, v := range result.BoundaryValues {
		if v.BoundaryCode == "D1" && v.Value != 25 {
			t.Errorf("Expected District Alpha rate = 25 per 100k, got %.2f", v.Value)
		}
	}
}
