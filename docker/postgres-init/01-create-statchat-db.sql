-- Create all required databases for the StatGate platform
CREATE DATABASE statchat;
CREATE DATABASE kaggle;
CREATE DATABASE statgate;

-- Create the statgate user if it doesn't exist, and grant privileges
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'statgate') THEN
    CREATE ROLE statgate WITH LOGIN PASSWORD 'Uganda2026';
  END IF;
END
$$;

-- Create the Statchat user if it doesn't exist
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'Statchat') THEN
    CREATE ROLE Statchat WITH LOGIN PASSWORD 'Statgate';
  END IF;
END
$$;

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
