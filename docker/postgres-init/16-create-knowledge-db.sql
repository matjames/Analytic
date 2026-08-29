-- Knowledge Portal (App 2 — Public Knowledge & Open Data) database and role.
-- P17 (Public Portals / CMS / Dissemination), P40 (National Digital Library),
-- P41 (Open Data / Public Evidence Portal).
-- Password injected from environment (KNOWLEDGE_DB_PASSWORD), never hardcoded.
SELECT 'CREATE DATABASE knowledge_portal' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'knowledge_portal') \gexec

\getenv knowledge_pw KNOWLEDGE_DB_PASSWORD
SELECT format('CREATE ROLE "KnowledgePortal" WITH LOGIN PASSWORD %L', :'knowledge_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'KnowledgePortal') \gexec

GRANT ALL PRIVILEGES ON DATABASE knowledge_portal TO "KnowledgePortal";

\c knowledge_portal

GRANT ALL ON SCHEMA public TO "KnowledgePortal";
CREATE SCHEMA IF NOT EXISTS knowledge AUTHORIZATION "KnowledgePortal";
ALTER DEFAULT PRIVILEGES IN SCHEMA knowledge GRANT ALL ON TABLES TO "KnowledgePortal";
ALTER DEFAULT PRIVILEGES IN SCHEMA knowledge GRANT ALL ON SEQUENCES TO "KnowledgePortal";
ALTER ROLE "KnowledgePortal" SET search_path TO knowledge, public;
