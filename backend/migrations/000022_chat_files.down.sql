ALTER TABLE chat_messages DROP CONSTRAINT IF EXISTS chk_chat_messages_body_or_file;
ALTER TABLE chat_messages DROP CONSTRAINT IF EXISTS chk_chat_messages_file_complete;

ALTER TABLE chat_messages DROP COLUMN IF EXISTS file_mime;
ALTER TABLE chat_messages DROP COLUMN IF EXISTS file_size;
ALTER TABLE chat_messages DROP COLUMN IF EXISTS file_name;
ALTER TABLE chat_messages DROP COLUMN IF EXISTS file_key;
