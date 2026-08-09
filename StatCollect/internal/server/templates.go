package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Template represents a questionnaire form definition stored in the database.
// The schema is a JSON structure that defines sections, fields, validation,
// and conditional logic — fully dynamic, no hardcoded forms.
type Template struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Version     string          `json:"version"`
	Category      string          `json:"category,omitempty"`
	Rating        int16           `json:"rating"`
	DownloadCount int             `json:"download_count"`
	Tags          []string        `json:"tags,omitempty"`
	Schema        json.RawMessage `json:"schema"`
	Status        string          `json:"status"`
	CreatedBy     string          `json:"created_by,omitempty"`
	TenantID      string          `json:"tenant_id"`
	IsShared      bool            `json:"is_shared"`
	ParentID      string          `json:"parent_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// TemplateSummary is a lightweight view for listing templates.
type TemplateSummary struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Version       string    `json:"version"`
	Category      string    `json:"category,omitempty"`
	Rating        int16     `json:"rating"`
	DownloadCount int       `json:"download_count"`
	Status        string    `json:"status"`
	TenantID      string    `json:"tenant_id"`
	IsShared      bool      `json:"is_shared"`
	ParentID      string    `json:"parent_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TemplateVersion represents a historical version of a template.
type TemplateVersion struct {
	ID         int64           `json:"id"`
	TemplateID string          `json:"template_id"`
	Version    string          `json:"version"`
	Schema     json.RawMessage `json:"schema"`
	ChangedBy  string          `json:"changed_by,omitempty"`
	ChangeNote string          `json:"change_note,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// TemplateLibraryEntry is an entry in the built-in template library catalog.
type TemplateLibraryEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	FieldCount  int      `json:"field_count"`
	Version     string   `json:"version"`
}

// SaveTemplate creates or updates a template in the database.
func SaveTemplate(t Template) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tenantID := t.TenantID
	if tenantID == "" {
		tenantID = "default"
	}

	// Auto-inject MSH configuration block if enabled
	injectedSchema := t.Schema
	if sh, err := GetSurveyHeader(t.ID); err == nil && sh.Enabled {
		if modified, err := InjectMSHIntoSchema(t.Schema, *sh); err == nil {
			injectedSchema = modified
		}
	} else if err != nil {
		// No custom SurveyHeader yet, inject the default MSH
		defSH := DefaultSurveyHeader(t.ID)
		if modified, err := InjectMSHIntoSchema(t.Schema, defSH); err == nil {
			injectedSchema = modified
		}
	}

	tagsJSON, _ := json.Marshal(t.Tags)
	_, err := dbPool.Exec(ctx, `INSERT INTO templates (id, name, description, version, schema, status, created_by, tenant_id, category, rating, download_count, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11, now())
		ON CONFLICT (id) DO UPDATE SET
			name=EXCLUDED.name,
			description=EXCLUDED.description,
			version=EXCLUDED.version,
			schema=EXCLUDED.schema,
			status=EXCLUDED.status,
			created_by=EXCLUDED.created_by,
			category=EXCLUDED.category,
			rating=EXCLUDED.rating,
			download_count=EXCLUDED.download_count,
			updated_at=now()`,
		t.ID, t.Name, t.Description, t.Version, injectedSchema, t.Status, t.CreatedBy, tenantID, t.Category, t.Rating, t.DownloadCount)
	if err != nil {
		return fmt.Errorf("save template: %w", err)
	}
	_ = tagsJSON // stored in schema metadata; column extension optional
	return nil
}

// GetTemplate retrieves a template by ID.
func GetTemplate(id string) (*Template, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var t Template
	var schema []byte
	var createdBy string
	err := dbPool.QueryRow(ctx, `SELECT id, name, COALESCE(description,''), version, schema, status, COALESCE(created_by,''), tenant_id, COALESCE(category,'General'), rating, download_count, created_at, updated_at
		FROM templates WHERE id=$1`, id).Scan(
		&t.ID, &t.Name, &t.Description, &t.Version, &schema, &t.Status, &createdBy, &t.TenantID, &t.Category, &t.Rating, &t.DownloadCount, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Schema = schema
	t.CreatedBy = createdBy
	return &t, nil
}

// ListTemplates returns all templates, optionally filtered by status.
func ListTemplates(status string) ([]TemplateSummary, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT id, name, COALESCE(description,''), version, status, tenant_id, COALESCE(category,'General'), rating, download_count, created_at, updated_at FROM templates`
	var args []interface{}
	if status != "" {
		query += ` WHERE status=$1`
		args = append(args, status)
	}
	query += ` ORDER BY name`
	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TemplateSummary
	for rows.Next() {
		var t TemplateSummary
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Version, &t.Status, &t.TenantID, &t.Category, &t.Rating, &t.DownloadCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// DeleteTemplate removes a template by ID.
func DeleteTemplate(id string) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `DELETE FROM templates WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	return nil
}

// UpdateTemplateStatus changes a template's status (active/draft/archived).
func UpdateTemplateStatus(id, status string) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `UPDATE templates SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	if err != nil {
		return fmt.Errorf("update template status: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────
// ENTERPRISE TEMPLATE LIBRARY — 7 comprehensive sector templates
// ──────────────────────────────────────────────────────────────

// GetTemplateLibrary returns the built-in catalog of enterprise templates.
// This is the "template picker" shown in the Visual Builder library modal.
func GetTemplateLibrary() []TemplateLibraryEntry {
	return []TemplateLibraryEntry{
		{ID: "community_health_survey", Name: "Community Health Survey", Description: "Household health, water & sanitation assessment", Category: "Health", Tags: []string{"health", "household", "WASH"}, FieldCount: 28, Version: "1.0"},
		{ID: "household_census", Name: "Household Census", Description: "Full household demographic & socioeconomic census", Category: "Demographics", Tags: []string{"census", "demographics", "household"}, FieldCount: 35, Version: "1.0"},
		{ID: "school_enrollment_survey", Name: "School Enrollment Survey", Description: "School enrollment, attendance & infrastructure", Category: "Education", Tags: []string{"education", "school", "enrollment"}, FieldCount: 22, Version: "1.0"},
		{ID: "nutrition_survey", Name: "Child Nutrition Survey", Description: "Under-5 anthropometric & dietary diversity assessment", Category: "Nutrition", Tags: []string{"nutrition", "MUAC", "anthropometry", "child"}, FieldCount: 30, Version: "1.0"},
		{ID: "agriculture_survey", Name: "Agriculture & Livelihood Survey", Description: "Crop production, input use & market access", Category: "Agriculture", Tags: []string{"agriculture", "crops", "livestock", "livelihood"}, FieldCount: 32, Version: "1.0"},
		{ID: "water_sanitation_survey", Name: "Water & Sanitation Survey", Description: "WASH infrastructure, access & hygiene practices", Category: "WASH", Tags: []string{"water", "sanitation", "hygiene", "WASH"}, FieldCount: 24, Version: "1.0"},
		{ID: "market_inspection", Name: "Market Price Survey", Description: "Commodity prices, availability & market conditions", Category: "Markets", Tags: []string{"markets", "prices", "food security"}, FieldCount: 20, Version: "1.0"},
		{ID: "facility_inspection", Name: "Facility Inspection", Description: "Infrastructure, asset condition & safety audit", Category: "Infrastructure", Tags: []string{"facility", "inspection", "infrastructure"}, FieldCount: 12, Version: "1.0"},
		{ID: "population_census", Name: "Population Census", Description: "National population & demographic audit", Category: "Demographics", Tags: []string{"census", "population", "national"}, FieldCount: 8, Version: "1.0"},
	}
}

// SeedDefaultTemplates inserts the built-in questionnaire templates if they
// don't already exist. This provides a starting point — templates are
// fully manageable via the API — not hardcoded in the frontend.
func SeedDefaultTemplates() error {
	if dbPool == nil {
		return nil
	}
	existing, err := ListTemplates("")
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	templates := buildEnterpriseTemplates()
	for _, t := range templates {
		if err := SaveTemplate(t); err != nil {
			return fmt.Errorf("seed %s: %w", t.ID, err)
		}
	}
	return nil
}

// buildEnterpriseTemplates constructs all 7 enterprise survey templates.
func buildEnterpriseTemplates() []Template {
	// ── 1. Community Health Survey ──────────────────────────────
	healthSchema := mustJSON(`{
  "sections": [
    {
      "id": "respondent_info",
      "title": "Respondent Information",
      "fields": [
        {"key":"respondent_name","label":"Full Name","type":"text","required":true},
        {"key":"respondent_age","label":"Age (years)","type":"number","required":true,"min":0,"max":120},
        {"key":"respondent_gender","label":"Gender","type":"select","required":true,"options":[
          {"value":"male","label":"Male"},{"value":"female","label":"Female"},{"value":"other","label":"Other / Prefer not to say"}
        ]},
        {"key":"respondent_phone","label":"Phone Number","type":"tel"},
        {"key":"household_size","label":"Household Size","type":"number","min":1,"max":30,"required":true},
        {"key":"district","label":"District","type":"select","required":true,"options":[
          {"value":"kampala","label":"Kampala"},{"value":"wakiso","label":"Wakiso"},
          {"value":"mukono","label":"Mukono"},{"value":"jinja","label":"Jinja"},
          {"value":"gulu","label":"Gulu"},{"value":"mbarara","label":"Mbarara"},
          {"value":"mbale","label":"Mbale"},{"value":"fort_portal","label":"Fort Portal"},
          {"value":"other","label":"Other"}
        ]},
        {"key":"village","label":"Village / Parish","type":"text","required":true},
        {"key":"gps_location","label":"GPS Location","type":"gps","required":true}
      ]
    },
    {
      "id": "health_status",
      "title": "Health Status",
      "fields": [
        {"key":"has_chronic_condition","label":"Do you have any chronic condition?","type":"select","options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]},
        {"key":"chronic_conditions","label":"Which chronic conditions?","type":"multiselect",
         "showWhen":{"field":"has_chronic_condition","value":"yes"},
         "options":[
           {"value":"diabetes","label":"Diabetes"},{"value":"hypertension","label":"Hypertension"},
           {"value":"asthma","label":"Asthma"},{"value":"hiv_aids","label":"HIV/AIDS"},
           {"value":"heart_disease","label":"Heart Disease"},{"value":"cancer","label":"Cancer"},
           {"value":"other","label":"Other"}
         ]},
        {"key":"has_insurance","label":"Do you have health insurance?","type":"select","options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]},
        {"key":"insurance_type","label":"Insurance type","type":"select",
         "showWhen":{"field":"has_insurance","value":"yes"},
         "options":[
           {"value":"national","label":"National Health Insurance"},{"value":"private","label":"Private"},
           {"value":"employer","label":"Employer-provided"},{"value":"community","label":"Community-based"}
         ]},
        {"key":"last_hospital_visit","label":"Last hospital visit","type":"date"},
        {"key":"visit_reason","label":"Reason for last visit","type":"select","options":[
          {"value":"checkup","label":"Routine checkup"},{"value":"illness","label":"Illness"},
          {"value":"injury","label":"Injury"},{"value":"maternity","label":"Maternity"},
          {"value":"followup","label":"Follow-up"}
        ]},
        {"key":"medication_count","label":"Number of current medications","type":"number","min":0,"max":20},
        {"key":"overall_health_rating","label":"Rate overall health (1=Poor, 5=Excellent)","type":"rating","required":true}
      ]
    },
    {
      "id": "water_sanitation",
      "title": "Water & Sanitation",
      "fields": [
        {"key":"water_source","label":"Primary water source","type":"select","required":true,"options":[
          {"value":"piped","label":"Piped water"},{"value":"borehole","label":"Borehole"},
          {"value":"protected_well","label":"Protected well"},{"value":"unprotected_well","label":"Unprotected well"},
          {"value":"surface","label":"Surface water (river/lake)"},{"value":"rainwater","label":"Rainwater"}
        ]},
        {"key":"water_treatment","label":"Do you treat water before drinking?","type":"select","options":[
          {"value":"boiling","label":"Yes - Boiling"},{"value":"chlorine","label":"Yes - Chlorine"},
          {"value":"filter","label":"Yes - Filter"},{"value":"solar","label":"Yes - Solar (SODIS)"},
          {"value":"no","label":"No treatment"}
        ]},
        {"key":"has_toilet","label":"Do you have a toilet facility?","type":"select","required":true,"options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]},
        {"key":"toilet_type","label":"Toilet type","type":"select",
         "showWhen":{"field":"has_toilet","value":"yes"},
         "options":[
           {"value":"flush","label":"Flush toilet"},{"value":"vip","label":"VIP latrine"},
           {"value":"pit","label":"Traditional pit latrine"},{"value":"composting","label":"Composting toilet"}
         ]},
        {"key":"handwashing_facility","label":"Handwashing facility with soap?","type":"select","options":[
          {"value":"yes_soap","label":"Yes - with soap"},{"value":"yes_no_soap","label":"Yes - without soap"},
          {"value":"no","label":"No"}
        ]},
        {"key":"solid_waste_disposal","label":"Solid waste disposal method","type":"select","options":[
          {"value":"collected","label":"Municipal collection"},{"value":"burning","label":"Burning"},
          {"value":"pit","label":"Open pit"},{"value":"composting","label":"Composting"},
          {"value":"dumping","label":"Open dumping"}
        ]}
      ]
    },
    {
      "id": "child_health",
      "title": "Child Health (Under 5)",
      "fields": [
        {"key":"has_children_under5","label":"Children under 5 in household?","type":"select","options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]},
        {"key":"children_under5_count","label":"Number of children under 5","type":"number",
         "showWhen":{"field":"has_children_under5","value":"yes"},"min":1,"max":10},
        {"key":"vaccination_complete","label":"All children fully vaccinated?","type":"select",
         "showWhen":{"field":"has_children_under5","value":"yes"},
         "options":[
           {"value":"yes","label":"Yes"},{"value":"no","label":"No"},{"value":"partial","label":"Partially"}
         ]},
        {"key":"malnutrition_signs","label":"Signs of malnutrition observed?","type":"select",
         "showWhen":{"field":"has_children_under5","value":"yes"},
         "options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]}
      ]
    },
    {
      "id": "feedback",
      "title": "Consent & Feedback",
      "fields": [
        {"key":"satisfaction_rating","label":"Rate overall satisfaction (1-5)","type":"rating","required":true},
        {"key":"comments","label":"Additional comments","type":"text"},
        {"key":"enumerator_signature","label":"Enumerator Signature","type":"signature","required":true},
        {"key":"consent","label":"Data use consent","type":"select","required":true,"options":[
          {"value":"yes","label":"Yes, I consent"},{"value":"no","label":"No"}
        ]}
      ]
    }
  ]
}`)

	// ── 2. Household Census ──────────────────────────────────────
	censusSchema := mustJSON(`{
  "sections": [
    {
      "id": "household_identification",
      "title": "Household Identification",
      "fields": [
        {"key":"ea_code","label":"Enumeration Area Code","type":"text","required":true},
        {"key":"household_number","label":"Household Number","type":"number","required":true},
        {"key":"head_name","label":"Head of Household","type":"text","required":true},
        {"key":"head_sex","label":"Sex of HH Head","type":"select","required":true,"options":[
          {"value":"male","label":"Male"},{"value":"female","label":"Female"}
        ]},
        {"key":"head_age","label":"Age of HH Head","type":"number","required":true,"min":15,"max":120},
        {"key":"head_education","label":"Education Level","type":"select","options":[
          {"value":"none","label":"None"},{"value":"primary","label":"Primary"},
          {"value":"secondary","label":"Secondary"},{"value":"tertiary","label":"Tertiary"},
          {"value":"vocational","label":"Vocational"}
        ]},
        {"key":"head_occupation","label":"Main Occupation","type":"select","options":[
          {"value":"farmer","label":"Farmer"},{"value":"trader","label":"Trader / Business"},
          {"value":"employed","label":"Salaried Employment"},{"value":"casual","label":"Casual Labour"},
          {"value":"unemployed","label":"Unemployed"},{"value":"other","label":"Other"}
        ]},
        {"key":"gps_coordinates","label":"GPS Location","type":"gps","required":true},
        {"key":"dwelling_photo","label":"Dwelling Photo","type":"photo"}
      ]
    },
    {
      "id": "household_members",
      "title": "Household Composition",
      "fields": [
        {"key":"total_members","label":"Total Household Members","type":"number","required":true,"min":1},
        {"key":"males_under5","label":"Males Under 5","type":"number","min":0},
        {"key":"females_under5","label":"Females Under 5","type":"number","min":0},
        {"key":"males_5_17","label":"Males 5–17","type":"number","min":0},
        {"key":"females_5_17","label":"Females 5–17","type":"number","min":0},
        {"key":"males_18_64","label":"Males 18–64","type":"number","min":0},
        {"key":"females_18_64","label":"Females 18–64","type":"number","min":0},
        {"key":"males_65plus","label":"Males 65+","type":"number","min":0},
        {"key":"females_65plus","label":"Females 65+","type":"number","min":0},
        {"key":"persons_with_disability","label":"Persons with Disability","type":"number","min":0}
      ]
    },
    {
      "id": "housing",
      "title": "Housing Conditions",
      "fields": [
        {"key":"dwelling_type","label":"Dwelling Type","type":"select","options":[
          {"value":"permanent","label":"Permanent"},{"value":"semi_permanent","label":"Semi-permanent"},
          {"value":"temporary","label":"Temporary"},{"value":"improvised","label":"Improvised"}
        ]},
        {"key":"roof_material","label":"Roof Material","type":"select","options":[
          {"value":"iron_sheets","label":"Iron Sheets"},{"value":"tiles","label":"Tiles"},
          {"value":"concrete","label":"Concrete / Slab"},{"value":"grass","label":"Grass / Thatch"},
          {"value":"other","label":"Other"}
        ]},
        {"key":"floor_material","label":"Floor Material","type":"select","options":[
          {"value":"cement","label":"Cement"},{"value":"tiles","label":"Tiles"},
          {"value":"earth","label":"Earth / Sand"},{"value":"wood","label":"Wood"}
        ]},
        {"key":"rooms_count","label":"Number of Rooms","type":"number","min":1},
        {"key":"has_electricity","label":"Electricity access?","type":"select","options":[
          {"value":"grid","label":"Yes - National grid"},{"value":"solar","label":"Yes - Solar"},
          {"value":"generator","label":"Yes - Generator"},{"value":"no","label":"No"}
        ]}
      ]
    },
    {
      "id": "socioeconomic",
      "title": "Socioeconomic",
      "fields": [
        {"key":"main_income_source","label":"Main income source","type":"select","options":[
          {"value":"farming","label":"Farming"},{"value":"business","label":"Business"},
          {"value":"employment","label":"Employment"},{"value":"remittances","label":"Remittances"},
          {"value":"transfers","label":"Social Transfers"},{"value":"other","label":"Other"}
        ]},
        {"key":"monthly_income_bracket","label":"Monthly income (UGX)","type":"select","options":[
          {"value":"below_100k","label":"Below 100,000"},{"value":"100k_300k","label":"100,000–300,000"},
          {"value":"300k_500k","label":"300,000–500,000"},{"value":"500k_1m","label":"500,000–1,000,000"},
          {"value":"above_1m","label":"Above 1,000,000"}
        ]},
        {"key":"food_insecurity_days","label":"Days with food insecurity (last 30 days)","type":"number","min":0,"max":30},
        {"key":"owns_mobile_phone","label":"Household owns mobile phone?","type":"select","options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]}
      ]
    },
    {
      "id": "consent_sign",
      "title": "Enumerator Declaration",
      "fields": [
        {"key":"enumerator_id","label":"Enumerator ID","type":"text","required":true},
        {"key":"interview_date","label":"Interview Date","type":"date","required":true},
        {"key":"respondent_consent","label":"Respondent consent obtained?","type":"select","required":true,"options":[
          {"value":"yes","label":"Yes"},{"value":"refused","label":"Refused"}
        ]},
        {"key":"enumerator_signature","label":"Enumerator Signature","type":"signature","required":true}
      ]
    }
  ]
}`)

	// ── 3. School Enrollment Survey ─────────────────────────────
	schoolSchema := mustJSON(`{
  "sections": [
    {
      "id": "school_info",
      "title": "School Information",
      "fields": [
        {"key":"school_emis","label":"School EMIS Code","type":"text","required":true},
        {"key":"school_name","label":"School Name","type":"text","required":true},
        {"key":"school_district","label":"District","type":"select","required":true,"options":[
          {"value":"kampala","label":"Kampala"},{"value":"wakiso","label":"Wakiso"},
          {"value":"mukono","label":"Mukono"},{"value":"jinja","label":"Jinja"},
          {"value":"gulu","label":"Gulu"},{"value":"mbarara","label":"Mbarara"},{"value":"other","label":"Other"}
        ]},
        {"key":"school_level","label":"School Level","type":"select","required":true,"options":[
          {"value":"nursery","label":"Nursery / Pre-Primary"},{"value":"primary","label":"Primary"},
          {"value":"secondary","label":"Secondary"},{"value":"tertiary","label":"Tertiary / University"}
        ]},
        {"key":"school_ownership","label":"Ownership","type":"select","options":[
          {"value":"government","label":"Government"},{"value":"private","label":"Private"},
          {"value":"community","label":"Community"},{"value":"religious","label":"Faith-based"}
        ]},
        {"key":"gps_location","label":"GPS Location","type":"gps","required":true},
        {"key":"school_photo","label":"School Photo","type":"photo"}
      ]
    },
    {
      "id": "enrollment_data",
      "title": "Enrollment Data",
      "fields": [
        {"key":"total_students","label":"Total Students Enrolled","type":"number","required":true,"min":0},
        {"key":"boys_enrolled","label":"Boys Enrolled","type":"number","required":true,"min":0},
        {"key":"girls_enrolled","label":"Girls Enrolled","type":"number","required":true,"min":0},
        {"key":"boys_attending","label":"Boys Attending (avg daily)","type":"number","min":0},
        {"key":"girls_attending","label":"Girls Attending (avg daily)","type":"number","min":0},
        {"key":"total_teachers","label":"Total Teachers","type":"number","required":true,"min":0},
        {"key":"qualified_teachers","label":"Qualified Teachers","type":"number","min":0},
        {"key":"dropout_rate","label":"Dropout Rate (%)","type":"number","min":0,"max":100}
      ]
    },
    {
      "id": "facilities",
      "title": "School Facilities",
      "fields": [
        {"key":"has_library","label":"Has Library?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"has_lab","label":"Has Science Lab?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"has_water","label":"Has Piped Water?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"has_electricity","label":"Has Electricity?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"has_internet","label":"Has Internet Access?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"classroom_count","label":"Number of Classrooms","type":"number","min":0},
        {"key":"latrine_count_boys","label":"Latrines for Boys","type":"number","min":0},
        {"key":"latrine_count_girls","label":"Latrines for Girls","type":"number","min":0},
        {"key":"has_school_meals","label":"School Meals Program?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"facility_rating","label":"Overall Facility Rating (1-5)","type":"rating"}
      ]
    },
    {
      "id": "sign_off",
      "title": "Sign-Off",
      "fields": [
        {"key":"data_collector","label":"Data Collector Name","type":"text","required":true},
        {"key":"collection_date","label":"Date","type":"date","required":true},
        {"key":"headteacher_consent","label":"Head Teacher Consented?","type":"select","required":true,"options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"enumerator_signature","label":"Signature","type":"signature","required":true}
      ]
    }
  ]
}`)

	// ── 4. Child Nutrition Survey ────────────────────────────────
	nutritionSchema := mustJSON(`{
  "sections": [
    {
      "id": "child_identification",
      "title": "Child Identification",
      "fields": [
        {"key":"child_id","label":"Child ID / Household Number","type":"text","required":true},
        {"key":"child_name","label":"Child Name","type":"text","required":true},
        {"key":"child_sex","label":"Child Sex","type":"select","required":true,"options":[
          {"value":"male","label":"Male"},{"value":"female","label":"Female"}
        ]},
        {"key":"child_dob","label":"Date of Birth","type":"date","required":true},
        {"key":"child_age_months","label":"Age in Months (calculated)","type":"number","min":0,"max":59},
        {"key":"caregiver_name","label":"Caregiver Name","type":"text","required":true},
        {"key":"gps_location","label":"GPS Location","type":"gps","required":true}
      ]
    },
    {
      "id": "anthropometry",
      "title": "Anthropometric Measurements",
      "fields": [
        {"key":"weight_kg","label":"Weight (kg)","type":"number","required":true,"min":0,"max":30},
        {"key":"height_cm","label":"Height / Length (cm)","type":"number","required":true,"min":30,"max":130},
        {"key":"measurement_position","label":"Measurement Position","type":"select","options":[
          {"value":"standing","label":"Standing"},{"value":"lying","label":"Lying (recumbent)"}
        ]},
        {"key":"muac_cm","label":"MUAC (cm)","type":"number","required":true,"min":5,"max":25},
        {"key":"oedema","label":"Bilateral Pitting Oedema?","type":"select","required":true,"options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]},
        {"key":"muac_category","label":"MUAC Category","type":"select","options":[
          {"value":"sam","label":"SAM (< 11.5 cm)"},{"value":"mam","label":"MAM (11.5 – 12.5 cm)"},
          {"value":"normal","label":"Normal (≥ 12.5 cm)"}
        ]}
      ]
    },
    {
      "id": "dietary_diversity",
      "title": "Dietary Diversity (24hr Recall)",
      "fields": [
        {"key":"ate_grains","label":"Cereals / Grains / Roots","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"ate_legumes","label":"Legumes / Nuts / Seeds","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"ate_dairy","label":"Dairy Products","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"ate_meat","label":"Meat / Fish / Poultry / Eggs","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"ate_vegetables","label":"Dark Green Leafy Vegetables","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"ate_vitA_veg","label":"Vitamin A-rich Vegetables / Fruits","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"meals_per_day","label":"Meals per Day","type":"number","min":0,"max":8},
        {"key":"breastfeeding","label":"Still Breastfeeding?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"},{"value":"na","label":"N/A (> 24 months)"}]}
      ]
    },
    {
      "id": "health_morbidity",
      "title": "Health & Morbidity",
      "fields": [
        {"key":"fever_2weeks","label":"Fever in last 2 weeks?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"diarrhea_2weeks","label":"Diarrhoea in last 2 weeks?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"ari_2weeks","label":"Acute respiratory infection?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"vitamin_a_supplement","label":"Received Vitamin A supplement last 6 months?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"},{"value":"unknown","label":"Unknown"}]},
        {"key":"deworming_last_6mo","label":"Dewormed in last 6 months?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]}
      ]
    },
    {
      "id": "sign_off",
      "title": "Sign-Off",
      "fields": [
        {"key":"measurer_name","label":"Measurer Name","type":"text","required":true},
        {"key":"measurement_date","label":"Date","type":"date","required":true},
        {"key":"enumerator_signature","label":"Signature","type":"signature","required":true}
      ]
    }
  ]
}`)

	// ── 5. Agriculture & Livelihood Survey ───────────────────────
	agricSchema := mustJSON(`{
  "sections": [
    {
      "id": "farmer_id",
      "title": "Farmer Identification",
      "fields": [
        {"key":"farmer_id","label":"Farmer ID","type":"text","required":true},
        {"key":"farmer_name","label":"Farmer Name","type":"text","required":true},
        {"key":"sex","label":"Sex","type":"select","required":true,"options":[{"value":"male","label":"Male"},{"value":"female","label":"Female"}]},
        {"key":"age","label":"Age","type":"number","required":true,"min":15,"max":100},
        {"key":"district","label":"District","type":"select","required":true,"options":[
          {"value":"kampala","label":"Kampala"},{"value":"jinja","label":"Jinja"},
          {"value":"gulu","label":"Gulu"},{"value":"mbarara","label":"Mbarara"},{"value":"other","label":"Other"}
        ]},
        {"key":"subcounty","label":"Sub-County / Parish","type":"text","required":true},
        {"key":"gps_farm_location","label":"Farm GPS Location","type":"gps","required":true},
        {"key":"farm_photo","label":"Farm Photo","type":"photo"}
      ]
    },
    {
      "id": "land_crops",
      "title": "Land & Crops",
      "fields": [
        {"key":"total_land_acres","label":"Total Land Cultivated (acres)","type":"number","required":true,"min":0},
        {"key":"land_ownership","label":"Land Ownership","type":"select","options":[
          {"value":"owned","label":"Owned"},{"value":"rented","label":"Rented"},
          {"value":"borrowed","label":"Borrowed"},{"value":"communal","label":"Communal"}
        ]},
        {"key":"main_crops","label":"Main Crops Grown","type":"multiselect","options":[
          {"value":"maize","label":"Maize"},{"value":"beans","label":"Beans"},
          {"value":"cassava","label":"Cassava"},{"value":"rice","label":"Rice"},
          {"value":"groundnuts","label":"Groundnuts"},{"value":"sorghum","label":"Sorghum"},
          {"value":"millet","label":"Millet"},{"value":"sweet_potato","label":"Sweet Potato"},
          {"value":"coffee","label":"Coffee"},{"value":"tea","label":"Tea"},
          {"value":"sugarcane","label":"Sugarcane"},{"value":"vegetables","label":"Vegetables"}
        ]},
        {"key":"crop_area_main_acres","label":"Area for Main Crop (acres)","type":"number","min":0},
        {"key":"expected_yield_kg","label":"Expected Yield (kg)","type":"number","min":0},
        {"key":"irrigation_used","label":"Irrigation used?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"improved_seeds","label":"Using improved/certified seeds?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]}
      ]
    },
    {
      "id": "inputs_finance",
      "title": "Inputs & Finance",
      "fields": [
        {"key":"uses_fertilizer","label":"Uses fertilizer?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"fertilizer_type","label":"Fertilizer type","type":"select","showWhen":{"field":"uses_fertilizer","value":"yes"},"options":[
          {"value":"organic","label":"Organic / compost"},{"value":"inorganic","label":"Inorganic (chemical)"},
          {"value":"both","label":"Both"}
        ]},
        {"key":"uses_pesticides","label":"Uses pesticides?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"extension_services","label":"Received extension services last season?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"has_credit","label":"Has access to agricultural credit?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"credit_source","label":"Credit source","type":"select","showWhen":{"field":"has_credit","value":"yes"},"options":[
          {"value":"bank","label":"Commercial bank"},{"value":"sacco","label":"SACCO / MFI"},
          {"value":"group","label":"Farmer group"},{"value":"ngo","label":"NGO program"},
          {"value":"other","label":"Other"}
        ]}
      ]
    },
    {
      "id": "livestock_market",
      "title": "Livestock & Markets",
      "fields": [
        {"key":"keeps_livestock","label":"Keeps livestock?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"livestock_types","label":"Livestock types","type":"multiselect","showWhen":{"field":"keeps_livestock","value":"yes"},"options":[
          {"value":"cattle","label":"Cattle"},{"value":"goats","label":"Goats"},
          {"value":"sheep","label":"Sheep"},{"value":"pigs","label":"Pigs"},
          {"value":"poultry","label":"Poultry"},{"value":"fish","label":"Fish (aquaculture)"}
        ]},
        {"key":"sells_produce","label":"Sells produce?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"market_distance_km","label":"Distance to nearest market (km)","type":"number","min":0},
        {"key":"storage_facility","label":"Has post-harvest storage facility?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"post_harvest_loss_pct","label":"Estimated post-harvest loss (%)","type":"number","min":0,"max":100},
        {"key":"climate_challenges","label":"Main climate challenge","type":"select","options":[
          {"value":"drought","label":"Drought"},{"value":"floods","label":"Floods"},
          {"value":"pests","label":"Pests & diseases"},{"value":"none","label":"None"}
        ]}
      ]
    },
    {
      "id": "sign_off",
      "title": "Sign-Off",
      "fields": [
        {"key":"enumerator_id","label":"Enumerator ID","type":"text","required":true},
        {"key":"survey_date","label":"Survey Date","type":"date","required":true},
        {"key":"enumerator_signature","label":"Enumerator Signature","type":"signature","required":true}
      ]
    }
  ]
}`)

	// ── 6. Water & Sanitation Survey (WASH) ─────────────────────
	washSchema := mustJSON(`{
  "sections": [
    {
      "id": "location_id",
      "title": "Location & Identification",
      "fields": [
        {"key":"site_id","label":"Site ID","type":"text","required":true},
        {"key":"facility_name","label":"Facility / Point Name","type":"text","required":true},
        {"key":"facility_type","label":"Facility Type","type":"select","required":true,"options":[
          {"value":"borehole","label":"Borehole"},{"value":"protected_spring","label":"Protected Spring"},
          {"value":"public_tap","label":"Public Standpipe / Tap"},{"value":"dam","label":"Dam / Reservoir"},
          {"value":"rain_harvest","label":"Rainwater Harvesting"},{"value":"other","label":"Other"}
        ]},
        {"key":"district","label":"District","type":"select","required":true,"options":[
          {"value":"kampala","label":"Kampala"},{"value":"wakiso","label":"Wakiso"},{"value":"other","label":"Other"}
        ]},
        {"key":"gps_location","label":"GPS Location","type":"gps","required":true},
        {"key":"facility_photo","label":"Facility Photo","type":"photo"}
      ]
    },
    {
      "id": "functionality",
      "title": "Functionality Status",
      "fields": [
        {"key":"is_functional","label":"Is the water point functional?","type":"select","required":true,"options":[
          {"value":"yes","label":"Yes - fully functional"},
          {"value":"partial","label":"Partial - limited output"},
          {"value":"no","label":"No - non-functional"}
        ]},
        {"key":"non_functional_reason","label":"Reason non-functional","type":"select",
         "showWhen":{"field":"is_functional","value":"no"},
         "options":[
           {"value":"broken_pump","label":"Broken pump"},{"value":"no_water","label":"No water"},
           {"value":"vandalism","label":"Vandalism"},{"value":"funding","label":"Lack of funding"},
           {"value":"other","label":"Other"}
         ]},
        {"key":"last_rehabilitation","label":"Date of last rehabilitation","type":"date"},
        {"key":"serves_households","label":"Estimated households served","type":"number","min":0},
        {"key":"queue_time_minutes","label":"Average queue time (minutes)","type":"number","min":0}
      ]
    },
    {
      "id": "water_quality",
      "title": "Water Quality",
      "fields": [
        {"key":"colour","label":"Water colour","type":"select","options":[
          {"value":"clear","label":"Clear"},{"value":"turbid","label":"Turbid"},{"value":"coloured","label":"Coloured"}
        ]},
        {"key":"odour","label":"Odour","type":"select","options":[
          {"value":"none","label":"None"},{"value":"slight","label":"Slight"},{"value":"strong","label":"Strong"}
        ]},
        {"key":"chlorine_residual_mg","label":"Free chlorine residual (mg/L)","type":"number","min":0},
        {"key":"turbidity_ntu","label":"Turbidity (NTU)","type":"number","min":0},
        {"key":"fc_ecoli","label":"E. coli contamination?","type":"select","options":[
          {"value":"none","label":"None detected"},{"value":"low","label":"Low (< 10 CFU/100mL)"},
          {"value":"medium","label":"Medium"},{"value":"high","label":"High"}
        ]}
      ]
    },
    {
      "id": "sanitation_hygiene",
      "title": "Sanitation & Hygiene",
      "fields": [
        {"key":"latrines_count","label":"Number of latrine stances","type":"number","min":0},
        {"key":"latrines_functional","label":"Functional latrine stances","type":"number","min":0},
        {"key":"latrine_cleanliness","label":"Latrine cleanliness","type":"rating"},
        {"key":"handwashing_station","label":"Handwashing station present?","type":"select","options":[
          {"value":"yes_soap","label":"Yes - with soap"},
          {"value":"yes_no_soap","label":"Yes - without soap"},
          {"value":"no","label":"No"}
        ]},
        {"key":"open_defecation","label":"Open defecation observed?","type":"select","options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]},
        {"key":"hygiene_promotion","label":"Hygiene promotion activities in last 3 months?","type":"select","options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]}
      ]
    },
    {
      "id": "management",
      "title": "Management & Governance",
      "fields": [
        {"key":"wuc_exists","label":"Water User Committee exists?","type":"select","options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]},
        {"key":"wuc_functional","label":"WUC is functional?","type":"select",
         "showWhen":{"field":"wuc_exists","value":"yes"},
         "options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"tariff_collected","label":"User tariff collected?","type":"select","options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]},
        {"key":"maintenance_fund","label":"Maintenance fund available?","type":"select","options":[
          {"value":"yes","label":"Yes"},{"value":"no","label":"No"}
        ]},
        {"key":"enumerator_signature","label":"Enumerator Signature","type":"signature","required":true}
      ]
    }
  ]
}`)

	// ── 7. Market Price Survey ───────────────────────────────────
	marketSchema := mustJSON(`{
  "sections": [
    {
      "id": "market_info",
      "title": "Market Information",
      "fields": [
        {"key":"market_id","label":"Market ID / Code","type":"text","required":true},
        {"key":"market_name","label":"Market Name","type":"text","required":true},
        {"key":"district","label":"District","type":"select","required":true,"options":[
          {"value":"kampala","label":"Kampala"},{"value":"jinja","label":"Jinja"},
          {"value":"gulu","label":"Gulu"},{"value":"mbarara","label":"Mbarara"},{"value":"other","label":"Other"}
        ]},
        {"key":"market_day","label":"Survey Date","type":"date","required":true},
        {"key":"gps_location","label":"Market GPS Location","type":"gps","required":true},
        {"key":"market_photo","label":"Market Photo","type":"photo"}
      ]
    },
    {
      "id": "staple_prices",
      "title": "Staple Food Prices",
      "fields": [
        {"key":"maize_grain_kg","label":"Maize (dry grain) — price per kg (UGX)","type":"number","min":0},
        {"key":"maize_flour_kg","label":"Maize flour — price per kg (UGX)","type":"number","min":0},
        {"key":"beans_kg","label":"Beans — price per kg (UGX)","type":"number","min":0},
        {"key":"rice_kg","label":"Rice — price per kg (UGX)","type":"number","min":0},
        {"key":"cassava_fresh_kg","label":"Fresh cassava — price per kg (UGX)","type":"number","min":0},
        {"key":"groundnuts_kg","label":"Groundnuts — price per kg (UGX)","type":"number","min":0},
        {"key":"sorghum_kg","label":"Sorghum — price per kg (UGX)","type":"number","min":0},
        {"key":"cooking_oil_litre","label":"Cooking oil — price per litre (UGX)","type":"number","min":0},
        {"key":"salt_kg","label":"Salt — price per kg (UGX)","type":"number","min":0}
      ]
    },
    {
      "id": "protein_vegetables",
      "title": "Protein & Vegetables",
      "fields": [
        {"key":"beef_kg","label":"Beef — price per kg (UGX)","type":"number","min":0},
        {"key":"chicken_kg","label":"Chicken — price per kg (UGX)","type":"number","min":0},
        {"key":"fish_mukene_kg","label":"Fish (mukene/dagaa) — price per kg (UGX)","type":"number","min":0},
        {"key":"eggs_tray","label":"Eggs — price per tray (UGX)","type":"number","min":0},
        {"key":"tomatoes_kg","label":"Tomatoes — price per kg (UGX)","type":"number","min":0},
        {"key":"onions_kg","label":"Onions — price per kg (UGX)","type":"number","min":0},
        {"key":"cabbages_each","label":"Cabbage — price each (UGX)","type":"number","min":0}
      ]
    },
    {
      "id": "market_conditions",
      "title": "Market Conditions",
      "fields": [
        {"key":"availability_rating","label":"Overall food availability (1=Very poor, 5=Excellent)","type":"rating","required":true},
        {"key":"price_trend","label":"Price trend vs last month","type":"select","options":[
          {"value":"increased","label":"Increased significantly"},{"value":"slight_increase","label":"Slight increase"},
          {"value":"stable","label":"Stable"},{"value":"decreased","label":"Decreased"}
        ]},
        {"key":"supply_constraint","label":"Main supply constraint","type":"select","options":[
          {"value":"none","label":"None"},{"value":"transport","label":"Transport issues"},
          {"value":"seasonal","label":"Seasonal shortage"},{"value":"security","label":"Security"},
          {"value":"weather","label":"Weather / drought"},{"value":"other","label":"Other"}
        ]},
        {"key":"price_collector","label":"Price Collector Name","type":"text","required":true},
        {"key":"enumerator_signature","label":"Signature","type":"signature","required":true}
      ]
    }
  ]
}`)

	// ── 8. Facility Inspection ───────────────────────────────────
	facilitySchema := mustJSON(`{
  "sections": [
    {
      "id": "inspection_details",
      "title": "Inspection Details",
      "fields": [
        {"key":"facility_id","label":"Facility ID","type":"text","required":true},
        {"key":"facility_name","label":"Facility Name","type":"text","required":true},
        {"key":"facility_type","label":"Facility Type","type":"select","options":[
          {"value":"health","label":"Health Centre"},{"value":"school","label":"School"},
          {"value":"water","label":"Water Point"},{"value":"road","label":"Road / Bridge"},
          {"value":"market","label":"Market"},{"value":"admin","label":"Administrative Office"},
          {"value":"other","label":"Other"}
        ]},
        {"key":"inspection_date","label":"Inspection Date","type":"date","required":true},
        {"key":"gps_location","label":"GPS Location","type":"gps","required":true},
        {"key":"facility_photo","label":"Facility Photo","type":"photo"},
        {"key":"overall_rating","label":"Overall Condition Rating (1=Very poor, 5=Excellent)","type":"rating","required":true}
      ]
    },
    {
      "id": "infrastructure",
      "title": "Infrastructure Assessment",
      "fields": [
        {"key":"building_condition","label":"Building structural condition","type":"select","options":[
          {"value":"good","label":"Good"},{"value":"fair","label":"Fair"},
          {"value":"poor","label":"Poor"},{"value":"critical","label":"Critical — unsafe"}
        ]},
        {"key":"roof_condition","label":"Roof condition","type":"select","options":[
          {"value":"good","label":"Good"},{"value":"leaking","label":"Leaking"},{"value":"damaged","label":"Damaged"}
        ]},
        {"key":"water_supply","label":"Water supply available?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"electricity_supply","label":"Electricity available?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"sanitation_functional","label":"Sanitation functional?","type":"select","options":[{"value":"yes","label":"Yes"},{"value":"no","label":"No"}]},
        {"key":"maintenance_notes","label":"Maintenance issues observed","type":"text"},
        {"key":"inspector_signature","label":"Inspector Signature","type":"signature","required":true}
      ]
    }
  ]
}`)

	// ── 9. Population Census ─────────────────────────────────────
	popCensusSchema := mustJSON(`{
  "sections": [
    {
      "id": "demographics",
      "title": "Household Demographics",
      "fields": [
        {"key":"ea_code","label":"EA Code","type":"text","required":true},
        {"key":"head_name","label":"Head of Household Name","type":"text","required":true},
        {"key":"total_members","label":"Total Household Members","type":"number","required":true,"min":1},
        {"key":"gps_loc","label":"GPS Coordinates","type":"gps","required":true},
        {"key":"enumerator_signature","label":"Enumerator Signature","type":"signature","required":true}
      ]
    }
  ]
}`)

	return []Template{
		{ID: "community_health_survey", Name: "Community Health Survey", Description: "Household health, water & sanitation assessment", Version: "1.0", Category: "Health", Rating: 5, DownloadCount: 156, Schema: json.RawMessage(healthSchema), Status: "active", CreatedBy: "system", TenantID: "default"},
		{ID: "household_census", Name: "Household Census", Description: "Full household demographic & socioeconomic census", Version: "1.0", Category: "Demographics", Rating: 5, DownloadCount: 412, Schema: json.RawMessage(censusSchema), Status: "active", CreatedBy: "system", TenantID: "default"},
		{ID: "school_enrollment_survey", Name: "School Enrollment Survey", Description: "School enrollment, attendance & infrastructure assessment", Version: "1.0", Category: "Education", Rating: 4, DownloadCount: 88, Schema: json.RawMessage(schoolSchema), Status: "active", CreatedBy: "system", TenantID: "default"},
		{ID: "nutrition_survey", Name: "Child Nutrition Survey", Description: "Under-5 anthropometric & dietary diversity assessment", Version: "1.0", Category: "Nutrition", Rating: 5, DownloadCount: 92, Schema: json.RawMessage(nutritionSchema), Status: "active", CreatedBy: "system", TenantID: "default"},
		{ID: "agriculture_survey", Name: "Agriculture & Livelihood Survey", Description: "Crop production, input use & market access", Version: "1.0", Category: "Agriculture", Rating: 4, DownloadCount: 104, Schema: json.RawMessage(agricSchema), Status: "active", CreatedBy: "system", TenantID: "default"},
		{ID: "water_sanitation_survey", Name: "Water & Sanitation (WASH) Survey", Description: "WASH infrastructure, access & hygiene practices", Version: "1.0", Category: "WASH", Rating: 5, DownloadCount: 75, Schema: json.RawMessage(washSchema), Status: "active", CreatedBy: "system", TenantID: "default"},
		{ID: "market_inspection", Name: "Market Price Survey", Description: "Commodity prices, availability & market conditions", Version: "1.0", Category: "Markets", Rating: 4, DownloadCount: 45, Schema: json.RawMessage(marketSchema), Status: "active", CreatedBy: "system", TenantID: "default"},
		{ID: "facility_inspection", Name: "Facility Inspection", Description: "Infrastructure, asset condition & safety audit", Version: "1.0", Category: "Infrastructure", Rating: 4, DownloadCount: 30, Schema: json.RawMessage(facilitySchema), Status: "active", CreatedBy: "system", TenantID: "default"},
		{ID: "population_census", Name: "Population Census", Description: "National population & demographic audit", Version: "1.0", Category: "Demographics", Rating: 5, DownloadCount: 520, Schema: json.RawMessage(popCensusSchema), Status: "active", CreatedBy: "system", TenantID: "default"},
	}
}

// mustJSON panics if json is invalid — only used in compile-time seed data.
func mustJSON(s string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic("invalid seed JSON: " + err.Error())
	}
	return s
}

// ── XLSForm / ODK Export ──────────────────────────────────────

// ExportTemplateAsXLSForm converts a template schema to XLSForm TSV format.
// XLSForm uses three sheets: survey, choices, settings.
func ExportTemplateAsXLSForm(t *Template) (string, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal(t.Schema, &schema); err != nil {
		return "", fmt.Errorf("parse schema: %w", err)
	}
	sections, _ := schema["sections"].([]interface{})

	var sb strings.Builder
	// ── survey sheet ──
	sb.WriteString("=== survey ===\n")
	sb.WriteString("type\tname\tlabel\trequired\trelevant\tconstraint\tappearance\n")

	for _, sec := range sections {
		s, _ := sec.(map[string]interface{})
		secID, _ := s["id"].(string)
		secTitle, _ := s["title"].(string)
		sb.WriteString(fmt.Sprintf("begin_group\t%s\t%s\t\t\t\n", secID, secTitle))

		fields, _ := s["fields"].([]interface{})
		for _, fld := range fields {
			f, _ := fld.(map[string]interface{})
			key, _ := f["key"].(string)
			label, _ := f["label"].(string)
			typ, _ := f["type"].(string)
			req, _ := f["required"].(bool)
			showWhen, hasCondition := f["showWhen"].(map[string]interface{})

			reqStr := ""
			if req {
				reqStr = "yes"
			}
			relevantStr := ""
			if hasCondition {
				condField, _ := showWhen["field"].(string)
				condVal, _ := showWhen["value"].(string)
				if condField != "" && condVal != "" {
					relevantStr = fmt.Sprintf("${%s}='%s'", condField, condVal)
				}
			}

			xlsType := mapTypeToXLS(typ, f)
			sb.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t\t\n", xlsType, key, label, reqStr, relevantStr))
		}
		sb.WriteString(fmt.Sprintf("end_group\t%s\t\t\t\t\n", secID))
	}

	// ── choices sheet ──
	sb.WriteString("\n=== choices ===\n")
	sb.WriteString("list_name\tname\tlabel\n")
	for _, sec := range sections {
		s, _ := sec.(map[string]interface{})
		fields, _ := s["fields"].([]interface{})
		for _, fld := range fields {
			f, _ := fld.(map[string]interface{})
			key, _ := f["key"].(string)
			options, _ := f["options"].([]interface{})
			for _, opt := range options {
				o, _ := opt.(map[string]interface{})
				val, _ := o["value"].(string)
				lbl, _ := o["label"].(string)
				sb.WriteString(fmt.Sprintf("%s\t%s\t%s\n", key, val, lbl))
			}
		}
	}

	// ── settings sheet ──
	sb.WriteString("\n=== settings ===\n")
	sb.WriteString("form_title\tform_id\tversion\n")
	sb.WriteString(fmt.Sprintf("%s\t%s\t%s\n", t.Name, t.ID, t.Version))

	return sb.String(), nil
}

// ExportTemplateAsODKXML converts a template schema to ODK XForm XML format.
func ExportTemplateAsODKXML(t *Template) (string, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal(t.Schema, &schema); err != nil {
		return "", fmt.Errorf("parse schema: %w", err)
	}
	sections, _ := schema["sections"].([]interface{})

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(fmt.Sprintf(`<h:html xmlns:h="http://www.w3.org/1999/xhtml"
        xmlns:ev="http://www.w3.org/2001/xml-events"
        xmlns:xsd="http://www.w3.org/2001/XMLSchema"
        xmlns:jr="http://openrosa.org/javarosa"
        xmlns="http://www.w3.org/2002/xforms">
  <h:head>
    <h:title>%s</h:title>
    <model>
      <instance>
        <data id="%s" version="%s">
`, t.Name, t.ID, t.Version))

	// instance
	for _, sec := range sections {
		s, _ := sec.(map[string]interface{})
		secID, _ := s["id"].(string)
		sb.WriteString(fmt.Sprintf("          <%s>\n", secID))
		fields, _ := s["fields"].([]interface{})
		for _, fld := range fields {
			f, _ := fld.(map[string]interface{})
			key, _ := f["key"].(string)
			sb.WriteString(fmt.Sprintf("            <%s/>\n", key))
		}
		sb.WriteString(fmt.Sprintf("          </%s>\n", secID))
	}

	sb.WriteString(`        </data>
      </instance>
`)

	// binds
	for _, sec := range sections {
		s, _ := sec.(map[string]interface{})
		secID, _ := s["id"].(string)
		fields, _ := s["fields"].([]interface{})
		for _, fld := range fields {
			f, _ := fld.(map[string]interface{})
			key, _ := f["key"].(string)
			typ, _ := f["type"].(string)
			req, _ := f["required"].(bool)
			reqStr := ""
			if req {
				reqStr = ` required="true()"`
			}
			sb.WriteString(fmt.Sprintf(`      <bind nodeset="/data/%s/%s" type="%s"%s/>`+"\n", secID, key, mapTypeToXSD(typ), reqStr))
		}
	}

	sb.WriteString(`    </model>
  </h:head>
  <h:body>
`)

	// body
	for _, sec := range sections {
		s, _ := sec.(map[string]interface{})
		secID, _ := s["id"].(string)
		secTitle, _ := s["title"].(string)
		sb.WriteString(fmt.Sprintf("    <group ref=\"/data/%s\">\n", secID))
		sb.WriteString(fmt.Sprintf("      <label>%s</label>\n", secTitle))

		fields, _ := s["fields"].([]interface{})
		for _, fld := range fields {
			f, _ := fld.(map[string]interface{})
			key, _ := f["key"].(string)
			label, _ := f["label"].(string)
			typ, _ := f["type"].(string)
			options, _ := f["options"].([]interface{})

			if typ == "select" || typ == "multiselect" {
				inputTag := "select1"
				if typ == "multiselect" {
					inputTag = "select"
				}
				sb.WriteString(fmt.Sprintf("      <%s ref=\"/data/%s/%s\">\n        <label>%s</label>\n", inputTag, secID, key, label))
				for _, opt := range options {
					o, _ := opt.(map[string]interface{})
					val, _ := o["value"].(string)
					lbl, _ := o["label"].(string)
					sb.WriteString(fmt.Sprintf("        <item><label>%s</label><value>%s</value></item>\n", lbl, val))
				}
				sb.WriteString(fmt.Sprintf("      </%s>\n", inputTag))
			} else if typ == "gps" {
				sb.WriteString(fmt.Sprintf("      <input ref=\"/data/%s/%s\" appearance=\"maps\">\n        <label>%s</label>\n      </input>\n", secID, key, label))
			} else if typ == "photo" {
				sb.WriteString(fmt.Sprintf("      <upload ref=\"/data/%s/%s\" mediatype=\"image/*\">\n        <label>%s</label>\n      </upload>\n", secID, key, label))
			} else if typ == "signature" {
				sb.WriteString(fmt.Sprintf("      <upload ref=\"/data/%s/%s\" mediatype=\"image/*\" appearance=\"signature\">\n        <label>%s</label>\n      </upload>\n", secID, key, label))
			} else {
				sb.WriteString(fmt.Sprintf("      <input ref=\"/data/%s/%s\">\n        <label>%s</label>\n      </input>\n", secID, key, label))
			}
		}
		sb.WriteString("    </group>\n")
	}

	sb.WriteString("  </h:body>\n</h:html>")
	return sb.String(), nil
}

// ImportTemplateFromJSON parses a raw JSON schema body and returns a Template.
func ImportTemplateFromJSON(id, name, description string, schemaBytes []byte) (*Template, error) {
	// validate that it is valid JSON and has a sections array
	var schemaCheck map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaCheck); err != nil {
		return nil, fmt.Errorf("invalid JSON schema: %w", err)
	}
	if _, ok := schemaCheck["sections"]; !ok {
		return nil, fmt.Errorf("schema must contain a 'sections' array")
	}
	return &Template{
		ID:          id,
		Name:        name,
		Description: description,
		Version:     "1.0",
		Schema:      json.RawMessage(schemaBytes),
		Status:      "draft",
		CreatedBy:   "import",
		TenantID:    "default",
	}, nil
}

// mapTypeToXLS maps internal field types to XLSForm types.
func mapTypeToXLS(typ string, f map[string]interface{}) string {
	switch typ {
	case "text", "tel", "email":
		return "text"
	case "number":
		return "integer"
	case "date":
		return "date"
	case "select":
		key, _ := f["key"].(string)
		return "select_one " + key
	case "multiselect":
		key, _ := f["key"].(string)
		return "select_multiple " + key
	case "gps":
		return "geopoint"
	case "photo":
		return "image"
	case "signature":
		return "image"
	case "rating":
		return "integer"
	default:
		return "text"
	}
}

// mapTypeToXSD maps internal field types to XForm XSD types.
func mapTypeToXSD(typ string) string {
	switch typ {
	case "number", "rating":
		return "int"
	case "date":
		return "date"
	case "gps":
		return "geopoint"
	default:
		return "string"
	}
}

// ── Template Database Operations ──────────────────────────────

// GetTemplateSchema returns the parsed schema for a template.
func GetTemplateSchema(id string) (map[string]interface{}, error) {
	t, err := GetTemplate(id)
	if err != nil {
		return nil, err
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(t.Schema, &schema); err != nil {
		return nil, err
	}
	return schema, nil
}

// TemplateExists checks if a template ID exists.
func TemplateExists(id string) (bool, error) {
	if dbPool == nil {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var cnt int
	err := dbPool.QueryRow(ctx, `SELECT COUNT(1) FROM templates WHERE id=$1`, id).Scan(&cnt)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// GetPool returns the database pool (for use in handlers).
func GetPool() *pgxpool.Pool {
	return dbPool
}

// SaveTemplateVersion saves a new version snapshot to template_versions.
func SaveTemplateVersion(templateID, version, changeNote, changedBy string, schema []byte) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO template_versions (template_id, version, schema, changed_by, change_note)
		VALUES ($1,$2,$3,$4,$5)`,
		templateID, version, schema, changedBy, changeNote)
	if err != nil {
		return fmt.Errorf("save template version: %w", err)
	}
	return nil
}

// ListTemplateVersions returns version history for a template.
func ListTemplateVersions(templateID string) ([]TemplateVersion, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT id, template_id, version, schema, COALESCE(changed_by,''), COALESCE(change_note,''), created_at
		FROM template_versions WHERE template_id=$1 ORDER BY created_at DESC`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TemplateVersion
	for rows.Next() {
		var v TemplateVersion
		if err := rows.Scan(&v.ID, &v.TemplateID, &v.Version, &v.Schema, &v.ChangedBy, &v.ChangeNote, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// ShareTemplate marks a template as shared and creates a copy in target tenant.
func ShareTemplate(templateID, newID, targetTenantID string) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `UPDATE templates SET is_shared=true, parent_id=$2 WHERE id=$1`,
		templateID, templateID)
	if err != nil {
		return fmt.Errorf("share template: %w", err)
	}
	if newID != "" && targetTenantID != "" {
		var schema []byte
		var name, desc, version string
		err := dbPool.QueryRow(ctx, `SELECT name, description, version, schema FROM templates WHERE id=$1`, templateID).Scan(
			&name, &desc, &version, &schema)
		if err != nil {
			return err
		}
		_, err = dbPool.Exec(ctx, `INSERT INTO templates (id, name, description, version, schema, status, tenant_id, is_shared, parent_id)
			VALUES ($1,$2,$3,$4,$5,'active',$6,true,$7)`,
			newID, name, desc, version, schema, targetTenantID, templateID)
		if err != nil {
			return fmt.Errorf("create shared copy: %w", err)
		}
	}
	return nil
}

// UnshareTemplate removes shared status.
func UnshareTemplate(templateID string) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `UPDATE templates SET is_shared=false WHERE id=$1`, templateID)
	if err != nil {
		return fmt.Errorf("unshare template: %w", err)
	}
	return nil
}

// CloneTemplate duplicates an existing template with a new ID and name.
func CloneTemplate(sourceID, newID, newName string) (*Template, error) {
	tmpl, err := GetTemplate(sourceID)
	if err != nil {
		return nil, fmt.Errorf("source template not found: %w", err)
	}
	newTmpl := Template{
		ID:          newID,
		Name:        newName,
		Description: tmpl.Description + " (Copy)",
		Version:     "1.0",
		Schema:      tmpl.Schema,
		Status:      "draft",
		CreatedBy:   "system",
		TenantID:    tmpl.TenantID,
	}
	if err := SaveTemplate(newTmpl); err != nil {
		return nil, fmt.Errorf("save cloned template: %w", err)
	}
	return &newTmpl, nil
}
