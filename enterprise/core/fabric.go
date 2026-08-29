package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Fabric Data Store (In-Memory synchronized + DB Fallback) ───────

var fabricStore = struct {
	sync.RWMutex
	objects      map[string]CanonicalObject
	relations    map[string]FabricRelationship
	knowledge    map[string]GovernedKnowledgeItem
	catalogue    map[string]CatalogueDataset
	dictionary   map[string]DictionaryVariable
	mappings     map[string]SemanticConceptMapping
	applications map[string]RegisteredApplication
	memory       map[string]InstitutionalDecisionMemory
}{
	objects:      make(map[string]CanonicalObject),
	relations:    make(map[string]FabricRelationship),
	knowledge:    make(map[string]GovernedKnowledgeItem),
	catalogue:    make(map[string]CatalogueDataset),
	dictionary:   make(map[string]DictionaryVariable),
	mappings:     make(map[string]SemanticConceptMapping),
	applications: make(map[string]RegisteredApplication),
	memory:       make(map[string]InstitutionalDecisionMemory),
}

func bootstrapFabric() {
	fabricStore.Lock()
	defer fabricStore.Unlock()

	// 1. Bootstrap default registered applications
	apps := []RegisteredApplication{
		{
			ApplicationID:    "pms",
			Name:             "Project Management System",
			Version:          "2.4.0",
			Owner:            "Planning Directorate",
			APIURL:           getEnv("PMS_API_URL", "http://localhost:8091"),
			UIURL:            "http://localhost:3010",
			HealthEndpoint:   "/api/health",
			Capabilities:     []string{"projects", "milestones", "work_plans", "budget_tracking"},
			SupportedObjects: []string{"project", "milestone", "deliverable"},
			SupportedEvents:  []string{"project.created", "project.updated", "milestone.reached", "project.completed"},
			AuthMethod:       "bearer_jwt",
			Status:           "active",
			LastHeartbeat:    nowUTC(),
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
		},
		{
			ApplicationID:    "rms",
			Name:             "Research Management System",
			Version:          "2.1.0",
			Owner:            "Research & Ethics Board",
			APIURL:           getEnv("RMS_API_URL", "http://localhost:8092"),
			UIURL:            "http://localhost:3011",
			HealthEndpoint:   "/api/health",
			Capabilities:     []string{"research_protocols", "ethics_review", "grants", "publications"},
			SupportedObjects: []string{"research", "protocol", "grant", "publication"},
			SupportedEvents:  []string{"research.proposed", "ethics.approved", "research.completed"},
			AuthMethod:       "bearer_jwt",
			Status:           "active",
			LastHeartbeat:    nowUTC(),
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
		},
		{
			ApplicationID:    "statcollect",
			Name:             "StatCollect Field System",
			Version:          "3.0.0",
			Owner:            "Field Operations",
			APIURL:           getEnv("STATCOLLECT_API_URL", "http://localhost:3006"),
			UIURL:            "http://localhost:3006",
			HealthEndpoint:   "/api/health",
			Capabilities:     []string{"field_surveys", "mobile_forms", "gps_collection", "offline_sync"},
			SupportedObjects: []string{"survey", "submission", "facility", "enumerator"},
			SupportedEvents:  []string{"survey.created", "submission.received", "facility.verified"},
			AuthMethod:       "bearer_jwt",
			Status:           "active",
			LastHeartbeat:    nowUTC(),
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
		},
		{
			ApplicationID:    "statgovernance",
			Name:             "StatGovernance Risk & Compliance",
			Version:          "8.0.0",
			Owner:            "Governance & Legal Affairs",
			APIURL:           getEnv("STATGOVERNANCE_API_URL", "http://localhost:8093"),
			UIURL:            "http://localhost:3012",
			HealthEndpoint:   "/health",
			Capabilities:     []string{"risk_register", "controls_library", "audit_findings", "policies"},
			SupportedObjects: []string{"risk", "control", "finding", "policy", "obligation"},
			SupportedEvents:  []string{"risk.identified", "control.tested", "audit.finding.opened"},
			AuthMethod:       "bearer_jwt",
			Status:           "active",
			LastHeartbeat:    nowUTC(),
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
		},
		{
			ApplicationID:    "statchat",
			Name:             "StatChat Enterprise Messaging",
			Version:          "1.8.0",
			Owner:            "Communication Services",
			APIURL:           getEnv("STATCHAT_API_URL", "http://localhost:4000"),
			UIURL:            "http://localhost:3009",
			HealthEndpoint:   "/health",
			Capabilities:     []string{"channels", "direct_messaging", "object_threads", "incident_rooms"},
			SupportedObjects: []string{"conversation", "message", "channel"},
			SupportedEvents:  []string{"statchat.message.posted", "channel.created"},
			AuthMethod:       "bearer_jwt",
			Status:           "active",
			LastHeartbeat:    nowUTC(),
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
		},
		{
			ApplicationID:    "helpdesk",
			Name:             "Enterprise HelpDesk Operations",
			Version:          "1.5.0",
			Owner:            "IT & Operations",
			APIURL:           getEnv("HELPDESK_API_URL", "http://localhost:5006"),
			UIURL:            "http://localhost:3005",
			HealthEndpoint:   "/health",
			Capabilities:     []string{"incident_tickets", "service_requests", "sla_monitoring", "routing"},
			SupportedObjects: []string{"ticket", "sla", "agent"},
			SupportedEvents:  []string{"ticket.created", "ticket.resolved", "sla.breached"},
			AuthMethod:       "bearer_jwt",
			Status:           "active",
			LastHeartbeat:    nowUTC(),
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
		},
		{
			ApplicationID:    "analytics",
			Name:             "StatGate Analytics & Intelligence Workspace",
			Version:          "4.2.0",
			Owner:            "Data Science & Statistical Analysis",
			APIURL:           getEnv("ANALYTICS_API_URL", "http://localhost:5000"),
			UIURL:            "http://localhost:5000",
			HealthEndpoint:   "/health",
			Capabilities:     []string{"statistical_modelling", "dashboards", "anomaly_detection", "export"},
			SupportedObjects: []string{"dataset", "indicator", "dashboard", "kpi", "anomaly"},
			SupportedEvents:  []string{"dataset.updated", "anomaly.detected", "kpi.calculated"},
			AuthMethod:       "bearer_jwt",
			Status:           "active",
			LastHeartbeat:    nowUTC(),
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
		},
	}

	for _, a := range apps {
		fabricStore.applications[a.ApplicationID] = a
	}

	// 2. Bootstrap default Data Catalogue items
	catalogueItems := []CatalogueDataset{
		{
			ID:                 "ds_nat_health_survey_2026",
			Name:               "National Health & Facility Survey Dataset",
			Description:        "Comprehensive multi-district healthcare operational, staffing, and clinical capacity survey dataset.",
			Owner:              "Ministry of Health — Directorate of Planning & Statistics",
			SourceApplication:  "statcollect",
			OrganizationID:     "MOH_UG",
			DataDomain:         "Health Facilities & Systems",
			UpdateFrequency:    "Daily",
			LastRefresh:        nowUTC(),
			RecordCount:        14850,
			QualityScore:       98.40,
			Completeness:       99.10,
			Coverage:           98.50,
			Sensitivity:        "official",
			GeographicCoverage: "Uganda (135 Districts, 4 Regions)",
			TemporalCoverage:   "2024-2026",
			VariablesCount:     24,
			LineageSource:      "StatCollect Mobile Ingestion → Pipeline Validation → PostgreSQL",
			FreshnessStatus:    "live",
			TenantID:           "tenant_uganda_inst",
			CreatedAt:          nowUTC(),
			UpdatedAt:          nowUTC(),
			SchemaDefinition: []CatalogueVariable{
				{Name: "facility_id", Type: "string", Description: "Unique national facility code (e.g. FAC-001)", AllowedVals: []string{"FAC-*"}},
				{Name: "facility_name", Type: "string", Description: "Official registered facility name"},
				{Name: "district", Type: "string", Description: "Administrative district"},
				{Name: "level", Type: "string", Description: "Facility level classification", AllowedVals: []string{"National Referral", "Regional Referral", "General Hospital", "Health Centre IV"}},
				{Name: "latitude", Type: "float64", Description: "WGS84 Latitude coordinate"},
				{Name: "longitude", Type: "float64", Description: "WGS84 Longitude coordinate"},
				{Name: "bed_occupancy_pct", Type: "float64", Description: "Current active bed occupancy rate", Unit: "%"},
				{Name: "stockout_status", Type: "string", Description: "Essential medicine stockout state", AllowedVals: []string{"none", "low", "critical"}},
			},
		},
		{
			ID:                 "ds_epidemic_surveillance_ug",
			Name:               "National Epidemiological Surveillance Matrix",
			Description:        "Real-time case incidence, laboratory confirmation, and regional syndromic surveillance indicators.",
			Owner:              "Uganda National Institute of Public Health (UNIPH)",
			SourceApplication:  "analytics",
			OrganizationID:     "UNIPH_UG",
			DataDomain:         "Epidemiology & Outbreak Response",
			UpdateFrequency:    "Hourly",
			LastRefresh:        nowUTC(),
			RecordCount:        82400,
			QualityScore:       97.80,
			Completeness:       98.90,
			Coverage:           99.20,
			Sensitivity:        "official",
			GeographicCoverage: "National (All Border Posts & Districts)",
			TemporalCoverage:   "2025-2026",
			VariablesCount:     18,
			LineageSource:      "UNIPH Lab API + DHIS2 Synced + StatGate Pipeline",
			FreshnessStatus:    "live",
			TenantID:           "tenant_uganda_inst",
			CreatedAt:          nowUTC(),
			UpdatedAt:          nowUTC(),
		},
		{
			ID:                 "ds_pms_institutional_projects",
			Name:               "Institutional Capital & Development Project Portfolio",
			Description:        "Central registry of all national infrastructure, laboratory upgrade, and medical digital health projects.",
			Owner:              "Ministry of Health PMO",
			SourceApplication:  "pms",
			OrganizationID:     "MOH_UG",
			DataDomain:         "Project Execution & Governance",
			UpdateFrequency:    "Daily",
			LastRefresh:        nowUTC(),
			RecordCount:        142,
			QualityScore:       99.50,
			Completeness:       100.0,
			Coverage:           100.0,
			Sensitivity:        "internal",
			GeographicCoverage: "National",
			TemporalCoverage:   "2024-2028",
			VariablesCount:     14,
			LineageSource:      "StatGate PMS Database",
			FreshnessStatus:    "live",
			TenantID:           "tenant_uganda_inst",
			CreatedAt:          nowUTC(),
			UpdatedAt:          nowUTC(),
		},
	}

	for _, ds := range catalogueItems {
		fabricStore.catalogue[ds.ID] = ds
	}

	// 3. Bootstrap Data Dictionary
	dictItems := []DictionaryVariable{
		{
			ID:                 "var_facility_id",
			CanonicalName:      "facility_id",
			Aliases:            []string{"facility_code", "health_facility_id", "site_id", "service_point_id"},
			Definition:         "Canonical identifier assigned to a recognized institutional healthcare or administrative facility in Uganda.",
			DataType:           "string",
			PermissibleValues:  []string{"FAC-001", "FAC-002", "FAC-003", "FAC-004", "FAC-005"},
			SourceApplications: []string{"statcollect", "pms", "rms", "analytics", "helpdesk"},
			Owner:              "National Health Data Standards Committee",
			RelatedIndicators:  []string{"facility_density_per_10k", "facility_readiness_index"},
			RelatedDatasets:    []string{"ds_nat_health_survey_2026"},
			Version:            1,
			Status:             "approved",
			TenantID:           "tenant_uganda_inst",
			CreatedAt:          nowUTC(),
			UpdatedAt:          nowUTC(),
		},
		{
			ID:                 "var_district",
			CanonicalName:      "district",
			Aliases:            []string{"district_name", "admin_district", "lga", "district_id"},
			Definition:         "Designated local government administrative district in Uganda (135 recognized districts).",
			DataType:           "string",
			PermissibleValues:  []string{"Kampala", "Wakiso", "Mukono", "Jinja", "Gulu", "Mbarara", "Mbale", "Arua", "Fort Portal", "Kabale"},
			SourceApplications: []string{"statcollect", "pms", "rms", "analytics"},
			Owner:              "Uganda Bureau of Statistics (UBOS)",
			Version:            1,
			Status:             "approved",
			TenantID:           "tenant_uganda_inst",
			CreatedAt:          nowUTC(),
			UpdatedAt:          nowUTC(),
		},
		{
			ID:                 "var_bed_occupancy",
			CanonicalName:      "bed_occupancy_rate",
			Aliases:            []string{"occupancy_pct", "bed_occupancy", "ward_capacity_utilization"},
			Definition:         "Ratio of occupied inpatient beds to total officially staffed bed capacity, expressed as a percentage.",
			DataType:           "float64",
			Unit:               "%",
			SourceApplications: []string{"statcollect", "analytics"},
			Owner:              "Directorate of Clinical Services",
			RelatedIndicators:  []string{"hospital_overcrowding_index"},
			Version:            1,
			Status:             "approved",
			TenantID:           "tenant_uganda_inst",
			CreatedAt:          nowUTC(),
			UpdatedAt:          nowUTC(),
		},
	}

	for _, v := range dictItems {
		fabricStore.dictionary[v.ID] = v
	}

	// 4. Bootstrap Semantic Concept Mappings
	mappings := []SemanticConceptMapping{
		{ID: "sem_01", SourceTerm: "Health Facility", CanonicalConcept: "facility", SourceApplication: "statcollect", StandardCode: "DHIS2_FACILITY", Confidence: 1.00, ApprovedBy: "Data Standards Board", TenantID: "tenant_uganda_inst", CreatedAt: nowUTC()},
		{ID: "sem_02", SourceTerm: "Service Point", CanonicalConcept: "facility", SourceApplication: "helpdesk", StandardCode: "ITIL_SERVICE_POINT", Confidence: 0.95, ApprovedBy: "Data Standards Board", TenantID: "tenant_uganda_inst", CreatedAt: nowUTC()},
		{ID: "sem_03", SourceTerm: "Site", CanonicalConcept: "facility", SourceApplication: "rms", StandardCode: "GCP_CLINICAL_SITE", Confidence: 0.98, ApprovedBy: "Data Standards Board", TenantID: "tenant_uganda_inst", CreatedAt: nowUTC()},
		{ID: "sem_04", SourceTerm: "Project Code", CanonicalConcept: "project_id", SourceApplication: "pms", StandardCode: "ISO_21500_PROJ", Confidence: 1.00, ApprovedBy: "Data Standards Board", TenantID: "tenant_uganda_inst", CreatedAt: nowUTC()},
		{ID: "sem_05", SourceTerm: "Study Protocol", CanonicalConcept: "research_id", SourceApplication: "rms", StandardCode: "ICH_GCP_PROTOCOL", Confidence: 1.00, ApprovedBy: "Data Standards Board", TenantID: "tenant_uganda_inst", CreatedAt: nowUTC()},
	}

	for _, m := range mappings {
		fabricStore.mappings[m.ID] = m
	}

	// 5. Bootstrap Governed Knowledge Items with Versioning & Classification
	knowledgeItems := []GovernedKnowledgeItem{
		{
			ID:               "know_cold_chain_sop_2026",
			Title:            "Standard Operating Procedure: Regional Cold-Chain Maintenance & Failure Protocol",
			Summary:          "Authoritative protocol for responding to vaccine storage thermal excursions in regional referral depots.",
			Content:          "This SOP mandates 15-minute escalation when cold storage temperatures rise above 8.0°C. Backup generators must auto-switch within 60 seconds. All excursions exceeding 2 hours require immediate notification to the National EPI Directorate and quarantine of affected batches.",
			Category:         "sop",
			Classification:   "verified",
			Author:           "Dr. Christine Namubiru (Director EPI)",
			Reviewer:         "National Vaccine Safety Board",
			ApprovalStatus:   "published",
			Version:          2,
			EffectiveDate:    "2026-01-15T00:00:00Z",
			ExpiryDate:       "2027-01-15T00:00:00Z",
			SourceReferences: []string{"EPI-REG-2025-09", "WHO-PQS-E003"},
			ChangeLog: []KnowledgeChangeEntry{
				{Version: 1, Timestamp: "2025-01-10T10:00:00Z", Author: "Dr. Christine Namubiru", Summary: "Initial institutional release"},
				{Version: 2, Timestamp: "2026-01-15T00:00:00Z", Author: "Dr. Christine Namubiru", Summary: "Updated with automated IoT sensor telemetry escalation rules"},
			},
			Tags:      []string{"cold_chain", "vaccine", "sop", "verified"},
			TenantID:  "tenant_uganda_inst",
			CreatedAt: "2026-01-15T00:00:00Z",
			UpdatedAt: nowUTC(),
		},
		{
			ID:               "know_malaria_seasonal_forecast",
			Title:            "Analytical Finding: Eastern Region Seasonal Malaria Incidence Surge Risk",
			Summary:          "Derived statistical projection showing 32% expected increase in Busoga and Bugisu districts following seasonal rainfall anomalies.",
			Content:          "Bayesian space-time modeling indicates high probability of caseload surge exceeding historical facility capacities between March and May. Recommends pre-positioning Artemisinin-based Combination Therapies (ACTs) and Rapid Diagnostic Tests (RDTs) by February 28.",
			Category:         "analytical_finding",
			Classification:   "derived",
			Author:           "StatGate Analytical Modeling Core",
			ApprovalStatus:   "approved",
			Version:          1,
			EffectiveDate:    nowUTC(),
			SourceReferences: []string{"ds_epidemic_surveillance_ug", "ds_nat_health_survey_2026"},
			Tags:             []string{"malaria", "surge_forecast", "derived_intelligence"},
			TenantID:         "tenant_uganda_inst",
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
		},
		{
			ID:               "know_ai_allocation_guidance",
			Title:            "AI Recommendation: Gulu Regional Referral Emergency Oxygen Buffer Rebalancing",
			Summary:          "Machine-generated optimization suggestion to reallocate 40 surplus oxygen cylinders from Kitgum to Gulu Regional Referral.",
			Content:          "Automated utilization modeling observed Gulu ICU operating at 94% oxygen capacity over the past 72 hours, while Kitgum maintains a 45-day surplus buffer at 22% utilization. Note: AI Recommendation requires human review and signoff before task dispatch.",
			Category:         "ai_recommendation",
			Classification:   "ai_recommendation",
			Author:           "StatGate AI Decision Assistant",
			ApprovalStatus:   "published",
			Version:          1,
			EffectiveDate:    nowUTC(),
			SourceReferences: []string{"ds_nat_health_survey_2026", "FAC-005"},
			Tags:             []string{"oxygen", "icu", "ai_recommendation", "advisory"},
			TenantID:         "tenant_uganda_inst",
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
		},
		{
			ID:               "know_dec_emergency_procurement",
			Title:            "Organizational Decision: Authorization of Emergency Medical Buffer Procurement",
			Summary:          "Formally accepted institutional decision authorizing emergency medical supplies replenishment for Northern sub-regions.",
			Content:          "Pursuant to Executive Alert EA-2026-088 and confirmed anomaly ANOM-001, Permanent Secretary authorized procurement of emergency diagnostic and protective stock under Fast-Track Health Emergency Framework.",
			Category:         "organizational_decision",
			Classification:   "decision",
			Author:           "Permanent Secretary — Ministry of Health",
			ApprovalStatus:   "published",
			Version:          1,
			EffectiveDate:    nowUTC(),
			SourceReferences: []string{"DEC-001", "ANOM-001"},
			Tags:             []string{"procurement", "emergency", "decision", "formal"},
			TenantID:         "tenant_uganda_inst",
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
		},
	}

	for _, k := range knowledgeItems {
		fabricStore.knowledge[k.ID] = k
	}

	// 6. Bootstrap Initial Canonical Objects & Graph Relationships
	bootstrapCanonicalRelationships()
}

