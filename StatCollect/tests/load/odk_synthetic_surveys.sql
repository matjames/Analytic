\set ON_ERROR_STOP on

-- Re-runnable, deterministic StatCollect load fixture.
-- Override with psql -v rows_per_survey=... -v tenant_id=...
\if :{?rows_per_survey}
\else
  \set rows_per_survey 100001
\endif
\if :{?tenant_id}
\else
  \set tenant_id default
\endif

BEGIN;

CREATE TEMP TABLE synthetic_survey_catalog (
  form_id text PRIMARY KEY,
  title text NOT NULL,
  sector text NOT NULL,
  subject text NOT NULL,
  unit text NOT NULL
);

INSERT INTO synthetic_survey_catalog VALUES
  ('odk_health_facility_v1',     'Health Facility Service Readiness', 'Health',                 'facility',   'patients'),
  ('odk_crop_household_v1',      'Agriculture and Food Security',     'Agriculture',            'household',  'hectares'),
  ('odk_school_quality_v1',      'Education and School Quality',      'Education',              'school',     'learners'),
  ('odk_water_sanitation_v1',    'Water and Sanitation Access',       'WASH',                   'waterpoint', 'litres'),
  ('odk_microenterprise_v1',     'Microenterprise Economic Survey',   'Business',               'enterprise', 'employees'),
  ('odk_transport_mobility_v1',  'Transport and Mobility Survey',     'Transport',              'trip',       'kilometres'),
  ('odk_housing_energy_v1',      'Housing and Household Energy',      'Housing and Energy',     'dwelling',   'kilowatts'),
  ('odk_environment_v1',         'Environment and Biodiversity',      'Environment',            'site',       'hectares'),
  ('odk_social_protection_v1',   'Social Protection Monitoring',      'Social Protection',      'beneficiary','payments'),
  ('odk_disaster_response_v1',   'Disaster Impact Rapid Assessment',  'Disaster Management',    'incident',   'people');

-- Register real questionnaire definitions. "type" values follow XLSForm/ODK names.
INSERT INTO templates (id, name, description, version, schema, status, created_by, tenant_id, is_shared)
SELECT c.form_id, c.title, 'Synthetic high-volume ODK test form for the ' || c.sector || ' sector', '1.0',
       jsonb_build_object(
         'form_id', c.form_id, 'title', c.title, 'sector', c.sector, 'version', '1.0',
         'fields', jsonb_build_array(
           jsonb_build_object('key','respondent_name','label','Respondent name','type','text','required',true),
           jsonb_build_object('key','respondent_age','label','Respondent age','type','integer','required',true),
           jsonb_build_object('key','measurement','label','Primary measurement ('||c.unit||')','type','decimal','required',true),
           jsonb_build_object('key','category','label','Category','type','select_one','choices',jsonb_build_array('public','private','community','other')),
           jsonb_build_object('key','services','label','Services available','type','select_multiple','choices',jsonb_build_array('basic','advanced','emergency','outreach')),
           jsonb_build_object('key','visit_date','label','Visit date','type','date'),
           jsonb_build_object('key','visit_time','label','Visit time','type','time'),
           jsonb_build_object('key','start_at','label','Survey start','type','start'),
           jsonb_build_object('key','end_at','label','Survey end','type','end'),
           jsonb_build_object('key','gps_coordinates','label','Location','type','geopoint','required',true),
           jsonb_build_object('key','route','label','Survey route','type','geotrace'),
           jsonb_build_object('key','boundary','label','Survey boundary','type','geoshape'),
           jsonb_build_object('key','consent','label','Consent given','type','acknowledge','required',true),
           jsonb_build_object('key','asset_barcode','label','Asset barcode','type','barcode'),
           jsonb_build_object('key','contact_phone','label','Contact phone','type','text'),
           jsonb_build_object('key','email','label','Contact email','type','text'),
           jsonb_build_object('key','quality_score','label','Calculated quality score','type','calculate'),
           jsonb_build_object('key','photo','label','Site photo','type','image'),
           jsonb_build_object('key','audio_note','label','Audio note','type','audio'),
           jsonb_build_object('key','field_video','label','Field video','type','video'),
           jsonb_build_object('key','signature','label','Respondent signature','type','image'),
           jsonb_build_object('key','comments','label','Enumerator comments','type','text','appearance','multiline'),
           jsonb_build_object('key','enumerator_id','label','Enumerator','type','text'),
           jsonb_build_object('key','device_id','label','Device','type','deviceid'),
           jsonb_build_object('key','subscriber_id','label','Subscriber','type','subscriberid'),
           jsonb_build_object('key','sim_serial','label','SIM serial','type','simserial'),
           jsonb_build_object('key','phone_number','label','Device phone','type','phonenumber'),
           jsonb_build_object('key','username','label','ODK username','type','username'),
           jsonb_build_object('key','case_id','label','Case identifier','type','text'),
           jsonb_build_object('key','admin_region','label','Region','type','select_one'),
           jsonb_build_object('key','admin_district','label','District','type','select_one'),
           jsonb_build_object('key','urban','label','Urban location','type','boolean'),
           jsonb_build_object('key','household_size','label','Household or unit size','type','integer'),
           jsonb_build_object('key','income_band','label','Income band','type','select_one'),
           jsonb_build_object('key','satisfaction','label','Satisfaction score','type','range'),
           jsonb_build_object('key','risk_level','label','Observed risk','type','select_one'),
           jsonb_build_object('key','observation_count','label','Observation count','type','integer'),
           jsonb_build_object('key','cost_estimate','label','Estimated cost','type','decimal'),
           jsonb_build_object('key','language','label','Interview language','type','select_one'),
           jsonb_build_object('key','review_required','label','Requires review','type','boolean')
         )) AS schema,
       'active', 'synthetic-load-generator', :'tenant_id', false
