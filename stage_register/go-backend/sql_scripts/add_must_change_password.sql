-- Add must_change_password for district users on first login (default password)
-- Run once: psql -d your_db -f sql_scripts/add_must_change_password.sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN DEFAULT false;