func bootstrapCanonicalRelationships() {
	// Canonical objects
	objs := []CanonicalObject{
		{CanonicalID: "tenant_uganda_inst:pms:project:PROJ-001", ObjectID: "PROJ-001", ObjectType: "project", SourceApplication: "pms", TenantID: "tenant_uganda_inst", OrganizationID: "MOH_UG", Title: "National Health Facility Infrastructure Upgrade", CreatedBy: "admin", Version: 1, Status: "active", Classification: "verified", Sensitivity: "official", CanonicalURL: "http://localhost:3010/projects/PROJ-001", CreatedAt: nowUTC(), UpdatedAt: nowUTC(), RelationshipsCount: 6},
		{CanonicalID: "tenant_uganda_inst:rms:research:RES-001", ObjectID: "RES-001", ObjectType: "research", SourceApplication: "rms", TenantID: "tenant_uganda_inst", OrganizationID: "MOH_UG", Title: "Clinical Trial Protocol: Novel Antimalarial Efficacy Study", CreatedBy: "pi_lead", Version: 1, Status: "active", Classification: "verified", Sensitivity: "confidential", CanonicalURL: "http://localhost:3011/research/RES-001", CreatedAt: nowUTC(), UpdatedAt: nowUTC(), RelationshipsCount: 4},
		{CanonicalID: "tenant_uganda_inst:statcollect:facility:FAC-001", ObjectID: "FAC-001", ObjectType: "facility", SourceApplication: "statcollect", TenantID: "tenant_uganda_inst", OrganizationID: "MOH_UG", Title: "Mulago National Referral Hospital", CreatedBy: "registry", Version: 1, Status: "active", Classification: "verified", Sensitivity: "public", CanonicalURL: "http://localhost:3006/facilities/FAC-001", CreatedAt: nowUTC(), UpdatedAt: nowUTC(), RelationshipsCount: 5},
		{CanonicalID: "tenant_uganda_inst:statcollect:survey:SUR-001", ObjectID: "SUR-001", ObjectType: "survey", SourceApplication: "statcollect", TenantID: "tenant_uganda_inst", OrganizationID: "MOH_UG", Title: "2026 National Hospital Bed & Oxygen Capacity Assessment", CreatedBy: "survey_coord", Version: 1, Status: "active", Classification: "verified", Sensitivity: "official", CanonicalURL: "http://localhost:3006/surveys/SUR-001", CreatedAt: nowUTC(), UpdatedAt: nowUTC(), RelationshipsCount: 4},
		{CanonicalID: "tenant_uganda_inst:analytics:dataset:ds_nat_health_survey_2026", ObjectID: "ds_nat_health_survey_2026", ObjectType: "dataset", SourceApplication: "analytics", TenantID: "tenant_uganda_inst", OrganizationID: "MOH_UG", Title: "National Health & Facility Survey Dataset", CreatedBy: "data_eng", Version: 1, Status: "active", Classification: "verified", Sensitivity: "official", CanonicalURL: "http://localhost:5000/datasets/ds_nat_health_survey_2026", CreatedAt: nowUTC(), UpdatedAt: nowUTC(), RelationshipsCount: 5},
		{CanonicalID: "tenant_uganda_inst:enterprise:decision:DEC-001", ObjectID: "DEC-001", ObjectType: "decision", SourceApplication: "enterprise", TenantID: "tenant_uganda_inst", OrganizationID: "MOH_UG", Title: "Approve Emergency Stock Replenishment for Northern Facilities", CreatedBy: "perm_sec", Version: 1, Status: "active", Classification: "decision", Sensitivity: "official", CanonicalURL: "/workspace/decision/DEC-001", CreatedAt: nowUTC(), UpdatedAt: nowUTC(), RelationshipsCount: 4},
	}

	for _, o := range objs {
		fabricStore.objects[o.CanonicalID] = o
	}

	// Governed relationships connecting the institutional ecosystem
	rels := []FabricRelationship{
		{ID: "frel_01", FromType: "project", FromID: "PROJ-001", ToType: "survey", ToID: "SUR-001", RelationType: "conducts_survey", Confidence: 1.00, Source: "pms", Provenance: "PMS Work Plan Deliverable", LifecycleStatus: "active", CreatedBy: "admin", CreatedAt: nowUTC(), TenantID: "tenant_uganda_inst"},
		{ID: "frel_02", FromType: "survey", FromID: "SUR-001", ToType: "dataset", ToID: "ds_nat_health_survey_2026", RelationType: "produces_dataset", Confidence: 1.00, Source: "statcollect", Provenance: "Survey Field Ingestion", LifecycleStatus: "active", CreatedBy: "survey_lead", CreatedAt: nowUTC(), TenantID: "tenant_uganda_inst"},
		{ID: "frel_03", FromType: "dataset", FromID: "ds_nat_health_survey_2026", ToType: "facility", ToID: "FAC-001", RelationType: "covers_facility", Confidence: 1.00, Source: "analytics", Provenance: "Facility Code Matching", LifecycleStatus: "active", CreatedBy: "analytics_pipeline", CreatedAt: nowUTC(), TenantID: "tenant_uganda_inst"},
		{ID: "frel_04", FromType: "dataset", FromID: "ds_nat_health_survey_2026", ToType: "decision", ToID: "DEC-001", RelationType: "evidences_decision", Confidence: 1.00, Source: "enterprise", Provenance: "Decision Evidence Citation", LifecycleStatus: "active", CreatedBy: "perm_sec", CreatedAt: nowUTC(), TenantID: "tenant_uganda_inst"},
		{ID: "frel_05", FromType: "decision", FromID: "DEC-001", ToType: "task", ToID: "TSK-001", RelationType: "assigns_action", Confidence: 1.00, Source: "enterprise", Provenance: "Decision Implementation Task", LifecycleStatus: "active", CreatedBy: "perm_sec", CreatedAt: nowUTC(), TenantID: "tenant_uganda_inst"},
		{ID: "frel_06", FromType: "task", FromID: "TSK-001", ToType: "ticket", ToID: "HD-101", RelationType: "creates_ticket", Confidence: 0.95, Source: "helpdesk", Provenance: "HelpDesk SLA Dispatch", LifecycleStatus: "active", CreatedBy: "ops_lead", CreatedAt: nowUTC(), TenantID: "tenant_uganda_inst"},
		{ID: "frel_07", FromType: "project", FromID: "PROJ-001", ToType: "research", ToID: "RES-001", RelationType: "collaborates_with", Confidence: 1.00, Source: "rms", Provenance: "Institutional Grant Affiliation", LifecycleStatus: "active", CreatedBy: "pi_lead", CreatedAt: nowUTC(), TenantID: "tenant_uganda_inst"},
	}

	for _, r := range rels {
		fabricStore.relations[r.ID] = r
	}
}

