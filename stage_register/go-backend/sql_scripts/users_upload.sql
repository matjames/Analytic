UPDATE public.users_upload u
SET district_id = a.mfl_uid
FROM public.admin_units a
WHERE a.level_id = 3
  AND a.name = u.username || ' District';

SELECT
    u.username,
    a.name AS district_name,
    a.mfl_uid
FROM public.users_upload u
JOIN public.admin_units a
  ON a.name = u.username || ' District'
WHERE a.level_id = 3;

SELECT
  firstname,
  lastname,
  LOWER(REPLACE(username, ' ', '')) || '.biostat' AS username,
  email,
  "role",
  district_id
FROM public.users_upload;


ALTER TABLE users ADD COLUMN first_name TEXT; 
ALTER TABLE users ADD COLUMN last_name TEXT;