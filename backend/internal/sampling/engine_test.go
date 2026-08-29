package sampling

import (
	"context"
	"testing"
)

func TestCalculateSampleSizeCochran(t *testing.T) {
	engine := NewEngine()

	req := SampleSizeRequest{
		PopulationSize:      100000,
		ConfidenceLevel:     0.95,
		MarginOfError:       0.05,
		EstimatedProportion: 0.5,
		DesignEffect:        1.0,
		NonResponseRate:     0.0,
	}

	result := engine.CalculateSampleSize(req)

	// Standard Cochran sample size for 95% CI, 5% margin of error is ~384-385
	if result.BaseSampleSize < 380 || result.BaseSampleSize > 390 {
		t.Errorf("Expected base sample size around 384, got %d", result.BaseSampleSize)
	}
}

func TestDesignSamplingPlanStratified(t *testing.T) {
	engine := NewEngine()

	req := SamplingPlanRequest{
		SurveyTitle: "National Household Budget Survey 2026",
		Methodology: MethodStratified,
		SampleSizeParams: SampleSizeRequest{
			PopulationSize:      500000,
			ConfidenceLevel:     0.95,
			MarginOfError:       0.03,
			EstimatedProportion: 0.5,
			DesignEffect:        1.5,
			NonResponseRate:     0.10,
		},
		StrataDefinitions: []struct {
			Name           string `json:"name"`
			PopulationSize int    `json:"population_size"`
		}{
			{Name: "Urban Strata", PopulationSize: 200000},
			{Name: "Rural Strata", PopulationSize: 300000},
		},
	}

	plan, err := engine.DesignSamplingPlan(context.Background(), req)
	if err != nil {
		t.Fatalf("DesignSamplingPlan failed: %v", err)
	}

	if len(plan.StratumAllocations) != 2 {
		t.Fatalf("Expected 2 stratum allocations, got %d", len(plan.StratumAllocations))
	}

	urbanAlloc := plan.StratumAllocations[0]
	ruralAlloc := plan.StratumAllocations[1]

	if urbanAlloc.AllocatedSampleSize >= ruralAlloc.AllocatedSampleSize {
		t.Errorf("Expected rural strata (60%% pop) to receive higher sample allocation than urban (40%% pop)")
	}
}
// TestSamplingFrameRegisterAndAllocate verifies the stateful frame registry:
// registration, listing, and reproducible enumeration-area allocation.
func TestSamplingFrameRegisterAndAllocate(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	frame := &MasterSamplingFrame{
		Title:    "National Master Sampling Frame",
		TenantID: "tenant-alpha",
		EnumerationAreas: []string{
			"EA-001", "EA-002", "EA-003", "EA-004", "EA-005",
			"EA-006", "EA-007", "EA-008", "EA-009", "EA-010",
		},
	}
	created, err := e.RegisterFrame(ctx, frame)
	if err != nil {
		t.Fatalf("RegisterFrame: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected an assigned frame ID")
	}
	if created.TotalUnits != 10 {
		t.Errorf("expected TotalUnits 10, got %d", created.TotalUnits)
	}

	// Listing returns the registered frame.
	frames := e.ListFrames(ctx, "tenant-alpha")
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame for tenant-alpha, got %d", len(frames))
	}

	// Allocation is reproducible for a fixed seed.
	a1, err := e.AllocateFrame(ctx, created.ID, 4, 42)
	if err != nil {
		t.Fatalf("AllocateFrame: %v", err)
	}
	if len(a1) != 4 {
		t.Fatalf("expected 4 allocated areas, got %d", len(a1))
	}
	a2, err := e.AllocateFrame(ctx, created.ID, 4, 42)
	if err != nil {
		t.Fatalf("AllocateFrame (repeat): %v", err)
	}
	if len(a1) != len(a2) {
		t.Fatalf("expected reproducible allocation, got %v vs %v", a1, a2)
	}
	for i := range a1 {
		if a1[i] != a2[i] {
			t.Fatalf("allocation not reproducible: %v vs %v", a1, a2)
		}
	}
}

// TestSamplingRegisterFrameValidation ensures invalid frames are rejected.
func TestSamplingRegisterFrameValidation(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	if _, err := e.RegisterFrame(ctx, &MasterSamplingFrame{}); err == nil {
		t.Fatal("expected an error for a frame with no title and no areas")
	}
	if _, err := e.RegisterFrame(ctx, &MasterSamplingFrame{Title: "bad", EnumerationAreas: nil}); err == nil {
		t.Fatal("expected an error for a frame with no enumeration areas")
	}
}

// TestSamplingAllocateMissingFrame ensures allocation on a missing frame errors.
func TestSamplingAllocateMissingFrame(t *testing.T) {
	e := NewEngine()
	if _, err := e.AllocateFrame(context.Background(), "does-not-exist", 3, 1); err == nil {
		t.Fatal("expected an error when allocating from an unknown frame")
	}
}