// ─── Fabric Business Logic & Graph Traversal ────────────────────────

func resolveCanonicalObject(objType, objID string) CanonicalObject {
	fabricStore.RLock()
	defer fabricStore.RUnlock()

	// Try lookup in canonical objects
	for _, obj := range fabricStore.objects {
		if strings.EqualFold(obj.ObjectType, objType) && strings.EqualFold(obj.ObjectID, objID) {
			return obj
		}
	}

	// Fallback: Dynamically generate canonical object representation
	return CanonicalObject{
		CanonicalID:        fmt.Sprintf("tenant_uganda_inst:enterprise:%s:%s", strings.ToLower(objType), objID),
		ObjectID:           objID,
		ObjectType:         objType,
		SourceApplication:  "enterprise",
		TenantID:           "tenant_uganda_inst",
		OrganizationID:     "MOH_UG",
		Title:              fmt.Sprintf("%s (%s)", strings.Title(strings.ReplaceAll(objType, "_", " ")), objID),
		CreatedBy:          "system",
		CreatedAt:          nowUTC(),
		UpdatedAt:          nowUTC(),
		Version:            1,
		Status:             "active",
		Classification:     "verified",
		Sensitivity:        "official",
		CanonicalURL:       fmt.Sprintf("/workspace/%s/%s", objType, objID),
		RelationshipsCount: len(getRelationshipsForObject(objType, objID)),
	}
}

