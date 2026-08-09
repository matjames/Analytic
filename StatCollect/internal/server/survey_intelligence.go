package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Automatic Survey Intelligence Engine
//
// Per the StatGate Architectural Directive, the platform should automatically
// understand the structure of every survey and generate meaningful analytics
// without requiring developers to build custom dashboards for each project.
//
// This engine:
//   - Reads the template schema from the database
//   - Classifies all fields by type (numeric, categorical, date, GPS, etc.)
//   - Computes descriptive stats, distributions, missing values, outlier detection
//   - Surfaces demographic breakdowns (gender, age) when applicable fields are found
//   - Generates a 90-day submission timeline
// ─────────────────────────────────────────────────────────────────────────────

// FieldType constants for automatic field classification
const (
	FieldTypeNumeric     = "numeric"
	FieldTypeCategorical = "categorical"
	FieldTypeDate        = "date"
	FieldTypeGPS         = "gps"
	FieldTypeImage       = "image"
	FieldTypeSignature   = "signature"
	FieldTypeText        = "text"
	FieldTypeRepeat      = "repeat_group"
	FieldTypeCalculated  = "calculated"
	FieldTypeNote        = "note"
	FieldTypeBoolean     = "boolean"
)

// FieldIntelligence describes a single survey field with its auto-detected type
// and computed statistics.
type FieldIntelligence struct {
	Key          string              `json:"key"`
	Label        string              `json:"label"`
	Type         string              `json:"type"`          // original schema type
	IntelType    string              `json:"intel_type"`    // classified type: numeric, categorical, etc.
	Required     bool                `json:"required"`
	TotalCount   int                 `json:"total_count"`
	FilledCount  int                 `json:"filled_count"`
	MissingCount int                 `json:"missing_count"`
	MissingPct   float64             `json:"missing_pct"`
	// Numeric fields
	DescStats    *DescriptiveStats   `json:"descriptive_stats,omitempty"`
	Outliers     []OutlierRecord     `json:"outliers,omitempty"`
	// Categorical fields
	Distribution map[string]int      `json:"distribution,omitempty"`
	TopValues    []ValueFrequency    `json:"top_values,omitempty"`
}

// DescriptiveStats holds computed statistics for a numeric field.
type DescriptiveStats struct {
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Mean      float64 `json:"mean"`
	Median    float64 `json:"median"`
	StdDev    float64 `json:"std_dev"`
	P25       float64 `json:"p25"`
	P75       float64 `json:"p75"`
	IQR       float64 `json:"iqr"`
	Count     int     `json:"count"`
}

// OutlierRecord flags a submission with an outlier value for a specific field.
type OutlierRecord struct {
	InstanceID string  `json:"instance_id"`
	Value      float64 `json:"value"`
	ZScore     float64 `json:"z_score"`
	Severity   string  `json:"severity"` // moderate, extreme
}

// ValueFrequency holds a value and its count for categorical distributions.
type ValueFrequency struct {
	Value string `json:"value"`
	Count int    `json:"count"`
	Pct   float64 `json:"pct"`
}

// DemographicBreakdown holds gender or age group distribution.
type DemographicBreakdown struct {
	FieldKey string           `json:"field_key"`
	FieldLabel string         `json:"field_label"`
	Buckets  []ValueFrequency `json:"buckets"`
}

