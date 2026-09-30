package uz.darsly.mentor.service

import android.content.pm.ServiceInfo
import android.os.Build

/**
 * [LessonService] ning **sof** qarorlari — JVM testida qulflangan.
 *
 * `Service` ni JVM'da yaratib bo'lmaydi, lekin undagi ikki qaror aynan
 * sinovga muhtoj: qaysi foreground tiplarini so'rash (noto'g'ri bit Android
 * 14+ da `SecurityException` — dars fon rejimida jim to'xtaydi) va
 * `startForeground` yiqilganda nima qilish (H6: avval `stopSelf()` edi va u
 * `onDestroy` orqali JONLI sessiyani o'ldirardi — izohda esa "darsni
 * to'xtatmaymiz" deb yozilgan edi).
 */
object ForegroundServicePlan {

    /**
     * Foreground servis tiplari (bit maskasi).
     *
     * · `MEDIA_PROJECTION` — API 29 (Q) dan va faqat proyeksiya ruxsati OLINGANDAN keyin.
     * · `MICROPHONE`/`CAMERA` — API 30 (R) dan va faqat ruxsat haqiqatan berilgan bo'lsa
     *   (B-2). Q da bu konstantalar tizimga notanish bit bo'lib ketardi.
     */
    fun serviceTypes(sdkInt: Int, withProjection: Boolean, micGranted: Boolean, camGranted: Boolean): Int {
        var types = 0
        if (sdkInt >= Build.VERSION_CODES.Q && withProjection) {
            types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PROJECTION
        }
        if (sdkInt >= Build.VERSION_CODES.R) {
            if (micGranted) types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE
            if (camGranted) types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_CAMERA
        }
        return types
    }

    /** `startForeground` yiqilganda servis o'zini nima qilishi kerak. */
    enum class OnForegroundFailed {
        /** Sessiya jonli — servis oddiy (fonsiz) holda TIRIK qoladi, dars davom etadi. */
        KEEP_RUNNING,

        /** Sessiya yo'q — ushlab turadigan narsa yo'q, servis to'xtaydi. */
        STOP_SELF,
    }

    fun onForegroundFailed(sessionActive: Boolean): OnForegroundFailed =
        if (sessionActive) OnForegroundFailed.KEEP_RUNNING else OnForegroundFailed.STOP_SELF
}