func getRelationshipsForObject(objType, objID string) []FabricRelationship {
	fabricStore.RLock()
	defer fabricStore.RUnlock()

	res := make([]FabricRelationship, 0)
	for _, r := range fabricStore.relations {
		if (strings.EqualFold(r.FromType, objType) && strings.EqualFold(r.FromID, objID)) ||
			(strings.EqualFold(r.ToType, objType) && strings.EqualFold(r.ToID, objID)) {
			res = append(res, r)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].CreatedAt > res[j].CreatedAt })
	return res
}

func getRelationshipsForObjectScoped(objType, objID, tenantID, role string) []FabricRelationship {
	fabricStore.RLock()
	defer fabricStore.RUnlock()

	res := make([]FabricRelationship, 0)
	for _, r := range fabricStore.relations {
		if !canAccessTenantResource(r.TenantID, tenantID, role) {
			continue
		}
		if (strings.EqualFold(r.FromType, objType) && strings.EqualFold(r.FromID, objID)) ||
			(strings.EqualFold(r.ToType, objType) && strings.EqualFold(r.ToID, objID)) {
			res = append(res, r)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].CreatedAt > res[j].CreatedAt })
	return res
}

func createFabricRelationship(link FabricRelationship) FabricRelationship {
	fabricStore.Lock()
	defer fabricStore.Unlock()

	if link.ID == "" {
		link.ID = fmt.Sprintf("frel_%d", time.Now().UnixNano())
	}
	if link.CreatedAt == "" {
		link.CreatedAt = nowUTC()
	}
	if link.Confidence == 0 {
		link.Confidence = 1.00
	}
	if link.LifecycleStatus == "" {
		link.LifecycleStatus = "active"
	}
	if link.Source == "" {
		link.Source = "enterprise"
	}
	if link.TenantID == "" {
		link.TenantID = "tenant_uganda_inst"
	}

	fabricStore.relations[link.ID] = link

	// Also sync to legacy knowledgeStore for backward compatibility
	knowledgeStore.Lock()
	knowledgeStore.relationships[link.ID] = KnowledgeRelationship{
		ID:        link.ID,
		FromType:  link.FromType,
		FromID:    link.FromID,
		ToType:    link.ToType,
		ToID:      link.ToID,
		Relation:  link.RelationType,
		Source:    link.Source,
		CreatedBy: link.CreatedBy,
		CreatedAt: link.CreatedAt,
	}
	knowledgeStore.Unlock()

	recordAudit("fabric.relationship.created", "enterprise", link.CreatedBy, map[string]interface{}{
		"from": link.FromType + ":" + link.FromID,
		"to":   link.ToType + ":" + link.ToID,
		"type": link.RelationType,
	})

	return link
}

