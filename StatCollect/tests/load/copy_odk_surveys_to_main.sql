\set ON_ERROR_STOP on
\if :{?source_password}
\else
  \echo 'source_password psql variable is required'
  \quit
\endif
\if :{?tenant_id}
\else
  \set tenant_id default
\endif

CREATE TABLE IF NOT EXISTS public.templates (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  version TEXT NOT NULL DEFAULT '1.0',
  schema JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_by TEXT,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  is_shared BOOLEAN NOT NULL DEFAULT false,
  parent_id TEXT,
  category TEXT NOT NULL DEFAULT 'General',
  rating SMALLINT NOT NULL DEFAULT 0,
  download_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.templates ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'General';
ALTER TABLE public.templates ADD COLUMN IF NOT EXISTS rating SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE public.templates ADD COLUMN IF NOT EXISTS download_count INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS public.submissions (
  id BIGSERIAL PRIMARY KEY,
  instance_id TEXT UNIQUE NOT NULL,
  form_id TEXT,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  meta JSONB,
  xml TEXT,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  submitted_by TEXT,
  status TEXT NOT NULL DEFAULT 'received',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_templates_tenant ON public.templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_templates_status ON public.templates(status);
CREATE INDEX IF NOT EXISTS idx_submissions_tenant ON public.submissions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_submissions_form ON public.submissions(form_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON public.submissions(status);

CREATE EXTENSION IF NOT EXISTS postgres_fdw;
DROP SERVER IF EXISTS synthetic_statcollect_source CASCADE;
CREATE SERVER synthetic_statcollect_source FOREIGN DATA WRAPPER postgres_fdw
  OPTIONS (host 'statcollect-postgres-1', dbname 'statcollect', port '5432');
CREATE USER MAPPING FOR CURRENT_USER SERVER synthetic_statcollect_source
  OPTIONS (user 'statcollect', password :'source_password');
DROP SCHEMA IF EXISTS synthetic_statcollect_transfer CASCADE;
CREATE SCHEMA synthetic_statcollect_transfer;
IMPORT FOREIGN SCHEMA public LIMIT TO (templates, submissions)
  FROM SERVER synthetic_statcollect_source INTO synthetic_statcollect_transfer;

BEGIN;
DELETE FROM public.submissions WHERE tenant_id = :'tenant_id';
DELETE FROM public.templates WHERE tenant_id = :'tenant_id';

INSERT INTO public.templates
  (id, name, description, version, schema, status, created_by, tenant_id,
   is_shared, parent_id, created_at, updated_at)
SELECT id, name, description, version, schema, status, created_by, tenant_id,
       is_shared, parent_id, created_at, updated_at
FROM synthetic_statcollect_transfer.templates
WHERE tenant_id = :'tenant_id'
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, description=EXCLUDED.description, version=EXCLUDED.version,
  schema=EXCLUDED.schema, status=EXCLUDED.status, created_by=EXCLUDED.created_by,
  tenant_id=EXCLUDED.tenant_id, is_shared=EXCLUDED.is_shared,
  parent_id=EXCLUDED.parent_id, updated_at=EXCLUDED.updated_at;

INSERT INTO public.submissions
  (instance_id, form_id, received_at, meta, xml, tenant_id, submitted_by,
   status, created_at, updated_at)
SELECT instance_id, form_id, received_at, meta, xml, tenant_id, submitted_by,
       status, created_at, updated_at
FROM synthetic_statcollect_transfer.submissions
WHERE tenant_id = :'tenant_id';
COMMIT;

ANALYZE public.templates;
ANALYZE public.submissions;

DROP SCHEMA synthetic_statcollect_transfer CASCADE;
DROP SERVER synthetic_statcollect_source CASCADE;

SELECT form_id, count(*) AS submissions
FROM public.submissions WHERE tenant_id = :'tenant_id'
GROUP BY form_id ORDER BY form_id;
SELECT count(*) AS templates,
       min(jsonb_array_length(schema->'fields')) AS min_fields,
       max(jsonb_array_length(schema->'fields')) AS max_fields
FROM public.templates WHERE tenant_id = :'tenant_id';
