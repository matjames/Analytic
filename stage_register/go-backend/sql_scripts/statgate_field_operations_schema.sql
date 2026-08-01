DROP VIEW IF EXISTS public.orgunits CASCADE;
DROP VIEW IF EXISTS public.mfl_details CASCADE;
DROP TABLE IF EXISTS public.facility_request_documents CASCADE;
DROP TABLE IF EXISTS public.facility_request_approvals CASCADE;
DROP TABLE IF EXISTS public.facility_requests CASCADE;
DROP TABLE IF EXISTS public.posts CASCADE;
DROP TABLE IF EXISTS public.facilities CASCADE;
DROP TABLE IF EXISTS public.admin_units CASCADE;
DROP TABLE IF EXISTS public.admin_level CASCADE;
DROP TABLE IF EXISTS public."level" CASCADE;
DROP TABLE IF EXISTS public.ownership CASCADE;
DROP TABLE IF EXISTS public.authority CASCADE;
DROP TABLE IF EXISTS public.users CASCADE;
DROP TABLE IF EXISTS public.public_documents CASCADE;
DROP TABLE IF EXISTS public.mflupload CASCADE;
DROP TABLE IF EXISTS public.orgunits_uploads CASCADE;

CREATE TABLE public.mflupload (
  id BIGINT, uid TEXT, name TEXT, shortname TEXT, longtitude TEXT, latitude TEXT,
  nhfrid TEXT, subcounty_uid TEXT, subcounty TEXT, district_uid TEXT, district TEXT,
  region TEXT, hflevel TEXT, ownership TEXT, authority TEXT, status TEXT, report TEXT
);
CREATE TABLE public.orgunits_uploads (
  facility TEXT, facility_uid TEXT, subcounty TEXT, subcounty_uid TEXT,
  municipality TEXT, municipality_uid TEXT, district TEXT, district_uid TEXT,
  region TEXT, region_uid TEXT
);
CREATE TABLE public.admin_level (
  id BIGSERIAL PRIMARY KEY, mfl_uid TEXT UNIQUE NOT NULL, name TEXT NOT NULL,
  level_number INTEGER UNIQUE NOT NULL, "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public.admin_units (
  id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL, mfl_uid TEXT UNIQUE NOT NULL, code TEXT,
  level_id BIGINT NOT NULL REFERENCES public.admin_level(id), parent_id BIGINT REFERENCES public.admin_units(id),
  path TEXT NOT NULL, "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public."level" (
  id BIGSERIAL PRIMARY KEY, mfl_uid TEXT UNIQUE NOT NULL, code TEXT UNIQUE NOT NULL, name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '', "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public.ownership (
  id BIGSERIAL PRIMARY KEY, mfl_uid TEXT UNIQUE NOT NULL, code TEXT UNIQUE NOT NULL, name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '', "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public.authority (
  id BIGSERIAL PRIMARY KEY, mfl_uid TEXT UNIQUE NOT NULL, code TEXT UNIQUE NOT NULL, name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '', "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public.facilities (
  id BIGSERIAL PRIMARY KEY, identifier TEXT UNIQUE, mfl_uid TEXT UNIQUE NOT NULL, name TEXT NOT NULL, short_name TEXT,
  historical_id TEXT, admin_unit_id BIGINT REFERENCES public.admin_units(id), level TEXT REFERENCES public."level"(mfl_uid),
  ownership TEXT REFERENCES public.ownership(mfl_uid), authority TEXT REFERENCES public.authority(mfl_uid),
  status TEXT, reporting BOOLEAN, licensed BOOLEAN, address TEXT, contact_personemail TEXT, contact_personmobile TEXT,
  contact_personname TEXT, contact_persontitle TEXT, longitude NUMERIC, latitude NUMERIC, opening_date DATE,
  closing_date DATE, bed_capacity INTEGER, services TEXT, user_id BIGINT, "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public.users (
  id BIGSERIAL PRIMARY KEY, first_name TEXT, last_name TEXT, username TEXT UNIQUE NOT NULL, email TEXT UNIQUE NOT NULL,
  role TEXT, password TEXT NOT NULL, organisation TEXT, phoneno TEXT, district_id TEXT, must_change_password BOOLEAN NOT NULL DEFAULT TRUE,
  "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public.public_documents (
  id BIGSERIAL PRIMARY KEY, title TEXT NOT NULL, description TEXT, category TEXT NOT NULL,
  filename TEXT NOT NULL, original_filename TEXT NOT NULL, file_path TEXT NOT NULL,
  file_size BIGINT, mime_type TEXT, uploaded_by BIGINT NOT NULL REFERENCES public.users(id),
  "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public.facility_requests (
  id BIGSERIAL PRIMARY KEY,
  request_type TEXT NOT NULL,
  facility_id BIGINT REFERENCES public.facilities(id),
  facility_data JSONB,
  initiated_by BIGINT NOT NULL REFERENCES public.users(id),
  current_status TEXT NOT NULL DEFAULT 'pending',
  current_stage TEXT NOT NULL DEFAULT 'district_approver',
  district_approver_id BIGINT REFERENCES public.users(id),
  moh_clinical_id BIGINT REFERENCES public.users(id),
  moh_publisher_id BIGINT REFERENCES public.users(id),
  rejection_reason TEXT,
  "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public.facility_request_documents (
  id BIGSERIAL PRIMARY KEY,
  request_id BIGINT NOT NULL REFERENCES public.facility_requests(id) ON DELETE CASCADE,
  filename TEXT NOT NULL, original_filename TEXT NOT NULL, file_path TEXT NOT NULL,
  file_size BIGINT, mime_type TEXT, doc_type TEXT,
  "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public.facility_request_approvals (
  id BIGSERIAL PRIMARY KEY,
  request_id BIGINT NOT NULL REFERENCES public.facility_requests(id) ON DELETE CASCADE,
  stage TEXT NOT NULL, action TEXT NOT NULL, approver_id BIGINT NOT NULL REFERENCES public.users(id),
  comments TEXT, "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE public.posts (
  id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  title TEXT NOT NULL, content TEXT NOT NULL,
  "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_admin_units_parent ON public.admin_units(parent_id);
CREATE INDEX idx_admin_units_level ON public.admin_units(level_id);
CREATE INDEX idx_facilities_admin_unit ON public.facilities(admin_unit_id);
CREATE INDEX idx_facility_requests_status ON public.facility_requests(current_status, current_stage);
CREATE INDEX idx_facility_requests_initiated_by ON public.facility_requests(initiated_by);
