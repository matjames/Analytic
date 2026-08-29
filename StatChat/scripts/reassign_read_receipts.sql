-- Reassign leftover read receipts from demo user-001 to real 355 (John Matovu).
BEGIN;
UPDATE read_receipts SET user_id = '355' WHERE user_id = 'user-001';
COMMIT;