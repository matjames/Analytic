-- StatCollect database and role.
-- Phase 11/43 convergence: field data collection joins the shared postgres.
-- Password injected from environment (STATCOLLECT_DB_PASSWORD), fail-closed
-- via docker-compose, never hardcoded.
SELECT 'CREATE DATABASE statcollect' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'statcollect') \gexec

\getenv statcollect_pw STATCOLLECT_DB_PASSWORD
SELECT format('CREATE ROLE "StatCollect" WITH LOGIN PASSWORD %L', :'statcollect_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'StatCollect') \gexec

GRANT ALL PRIVILEGES ON DATABASE statcollect TO "StatCollect";

\c statcollect

-- Schema-level permissions
GRANT ALL ON SCHEMA public TO "StatCollect";
CREATE SCHEMA IF NOT EXISTS statcollect AUTHORIZATION "StatCollect";
ALTER DEFAULT PRIVILEGES IN SCHEMA statcollect GRANT ALL ON TABLES TO "StatCollect";
ALTER DEFAULT PRIVILEGES IN SCHEMA statcollect GRANT ALL ON SEQUENCES TO "StatCollect";
ALTER ROLE "StatCollect" SET search_path TO statcollect, public;
