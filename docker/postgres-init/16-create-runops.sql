-- App 4: Platform Engineering / RunOps.  Dedicated schema in the platform database.
\connect statgate_enterprise;
CREATE SCHEMA IF NOT EXISTS runops;
GRANT USAGE, CREATE ON SCHEMA runops TO PUBLIC;
