package uz.darsly.mentor.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
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

    fun build(ctx: Context, text: String? = null): Notification {
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
            .build()
    }
}
