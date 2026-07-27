package uz.darsly.mentor.data.livekit

import android.content.Context
import uz.darsly.mentor.data.api.RoomToken

/**
 * Jonli dars sessiyasining process darajasidagi egasi.
 *
 * Nega Activity/ViewModel emas: ustoz ekranni bursa yoki tizim Activity'ni qayta
 * yaratsa LiveKit [LessonSession] qayta ulanmasligi kerak — dars uzilmasin.
 * `LessonService` (foreground) process'ni tirik ushlaydi, holat esa shu yerda.
 *
 * TODO(R1): egalikni to'liq `LessonService`ga ko'chirish + bir vaqtda faqat
 *  bitta sessiya bo'lishini majburlash (hozir `start` avvalgisini yopadi).
 */
object LessonSessionHolder {

    @Volatile
    private var current: LessonSession? = null

    val session: LessonSession? get() = current

    @Synchronized
    fun start(ctx: Context, lessonId: String, token: RoomToken): LessonSession {
        current?.release()
        return LessonSession(ctx.applicationContext, lessonId, token).also { current = it }
    }

    @Synchronized
    fun stop() {
        current?.release()
        current = null
    }
}
