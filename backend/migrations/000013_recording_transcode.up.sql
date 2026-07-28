-- =============================================================================
-- YOZUVNI QAYTA KODLASH (CRF) — Zoom darajasidagi fayl hajmi uchun.
--
-- Egress faylni JONLI yozadi, ya'ni belgilangan bitrate bilan: statik slaydga
-- ham, harakatli videoga ham bir xil oqim ketadi. Zoom esa uchrashuvdan KEYIN
-- qayta kodlaydi va CRF (sifatga bog'langan) rejimida statik kadrlar deyarli
-- bepul bo'ladi — 2.5 soatlik dars 300 MB atrofida chiqadi.
--
-- Shu bosqichni qo'shamiz: `egress_ended` dan keyin yozuv navbatga tushadi,
-- fon ishchisi uni ffmpeg bilan qayta kodlaydi va MinIO'dagi faylni almashtiradi.
--
-- Holatlar: pending (navbatda) · running (ishlanmoqda) · done · failed · skipped.
-- Eski qatorlar 'skipped' — ular allaqachon yozilgan, qayta kodlash ularni
-- yaxshilamaydi (faqat sifatni yo'qotardi).
-- =============================================================================
ALTER TABLE recordings
    ADD COLUMN IF NOT EXISTS transcode_status    VARCHAR(16) NOT NULL DEFAULT 'skipped',
    ADD COLUMN IF NOT EXISTS original_size_bytes BIGINT,
    ADD COLUMN IF NOT EXISTS transcode_started_at TIMESTAMPTZ;

-- Navbatni tanlash uchun tor indeks: jadval o'sib borsa ham `pending` qidiruvi
-- doimiy tez qoladi (aksariyat qatorlar 'done'/'skipped' bo'ladi).
CREATE INDEX IF NOT EXISTS idx_recordings_transcode_queue
    ON recordings (transcode_status, created_at)
    WHERE transcode_status IN ('pending', 'running');

COMMENT ON COLUMN recordings.transcode_status IS 'pending|running|done|failed|skipped';
COMMENT ON COLUMN recordings.original_size_bytes IS 'Qayta kodlashdan OLDINGI hajm (taqqoslash uchun)';
