-- ══════════════════════════════════════════════════════════════════════
-- STATGATE APP 8: IoT, SENSORS & MOBILE FIELD OPS (001_init.sql)
-- Phases: P27 (IoT & Edge Computing) + P43 (Mobile & Offline Field Ops)
-- ══════════════════════════════════════════════════════════════════════

CREATE SCHEMA IF NOT EXISTS statiot;
SET search_path TO statiot, public;

-- ──────────────────────────────────────────────────────────────────────
-- 1. IoT INFRASTRUCTURE & GATEWAYS (P27)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.iot_gateways (
    id                  VARCHAR(64) PRIMARY KEY,
    name                VARCHAR(255) NOT NULL,
    gateway_code        VARCHAR(64) UNIQUE NOT NULL,
    ip_address          VARCHAR(64),
    mac_address         VARCHAR(64),
    firmware_version    VARCHAR(64) DEFAULT '1.0.0',
    status              VARCHAR(32) NOT NULL DEFAULT 'OFFLINE', -- ONLINE, OFFLINE, DEGRADED, MAINTENANCE
    latitude            DOUBLE PRECISION,
    longitude           DOUBLE PRECISION,
    location_name       VARCHAR(255),
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    metadata            JSONB DEFAULT '{}'::jsonb,
    last_heartbeat      TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_iot_gateways_tenant ON statiot.iot_gateways(tenant_id);
CREATE INDEX IF NOT EXISTS idx_iot_gateways_status ON statiot.iot_gateways(status);

-- ──────────────────────────────────────────────────────────────────────
-- 2. IoT DEVICE REGISTRY & SENSORS (P27)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.iot_devices (
    id                  VARCHAR(64) PRIMARY KEY,
    device_uid          VARCHAR(128) UNIQUE NOT NULL, -- Hardware Serial / IMEI / MAC
    name                VARCHAR(255) NOT NULL,
    device_type         VARCHAR(64) NOT NULL, -- WEATHER_STATION, WATER_QUALITY, AIR_QUALITY, SOIL_SENSOR, COLD_CHAIN, ENERGY_METER, DRONE, LAB_DEVICE
    protocol            VARCHAR(32) NOT NULL DEFAULT 'MQTT', -- MQTT, HTTP, COAP, LORAWAN, MODBUS
    gateway_id          VARCHAR(64) REFERENCES statiot.iot_gateways(id) ON DELETE SET NULL,
    auth_token_hash     VARCHAR(255),
    firmware_version    VARCHAR(64) DEFAULT '1.0.0',
    status              VARCHAR(32) NOT NULL DEFAULT 'REGISTERED', -- REGISTERED, ACTIVE, INACTIVE, DECOMMISSIONED
    battery_level       DOUBLE PRECISION,
    signal_strength_dbm INTEGER,
    latitude            DOUBLE PRECISION,
    longitude           DOUBLE PRECISION,
    altitude            DOUBLE PRECISION,
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    org_id              VARCHAR(64),
    config_payload      JSONB DEFAULT '{}'::jsonb,
    tags                TEXT[] DEFAULT ARRAY[]::TEXT[],
    last_seen_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_iot_devices_tenant ON statiot.iot_devices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_iot_devices_type ON statiot.iot_devices(device_type);
CREATE INDEX IF NOT EXISTS idx_iot_devices_status ON statiot.iot_devices(status);

CREATE TABLE IF NOT EXISTS statiot.sensor_registry (
    id                  VARCHAR(64) PRIMARY KEY,
    device_id           VARCHAR(64) NOT NULL REFERENCES statiot.iot_devices(id) ON DELETE CASCADE,
    sensor_code         VARCHAR(64) NOT NULL,
    metric_name         VARCHAR(128) NOT NULL, -- temperature, humidity, pm2_5, ph, voltage, moisture
    unit_of_measure     VARCHAR(32) NOT NULL,  -- celsius, percentage, ug_m3, ph, volts, bar
    min_threshold       DOUBLE PRECISION,
    max_threshold       DOUBLE PRECISION,
    calibration_factor  DOUBLE PRECISION DEFAULT 1.0,
    calibration_offset  DOUBLE PRECISION DEFAULT 0.0,
    status              VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id, sensor_code)
);

CREATE INDEX IF NOT EXISTS idx_sensor_registry_device ON statiot.sensor_registry(device_id);

-- ──────────────────────────────────────────────────────────────────────
-- 3. HIGH-THROUGHPUT TIME-SERIES TELEMETRY (P27 - TimescaleDB compatible)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.telemetry_records (
    id                  VARCHAR(64) NOT NULL,
    device_id           VARCHAR(64) NOT NULL,
    sensor_code         VARCHAR(64) NOT NULL,
    metric_name         VARCHAR(128) NOT NULL,
    value               DOUBLE PRECISION NOT NULL,
    raw_payload         JSONB DEFAULT '{}'::jsonb,
    quality_score       DOUBLE PRECISION DEFAULT 1.0,
    is_anomaly          BOOLEAN DEFAULT FALSE,
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    recorded_at         TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, recorded_at)
);