// SubmissionTimelinePoint is a single day's submission count.
type SubmissionTimelinePoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// SurveyIntelligence is the full auto-generated intelligence report for a survey.
type SurveyIntelligence struct {
	FormID            string                  `json:"form_id"`
	TenantID          string                  `json:"tenant_id"`
	GeneratedAt       time.Time               `json:"generated_at"`
	TotalSubmissions  int                     `json:"total_submissions"`
	ApprovedCount     int                     `json:"approved_count"`
	RejectedCount     int                     `json:"rejected_count"`
	PendingCount      int                     `json:"pending_count"`
	ApprovalRate      float64                 `json:"approval_rate"`
	// Field-level intelligence
	Fields            []FieldIntelligence     `json:"fields"`
	NumericFields     []string                `json:"numeric_fields"`
	CategoricalFields []string                `json:"categorical_fields"`
	GPSFields         []string                `json:"gps_fields"`
	DateFields        []string                `json:"date_fields"`
	ImageFields       []string                `json:"image_fields"`
	// Demographics
	GenderBreakdown   *DemographicBreakdown   `json:"gender_breakdown,omitempty"`
	AgeDistribution   *DemographicBreakdown   `json:"age_distribution,omitempty"`
	// Geo coverage
	GPSCoveragePct    float64                 `json:"gps_coverage_pct"`
	RegionCount       int                     `json:"region_count"`
	DistrictCount     int                     `json:"district_count"`
	// Timeline
	Timeline          []SubmissionTimelinePoint `json:"timeline"`
	// Quality
	OverallMissingPct float64                 `json:"overall_missing_pct"`
	OutlierCount      int                     `json:"outlier_count"`
	HighRiskFields    []string                `json:"high_risk_fields"` // fields with >30% missing
	// Object relationships
	Projects          []string                `json:"projects"`
	Organizations     []string                `json:"organizations"`
	Regions           []string                `json:"regions"`
	Districts         []string                `json:"districts"`
}

// ─────────────────────────────────────────────────────────────────────────────
// classifyFieldType maps a schema field type to an intel type.
// ─────────────────────────────────────────────────────────────────────────────

func classifyFieldType(schemaType, fieldKey string) string {
	key := strings.ToLower(fieldKey)
	t := strings.ToLower(schemaType)

	// GPS: explicit type or key hints
	if t == "gps" || t == "geolocation" || t == "geo" ||
		strings.Contains(key, "gps") || strings.Contains(key, "lat") || strings.Contains(key, "lon") {
		return FieldTypeGPS
	}
	// Image/photo
	if t == "image" || t == "photo" || t == "camera" || strings.Contains(key, "photo") || strings.Contains(key, "image") {
		return FieldTypeImage
	}
	// Signature
	if t == "signature" || strings.Contains(key, "signature") {
		return FieldTypeSignature
	}
	// Date/time
	if t == "date" || t == "datetime" || t == "time" || strings.Contains(key, "_date") || strings.Contains(key, "date_") {
		return FieldTypeDate
	}
	// Numeric
	if t == "integer" || t == "decimal" || t == "number" || t == "float" || t == "range" || t == "rating" {
		return FieldTypeNumeric
	}
	// Categorical / select
	if t == "select" || t == "select_one" || t == "select_multiple" || t == "radio" ||
		t == "checkbox" || t == "dropdown" || t == "multi_select" || t == "boolean" || t == "yes_no" {
		return FieldTypeCategorical
	}
	// Calculated/note
	if t == "calculate" || t == "calculated" {
		return FieldTypeCalculated
	}
	if t == "note" || t == "text_only" {
		return FieldTypeNote
	}
	// Repeat group
	if t == "repeat" || t == "repeat_group" || t == "group" {
		return FieldTypeRepeat
	}
	// Numeric by key name heuristic
	numericHints := []string{"age", "number", "count", "total", "amount", "score", "quantity", "weight", "height", "income", "distance", "duration"}
	for _, hint := range numericHints {
		if strings.Contains(key, hint) {
			return FieldTypeNumeric
		}
	}
	// Gender/categorical heuristic
	if strings.Contains(key, "gender") || strings.Contains(key, "sex") ||
		strings.Contains(key, "education") || strings.Contains(key, "occupation") ||
		strings.Contains(key, "category") || strings.Contains(key, "type") ||
		strings.Contains(key, "status") || strings.Contains(key, "marital") {
		return FieldTypeCategorical
	}
	// Default to text
	return FieldTypeText
}

// ─────────────────────────────────────────────────────────────────────────────
// GenerateSurveyIntelligence computes the full auto-intelligence for a survey.
// ─────────────────────────────────────────────────────────────────────────────

