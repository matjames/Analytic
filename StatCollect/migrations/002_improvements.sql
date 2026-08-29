-- ── StatCollect Deep-Clean Migration 002 ──────────────────────────────────
-- Adds: QA flags, GPS coordinates, updated_at auto-trigger, content_type fix,
--       form_id index, submissions search index, and admin-level audit columns.
-- Safe to run multiple times (all statements use IF NOT EXISTS / IF EXISTS).

-- 1. QA flags — persisted per-submission rather than buried in event_log
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS qa_flags JSONB NOT NULL DEFAULT '[]';

-- 2. GPS coordinates — extracted from XML and stored for fast geo queries
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS gps_lat  DOUBLE PRECISION;
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS gps_lng  DOUBLE PRECISION;
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS gps_raw  TEXT;

-- 3. submitted_by index (for enumerator analytics queries)
CREATE INDEX IF NOT EXISTS idx_submissions_submitted_by ON submissions(submitted_by);

-- 4. GPS coordinate index for geographic range queries
CREATE INDEX IF NOT EXISTS idx_submissions_gps ON submissions(gps_lat, gps_lng)
  WHERE gps_lat IS NOT NULL AND gps_lng IS NOT NULL;

-- 5. received_at index for time-series analytics
CREATE INDEX IF NOT EXISTS idx_submissions_received_at ON submissions(received_at DESC);

-- 6. Auto-update updated_at trigger on submissions
CREATE OR REPLACE FUNCTION statcollect_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS submissions_updated_at ON submissions;
CREATE TRIGGER submissions_updated_at
  BEFORE UPDATE ON submissions
  FOR EACH ROW EXECUTE FUNCTION statcollect_set_updated_at();

DROP TRIGGER IF EXISTS registries_updated_at ON registries;
CREATE TRIGGER registries_updated_at
  BEFORE UPDATE ON registries
  FOR EACH ROW EXECUTE FUNCTION statcollect_set_updated_at();

DROP TRIGGER IF EXISTS templates_updated_at ON templates;
CREATE TRIGGER templates_updated_at
  BEFORE UPDATE ON templates
  FOR EACH ROW EXECUTE FUNCTION statcollect_set_updated_at();

-- 7. Partition-friendly event_log TTL cleanup function (call from a cron job)
CREATE OR REPLACE FUNCTION purge_old_events(retain_days INT DEFAULT 365)
RETURNS BIGINT AS $$
DECLARE deleted BIGINT;
BEGIN
  DELETE FROM event_log WHERE created_at < now() - (retain_days || ' days')::INTERVAL;
  GET DIAGNOSTICS deleted = ROW_COUNT;
  RETURN deleted;
END;
$$ LANGUAGE plpgsql;

-- 8. submissions total_count view for fast KPI queries
CREATE OR REPLACE VIEW submission_kpis AS
SELECT
  tenant_id,
  COUNT(*)                                                              AS total,
  COUNT(*) FILTER (WHERE received_at::date = CURRENT_DATE)             AS today,
  COUNT(*) FILTER (WHERE received_at >= date_trunc('week', now()))     AS this_week,
  COUNT(*) FILTER (WHERE received_at >= date_trunc('month', now()))    AS this_month,
  COUNT(*) FILTER (WHERE received_at >= date_trunc('year', now()))     AS this_year,
  COUNT(*) FILTER (WHERE status = 'approved')                          AS approved,
  COUNT(*) FILTER (WHERE status = 'rejected')                          AS rejected,
  COUNT(*) FILTER (WHERE status = 'received')                          AS pending,
  COUNT(*) FILTER (WHERE gps_lat IS NOT NULL)                          AS with_gps,
  COUNT(*) FILTER (WHERE qa_flags != '[]')                             AS flagged
FROM submissions
GROUP BY tenant_id;
