package sampling

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// SamplingMethodology defines standard official survey sampling techniques.
type SamplingMethodology string

const (
	MethodSimpleRandom SamplingMethodology = "SIMPLE_RANDOM"
	MethodStratified   SamplingMethodology = "STRATIFIED_RANDOM"
	MethodSystematic   SamplingMethodology = "SYSTEMATIC"
	MethodCluster      SamplingMethodology = "CLUSTER_MULTISTAGE"
)

// SampleSizeRequest encapsulates parameter inputs for sample size calculation.
type SampleSizeRequest struct {
	PopulationSize       int     `json:"population_size"`       // N (0 if infinite/unknown)
	ConfidenceLevel      float64 `json:"confidence_level"`      // e.g. 0.95 (95%), 0.99 (99%)
	MarginOfError        float64 `json:"margin_of_error"`        // e.g. 0.03 (3%), 0.05 (5%)
	EstimatedProportion  float64 `json:"estimated_proportion"`  // e.g. 0.5 (most conservative)
	DesignEffect         float64 `json:"design_effect"`         // DEFF: e.g. 1.5 - 2.0 for cluster surveys
	NonResponseRate      float64 `json:"non_response_rate"`      // e.g. 0.10 (10%)
}

// SampleSizeResult holds the computed target sample sizes.
type SampleSizeResult struct {
	BaseSampleSize       int     `json:"base_sample_size"`
	FinitePopulationAdj  int     `json:"finite_population_adjusted_size"`
	FinalTargetSample    int     `json:"final_target_sample_size"` // Adjusted for DEFF and Non-Response
	ZScore               float64 `json:"z_score"`
	DesignEffectApplied  float64 `json:"design_effect_applied"`
	NonResponseAllowance int     `json:"non_response_allowance"`
	CalculatedAt         time.Time `json:"calculated_at"`
}

// StratumAllocation represents allocation across demographic or geographic strata.
type StratumAllocation struct {
	StratumName        string  `json:"stratum_name"`
	PopulationSize     int     `json:"population_size"`
	ProportionOfTotal  float64 `json:"proportion_of_total"`
	AllocatedSampleSize int    `json:"allocated_sample_size"`
	WeightMultiplier   float64 `json:"weight_multiplier"`
}

// MasterSamplingFrame models an official census or administrative enumeration frame.
type MasterSamplingFrame struct {
	ID                 string              `json:"id"`
	Title              string              `json:"title"`
	TotalPopulation    int                 `json:"total_population"`
	TotalUnits         int                 `json:"total_units"`
	Strata             []StratumAllocation `json:"strata"`
	EnumerationAreas   []string            `json:"enumeration_areas"`
	CreatedAt          time.Time           `json:"created_at"`
	TenantID           string              `json:"tenant_id"`
}

// SamplingPlanRequest configures an end-to-end survey sampling design.
type SamplingPlanRequest struct {
	SurveyTitle      string              `json:"survey_title"`
	Methodology      SamplingMethodology `json:"methodology"`
	SampleSizeParams SampleSizeRequest   `json:"sample_size_params"`
	StrataDefinitions []struct {
		Name           string `json:"name"`
		PopulationSize int    `json:"population_size"`
	} `json:"strata_definitions,omitempty"`
	RandomSeed int64 `json:"random_seed,omitempty"`
}

// SamplingPlanResult summarizes the finalized sampling plan.
type SamplingPlanResult struct {
	PlanID              string                `json:"plan_id"`
	SurveyTitle         string                `json:"survey_title"`
	Methodology         SamplingMethodology   `json:"methodology"`
	SampleSize          SampleSizeResult      `json:"sample_size"`
	StratumAllocations  []StratumAllocation   `json:"stratum_allocations,omitempty"`
	SystematicIntervalK float64               `json:"systematic_interval_k,omitempty"`
	RandomSeed          int64                 `json:"random_seed"`
	DesignedAt          time.Time             `json:"designed_at"`
}

