package uz.darsly.mentor.data.livekit

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * [LocalRecorder] konveyerining sof qarorlari (M-2).
 *
 * Bu yerdagi har bir test JIMGINA buzuq yozuvning oldini oladi: muxer noto'g'ri
 * paytda boshlanishi (fayl ochilmaydi), PTS birligi (A/V siljishi), kodek-config
 * buferi sample sifatida yozilishi (`moov`siz fayl), boshlash/to'xtatish
 * takrori (crash yoki egress fallback'ini noto'g'ri tanlash).
 */
class RecorderPipelineTest {

    // ── muxerReady ────────────────────────────────────────────────────────────

    @Test fun muxerWaitsForBothTracks() {
        // BUG: muxer bitta trek bilan boshlansa, ikkinchi addTrack istisno otadi
        // va BUTUN yozuv yo'qoladi — shuning uchun ikkalasi ham kelmaguncha kutadi.
        assertFalse(RecorderPipeline.muxerReady(videoTrackIndex = -1, audioTrackIndex = -1))
        assertFalse(RecorderPipeline.muxerReady(videoTrackIndex = 0, audioTrackIndex = -1))
        assertFalse(RecorderPipeline.muxerReady(videoTrackIndex = -1, audioTrackIndex = 1))
        assertTrue(RecorderPipeline.muxerReady(videoTrackIndex = 0, audioTrackIndex = 1))
    }

    // ── shouldDropOldest ──────────────────────────────────────────────────────

    @Test fun queueDropsOnlyAboveCap() {
        // BUG: navbat cheksiz o'ssa past telefonda xotira tugaydi. Cap = 4;
        // 4 tagacha saqlanadi, 5-dan boshlab eng eski tashlanadi.
        assertFalse(RecorderPipeline.shouldDropOldest(0))
        assertFalse(RecorderPipeline.shouldDropOldest(RecorderPipeline.MAX_PENDING_FRAMES)) // 4 — hali emas
        assertTrue(RecorderPipeline.shouldDropOldest(RecorderPipeline.MAX_PENDING_FRAMES + 1)) // 5 — tashlanadi
    }

    // ── ptsMicros ─────────────────────────────────────────────────────────────

    @Test fun ptsIsMicrosecondsFromStart() {
        // BUG: MediaMuxer PTS'ni MIKROSEKUNDDA kutadi. nanoTime'ni 1000 ga bo'lmaslik
        // (nano) yoki 1_000_000 ga bo'lish (milli) A/V ni sekundlar bilan siljitardi.
        val start = 1_000_000_000L // 1s nanoTime bazasi
        assertEquals(0L, RecorderPipeline.ptsMicros(start, start))
        assertEquals(1_000L, RecorderPipeline.ptsMicros(start + 1_000_000L, start)) // 1ms = 1000µs
        assertEquals(1_000_000L, RecorderPipeline.ptsMicros(start + 1_000_000_000L, start)) // 1s = 1e6 µs
    }

    // ── drainAction ───────────────────────────────────────────────────────────

    @Test fun codecConfigBufferIsNeverWritten() {
        // BUG: SPS/PPS konfiguratsiya buferini sample sifatida yozish faylni buzadi
        // (pleyer boshini o'qiy olmaydi). Muxer boshlangan bo'lsa ham YOZILMAYDI.
        assertEquals(
            RecorderPipeline.DrainAction.SKIP_CONFIG,
            RecorderPipeline.drainAction(isCodecConfig = true, size = 100, muxerStarted = true),
        )
    }

    @Test fun bufferNotWrittenBeforeMuxerStarted() {
        // BUG: muxer boshlanmasdan writeSampleData chaqirilsa istisno otiladi.
        assertEquals(
            RecorderPipeline.DrainAction.RELEASE_ONLY,
            RecorderPipeline.drainAction(isCodecConfig = false, size = 100, muxerStarted = false),
        )
    }

    @Test fun emptyBufferIsNotWritten() {
        // BUG: 0 o'lchamli buferni yozish keraksiz (yozadigan narsa yo'q).
        assertEquals(
            RecorderPipeline.DrainAction.RELEASE_ONLY,
            RecorderPipeline.drainAction(isCodecConfig = false, size = 0, muxerStarted = true),
        )
    }

    @Test fun realFrameIsWrittenWhenMuxerReady() {
        assertEquals(
            RecorderPipeline.DrainAction.WRITE,
            RecorderPipeline.drainAction(isCodecConfig = false, size = 42, muxerStarted = true),
        )
    }

    // ── isUsableOutput ────────────────────────────────────────────────────────

    @Test fun emptyOutputFileIsRejected() {
        // BUG: 0 baytli fayl — buzuq yozuv. Uni yuklash serverni yaroqsiz fayl bilan
        // to'ldirardi; yuborishdan oldin darvozadan o'tkaziladi.
        assertFalse(RecorderPipeline.isUsableOutput(exists = true, lengthBytes = 0))
        assertFalse(RecorderPipeline.isUsableOutput(exists = false, lengthBytes = 1024))
        assertTrue(RecorderPipeline.isUsableOutput(exists = true, lengthBytes = 1))
    }

    // ── RecorderLifecycle ─────────────────────────────────────────────────────

    @Test fun secondStartIsNoOp() {
        // BUG: ikki marta start() qilinsa ikkinchi VirtualDisplay/encoder ochilib
        // birinchisi oqib ketardi. Ikkinchi beginStart() false qaytaradi (no-op).
        val lc = RecorderLifecycle()
        assertTrue("birinchi start o'tadi", lc.beginStart())
        assertFalse("ikkinchi start no-op", lc.beginStart())
        assertTrue(lc.isRunning())
    }

    @Test fun failedStartRollsBackSoRetryWorks() {
        // BUG: start ichida istisno bo'lsa holat "running" da qotib qolsa, keyingi
        // start hech qachon ishlamasdi. failStart() ni orqaga qaytaradi.
        val lc = RecorderLifecycle()
        lc.beginStart()
        lc.failStart()
        assertFalse(lc.isRunning())
        assertTrue("xatodan keyin qayta boshlash mumkin", lc.beginStart())
    }

    @Test fun secondStopIsNoOp() {
        // BUG: ikki marta stop() — allaqachon bo'shatilgan encoder/muxer ustidan
        // qayta ishlab crash bo'lardi. Ikkinchi beginStop() false qaytaradi.
        val lc = RecorderLifecycle()
        lc.beginStart()
        assertTrue("birinchi stop bajariladi", lc.beginStop())
        assertFalse("ikkinchi stop no-op", lc.beginStop())
        assertFalse(lc.isRunning())
    }

    @Test fun stopBeforeStartIsNoOp() {
        // start chaqirilmagan — stop hech nima qilmasin.
        assertFalse(RecorderLifecycle().beginStop())
    }
}
