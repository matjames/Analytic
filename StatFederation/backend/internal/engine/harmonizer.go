package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"statfederation-backend/internal/models"
	"statfederation-backend/internal/store"
)

// Harmonizer maps local agency schemas to national canonical SDMX/DDI standards
type Harmonizer struct {
	store store.Store
}

// HarmonizationResult describes translation of incoming record into canonical statistical ontology
type HarmonizationResult struct {
	SourceAgency      string                 `json:"source_agency"`
	StandardFramework string                 `json:"standard_framework"`
	OriginalRecord    map[string]interface{} `json:"original_record"`
	HarmonizedRecord  map[string]interface{} `json:"harmonized_record"`
	MappedConcepts    map[string]string      `json:"mapped_concepts"`
	Transformations   []string               `json:"transformations"`
	ConfidenceScore   float64                `json:"confidence_score"`
	Timestamp         time.Time              `json:"timestamp"`
}

// NewHarmonizer initializes the metadata harmonizer
func NewHarmonizer(s store.Store) *Harmonizer {
	return &Harmonizer{store: s}
}

// HarmonizeRecord transforms arbitrary agency key-value pairs into standard SDMX / DDI concepts
func (h *Harmonizer) HarmonizeRecord(
	ctx context.Context,
	sourceAgency string,
	framework string,
	rawRecord map[string]interface{},
	tenantID string,
) (*HarmonizationResult, error) {
	if framework == "" {
		framework = "SDMX_2.1"
	}

	// 1. Fetch agency-specific vocabulary mappings
	vocabs, err := h.store.ListVocabularies(ctx, tenantID, sourceAgency)
	if err != nil {
		vocabs = []*models.MetadataVocabulary{}
	}

	vocabMap := make(map[string]string)
	for _, v := range vocabs {
		vocabMap[strings.ToLower(v.SourceConceptTerm)] = v.TargetCanonicalConcept
	}

	// Default fallback canonical concepts (SDMX standard dimensions)
	canonicalDefaults := map[string]string{
		"sex":          "DIM_SEX",
		"gender":       "DIM_SEX",
		"district":     "DIM_GEO_ADMIN2",
		"region":       "DIM_GEO_ADMIN1",
		"age":          "DIM_AGE_GROUP",
		"year":         "DIM_TIME_PERIOD",
		"period":       "DIM_TIME_PERIOD",
		"amount":       "OBS_VALUE",
		"val":          "OBS_VALUE",
		"count":        "OBS_VALUE",
		"status":       "OBS_STATUS",
		"facility_id":  "REF_FACILITY",
		"mortality":    "IND_MORTALITY_RATE",
		"fertility":    "IND_FERTILITY_RATE",
		"gdp_growth":   "IND_GDP_GROWTH",
	}

	harmonized := make(map[string]interface{})
	mappedConcepts := make(map[string]string)
	transformations := make([]string, 0)

	for key, val := range rawRecord {
		cleanKey := strings.ToLower(strings.TrimSpace(key))
		canonicalTarget, exists := vocabMap[cleanKey]
		if !exists {
			canonicalTarget, exists = canonicalDefaults[cleanKey]
		}

		if exists {
			harmonized[canonicalTarget] = val
			mappedConcepts[key] = canonicalTarget
			transformations = append(transformations, fmt.Sprintf("Mapped '%s' -> '%s'", key, canonicalTarget))
		} else {
			// Keep with standard uppercase canonical prefix
			customConcept := "ATTR_" + strings.ToUpper(cleanKey)
			harmonized[customConcept] = val
			mappedConcepts[key] = customConcept
			transformations = append(transformations, fmt.Sprintf("Retained '%s' as generic attribute '%s'", key, customConcept))
		}
	}

	// Add SDMX header attributes
	harmonized["STRUCTURE"] = "urn:sdmx:org.sdmx.infomodel.datastructure.DataStructure=" + framework
	harmonized["DATAFLOW"] = fmt.Sprintf("STATGATE:%s_INDICATORS(1.0)", strings.ToUpper(sourceAgency))

	return &HarmonizationResult{
		SourceAgency:      sourceAgency,
		StandardFramework: framework,
		OriginalRecord:    rawRecord,
		HarmonizedRecord:  harmonized,
		MappedConcepts:    mappedConcepts,
		Transformations:   transformations,
		ConfidenceScore:   0.98,
		Timestamp:         time.Now().UTC(),
	}, nil
}
