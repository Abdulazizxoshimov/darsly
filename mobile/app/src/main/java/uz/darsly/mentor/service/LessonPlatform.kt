package uz.darsly.mentor.service

import android.app.Notification
import android.content.Context

/**
 * Darsning Android tizim chegarasi: foreground servis, bildirishnoma, efir-ramka.
 *
 * ## Nega interfeys
 *
 * `RoomViewModel`, `ScreenShareController` va [uz.darsly.mentor.data.livekit
 * .LessonSessionStore] avval `LessonService.start(ctx)`, `ShareFrameOverlay.show(ctx)`
 * va `LessonNotifications.updateSignals(ctx)` statik chaqiruvlarini to'g'ridan-to'g'ri
 * qilardi — ya'ni `Context` siz yaratilmasdi va JVM testida sinalmasdi. Aynan shu
 * sinflarda yakunlash/bo'shatish tartibi yashaydi; uni testsiz qoldirish
 * yozuvni yo'qotadigan xatolarni (C1, C2, H5, H6) qurilmagacha yashirdi.
 *
 * Bitta interfeys, uchta kichik emas: bularning hammasi "dars tizimda qanday
 * ko'rinadi" savoliga javob va bir xil umr (sessiya) bilan yashaydi.
 */
interface LessonPlatform {
    /**
     * Foreground servisni (qayta) ishga tushiradi.
     *
     * OTADI: Android 12+ fon rejimidan `ForegroundServiceStartNotAllowedException`.
     * Chaqiruvchi buni ushlashi shart — aks holda korutina jim o'lib, holat
     * yarim yo'lda qoladi (M4).
     */
    fun startService(withProjection: Boolean)

    fun stopService()

    /** Ekran ulashish bildirishnomasi (SDK `ScreenCaptureService` uchun). */
    fun screenShareNotification(): Notification?

    /** Darhol e'tibor talab qiladigan xabar — titrash bilan. */
    fun alert(text: String)

    /** Foreground bildirishnoma matni: qo'llar soni va oxirgi reaksiya. */
    fun updateSignals(hands: Int, lastReaction: String?, vibrate: Boolean)

    fun showShareFrame(recording: Boolean)

    fun hideShareFrame()
}

/** Haqiqiy Android implementatsiyasi — mavjud `object` yordamchilarga yo'naltiradi. */
class AndroidLessonPlatform(private val ctx: Context) : LessonPlatform {
    override fun startService(withProjection: Boolean) = LessonService.start(ctx, withProjection)
    override fun stopService() = LessonService.stop(ctx)
    override fun screenShareNotification(): Notification =
        LessonNotifications.build(ctx, "Ekran ulashilmoqda", summary = false)
    override fun alert(text: String) = LessonNotifications.alert(ctx, text)
    override fun updateSignals(hands: Int, lastReaction: String?, vibrate: Boolean) =
        LessonNotifications.updateSignals(ctx, hands, lastReaction, vibrate)
    override fun showShareFrame(recording: Boolean) =
        ShareFrameOverlay.show(ctx, ShareFrameOverlay.colorFor(recording))
    override fun hideShareFrame() = ShareFrameOverlay.hide()
}
