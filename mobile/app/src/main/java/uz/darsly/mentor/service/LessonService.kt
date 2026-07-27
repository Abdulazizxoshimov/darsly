package uz.darsly.mentor.service

import android.Manifest
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import androidx.core.app.ServiceCompat
import androidx.core.content.ContextCompat
import io.livekit.android.util.LKLog

/**
 * Dars foreground servisi (M15 — "fon rejimida davom etish").
 *
 * MAQSAD: ustoz PDF, GeoGebra yoki brauzerni ochsa ham dars uzilmasin.
 * Android'da ilova fon rejimiga o'tganda mikrofon/kamera/ekran yozib olish
 * foreground servis BO'LMASA jim to'xtaydi.
 *
 * Tип: `mediaProjection|microphone|camera` (manifest'da).
 *
 * DIQQAT — Android 14+ (API 34) qoidasi:
 *  `mediaProjection` tipini so'raganda MediaProjection ruxsati allaqachon berilgan
 *  bo'lishi kerak. Shuning uchun bu servis IKKI rejimda ishga tushadi:
 *    · `EXTRA_WITH_PROJECTION=false` → faqat microphone|camera tipi (dars boshida)
 *    · `EXTRA_WITH_PROJECTION=true`  → mediaProjection ham qo'shiladi (ulashishdan keyin)
 *
 * TODO(R1): Room obyektini shu servisga to'liq ko'chirish (hozir [uz.darsly.mentor
 *  .data.livekit.LessonSessionHolder] process-singleton'da — servis uni faqat tirik
 *  ushlaydi). Servis Room'ni o'zi egallasa, Activity o'lganda ham dars 100% davom etadi.
 * TODO(R1): MIUI/EMUI batareya optimizatsiyasini o'chirish onboarding'i (R-2 risk).
 */
class LessonService : Service() {

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val withProjection = intent?.getBooleanExtra(EXTRA_WITH_PROJECTION, false) ?: false
        val notification = LessonNotifications.build(this)

        // B-2 TUZATISH: tipni FAQAT ruxsat haqiqatan berilgan bo'lsa so'raymiz.
        // Android 14+ da `microphone` tipi RECORD_AUDIO'siz (yoki `camera` tipi
        // CAMERA'siz) so'ralsa `SecurityException` chiqadi va servis o'ladi —
        // ya'ni ustoz fon rejimiga o'tishi bilan dars jim to'xtaydi.
        var types = 0
        // `MEDIA_PROJECTION` — API 29 (Q) dan mavjud.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q && withProjection) {
            types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PROJECTION
        }
        // `MICROPHONE` va `CAMERA` esa API 30 (R) dan. Avval bu shart `>= Q` edi:
        // konstantalar kompilyatsiyada inline bo'lgani uchun Android 10 da
        // tizim tanimaydigan bit yuborilardi (qurilma matritsasida Android 8–10 bor,
        // u yerda sinalmagan). Endi faqat R+ da qo'yiladi; Q va undan pastda servis
        // tipsiz (umumiy) foreground servis bo'lib ishlaydi — bu o'sha versiyalarda
        // to'g'ri xulq, chunki tip talabi Android 14 da kiritilgan.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            if (granted(Manifest.permission.RECORD_AUDIO)) {
                types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE
            }
            if (granted(Manifest.permission.CAMERA)) {
                types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_CAMERA
            }
        }

        try {
            ServiceCompat.startForeground(
                this,
                LessonNotifications.LESSON_NOTIFICATION_ID,
                notification,
                types,
            )
        } catch (e: Exception) {
            // Android 14+ da tip mos kelmasa yoki ruxsat bo'lmasa SecurityException/
            // ForegroundServiceStartNotAllowedException. Darsni to'xtatmaymiz —
            // faqat fon rejimi kafolatlanmaydi.
            LKLog.e(e) { "LessonService startForeground muvaffaqiyatsiz (types=$types)" }
            stopSelf()
            return START_NOT_STICKY
        }
        return START_STICKY
    }

    private fun granted(permission: String): Boolean =
        ContextCompat.checkSelfPermission(this, permission) == PackageManager.PERMISSION_GRANTED

    override fun onDestroy() {
        super.onDestroy()
        LKLog.i { "LessonService to'xtadi" }
    }

    companion object {
        private const val EXTRA_WITH_PROJECTION = "with_projection"

        fun start(ctx: Context, withProjection: Boolean = false) {
            val i = Intent(ctx, LessonService::class.java)
                .putExtra(EXTRA_WITH_PROJECTION, withProjection)
            ctx.startForegroundService(i)
        }

        fun stop(ctx: Context) {
            ctx.stopService(Intent(ctx, LessonService::class.java))
        }
    }
}