// Engine implements sample size calculations and survey frame allocations.
type Engine struct {
	mu     sync.RWMutex
	frames map[string]*MasterSamplingFrame // framework registry keyed by frame ID
}

// NewEngine creates a new sampling engine.
func NewEngine() *Engine {
	return &Engine{
		frames: make(map[string]*MasterSamplingFrame),
	}
}

// RegisterFrame persists a new master sampling frame. A deterministic ID is
// assigned when the incoming frame has none.
func (e *Engine) RegisterFrame(ctx context.Context, frame *MasterSamplingFrame) (*MasterSamplingFrame, error) {
	if frame.Title == "" {
		return nil, fmt.Errorf("frame title is required")
	}
	if len(frame.EnumerationAreas) == 0 {
		return nil, fmt.Errorf("enumeration_areas cannot be empty")
	}
	if frame.ID == "" {
		frame.ID = fmt.Sprintf("frame-%d", time.Now().UnixNano())
	}
	if frame.TotalUnits == 0 {
		frame.TotalUnits = len(frame.EnumerationAreas)
	}
	if frame.TotalPopulation == 0 {
		// Fall back to unit count when population is unknown.
		frame.TotalPopulation = frame.TotalUnits
	}
	if frame.CreatedAt.IsZero() {
		frame.CreatedAt = time.Now().UTC()
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.frames[frame.ID] = frame
	return frame, nil
}

// GetFrame retrieves a registered sampling frame by ID.
func (e *Engine) GetFrame(ctx context.Context, id string) (*MasterSamplingFrame, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	f, ok := e.frames[id]
	if !ok {
		return nil, fmt.Errorf("sampling frame not found: %s", id)
	}
	return f, nil
}

// ListFrames returns all registered frames for a tenant (or all when tenant is empty).
func (e *Engine) ListFrames(ctx context.Context, tenantID string) []*MasterSamplingFrame {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*MasterSamplingFrame, 0, len(e.frames))
	for _, f := range e.frames {
		if tenantID == "" || f.TenantID == tenantID {
			out = append(out, f)
		}
	}
	return out
}

// AllocateFrame retrieves a stored frame and allocates targetEAs enumeration
// areas from it using a reproducible seed.
func (e *Engine) AllocateFrame(ctx context.Context, id string, targetEAs int, seed int64) ([]string, error) {
	frame, err := e.GetFrame(ctx, id)
	if err != nil {
		return nil, err
	}
	allocated := e.AllocateEnumerationAreas(frame, targetEAs, seed)
	if len(allocated) == 0 {
		return nil, fmt.Errorf("cannot allocate %d enumeration areas from frame %s (has %d)", targetEAs, id, len(frame.EnumerationAreas))
	}
	return allocated, nil
}

// CalculateSampleSize computes sample sizes using Cochran's formula and Finite Population Correction (FPC).
func (e *Engine) CalculateSampleSize(req SampleSizeRequest) SampleSizeResult {
	conf := req.ConfidenceLevel
	if conf <= 0 {
		conf = 0.95
	}
	moe := req.MarginOfError
	if moe <= 0 {
		moe = 0.05
	}
	p := req.EstimatedProportion
	if p <= 0 {
		p = 0.5
	}
	deff := req.DesignEffect
	if deff <= 0 {
		deff = 1.0
	}
	nr := req.NonResponseRate
	if nr < 0 || nr >= 1.0 {
		nr = 0.05
	}

	// 1. Determine Z-score from confidence level
	z := 1.96 // Default for 95%
	switch {
	case conf >= 0.99:
		z = 2.576
	case conf >= 0.95:
		z = 1.960
	case conf >= 0.90:
		z = 1.645
	}

	// 2. Cochran's Base Sample Size: n0 = (Z^2 * p * (1-p)) / e^2
	n0 := (z * z * p * (1.0 - p)) / (moe * moe)
	baseN := int(math.Ceil(n0))

	// 3. Finite Population Correction if N > 0: n_adj = n0 / (1 + (n0 - 1) / N)
	fpcN := baseN
	if req.PopulationSize > 0 && req.PopulationSize > baseN {
		fpc := float64(baseN) / (1.0 + float64(baseN-1)/float64(req.PopulationSize))
		fpcN = int(math.Ceil(fpc))
	}

	// 4. Apply Design Effect (DEFF) and Non-Response Allowance
	withDEFF := float64(fpcN) * deff
	finalN := int(math.Ceil(withDEFF / (1.0 - nr)))
	allowance := finalN - fpcN

	return SampleSizeResult{
		BaseSampleSize:       baseN,
		FinitePopulationAdj:  fpcN,
		FinalTargetSample:    finalN,
		ZScore:               z,
		DesignEffectApplied:  deff,
		NonResponseAllowance: allowance,
		CalculatedAt:         time.Now().UTC(),
	}
}

