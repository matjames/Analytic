package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Directive 21 — Mandatory Survey Header (MSH) Engine
// Every survey created in StatCollect begins with a standardised MSH section
// before any survey-specific questions. The MSH covers 8 standard sections:
// Survey Metadata, Assignment Info, Respondent Info, Geographic Info,
// Collection Metadata, Consent, Attachments, and Audit Metadata.
// ─────────────────────────────────────────────────────────────────────────────

// SurveyHeader holds the MSH configuration for a single template.
type SurveyHeader struct {
	ID                  int64           `json:"id"`
	TemplateID          string          `json:"template_id"`
	Enabled             bool            `json:"enabled"`
	IncludeMetadata     bool            `json:"include_metadata"`
	IncludeAssignment   bool            `json:"include_assignment"`
	IncludeRespondent   bool            `json:"include_respondent"`
	IncludeGeography    bool            `json:"include_geography"`
	IncludeCollection   bool            `json:"include_collection"`
	IncludeConsent      bool            `json:"include_consent"`
	IncludeAttachments  bool            `json:"include_attachments"`
	IncludeAudit        bool            `json:"include_audit"`
	RespondentTypes     json.RawMessage `json:"respondent_types"`
	CountryCode         string          `json:"country_code"`
	ConsentConfig       json.RawMessage `json:"consent_config"`
	PublishToResearch   bool            `json:"publish_to_research"`
	PublishToProjects   bool            `json:"publish_to_projects"`
	PublishToStatistics bool            `json:"publish_to_statistics"`
	PublishToGIS        bool            `json:"publish_to_gis"`
	PublishToReporting  bool            `json:"publish_to_reporting"`
	PublishToDocuments  bool            `json:"publish_to_documents"`
	AutoCreateTasks     bool            `json:"auto_create_tasks"`
	TenantID            string          `json:"tenant_id"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// MSHRespondentType represents a configurable respondent category.
type MSHRespondentType struct {
	ID           string          `json:"id"`
	Label        string          `json:"label"`
	Description  string          `json:"description"`
	CommonFields json.RawMessage `json:"common_fields"`
	Enabled      bool            `json:"enabled"`
	SortOrder    int             `json:"sort_order"`
}

// MSHAdminLevel represents one level in a country's administrative hierarchy.
type MSHAdminLevel struct {
	ID          int64  `json:"id"`
	CountryCode string `json:"country_code"`
	LevelNumber int    `json:"level_number"`
	LevelName   string `json:"level_name"`
	Required    bool   `json:"required"`
}

// DefaultSurveyHeader returns a fully-enabled MSH configuration with defaults.
func DefaultSurveyHeader(templateID string) SurveyHeader {
	return SurveyHeader{
		TemplateID:          templateID,
		Enabled:             true,
		IncludeMetadata:     true,
		IncludeAssignment:   true,
		IncludeRespondent:   true,
		IncludeGeography:    true,
		IncludeCollection:   true,
		IncludeConsent:      true,
		IncludeAttachments:  false,
		IncludeAudit:        true,
		RespondentTypes:     json.RawMessage(`["individual","household","facility"]`),
		CountryCode:         "UGA",
		ConsentConfig:       json.RawMessage(`{"verbal":true,"written":false,"guardian":false,"digital_signature":true,"audio":false}`),
		PublishToResearch:   true,
		PublishToProjects:   true,
		PublishToStatistics: true,
		PublishToGIS:        true,
		PublishToReporting:  true,
		PublishToDocuments:  false,
		AutoCreateTasks:     true,
		TenantID:            "default",
	}
}

// GetSurveyHeader loads the MSH configuration for a template.
func GetSurveyHeader(templateID string) (*SurveyHeader, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sh SurveyHeader
	var rt, cc []byte
	err := dbPool.QueryRow(ctx, `SELECT id, template_id, enabled,
		include_metadata, include_assignment, include_respondent, include_geography,
		include_collection, include_consent, include_attachments, include_audit,
		respondent_types, country_code, consent_config,
		publish_to_research, publish_to_projects, publish_to_statistics,
		publish_to_gis, publish_to_reporting, publish_to_documents,
		auto_create_tasks, tenant_id, created_at, updated_at
		FROM survey_headers WHERE template_id=$1`, templateID).Scan(
		&sh.ID, &sh.TemplateID, &sh.Enabled,
		&sh.IncludeMetadata, &sh.IncludeAssignment, &sh.IncludeRespondent, &sh.IncludeGeography,
		&sh.IncludeCollection, &sh.IncludeConsent, &sh.IncludeAttachments, &sh.IncludeAudit,
		&rt, &sh.CountryCode, &cc,
		&sh.PublishToResearch, &sh.PublishToProjects, &sh.PublishToStatistics,
		&sh.PublishToGIS, &sh.PublishToReporting, &sh.PublishToDocuments,
		&sh.AutoCreateTasks, &sh.TenantID, &sh.CreatedAt, &sh.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get survey header: %w", err)
	}
	sh.RespondentTypes = rt
	sh.ConsentConfig = cc
	return &sh, nil
}

// SaveSurveyHeader upserts an MSH configuration for a template.
func SaveSurveyHeader(sh SurveyHeader) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	if sh.TenantID == "" {
		sh.TenantID = "default"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO survey_headers
		(template_id, enabled, include_metadata, include_assignment, include_respondent,
		 include_geography, include_collection, include_consent, include_attachments, include_audit,
		 respondent_types, country_code, consent_config,
		 publish_to_research, publish_to_projects, publish_to_statistics,
		 publish_to_gis, publish_to_reporting, publish_to_documents,
		 auto_create_tasks, tenant_id, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,now())
		ON CONFLICT (template_id) DO UPDATE SET
			enabled=EXCLUDED.enabled,
			include_metadata=EXCLUDED.include_metadata,
			include_assignment=EXCLUDED.include_assignment,
			include_respondent=EXCLUDED.include_respondent,
			include_geography=EXCLUDED.include_geography,
			include_collection=EXCLUDED.include_collection,
			include_consent=EXCLUDED.include_consent,
			include_attachments=EXCLUDED.include_attachments,
			include_audit=EXCLUDED.include_audit,
			respondent_types=EXCLUDED.respondent_types,
			country_code=EXCLUDED.country_code,
			consent_config=EXCLUDED.consent_config,
			publish_to_research=EXCLUDED.publish_to_research,
			publish_to_projects=EXCLUDED.publish_to_projects,
			publish_to_statistics=EXCLUDED.publish_to_statistics,
			publish_to_gis=EXCLUDED.publish_to_gis,
			publish_to_reporting=EXCLUDED.publish_to_reporting,
			publish_to_documents=EXCLUDED.publish_to_documents,
			auto_create_tasks=EXCLUDED.auto_create_tasks,
			updated_at=now()`,
		sh.TemplateID, sh.Enabled,
		sh.IncludeMetadata, sh.IncludeAssignment, sh.IncludeRespondent,
		sh.IncludeGeography, sh.IncludeCollection, sh.IncludeConsent, sh.IncludeAttachments, sh.IncludeAudit,
		[]byte(sh.RespondentTypes), sh.CountryCode, []byte(sh.ConsentConfig),
		sh.PublishToResearch, sh.PublishToProjects, sh.PublishToStatistics,
		sh.PublishToGIS, sh.PublishToReporting, sh.PublishToDocuments,
		sh.AutoCreateTasks, sh.TenantID)
	return err
}

