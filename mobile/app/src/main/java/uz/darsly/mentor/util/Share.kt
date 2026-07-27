package uz.darsly.mentor.util

import android.app.Activity
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.core.content.getSystemService
import io.livekit.android.util.LKLog

/**
 * Native "Ulashish" (M8).
 *
 * NEGA MUHIM: ustozlar join havolasini amalda **Telegram** orqali yuboradi. Tizim
 * chooser'i Telegram, WhatsApp, SMS — hammasini bitta ro'yxatda beradi va bizga
 * hech qanday integratsiya kerak emas.
 */
object Share {

    /** Matnni tizim "Ulashish" oynasi orqali yuboradi. `false` = ochadigan ilova yo'q. */
    fun sendText(context: Context, text: String, title: String = "Dars havolasini ulashish"): Boolean {
        val send = Intent(Intent.ACTION_SEND).apply {
            type = "text/plain"
            putExtra(Intent.EXTRA_TEXT, text)
            putExtra(Intent.EXTRA_SUBJECT, title)
        }
        val chooser = Intent.createChooser(send, title).apply {
            // Activity bo'lmagan kontekstdan (masalan Application) ochilsa shart.
            if (context !is Activity) addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return runCatching { context.startActivity(chooser) }
            .onFailure { LKLog.w(it) { "ulashish oynasi ochilmadi" } }
            .isSuccess
    }

    /**
     * Klipbordga nusxa oladi.
     *
     * `true` qaytsa — ilova o'zi "nusxa olindi" deb aytishi kerak. Android 13+ da
     * tizim buni **o'zi** ko'rsatadi, ikkinchi xabar ortiqcha shovqin bo'lardi.
     */
    fun copyToClipboard(context: Context, text: String, label: String = "Dars havolasi"): Boolean {
        val cm = context.getSystemService<ClipboardManager>() ?: return false
        return runCatching {
            cm.setPrimaryClip(ClipData.newPlainText(label, text))
            Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU
        }.getOrElse {
            LKLog.w(it) { "klipbordga nusxa olinmadi" }
            false
        }
    }
}
