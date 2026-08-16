-- StatData (App 12 — P37 Data Engineering, P38 Scientific Computing,
-- P47 Enterprise Search) database and role.
-- Password injected from environment (STATDATA_DB_PASSWORD), never hardcoded.
CREATE DATABASE statdata;

\getenv statdata_pw STATDATA_DB_PASSWORD
SELECT format('CREATE ROLE "StatDataEngine" WITH LOGIN PASSWORD %L', :'statdata_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'StatDataEngine') \gexec

GRANT ALL PRIVILEGES ON DATABASE statdata TO "StatDataEngine";

\c statdata

GRANT ALL ON SCHEMA public TO "StatDataEngine";
CREATE SCHEMA IF NOT EXISTS statdata AUTHORIZATION "StatDataEngine";
ALTER DEFAULT PRIVILEGES IN SCHEMA statdata GRANT ALL ON TABLES TO "StatDataEngine";
ALTER DEFAULT PRIVILEGES IN SCHEMA statdata GRANT ALL ON SEQUENCES TO "StatDataEngine";
ALTER ROLE "StatDataEngine" SET search_path TO statdata, public;