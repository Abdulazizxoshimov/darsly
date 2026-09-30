package uz.darsly.mentor.data.ws

import io.livekit.android.util.LKLog
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import uz.darsly.mentor.BuildConfig
import java.util.concurrent.TimeUnit
import kotlin.random.Random

/** Real-time kanal holati — UI "jonli/uzilgan" ko'rsatishi uchun. */
enum class RealtimeState { DISCONNECTED, CONNECTING, CONNECTED }

/**
 * Serverning real-time kanali (`GET /api/v1/ws`).
 *
 * ## Nega kerak
 * Kutish xonasidagi o'quvchi **3 soniyada** ustozga ko'rinishi kerak (mezon D-1) —
 * so'rovni takroriy `GET` bilan tortish (polling) na tezkor, na batareyaga do'st.
 * Bildirishnomalar va dars holati ham shu kanaldan keladi.
 *
 * ## Protokol
 * Konvert web klienti bilan **bir xil**: `{type, room, payload, created_at}`,
 * qayta ulanish eksponensial + jitter ([Backoff]).
 *
 * Token **`Authorization` header'ida** (S1). Brauzer WS handshake'da header
 * yubora olmagani uchun web `?token=` ishlatadi; backend ikkalasini ham qabul
 * qiladi va header'ni AFZAL ko'radi (`api/middleware/auth.go`). Mobil OkHttp
 * header yubora oladi — access JWT'ni URL'ga (access-log, proxy, Referer)
 * qo'yish uchun sabab yo'q.
 *
 * ## Qayta ulanish (H1 · M1)
 * Sikl FAQAT [stop] (logout) bilan tugaydi. Serverning o'zi yopgan ulanish —
 * hatto 1000 (normal) kodi bilan ham — qayta ulanish sababi: backend deploy'da
 * aynan 1000 yuboradi (`websocket.go` `Stop`) va avval klient uni "tugadi" deb
 * chiqib ketardi — har deploy'dan keyin barcha mentorlar kutish xonasi
 * so'rovlarini qayta login qilguncha ko'rmasdi. Har sabab (401 ham) backoff
 * bilan: avval 401 da zero-delay issiq sikl bor edi.
 *
 * ## Token eskirishi
 * Handshake **401** bersa, token eskirgan. Bu holda oddiy authed so'rov yuboriladi —
 * `Net` dagi [uz.darsly.mentor.data.api.TokenAuthenticator] uni ushlab **single-flight
 * refresh** qiladi — va yangi token bilan qayta ulanamiz. Refresh ham yiqilsa
 * authenticator toza logout qiladi → [accessToken] `null` → sikl tarmoqqa
 * chiqmasdan kutadi (sikl yo'q).
 */
