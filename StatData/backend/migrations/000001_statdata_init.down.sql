-- ==============================================================================
-- STATGATE APP 12: DATA ENGINEERING, SCIENCE & SEARCH
-- TEARDOWN / ROLLBACK SCRIPT
-- ==============================================================================

DROP TABLE IF EXISTS statdata.object_links CASCADE;
DROP TABLE IF EXISTS statdata.audit_logs CASCADE;
DROP TABLE IF EXISTS statdata.saved_searches CASCADE;
DROP TABLE IF EXISTS statdata.indexed_documents CASCADE;
DROP TABLE IF EXISTS statdata.search_indexes CASCADE;
DROP TABLE IF EXISTS statdata.compute_jobs CASCADE;
DROP TABLE IF EXISTS statdata.compute_nodes CASCADE;
DROP TABLE IF EXISTS statdata.model_versions CASCADE;
DROP TABLE IF EXISTS statdata.registered_models CASCADE;
DROP TABLE IF EXISTS statdata.experiment_runs CASCADE;
DROP TABLE IF EXISTS statdata.experiments CASCADE;
DROP TABLE IF EXISTS statdata.notebook_sessions CASCADE;
DROP TABLE IF EXISTS statdata.feature_records CASCADE;
DROP TABLE IF EXISTS statdata.feature_views CASCADE;
DROP TABLE IF EXISTS statdata.streaming_jobs CASCADE;
DROP TABLE IF EXISTS statdata.pipeline_runs CASCADE;
DROP TABLE IF EXISTS statdata.pipelines CASCADE;
DROP TABLE IF EXISTS statdata.lineage_edges CASCADE;
DROP TABLE IF EXISTS statdata.lineage_nodes CASCADE;
DROP TABLE IF EXISTS statdata.data_quality_reports CASCADE;
DROP TABLE IF EXISTS statdata.data_quality_rules CASCADE;
DROP TABLE IF EXISTS statdata.data_contracts CASCADE;
DROP TABLE IF EXISTS statdata.schema_registry CASCADE;
DROP TABLE IF EXISTS statdata.data_sources CASCADE;
DROP TABLE IF EXISTS statdata.datasets CASCADE;

DROP SCHEMA IF EXISTS statdata CASCADE;
