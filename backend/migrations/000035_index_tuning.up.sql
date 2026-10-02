-- Indeks tuzatishlari (DB audit): o'lik indekslar, ListByMentor kompozit, retention mosligi.

-- O'lik: predikat/planner ishlatmaydi (past selektivlik yoki so'rov yo'q).
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_wrr_created_at;
DROP INDEX IF EXISTS idx_chat_messages_dm;

-- ListByMentor: WHERE mentor_id=? AND deleted_at IS NULL ORDER BY created_at DESC.
-- Kompozit avvalgi (mentor_id) indeksini to'liq qoplaydi (prefiks).
CREATE INDEX IF NOT EXISTS idx_lessons_mentor_created ON lessons (mentor_id, created_at DESC) WHERE deleted_at IS NULL;
DROP INDEX IF EXISTS idx_lessons_mentor_id;

-- Retention so'rovi COALESCE(ended_at, created_at) ishlatadi — indeks ifodasi mos bo'lishi shart.
CREATE INDEX IF NOT EXISTS idx_recordings_retention_coalesce
    ON recordings ((COALESCE(ended_at, created_at))) WHERE status = 'ready';
DROP INDEX IF EXISTS idx_recordings_retention;