func GenerateSurveyIntelligence(ctx context.Context, formID, tenantID string) (*SurveyIntelligence, error) {
	si := &SurveyIntelligence{
		FormID:      formID,
		TenantID:    tenantID,
		GeneratedAt: time.Now(),
		Fields:      []FieldIntelligence{},
		Timeline:    []SubmissionTimelinePoint{},
		Projects:    []string{},
		Organizations: []string{},
		Regions:     []string{},
		Districts:   []string{},
		NumericFields:     []string{},
		CategoricalFields: []string{},
		GPSFields:         []string{},
		DateFields:        []string{},
		ImageFields:       []string{},
		HighRiskFields:    []string{},
	}

	if dbPool == nil {
		return si, nil
	}

	// ── 1. Basic submission counts ──────────────────────────────────────────
	_ = dbPool.QueryRow(ctx, `SELECT
		COUNT(1),
		COUNT(1) FILTER (WHERE status='approved'),
		COUNT(1) FILTER (WHERE status='rejected'),
		COUNT(1) FILTER (WHERE status NOT IN ('approved','rejected'))
		FROM submissions WHERE form_id=$1 AND tenant_id=$2`, formID, tenantID).Scan(
		&si.TotalSubmissions, &si.ApprovedCount, &si.RejectedCount, &si.PendingCount)

	if si.TotalSubmissions > 0 {
		si.ApprovalRate = math.Round((float64(si.ApprovedCount)/float64(si.TotalSubmissions))*100*10) / 10
	}

	// ── 2. Object relationships from MSH metadata ───────────────────────────
	rows, err := dbPool.Query(ctx, `SELECT DISTINCT COALESCE(meta->>'_msh_project','') FROM submissions WHERE form_id=$1 AND tenant_id=$2 AND meta->>'_msh_project' IS NOT NULL LIMIT 20`, formID, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var v string
			if rows.Scan(&v) == nil && v != "" {
				si.Projects = append(si.Projects, v)
			}
		}
	}

	rows2, err := dbPool.Query(ctx, `SELECT DISTINCT COALESCE(meta->>'_msh_organization','') FROM submissions WHERE form_id=$1 AND tenant_id=$2 AND meta->>'_msh_organization' IS NOT NULL LIMIT 20`, formID, tenantID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var v string
			if rows2.Scan(&v) == nil && v != "" {
				si.Organizations = append(si.Organizations, v)
			}
		}
	}

	rows3, err := dbPool.Query(ctx, `SELECT DISTINCT COALESCE(meta->>'_msh_region','') FROM submissions WHERE form_id=$1 AND tenant_id=$2 AND meta->>'_msh_region' IS NOT NULL LIMIT 20`, formID, tenantID)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var v string
			if rows3.Scan(&v) == nil && v != "" {
				si.Regions = append(si.Regions, v)
			}
		}
	}
	si.RegionCount = len(si.Regions)

	rows4, err := dbPool.Query(ctx, `SELECT DISTINCT COALESCE(meta->>'_msh_district','') FROM submissions WHERE form_id=$1 AND tenant_id=$2 AND meta->>'_msh_district' IS NOT NULL LIMIT 50`, formID, tenantID)
	if err == nil {
		defer rows4.Close()
		for rows4.Next() {
			var v string
			if rows4.Scan(&v) == nil && v != "" {
				si.Districts = append(si.Districts, v)
			}
		}
	}
	si.DistrictCount = len(si.Districts)

	// ── 3. GPS coverage ──────────────────────────────────────────────────────
	var withGPS int
	_ = dbPool.QueryRow(ctx, `SELECT COUNT(1) FROM submissions WHERE form_id=$1 AND tenant_id=$2 AND (meta->>'gps_lat' IS NOT NULL AND meta->>'gps_lat' != '')`, formID, tenantID).Scan(&withGPS)
	if si.TotalSubmissions > 0 {
		si.GPSCoveragePct = math.Round((float64(withGPS)/float64(si.TotalSubmissions))*100*10) / 10
	}

	// ── 4. 90-day submission timeline ────────────────────────────────────────
	tlRows, err := dbPool.Query(ctx, `SELECT to_char(received_at, 'YYYY-MM-DD'), COUNT(1)
		FROM submissions WHERE form_id=$1 AND tenant_id=$2
		AND received_at >= NOW() - INTERVAL '90 days'
		GROUP BY 1 ORDER BY 1`, formID, tenantID)
	if err == nil {
		defer tlRows.Close()
		for tlRows.Next() {
			var p SubmissionTimelinePoint
			if tlRows.Scan(&p.Date, &p.Count) == nil {
				si.Timeline = append(si.Timeline, p)
			}
		}
	}

	// ── 5. Load template schema and classify fields ──────────────────────────
	if si.TotalSubmissions == 0 {
		return si, nil
	}

	var schemaBytes []byte
	_ = dbPool.QueryRow(ctx, `SELECT schema FROM templates WHERE id=$1`, formID).Scan(&schemaBytes)

	type SchemaField struct {
		Key      string `json:"key"`
		Label    string `json:"label"`
		Type     string `json:"type"`
		Required bool   `json:"required"`
	}
	type SchemaSection struct {
		Fields []SchemaField `json:"fields"`
	}
	var schemaObj struct {
		Sections []SchemaSection `json:"sections"`
	}

	if len(schemaBytes) > 0 {
		_ = json.Unmarshal(schemaBytes, &schemaObj)
	}

	// Collect all fields with their intel type
	var allFields []SchemaField
	for _, sec := range schemaObj.Sections {
		allFields = append(allFields, sec.Fields...)
	}

	totalMissingSum := 0.0
	totalMissingCount := 0
	totalOutliers := 0

	for _, f := range allFields {
		if f.Key == "" {
			continue
		}
		intelType := classifyFieldType(f.Type, f.Key)
		// Skip non-data fields
		if intelType == FieldTypeNote || intelType == FieldTypeCalculated || intelType == FieldTypeRepeat {
			continue
		}

		fi := FieldIntelligence{
			Key:       f.Key,
			Label:     f.Label,
			Type:      f.Type,
			IntelType: intelType,
			Required:  f.Required,
		}

		// Count filled vs missing
		_ = dbPool.QueryRow(ctx, fmt.Sprintf(
			`SELECT COUNT(1),
			 COUNT(1) FILTER (WHERE meta->>'%s' IS NOT NULL AND meta->>'%s' != ''),
			 COUNT(1) FILTER (WHERE meta->>'%s' IS NULL OR meta->>'%s' = '')
			 FROM submissions WHERE form_id=$1 AND tenant_id=$2`, f.Key, f.Key, f.Key, f.Key),
			formID, tenantID).Scan(&fi.TotalCount, &fi.FilledCount, &fi.MissingCount)

		if fi.TotalCount > 0 {
			fi.MissingPct = math.Round((float64(fi.MissingCount)/float64(fi.TotalCount))*100*10) / 10
		}
		totalMissingSum += fi.MissingPct
		totalMissingCount++

		if fi.MissingPct > 30 {
			si.HighRiskFields = append(si.HighRiskFields, f.Key)
		}

		switch intelType {
		case FieldTypeNumeric:
			si.NumericFields = append(si.NumericFields, f.Key)
			ds, outliers := computeNumericStats(ctx, f.Key, formID, tenantID)
			fi.DescStats = ds
			fi.Outliers = outliers
			totalOutliers += len(outliers)

		case FieldTypeCategorical:
			si.CategoricalFields = append(si.CategoricalFields, f.Key)
			dist := computeCategoricalDist(ctx, f.Key, formID, tenantID, fi.FilledCount)
			fi.Distribution = make(map[string]int)
			for _, vf := range dist {
				fi.Distribution[vf.Value] = vf.Count
			}
			fi.TopValues = dist
			// Check for gender/age keys and build demographics
			keyLower := strings.ToLower(f.Key)
			if strings.Contains(keyLower, "gender") || strings.Contains(keyLower, "sex") {
				si.GenderBreakdown = &DemographicBreakdown{
					FieldKey:   f.Key,
					FieldLabel: f.Label,
					Buckets:    dist,
				}
			}

		case FieldTypeGPS:
			si.GPSFields = append(si.GPSFields, f.Key)

		case FieldTypeDate:
			si.DateFields = append(si.DateFields, f.Key)

		case FieldTypeImage, FieldTypeSignature:
			si.ImageFields = append(si.ImageFields, f.Key)
		}

		si.Fields = append(si.Fields, fi)
	}

	// ── 6. Age distribution (auto-detected from numeric "age" fields) ────────
	for _, f := range allFields {
		keyLower := strings.ToLower(f.Key)
		if strings.Contains(keyLower, "age") && classifyFieldType(f.Type, f.Key) == FieldTypeNumeric {
			buckets := computeAgeBuckets(ctx, f.Key, formID, tenantID)
			if len(buckets) > 0 {
				si.AgeDistribution = &DemographicBreakdown{
					FieldKey:   f.Key,
					FieldLabel: f.Label,
					Buckets:    buckets,
				}
				break
			}
		}
	}

	// ── 7. Overall quality indicators ───────────────────────────────────────
	if totalMissingCount > 0 {
		si.OverallMissingPct = math.Round((totalMissingSum/float64(totalMissingCount))*10) / 10
	}
	si.OutlierCount = totalOutliers

	return si, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// computeNumericStats calculates descriptive statistics for a numeric field
