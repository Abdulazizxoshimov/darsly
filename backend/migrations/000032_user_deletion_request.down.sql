DROP INDEX IF EXISTS idx_users_deletion_requested;
ALTER TABLE users DROP COLUMN IF EXISTS deletion_requested_at;
