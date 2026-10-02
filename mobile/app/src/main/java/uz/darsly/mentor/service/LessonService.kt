package uz.darsly.mentor.service

import android.Manifest
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.IBinder
import androidx.core.app.ServiceCompat
import androidx.core.content.ContextCompat
import dagger.hilt.android.AndroidEntryPoint
import io.livekit.android.util.LKLog
import uz.darsly.mentor.data.livekit.LessonSessionStore
import javax.inject.Inject

/**
 * Dars foreground servisi (M15 — "fon rejimida davom etish").
 *
 * MAQSAD: ustoz PDF, GeoGebra yoki brauzerni ochsa ham dars uzilmasin.
 * Android'da ilova fon rejimiga o'tganda mikrofon/kamera/ekran yozib olish
 * foreground servis BO'LMASA jim to'xtaydi.
 *
 * Tip: `mediaProjection|microphone|camera` (manifest'da).
 *
 * DIQQAT — Android 14+ (API 34) qoidasi:
 *  `mediaProjection` tipini so'raganda MediaProjection ruxsati allaqachon berilgan
 *  bo'lishi kerak. Shuning uchun bu servis IKKI rejimda ishga tushadi:
 *    · `EXTRA_WITH_PROJECTION=false` → faqat microphone|camera tipi (dars boshida)
 *    · `EXTRA_WITH_PROJECTION=true`  → mediaProjection ham qo'shiladi (ulashishdan keyin)
 *
 * ## Egalik (M2)
 * Sessiya [LessonSessionStore] da; servis uning hayotiy-sikl QOROVULI: ustoz
 * ilovani recents'dan surib tashlasa ([onTaskRemoved]) sessiyani bo'shatadi.
 * Sessiya bo'shatilganda esa store servisni to'xtatadi — ikki yo'l bitta
 * nuqtaga (`LessonSessionStore.release`) keladi va ikkalasi idempotent.
 *
 * TODO(R1): MIUI/EMUI batareya optimizatsiyasini o'chirish onboarding'i (R-2 risk).
 */
@AndroidEntryPoint
class LessonService : Service() {

    @Inject lateinit var sessions: LessonSessionStore

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val withProjection = intent?.getBooleanExtra(EXTRA_WITH_PROJECTION, false) ?: false
        val notification = LessonNotifications.build(this)

        // Tiplar sof rejadan (`ForegroundServicePlan`): faqat ruxsat haqiqatan
        // berilgan va API mos bo'lsa so'raladi (B-2).
        val types = ForegroundServicePlan.serviceTypes(
            sdkInt = Build.VERSION.SDK_INT,
            withProjection = withProjection,
            micGranted = granted(Manifest.permission.RECORD_AUDIO),
            camGranted = granted(Manifest.permission.CAMERA),
        )

        try {
            ServiceCompat.startForeground(
                this,
                LessonNotifications.LESSON_NOTIFICATION_ID,
                notification,
                types,
            )
        } catch (e: Exception) {
            // Android 14+ da tip mos kelmasa yoki ruxsat bo'lmasa SecurityException/
            // ForegroundServiceStartNotAllowedException.
            //
            // H6: avval bu yerda `stopSelf()` turardi — u `onDestroy` orqali JONLI
            // sessiyani bo'shatardi, ya'ni "fon kafolati yo'q" degan kichik nuqson
            // "dars o'ldi" degan katta nuqsonga aylanardi. Endi sessiya bo'lsa
            // servis tirik qoladi (fonsiz), sessiya bo'lmasa — to'xtaydi.
            LKLog.e(e) { "LessonService startForeground muvaffaqiyatsiz (types=$types)" }
            val active = sessions.session.value != null
            if (ForegroundServicePlan.onForegroundFailed(active) == ForegroundServicePlan.OnForegroundFailed.STOP_SELF) {
                stopSelf()
            }
            return START_NOT_STICKY
        }
        // START_NOT_STICKY (avval START_STICKY edi).
        //
        // START_STICKY da tizim servisni NULL intent bilan qayta ishga tushiradi.
        // Bunda `withProjection=false` bo'ladi va bildirishnoma "Dars davom
        // etmoqda" deb turadi — holbuki hech qanday sessiya yo'q: LiveKit
        // ulanishi ham, ekran ulashish ham allaqachon o'lgan. Foydalanuvchi
        // uchun bu yolg'on holat: u darsga qaytolmaydi, lekin ilova davom
        // etayotgandek ko'rsatadi.
        //
        // Dars sessiyasi qayta tiklanishi kerak bo'lsa, buni Activity boshqaradi
        // (`LessonSessionStore` + token bilan) — tizimning "ko'r" restarti emas.
        return START_NOT_STICKY
    }

    /**
     * Foydalanuvchi ilovani "recents" (so'nggi ilovalar) ro'yxatidan surib
     * tashladi.
     *
     * # Nega bu MAXFIYLIK masalasi (M14)
     *
     * Activity o'ladi, lekin foreground servis tirik qoladi — u bilan birga
     * **MediaProjection ham**. Ya'ni ustoz ilovani yopdim deb o'ylaydi, ekrani
     * esa yozib olinishda/ulashilishda davom etaveradi. Bu shunchaki resurs
     * isrofi emas: ekranda parol, shaxsiy xat yoki boshqa dars ochilishi mumkin.
     *
     * Bo'shatish store'ning ilova qamrovida (asinxron): yozuvni yakunlash
     * bir necha soniya olishi mumkin va servis callback'ida bloklab bo'lmaydi.
     */
    override fun onTaskRemoved(rootIntent: Intent?) {
        LKLog.i { "LessonService: ilova recents'dan olib tashlandi — sessiya tozalanmoqda" }
        sessions.releaseAllAsync()
        stopSelf()
        super.onTaskRemoved(rootIntent)
    }

    private fun granted(permission: String): Boolean =
        ContextCompat.checkSelfPermission(this, permission) == PackageManager.PERMISSION_GRANTED

    override fun onDestroy() {
        super.onDestroy()
        // Bu yerda sessiya ATAYLAB bo'shatilmaydi: servisni store'ning o'zi
        // to'xtatadi (sessiya allaqachon bo'shatilgan), `onDestroy` esa asosiy
        // oqimga KECHIKIB keladi — o'sha paytda yangi dars boshlangan bo'lishi
        // mumkin va "joriy sessiyani bo'shat" yangi darsni o'ldirardi. Tizim
        // foreground servisni xotira uchun o'ldirsa, u bilan process ham ketadi.
        // Foydalanuvchi tashabbusi (`onTaskRemoved`) yuqorida alohida qoplangan.
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
