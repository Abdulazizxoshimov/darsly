package uz.darsly.mentor.ui.room

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.livekit.LessonSession
import uz.darsly.mentor.data.livekit.NetworkSwitchPolicy
import uz.darsly.mentor.data.livekit.Transport
import uz.darsly.mentor.data.livekit.TransportSource

/**
 * Tarmoq almashuvi va majburiy qayta ulanish (C-11).
 *
 * ## Nega `RoomViewModel` dan ajratildi
 *
 * Bu blokda uchta o'zaro bog'liq narsa bor: tarmoq kuzatuvi, backoff bilan
 * takrorlanadigan qayta ulanish sikli va `intentionalReconnect` bayrog'i
 * (u `onDisconnected` ni chalg'itmasligi uchun kerak). Ular `RoomViewModel`
 * ichida boshqa yettita mas'uliyat bilan aralashib yotgan edi.
 *
 * Qaror [NetworkSwitchPolicy] da (sof, testlar bilan qotirilgan) — bu yerda
 * faqat uni bajarish va sikl holatini ushlash. Tarmoq manbai [TransportSource]
 * orqali — testda qo'lda boshqariladi; kutish [sleep] bilan almashtiriladi.
 */
class ReconnectController(
    private val transports: TransportSource,
    private val scope: CoroutineScope,
    private val onState: ((RoomUiState) -> RoomUiState) -> Unit,
    private val onLog: (String) -> Unit,
    /** Qayta ulangach ekran ulashishni tiklash imkonini beradi. */
    private val onReconnected: () -> Unit,
    /** Test uchun: haqiqiy kutishni almashtirish. */
    private val sleep: suspend (Long) -> Unit = { delay(it) },
) {

    /**
     * Hozir O'ZIMIZ qayta ulanish uchun uzayapmizmi.
     *
     * `onDisconnected` shu bayroqqa qarab "dars tugadi" degan xulosa chiqarmaydi:
     * bizning `disconnect()` chaqiruvimiz ham xuddi haqiqiy uzilish kabi hodisa
     * beradi va usiz dars o'rtasida "Dars tugadi" ekrani chiqib qolardi.
     */
    @Volatile
    var intentional: Boolean = false
        private set

    private var job: Job? = null

    /**
     * Tarmoq almashuvini kuzatadi.
     *
     * LiveKit SDK'sining o'z qayta ulanishi bor, lekin Android'da eski interfeys
     * DARHOL o'lmaydi: soket ochiq ko'rinadi, paket ketmaydi. SDK buni faqat
     * timeout orqali sezadi va ustoz shu vaqt jim ekranga qarab turadi.
     * Transport o'zgarishi esa "eski yo'l yaroqsiz" degan ANIQ signal.
     */
    fun observeNetwork(state: MutableStateFlow<RoomUiState>, session: () -> LessonSession?): Job =
        scope.launch {
            var prev: Transport? = null
            transports.transports().collect { now ->
                val connected = state.value.connState.inRoom
                val note = NetworkSwitchPolicy.label(prev, now)
                if (note != null) onLog("tarmoq: $note")
                onState { it.copy(networkNote = note) }

                if (NetworkSwitchPolicy.shouldForceReconnect(prev, now, connected)) {
                    onLog("tarmoq almashdi — majburiy qayta ulanish")
                    session()?.let { force(it) }
                }
                prev = now
            }
        }

    /**
     * Majburiy qayta ulanish: eski (yaroqsiz) ulanishni uzib, qaytadan ulanadi.
     *
     * ## Nega TAKRORIY urinish shart (qurilma sinovi, 2026-07-28)
     * Birinchi versiya bir marta urinardi. LTE'ga o'tishda o'sha yagona urinish
     * `UnknownHostException` bilan yiqildi (tarmoq hali DNS uchun tayyor emas edi)
     * va dars **butunlay o'lik qoldi**: `disconnect()` chaqirilgani uchun SDK'ning
     * o'z retry sikli ham ishlamasdi. Ya'ni tuzatish o'zi yangi nosozlik yasagandi.
     *
     * Endi urinishlar backoff bilan takrorlanadi va ulanish tiklanishi bilan
     * to'xtaydi. Chegaradan oshsa — foydalanuvchiga rost xabar beriladi.
     */
    fun force(s: LessonSession, isCurrent: (LessonSession) -> Boolean = { true }) {
        // Bir vaqtda bitta qayta ulanish sikli.
        if (job?.isActive == true) return
        job = scope.launch {
            intentional = true
            try {
                onState { it.copy(reconnecting = true) }
                runCatching { s.room.disconnect() }

                for ((attempt, delayMs) in RECONNECT_DELAYS.withIndex()) {
                    sleep(delayMs)
                    if (!isCurrent(s)) return@launch // sessiya almashdi/yopildi
                    val result = runCatching { s.connect() }
                    if (result.isSuccess) {
                        onLog("qayta ulandi (${attempt + 1}-urinish)")
                        onState { it.copy(reconnecting = false, networkNote = null) }
                        onReconnected()
                        return@launch
                    }
                    // SABAB ham yoziladi: qurilma sinovida "muvaffaqiyatsiz" degan
                    // quruq qator xatoni topishga yordam bermadi (DNS mi, imzo mi,
                    // dublikat identity mi — bilib bo'lmasdi).
                    val t = result.exceptionOrNull()
                    onLog(
                        "qayta ulanish ${attempt + 1}-urinish muvaffaqiyatsiz: " +
                            "${t?.let { it::class.simpleName }}: ${t?.message}",
                    )
                }
                onLog("qayta ulanib bo'lmadi — urinishlar tugadi")
                onState { it.copy(reconnecting = false, networkNote = "Qayta ulanib bo'lmadi") }
            } finally {
                // Bayroq HAR QANDAY holatda tushadi: aks holda keyingi HAQIQIY
                // uzilish ham jimgina yutilib, dars osilib qolardi.
                intentional = false
            }
        }
    }

    fun cancel() {
        job?.cancel()
        job = null
        intentional = false
    }

    internal companion object {
        /**
         * Majburiy qayta ulanish urinishlari orasidagi kutish (ms).
         *
         * Birinchisi 800 ms: tarmoq TASDIQLANGAN bo'lsa ham marshrut/DNS keshi bir
         * lahza kechikishi mumkin. Keyingilari o'sib boradi — umumiy oyna ~46 soniya,
         * bu [uz.darsly.mentor.data.livekit.LessonReconnectPolicy] oynasi bilan mos.
         *
         * ⚠️ Bu qiymatlar QURILMADA o'lchangan (C-11, 2026-07-28). Ularni
         * o'zgartirish tarmoq almashuvidagi tiklanishni buzishi mumkin —
         * o'zgartirilsa qayta o'lchash shart.
         */
        val RECONNECT_DELAYS = longArrayOf(800, 1500, 3000, 6000, 10000, 10000, 15000)
    }
}
