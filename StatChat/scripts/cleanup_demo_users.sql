-- StatChat demo-user cleanup (one-shot migration).
-- Removes the 100 legacy demo users (user-001..user-100) and their associated
-- profile/notification/presence/connection rows, and drops them from every
-- conversation's member roster. Runs atomically.
BEGIN;

-- 1. Child rows in FK-referencing tables for the demo users.
DELETE FROM notifications              WHERE user_id LIKE 'user-%';
DELETE FROM user_settings              WHERE user_id LIKE 'user-%';
DELETE FROM user_presence              WHERE user_id LIKE 'user-%';
DELETE FROM connections                WHERE user_id LIKE 'user-%' OR connected_to_id LIKE 'user-%';
DELETE FROM favourite_conversations    WHERE user_id LIKE 'user-%';
DELETE FROM conversation_mutes         WHERE user_id LIKE 'user-%';

-- 2. Strip demo users from every conversation's member_ids (keep real staff).
UPDATE conversations
SET member_ids = COALESCE((
    SELECT jsonb_agg(value ORDER BY ord)
    FROM (
        SELECT value, ord
        FROM jsonb_array_elements_text(member_ids) WITH ORDINALITY AS t(value, ord)
    ) t
    WHERE value NOT LIKE 'user-%'
), '[]'::jsonb)
WHERE member_ids::text LIKE '%user-%';

-- 3. Delete the demo users themselves.
DELETE FROM users WHERE id LIKE 'user-%';

COMMIT;