func canAccessTenantResource(resourceTenantID, callerTenantID, role string) bool {
	if isPlatformAccessRole(role) {
		return true
	}
	return resourceTenantID != "" && callerTenantID != "" && resourceTenantID == callerTenantID
}

func buildFabricGraph(objType, objID string, maxDepth int) map[string]interface{} {
	root := resolveCanonicalObject(objType, objID)
	directRels := getRelationshipsForObject(objType, objID)

	nodes := map[string]CanonicalObject{
		root.CanonicalID: root,
	}
	edges := make([]FabricRelationship, 0, len(directRels))

	for _, rel := range directRels {
		edges = append(edges, rel)

		// Resolve other node
		otherType := rel.ToType
		otherID := rel.ToID
		if strings.EqualFold(rel.ToType, objType) && strings.EqualFold(rel.ToID, objID) {
			otherType = rel.FromType
			otherID = rel.FromID
		}

		otherNode := resolveCanonicalObject(otherType, otherID)
		nodes[otherNode.CanonicalID] = otherNode
	}

	nodesList := make([]CanonicalObject, 0, len(nodes))
	for _, n := range nodes {
		nodesList = append(nodesList, n)
	}

	return map[string]interface{}{
		"root":        root,
		"nodes_count": len(nodesList),
		"edges_count": len(edges),
		"nodes":       nodesList,
		"edges":       edges,
	}
}

