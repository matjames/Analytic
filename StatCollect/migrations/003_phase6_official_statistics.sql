-- ── Phase 6 — Official Statistics, Surveys & Census Management Migration ──
-- Implements complete GSBPM-aligned official statistics infrastructure:
-- 1. Question Bank & Survey Projects
-- 2. Enumeration Areas & Master Sampling Frames
-- 3. Field Workforce (Enumerators & Supervisors) & Assignment Plans
-- 4. Supervisor Field Quality Reviews & Geofence Verification
-- 5. Census Rounds, Census Operations & PES (Post-Enumeration Survey)
-- 6. UN Sustainable Development Goals (SDG) Monitoring Framework (Goals 1-17)
-- 7. Dissemination Portal, Publications & Microdata Releases
-- 8. Tabulation Outputs & Published Tables

-- 1. Question Bank: Reusable standard questions for official surveys
CREATE TABLE IF NOT EXISTS question_bank (
  id TEXT PRIMARY KEY,
  code TEXT UNIQUE NOT NULL,
  domain TEXT NOT NULL, -- demographics, health, education, labor_economy, agriculture, water_sanitation, housing
  label TEXT NOT NULL,
  hint TEXT,
  question_type TEXT NOT NULL, -- text, integer, decimal, select_one, select_multiple, date, geopoint, barcode, image
  options JSONB NOT NULL DEFAULT '[]', -- list of {value, label, code}
  validation JSONB NOT NULL DEFAULT '{}', -- {required: bool, min: num, max: num, regex: str}
  skip_logic JSONB NOT NULL DEFAULT '{}', -- {relevant_expr: str, condition: str}
  sdg_indicator_code TEXT, -- e.g. 1.1.1, 3.1.1, 4.1.1
  tags JSONB NOT NULL DEFAULT '[]',
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_question_bank_domain ON question_bank(domain);
CREATE INDEX IF NOT EXISTS idx_question_bank_sdg ON question_bank(sdg_indicator_code);
CREATE INDEX IF NOT EXISTS idx_question_bank_tenant ON question_bank(tenant_id);

-- 2. Survey Projects: GSBPM lifecycle management
CREATE TABLE IF NOT EXISTS survey_projects (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT,
  gsbpm_phase TEXT NOT NULL DEFAULT 'specify_needs', -- specify_needs, design, build, collect, process, analyse, disseminate, archive
  survey_type TEXT NOT NULL DEFAULT 'household', -- household, facility, enterprise, agricultural, census, rapid
  target_sample_size INT NOT NULL DEFAULT 0,
  start_date DATE,
  end_date DATE,
  status TEXT NOT NULL DEFAULT 'draft', -- draft, active, fieldwork, processing, closed, archived
  lead_agency TEXT NOT NULL DEFAULT 'National Statistics Office (NSO)',
  clearance_level TEXT NOT NULL DEFAULT 'OFFICIAL',
  sample_frame_id TEXT,
  sampling_design_id BIGINT,
  template_ids JSONB NOT NULL DEFAULT '[]',
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_survey_projects_status ON survey_projects(status);
CREATE INDEX IF NOT EXISTS idx_survey_projects_phase ON survey_projects(gsbpm_phase);
CREATE INDEX IF NOT EXISTS idx_survey_projects_tenant ON survey_projects(tenant_id);

-- 3. Enumeration Areas (EAs): Official geographic building blocks
CREATE TABLE IF NOT EXISTS enumeration_areas (
  id TEXT PRIMARY KEY,
  ea_code TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  country_code TEXT NOT NULL DEFAULT 'UGA',
  region TEXT NOT NULL,
  district TEXT NOT NULL,
  subcounty TEXT,
  parish TEXT,
  village TEXT,
  urban_rural TEXT NOT NULL DEFAULT 'Rural', -- Urban, Rural, Peri-urban
  estimated_households INT NOT NULL DEFAULT 0,
  estimated_population INT NOT NULL DEFAULT 0,
  centroid_lat DOUBLE PRECISION,
  centroid_lng DOUBLE PRECISION,
  boundary_geojson JSONB NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'unassigned', -- unassigned, assigned, in_progress, completed
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ea_district ON enumeration_areas(district);
CREATE INDEX IF NOT EXISTS idx_ea_region ON enumeration_areas(region);
CREATE INDEX IF NOT EXISTS idx_ea_status ON enumeration_areas(status);
CREATE INDEX IF NOT EXISTS idx_ea_urban_rural ON enumeration_areas(urban_rural);
CREATE INDEX IF NOT EXISTS idx_ea_coords ON enumeration_areas(centroid_lat, centroid_lng)
  WHERE centroid_lat IS NOT NULL AND centroid_lng IS NOT NULL;

-- 4. Master Sampling Frames: Statistical frames for sample design
CREATE TABLE IF NOT EXISTS sample_frames (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  frame_type TEXT NOT NULL DEFAULT 'census_frame', -- census_frame, area_frame, list_frame, dual_frame
  total_eas INT NOT NULL DEFAULT 0,
  total_households BIGINT NOT NULL DEFAULT 0,
  total_population BIGINT NOT NULL DEFAULT 0,
  strata JSONB NOT NULL DEFAULT '[]', -- list of {name, count_eas, population, weight}
  ea_codes JSONB NOT NULL DEFAULT '[]', -- list of EA codes in this frame
  year INT NOT NULL DEFAULT 2026,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sample_frames_tenant ON sample_frames(tenant_id);

-- 5. Field Supervisors: Regional and district field management
CREATE TABLE IF NOT EXISTS field_supervisors (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  code TEXT UNIQUE NOT NULL,
  phone TEXT,
  email TEXT,
  assigned_region TEXT NOT NULL,
  assigned_district TEXT,
  team_size INT NOT NULL DEFAULT 0,
  active BOOLEAN NOT NULL DEFAULT true,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_supervisors_region ON field_supervisors(assigned_region);
CREATE INDEX IF NOT EXISTS idx_supervisors_active ON field_supervisors(active);

-- 6. Field Enumerators: CAPI field interviewers
CREATE TABLE IF NOT EXISTS enumerators (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  code TEXT UNIQUE NOT NULL,
  phone TEXT,
  email TEXT,
  supervisor_id TEXT REFERENCES field_supervisors(id) ON DELETE SET NULL,
  assigned_device_id TEXT,
  primary_region TEXT NOT NULL,
  languages JSONB NOT NULL DEFAULT '["English"]',
  status TEXT NOT NULL DEFAULT 'Active', -- Active, Inactive, Training, Suspended
  rating FLOAT NOT NULL DEFAULT 5.0,
  total_submissions INT NOT NULL DEFAULT 0,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_enumerators_supervisor ON enumerators(supervisor_id);
CREATE INDEX IF NOT EXISTS idx_enumerators_status ON enumerators(status);
CREATE INDEX IF NOT EXISTS idx_enumerators_region ON enumerators(primary_region);

-- 7. Assignment Plans: Detailed operational allocations
CREATE TABLE IF NOT EXISTS assignment_plans (
  id BIGSERIAL PRIMARY KEY,
  survey_id TEXT NOT NULL REFERENCES survey_projects(id) ON DELETE CASCADE,
  ea_id TEXT NOT NULL REFERENCES enumeration_areas(id) ON DELETE CASCADE,
  enumerator_id TEXT REFERENCES enumerators(id) ON DELETE SET NULL,
  supervisor_id TEXT REFERENCES field_supervisors(id) ON DELETE SET NULL,
  target_quota INT NOT NULL DEFAULT 30,
  completed_count INT NOT NULL DEFAULT 0,
  start_date DATE,
  deadline DATE,
  status TEXT NOT NULL DEFAULT 'assigned', -- assigned, in_progress, completed, overdue, flagged
  notes TEXT,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assign_plans_survey ON assignment_plans(survey_id);
CREATE INDEX IF NOT EXISTS idx_assign_plans_ea ON assignment_plans(ea_id);
CREATE INDEX IF NOT EXISTS idx_assign_plans_enum ON assignment_plans(enumerator_id);
CREATE INDEX IF NOT EXISTS idx_assign_plans_sup ON assignment_plans(supervisor_id);
CREATE INDEX IF NOT EXISTS idx_assign_plans_status ON assignment_plans(status);

-- 8. Supervisor Quality Reviews: Field validations, GPS checks, and return-for-revisit
CREATE TABLE IF NOT EXISTS supervisor_reviews (
  id BIGSERIAL PRIMARY KEY,
  submission_instance_id TEXT NOT NULL REFERENCES submissions(instance_id) ON DELETE CASCADE,
  supervisor_id TEXT NOT NULL,
  decision TEXT NOT NULL, -- approved, rejected, flag_for_revisit
  revisit_reason TEXT,
  gps_verified BOOLEAN NOT NULL DEFAULT true,
  distance_from_ea_centroid_meters FLOAT DEFAULT 0,
  geofence_breach BOOLEAN NOT NULL DEFAULT false,
  duration_seconds INT DEFAULT 0,
  speed_anomaly BOOLEAN NOT NULL DEFAULT false,
  notes TEXT,
  reviewed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sup_reviews_submission ON supervisor_reviews(submission_instance_id);
CREATE INDEX IF NOT EXISTS idx_sup_reviews_sup ON supervisor_reviews(supervisor_id);
CREATE INDEX IF NOT EXISTS idx_sup_reviews_decision ON supervisor_reviews(decision);

-- 9. Census Rounds & Operations: National Population and Housing Censuses
CREATE TABLE IF NOT EXISTS census_rounds (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  round_year INT NOT NULL,
  legal_mandate TEXT NOT NULL,
  reference_night TIMESTAMPTZ,
  pre_enumeration_status TEXT NOT NULL DEFAULT 'completed', -- pending, in_progress, completed
  enumeration_status TEXT NOT NULL DEFAULT 'active', -- not_started, active, completed
  post_enumeration_status TEXT NOT NULL DEFAULT 'pending', -- pending, in_progress, completed
  total_projected_pop BIGINT NOT NULL DEFAULT 0,
  total_enumerated_pop BIGINT NOT NULL DEFAULT 0,
  households_enumerated BIGINT NOT NULL DEFAULT 0,
  coverage_percentage FLOAT NOT NULL DEFAULT 0.0,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_census_rounds_year ON census_rounds(round_year);
CREATE INDEX IF NOT EXISTS idx_census_rounds_status ON census_rounds(enumeration_status);

-- 10. Census Post-Enumeration Survey (PES) Dual-System Estimation
CREATE TABLE IF NOT EXISTS census_pes (
  id BIGSERIAL PRIMARY KEY,
  census_round_id TEXT NOT NULL REFERENCES census_rounds(id) ON DELETE CASCADE,
  pes_sample_eas_count INT NOT NULL DEFAULT 0,
  pes_sample_size BIGINT NOT NULL DEFAULT 0,
  matched_records BIGINT NOT NULL DEFAULT 0,
  census_only_records BIGINT NOT NULL DEFAULT 0,
  pes_only_records BIGINT NOT NULL DEFAULT 0,
  estimated_true_population BIGINT NOT NULL DEFAULT 0,
  net_undercount_rate FLOAT NOT NULL DEFAULT 0.0,
  coverage_rate FLOAT NOT NULL DEFAULT 0.0,
  gross_omission_rate FLOAT NOT NULL DEFAULT 0.0,
  confidence_interval_95 JSONB NOT NULL DEFAULT '{"lower":0,"upper":0}',
  calculated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_census_pes_round ON census_pes(census_round_id);

-- 11. UN Sustainable Development Goals (SDG) Monitoring Framework
CREATE TABLE IF NOT EXISTS sdg_indicators (
  id TEXT PRIMARY KEY,
  goal_number INT NOT NULL, -- 1 to 17
  goal_title TEXT NOT NULL,
  target_code TEXT NOT NULL, -- e.g. 1.1, 3.1, 4.1
  target_desc TEXT NOT NULL,
  indicator_code TEXT UNIQUE NOT NULL, -- e.g. 1.1.1, 3.1.1, 4.1.1
  indicator_desc TEXT NOT NULL,
  tier TEXT NOT NULL DEFAULT 'Tier I', -- Tier I, Tier II, Tier III
  custodian_agency TEXT NOT NULL,
  baseline_value FLOAT,
  baseline_year INT DEFAULT 2015,
  latest_value FLOAT,
  latest_year INT DEFAULT 2026,
  target_2030 FLOAT,
  unit TEXT NOT NULL DEFAULT '%',
  status TEXT NOT NULL DEFAULT 'On Track', -- On Track, Target Met, Stagnant, Regressing, Insufficient Data
  data_source TEXT,
  survey_project_id TEXT REFERENCES survey_projects(id) ON DELETE SET NULL,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sdg_goal ON sdg_indicators(goal_number);
CREATE INDEX IF NOT EXISTS idx_sdg_code ON sdg_indicators(indicator_code);
CREATE INDEX IF NOT EXISTS idx_sdg_status ON sdg_indicators(status);
CREATE INDEX IF NOT EXISTS idx_sdg_tier ON sdg_indicators(tier);

-- 12. Statistical Publications & Dissemination
CREATE TABLE IF NOT EXISTS statistical_publications (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  abstract TEXT,
  publication_type TEXT NOT NULL DEFAULT 'bulletin', -- bulletin, analytical_report, census_report, microdata_release, sdmx_feed
  domain TEXT NOT NULL DEFAULT 'General',
  survey_id TEXT REFERENCES survey_projects(id) ON DELETE SET NULL,
  census_id TEXT REFERENCES census_rounds(id) ON DELETE SET NULL,
  access_level TEXT NOT NULL DEFAULT 'public', -- public, registered, restricted
  download_url TEXT,
  file_format TEXT NOT NULL DEFAULT 'PDF', -- PDF, CSV, SDMX, XLSX
  downloads_count INT NOT NULL DEFAULT 0,
  is_published BOOLEAN NOT NULL DEFAULT true,
  published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  tenant_id TEXT NOT NULL DEFAULT 'default'
);

CREATE INDEX IF NOT EXISTS idx_stat_pub_domain ON statistical_publications(domain);
CREATE INDEX IF NOT EXISTS idx_stat_pub_type ON statistical_publications(publication_type);
CREATE INDEX IF NOT EXISTS idx_stat_pub_published ON statistical_publications(is_published);

-- 13. Tabulation Outputs: Saved multi-way tables for official reporting
CREATE TABLE IF NOT EXISTS tabulation_outputs (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  table_number TEXT NOT NULL,
  survey_id TEXT REFERENCES survey_projects(id) ON DELETE SET NULL,
  row_var TEXT NOT NULL,
  col_var TEXT NOT NULL,
  measure_var TEXT,
  aggregation TEXT NOT NULL DEFAULT 'COUNT',
  is_weighted BOOLEAN NOT NULL DEFAULT false,
  weight_var TEXT,
  matrix_data JSONB NOT NULL DEFAULT '{}',
  row_totals JSONB NOT NULL DEFAULT '{}',
  col_totals JSONB NOT NULL DEFAULT '{}',
  grand_total FLOAT NOT NULL DEFAULT 0,
  statistical_tests JSONB NOT NULL DEFAULT '[]', -- chi-square test, p-value, df
  status TEXT NOT NULL DEFAULT 'published', -- draft, reviewed, published
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tab_outputs_survey ON tabulation_outputs(survey_id);
CREATE INDEX IF NOT EXISTS idx_tab_outputs_status ON tabulation_outputs(status);

-- ── Pre-seeding Core Official Statistics Assets ──

-- Seed standard statistical questions across key official domains
INSERT INTO question_bank (id, code, domain, label, hint, question_type, options, validation, skip_logic, sdg_indicator_code, tags) VALUES
  ('qb_demo_age', 'AGE', 'demographics', 'What is the age in completed years of the household member?', 'Record age in completed years. For infants under 1 year, record 0.', 'integer', '[]', '{"required":true,"min":0,"max":120}', '{}', '1.1.1', '["demographics","census","age"]'),
  ('qb_demo_sex', 'SEX', 'demographics', 'What is the sex of the household member?', 'Observe or ask politely.', 'select_one', '[{"value":"M","label":"Male"},{"value":"F","label":"Female"}]', '{"required":true}', '{}', '5.5.1', '["demographics","census","gender"]'),
  ('qb_demo_rel', 'RELATIONSHIP', 'demographics', 'What is the relationship of this member to the household head?', 'Single selection.', 'select_one', '[{"value":"head","label":"Head"},{"value":"spouse","label":"Spouse"},{"value":"child","label":"Son/Daughter"},{"value":"parent","label":"Parent"},{"value":"grandchild","label":"Grandchild"},{"value":"other_rel","label":"Other Relative"},{"value":"non_rel","label":"Non-relative"}]', '{"required":true}', '{}', NULL, '["demographics","roster"]'),
  ('qb_demo_marital', 'MARITAL_STATUS', 'demographics', 'What is the current marital status?', 'Applies to persons aged 12 and above.', 'select_one', '[{"value":"never","label":"Never Married"},{"value":"married_monog","label":"Married (Monogamous)"},{"value":"married_polyg","label":"Married (Polygamous)"},{"value":"cohabiting","label":"Living Together/Consensual Union"},{"value":"divorced","label":"Divorced"},{"value":"separated","label":"Separated"},{"value":"widowed","label":"Widowed"}]', '{"required":true}', '{"relevant_expr":"AGE >= 12"}', NULL, '["demographics","marital"]'),
  
  ('qb_edu_attend', 'SCHOOL_ATTENDANCE', 'education', 'Has the person ever attended formal school?', 'For all persons aged 3 years and above.', 'select_one', '[{"value":"currently","label":"Currently Attending"},{"value":"past","label":"Attended in the Past"},{"value":"never","label":"Never Attended"}]', '{"required":true}', '{"relevant_expr":"AGE >= 3"}', '4.1.1', '["education","attendance"]'),
  ('qb_edu_level', 'HIGHEST_EDUCATION', 'education', 'What is the highest level of education completed?', 'Applies to persons who have ever attended school.', 'select_one', '[{"value":"none","label":"Some Primary / None"},{"value":"primary","label":"Completed Primary (P7)"},{"value":"lower_sec","label":"Lower Secondary (O-Level)"},{"value":"upper_sec","label":"Upper Secondary (A-Level)"},{"value":"tertiary","label":"Vocational/Technical Diploma"},{"value":"degree","label":"University Degree or Higher"}]', '{"required":true}', '{"relevant_expr":"SCHOOL_ATTENDANCE != ''never''"}', '4.1.2', '["education","attainment"]'),
  ('qb_edu_literacy', 'LITERACY', 'education', 'Can this person read and write with understanding in any language?', 'Simple sentence test.', 'select_one', '[{"value":"both","label":"Can read and write"},{"value":"read_only","label":"Can read only"},{"value":"none","label":"Cannot read or write"}]', '{"required":true}', '{"relevant_expr":"AGE >= 10"}', '4.6.1', '["education","literacy"]'),

  ('qb_econ_activity', 'ECONOMIC_ACTIVITY', 'labor_economy', 'During the last 7 days, did this person do any work for pay, profit, or family gain?', 'Reference period: past 7 days.', 'select_one', '[{"value":"worked","label":"Worked for pay / profit / family business"},{"value":"agriculture","label":"Worked on own farm / livestock"},{"value":"temporary_absence","label":"Had a job/business but temporarily absent"},{"value":"seeking","label":"Did not work, actively seeking work"},{"value":"unavailable","label":"Did not work, student / homemaker / retired / disabled"}]', '{"required":true}', '{"relevant_expr":"AGE >= 14"}', '8.5.2', '["labor","employment"]'),
  ('qb_econ_occupation', 'OCCUPATION_ISCO', 'labor_economy', 'What was the main occupation/kind of work done?', 'Describe primary duty.', 'text', '[]', '{"required":true}', '{"relevant_expr":"ECONOMIC_ACTIVITY in (''worked'',''agriculture'',''temporary_absence'')"}', '8.5.1', '["labor","isco"]'),

  ('qb_health_insurance', 'HEALTH_INSURANCE', 'health', 'Is this person covered by any health insurance scheme?', 'Social, community, or private insurance.', 'select_one', '[{"value":"yes_national","label":"National Health Insurance"},{"value":"yes_private","label":"Private/Employer Insurance"},{"value":"yes_community","label":"Community-based Health Fund"},{"value":"no","label":"No Health Insurance"}]', '{"required":true}', '{}', '3.8.2', '["health","insurance"]'),
  ('qb_health_vaccine', 'CHILD_IMMUNIZATION', 'health', 'Has the child received all basic infant vaccinations (BCG, Polio, DPT/Pentavalent, Measles)?', 'Verify with child health card where available.', 'select_one', '[{"value":"card_complete","label":"Fully Vaccinated (Card Verified)"},{"value":"report_complete","label":"Fully Vaccinated (Mother/Caregiver Report)"},{"value":"partial","label":"Partially Vaccinated"},{"value":"none","label":"Not Vaccinated"}]', '{"required":true}', '{"relevant_expr":"AGE < 5"}', '3.b.1', '["health","immunization"]'),

  ('qb_wash_water', 'DRINKING_WATER_SOURCE', 'water_sanitation', 'What is the main source of drinking water for members of this household?', 'JMP WHO/UNICEF standard categories.', 'select_one', '[{"value":"piped_dwelling","label":"Piped into dwelling/compound"},{"value":"public_tap","label":"Public tap/standpipe"},{"value":"borehole","label":"Protected borehole / tubewell"},{"value":"protected_spring","label":"Protected spring"},{"value":"rainwater","label":"Rainwater collection"},{"value":"unprotected_well","label":"Unprotected dug well/spring"},{"value":"surface_water","label":"Surface water (river, dam, lake)"},{"value":"bottled","label":"Bottled / sachet water"}]', '{"required":true}', '{}', '6.1.1', '["wash","water","sdg6"]'),
  ('qb_wash_toilet', 'SANITATION_FACILITY', 'water_sanitation', 'What kind of toilet facility does this household mainly use?', 'Improved vs unimproved classification.', 'select_one', '[{"value":"flush_sewer","label":"Flush to piped sewer system"},{"value":"flush_septic","label":"Flush to septic tank"},{"value":"pit_vip","label":"Ventilated Improved Pit latrine (VIP)"},{"value":"pit_slab","label":"Pit latrine with slab"},{"value":"pit_open","label":"Open pit latrine without slab"},{"value":"bucket","label":"Bucket toilet / hanging"},{"value":"open_defecation","label":"No facility / bush / field"}]', '{"required":true}', '{}', '6.2.1', '["wash","sanitation","sdg6"]'),

  ('qb_energy_light', 'LIGHTING_ENERGY', 'housing', 'What is the primary source of energy used for lighting in this household?', 'Energy access tiering.', 'select_one', '[{"value":"grid_electricity","label":"National Grid Electricity"},{"value":"solar_home","label":"Solar Home System / Panels"},{"value":"solar_lantern","label":"Solar Lantern / Torch"},{"value":"biogas","label":"Biogas"},{"value":"kerosene","label":"Kerosene lamp / Lantern"},{"value":"candles","label":"Candles / Battery torch"},{"value":"firewood","label":"Firewood"}]', '{"required":true}', '{}', '7.1.1', '["energy","housing","sdg7"]'),
  ('qb_energy_cook', 'COOKING_FUEL', 'housing', 'What is the primary cooking fuel used by this household?', 'Clean vs polluting fuels.', 'select_one', '[{"value":"electricity","label":"Electricity"},{"value":"lpg","label":"LPG / Bottled Gas"},{"value":"biogas","label":"Biogas"},{"value":"improved_charcoal","label":"Charcoal (Improved cookstove)"},{"value":"traditional_charcoal","label":"Charcoal (Traditional stove)"},{"value":"firewood","label":"Firewood / Crop residues"},{"value":"dung","label":"Animal dung"}]', '{"required":true}', '{}', '7.1.2', '["energy","clean_cooking","sdg7"]')
ON CONFLICT (id) DO NOTHING;

-- Seed default Survey Project: Uganda National Household Survey 2026
INSERT INTO survey_projects (id, title, description, gsbpm_phase, survey_type, target_sample_size, start_date, end_date, status, lead_agency, clearance_level) VALUES
  ('sp_unhs_2026', 'Uganda National Household Survey 2026 (UNHS)', 'Comprehensive multi-topic national survey on socioeconomic conditions, poverty, labor, and household welfare.', 'collect', 'household', 15600, '2026-01-15', '2026-11-30', 'fieldwork', 'Uganda Bureau of Statistics (UBOS)', 'OFFICIAL_NATIONAL'),
  ('sp_dhs_2026', 'Demographic and Health Survey 2026 (DHS)', 'National sample survey covering fertility, maternal and child health, nutrition, and mortality.', 'design', 'household', 12000, '2026-06-01', '2027-02-28', 'active', 'Ministry of Health & NSO', 'OFFICIAL_NATIONAL')
ON CONFLICT (id) DO NOTHING;

-- Seed sample Enumeration Areas (EAs)
INSERT INTO enumeration_areas (id, ea_code, name, country_code, region, district, subcounty, parish, village, urban_rural, estimated_households, estimated_population, centroid_lat, centroid_lng, status) VALUES
  ('ea_kla_001', 'EA-KLA-CEN-001', 'Nakasero I Central', 'UGA', 'Central', 'Kampala', 'Central Division', 'Nakasero', 'Civic Centre', 'Urban', 184, 736, 0.3162, 32.5825, 'in_progress'),
  ('ea_kla_002', 'EA-KLA-CEN-002', 'Kololo East', 'UGA', 'Central', 'Kampala', 'Central Division', 'Kololo', 'Upper Kololo', 'Urban', 210, 840, 0.3280, 32.5950, 'assigned'),
  ('ea_kla_003', 'EA-KLA-KAW-003', 'Bwaise North', 'UGA', 'Central', 'Kampala', 'Kawempe Division', 'Bwaise II', 'St. Francis Zone', 'Urban', 320, 1450, 0.3520, 32.5640, 'in_progress'),
  ('ea_mkn_001', 'EA-MKN-CEN-001', 'Mukono Town West', 'UGA', 'Central', 'Mukono', 'Mukono Municipality', 'Goma', 'Seeta Zone A', 'Peri-urban', 240, 1020, 0.3540, 32.7210, 'assigned'),
  ('ea_jja_001', 'EA-JJA-WAL-001', 'Walukuba East Area 4', 'UGA', 'Eastern', 'Jinja', 'Jinja City South', 'Walukuba', 'Masese I', 'Urban', 260, 1180, 0.4350, 33.2200, 'in_progress'),
  ('ea_mbale_001', 'EA-MBL-IND-001', 'Industrial Ward A', 'UGA', 'Eastern', 'Mbale', 'Industrial City Division', 'Industrial', 'Half London', 'Urban', 195, 890, 1.0780, 34.1750, 'completed'),
  ('ea_mbarara_001', 'EA-MBR-RUT-001', 'Ruti Trading Centre', 'UGA', 'Western', 'Mbarara', 'Nyamitanga', 'Ruti', 'Ruti Central', 'Peri-urban', 225, 990, -0.6320, 30.6410, 'in_progress'),
  ('ea_kabale_001', 'EA-KBL-RUR-001', 'Kitumba Rural Cluster', 'UGA', 'Western', 'Kabale', 'Kitumba', 'Bwama', 'Kijuguta', 'Rural', 145, 690, -1.2750, 29.9850, 'unassigned'),
  ('ea_gulu_001', 'EA-GLU-BAR-001', 'Bardege Central A', 'UGA', 'Northern', 'Gulu', 'Bardege-Layibi', 'Bardege', 'Commercial Rd', 'Urban', 215, 1010, 2.7740, 32.2980, 'assigned'),
  ('ea_arua_001', 'EA-ARA-RUR-001', 'Arua Hill North', 'UGA', 'Northern', 'Arua', 'Arua Central', 'Arua Hill', 'Hospital Zone', 'Urban', 190, 875, 3.0310, 30.9110, 'unassigned')
ON CONFLICT (id) DO NOTHING;

-- Seed Master Sampling Frame (MSF 2026)
INSERT INTO sample_frames (id, name, description, frame_type, total_eas, total_households, total_population, strata, year) VALUES
  ('msf_uganda_2026', 'Uganda Master Statistical Frame 2026', 'National enumeration frame updated with digital cartography, demarcating 78,500 EAs across 135 districts.', 'census_frame', 78500, 10250000, 45850000,
   '[{"name":"Central Urban","count_eas":12400,"population":8950000,"weight":1.0},{"name":"Central Rural","count_eas":14200,"population":7800000,"weight":1.0},{"name":"Eastern Urban","count_eas":6100,"population":3650000,"weight":1.0},{"name":"Eastern Rural","count_eas":18500,"population":9850000,"weight":1.0},{"name":"Western Urban","count_eas":5800,"population":3450000,"weight":1.0},{"name":"Western Rural","count_eas":13200,"population":7400000,"weight":1.0},{"name":"Northern Urban","count_eas":3200,"population":1950000,"weight":1.0},{"name":"Northern Rural","count_eas":5100,"population":2800000,"weight":1.0}]',
   2026)
ON CONFLICT (id) DO NOTHING;

-- Seed Field Supervisors
INSERT INTO field_supervisors (id, name, code, phone, email, assigned_region, assigned_district, team_size) VALUES
  ('sup_kla_01', 'Grace Namutebi', 'SUP-KLA-01', '+256772100201', 'g.namutebi@statistics.go.ug', 'Central', 'Kampala', 5),
  ('sup_jja_01', 'Peter Okello', 'SUP-JJA-01', '+256782200302', 'p.okello@statistics.go.ug', 'Eastern', 'Jinja', 4),
  ('sup_mbr_01', 'David Tumusiime', 'SUP-MBR-01', '+256701300403', 'd.tumusiime@statistics.go.ug', 'Western', 'Mbarara', 4),
  ('sup_glu_01', 'Akello Sarah', 'SUP-GLU-01', '+256774400504', 's.akello@statistics.go.ug', 'Northern', 'Gulu', 3)
ON CONFLICT (id) DO NOTHING;

-- Seed Enumerators
INSERT INTO enumerators (id, name, code, phone, email, supervisor_id, assigned_device_id, primary_region, languages, status, rating, total_submissions) VALUES
  ('enum_001', 'John Baptist Kato', 'ENUM-001', '+256771111111', 'j.kato@field.statgate.org', 'sup_kla_01', 'dev_tablet_01', 'Central', '["English","Luganda"]', 'Active', 4.9, 142),
  ('enum_002', 'Mary Kyomugisha', 'ENUM-002', '+256772222222', 'm.kyomugisha@field.statgate.org', 'sup_kla_01', 'dev_tablet_02', 'Central', '["English","Luganda","Runyankole"]', 'Active', 4.8, 128),
  ('enum_003', 'Moses Wafula', 'ENUM-003', '+256783333333', 'm.wafula@field.statgate.org', 'sup_jja_01', 'dev_tablet_03', 'Eastern', '["English","Lusoga","Lugwere"]', 'Active', 4.7, 95),
  ('enum_004', 'Emmanuel Mugisha', 'ENUM-004', '+256704444444', 'e.mugisha@field.statgate.org', 'sup_mbr_01', 'dev_tablet_04', 'Western', '["English","Runyankole","Rukiga"]', 'Active', 5.0, 110),
  ('enum_005', 'Susan Auma', 'ENUM-005', '+256775555555', 's.auma@field.statgate.org', 'sup_glu_01', 'dev_tablet_05', 'Northern', '["English","Acholi","Lango"]', 'Active', 4.9, 87)
ON CONFLICT (id) DO NOTHING;

-- Seed Assignment Plans
INSERT INTO assignment_plans (survey_id, ea_id, enumerator_id, supervisor_id, target_quota, completed_count, start_date, deadline, status, notes) VALUES
  ('sp_unhs_2026', 'ea_kla_001', 'enum_001', 'sup_kla_01', 35, 28, '2026-02-01', '2026-03-31', 'in_progress', 'Priority commercial and civic cluster.'),
  ('sp_unhs_2026', 'ea_kla_002', 'enum_002', 'sup_kla_01', 30, 22, '2026-02-01', '2026-03-31', 'in_progress', 'Residential affluent zone.'),
  ('sp_unhs_2026', 'ea_jja_001', 'enum_003', 'sup_jja_01', 35, 31, '2026-02-05', '2026-03-25', 'in_progress', 'High density urban sample cluster.'),
  ('sp_unhs_2026', 'ea_mbarara_001', 'enum_004', 'sup_mbr_01', 30, 30, '2026-02-01', '2026-03-15', 'completed', 'Completed ahead of schedule with 100% verification.'),
  ('sp_unhs_2026', 'ea_gulu_001', 'enum_005', 'sup_glu_01', 30, 18, '2026-02-10', '2026-04-10', 'in_progress', 'Bardege market community sample.')
ON CONFLICT DO NOTHING;

-- Seed Census Round: 2026 National Population and Housing Census
INSERT INTO census_rounds (id, name, round_year, legal_mandate, reference_night, pre_enumeration_status, enumeration_status, post_enumeration_status, total_projected_pop, total_enumerated_pop, households_enumerated, coverage_percentage) VALUES
  ('census_2026', '2026 National Population and Housing Census (NPHC 2026)', 2026, 'Uganda Bureau of Statistics Act (1998) & National Census Order 2025', '2026-05-09 23:59:59+03', 'completed', 'active', 'in_progress', 45850000, 42180000, 9420000, 92.0)
ON CONFLICT (id) DO NOTHING;

-- Seed Census PES Chandra-Sekar-Deming evaluation
INSERT INTO census_pes (census_round_id, pes_sample_eas_count, pes_sample_size, matched_records, census_only_records, pes_only_records, estimated_true_population, net_undercount_rate, coverage_rate, gross_omission_rate, confidence_interval_95) VALUES
  ('census_2026', 450, 185000, 172050, 9950, 12950, 45350000, 7.0, 93.0, 7.0, '{"lower":5.8,"upper":8.2}')
ON CONFLICT DO NOTHING;

-- Seed UN SDG Indicators (Official UN Framework across all 17 Goals)
INSERT INTO sdg_indicators (id, goal_number, goal_title, target_code, target_desc, indicator_code, indicator_desc, tier, custodian_agency, baseline_value, baseline_year, latest_value, latest_year, target_2030, unit, status, data_source) VALUES
  ('sdg_1_1_1', 1, 'No Poverty', '1.1', 'By 2030, eradicate extreme poverty for all people everywhere', '1.1.1', 'Proportion of the population living below the international poverty line ($2.15 a day)', 'Tier I', 'World Bank', 21.4, 2015, 14.8, 2026, 0.0, '%', 'On Track', 'National Household Survey (UNHS)'),
  ('sdg_1_2_1', 1, 'No Poverty', '1.2', 'Reduce by at least half the proportion of men, women and children living in poverty', '1.2.1', 'Proportion of population living below the national poverty line', 'Tier I', 'World Bank / NSO', 28.0, 2015, 20.3, 2026, 10.0, '%', 'On Track', 'National Household Survey'),
  ('sdg_2_1_1', 2, 'Zero Hunger', '2.1', 'By 2030, end hunger and ensure access by all people to safe, nutritious and sufficient food', '2.1.1', 'Prevalence of undernourishment in the total population', 'Tier I', 'FAO', 24.5, 2015, 18.2, 2026, 5.0, '%', 'On Track', 'Agricultural & Consumption Survey'),
  ('sdg_3_1_1', 3, 'Good Health & Well-being', '3.1', 'By 2030, reduce the global maternal mortality ratio to less than 70 per 100,000 live births', '3.1.1', 'Maternal mortality ratio per 100,000 live births', 'Tier I', 'WHO / UNICEF', 336.0, 2015, 189.0, 2026, 70.0, 'per 100k', 'On Track', 'DHS & Health Management Information System (HMIS)'),
  ('sdg_3_2_1', 3, 'Good Health & Well-being', '3.2', 'By 2030, end preventable deaths of newborns and children under 5 years of age', '3.2.1', 'Under-five mortality rate per 1,000 live births', 'Tier I', 'UNICEF / WHO', 64.0, 2015, 42.0, 2026, 25.0, 'per 1k', 'On Track', 'DHS & Civil Registration'),
  ('sdg_4_1_1', 4, 'Quality Education', '4.1', 'Ensure that all girls and boys complete free, equitable and quality primary and secondary education', '4.1.1', 'Proportion of children achieving minimum proficiency in reading and mathematics', 'Tier I', 'UNESCO-UIS', 38.5, 2015, 54.2, 2026, 85.0, '%', 'On Track', 'National Assessment of Progress in Education (NAPE)'),
  ('sdg_5_5_1', 5, 'Gender Equality', '5.5', 'Ensure women’s full and effective participation and equal opportunities for leadership', '5.5.1', 'Proportion of seats held by women in national parliaments and local governments', 'Tier I', 'IPU / UN Women', 34.0, 2015, 39.5, 2026, 50.0, '%', 'On Track', 'Electoral Commission Administrative Records'),
  ('sdg_6_1_1', 6, 'Clean Water & Sanitation', '6.1', 'By 2030, achieve universal and equitable access to safe and affordable drinking water', '6.1.1', 'Proportion of population using safely managed drinking water services', 'Tier I', 'WHO / UNICEF JMP', 48.0, 2015, 68.4, 2026, 100.0, '%', 'On Track', 'UNHS & Water Supply Atlas'),
  ('sdg_6_2_1', 6, 'Clean Water & Sanitation', '6.2', 'Achieve access to adequate and equitable sanitation and hygiene for all', '6.2.1', 'Proportion of population using safely managed sanitation services including handwashing', 'Tier I', 'WHO / UNICEF JMP', 19.0, 2015, 36.2, 2026, 100.0, '%', 'On Track', 'UNHS & WASH Inventory'),
  ('sdg_7_1_1', 7, 'Affordable & Clean Energy', '7.1', 'By 2030, ensure universal access to affordable, reliable and modern energy services', '7.1.1', 'Proportion of population with access to electricity', 'Tier I', 'World Bank / IEA', 20.4, 2015, 52.8, 2026, 100.0, '%', 'On Track', 'Rural Electrification Agency & UNHS'),
  ('sdg_8_5_2', 8, 'Decent Work & Economic Growth', '8.5', 'By 2030, achieve full and productive employment and decent work for all women and men', '8.5.2', 'Unemployment rate, by sex, age and persons with disabilities', 'Tier I', 'ILO', 9.2, 2015, 6.8, 2026, 4.0, '%', 'On Track', 'National Labor Force Survey (NLFS)'),
  ('sdg_9_2_1', 9, 'Industry, Innovation & Infrastructure', '9.2', 'Promote inclusive and sustainable industrialization', '9.2.1', 'Manufacturing value added as a proportion of GDP and per capita', 'Tier I', 'UNIDO', 12.5, 2015, 17.2, 2026, 25.0, '%', 'On Track', 'National Accounts (Macroeconomic Statistics)'),
  ('sdg_10_1_1', 10, 'Reduced Inequalities', '10.1', 'Progressively achieve and sustain income growth of the bottom 40 per cent of the population', '10.1.1', 'Growth rates of household expenditure or income per capita among the bottom 40 per cent', 'Tier I', 'World Bank', 2.8, 2015, 4.1, 2026, 6.0, '%', 'On Track', 'Poverty & Consumption Dynamics'),
  ('sdg_11_1_1', 11, 'Sustainable Cities & Communities', '11.1', 'Ensure access for all to adequate, safe and affordable housing and basic services', '11.1.1', 'Proportion of urban population living in slums, informal settlements or inadequate housing', 'Tier I', 'UN-Habitat', 54.0, 2015, 41.2, 2026, 20.0, '%', 'On Track', 'Urban Profiles & Census Cartography'),
  ('sdg_12_2_1', 12, 'Responsible Consumption & Production', '12.2', 'Achieve the sustainable management and efficient use of natural resources', '12.2.1', 'Material footprint, material footprint per capita, and material footprint per GDP', 'Tier I', 'UNEP', 4.1, 2015, 3.4, 2026, 2.5, 'tons/capita', 'On Track', 'Environmental-Economic Accounts'),
  ('sdg_13_1_1', 13, 'Climate Action', '13.1', 'Strengthen resilience and adaptive capacity to climate-related hazards and natural disasters', '13.1.1', 'Number of deaths, missing persons and directly affected persons attributed to disasters per 100,000 population', 'Tier I', 'UNDRR', 18.2, 2015, 8.5, 2026, 2.0, 'per 100k', 'On Track', 'Disaster Loss Database & Met Agency'),
  ('sdg_14_5_1', 14, 'Life Below Water', '14.5', 'Conserve at least 10 per cent of coastal and marine areas / freshwater ecosystems', '14.5.1', 'Coverage of protected areas in relation to marine/freshwater areas', 'Tier I', 'UNEP-WCMC', 14.8, 2015, 18.2, 2026, 25.0, '%', 'Target Met', 'National Protected Areas System (NFA/UWA)'),
  ('sdg_15_1_1', 15, 'Life on Land', '15.1', 'Ensure the conservation, restoration and sustainable use of terrestrial and inland freshwater ecosystems', '15.1.1', 'Forest area as a proportion of total land area', 'Tier I', 'FAO', 11.2, 2015, 13.4, 2026, 18.0, '%', 'On Track', 'National Forestry Authority (NFA) GIS'),
  ('sdg_16_1_1', 16, 'Peace, Justice & Strong Institutions', '16.1', 'Significantly reduce all forms of violence and related death rates everywhere', '16.1.1', 'Number of victims of intentional homicide per 100,000 population, by sex and age', 'Tier I', 'UNODC / WHO', 9.8, 2015, 4.6, 2026, 2.0, 'per 100k', 'On Track', 'Police Crime & Safety Survey'),
  ('sdg_17_1_1', 17, 'Partnerships for the Goals', '17.1', 'Strengthen domestic resource mobilization', '17.1.1', 'Total government revenue as a proportion of GDP, by source', 'Tier I', 'IMF', 13.5, 2015, 16.8, 2026, 22.0, '%', 'On Track', 'Revenue Authority & Government Finance Statistics')
ON CONFLICT (id) DO NOTHING;

-- Seed Statistical Publications in Dissemination Portal
INSERT INTO statistical_publications (id, title, abstract, publication_type, domain, survey_id, census_id, access_level, download_url, file_format) VALUES
  ('pub_unhs_prelim', 'Uganda National Household Survey 2026: Preliminary Key Indicators Report', 'Key findings on poverty headcount, employment rates, household asset ownership, and access to basic services.', 'bulletin', 'Social & Economic', 'sp_unhs_2026', NULL, 'public', '/api/v1/open-data/download?id=pub_unhs_prelim', 'PDF'),
  ('pub_census_cartography', 'National Population & Housing Census 2026: Digital Cartography and EA Atlas', 'Complete administrative boundary delineation, EA profiles, and pre-enumeration listing totals.', 'census_report', 'Demography & Census', NULL, 'census_2026', 'public', '/api/v1/open-data/download?id=pub_census_cartography', 'PDF'),
  ('pub_sdg_report_2026', 'National Voluntary Review (VNR) 2026: SDG Progress Assessment', 'Official reporting on the 17 Sustainable Development Goals, baseline trends, and 2030 trajectory indicators.', 'analytical_report', 'Sustainable Development', NULL, NULL, 'public', '/api/v1/open-data/download?id=pub_sdg_report_2026', 'PDF'),
  ('pub_microdata_unhs_anonymized', 'UNHS 2026 Anonymized Public-Use Microdata Sample (PUMS)', 'Cleaned, top-coded, k-anonymized household survey records for academic and policy researchers.', 'microdata_release', 'Microdata', 'sp_unhs_2026', NULL, 'registered', '/api/v1/open-data/download?id=pub_microdata_unhs_anonymized', 'CSV')
ON CONFLICT (id) DO NOTHING;

-- Seed Sample Official Tabulation Output
INSERT INTO tabulation_outputs (id, title, table_number, survey_id, row_var, col_var, measure_var, aggregation, is_weighted, matrix_data, row_totals, col_totals, grand_total, statistical_tests, status) VALUES
  ('tab_pop_sex_region', 'Table 1.1: Estimated Population Distribution by Region and Sex', 'Table 1.1', 'sp_unhs_2026', 'region', 'sex', 'population', 'SUM', true,
   '{"Central":{"Male":8240000,"Female":8510000},"Eastern":{"Male":6650000,"Female":6850000},"Western":{"Male":5350000,"Female":5550000},"Northern":{"Male":4750000,"Female":4950000}}',
   '{"Central":16750000,"Eastern":13500000,"Western":10900000,"Northern":9700000}',
   '{"Male":24990000,"Female":25860000}',
   50850000,
   '[{"test_name":"Chi-Square Test of Independence","statistic":4.82,"degrees_of_freedom":3,"p_value":0.185,"significant":false}]',
   'published'),
  ('tab_water_urban_rural', 'Table 3.4: Main Drinking Water Source by Urban/Rural Residence (%)', 'Table 3.4', 'sp_unhs_2026', 'water_source', 'residence', 'percentage', 'PERCENTAGE', true,
   '{"Piped Water":{"Urban":68.5,"Rural":14.2},"Borehole":{"Urban":18.2,"Rural":58.4},"Protected Spring":{"Urban":8.1,"Rural":16.5},"Surface Water":{"Urban":5.2,"Rural":10.9}}',
   '{"Piped Water":32.5,"Borehole":45.2,"Protected Spring":13.6,"Surface Water":8.7}',
   '{"Urban":100.0,"Rural":100.0}',
   100.0,
   '[{"test_name":"Chi-Square Test of Independence","statistic":214.6,"degrees_of_freedom":3,"p_value":0.0001,"significant":true}]',
   'published')
ON CONFLICT (id) DO NOTHING;
