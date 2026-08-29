-- Reassign leftover demo content authored by demo user-001 ("Matovu j")
-- to the equivalent real Registry user: 355 "John Matovu" (matovu.j).
BEGIN;

-- 1. Post likes from demo user-001 -> real 355.
UPDATE post_likes SET user_id = '355' WHERE user_id = 'user-001';

-- 2. Knowledge post authored/created by demo user-001 -> real 355.
UPDATE knowledge_posts
SET author = 'John Matovu', created_by = '355'
WHERE created_by = 'user-001' OR author = 'user-001' OR author = 'Matovu j';

-- 3. Call sessions hosted by demo user-001 -> real 355.
UPDATE call_sessions
SET host_id = '355', host_name = 'John Matovu'
WHERE host_id = 'user-001';

-- 4. Call participants who were demo user-001 -> real 355.
UPDATE call_participants
SET user_id = '355', user_name = 'John Matovu'
WHERE user_id = 'user-001';

COMMIT;