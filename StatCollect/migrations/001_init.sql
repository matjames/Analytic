-- create submissions and attachments tables
CREATE TABLE IF NOT EXISTS submissions (
  id BIGSERIAL PRIMARY KEY,
  instance_id TEXT UNIQUE NOT NULL,
  form_id TEXT,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  meta JSONB,
  xml TEXT
);

ALTER TABLE submissions ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS submitted_by TEXT;
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'received';
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE IF NOT EXISTS attachments (
  id BIGSERIAL PRIMARY KEY,
  submission_instance_id TEXT REFERENCES submissions(instance_id) ON DELETE CASCADE,
  filename TEXT NOT NULL,
  path TEXT NOT NULL,
  size BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE attachments ADD COLUMN IF NOT EXISTS content_type TEXT;

-- Object linkages: connect submissions to other StatGate platform objects
CREATE TABLE IF NOT EXISTS object_links (
  id BIGSERIAL PRIMARY KEY,
  source_type TEXT NOT NULL,
  source_id TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  relationship TEXT NOT NULL DEFAULT 'references',
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (source_type, source_id, target_type, target_id, relationship)
);

-- Cross-module event log (for audit / traceability)
CREATE TABLE IF NOT EXISTS event_log (
  id BIGSERIAL PRIMARY KEY,
  event_type TEXT NOT NULL,
  source TEXT NOT NULL,
  object_type TEXT NOT NULL,
  object_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  payload JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- StatChat object discussion links
CREATE TABLE IF NOT EXISTS statchat_links (
  id BIGSERIAL PRIMARY KEY,
  object_type TEXT NOT NULL,
  object_id TEXT NOT NULL,
  conversation_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (object_type, object_id)
);

-- Submission approvals / validation workflow
CREATE TABLE IF NOT EXISTS submission_validations (
  id BIGSERIAL PRIMARY KEY,
  instance_id TEXT NOT NULL REFERENCES submissions(instance_id) ON DELETE CASCADE,
  validator TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  notes TEXT,
  validated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Questionnaire templates (form definitions stored as JSON schema)
CREATE TABLE IF NOT EXISTS templates (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  version TEXT NOT NULL DEFAULT '1.0',
  schema JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_by TEXT,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  is_shared BOOLEAN NOT NULL DEFAULT false,
  parent_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE templates ADD COLUMN IF NOT EXISTS is_shared BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE templates ADD COLUMN IF NOT EXISTS parent_id TEXT;

CREATE INDEX IF NOT EXISTS idx_templates_tenant ON templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_templates_status ON templates(status);
CREATE INDEX IF NOT EXISTS idx_templates_shared ON templates(is_shared);
CREATE INDEX IF NOT EXISTS idx_templates_parent ON templates(parent_id);

-- Template version history
CREATE TABLE IF NOT EXISTS template_versions (
  id BIGSERIAL PRIMARY KEY,
  template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
  version TEXT NOT NULL,
  schema JSONB NOT NULL,
  changed_by TEXT,
  change_note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_template_versions_template ON template_versions(template_id);
CREATE INDEX IF NOT EXISTS idx_template_versions_created ON template_versions(created_at);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_submissions_tenant ON submissions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_submissions_form ON submissions(form_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
CREATE INDEX IF NOT EXISTS idx_attachments_submission ON attachments(submission_instance_id);
CREATE INDEX IF NOT EXISTS idx_object_links_source ON object_links(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_object_links_target ON object_links(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_event_log_object ON event_log(object_type, object_id);
CREATE INDEX IF NOT EXISTS idx_event_log_created ON event_log(created_at);

-- ── Phase Y Enterprise Capabilities Scemas ──

-- Managed Devices
CREATE TABLE IF NOT EXISTS devices (
  device_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  owner TEXT,
  status TEXT NOT NULL DEFAULT 'Active', -- Active, Suspended, Retired
  battery_level INT,
  storage_utilization FLOAT,
  os TEXT,
  app_version TEXT,
  security_compliant BOOLEAN DEFAULT true,
  last_sync_at TIMESTAMPTZ,
  registered_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Survey Assignments
CREATE TABLE IF NOT EXISTS assignments (
  id BIGSERIAL PRIMARY KEY,
  survey_id TEXT NOT NULL,
  target_type TEXT NOT NULL, -- enumerator, team, district, region, school, facility, boundary
  target_id TEXT NOT NULL,
  start_date TIMESTAMPTZ,
  end_date TIMESTAMPTZ,
  deadline TIMESTAMPTZ,
  daily_target INT DEFAULT 0,
  completion_threshold FLOAT DEFAULT 0.0,
  reminder_cron TEXT,
  status TEXT NOT NULL DEFAULT 'Active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Registries (Individuals, Households, Facilities, etc.)
CREATE TABLE IF NOT EXISTS registries (
  id TEXT PRIMARY KEY,
  registry_type TEXT NOT NULL, -- individual, household, health_facility, school, farm, business, community, infrastructure
  name TEXT NOT NULL,
  attributes JSONB NOT NULL DEFAULT '{}',
  parent_registry_id TEXT REFERENCES registries(id) ON DELETE SET NULL,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Longitudinal link from submissions to registry assets
CREATE TABLE IF NOT EXISTS longitudinal_links (
  id BIGSERIAL PRIMARY KEY,
  submission_instance_id TEXT NOT NULL REFERENCES submissions(instance_id) ON DELETE CASCADE,
  registry_id TEXT NOT NULL REFERENCES registries(id) ON DELETE CASCADE,
  visit_number INT NOT NULL DEFAULT 1,
  phase TEXT NOT NULL DEFAULT 'follow-up', -- baseline, midline, endline, follow-up
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sampling Designs
CREATE TABLE IF NOT EXISTS sampling_designs (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  method TEXT NOT NULL, -- simple_random, stratified, cluster, systematic, pps
  sample_size INT NOT NULL,
  seed BIGINT NOT NULL,
  frame_data JSONB NOT NULL,
  selected_ids JSONB NOT NULL,
  audit_trail TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Approval Workflows
CREATE TABLE IF NOT EXISTS survey_workflows (
  id BIGSERIAL PRIMARY KEY,
  survey_id TEXT UNIQUE NOT NULL,
  stages JSONB NOT NULL, -- array of states in order, e.g. ["draft", "qa", "approved"]
  current_stage TEXT NOT NULL DEFAULT 'draft',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Automated Schedules
CREATE TABLE IF NOT EXISTS survey_schedules (
  id BIGSERIAL PRIMARY KEY,
  survey_id TEXT NOT NULL,
  cron_expression TEXT NOT NULL,
  next_run_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'Active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Survey Comments / StatChat Threads
CREATE TABLE IF NOT EXISTS survey_comments (
  id BIGSERIAL PRIMARY KEY,
  object_type TEXT NOT NULL, -- survey, submission
  object_id TEXT NOT NULL,
  author TEXT NOT NULL,
  comment TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Enterprise Notifications
CREATE TABLE IF NOT EXISTS notifications (
  id BIGSERIAL PRIMARY KEY,
  user_id TEXT NOT NULL,
  channel TEXT NOT NULL, -- app, email, statchat
  message TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'Pending',
  sent_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_registries_type ON registries(registry_type);
CREATE INDEX IF NOT EXISTS idx_longitudinal_links_sub ON longitudinal_links(submission_instance_id);
CREATE INDEX IF NOT EXISTS idx_longitudinal_links_reg ON longitudinal_links(registry_id);

-- ── Phase Y Production Additions ──

-- Additional performance indexes
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
CREATE INDEX IF NOT EXISTS idx_assignments_survey ON assignments(survey_id);
CREATE INDEX IF NOT EXISTS idx_assignments_target ON assignments(target_id);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, status);
CREATE INDEX IF NOT EXISTS idx_comments_object ON survey_comments(object_type, object_id);
CREATE INDEX IF NOT EXISTS idx_schedules_survey ON survey_schedules(survey_id);
CREATE INDEX IF NOT EXISTS idx_workflows_survey ON survey_workflows(survey_id);

-- Template marketplace: add star rating
ALTER TABLE templates ADD COLUMN IF NOT EXISTS rating SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE templates ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'General';
ALTER TABLE templates ADD COLUMN IF NOT EXISTS download_count INT NOT NULL DEFAULT 0;

-- Multi-level approval pipeline: extend submission_validations
ALTER TABLE submission_validations ADD COLUMN IF NOT EXISTS stage TEXT NOT NULL DEFAULT 'enumerator';
ALTER TABLE submission_validations ADD COLUMN IF NOT EXISTS approver_role TEXT;
ALTER TABLE submission_validations ADD COLUMN IF NOT EXISTS escalated_to TEXT;
ALTER TABLE submission_validations ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_submission_validations_instance ON submission_validations(instance_id, stage);
CREATE INDEX IF NOT EXISTS idx_submission_validations_status ON submission_validations(status);

-- Enterprise Integration Hub: store external system connector configs
CREATE TABLE IF NOT EXISTS integrations (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  system_type TEXT NOT NULL, -- dhis2, odk_central, kobotoolbox, openmrs, redcap, gis, erp, fhir
  base_url TEXT NOT NULL,
  credentials JSONB NOT NULL DEFAULT '{}', -- encrypted connection details
  field_mapping JSONB NOT NULL DEFAULT '{}', -- field-level mapping
  status TEXT NOT NULL DEFAULT 'Active', -- Active, Paused, Error
  last_push_at TIMESTAMPTZ,
  push_count INT NOT NULL DEFAULT 0,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_integrations_type ON integrations(system_type);
CREATE INDEX IF NOT EXISTS idx_integrations_tenant ON integrations(tenant_id);

-- Notification dispatch log
CREATE TABLE IF NOT EXISTS notification_dispatch_log (
  id BIGSERIAL PRIMARY KEY,
  notification_id BIGINT REFERENCES notifications(id) ON DELETE CASCADE,
  channel TEXT NOT NULL,
  attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  success BOOLEAN NOT NULL DEFAULT false,
  error_message TEXT
);

-- Plugin registry
CREATE TABLE IF NOT EXISTS plugins (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '1.0',
  description TEXT,
  entry_point TEXT NOT NULL, -- JS module path or webhook URL
  plugin_type TEXT NOT NULL, -- biometric, nfc, gps_device, iot, imagery, custom
  config JSONB NOT NULL DEFAULT '{}',
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Business rules engine
CREATE TABLE IF NOT EXISTS business_rules (
  id BIGSERIAL PRIMARY KEY,
  template_id TEXT REFERENCES templates(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  rule_type TEXT NOT NULL, -- validation, calculation, skip_logic, approval_trigger, notification
  condition_expr TEXT NOT NULL, -- JSONLogic or custom expression
  action_expr TEXT NOT NULL,
  priority INT NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_business_rules_template ON business_rules(template_id);

-- ── Directive 21 — Mandatory Survey Header (MSH) ──────────────────────────

-- MSH configuration per template (which sections are enabled, required, etc.)
CREATE TABLE IF NOT EXISTS survey_headers (
  id BIGSERIAL PRIMARY KEY,
  template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
  enabled BOOLEAN NOT NULL DEFAULT true,
  -- Section toggles
  include_metadata BOOLEAN NOT NULL DEFAULT true,
  include_assignment BOOLEAN NOT NULL DEFAULT true,
  include_respondent BOOLEAN NOT NULL DEFAULT true,
  include_geography BOOLEAN NOT NULL DEFAULT true,
  include_collection BOOLEAN NOT NULL DEFAULT true,
  include_consent BOOLEAN NOT NULL DEFAULT true,
  include_attachments BOOLEAN NOT NULL DEFAULT false,
  include_audit BOOLEAN NOT NULL DEFAULT true,
  -- Respondent configuration (JSON array of active respondent types)
  respondent_types JSONB NOT NULL DEFAULT '["individual","household","facility"]',
  -- Country code for admin level configuration
  country_code TEXT NOT NULL DEFAULT 'UGA',
  -- Consent configuration (JSON: verbal, written, guardian, digital_signature, audio)
  consent_config JSONB NOT NULL DEFAULT '{"verbal":true,"written":false,"guardian":false,"digital_signature":true,"audio":false}',
  -- Platform integration: which modules receive published data
  publish_to_research BOOLEAN NOT NULL DEFAULT true,
  publish_to_projects BOOLEAN NOT NULL DEFAULT true,
  publish_to_statistics BOOLEAN NOT NULL DEFAULT true,
  publish_to_gis BOOLEAN NOT NULL DEFAULT true,
  publish_to_reporting BOOLEAN NOT NULL DEFAULT true,
  publish_to_documents BOOLEAN NOT NULL DEFAULT false,
  -- Auto-task generation on events
  auto_create_tasks BOOLEAN NOT NULL DEFAULT true,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (template_id)
);

-- Configurable respondent type registry (platform-wide)
CREATE TABLE IF NOT EXISTS msh_respondent_types (
  id TEXT PRIMARY KEY, -- individual, household, facility, school, business, farm, community, institution
  label TEXT NOT NULL,
  description TEXT,
  common_fields JSONB NOT NULL DEFAULT '[]', -- field definitions specific to this type
  enabled BOOLEAN NOT NULL DEFAULT true,
  sort_order INT NOT NULL DEFAULT 0,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Country-configurable administrative boundary levels
CREATE TABLE IF NOT EXISTS msh_admin_levels (
  id BIGSERIAL PRIMARY KEY,
  country_code TEXT NOT NULL,  -- ISO 3166-1 alpha-3, e.g. UGA, KEN, TZA
  level_number INT NOT NULL,   -- 1=Country, 2=Region, 3=District, 4=County, 5=Sub-county, 6=Parish, 7=Village
  level_name TEXT NOT NULL,    -- e.g. "Region", "District", "Tehsil"
  required BOOLEAN NOT NULL DEFAULT false,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (country_code, level_number)
);

-- MSH consent workflow configurations per template
CREATE TABLE IF NOT EXISTS msh_consent_configs (
  id BIGSERIAL PRIMARY KEY,
  template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
  consent_type TEXT NOT NULL, -- verbal, written, guardian, digital_signature, audio
  required BOOLEAN NOT NULL DEFAULT false,
  participant_info_text TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Directive 22 — Unified Platform Integration ────────────────────────────

-- Cross-module object links (submission → Research Project, Task, GIS Layer, Report, Document)
CREATE TABLE IF NOT EXISTS platform_links (
  id BIGSERIAL PRIMARY KEY,
  source_module TEXT NOT NULL,    -- statcollect
  source_type TEXT NOT NULL,      -- submission, template, attachment
  source_id TEXT NOT NULL,
  target_module TEXT NOT NULL,    -- research, projects, gis, statistics, reporting, documents, statchat, tasks
  target_type TEXT NOT NULL,      -- project, dataset, layer, report, document, task
  target_id TEXT NOT NULL,
  relationship TEXT NOT NULL DEFAULT 'contributes_to',
  metadata JSONB NOT NULL DEFAULT '{}',
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (source_module, source_type, source_id, target_module, target_type, target_id)
);

-- Workflow tasks auto-generated from survey lifecycle events
CREATE TABLE IF NOT EXISTS workflow_tasks (
  id BIGSERIAL PRIMARY KEY,
  task_type TEXT NOT NULL,          -- review_submission, validate_gps, correct_quality, approve_dataset, reassign_visit
  title TEXT NOT NULL,
  description TEXT,
  assigned_to TEXT,                  -- user or role
  assigned_role TEXT,
  source_module TEXT NOT NULL DEFAULT 'statcollect',
  source_type TEXT NOT NULL,        -- submission, template
  source_id TEXT NOT NULL,
  priority TEXT NOT NULL DEFAULT 'normal', -- low, normal, high, critical
  status TEXT NOT NULL DEFAULT 'open',     -- open, in_progress, completed, cancelled
  due_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Dashboard subscription registry (which dashboards refresh on submission events)
CREATE TABLE IF NOT EXISTS dashboard_subscriptions (
  id BIGSERIAL PRIMARY KEY,
  dashboard_id TEXT NOT NULL,
  dashboard_name TEXT NOT NULL,
  module TEXT NOT NULL,              -- projects, statistics, gis, reporting
  form_id TEXT,                      -- subscribe to specific form, or NULL for all
  event_type TEXT NOT NULL DEFAULT 'submission.approved', -- submission.received, submission.approved, submission.rejected
  last_triggered_at TIMESTAMPTZ,
  enabled BOOLEAN NOT NULL DEFAULT true,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Statistics export tracking (which submissions have been pushed to Statistics module)
CREATE TABLE IF NOT EXISTS stats_exports (
  id BIGSERIAL PRIMARY KEY,
  submission_instance_id TEXT NOT NULL REFERENCES submissions(instance_id) ON DELETE CASCADE,
  form_id TEXT NOT NULL,
  dataset_id TEXT,                   -- Statistics module dataset ID
  export_status TEXT NOT NULL DEFAULT 'pending', -- pending, exported, failed
  export_attempt INT NOT NULL DEFAULT 0,
  exported_at TIMESTAMPTZ,
  error_message TEXT,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (submission_instance_id)
);

-- GIS synchronization queue (GPS data from approved submissions)
CREATE TABLE IF NOT EXISTS gis_sync_queue (
  id BIGSERIAL PRIMARY KEY,
  submission_instance_id TEXT NOT NULL REFERENCES submissions(instance_id) ON DELETE CASCADE,
  form_id TEXT NOT NULL,
  gis_layer_id TEXT,                 -- target GIS layer/dataset ID
  latitude DOUBLE PRECISION,
  longitude DOUBLE PRECISION,
  accuracy FLOAT,
  feature_properties JSONB NOT NULL DEFAULT '{}',
  sync_status TEXT NOT NULL DEFAULT 'pending', -- pending, synced, failed, skipped
  sync_attempt INT NOT NULL DEFAULT 0,
  synced_at TIMESTAMPTZ,
  error_message TEXT,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (submission_instance_id)
);

-- Document management links (surveys, attachments, consent forms, exports)
CREATE TABLE IF NOT EXISTS document_links (
  id BIGSERIAL PRIMARY KEY,
  source_module TEXT NOT NULL DEFAULT 'statcollect',
  source_type TEXT NOT NULL,         -- template, submission, attachment, report
  source_id TEXT NOT NULL,
  doc_title TEXT NOT NULL,
  doc_url TEXT NOT NULL,
  doc_type TEXT NOT NULL DEFAULT 'file', -- file, form, report, consent, export
  stored_at TEXT,                    -- Document Management System path
  tenant_id TEXT NOT NULL DEFAULT 'default',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Platform event publish log (audit of all cross-module pushes)
CREATE TABLE IF NOT EXISTS platform_publish_log (
  id BIGSERIAL PRIMARY KEY,
  source_module TEXT NOT NULL DEFAULT 'statcollect',
  event_type TEXT NOT NULL,           -- submission.approved, submission.received, etc.
  source_id TEXT NOT NULL,
  target_module TEXT NOT NULL,
  target_endpoint TEXT,
  http_status INT,
  success BOOLEAN NOT NULL DEFAULT false,
  error_message TEXT,
  payload_summary JSONB,
  published_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for Directive 21 & 22
CREATE INDEX IF NOT EXISTS idx_survey_headers_template ON survey_headers(template_id);
CREATE INDEX IF NOT EXISTS idx_msh_admin_levels_country ON msh_admin_levels(country_code);
CREATE INDEX IF NOT EXISTS idx_msh_consent_template ON msh_consent_configs(template_id);
CREATE INDEX IF NOT EXISTS idx_platform_links_source ON platform_links(source_module, source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_platform_links_target ON platform_links(target_module, target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_assigned ON workflow_tasks(assigned_to, status);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_source ON workflow_tasks(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_subscriptions_module ON dashboard_subscriptions(module, event_type);
CREATE INDEX IF NOT EXISTS idx_stats_exports_status ON stats_exports(export_status);
CREATE INDEX IF NOT EXISTS idx_gis_sync_queue_status ON gis_sync_queue(sync_status);
CREATE INDEX IF NOT EXISTS idx_platform_publish_log_source ON platform_publish_log(source_id, target_module);

-- Seed default respondent types
INSERT INTO msh_respondent_types (id, label, description, sort_order) VALUES
  ('individual',   'Individual',   'A single person respondent',                          1),
  ('household',    'Household',    'A household unit with a head of household',            2),
  ('facility',     'Health Facility', 'A health facility (clinic, hospital, dispensary)', 3),
  ('school',       'School',       'An educational institution',                          4),
  ('business',     'Business/Enterprise', 'A business or commercial enterprise',          5),
  ('farm',         'Farm/Agricultural Unit', 'An agricultural production unit',           6),
  ('community',    'Community',    'A community or village group',                        7),
  ('institution',  'Institution',  'A government or NGO institution',                     8)
ON CONFLICT (id) DO NOTHING;

-- Seed default admin levels for Uganda (UGA)
INSERT INTO msh_admin_levels (country_code, level_number, level_name, required) VALUES
  ('UGA', 1, 'Country',     true),
  ('UGA', 2, 'Region',      true),
  ('UGA', 3, 'District',    true),
  ('UGA', 4, 'County',      false),
  ('UGA', 5, 'Sub-county',  false),
  ('UGA', 6, 'Parish',      false),
  ('UGA', 7, 'Village',     false)
ON CONFLICT (country_code, level_number) DO NOTHING;

-- Seed admin levels for Kenya (KEN)
INSERT INTO msh_admin_levels (country_code, level_number, level_name, required) VALUES
  ('KEN', 1, 'Country',   true),
  ('KEN', 2, 'County',    true),
  ('KEN', 3, 'Sub-County', true),
  ('KEN', 4, 'Ward',      false),
  ('KEN', 5, 'Village',   false)
ON CONFLICT (country_code, level_number) DO NOTHING;

-- Seed admin levels for Tanzania (TZA)
INSERT INTO msh_admin_levels (country_code, level_number, level_name, required) VALUES
  ('TZA', 1, 'Country',  true),
  ('TZA', 2, 'Region',   true),
  ('TZA', 3, 'District', true),
  ('TZA', 4, 'Division', false),
  ('TZA', 5, 'Ward',     false),
  ('TZA', 6, 'Village',  false)
ON CONFLICT (country_code, level_number) DO NOTHING;
