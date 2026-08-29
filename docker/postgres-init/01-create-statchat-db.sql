-- Create all required databases for the StatGate platform
SELECT 'CREATE DATABASE statchat' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'statchat') \gexec
SELECT 'CREATE DATABASE kaggle' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'kaggle') \gexec
SELECT 'CREATE DATABASE statgate' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'statgate') \gexec

-- Passwords are injected from the container environment (fail-closed via
-- docker-compose). Never hardcode credentials in init scripts.
\getenv statgate_pw HELPDESK_DB_PASSWORD
\getenv statchat_pw STATCHAT_DB_PASSWORD

-- Create the statgate user if it doesn't exist, and grant privileges
SELECT format('CREATE ROLE statgate WITH LOGIN PASSWORD %L', :'statgate_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'statgate') \gexec

-- Create the Statchat user if it doesn't exist
SELECT format('CREATE ROLE "Statchat" WITH LOGIN PASSWORD %L', :'statchat_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'Statchat') \gexec

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE statgate TO statgate;
GRANT ALL PRIVILEGES ON DATABASE kaggle TO statgate;
GRANT ALL PRIVILEGES ON DATABASE statchat TO "Statchat";

-- Grant schema-level permissions (must be run per-database)
\c statgate
GRANT ALL ON SCHEMA public TO statgate;
CREATE SCHEMA IF NOT EXISTS statgate AUTHORIZATION statgate;
ALTER DEFAULT PRIVILEGES IN SCHEMA statgate GRANT ALL ON TABLES TO statgate;
ALTER DEFAULT PRIVILEGES IN SCHEMA statgate GRANT ALL ON SEQUENCES TO statgate;
ALTER ROLE statgate SET search_path TO statgate, public;

\c kaggle
GRANT ALL ON SCHEMA public TO statgate;

\c statchat
GRANT ALL ON SCHEMA public TO "Statchat";
CREATE SCHEMA IF NOT EXISTS public;
