package uz.darsly.mentor.ui.room

import android.content.Context
import android.content.Intent
import android.os.Build
import io.livekit.android.util.LKLog
import uz.darsly.mentor.data.livekit.LessonSession
import uz.darsly.mentor.data.livekit.ScreenSharePlan
import uz.darsly.mentor.service.LessonNotifications
import uz.darsly.mentor.service.LessonService

/**
 * Ekran ulashish: boshlash, to'xtatish va uzilishdan keyin tiklash.
 *
 * ## Nega `RoomViewModel` dan ajratildi
 *
 * `RoomViewModel` 1100 qatorga yetgan va ettita mustaqil mas'uliyatni birga
 * ushlagan edi (ulanish, tarmoq, ekran, moderatsiya, chat, servis, media).
 * Ekran ulashish ular ichida eng murakkabi: unda **o'z holati** bor
 * (`wanted` niyati va saqlangan rozilik), Android 14+ ning maxsus qoidalari
 * va uch tarmoqli tiklash mantiqi.
 *
 * Alohida sinf sifatida u LiveKit'siz, Compose'siz sinaladi: qaror
 * [ScreenSharePlan] da (sof), bu yerda esa faqat uni bajarish va holatni
 * ushlash.
 *
 * ## Nega ViewModel emas
 *
 * Bu obyektning umri xona sessiyasi bilan bir xil va u `RoomUiState` ning bir
 * qismini yangilaydi. Alohida ViewModel qilinsa ikki holat manbai paydo
 * bo'lardi va ularni sinxronlash kerak bo'lardi — hozirgi muammodan yomonroq.
 *
 * @param onState UI holatini yangilash (ViewModel `_state.update` ni beradi).
 * @param onLog   diagnostika jurnaliga yozish.
 */