func buildFabricGraphScoped(objType, objID, tenantID, role string, maxDepth int) map[string]interface{} {
	root := resolveCanonicalObject(objType, objID)
	directRels := getRelationshipsForObjectScoped(objType, objID, tenantID, role)

	nodes := map[string]CanonicalObject{}
	if canAccessTenantResource(root.TenantID, tenantID, role) {
		nodes[root.CanonicalID] = root
	}
	edges := make([]FabricRelationship, 0, len(directRels))

	for _, rel := range directRels {
		edges = append(edges, rel)

		otherType := rel.ToType
		otherID := rel.ToID
		if strings.EqualFold(rel.ToType, objType) && strings.EqualFold(rel.ToID, objID) {
			otherType = rel.FromType
			otherID = rel.FromID
		}

		otherNode := resolveCanonicalObject(otherType, otherID)
		if canAccessTenantResource(otherNode.TenantID, tenantID, role) {
			nodes[otherNode.CanonicalID] = otherNode
		}
	}

	nodesList := make([]CanonicalObject, 0, len(nodes))
	for _, n := range nodes {
		nodesList = append(nodesList, n)
	}

	return map[string]interface{}{
		"root":        root,
		"nodes_count": len(nodesList),
		"edges_count": len(edges),
		"nodes":       nodesList,
		"edges":       edges,
	}
}

