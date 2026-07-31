package uz.darsly.mentor.service

import android.content.Context
import android.content.Intent
import android.graphics.PixelFormat
import android.net.Uri
import android.provider.Settings
import android.view.Gravity
import android.view.View
import android.view.WindowManager

/**
 * Ekran chetidagi rangli RAMKA — «efir ketyapti» indikatori (M21).
 *
 * NEGA KERAK: ustoz ekran ulashib boshqa ilovaga (PDF, GeoGebra) o'tganda
 * ilovamiz ko'rinmaydi va u ulashish/yozuv davom etayotganini BILMAYDI.
 * Ramka har qanday ilova ustida turadi: bor — efir ketyapti, yo'q — tugagan.
 *
 * OGOHLANTIRISH (mahsulot qarori): overlay MediaProjection'ga TUSHADI, ya'ni
 * o'quvchilar ham ekran chetida shu ingichka ramkani ko'radi. Bu ataylab
 * qabul qilingan savdo: Google Meet'ning qizil ramkasi kabi — «yozilyapti»
 * signali ikkala tomon uchun ham halol. Ramka ~3dp, kontent o'qishiga xalaqit
 * bermaydi.
 *
 * RUXSAT: SYSTEM_ALERT_WINDOW («boshqa ilovalar ustida ko'rsatish») kerak.
 * Berilmagan bo'lsa jimgina ko'rsatilmaydi — ulashish baribir ishlaydi
 * (ramka yordamchi signal, bloklovchi talab emas).
 */
object ShareFrameOverlay {

    /** Yozuv ketayotganda — LIVE-qizil; faqat ulashishda — brend minti. */
    const val COLOR_RECORDING = 0xFFFF2D55.toInt()
    const val COLOR_SHARING = 0xFF19D3A2.toInt()

    private const val THICKNESS_DP = 3

    private var views: List<View> = emptyList()
    private var appCtx: Context? = null

    fun canDraw(ctx: Context): Boolean = Settings.canDrawOverlays(ctx)

    /** «Boshqa ilovalar ustida» ruxsat sozlamasini ochadi. */
    fun permissionIntent(ctx: Context): Intent =
        Intent(
            Settings.ACTION_MANAGE_OVERLAY_PERMISSION,
            Uri.parse("package:" + ctx.packageName),
        ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)

    /** Ramkani ko'rsatadi (allaqachon ko'rinib turgan bo'lsa rangini yangilaydi). */
    fun show(ctx: Context, color: Int) {
        val app = ctx.applicationContext
        if (!canDraw(app)) return
        hide() // rang almashganda eskisini olib tashlab qaytadan
        val wm = app.getSystemService(Context.WINDOW_SERVICE) as WindowManager
        val t = (THICKNESS_DP * app.resources.displayMetrics.density).toInt().coerceAtLeast(2)

        fun strip(gravity: Int, w: Int, h: Int): View {
            val v = View(app)
            v.setBackgroundColor(color)
            val lp = WindowManager.LayoutParams(
                w,
                h,
                WindowManager.LayoutParams.TYPE_APPLICATION_OVERLAY,
                WindowManager.LayoutParams.FLAG_NOT_FOCUSABLE or
                    WindowManager.LayoutParams.FLAG_NOT_TOUCHABLE or
                    WindowManager.LayoutParams.FLAG_LAYOUT_IN_SCREEN or
                    WindowManager.LayoutParams.FLAG_LAYOUT_NO_LIMITS,
                PixelFormat.TRANSLUCENT,
            )
            lp.gravity = gravity
            wm.addView(v, lp)
            return v
        }

        val match = WindowManager.LayoutParams.MATCH_PARENT
        views = listOf(
            strip(Gravity.TOP, match, t),
            strip(Gravity.BOTTOM, match, t),
            strip(Gravity.START, t, match),
            strip(Gravity.END, t, match),
        )
        appCtx = app
    }

    fun hide() {
        val app = appCtx ?: views.firstOrNull()?.context
        if (views.isEmpty() || app == null) return
        val wm = app.getSystemService(Context.WINDOW_SERVICE) as WindowManager
        views.forEach { runCatching { wm.removeView(it) } }
        views = emptyList()
        appCtx = null
    }

    /** Rang tanlash: yozuv yoniq bo'lsa qizil, aks holda mint. */
    fun colorFor(recording: Boolean): Int = if (recording) COLOR_RECORDING else COLOR_SHARING
}
