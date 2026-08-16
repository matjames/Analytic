-- ══════════════════════════════════════════════════════════════
-- STATGATE APP 8: IoT, SENSORS & MOBILE FIELD OPS DATABASE BOOTSTRAP
-- Phases: P27 (IoT & Edge Computing), P43 (Mobile & Offline Field Ops)
-- ══════════════════════════════════════════════════════════════

CREATE DATABASE statiot;

\connect statiot

CREATE SCHEMA IF NOT EXISTS statiot;
CREATE SCHEMA IF NOT EXISTS public;

-- Enable UUID extension if available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Grant privileges
GRANT ALL PRIVILEGES ON SCHEMA statiot TO postgres;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA statiot TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA statiot TO postgres;

ALTER DEFAULT PRIVILEGES IN SCHEMA statiot GRANT ALL ON TABLES TO postgres;
ALTER DEFAULT PRIVILEGES IN SCHEMA statiot GRANT ALL ON SEQUENCES TO postgres;
