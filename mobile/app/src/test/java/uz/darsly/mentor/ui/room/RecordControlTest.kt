package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Lokal yozuv orkestratsiyasi (M-1: recording control · "don't lose recordings").
 *
 * Yozuv EKRANni yozadi — ekran ulashilmasa yozuv bo'sh (yaroqsiz) chiqadi.
 * To'xtatishda ID yoki fayl bo'lmasa yuklash ma'nosiz, yetim fayl esa
 * tozalanishi kerak.
 */
class RecordControlTest {

    // ── toggle ────────────────────────────────────────────────────────────────

    @Test fun toggleStopsWhenAlreadyRecording() {
        assertEquals(RecordControl.Action.STOP, RecordControl.toggle(recording = true, screenSharing = false))
        // Yozayotganda ekran holati ahamiyatsiz — baribir to'xtaydi.
        assertEquals(RecordControl.Action.STOP, RecordControl.toggle(recording = true, screenSharing = true))
    }

    @Test fun toggleStartsOnlyWhenScreenShared() {
        assertEquals(RecordControl.Action.START, RecordControl.toggle(recording = false, screenSharing = true))
    }

    @Test fun toggleHintsWhenNoScreenShare() {
        // BUG: ekran ulashilmasa yozuv bo'sh/qora chiqadi. Boshlash o'rniga hint.
        assertEquals(
            RecordControl.Action.HINT_NO_SCREEN,
            RecordControl.toggle(recording = false, screenSharing = false),
        )
    }

    // ── canUpload / deleteOrphan ──────────────────────────────────────────────

    @Test fun uploadNeedsBothIdAndFile() {
        assertTrue(RecordControl.canUpload("rec-1", hasFile = true))
        // BUG: ID null bo'lsa (local-start ishlamagan) `null` bilan yuklash urinishi.
        assertFalse(RecordControl.canUpload(null, hasFile = true))
        assertFalse(RecordControl.canUpload("rec-1", hasFile = false))
    }

    @Test fun orphanFileIsDeletedWhenNotUploadable() {
        // BUG: ID yo'q-u fayl bor bo'lsa (yarim yozuv) — fayl yetim qolib
        // diskni band qilardi. O'chiriladi.
        assertTrue(RecordControl.deleteOrphan(recordingId = null, hasFile = true))
        // Fayl umuman yo'q bo'lsa o'chirishga narsa yo'q.
        assertFalse(RecordControl.deleteOrphan(recordingId = null, hasFile = false))
        // Yuklanadigan holatda o'chirmaymiz.
        assertFalse(RecordControl.deleteOrphan(recordingId = "rec-1", hasFile = true))
    }

    // ── durationSec ───────────────────────────────────────────────────────────

    @Test fun durationIsAtLeastOneSecond() {
        // BUG: 0-sekundli yozuv bema'ni. Juda qisqa yozuv ham 1s deb yuboriladi.
        assertEquals(1, RecordControl.durationSec(startMs = 1_000, nowMs = 1_000))
        assertEquals(1, RecordControl.durationSec(startMs = 1_000, nowMs = 1_500)) // 0.5s → 1
        assertEquals(5, RecordControl.durationSec(startMs = 1_000, nowMs = 6_000)) // 5s
    }

    // ── onShareChanged (H5) ────────────────────────────────────────────────────

    @Test
    fun `trek ketsa yozuv segmenti yakunlanadi`() {
        // Tizim "Stop sharing" / qayta ulanish: recorder o'lik trekka qarab
        // qolmasin (video qotgan, REC yonib turgan).
        assertEquals(RecordControl.ShareAction.STOP, RecordControl.onShareChanged(sharing = false, wanted = true, recording = true))
        assertEquals(RecordControl.ShareAction.STOP, RecordControl.onShareChanged(sharing = false, wanted = false, recording = true))
    }

    @Test
    fun `trek qaytsa faqat xohlangan yozuv qayta boshlanadi`() {
        assertEquals(RecordControl.ShareAction.START, RecordControl.onShareChanged(sharing = true, wanted = true, recording = false))
        // Ustoz yozuvni o'zi to'xtatgan — qayta ulanish uni jimgina tiklamaydi (maxfiylik).
        assertEquals(RecordControl.ShareAction.NONE, RecordControl.onShareChanged(sharing = true, wanted = false, recording = false))
    }

    @Test
    fun `yozuv allaqachon ketayotgan bolsa hech nima`() {
        assertEquals(RecordControl.ShareAction.NONE, RecordControl.onShareChanged(sharing = true, wanted = true, recording = true))
        assertEquals(RecordControl.ShareAction.NONE, RecordControl.onShareChanged(sharing = false, wanted = true, recording = false))
    }
}
