-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";        -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "citext";          -- case-insensitive email
CREATE EXTENSION IF NOT EXISTS "pg_trgm";         -- trigram search (qidiruv uchun)
CREATE EXTENSION IF NOT EXISTS "btree_gin";       -- GIN indexes uchun

-- updated_at ni avtomatik yangilash uchun trigger funksiyasi
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