func buildFullLineage(entityType, entityID string) []FabricLineageStep {
	// Full institutional lineage trace:
	// Source -> Collection -> Submission -> Dataset -> Transformation -> Indicator -> Analysis -> Report -> Decision -> Task -> Outcome
	return []FabricLineageStep{
		{Stage: "source", Label: "National Health Facilities (Mulago, Jinja, Gulu)", SourceApp: "statcollect", EntityID: "FAC-001", Description: "Primary facility ward telemetry and operational returns.", Timestamp: "2026-08-10T08:00:00Z", Status: "verified"},
		{Stage: "collection", Label: "Field Survey Protocol (SUR-001)", SourceApp: "statcollect", EntityID: "SUR-001", Description: "Hospital capacity & drug availability mobile census.", Timestamp: "2026-08-11T10:30:00Z", Status: "verified"},
		{Stage: "submission", Label: "Batch Ingestion & GPS Verification", SourceApp: "statcollect", EntityID: "SUB-8891", Description: "14,850 enumerator survey submissions validated with SHA-256 signature.", Timestamp: "2026-08-12T14:15:00Z", Status: "verified"},
		{Stage: "dataset", Label: "National Health Survey Dataset", SourceApp: "analytics", EntityID: "ds_nat_health_survey_2026", Description: "Unified enterprise dataset with 98.4% data quality score.", Timestamp: "2026-08-13T09:00:00Z", Status: "active"},
		{Stage: "transformation", Label: "Statistical Aggregation & Outlier Cleaning", SourceApp: "analytics", EntityID: "PIPE-AGGR-01", Description: "Z-score normalization and geographic boundary harmonization.", Timestamp: "2026-08-13T12:00:00Z", Status: "completed"},
		{Stage: "indicator", Label: "Essential Drug Stockout & Bed Utilization KPI", SourceApp: "analytics", EntityID: "KPI-FAC-01", Description: "Standardized institutional readiness metric across 135 districts.", Timestamp: "2026-08-14T08:00:00Z", Status: "calculated"},
		{Stage: "report", Label: "Quarterly Healthcare Delivery Report", SourceApp: "enterprise", EntityID: "REP-2026-Q3", Description: "Reviewed and approved by Planning Directorate.", Timestamp: "2026-08-14T16:00:00Z", Status: "approved"},
		{Stage: "decision", Label: "Executive Stock Replenishment Authorization", SourceApp: "enterprise", EntityID: "DEC-001", Description: "Formal authorization to dispatch emergency buffer buffer inventory.", Timestamp: "2026-08-15T09:30:00Z", Status: "executed"},
		{Stage: "task", Label: "Logistics Dispatch Execution Task", SourceApp: "pms", EntityID: "TSK-001", Description: "Warehouse shipment dispatch assigned to Logistics Unit.", Timestamp: "2026-08-15T11:00:00Z", Status: "in_progress"},
		{Stage: "outcome", Label: "Stockout Resolution Verification", SourceApp: "enterprise", EntityID: "OUT-001", Description: "Automated post-action telemetry monitoring for facility stock recovery.", Timestamp: "2026-08-15T14:00:00Z", Status: "monitoring"},
	}
}

// ─── HTTP Handler Implementations ───────────────────────────────────

func handleFabricResolveObject(c *gin.Context) {
	objType := c.Query("type")
	objID := c.Query("id")
	if objType == "" || objID == "" {
		c.JSON(400, gin.H{"error": "type and id query parameters required"})
		return
	}
	obj := resolveCanonicalObject(objType, objID)
	if !canAccessTenantResource(obj.TenantID, getContextTenantID(c), getContextRole(c)) {
		c.JSON(404, gin.H{"error": "object not found"})
		return
	}
	c.JSON(200, obj)
}

func handleFabricListCanonicalObjects(c *gin.Context) {
	objType := c.Query("type")
	query := strings.ToLower(strings.TrimSpace(c.Query("q")))
	tenantID := getContextTenantID(c)
	role := getContextRole(c)

	fabricStore.RLock()
	defer fabricStore.RUnlock()

	out := make([]CanonicalObject, 0)
	for _, obj := range fabricStore.objects {
		if !canAccessTenantResource(obj.TenantID, tenantID, role) {
			continue
		}
		if objType != "" && !strings.EqualFold(obj.ObjectType, objType) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(obj.Title+" "+obj.Description+" "+obj.ObjectID), query) {
			continue
		}
		out = append(out, obj)
	}

	c.JSON(200, gin.H{"count": len(out), "objects": out})
}

func handleFabricRelationshipGraph(c *gin.Context) {
	objType := c.Query("type")
	objID := c.Query("id")
	if objType == "" || objID == "" {
		c.JSON(400, gin.H{"error": "type and id are required"})
		return
	}
	c.JSON(200, buildFabricGraphScoped(objType, objID, getContextTenantID(c), getContextRole(c), 2))
}

func handleFabricCreateRelationship(c *gin.Context) {
	var link FabricRelationship
	if err := c.ShouldBindJSON(&link); err != nil || link.FromType == "" || link.FromID == "" || link.ToType == "" || link.ToID == "" {
		c.JSON(400, gin.H{"error": "from_type, from_id, to_type and to_id are required"})
		return
	}
	tenantID := getContextTenantID(c)
	role := getContextRole(c)
	if tenantID == "" {
		c.JSON(403, gin.H{"error": "tenant_context_required"})
		return
	}
	if !isPlatformAccessRole(role) && link.TenantID != "" && link.TenantID != tenantID {
		c.JSON(403, gin.H{"error": "forbidden", "message": "relationship tenant must match authenticated tenant"})
		return
	}
	if link.TenantID == "" || !isPlatformAccessRole(role) {
		link.TenantID = tenantID
	}
	if link.CreatedBy == "" {
		link.CreatedBy = knowledgeActor(c)
	}
	c.JSON(201, createFabricRelationship(link))
}

func handleFabricListGovernedKnowledge(c *gin.Context) {
	class := c.Query("classification")
	status := c.Query("status")
	query := strings.ToLower(strings.TrimSpace(c.Query("q")))

	fabricStore.RLock()
	defer fabricStore.RUnlock()

	out := make([]GovernedKnowledgeItem, 0)
	for _, k := range fabricStore.knowledge {
		if class != "" && !strings.EqualFold(k.Classification, class) {
			continue
		}
		if status != "" && !strings.EqualFold(k.ApprovalStatus, status) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(k.Title+" "+k.Summary+" "+k.Content), query) {
			continue
		}
		out = append(out, k)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	c.JSON(200, gin.H{"count": len(out), "knowledge": out})
}

