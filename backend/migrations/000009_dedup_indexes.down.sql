-- Dublikat indeksni tiklash (rollback).
CREATE UNIQUE INDEX IF NOT EXISTS idx_lessons_join_slug ON lessons (join_slug);
