package uz.darsly.mentor.data.livekit

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import kotlin.math.abs

/**
 * №23 — ekran ulashish MANBASI qurilma nisbatiga moslashadi.
 *
 * O'lchov (`docs/PRODUCT.md`, 2026-08-01): qat'iy 1280×720 tik telefonda
 * kadrning 78% ini qora yo'lga sarflagan. Bu testlar shu regressiyani
 * qaytarib qo'ymaslikni qo'riqlaydi.
 */
class ScreenCaptureSizeTest {

    private val fps = 15

    /** Nisbat farqi (uzun/qisqa bo'yicha) — yaxlitlash chidamligi bilan. */
    private fun aspect(w: Int, h: Int) = maxOf(w, h).toDouble() / minOf(w, h)

    @Test
    fun `tik telefonning nisbati saqlanadi`() {
        // Redmi Note 11 kabi 20:9 telefon, tik ushlangan.
        val p = ScreenCaptureSize.forDisplay(1080, 2400, fps)
        assertTrue(
            "nisbat qurilmanikiga teng bo'lsin (qora yo'l qolmasin)",
            abs(aspect(p.width, p.height) - aspect(1080, 2400)) < 0.02,
        )
    }

    @Test
    fun `olcham har doim YOTIQ tartibda beriladi`() {
        // ⭐ LiveKit SDK bilan shartnoma (2.27.0 · bytecode'dan tekshirilgan):
        // `LocalScreencastVideoTrack.getCaptureDimensions` tik qurilmada
        // tomonlarni O'ZI almashtiradi. Tik qiymat bersak u yana ag'darilib,
        // aynan tuzatmoqchi bo'lgan xatoni qaytarardi.
        val portrait = ScreenCaptureSize.forDisplay(1080, 2400, fps)
        val landscape = ScreenCaptureSize.forDisplay(2400, 1080, fps)
        assertTrue("uzun tomon oldin", portrait.width >= portrait.height)
        assertEquals("burilish natijani o'zgartirmasin", portrait, landscape)
    }

    @Test
    fun `uzun tomon 1280 dan oshmaydi`() {
        val p = ScreenCaptureSize.forDisplay(1440, 3200, fps)
        assertEquals(ScreenCaptureSize.MAX_LONG_SIDE, p.width)
    }

    @Test
    fun `piksel byudjeti oshmaydi — planshetda ham bitreyt yetadi`() {
        // 4:3 planshet: faqat uzun tomonni cheklash 1280×960 = 1.23 Mpiksel
        // berardi va u 1.5 Mbps ga sig'masdi (matn xiralashardi).
        val p = ScreenCaptureSize.forDisplay(1600, 2560, fps)
        assertTrue(
            "piksel soni byudjetdan oshmasin (bor: ${p.width}×${p.height})",
            p.width * p.height <= ScreenCaptureSize.MAX_PIXELS,
        )
        assertTrue(abs(aspect(p.width, p.height) - aspect(1600, 2560)) < 0.02)
    }

    @Test
    fun `Zoom olchagan planshet kadriga yaqin chiqadi`() {
        // Zoom yozuvi 1280×800 (16:10) edi — mos qurilmada biz ham shunga
        // tushamiz. Bu tasodifiy emas: chegaralar aynan o'sha kadrdan olingan.
        val p = ScreenCaptureSize.forDisplay(1600, 2560, fps)
        assertEquals(1280, p.width)
        assertEquals(800, p.height)
    }

    @Test
    fun `kichik ekran KATTALASHTIRILMAYDI`() {
        // Manbada yo'q pikselni "chizib" bo'lmaydi — faqat kodlagich ishi ortardi.
        val p = ScreenCaptureSize.forDisplay(800, 480, fps)
        assertEquals(800, p.width)
        assertEquals(480, p.height)
    }

    @Test
    fun `olchamlar JUFT — kodlagich toq qiymatni rad etadi`() {
        // 4:2:0 xroma toq o'lchamda ishlamaydi; ba'zi qurilma kodlagichlari
        // buni JIMGINA qiladi: trek e'lon bo'ladi, kadr kelmaydi.
        listOf(1080 to 2337, 1179 to 2556, 1440 to 3120, 999 to 333).forEach { (w, h) ->
            val p = ScreenCaptureSize.forDisplay(w, h, fps)
            assertEquals("kenglik juft bo'lsin ($w×$h)", 0, p.width % 2)
            assertEquals("balandlik juft bo'lsin ($w×$h)", 0, p.height % 2)
        }
    }

    @Test
    fun `olcham nomalum bolsa eski xulq — 720p`() {
        val p = ScreenCaptureSize.forDisplay(0, 0, fps)
        assertEquals(1280, p.width)
        assertEquals(720, p.height)
        assertEquals(fps, p.maxFps)
    }

    @Test
    fun `fps olchamdan mustaqil otadi`() {
        assertEquals(fps, ScreenCaptureSize.forDisplay(1080, 2400, fps).maxFps)
    }

    @Test
    fun `simulcast narvoni ishlashi uchun uzun tomon 960 dan katta qoladi`() {
        // LiveKit `computeVideoEncodings` o'rta qatlamni FAQAT manba uzun tomoni
        // ≥ 960 bo'lganda qo'shadi (2.27.0 bytecode'da tekshirilgan). Ya'ni
        // o'lchamni "tejash" jimgina uch pog'onani ikkiga tushirardi va
        // `MediaTuningTest` qo'riqlayotgan bo'shliq qaytib kelardi.
        listOf(1080 to 2400, 1440 to 3200, 1600 to 2560, 1200 to 2000).forEach { (w, h) ->
            val p = ScreenCaptureSize.forDisplay(w, h, fps)
            assertTrue("$w×$h → ${p.width}×${p.height}", maxOf(p.width, p.height) >= 960)
        }
    }
}
