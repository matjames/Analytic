-- Geospatial & Remote Sensing (App 9 — P44 GIS, Remote Sensing & Drone
-- Integration) database and role.
-- Password injected from environment (GIS_DB_PASSWORD), never hardcoded.
SELECT 'CREATE DATABASE gis_intelligence' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'gis_intelligence') \gexec

\getenv gis_pw GIS_DB_PASSWORD
SELECT format('CREATE ROLE "GISEngine" WITH LOGIN PASSWORD %L', :'gis_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'GISEngine') \gexec

GRANT ALL PRIVILEGES ON DATABASE gis_intelligence TO "GISEngine";

\c gis_intelligence

GRANT ALL ON SCHEMA public TO "GISEngine";
CREATE SCHEMA IF NOT EXISTS gis AUTHORIZATION "GISEngine";
ALTER DEFAULT PRIVILEGES IN SCHEMA gis GRANT ALL ON TABLES TO "GISEngine";
ALTER DEFAULT PRIVILEGES IN SCHEMA gis GRANT ALL ON SEQUENCES TO "GISEngine";
ALTER ROLE "GISEngine" SET search_path TO gis, public;