-- Remove demo conversations left empty after demo-user cleanup (FK-safe order).
BEGIN;

-- Target IDs once.
CREATE TEMP TABLE _empty_conv AS
SELECT id FROM conversations
WHERE jsonb_array_length(COALESCE(member_ids, '[]'::jsonb)) = 0;

-- Messages inside those conversations and their children.
DELETE FROM message_reactions
WHERE message_id IN (SELECT id FROM messages WHERE conversation_id IN (SELECT id FROM _empty_conv));

DELETE FROM message_attachments
WHERE message_id IN (SELECT id FROM messages WHERE conversation_id IN (SELECT id FROM _empty_conv));

DELETE FROM read_receipts
WHERE message_id IN (SELECT id FROM messages WHERE conversation_id IN (SELECT id FROM _empty_conv));

DELETE FROM messages
WHERE conversation_id IN (SELECT id FROM _empty_conv);

-- Direct conversation children.
DELETE FROM pinned_messages            WHERE conversation_id IN (SELECT id FROM _empty_conv);
DELETE FROM favourite_conversations    WHERE conversation_id IN (SELECT id FROM _empty_conv);
DELETE FROM conversation_mutes         WHERE conversation_id IN (SELECT id FROM _empty_conv);
DELETE FROM calendar_events            WHERE conversation_id IN (SELECT id FROM _empty_conv);

-- The conversations themselves.
DELETE FROM conversations WHERE id IN (SELECT id FROM _empty_conv);

DROP TABLE _empty_conv;
COMMIT;