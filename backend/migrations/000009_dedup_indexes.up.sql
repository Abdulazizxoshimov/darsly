-- join_slug ustunida UNIQUE constraint (lessons_join_slug_key) allaqachon indeks
-- yaratadi; qo'lda qo'shilgan idx_lessons_join_slug — ayni dublikat. Har INSERT/UPDATE
-- ikkala indeksni yangilaydi (ortiqcha yozuv-yuki). Dublikatni olib tashlaymiz.
DROP INDEX IF EXISTS idx_lessons_join_slug;
