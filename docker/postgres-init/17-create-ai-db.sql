-- AI & Autonomy (App 5 — P22 Digital Twins, P31 Multi-Agent Systems,
-- P39 Knowledge Graph) database and role.
-- Password injected from environment (AIENG_DB_PASSWORD), never hardcoded.
SELECT 'CREATE DATABASE ai_intelligence' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'ai_intelligence') \gexec

\getenv aieng_pw AIENG_DB_PASSWORD
SELECT format('CREATE ROLE "AIEngine" WITH LOGIN PASSWORD %L', :'aieng_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'AIEngine') \gexec

GRANT ALL PRIVILEGES ON DATABASE ai_intelligence TO "AIEngine";

\c ai_intelligence

GRANT ALL ON SCHEMA public TO "AIEngine";
CREATE SCHEMA IF NOT EXISTS ai AUTHORIZATION "AIEngine";
ALTER DEFAULT PRIVILEGES IN SCHEMA ai GRANT ALL ON TABLES TO "AIEngine";
ALTER DEFAULT PRIVILEGES IN SCHEMA ai GRANT ALL ON SEQUENCES TO "AIEngine";
ALTER ROLE "AIEngine" SET search_path TO ai, public;