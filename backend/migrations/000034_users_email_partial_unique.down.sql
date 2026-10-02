-- OGOHLANTIRISH: agar soft-delete'dan keyin email qayta ishlatilgan bo'lsa (dublikat
-- email bor), jadval-darajali UNIQUE ni qayta yaratish YIQILADI. Avval dublikatlarni tozalang.
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email) WHERE deleted_at IS NULL;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
DROP INDEX IF EXISTS users_email_active;
