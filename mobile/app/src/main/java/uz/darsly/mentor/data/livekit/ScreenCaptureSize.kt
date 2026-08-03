package uz.darsly.mentor.data.livekit

import io.livekit.android.room.track.VideoCaptureParameter
import kotlin.math.min
import kotlin.math.roundToInt
import kotlin.math.sqrt

/**
 * Ekran ulashish MANBASINING o'lchami — qurilma nisbatiga moslashadi (№23/№25).
 *
 * ## Muammo (o'lchangan, `docs/PRODUCT.md` · 2026-08-01)
 * Avval bu yerda qotib qolgan 1280×720 (16:9) turardi. Telefon ekrani esa
 * 20:9 (masalan 1080×2400). `MediaProjection` mirror-displayni maqsad sirtiga
 * **nisbatni saqlab** joylashtiradi — ya'ni mos kelmagan qismi QORA yo'l bo'lib
 * qoladi. Zoom yozuvi 1280×800 (manba nisbati, kontent butun kadrda), Jonly
 * yozuvi esa 1280×720 bo'lib, kontent pikseli 22% ga tushgan edi.
 *
 * Backend yozuv kadrini manba nisbatiga moslashtirsa ham, MANBANING o'zi
 * qora yo'lli bo'lsa foyda yo'q: qora yo'l yozuvga tushib qoladi.
 *
 * ## LiveKit SDK bilan shartnoma (2.27.0 · bytecode'dan o'qib tekshirilgan)
 * `LocalScreencastVideoTrack.getCaptureDimensions(displayW, displayH)`:
 *
 * ```
 * if (captureParams.width == 0 && captureParams.height == 0) → (displayW, displayH)
 * else if (displayW > displayH) → (captureParams.width, captureParams.height)
 * else                          → (captureParams.height, captureParams.width)   // ← almashadi
 * ```
 *
 * Ikki muhim xulosa:
 *  1. Orientatsiyani SDK O'ZI hal qiladi (va `OrientationEventListener` bilan
 *     burilishda `changeCaptureFormat` chaqiradi) — biz buni takrorlamaymiz.
 *  2. Shu sababli o'lcham **har doim yotiq (landscape) tartibda** berilishi
 *     kerak: uzun tomon oldin. Tik qiymat berilsa SDK uni tik qurilmada yana
 *     bir marta ag'darib, aynan biz tuzatmoqchi bo'lgan xatoni qaytarardi.
 *
 * Ya'ni bu yerdagi vazifa bitta: **nisbatni** qurilmanikiga tenglashtirish va
 * o'lchamni kodlagich uchun mantiqiy chegarada ushlash.
 */
object ScreenCaptureSize {

    /**
     * Uzun tomon chegarasi.
     *
     * 1280 — `MediaTuning.SCREEN_HIGH` (720p/15fps/1.5 Mbps) narvonining tepasi
     * bilan bir xil, ya'ni bitreyt byudjeti o'zgarmaydi. Undan yuqorisi
     * telefon kodlagichini qizdiradi va past internetdagi o'quvchiga baribir
     * yetib bormaydi.
     */
    const val MAX_LONG_SIDE = 1280

    /**
     * Piksel byudjeti — 1 024 000 (= 1280×800).
     *
     * Aynan Zoom o'lchagan kadr (16:10 planshet). Uzun tomonni cheklashning
     * o'zi yetarli emas: 4:3 planshetda 1280×960 = 1.23 Mpiksel chiqadi va
     * o'sha 1.5 Mbps ga sig'maydi — matn keskinligi yo'qoladi.
     */
    const val MAX_PIXELS = 1_024_000

    /** Manba o'lchami noma'lum bo'lsa — eski xulq (16:9). */
    private const val FALLBACK_LONG = 1280
    private const val FALLBACK_SHORT = 720

    /**
     * Qurilma ekrani o'lchamidan capture parametrini yasaydi.
     *
     * @param displayWidth qurilmaning HAQIQIY ekran kengligi (piksel).
     * @param displayHeight qurilmaning haqiqiy ekran balandligi (piksel).
     * @param maxFps kadr chastotasi (o'lchamdan mustaqil — `MediaTuning` dan).
     * @return yotiq tartibda (kenglik ≥ balandlik) berilgan capture parametri;
     *   tik qurilmada SDK uni o'zi ag'daradi.
     */
    fun forDisplay(
        displayWidth: Int,
        displayHeight: Int,
        maxFps: Int,
        maxLongSide: Int = MAX_LONG_SIDE,
        maxPixels: Int = MAX_PIXELS,
    ): VideoCaptureParameter {
        if (displayWidth <= 0 || displayHeight <= 0) {
            return VideoCaptureParameter(FALLBACK_LONG, FALLBACK_SHORT, maxFps)
        }

        var long = maxOf(displayWidth, displayHeight).toDouble()
        var short = min(displayWidth, displayHeight).toDouble()

        // 1) Uzun tomon chegarasi. HECH QACHON kattalashtirmaymiz: kichik
        //    ekranli qurilmada manbadan yo'q pikselni "chizib" bo'lmaydi,
        //    faqat kodlagich ishi ortardi.
        if (long > maxLongSide) {
            val k = maxLongSide / long
            long *= k
            short *= k
        }

        // 2) Piksel byudjeti (keng nisbatli emas, KVADRATROQ ekranlar uchun).
        val pixels = long * short
        if (pixels > maxPixels) {
            val k = sqrt(maxPixels / pixels)
            long *= k
            short *= k
        }

        return VideoCaptureParameter(even(long), even(short), maxFps)
    }

    /**
     * Juft songa yaxlitlaydi.
     *
     * Toq o'lcham H.264/VP8 ning 4:2:0 xromasi bilan mos kelmaydi va ba'zi
     * qurilma kodlagichlari uni jimgina rad etadi (trek e'lon bo'ladi, kadr
     * kelmaydi). Nisbatga ta'siri — bir pikseldan ko'p emas.
     */
    private fun even(value: Double): Int {
        val rounded = value.roundToInt().coerceAtLeast(2)
        return rounded - (rounded % 2)
    }
}