// GetAdminLevels returns the admin boundary levels for a country.
func GetAdminLevels(countryCode string) ([]MSHAdminLevel, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx,
		`SELECT id, country_code, level_number, level_name, required
		 FROM msh_admin_levels WHERE country_code=$1 ORDER BY level_number`, countryCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var levels []MSHAdminLevel
	for rows.Next() {
		var l MSHAdminLevel
		if err := rows.Scan(&l.ID, &l.CountryCode, &l.LevelNumber, &l.LevelName, &l.Required); err != nil {
			return nil, err
		}
		levels = append(levels, l)
	}
	return levels, nil
}

// GetRespondentTypes returns all enabled respondent types from the registry.
func GetRespondentTypes() ([]MSHRespondentType, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx,
		`SELECT id, label, COALESCE(description,''), common_fields, enabled, sort_order
		 FROM msh_respondent_types WHERE enabled=true ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var types []MSHRespondentType
	for rows.Next() {
		var rt MSHRespondentType
		var cf []byte
		if err := rows.Scan(&rt.ID, &rt.Label, &rt.Description, &cf, &rt.Enabled, &rt.SortOrder); err != nil {
			return nil, err
		}
		rt.CommonFields = cf
		types = append(types, rt)
	}
	return types, nil
}

// BuildMSHSchema constructs the MSH section JSON that gets prepended to any
// survey schema. It reads the MSH config and admin levels for the template's
// configured country, then assembles a full "sections" array.
func BuildMSHSchema(sh SurveyHeader) (json.RawMessage, error) {
	levels, err := GetAdminLevels(sh.CountryCode)
	if err != nil {
		// graceful fallback — use Uganda defaults if DB unavailable
		levels = defaultUgandaLevels()
	}

	var sections []map[string]interface{}

	// ── Section 1: Survey Metadata ────────────────────────────────────────────
	if sh.IncludeMetadata {
		sections = append(sections, map[string]interface{}{
			"id": "_msh_metadata", "title": "Survey Metadata", "protected": false,
			"description": "Mandatory survey identification and classification information",
			"fields": []map[string]interface{}{
				{"key": "_msh_survey_title", "label": "Survey Title", "type": "text", "required": true, "readonly": true, "auto": "template.name"},
				{"key": "_msh_survey_code", "label": "Survey Code", "type": "text", "required": true},
				{"key": "_msh_survey_version", "label": "Survey Version", "type": "text", "required": true, "readonly": true, "auto": "template.version"},
				{"key": "_msh_organization", "label": "Organization", "type": "text", "required": true},
				{"key": "_msh_project", "label": "Project", "type": "text", "required": false},
				{"key": "_msh_programme", "label": "Programme", "type": "text", "required": false},
				{"key": "_msh_department", "label": "Department", "type": "text", "required": false},
				{"key": "_msh_funding_partner", "label": "Funding Partner", "type": "text", "required": false},
				{"key": "_msh_survey_category", "label": "Survey Category", "type": "select", "required": true, "options": []map[string]string{
					{"value": "health", "label": "Health"}, {"value": "census", "label": "Census"},
					{"value": "agriculture", "label": "Agriculture"}, {"value": "education", "label": "Education"},
					{"value": "water_sanitation", "label": "Water & Sanitation"}, {"value": "livelihoods", "label": "Livelihoods"},
					{"value": "nutrition", "label": "Nutrition"}, {"value": "protection", "label": "Protection"},
					{"value": "emergency", "label": "Emergency / Rapid Assessment"}, {"value": "research", "label": "Research"},
					{"value": "other", "label": "Other"},
				}},
				{"key": "_msh_survey_type", "label": "Survey Type", "type": "select", "required": true, "options": []map[string]string{
					{"value": "baseline", "label": "Baseline"}, {"value": "midline", "label": "Midline"},
					{"value": "endline", "label": "Endline"}, {"value": "routine", "label": "Routine Monitoring"},
					{"value": "rapid", "label": "Rapid Assessment"}, {"value": "census", "label": "Census"},
					{"value": "longitudinal", "label": "Longitudinal / Follow-up"}, {"value": "adhoc", "label": "Ad-hoc"},
				}},
				{"key": "_msh_survey_description", "label": "Survey Description", "type": "textarea", "required": false},
			},
		})
	}

	// ── Section 2: Assignment Information ─────────────────────────────────────
	if sh.IncludeAssignment {
		sections = append(sections, map[string]interface{}{
			"id": "_msh_assignment", "title": "Assignment Information", "protected": false,
			"description": "Automatically populated from your assignment. Contact your supervisor to update.",
			"fields": []map[string]interface{}{
				{"key": "_msh_enumerator_name", "label": "Enumerator Name", "type": "text", "required": true, "auto": "session.user_name"},
				{"key": "_msh_enumerator_id", "label": "Enumerator ID", "type": "text", "required": true, "auto": "session.user_id", "readonly": true},
				{"key": "_msh_supervisor", "label": "Supervisor", "type": "text", "required": false, "auto": "assignment.supervisor"},
				{"key": "_msh_team", "label": "Team", "type": "text", "required": false, "auto": "assignment.team"},
				{"key": "_msh_enumerator_org", "label": "Organization", "type": "text", "required": false, "auto": "session.organization"},
				{"key": "_msh_assigned_area", "label": "Assigned Area", "type": "text", "required": false, "auto": "assignment.area"},
				{"key": "_msh_assigned_facility", "label": "Assigned Facility", "type": "text", "required": false, "auto": "assignment.facility"},
				{"key": "_msh_assigned_household", "label": "Assigned Household ID", "type": "text", "required": false, "auto": "assignment.household_id"},
				{"key": "_msh_assigned_target", "label": "Assigned Target", "type": "text", "required": false, "auto": "assignment.target"},
			},
		})
	}

	// ── Section 3: Respondent Information ─────────────────────────────────────
	if sh.IncludeRespondent {
		sections = append(sections, map[string]interface{}{
			"id": "_msh_respondent", "title": "Respondent Information", "protected": false,
			"description": "Information about the individual, household, facility, or other entity being surveyed",
			"fields": []map[string]interface{}{
				{"key": "_msh_respondent_type", "label": "Respondent Type", "type": "select", "required": true, "options": []map[string]string{
					{"value": "individual", "label": "Individual"}, {"value": "household", "label": "Household"},
					{"value": "facility", "label": "Health Facility"}, {"value": "school", "label": "School"},
					{"value": "business", "label": "Business/Enterprise"}, {"value": "farm", "label": "Farm/Agricultural Unit"},
					{"value": "community", "label": "Community"}, {"value": "institution", "label": "Institution"},
				}},
				{"key": "_msh_respondent_id", "label": "Respondent Identifier", "type": "text", "required": false,
					"description": "National ID, household number, facility code, or other unique identifier"},
				{"key": "_msh_respondent_name", "label": "Respondent Name", "type": "text", "required": false,
					"description": "Only collect where permitted by privacy policy and consent"},
				{"key": "_msh_respondent_sex", "label": "Sex / Gender", "type": "select", "required": false,
					"showWhen": map[string]interface{}{"field": "_msh_respondent_type", "values": []string{"individual"}},
					"options": []map[string]string{
						{"value": "male", "label": "Male"}, {"value": "female", "label": "Female"},
						{"value": "other", "label": "Other"}, {"value": "prefer_not", "label": "Prefer not to say"},
					}},
				{"key": "_msh_respondent_dob", "label": "Date of Birth", "type": "date", "required": false,
					"showWhen": map[string]interface{}{"field": "_msh_respondent_type", "values": []string{"individual"}}},
				{"key": "_msh_respondent_age", "label": "Age (years)", "type": "number", "min": 0, "max": 150, "required": false,
					"showWhen": map[string]interface{}{"field": "_msh_respondent_type", "values": []string{"individual"}}},
				{"key": "_msh_respondent_contact", "label": "Contact Information", "type": "tel", "required": false},
				{"key": "_msh_consent_status", "label": "Consent Status", "type": "select", "required": true, "options": []map[string]string{
					{"value": "obtained", "label": "Consent Obtained"},
					{"value": "refused", "label": "Consent Refused"},
					{"value": "not_applicable", "label": "Not Applicable"},
				}},
			},
		})
	}

	// ── Section 4: Geographic Information ─────────────────────────────────────
	if sh.IncludeGeography {
		geoFields := []map[string]interface{}{
			{"key": "_msh_country", "label": "Country", "type": "text", "required": true, "auto": "device.country", "readonly": false},
		}
		for _, l := range levels {
			if l.LevelNumber <= 1 {
				continue // Country already added above
			}
			geoFields = append(geoFields, map[string]interface{}{
				"key": fmt.Sprintf("_msh_admin_%d", l.LevelNumber), "label": l.LevelName,
				"type": "select_cascade", "required": l.Required,
				"cascade_from": fmt.Sprintf("_msh_admin_%d", l.LevelNumber-1),
				"level":        l.LevelNumber,
			})
		}
		geoFields = append(geoFields,
			map[string]interface{}{"key": "_msh_facility_name", "label": "Facility / Venue", "type": "text", "required": false},
			map[string]interface{}{"key": "_msh_gps", "label": "GPS Coordinates", "type": "gps", "required": true, "auto": "device.gps"},
			map[string]interface{}{"key": "_msh_location_accuracy", "label": "Location Accuracy (m)", "type": "number", "required": false, "auto": "device.gps_accuracy", "readonly": true},
			map[string]interface{}{"key": "_msh_admin_boundary", "label": "Administrative Boundary", "type": "text", "required": false, "auto": "device.admin_boundary"},
		)
		sections = append(sections, map[string]interface{}{
			"id": "_msh_geography", "title": "Geographic Information", "protected": false,
			"description": "Geographic location of data collection",
			"fields":      geoFields,
		})
	}

	// ── Section 5: Collection Metadata ────────────────────────────────────────
	if sh.IncludeCollection {
		sections = append(sections, map[string]interface{}{
			"id": "_msh_collection", "title": "Collection Metadata", "protected": true,
			"description": "Automatically captured by the system. These fields cannot be manually edited.",
			"fields": []map[string]interface{}{
				{"key": "_msh_interview_date", "label": "Interview Date", "type": "date", "required": true, "auto": "device.date", "readonly": true},
				{"key": "_msh_start_time", "label": "Interview Start Time", "type": "time", "required": true, "auto": "device.start_time", "readonly": true},
				{"key": "_msh_end_time", "label": "Interview End Time", "type": "time", "required": false, "auto": "device.end_time", "readonly": true},
				{"key": "_msh_duration", "label": "Duration (seconds)", "type": "number", "required": false, "auto": "device.duration", "readonly": true},
				{"key": "_msh_device_id", "label": "Device ID", "type": "text", "required": false, "auto": "device.id", "readonly": true},
				{"key": "_msh_app_version", "label": "Application Version", "type": "text", "required": false, "auto": "device.app_version", "readonly": true},
				{"key": "_msh_survey_version_captured", "label": "Survey Version (at collection)", "type": "text", "required": false, "auto": "template.version", "readonly": true},
				{"key": "_msh_network_status", "label": "Network Status", "type": "text", "required": false, "auto": "device.network_status", "readonly": true},
				{"key": "_msh_offline_mode", "label": "Offline / Online Mode", "type": "select", "required": false, "auto": "device.offline_mode", "readonly": true,
					"options": []map[string]string{{"value": "online", "label": "Online"}, {"value": "offline", "label": "Offline"}}},
				{"key": "_msh_sync_status", "label": "Synchronization Status", "type": "text", "required": false, "auto": "device.sync_status", "readonly": true},
			},
		})
	}

	// ── Section 6: Consent ────────────────────────────────────────────────────
	if sh.IncludeConsent {
		sections = append(sections, map[string]interface{}{
			"id": "_msh_consent", "title": "Consent", "protected": false,
			"description": "Consent must be obtained before the interview proceeds",
			"fields": []map[string]interface{}{
				{"key": "_msh_participant_info_read", "label": "Was participant information read to the respondent?", "type": "select", "required": true,
					"options": []map[string]string{{"value": "yes", "label": "Yes"}, {"value": "no", "label": "No"}}},
				{"key": "_msh_verbal_consent", "label": "Verbal Consent Obtained", "type": "select", "required": true,
					"options": []map[string]string{{"value": "yes", "label": "Yes, consent given"}, {"value": "no", "label": "No, consent refused"}}},
				{"key": "_msh_written_consent", "label": "Written Consent Obtained", "type": "select", "required": false,
					"options": []map[string]string{{"value": "yes", "label": "Yes"}, {"value": "no", "label": "No"}, {"value": "na", "label": "Not Applicable"}}},
				{"key": "_msh_guardian_consent", "label": "Guardian / Parent Consent", "type": "select", "required": false,
					"description": "Required for respondents under 18",
					"options": []map[string]string{{"value": "yes", "label": "Yes"}, {"value": "no", "label": "No"}, {"value": "na", "label": "Not Applicable"}}},
				{"key": "_msh_digital_signature", "label": "Digital Signature", "type": "signature", "required": false},
				{"key": "_msh_consent_timestamp", "label": "Consent Timestamp", "type": "datetime", "required": false, "auto": "device.datetime", "readonly": true},
				{"key": "_msh_consent_audio", "label": "Audio Consent Recording", "type": "audio", "required": false,
					"description": "Record verbal consent where written consent is not possible"},
			},
		})
	}

	// ── Section 7: Attachments ────────────────────────────────────────────────
	if sh.IncludeAttachments {
		sections = append(sections, map[string]interface{}{
			"id": "_msh_attachments", "title": "Attachments", "protected": false,
			"description": "Optional attachments to support this interview record",
			"fields": []map[string]interface{}{
				{"key": "_msh_participant_photo", "label": "Participant Photo", "type": "photo", "required": false},
				{"key": "_msh_id_document", "label": "Identification Document", "type": "photo", "required": false},
				{"key": "_msh_supporting_doc", "label": "Supporting Document", "type": "file", "required": false},
				{"key": "_msh_facility_photo", "label": "Facility / Site Photo", "type": "photo", "required": false},
			},
		})
	}

	// ── Section 8: Audit Metadata ─────────────────────────────────────────────
	if sh.IncludeAudit {
		sections = append(sections, map[string]interface{}{
			"id": "_msh_audit", "title": "Audit Metadata", "protected": true,
			"description": "Automatically maintained by the platform. Protected from unauthorized editing.",
			"fields": []map[string]interface{}{
				{"key": "_msh_created_by", "label": "Created By", "type": "text", "required": false, "auto": "session.user_id", "readonly": true},
				{"key": "_msh_created_date", "label": "Created Date", "type": "datetime", "required": false, "auto": "device.datetime", "readonly": true},
				{"key": "_msh_modified_by", "label": "Last Modified By", "type": "text", "required": false, "auto": "session.user_id", "readonly": true},
				{"key": "_msh_modified_date", "label": "Last Modified Date", "type": "datetime", "required": false, "auto": "device.datetime", "readonly": true},
				{"key": "_msh_approved_by", "label": "Approved By", "type": "text", "required": false, "readonly": true},
				{"key": "_msh_approval_date", "label": "Approval Date", "type": "datetime", "required": false, "readonly": true},
				{"key": "_msh_submission_status", "label": "Submission Status", "type": "text", "required": false, "auto": "submission.status", "readonly": true},
				{"key": "_msh_review_status", "label": "Review Status", "type": "text", "required": false, "auto": "submission.review_status", "readonly": true},
				{"key": "_msh_quality_score", "label": "Quality Score", "type": "number", "required": false, "readonly": true},
			},
		})
	}

	raw, err := json.Marshal(sections)
	if err != nil {
		return nil, fmt.Errorf("build MSH schema: %w", err)
	}
	return raw, nil
}

// InjectMSHIntoSchema takes an existing template schema JSON and prepends
// the MSH as the first group of sections.
func InjectMSHIntoSchema(existingSchema json.RawMessage, sh SurveyHeader) (json.RawMessage, error) {
	mshSections, err := BuildMSHSchema(sh)
	if err != nil {
		return existingSchema, err
	}

	// Parse the existing schema to get its sections array
	var schemaDef map[string]json.RawMessage
	if err := json.Unmarshal(existingSchema, &schemaDef); err != nil {
		return existingSchema, fmt.Errorf("parse schema: %w", err)
	}

	var existingSections []json.RawMessage
	if raw, ok := schemaDef["sections"]; ok {
		if err := json.Unmarshal(raw, &existingSections); err != nil {
			return existingSchema, err
		}
	}

	// Remove any previously-injected MSH sections (id starts with _msh_)
	var cleanedSections []json.RawMessage
	for _, s := range existingSections {
		var sec map[string]interface{}
		if err := json.Unmarshal(s, &sec); err == nil {
			if id, ok := sec["id"].(string); ok && len(id) > 4 && id[:5] == "_msh_" {
				continue
			}
		}
		cleanedSections = append(cleanedSections, s)
	}

	// Prepend MSH sections
	var mshArr []json.RawMessage
	if err := json.Unmarshal(mshSections, &mshArr); err != nil {
		return existingSchema, err
	}
	allSections := append(mshArr, cleanedSections...)

	sectionsJSON, err := json.Marshal(allSections)
	if err != nil {
		return existingSchema, err
	}

	schemaDef["sections"] = sectionsJSON
	result, err := json.Marshal(schemaDef)
	if err != nil {
		return existingSchema, err
	}
	return result, nil
}

// ValidateMSHFields checks that all mandatory MSH fields are present in a submission.
// Returns a list of missing field keys.
func ValidateMSHFields(meta map[string]interface{}, sh SurveyHeader) []string {
	var missing []string
	mandatoryFields := []string{"_msh_enumerator_id", "_msh_interview_date", "_msh_gps"}
	if sh.IncludeConsent {
		mandatoryFields = append(mandatoryFields, "_msh_verbal_consent", "_msh_consent_status")
	}
	for _, key := range mandatoryFields {
		if v, ok := meta[key]; !ok || v == nil || v == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

// defaultUgandaLevels provides fallback admin levels when the DB is unavailable.
func defaultUgandaLevels() []MSHAdminLevel {
	return []MSHAdminLevel{
		{LevelNumber: 2, LevelName: "Region", Required: true, CountryCode: "UGA"},
		{LevelNumber: 3, LevelName: "District", Required: true, CountryCode: "UGA"},
		{LevelNumber: 4, LevelName: "County", Required: false, CountryCode: "UGA"},
		{LevelNumber: 5, LevelName: "Sub-county", Required: false, CountryCode: "UGA"},
		{LevelNumber: 6, LevelName: "Parish", Required: false, CountryCode: "UGA"},
		{LevelNumber: 7, LevelName: "Village", Required: false, CountryCode: "UGA"},
	}
}