// using PostgreSQL aggregate functions and IQR-based outlier detection.
// ─────────────────────────────────────────────────────────────────────────────

func computeNumericStats(ctx context.Context, field, formID, tenantID string) (*DescriptiveStats, []OutlierRecord) {
	ds := &DescriptiveStats{}

	// Use PostgreSQL for efficient aggregate computation
	q := fmt.Sprintf(`SELECT
		COUNT(1),
		MIN((meta->>'%s')::numeric),
		MAX((meta->>'%s')::numeric),
		AVG((meta->>'%s')::numeric),
		STDDEV((meta->>'%s')::numeric),
		PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY (meta->>'%s')::numeric),
		PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY (meta->>'%s')::numeric),
		PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY (meta->>'%s')::numeric)
		FROM submissions
		WHERE form_id=$1 AND tenant_id=$2
		AND meta->>'%s' IS NOT NULL
		AND meta->>'%s' != ''
		AND (meta->>'%s') ~ '^-?[0-9]+(\.[0-9]+)?$'`,
		field, field, field, field, field, field, field, field, field, field)

	if err := dbPool.QueryRow(ctx, q, formID, tenantID).Scan(
		&ds.Count, &ds.Min, &ds.Max, &ds.Mean, &ds.StdDev, &ds.P25, &ds.Median, &ds.P75,
	); err != nil {
		return nil, nil
	}
	if ds.Count == 0 {
		return nil, nil
	}
	ds.Min = math.Round(ds.Min*100) / 100
	ds.Max = math.Round(ds.Max*100) / 100
	ds.Mean = math.Round(ds.Mean*100) / 100
	ds.Median = math.Round(ds.Median*100) / 100
	if ds.StdDev > 0 {
		ds.StdDev = math.Round(ds.StdDev*100) / 100
	}
	ds.IQR = math.Round((ds.P75-ds.P25)*100) / 100

	// IQR-based outlier detection
	lowerFence := ds.P25 - 1.5*ds.IQR
	upperFence := ds.P75 + 1.5*ds.IQR

	oRows, err := dbPool.Query(ctx, fmt.Sprintf(`SELECT instance_id, (meta->>'%s')::numeric
		FROM submissions
		WHERE form_id=$1 AND tenant_id=$2
		AND meta->>'%s' IS NOT NULL
		AND meta->>'%s' != ''
		AND (meta->>'%s') ~ '^-?[0-9]+(\.[0-9]+)?$'
		AND ((meta->>'%s')::numeric < $3 OR (meta->>'%s')::numeric > $4)
		LIMIT 50`, field, field, field, field, field, field),
		formID, tenantID, lowerFence, upperFence)
	if err != nil {
		return ds, nil
	}
	defer oRows.Close()

	var outliers []OutlierRecord
	for oRows.Next() {
		var iid string
		var val float64
		if oRows.Scan(&iid, &val) == nil {
			zScore := 0.0
			if ds.StdDev > 0 {
				zScore = math.Abs((val - ds.Mean) / ds.StdDev)
			}
			severity := "moderate"
			if zScore > 3.0 {
				severity = "extreme"
			}
			outliers = append(outliers, OutlierRecord{
				InstanceID: iid,
				Value:      math.Round(val*100) / 100,
				ZScore:     math.Round(zScore*100) / 100,
				Severity:   severity,
			})
		}
	}
	return ds, outliers
}

