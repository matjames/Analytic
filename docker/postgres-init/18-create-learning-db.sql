-- Learning, Community & Commercial (App 7 — P24 LMS/CPD, P26 CRM,
-- P35 Stakeholder Stewardship) database and role.
-- Password injected from environment (LMS_DB_PASSWORD), never hardcoded.
SELECT 'CREATE DATABASE learning_crm' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'learning_crm') \gexec

\getenv lms_pw LMS_DB_PASSWORD
SELECT format('CREATE ROLE "LearningCRM" WITH LOGIN PASSWORD %L', :'lms_pw')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'LearningCRM') \gexec

GRANT ALL PRIVILEGES ON DATABASE learning_crm TO "LearningCRM";

\c learning_crm

GRANT ALL ON SCHEMA public TO "LearningCRM";
CREATE SCHEMA IF NOT EXISTS learning AUTHORIZATION "LearningCRM";
ALTER DEFAULT PRIVILEGES IN SCHEMA learning GRANT ALL ON TABLES TO "LearningCRM";
ALTER DEFAULT PRIVILEGES IN SCHEMA learning GRANT ALL ON SEQUENCES TO "LearningCRM";
ALTER ROLE "LearningCRM" SET search_path TO learning, public;