FROM synthetic_survey_catalog c
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, description=EXCLUDED.description, schema=EXCLUDED.schema,
  updated_at=now(), tenant_id=EXCLUDED.tenant_id;

-- Delete only this fixture's deterministic ID range, making the load idempotent.
DELETE FROM submissions s
USING synthetic_survey_catalog c
WHERE s.form_id = c.form_id AND s.tenant_id = :'tenant_id'
  AND s.instance_id LIKE 'uuid:synthetic-%';

INSERT INTO submissions
  (instance_id, form_id, received_at, meta, xml, tenant_id, submitted_by, status, created_at, updated_at)
SELECT
  'uuid:synthetic-' || :'tenant_id' || '-' || c.form_id || '-' || lpad(g.n::text, 7, '0'),
  c.form_id,
  timestamptz '2025-01-01 06:00:00+00' + ((g.n + f.form_no * 7919) % 525600) * interval '1 minute',
  d.meta,
  format('<data id="%s" version="1.0"><meta><instanceID>uuid:synthetic-%s-%s-%s</instanceID></meta><respondent_name>%s</respondent_name><respondent_age>%s</respondent_age><measurement>%s</measurement><category>%s</category><services>%s</services><visit_date>%s</visit_date><visit_time>%s</visit_time><gps_coordinates>%s</gps_coordinates><consent>true</consent><asset_barcode>%s</asset_barcode><quality_score>%s</quality_score><comments>%s</comments></data>',
         c.form_id, :'tenant_id', c.form_id, lpad(g.n::text,7,'0'), d.meta->>'respondent_name', d.meta->>'respondent_age',
         d.meta->>'measurement', d.meta->>'category', d.meta->>'services', d.meta->>'visit_date',
         d.meta->>'visit_time', d.meta->>'gps_coordinates', d.meta->>'asset_barcode',
         d.meta->>'quality_score', d.meta->>'comments'),
  :'tenant_id', d.meta->>'enumerator_id',
  CASE g.n % 20 WHEN 0 THEN 'rejected' WHEN 1 THEN 'pending' WHEN 2 THEN 'received' ELSE 'approved' END,
  now(), now()
