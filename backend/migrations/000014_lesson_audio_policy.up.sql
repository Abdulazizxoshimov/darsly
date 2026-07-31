-- =============================================================================
-- ZOOM MODELI — ovoz nazorati sozlamalari (№11)
--
-- mute_on_entry:      yangi ishtirokchi kirganda mikrofoni server tomonda
--                     o'chirilgan holda boshlanadi (Zoom "mute on entry").
-- allow_self_unmute:  o'quvchi o'z mikrofonini o'zi yoqa oladimi. FALSE bo'lsa
--                     server har audio publish'ni qayta mute qiladi (Zoom
--                     "allow participants to unmute themselves" checkbox'i).
--
-- Ikkalasi ham DEFAULT TRUE — Zoom default'lari bilan bir xil.
-- =============================================================================
ALTER TABLE lessons
    ADD COLUMN mute_on_entry     BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN allow_self_unmute BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN lessons.mute_on_entry     IS 'Kirganda mikrofon o''chiq boshlanadi (Zoom mute-on-entry)';
COMMENT ON COLUMN lessons.allow_self_unmute IS 'O''quvchi o''zini unmute qila oladimi (FALSE → server qayta mute qiladi)';
