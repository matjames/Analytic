-- ══════════════════════════════════════════════════════════════
-- STATGATE APP 8: IoT, SENSORS & MOBILE FIELD OPS DATABASE BOOTSTRAP
-- Phases: P27 (IoT & Edge Computing), P43 (Mobile & Offline Field Ops)
-- ══════════════════════════════════════════════════════════════

SELECT 'CREATE DATABASE statiot'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'statiot') \gexec

\connect statiot

CREATE SCHEMA IF NOT EXISTS statiot;
CREATE SCHEMA IF NOT EXISTS public;

-- Enable UUID extension if available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Grant privileges
\getenv pg_admin POSTGRES_USER
GRANT ALL PRIVILEGES ON SCHEMA statiot TO :"pg_admin";
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA statiot TO :"pg_admin";
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA statiot TO :"pg_admin";

ALTER DEFAULT PRIVILEGES IN SCHEMA statiot GRANT ALL ON TABLES TO :"pg_admin";
ALTER DEFAULT PRIVILEGES IN SCHEMA statiot GRANT ALL ON SEQUENCES TO :"pg_admin";
