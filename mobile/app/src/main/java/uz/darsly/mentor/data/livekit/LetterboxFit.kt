package uz.darsly.mentor.data.livekit

import kotlin.math.min
import kotlin.math.roundToInt

/**
 * Manba video kadrini gorizontal "tuval" (canvas) ichiga NISBATNI saqlab
 * joylashtirish hisobi — letterbox (qora chegara bilan).
 *
 * ## Nega bu kerak
 * [LocalRecorder] yozuvni endi DOIM gorizontal kadrga (1280×720) chiqaradi,
 * shunda fayl istalgan pleyerda kino kabi to'liq ekranga ochiladi (Zoom kabi).
 * Manba esa tik (portret telefon ekrani) yoki gorizontal (planshet/doska
 * ilovasi) bo'lishi mumkin, hatto dars o'rtasida burilishi ham mumkin. Bu
 * yerdagi vazifa — manbani nisbatini saqlab tuval markaziga joylash.
 *
 * ## Nega alohida, sof obyekt (Android'siz, JVM testida)
 * Bitta piksel xato butun kadrni siljitadi yoki I420 (4:2:0) xroma tekisligini
 * buzadi (natijada yashil/pushti chiziqlar). Shuning uchun hisob — qaysi
 * o'lcham va qayerga qo'yilishi — test bilan qotirilgan. Aynan
 * [ScreenCaptureSize] va [RecordingFormat] bilan bir uslub.
 */
object LetterboxFit {

    /** Tuval ichidagi joylashuv: (x,y) chap-tepa burchak, (w,h) o'lcham. Barchasi JUFT. */
    data class Rect(val x: Int, val y: Int, val w: Int, val h: Int)

    /**
     * [srcW]×[srcH] manbani [canvasW]×[canvasH] tuval ichiga nisbatni saqlab
     * joylashtiradi. Natija markazda; qolgan joyni chaqiruvchi qora bilan
     * to'ldiradi.
     *
     * Barcha qiymatlar JUFT va tuval ichida ([x]+[w] ≤ canvasW, [y]+[h] ≤
     * canvasH) — I420 xromasi (yarim o'lcham) toq qiymatni ko'tarmaydi. Manba
     * o'lchami noma'lum (≤0) bo'lsa butun tuval qaytadi.
     */
    fun fit(srcW: Int, srcH: Int, canvasW: Int, canvasH: Int): Rect {
        if (srcW <= 0 || srcH <= 0) return Rect(0, 0, canvasW, canvasH)
        // Nisbatni saqlab sig'diradigan yagona masshtab: eng qattiq o'lcham
        // hal qiladi (kenglik yoki balandlik — qaysi biri avval to'lsa).
        val scale = min(canvasW.toDouble() / srcW, canvasH.toDouble() / srcH)
        val w = evenDown((srcW * scale).roundToInt()).coerceIn(2, canvasW)
        val h = evenDown((srcH * scale).roundToInt()).coerceIn(2, canvasH)
        val x = evenDown((canvasW - w) / 2)
        val y = evenDown((canvasH - h) / 2)
        return Rect(x, y, w, h)
    }

    /** Juft songa PASTGA yaxlitlaydi (toq o'lcham 4:2:0 xromasini buzadi). */
    private fun evenDown(v: Int): Int = v - (v and 1)
}