class ScreenShareController(
    private val appContext: Context,
    private val onState: ((RoomUiState) -> RoomUiState) -> Unit,
    private val onLog: (String) -> Unit,
) {

    /**
     * Ustoz ulashishni XOHLAYDIMI (niyat), hozir ulashilayotgani emas.
     *
     * Bu farq muhim: qayta ulanishda trek yo'qoladi, lekin niyat qoladi —
     * aynan shu tiklash kerakligini bildiradi. Aksincha, ustoz o'zi
     * to'xtatgan bo'lsa niyat ham o'chadi va tiklash BO'LMAYDI.
     */
    var wanted: Boolean = false
        private set

    /**
     * Saqlangan MediaProjection roziligi.
     *
     * Android 14+ da har sessiya uchun yangi rozilik kerak (eskisi
     * `SecurityException` beradi), shuning uchun u faqat eski versiyalarda
     * qayta ishlatiladi — qaror [ScreenSharePlan] da.
     */
    private var consent: Intent? = null

    /** Dars tugaganda yoki chiqishda — niyat ham, rozilik ham tashlanadi. */
    fun reset() {
        wanted = false
        consent = null
    }

    suspend fun start(session: LessonSession, resultData: Intent) {
        // TARTIB (Android 14+ uchun majburiy): avval `mediaProjection` tipli
        // foreground servis, keyin capture. Teskarisida tizim tipni rad etadi.
        LessonService.start(appContext, withProjection = true)
        onLog("FGS mediaProjection tipi bilan ishga tushdi")

        val notification = LessonNotifications.build(appContext, "Ekran ulashilmoqda", summary = false)
        runCatching { session.startScreenShare(resultData, notification) }
            .onSuccess {
                onLog("EKRAN ULASHISH BOSHLANDI (720p/15fps)")
                wanted = true
                consent = resultData
                onState { it.copy(restoreShare = false) }

                // Ekran audiosi mikrofon track'iga mikslanadi (API 29+ talab qiladi).
                val ok = session.startScreenAudio()
                onLog(if (ok) "ekran audiosi YOQILDI" else "ekran audiosi yoqilmadi (API<29 yoki mikrofon o'chiq)")
            }
            .onFailure { t ->
                LKLog.w(t) { "ekran ulashish boshlanmadi" }
                onLog("EKRAN ULASHISH XATO: ${t::class.simpleName}: ${t.message}")
                // Saqlangan rozilik yaroqsiz bo'lib chiqdi (Android uni bir marta
                // beradi) — uni tashlaymiz, aks holda keyingi tiklash ham shu
                // o'lik token bilan urinardi.
                consent = null
                onState {
                    if (wanted) {
                        // Ustoz ulashayotgan edi: bu TIKLASH urinishining yiqilishi.
                        // "Qayta urinib ko'ring" o'rniga bir bosishlik taklif kerak.
                        it.copy(restoreShare = true)
                    } else {
                        it.copy(error = "Ekranni ulashib bo'lmadi — qayta urinib ko'ring")
                    }
                }
                LessonService.start(appContext, withProjection = false)
            }
    }

    suspend fun stop(session: LessonSession) {
        // Ustozning O'Z qarori — endi tiklanmaydi.
        reset()
        onState { it.copy(restoreShare = false) }
        runCatching { session.stopScreenShare() }
            .onSuccess { onLog("ekran ulashish to'xtatildi") }
            .onFailure { onLog("to'xtatish XATO: ${it.message}") }
        LessonService.start(appContext, withProjection = false)
    }

    /** Ustoz "Keyinroq" dedi — taklif yopiladi, lekin niyat saqlanadi. */
    fun dismissRestorePrompt() {
        onState { it.copy(restoreShare = false) }
        onLog("ulashishni tiklash taklifi yopildi")
    }

    /**
     * ⭐ UZILISHDAN KEYIN TIKLASH (C-11 · 3-gipoteza).
     *
     * ## Nega Android 14+ da avtomatik EMAS
     * Platforma har yozib olish sessiyasi uchun yangi rozilik talab qiladi —
     * eski `Intent` qayta ishlatilsa `SecurityException`. Ya'ni "hech narsa
     * so'ramay tiklash" texnik jihatdan MUMKIN EMAS. Shuning uchun taklif ikki
     * kanal orqali beriladi: ekranda karta VA bildirishnoma — ustoz ulashish
     * paytida odatda boshqa ilovada (PDF, GeoGebra) bo'ladi va kartani ko'rmaydi.
     *
     * @return qayta ulashish uchun rozilik (mavjud bo'lsa) — chaqiruvchi
     *   [start] ni o'zi chaqiradi, chunki u korutina qamroviga ega.
     */
    fun planAfterReconnect(session: LessonSession): Intent? {
        // HAQIQAT bayroqdan emas, e'lon qilingan TREKDAN olinadi: qayta ulanishda
        // trek yo'qoladi, bayroq esa `true` qolib "ulashilmoqda" deb yolg'on
        // ko'rsatardi (qurilmada 2026-07-28 da aynan shu ko'rindi).
        val sharingNow = session.reconcileScreenShare()
        if (wanted) onLog("qayta ulanishdan keyin ekran treki: ${if (sharingNow) "bor" else "yo'q"}")

        return when (
            ScreenSharePlan.afterReconnect(
                wanted = wanted,
                sharingNow = sharingNow,
                hasToken = consent != null,
                sdkInt = Build.VERSION.SDK_INT,
            )
        ) {
            ScreenSharePlan.Action.NONE -> null

            ScreenSharePlan.Action.REUSE_TOKEN -> {
                onLog("ekran ulashish avtomatik tiklanmoqda")
                consent
            }

            ScreenSharePlan.Action.ASK_CONSENT -> {
                onLog("ekran ulashish uzildi — rozilik qayta so'raladi (Android 14+)")
                onState { it.copy(restoreShare = true) }
                // Ustoz boshqa ilovada bo'lsa kartani ko'rmaydi — titratamiz.
                LessonNotifications.alert(
                    appContext,
                    "Ekran ulashish uzildi — davom ettirish uchun bosing",
                )
                null
            }
        }
    }
}
