package uz.darsly.mentor.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.VibrationEffect
import android.os.Vibrator
import android.os.VibratorManager
import androidx.core.app.NotificationCompat
import uz.darsly.mentor.MainActivity
import uz.darsly.mentor.R

/** Dars uchun bildirishnoma kanali va qurilishi. */
object LessonNotifications {

    const val CHANNEL_ID = "darsly_lesson"
    const val LESSON_NOTIFICATION_ID = 1101

    /**
     * LiveKit SDK'ning ScreenCaptureService'i uchun ID.
     * SDK standarti 2345; biz o'z bildirishnomamizni beramiz, lekin ID'ni
     * dars bildirishnomasidan FARQLI ushlaymiz (aks holda biri ikkinchisini bosadi).
     */
    const val SCREEN_CAPTURE_NOTIFICATION_ID = 1102

    fun ensureChannel(ctx: Context) {
        val mgr = ctx.getSystemService(NotificationManager::class.java) ?: return
        // minSdk 26 → NotificationChannel har doim mavjud, versiya tekshiruvi shart emas.
        if (mgr.getNotificationChannel(CHANNEL_ID) != null) return
        mgr.createNotificationChannel(
            NotificationChannel(
                CHANNEL_ID,
                ctx.getString(R.string.lesson_channel_name),
                NotificationManager.IMPORTANCE_LOW, // ovozsiz — dars ustidan bezovta qilmasin
            ),
        )
    }

    /**
     * Bildirishnomalar guruhi.
     *
     * ## Nega kerak (qurilma sinovida topildi)
     * Ilova ikkita doimiy bildirishnoma chiqaradi: dars (1101) va ekran yozib
     * olish servisi (1102). Android ularni O'ZI guruhlab, pardada faqat
     * "Darsly Mentor" degan quruq sarlavha ko'rsatardi — ya'ni `✋ 1 · 👍 Ali`
     * matni yashirin qolardi va butun signal mexanizmi ma'nosini yo'qotardi.
     *
     * Guruhni O'ZIMIZ e'lon qilib, dars bildirishnomasini SARLAVHA (summary)
     * qilamiz: endi yig'ilgan holatda ham aynan bizning matn ko'rinadi.
     */
    private const val GROUP_KEY = "darsly_lesson_group"

    /**
     * @param summary bu bildirishnoma guruh sarlavhasimi. Faqat dars
     *   bildirishnomasi (1101) uchun `true` — ekran yozib olishniki (1102)
     *   guruhning bolasi bo'lib qoladi.
     */
    fun build(ctx: Context, text: String? = null, summary: Boolean = true): Notification {
        ensureChannel(ctx)
        val openApp = android.app.PendingIntent.getActivity(
            ctx,
            0,
            Intent(ctx, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
            },
            android.app.PendingIntent.FLAG_IMMUTABLE,
        )
        return NotificationCompat.Builder(ctx, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.presence_video_online)
            .setContentTitle(ctx.getString(R.string.lesson_notification_title))
            .setContentText(text ?: ctx.getString(R.string.lesson_notification_text))
            .setOngoing(true)
            .setSilent(true)
            .setContentIntent(openApp)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setGroup(GROUP_KEY)
            .setGroupSummary(summary)
            // Yig'ilgan holatda ham matn ko'rinsin: qisqa matn uchun ham
            // `BigTextStyle` sarlavha qatoriga tushib qolishdan saqlaydi.
            .setStyle(NotificationCompat.BigTextStyle().bigText(text ?: ctx.getString(R.string.lesson_notification_text)))
            .build()
    }

    /**
     * Dars signallarini (qo'l, reaksiya) bildirishnoma matniga chiqaradi.
     *
     * ## Nega aynan bildirishnoma
     * Ustoz ekranini ulashganda ilova fonda qoladi: Compose render qilmaydi va
     * ekrandagi ro'yxat unga ko'rinmaydi. Overlay (`SYSTEM_ALERT_WINDOW`) esa bu
     * yerda yaramaydi — `MediaProjection` ekranni **bor holicha** yozib beradi,
     * ya'ni suzuvchi oyna O'QUVCHILARGA HAM ko'rinardi. Bildirishnoma esa
     * ulashuvga tushmaydi (pardani ochmaguncha) va hech qanday ruxsat talab qilmaydi.
     *
     * `vibrate` faqat YANGI qo'l uchun: kanal `IMPORTANCE_LOW` bo'lgani uchun
     * ovoz chiqmaydi, titrash esa ustozni "kimdir so'radi" deb ogohlantiradi.
     */
    fun updateSignals(ctx: Context, hands: Int, lastReaction: String?, vibrate: Boolean) {
        val parts = buildList {
            if (hands > 0) add("✋ $hands")
            lastReaction?.let { add(it) }
        }
        val text = if (parts.isEmpty()) null else parts.joinToString(" · ")
        notify(ctx, text, vibrate)
    }

    /**
     * Darhol e'tibor talab qiladigan xabar (C-11 · ulashishni tiklash).
     *
     * Ustoz ekranini ulashganda ilova FONDA bo'ladi — ekrandagi karta unga
     * ko'rinmaydi. Ulashish uzilib qolganini u faqat o'quvchilar aytganda
     * bilardi. Titrash bilan chiqadigan bildirishnoma shu bo'shliqni yopadi;
     * bosilsa ilova ochiladi va tiklash kartasi ekranda turadi.
     */
    fun alert(ctx: Context, text: String) = notify(ctx, text, vibrate = true)

    private fun notify(ctx: Context, text: String?, vibrate: Boolean) {
        val mgr = ctx.getSystemService(NotificationManager::class.java) ?: return
        ensureChannel(ctx)
        val n = build(ctx, text)
        // Titratish bildirishnomaning o'zidan emas (kanal LOW — u e'tiborsiz
        // qoladi), balki alohida chaqiriladi: kanal muhimligini oshirish darsning
        // o'rtasida ovoz chiqarib yuborardi.
        if (vibrate) vibrate(ctx)
        runCatching { mgr.notify(LESSON_NOTIFICATION_ID, n) }
    }

    private fun vibrate(ctx: Context) {
        val effect = VibrationEffect.createOneShot(VIBRATE_MS, VibrationEffect.DEFAULT_AMPLITUDE)
        runCatching {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                val vm = ctx.getSystemService(VibratorManager::class.java) ?: return
                vm.defaultVibrator.vibrate(effect)
            } else {
                @Suppress("DEPRECATION")
                val v = ctx.getSystemService(Vibrator::class.java) ?: return
                v.vibrate(effect)
            }
        }
    }

    private const val VIBRATE_MS = 120L
}