CREATE INDEX IF NOT EXISTS idx_telemetry_device_time ON statiot.telemetry_records(device_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_telemetry_metric_time ON statiot.telemetry_records(metric_name, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_telemetry_tenant_time ON statiot.telemetry_records(tenant_id, recorded_at DESC);

-- ──────────────────────────────────────────────────────────────────────
-- 4. EDGE CONFIGURATION & OTA FIRMWARE (P27)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.edge_configurations (
    id                  VARCHAR(64) PRIMARY KEY,
    device_id           VARCHAR(64) NOT NULL REFERENCES statiot.iot_devices(id) ON DELETE CASCADE,
    version             INTEGER NOT NULL DEFAULT 1,
    desired_config      JSONB NOT NULL DEFAULT '{}'::jsonb,
    reported_config     JSONB DEFAULT '{}'::jsonb,
    sync_status         VARCHAR(32) NOT NULL DEFAULT 'PENDING', -- PENDING, APPLIED, FAILED
    applied_at          TIMESTAMPTZ,
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statiot.firmware_releases (
    id                  VARCHAR(64) PRIMARY KEY,
    device_type         VARCHAR(64) NOT NULL,
    version             VARCHAR(64) NOT NULL,
    binary_url          TEXT NOT NULL,
    checksum_sha256     VARCHAR(64) NOT NULL,
    changelog           TEXT,
    is_critical         BOOLEAN DEFAULT FALSE,
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    released_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ──────────────────────────────────────────────────────────────────────
-- 5. IoT ALERTS & THRESHOLD BREACHES (P27)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.iot_alerts (
    id                  VARCHAR(64) PRIMARY KEY,
    device_id           VARCHAR(64) NOT NULL,
    sensor_code         VARCHAR(64),
    alert_type          VARCHAR(64) NOT NULL, -- THRESHOLD_BREACH, OFFLINE_TIMEOUT, ANOMALY, HARDWARE_FAULT
    severity            VARCHAR(32) NOT NULL DEFAULT 'WARNING', -- INFO, WARNING, CRITICAL, EMERGENCY
    message             TEXT NOT NULL,
    value               DOUBLE PRECISION,
    threshold           DOUBLE PRECISION,
    status              VARCHAR(32) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, ACKNOWLEDGED, RESOLVED
    acknowledged_by     VARCHAR(64),
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at         TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_iot_alerts_device ON statiot.iot_alerts(device_id);
CREATE INDEX IF NOT EXISTS idx_iot_alerts_status ON statiot.iot_alerts(status);

-- ──────────────────────────────────────────────────────────────────────
-- 6. MOBILE FIELD WORKFORCE & DEVICES (P43)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.field_workers (
    id                  VARCHAR(64) PRIMARY KEY,
    user_id             VARCHAR(64) NOT NULL,
    full_name           VARCHAR(255) NOT NULL,
    phone_number        VARCHAR(64),
    team_name           VARCHAR(128),
    role                VARCHAR(64) NOT NULL DEFAULT 'ENUMERATOR', -- ENUMERATOR, SUPERVISOR, INSPECTOR, TECHNICIAN
    assigned_district   VARCHAR(128),
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    org_id              VARCHAR(64),
    is_active           BOOLEAN DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statiot.mobile_devices (
    id                  VARCHAR(64) PRIMARY KEY,
    device_uuid         VARCHAR(128) UNIQUE NOT NULL,
    assigned_worker_id  VARCHAR(64) REFERENCES statiot.field_workers(id) ON DELETE SET NULL,
    platform            VARCHAR(32) NOT NULL, -- ANDROID, IOS, HARMONYOS, PWA
    app_version         VARCHAR(64) NOT NULL,
    os_version          VARCHAR(64),
    battery_level       DOUBLE PRECISION,
    storage_free_mb     BIGINT,
    last_sync_at        TIMESTAMPTZ,
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    status              VARCHAR(32) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, LOCKED, WIPED
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ──────────────────────────────────────────────────────────────────────
-- 7. FIELD VISITS & TASKS (P43)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.field_visits (
    id                  VARCHAR(64) PRIMARY KEY,
    worker_id           VARCHAR(64) NOT NULL REFERENCES statiot.field_workers(id),
    title               VARCHAR(255) NOT NULL,
    target_entity_type  VARCHAR(64) NOT NULL, -- FACILITY, FARM, HOUSEHOLD, SENSOR_STATION, SCHOOL
    target_entity_id    VARCHAR(64) NOT NULL,
    scheduled_start     TIMESTAMPTZ NOT NULL,
    scheduled_end       TIMESTAMPTZ NOT NULL,
    actual_start        TIMESTAMPTZ,
    actual_end          TIMESTAMPTZ,
    status              VARCHAR(32) NOT NULL DEFAULT 'SCHEDULED', -- SCHEDULED, IN_PROGRESS, COMPLETED, CANCELLED
    latitude            DOUBLE PRECISION,
    longitude           DOUBLE PRECISION,
    notes               TEXT,
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_field_visits_worker ON statiot.field_visits(worker_id);
CREATE INDEX IF NOT EXISTS idx_field_visits_status ON statiot.field_visits(status);

CREATE TABLE IF NOT EXISTS statiot.visit_tasks (
    id                  VARCHAR(64) PRIMARY KEY,
    visit_id            VARCHAR(64) NOT NULL REFERENCES statiot.field_visits(id) ON DELETE CASCADE,
    title               VARCHAR(255) NOT NULL,
    task_type           VARCHAR(64) NOT NULL, -- FORM_COLLECT, SENSOR_INSPECT, CALIBRATE, PHOTO_AUDIT
    form_id             VARCHAR(64),
    status              VARCHAR(32) NOT NULL DEFAULT 'PENDING', -- PENDING, COMPLETED, SKIPPED
    result_summary      JSONB DEFAULT '{}'::jsonb,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ──────────────────────────────────────────────────────────────────────
-- 8. MOBILE FORM DEFINITIONS & OFFLINE SUBMISSIONS (P43)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.mobile_form_definitions (
    id                  VARCHAR(64) PRIMARY KEY,
    form_code           VARCHAR(64) UNIQUE NOT NULL,
    title               VARCHAR(255) NOT NULL,
    version             INTEGER NOT NULL DEFAULT 1,
    schema_definition   JSONB NOT NULL,
    is_active           BOOLEAN DEFAULT TRUE,
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS statiot.mobile_submissions (
    id                  VARCHAR(64) PRIMARY KEY,
    client_submission_id VARCHAR(128) UNIQUE NOT NULL, -- Idempotency key from mobile client
    form_id             VARCHAR(64) NOT NULL REFERENCES statiot.mobile_form_definitions(id),
    worker_id           VARCHAR(64) NOT NULL REFERENCES statiot.field_workers(id),
    visit_id            VARCHAR(64) REFERENCES statiot.field_visits(id) ON DELETE SET NULL,
    data_payload        JSONB NOT NULL,
    geo_point           JSONB, -- {"lat": ..., "lng": ..., "accuracy": ...}
    collected_at        TIMESTAMPTZ NOT NULL,
    synced_at           TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sync_status         VARCHAR(32) NOT NULL DEFAULT 'COMMITTED', -- COMMITTED, CONFLICT_DETECTED, MERGED, REJECTED
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    version             INTEGER NOT NULL DEFAULT 1,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_submissions_form ON statiot.mobile_submissions(form_id);
CREATE INDEX IF NOT EXISTS idx_submissions_worker ON statiot.mobile_submissions(worker_id);

-- ──────────────────────────────────────────────────────────────────────
-- 9. GPS SPATIAL BREADCRUMBS & GEOFENCING (P43 - TimescaleDB compatible)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.gps_breadcrumbs (
    id                  VARCHAR(64) NOT NULL,
    worker_id           VARCHAR(64) NOT NULL,
    device_uuid         VARCHAR(128) NOT NULL,
    latitude            DOUBLE PRECISION NOT NULL,
    longitude           DOUBLE PRECISION NOT NULL,
    altitude            DOUBLE PRECISION,
    accuracy_meters     DOUBLE PRECISION,
    speed_mps           DOUBLE PRECISION,
    heading_deg         DOUBLE PRECISION,
    battery_level       DOUBLE PRECISION,
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    recorded_at         TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, recorded_at)
);

CREATE INDEX IF NOT EXISTS idx_gps_worker_time ON statiot.gps_breadcrumbs(worker_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_gps_tenant_time ON statiot.gps_breadcrumbs(tenant_id, recorded_at DESC);

CREATE TABLE IF NOT EXISTS statiot.geofence_zones (
    id                  VARCHAR(64) PRIMARY KEY,
    name                VARCHAR(255) NOT NULL,
    zone_type           VARCHAR(64) NOT NULL, -- ENUMERATION_AREA, QUARANTINE_ZONE, PROJECT_SITE
    center_lat          DOUBLE PRECISION NOT NULL,
    center_lng          DOUBLE PRECISION NOT NULL,
    radius_meters       DOUBLE PRECISION NOT NULL,
    polygon_geojson     JSONB,
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ──────────────────────────────────────────────────────────────────────
-- 10. OFFLINE SYNC TRANSACTIONS & BIDIRECTIONAL CONFLICT RECORDS (P43)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.sync_transactions (
    id                  VARCHAR(64) PRIMARY KEY,
    worker_id           VARCHAR(64) NOT NULL,
    device_uuid         VARCHAR(128) NOT NULL,
    sync_direction      VARCHAR(16) NOT NULL, -- PUSH, PULL, FULL
    records_synced      INTEGER NOT NULL DEFAULT 0,
    conflicts_detected  INTEGER NOT NULL DEFAULT 0,
    status              VARCHAR(32) NOT NULL DEFAULT 'SUCCESS', -- SUCCESS, PARTIAL, FAILED
    error_message       TEXT,
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    started_at          TIMESTAMPTZ NOT NULL,
    completed_at        TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS statiot.conflict_records (
    id                  VARCHAR(64) PRIMARY KEY,
    entity_type         VARCHAR(64) NOT NULL, -- SUBMISSION, VISIT, TASK, CONFIG
    entity_id           VARCHAR(64) NOT NULL,
    client_version      INTEGER NOT NULL,
    server_version      INTEGER NOT NULL,
    client_payload      JSONB NOT NULL,
    server_payload      JSONB NOT NULL,
    resolution_strategy VARCHAR(64) NOT NULL DEFAULT 'LAST_WRITE_WINS', -- LAST_WRITE_WINS, FIELD_LEVEL_MERGE, MANUAL_SUPERVISOR
    resolved_payload    JSONB,
    status              VARCHAR(32) NOT NULL DEFAULT 'RESOLVED_AUTO', -- PENDING_MANUAL, RESOLVED_AUTO, RESOLVED_MANUAL
    resolved_by         VARCHAR(64),
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at         TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_conflicts_status ON statiot.conflict_records(status);
CREATE INDEX IF NOT EXISTS idx_conflicts_entity ON statiot.conflict_records(entity_type, entity_id);

-- ──────────────────────────────────────────────────────────────────────
-- 11. OBJECT LINKS SEED / SUPPORT (Mandatory Hook)
-- ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS statiot.object_links (
    id                  SERIAL PRIMARY KEY,
    source_type         VARCHAR(64) NOT NULL,
    source_id           VARCHAR(255) NOT NULL,
    target_type         VARCHAR(64) NOT NULL,
    target_id           VARCHAR(255) NOT NULL,
    relationship        VARCHAR(64) NOT NULL DEFAULT 'monitored_by',
    tenant_id           VARCHAR(64) NOT NULL DEFAULT 'default',
    created_by          VARCHAR(128),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source_type, source_id, target_type, target_id, relationship)
);

CREATE INDEX IF NOT EXISTS idx_statiot_object_links_src ON statiot.object_links(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_statiot_object_links_tgt ON statiot.object_links(target_type, target_id);
