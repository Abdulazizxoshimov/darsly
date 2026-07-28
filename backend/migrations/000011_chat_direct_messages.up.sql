-- =============================================================================
-- CHAT — shaxsiy (direct) xabarlar
-- =============================================================================
-- Avval chat butunlay "hammaga" edi va faqat HOST xabari saqlanardi: o'quvchi
-- yozgani hech qayerda qolmasdi (tarixda ham, yozuvda ham), kech kirgan esa
-- hech nima ko'rmasdi. Endi barcha xabarlar saqlanadi, `to_identity` esa
-- yo'naltirishni beradi:
--
--   to_identity IS NULL  → xonaga (hammaga)
--   to_identity = '<id>' → faqat o'sha ishtirokchiga (va yuboruvchiga)
--
-- VARCHAR(128) — `sender_identity` bilan bir xil: LiveKit identity'si guest'da
-- ixtiyoriy satr bo'lishi mumkin, foydalanuvchi UUID'si emas.
ALTER TABLE chat_messages ADD COLUMN to_identity VARCHAR(128);

-- Shaxsiy xabarlar ro'yxati (suhbatdosh bo'yicha). QISMAN indeks: xabarlarning
-- aksariyati ommaviy bo'ladi, ular bu indeksda o'rin egallamasin.
CREATE INDEX idx_chat_messages_dm
    ON chat_messages (lesson_id, to_identity, created_at)
    WHERE to_identity IS NOT NULL;

COMMENT ON COLUMN chat_messages.to_identity IS
    'NULL = xonaga (ommaviy); aks holda — faqat shu identity va yuboruvchi ko''radi';