// DesignSamplingPlan designs a comprehensive sampling plan with stratum allocations.
func (e *Engine) DesignSamplingPlan(ctx context.Context, req SamplingPlanRequest) (*SamplingPlanResult, error) {
	if req.SurveyTitle == "" {
		return nil, fmt.Errorf("survey_title is required")
	}
	if req.Methodology == "" {
		req.Methodology = MethodStratified
	}

	seed := req.RandomSeed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	// 1. Calculate overall sample size
	sampleSize := e.CalculateSampleSize(req.SampleSizeParams)

	// 2. Compute Proportional Allocation across Strata if defined
	strataAllocations := make([]StratumAllocation, 0)
	totalStrataPop := 0
	for _, s := range req.StrataDefinitions {
		totalStrataPop += s.PopulationSize
	}

	if totalStrataPop > 0 && len(req.StrataDefinitions) > 0 {
		for _, s := range req.StrataDefinitions {
			prop := float64(s.PopulationSize) / float64(totalStrataPop)
			allocated := int(math.Ceil(float64(sampleSize.FinalTargetSample) * prop))
			weight := 1.0
			if allocated > 0 {
				weight = math.Round((float64(s.PopulationSize)/float64(allocated))*100) / 100
			}
			strataAllocations = append(strataAllocations, StratumAllocation{
				StratumName:         s.Name,
				PopulationSize:      s.PopulationSize,
				ProportionOfTotal:   math.Round(prop*1000) / 10,
				AllocatedSampleSize: allocated,
				WeightMultiplier:    weight,
			})
		}
	}

	// 3. Compute Systematic Sampling Interval K
	intervalK := 0.0
	if req.SampleSizeParams.PopulationSize > 0 && sampleSize.FinalTargetSample > 0 {
		intervalK = math.Round((float64(req.SampleSizeParams.PopulationSize)/float64(sampleSize.FinalTargetSample))*10) / 10
	}

	planID := fmt.Sprintf("plan-smp-%d", time.Now().Unix()%1000000)

	return &SamplingPlanResult{
		PlanID:              planID,
		SurveyTitle:         req.SurveyTitle,
		Methodology:         req.Methodology,
		SampleSize:          sampleSize,
		StratumAllocations:  strataAllocations,
		SystematicIntervalK: intervalK,
		RandomSeed:          seed,
		DesignedAt:          time.Now().UTC(),
	}, nil
}

// AllocateEnumerationAreas draws random or systematic EAs from a Master Sampling Frame.
func (e *Engine) AllocateEnumerationAreas(frame *MasterSamplingFrame, targetEAs int, seed int64) []string {
	if len(frame.EnumerationAreas) == 0 || targetEAs <= 0 {
		return nil
	}
	if targetEAs >= len(frame.EnumerationAreas) {
		return frame.EnumerationAreas
	}

	r := rand.New(rand.NewSource(seed))
	shuffled := make([]string, len(frame.EnumerationAreas))
	copy(shuffled, frame.EnumerationAreas)
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:targetEAs]
}
