-- =============================================================================
-- LESSON BANS — dars-darajali ban'ning DURABLE nusxasi (system-design audit R2)
--
-- Avval dars-darajali ban FAQAT Redis'da (`room:ban:<lessonID>:<identity>`, 6h
-- TTL) edi. Muammo: Redis compound SPOF — u qayta ishga tushsa (yoki FLUSHDB)
-- barcha ban'lar yo'qolar va chiqarilgan ("kick") ishtirokchi darsga QAYTA
-- KIRA olardi. Mentor darajasidagi doimiy blocklist (mentor_blocklist) PG'da
-- edi, lekin bitta darslik ban emas.
--
-- Endi ban PG'ga ham yoziladi (write-through), Redis esa tezkor kesh bo'lib
-- qoladi. Asosiy gate (`room.EnforceJoin` webhook'i va token berish) Redis
-- miss/uzilishida PG'dan durable o'qiydi — ya'ni Redis o'chsa ham kick amalda
-- qoladi. Ochiq amal guardlari (chat/poll/roomstate) Redis-only fail-open
-- bo'lib qoladi (ataylab — ular moderatsiya qulayligi, auth emas).
--
-- Identity: o'quvchida akkaunt yo'q, LiveKit identity har join'da yangi
-- ("guest_<uuid>"). Shuning uchun bu ban ham joriy token oynasi (30 min) +
-- qisqa qayta-ulanish uchun. Doimiy (ismli) himoya mentor_blocklist'da.
-- =============================================================================
CREATE TABLE IF NOT EXISTS lesson_bans (
    lesson_id    UUID         NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    identity     VARCHAR(128) NOT NULL,
    display_name VARCHAR(128) NOT NULL DEFAULT '', -- ma'lum bo'lsa (audit uchun)
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (lesson_id, identity)
);

COMMENT ON TABLE lesson_bans IS 'Dars-darajali ban durable nusxasi (Redis room:ban:* keshiga zaxira, audit R2)';
