package uz.darsly.mentor.service

import android.app.Notification

/**
 * Android'siz tizim chegarasi — chaqiruvlarni yozib boradi.
 *
 * [startFailure] — `startService` otadigan xato (Android 12+ fon rejimi
 * `ForegroundServiceStartNotAllowedException` ni taqlid qilish uchun).
 */
class FakeLessonPlatform(var startFailure: Throwable? = null) : LessonPlatform {
    val calls = mutableListOf<String>()
    val stopServiceCalls: Int get() = calls.count { it == "stopService" }
    val alerts = mutableListOf<String>()

    override fun startService(withProjection: Boolean) {
        calls += "startService($withProjection)"
        startFailure?.let { throw it }
    }

    override fun stopService() { calls += "stopService" }
    override fun screenShareNotification(): Notification? = null
    override fun alert(text: String) { alerts += text; calls += "alert" }
    override fun updateSignals(hands: Int, lastReaction: String?, vibrate: Boolean) { calls += "signals($hands)" }
    override fun showShareFrame(recording: Boolean) { calls += "showFrame($recording)" }
    override fun hideShareFrame() { calls += "hideFrame" }
}