class RealtimeClient(
    private val client: OkHttpClient = defaultClient,
    private val parser: RealtimeParser = RealtimeParser(),
    private val baseUrl: String = BuildConfig.API_BASE_URL,
    /** Test uchun: haqiqiy kutishni almashtirish. */
    private val sleep: suspend (Long) -> Unit = { delay(it) },
    /** Jitter manbai `[0, 1)` — testda deterministik. */
    private val jitter: () -> Double = { Random.nextDouble() },
    /**
     * Joriy access token manbai. Funksiya sifatida uzatiladi (qiymat emas):
     * soket qayta ulanganda EN SO'NGGI token kerak, ulanish yaratilgandagisi emas.
     */
    private val accessToken: () -> String? = { null },
    /**
     * Tokenni yangilaydi (401 dan keyin). Ichida oddiy authed so'rov bajariladi —
     * OkHttp authenticator'i uni ushlab refresh qiladi. Funksiya sifatida
     * uzatiladi, chunki `RealtimeClient` API turini bilishi shart emas.
     */
    private val refreshToken: suspend () -> Unit = {},
) {

    private val _events = MutableSharedFlow<RealtimeEvent>(extraBufferCapacity = 32)

    /** Hodisalar oqimi. Obunachi bo'lmasa xabar tashlanadi (holat emas, hodisa). */
    val events: SharedFlow<RealtimeEvent> = _events

    private val _state = MutableStateFlow(RealtimeState.DISCONNECTED)
    val state: StateFlow<RealtimeState> = _state.asStateFlow()

    private var socket: WebSocket? = null
    private var loop: Job? = null

    /** Sessiya ochilganda chaqiriladi. Takroriy chaqiruv zararsiz. */
    @Synchronized
    fun start(scope: CoroutineScope) {
        if (loop?.isActive == true) return
        loop = scope.launch { runLoop() }
    }

    /** Logoutda chaqiriladi — soket yopiladi va qayta ulanish to'xtaydi. */
    @Synchronized
    fun stop() {
        loop?.cancel()
        loop = null
        socket?.close(NORMAL_CLOSURE, "logout")
        socket = null
        _state.value = RealtimeState.DISCONNECTED
    }

    /**
     * Ulanish sikli: ulan → uzilguncha kut → kechikib qayta urin.
     *
     * Sikl **korutina bekor qilinmaguncha** davom etadi; har muvaffaqiyatli ulanishda
     * urinishlar hisobi nolga tushadi (uzoq dars davomida kechikish o'sib ketmasin).
     */
    private suspend fun runLoop() {
        var attempt = 0
        while (currentScopeActive()) {
            val token = accessToken()
            if (token.isNullOrBlank()) {
                // Hali login bo'lmagan — qisqa kutib qayta ko'ramiz.
                sleep(Backoff.BASE_MS)
                continue
            }

            _state.value = RealtimeState.CONNECTING
            val outcome = connectAndWait(token)
            _state.value = RealtimeState.DISCONNECTED

            if (outcome.connected) attempt = 0
            if (outcome.reason == CloseReason.UNAUTHORIZED) {
                // Token eskirgan — OkHttp authenticator'i orqali yangilaymiz.
                runCatching { refreshToken() }
                    .onFailure { LKLog.w(it) { "realtime: token yangilanmadi" } }
            }

            val wait = Backoff.jitteredMs(attempt, jitter())
            LKLog.i { "realtime: qayta ulanish ${wait}ms dan keyin (urinish=$attempt, sabab=${outcome.reason})" }
            sleep(wait)
            attempt++
        }
    }

    private enum class CloseReason { SERVER_CLOSED, ERROR, UNAUTHORIZED }

    /** Bitta ulanishning yakuni: sabab va ulanish umuman ochilganmi (backoff'ni nollash uchun). */
    private class Outcome(val reason: CloseReason, val connected: Boolean)

    /** Bitta ulanish: ochadi, hodisalarni uzatadi va yopilgunicha kutadi. */
    private suspend fun connectAndWait(token: String): Outcome {
        val url = baseUrl.trimEnd('/').replaceFirst("http", "ws") + WS_PATH

        var connected = false
        val done = CompletableDeferred<CloseReason>()

        val ws = client.newWebSocket(
            Request.Builder().url(url).header("Authorization", "Bearer $token").build(),
            object : WebSocketListener() {
                override fun onOpen(webSocket: WebSocket, response: Response) {
                    connected = true
                    _state.value = RealtimeState.CONNECTED
                    LKLog.i { "realtime: ulandi" }
                }

                override fun onMessage(webSocket: WebSocket, text: String) {
                    val event = parser.parse(text)
                    if (event == null) {
                        LKLog.w { "realtime: xabar o'qilmadi (e'tiborsiz)" }
                        return
                    }
                    if (event is RealtimeEvent.Unknown) {
                        LKLog.i { "realtime: notanish tur '${event.type}' — e'tiborsiz" }
                    }
                    _events.tryEmit(event)
                }

                override fun onClosing(webSocket: WebSocket, code: Int, reasonText: String) {
                    // Server yopdi (deploy, sessiya bekor, chegara) — kodi qanday
                    // bo'lmasin qayta ulanamiz; faqat [stop] siklni tugatadi.
                    webSocket.close(NORMAL_CLOSURE, null)
                    LKLog.i { "realtime: server yopdi (kod=$code, sabab='$reasonText')" }
                    done.complete(CloseReason.SERVER_CLOSED)
                }

                override fun onClosed(webSocket: WebSocket, code: Int, reasonText: String) {
                    done.complete(CloseReason.SERVER_CLOSED)
                }

                override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                    val code = response?.code
                    LKLog.w(t) { "realtime: uzildi (kod=$code)" }
                    done.complete(if (code == 401 || code == 403) CloseReason.UNAUTHORIZED else CloseReason.ERROR)
                }
            },
        )
        socket = ws
        val reason = done.await()
        socket = null
        return Outcome(reason, connected)
    }

    /** Korutina hali tirikmi (bekor qilinmaganmi). */
    private suspend fun currentScopeActive(): Boolean =
        kotlin.coroutines.coroutineContext[Job]?.isActive ?: true

    companion object {
        private const val NORMAL_CLOSURE = 1000
        private const val WS_PATH = "/api/v1/ws"

        /**
         * WS uchun ALOHIDA klient: `pingInterval` bilan.
         *
         * Mobil tarmoqlarda NAT jimgina yopilib qoladi va soket "tirik" ko'rinib
         * turaveradi. 30 soniyalik ping uzilishni tez aniqlaydi — ustoz kutish
         * xonasidagi o'quvchini o'tkazib yubormaydi.
         */
        private val defaultClient: OkHttpClient by lazy {
            OkHttpClient.Builder()
                .pingInterval(30, TimeUnit.SECONDS)
                .connectTimeout(15, TimeUnit.SECONDS)
                .readTimeout(0, TimeUnit.MILLISECONDS) // WS uchun cheksiz
                .build()
        }
    }
}
