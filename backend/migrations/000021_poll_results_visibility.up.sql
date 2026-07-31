-- =============================================================================
-- POLL — ikki rejim + «E'lon qilish» (№7, PRODUCT.md asoschi qarori)
--
-- Mentor so'rovnoma YARATISHDA natijaning taqdirini tanlaydi:
--   'mentor_only' (default) — natijani FAQAT mentor ko'radi, hech qachon
--                             o'quvchilarga chiqmaydi (e'lon qilib ham bo'lmaydi);
--   'public'                 — natijani o'quvchilar ham ko'rishi MUMKIN, lekin
--                             faqat mentor «E'lon qilish» tugmasini bosgach.
--
-- Ya'ni ikkita mustaqil shart: REJIM (yaratishda, o'zgarmas) va E'LON (dars
-- davomida, bir tomonlama). Ikkalasi ham bajarilmaguncha o'quvchi natijani
-- ololmaydi. Default ataylab yopiq tomonda: eski (migratsiyagacha yaratilgan)
-- so'rovnomalar ham to'satdan ochilib ketmasin.
-- =============================================================================
ALTER TABLE polls
    ADD COLUMN results_visibility VARCHAR(16) NOT NULL DEFAULT 'mentor_only';

ALTER TABLE polls
    ADD COLUMN results_published_at TIMESTAMPTZ;

-- Qiymatlar to'plami DB darajasida qulflanadi: kod xato qiymat yozib qo'ysa
-- (masalan bo'sh satr) natijani ko'rish mantig'i jimgina "hech kimga" ga
-- aylanardi va sababi ko'rinmasdi.
ALTER TABLE polls
    ADD CONSTRAINT chk_polls_results_visibility
    CHECK (results_visibility IN ('mentor_only', 'public'));

COMMENT ON COLUMN polls.results_visibility IS
    'mentor_only = natija hech qachon o''quvchiga ko''rinmaydi; public = e''lon qilingach ko''rinadi';
COMMENT ON COLUMN polls.results_published_at IS
    'Mentor «E''lon qilish» bosgan vaqt (NULL = hali e''lon qilinmagan)';