FROM synthetic_survey_catalog c
CROSS JOIN LATERAL (SELECT row_number() OVER (ORDER BY form_id) AS form_no FROM synthetic_survey_catalog x WHERE x.form_id <= c.form_id ORDER BY form_id DESC LIMIT 1) f
CROSS JOIN generate_series(1, :rows_per_survey::integer) g(n)
CROSS JOIN LATERAL (
  SELECT jsonb_build_object(
    'respondent_name', 'Synthetic Respondent '||g.n, 'respondent_age', 18+(g.n%73),
    'measurement', round((10+(g.n%10000)/17.0)::numeric,2), 'category', (ARRAY['public','private','community','other'])[1+(g.n%4)],
    'services', (ARRAY['basic outreach','basic advanced','emergency','advanced emergency'])[1+(g.n%4)],
    'visit_date', to_char(date '2025-01-01'+((g.n+f.form_no*31)%365)::int,'YYYY-MM-DD'),
    'visit_time', to_char(time '06:00'+((g.n%840)*interval '1 minute'),'HH24:MI:SS'),
    'start_at', to_char(timestamptz '2025-01-01 06:00+00'+(g.n%525600)*interval '1 minute','YYYY-MM-DD"T"HH24:MI:SSOF'),
    'end_at', to_char(timestamptz '2025-01-01 06:12+00'+(g.n%525600)*interval '1 minute','YYYY-MM-DD"T"HH24:MI:SSOF'),
    'gps_coordinates', (-1.45+((g.n%1000)/10000.0))||' '||(29.55+((g.n%1700)/10000.0))||' '||(1100+g.n%400)||' '||(3+g.n%12),
    'gps_lat', -1.45+((g.n%1000)/10000.0), 'gps_lon', 29.55+((g.n%1700)/10000.0), 'gps_accuracy', 3+(g.n%12),
    'route', '-1.40 29.60;-1.39 29.61', 'boundary', '-1.40 29.60;-1.40 29.61;-1.39 29.61;-1.40 29.60',
    'consent', true, 'asset_barcode', 'BC-'||f.form_no||'-'||lpad(g.n::text,8,'0'),
    'contact_phone', '+25078'||lpad((g.n%10000000)::text,7,'0'), 'email', 'synthetic.'||g.n||'@example.invalid',
    'quality_score', 60+(g.n%41), 'photo', 'photo_'||g.n||'.jpg', 'audio_note', 'audio_'||g.n||'.m4a',
    'field_video', 'video_'||g.n||'.mp4', 'signature', 'signature_'||g.n||'.png',
    'comments', 'Synthetic '||c.sector||' observation for '||c.subject, 'enumerator_id', 'enum-'||lpad((g.n%250)::text,3,'0'),
    'device_id', 'device-'||(g.n%80), 'subscriber_id', 'subscriber-'||(g.n%100), 'sim_serial', 'SIM'||lpad(g.n::text,12,'0'),
    'phone_number', '+25072'||lpad((g.n%10000000)::text,7,'0'), 'username', 'odk_enum_'||(g.n%250),
    'case_id', upper(substr(md5(c.form_id||g.n::text),1,16)), 'admin_region', (ARRAY['North','South','East','West','Central'])[1+(g.n%5)],
    'admin_district', 'District '||(1+g.n%30), 'urban', (g.n%3=0), 'household_size', 1+(g.n%12),
    'income_band', (ARRAY['low','lower_middle','upper_middle','high'])[1+(g.n%4)], 'satisfaction', g.n%11,
    'risk_level', (ARRAY['low','medium','high','critical'])[1+(g.n%4)], 'observation_count', g.n%50,
    'cost_estimate', round((25+(g.n%50000)/13.0)::numeric,2), 'language', (ARRAY['English','French','Kiswahili','Kinyarwanda'])[1+(g.n%4)],
    'review_required', (g.n%17=0), '_msh_sector', c.sector, '_msh_project', c.title,
    '_msh_organization', 'StatGate Synthetic Testing', '_msh_region', (ARRAY['North','South','East','West','Central'])[1+(g.n%5)],
    '_msh_district', 'District '||(1+g.n%30), '_msh_duration', 240+(g.n%1800)
  ) AS meta
) d;

ANALYZE submissions;
COMMIT;

WITH counts AS (
  SELECT s.form_id, count(*) AS submissions
  FROM submissions s
  WHERE s.tenant_id = :'tenant_id' AND s.form_id IN (SELECT form_id FROM synthetic_survey_catalog)
  GROUP BY s.form_id
)
SELECT counts.form_id, counts.submissions, fields.distinct_fields
FROM counts
CROSS JOIN LATERAL (
  SELECT count(*) AS distinct_fields
  FROM jsonb_object_keys((
    SELECT s.meta FROM submissions s
    WHERE s.tenant_id = :'tenant_id' AND s.form_id = counts.form_id LIMIT 1
  ))
) fields
ORDER BY counts.form_id;