// ─────────────────────────────────────────────────────────────────────────────
// computeCategoricalDist calculates value frequencies for a categorical field.
// ─────────────────────────────────────────────────────────────────────────────

func computeCategoricalDist(ctx context.Context, field, formID, tenantID string, total int) []ValueFrequency {
	rows, err := dbPool.Query(ctx, fmt.Sprintf(`SELECT COALESCE(meta->>'%s', ''), COUNT(1)
		FROM submissions WHERE form_id=$1 AND tenant_id=$2 AND meta->>'%s' IS NOT NULL AND meta->>'%s' != ''
		GROUP BY 1 ORDER BY 2 DESC LIMIT 20`, field, field, field), formID, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var dist []ValueFrequency
	for rows.Next() {
		var vf ValueFrequency
		if rows.Scan(&vf.Value, &vf.Count) == nil && vf.Value != "" {
			if total > 0 {
				vf.Pct = math.Round((float64(vf.Count)/float64(total))*100*10) / 10
			}
			dist = append(dist, vf)
		}
	}
	return dist
}

// ─────────────────────────────────────────────────────────────────────────────
// computeAgeBuckets groups age values into standard demographic buckets.
// ─────────────────────────────────────────────────────────────────────────────

func computeAgeBuckets(ctx context.Context, field, formID, tenantID string) []ValueFrequency {
	buckets := []struct{ label string; min, max int }{
		{"0–4", 0, 4}, {"5–14", 5, 14}, {"15–24", 15, 24},
		{"25–34", 25, 34}, {"35–44", 35, 44}, {"45–54", 45, 54},
		{"55–64", 55, 64}, {"65+", 65, 999},
	}

	var result []ValueFrequency
	var grandTotal int
	_ = dbPool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(1) FROM submissions WHERE form_id=$1 AND tenant_id=$2 AND meta->>'%s' IS NOT NULL AND (meta->>'%s') ~ '^[0-9]+$'`, field, field), formID, tenantID).Scan(&grandTotal)

	for _, b := range buckets {
		var cnt int
		_ = dbPool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(1) FROM submissions WHERE form_id=$1 AND tenant_id=$2 AND (meta->>'%s')::numeric BETWEEN $3 AND $4`, field), formID, tenantID, b.min, b.max).Scan(&cnt)
		if cnt > 0 {
			pct := 0.0
			if grandTotal > 0 {
				pct = math.Round((float64(cnt)/float64(grandTotal))*100*10) / 10
			}
			result = append(result, ValueFrequency{Value: b.label, Count: cnt, Pct: pct})
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP Handler
// ─────────────────────────────────────────────────────────────────────────────

func handleGetSurveyIntelligence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	formID := r.URL.Query().Get("form_id")
	if formID == "" {
		http.Error(w, "form_id is required", http.StatusBadRequest)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	si, err := GenerateSurveyIntelligence(r.Context(), formID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(si)
}

// ─────────────────────────────────────────────────────────────────────────────
// Indicator Calculation Engine
// Exposes on-demand frequency, cross-tabulation, and descriptive statistics.
// ─────────────────────────────────────────────────────────────────────────────

// FrequencyResult holds a frequency distribution for a single field.
type FrequencyResult struct {
	FormID   string           `json:"form_id"`
	Field    string           `json:"field"`
	Total    int              `json:"total"`
	Values   []ValueFrequency `json:"values"`
}

// CrossTabResult holds a cross-tabulation table.
type CrossTabResult struct {
	FormID   string                       `json:"form_id"`
	RowField string                       `json:"row_field"`
	ColField string                       `json:"col_field"`
	Rows     []string                     `json:"rows"`
	Cols     []string                     `json:"cols"`
	Cells    map[string]map[string]int    `json:"cells"`
	Total    int                          `json:"total"`
}

// handleGetFrequency computes frequency distribution for a field.
// GET /api/analytics/frequency?form_id=X&field=Y&tenant_id=Z
func handleGetFrequency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	formID := r.URL.Query().Get("form_id")
	field := r.URL.Query().Get("field")
	tenantID := r.URL.Query().Get("tenant_id")
	if formID == "" || field == "" {
		http.Error(w, "form_id and field are required", http.StatusBadRequest)
		return
	}
	if tenantID == "" {
		tenantID = "default"
	}

	var total int
	_ = dbPool.QueryRow(r.Context(), `SELECT COUNT(1) FROM submissions WHERE form_id=$1 AND tenant_id=$2`, formID, tenantID).Scan(&total)

	dist := computeCategoricalDist(r.Context(), field, formID, tenantID, total)
	result := FrequencyResult{
		FormID: formID,
		Field:  field,
		Total:  total,
		Values: dist,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// handleGetDescriptive computes descriptive statistics for a numeric field.
// GET /api/analytics/descriptive?form_id=X&field=Y&tenant_id=Z
func handleGetDescriptive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	formID := r.URL.Query().Get("form_id")
	field := r.URL.Query().Get("field")
	tenantID := r.URL.Query().Get("tenant_id")
	if formID == "" || field == "" {
		http.Error(w, "form_id and field are required", http.StatusBadRequest)
		return
	}
	if tenantID == "" {
		tenantID = "default"
	}

	ds, outliers := computeNumericStats(r.Context(), field, formID, tenantID)
	if ds == nil {
		http.Error(w, "no numeric data found for this field", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"form_id":  formID,
		"field":    field,
		"stats":    ds,
		"outliers": outliers,
	})
}

// handleGetCrossTab computes a cross-tabulation between two fields.
// GET /api/analytics/crosstab?form_id=X&row_field=Y&col_field=Z&tenant_id=T
func handleGetCrossTab(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	formID := r.URL.Query().Get("form_id")
	rowField := r.URL.Query().Get("row_field")
	colField := r.URL.Query().Get("col_field")
	tenantID := r.URL.Query().Get("tenant_id")
	if formID == "" || rowField == "" || colField == "" {
		http.Error(w, "form_id, row_field, col_field required", http.StatusBadRequest)
		return
	}
	if tenantID == "" {
		tenantID = "default"
	}

	ctx := r.Context()
	result := CrossTabResult{
		FormID:   formID,
		RowField: rowField,
		ColField: colField,
		Cells:    make(map[string]map[string]int),
	}

	query := fmt.Sprintf(`SELECT
		COALESCE(meta->>'%s', 'Unknown'),
		COALESCE(meta->>'%s', 'Unknown'),
		COUNT(1)
		FROM submissions WHERE form_id=$1 AND tenant_id=$2
		AND meta->>'%s' IS NOT NULL AND meta->>'%s' IS NOT NULL
		GROUP BY 1, 2 ORDER BY 1, 2`, rowField, colField, rowField, colField)

	rows, err := dbPool.Query(ctx, query, formID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	rowSet := make(map[string]bool)
	colSet := make(map[string]bool)
	for rows.Next() {
		var rv, cv string
		var cnt int
		if rows.Scan(&rv, &cv, &cnt) == nil {
			rowSet[rv] = true
			colSet[cv] = true
			if result.Cells[rv] == nil {
				result.Cells[rv] = make(map[string]int)
			}
			result.Cells[rv][cv] = cnt
			result.Total += cnt
		}
	}

	for k := range rowSet {
		result.Rows = append(result.Rows, k)
	}
	for k := range colSet {
		result.Cols = append(result.Cols, k)
	}
	sort.Strings(result.Rows)
	sort.Strings(result.Cols)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// SetupIntelligenceRoutes registers survey intelligence endpoints.
func SetupIntelligenceRoutes() {
	http.HandleFunc("/api/analytics/survey-intelligence", handleGetSurveyIntelligence)
	http.HandleFunc("/api/analytics/frequency",           handleGetFrequency)
	http.HandleFunc("/api/analytics/descriptive",         handleGetDescriptive)
	http.HandleFunc("/api/analytics/crosstab",            handleGetCrossTab)
}
