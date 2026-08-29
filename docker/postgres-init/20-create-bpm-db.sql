-- Business Process Management (App 11 — P48 BPM, Case Management, Process
-- Mining & Automation) database and role.
-- Password injected from environment (BPM_DB_PASSWORD), never hardcoded.
SELECT 'CREATE DATABASE bpm_hub' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'bpm_hub') \gexec

\getenv bpm_pw BPM_DB_PASSWORD
SELECT format('CREATE ROLE "BPMEngine" WITH LOGIN PASSWORD %L', :'bpm_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'BPMEngine') \gexec

GRANT ALL PRIVILEGES ON DATABASE bpm_hub TO "BPMEngine";

\c bpm_hub

GRANT ALL ON SCHEMA public TO "BPMEngine";
CREATE SCHEMA IF NOT EXISTS bpm AUTHORIZATION "BPMEngine";
ALTER DEFAULT PRIVILEGES IN SCHEMA bpm GRANT ALL ON TABLES TO "BPMEngine";
ALTER DEFAULT PRIVILEGES IN SCHEMA bpm GRANT ALL ON SEQUENCES TO "BPMEngine";
ALTER ROLE "BPMEngine" SET search_path TO bpm, public;