-- StatSpatial database and role.
-- Phase y convergence: StatSpatial is brought into the platform and deployed as
-- a first-class StatGate service. Password injected from environment
-- (STATSPATIAL_DB_PASSWORD), fail-closed via docker-compose never hardcoded.
SELECT 'CREATE DATABASE statspatial' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'statspatial') \gexec

\getenv statspatial_pw STATSPATIAL_DB_PASSWORD
SELECT format('CREATE ROLE "StatSpatial" WITH LOGIN PASSWORD %L', :'statspatial_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'StatSpatial') \gexec

GRANT ALL PRIVILEGES ON DATABASE statspatial TO "StatSpatial";

\c statspatial

-- Schema-level permissions
GRANT ALL ON SCHEMA public TO "StatSpatial";
CREATE SCHEMA IF NOT EXISTS statspatial AUTHORIZATION "StatSpatial";
ALTER DEFAULT PRIVILEGES IN SCHEMA statspatial GRANT ALL ON TABLES TO "StatSpatial";
ALTER DEFAULT PRIVILEGES IN SCHEMA statspatial GRANT ALL ON SEQUENCES TO "StatSpatial";
ALTER ROLE "StatSpatial" SET search_path TO statspatial, public;