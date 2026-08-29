-- StatCitizen database and role.
-- Sovereign citizen engagement, feedback, reporting & public evidence platform.
-- Password injected from environment (STATCITIZEN_DB_PASSWORD), fail-closed
-- via docker-compose, never hardcoded.
SELECT 'CREATE DATABASE statcitizen' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'statcitizen') \gexec

\getenv statcitizen_pw STATCITIZEN_DB_PASSWORD
SELECT format('CREATE ROLE "statcitizen" WITH LOGIN PASSWORD %L', :'statcitizen_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'statcitizen') \gexec

GRANT ALL PRIVILEGES ON DATABASE statcitizen TO "statcitizen";

\c statcitizen

-- Schema-level permissions
GRANT ALL ON SCHEMA public TO "statcitizen";
CREATE SCHEMA IF NOT EXISTS statcitizen AUTHORIZATION "statcitizen";
ALTER DEFAULT PRIVILEGES IN SCHEMA statcitizen GRANT ALL ON TABLES TO "statcitizen";
ALTER DEFAULT PRIVILEGES IN SCHEMA statcitizen GRANT ALL ON SEQUENCES TO "statcitizen";
ALTER ROLE "statcitizen" SET search_path TO statcitizen, public;