func handleFabricCreateGovernedKnowledge(c *gin.Context) {
	var item GovernedKnowledgeItem
	if err := c.ShouldBindJSON(&item); err != nil || item.Title == "" || item.Content == "" {
		c.JSON(400, gin.H{"error": "title and content are required"})
		return
	}

	if item.ID == "" {
		item.ID = fmt.Sprintf("know_%d", time.Now().UnixNano())
	}
	if item.Classification == "" {
		item.Classification = "verified"
	}
	if item.ApprovalStatus == "" {
		item.ApprovalStatus = "published"
	}
	if item.Version == 0 {
		item.Version = 1
	}
	if item.Author == "" {
		item.Author = knowledgeActor(c)
	}
	if item.EffectiveDate == "" {
		item.EffectiveDate = nowUTC()
	}
	item.CreatedAt = nowUTC()
	item.UpdatedAt = nowUTC()
	item.TenantID = "tenant_uganda_inst"

	fabricStore.Lock()
	fabricStore.knowledge[item.ID] = item
	fabricStore.Unlock()

	recordAudit("knowledge.governed.created", "enterprise", item.Author, map[string]interface{}{
		"id":             item.ID,
		"classification": item.Classification,
		"version":        item.Version,
	})

	c.JSON(201, item)
}

func handleFabricListCatalogue(c *gin.Context) {
	domain := c.Query("domain")
	source := c.Query("source")
	query := strings.ToLower(strings.TrimSpace(c.Query("q")))

	fabricStore.RLock()
	defer fabricStore.RUnlock()

	out := make([]CatalogueDataset, 0)
	for _, ds := range fabricStore.catalogue {
		if domain != "" && !strings.EqualFold(ds.DataDomain, domain) {
			continue
		}
		if source != "" && !strings.EqualFold(ds.SourceApplication, source) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(ds.Name+" "+ds.Description+" "+ds.ID), query) {
			continue
		}
		out = append(out, ds)
	}

	c.JSON(200, gin.H{"count": len(out), "datasets": out})
}

func handleFabricGetCatalogueItem(c *gin.Context) {
	id := c.Param("id")

	fabricStore.RLock()
	ds, ok := fabricStore.catalogue[id]
	fabricStore.RUnlock()

	if !ok {
		c.JSON(404, gin.H{"error": "dataset not found in catalogue"})
		return
	}
	c.JSON(200, ds)
}

func handleFabricListDictionary(c *gin.Context) {
	query := strings.ToLower(strings.TrimSpace(c.Query("q")))

	fabricStore.RLock()
	defer fabricStore.RUnlock()

	out := make([]DictionaryVariable, 0)
	for _, v := range fabricStore.dictionary {
		if query != "" {
			match := strings.Contains(strings.ToLower(v.CanonicalName+" "+v.Definition), query)
			for _, a := range v.Aliases {
				if strings.Contains(strings.ToLower(a), query) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		out = append(out, v)
	}

	c.JSON(200, gin.H{"count": len(out), "variables": out})
}

func handleFabricListSemanticMappings(c *gin.Context) {
	fabricStore.RLock()
	defer fabricStore.RUnlock()

	out := make([]SemanticConceptMapping, 0, len(fabricStore.mappings))
	for _, m := range fabricStore.mappings {
		out = append(out, m)
	}
	c.JSON(200, gin.H{"count": len(out), "mappings": out})
}

func handleFabricResolveSemanticTerm(c *gin.Context) {
	term := strings.ToLower(strings.TrimSpace(c.Query("term")))
	if term == "" {
		c.JSON(400, gin.H{"error": "term query parameter required"})
		return
	}

	fabricStore.RLock()
	defer fabricStore.RUnlock()

	for _, m := range fabricStore.mappings {
		if strings.EqualFold(m.SourceTerm, term) {
			c.JSON(200, gin.H{
				"source_term":       term,
				"canonical_concept": m.CanonicalConcept,
				"standard_code":     m.StandardCode,
				"confidence":        m.Confidence,
				"source":            m.SourceApplication,
			})
			return
		}
	}

	// Default fallback to term itself
	c.JSON(200, gin.H{
		"source_term":       term,
		"canonical_concept": term,
		"confidence":        0.75,
		"source":            "inferred",
	})
}

func handleFabricListApplications(c *gin.Context) {
	fabricStore.RLock()
	defer fabricStore.RUnlock()

	out := make([]RegisteredApplication, 0, len(fabricStore.applications))
	for _, a := range fabricStore.applications {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	c.JSON(200, gin.H{"count": len(out), "applications": out})
}

func handleFabricRegisterApplication(c *gin.Context) {
	var app RegisteredApplication
	if err := c.ShouldBindJSON(&app); err != nil || app.ApplicationID == "" || app.Name == "" {
		c.JSON(400, gin.H{"error": "application_id and name are required"})
		return
	}

	app.LastHeartbeat = nowUTC()
	app.UpdatedAt = nowUTC()
	if app.CreatedAt == "" {
		app.CreatedAt = nowUTC()
	}
	if app.Status == "" {
		app.Status = "active"
	}

	fabricStore.Lock()
	fabricStore.applications[app.ApplicationID] = app
	fabricStore.Unlock()

	recordAudit("fabric.application.registered", "enterprise", "system", map[string]interface{}{
		"application_id": app.ApplicationID,
		"version":        app.Version,
	})

	c.JSON(200, app)
}

func handleFabricFullLineage(c *gin.Context) {
	entityType := c.Query("type")
	entityID := c.Query("id")
	if entityType == "" {
		entityType = "kpi"
	}
	if entityID == "" {
		entityID = "KPI-FAC-01"
	}

	lineage := buildFullLineage(entityType, entityID)
	c.JSON(200, gin.H{
		"target_entity": gin.H{"type": entityType, "id": entityID},
		"lineage_depth": len(lineage),
		"trace":         lineage,
	})
}

func handleFabricInstitutionalMemory(c *gin.Context) {
	status := c.Query("status")

	fabricStore.RLock()
	defer fabricStore.RUnlock()

	out := make([]InstitutionalDecisionMemory, 0)
	for _, dec := range fabricStore.memory {
		if status != "" && !strings.EqualFold(dec.OutcomeStatus, status) {
			continue
		}
		out = append(out, dec)
	}

	c.JSON(200, gin.H{"count": len(out), "decisions_memory": out})
}
