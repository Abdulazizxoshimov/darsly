package uz.darsly.mentor.data.ws

import io.livekit.android.util.LKLog
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
import uz.darsly.mentor.data.api.Net
import uz.darsly.mentor.data.api.Session
import java.net.URLEncoder
import java.util.concurrent.TimeUnit

/** Real-time kanal holati — UI "jonli/uzilgan" ko'rsatishi uchun. */
enum class RealtimeState { DISCONNECTED, CONNECTING, CONNECTED }

/**
 * Serverning real-time kanali (`GET /api/v1/ws?token=`).
 *
 * ## Nega kerak
 * Kutish xonasidagi o'quvchi **3 soniyada** ustozga ko'rinishi kerak (mezon D-1) —
 * so'rovni takroriy `GET` bilan tortish (polling) na tezkor, na batareyaga do'st.
 * Bildirishnomalar va dars holati ham shu kanaldan keladi.
 *
 * ## Protokol
 * Web klienti bilan **bir xil**: token **query** da (`?token=`), konvert
 * `{type, room, payload, created_at}`, qayta ulanish eksponensial ([Backoff]).
 * Token query'da bo'lishi backend talabi — brauzer WS handshake'da header yubora olmaydi
 * (`CLAUDE.md` · BE-1), mobil ham xuddi shu yo'ldan boradi.
 *
 * ## Token eskirishi
 * Handshake **401** bersa, token eskirgan. Bu holda oddiy authed so'rov yuboriladi —
 * `Net` dagi [uz.darsly.mentor.data.api.TokenAuthenticator] uni ushlab **single-flight
 * refresh** qiladi — va yangi token bilan qayta ulanamiz. Ya'ni refresh mantiqi bitta
 * joyda qoladi, bu yerda takrorlanmaydi.
 */
class RealtimeClient(
    private val client: OkHttpClient = defaultClient,
    private val parser: RealtimeParser = RealtimeParser(),
    private val baseUrl: String = BuildConfig.API_BASE_URL,
    /** Test uchun: haqiqiy kutishni almashtirish. */
    private val sleep: suspend (Long) -> Unit = { delay(it) },
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
            val token = Session.accessToken
            if (token.isNullOrBlank()) {
                // Hali login bo'lmagan — qisqa kutib qayta ko'ramiz.
                sleep(Backoff.BASE_MS)
                continue
            }

            _state.value = RealtimeState.CONNECTING
            val closeReason = connectAndWait(token)
            _state.value = RealtimeState.DISCONNECTED

            if (closeReason == CloseReason.UNAUTHORIZED) {
                // Token eskirgan — `Net` ning authenticator'i orqali yangilaymiz.
                runCatching { Net.api.me() }
                    .onFailure { LKLog.w(it) { "realtime: token yangilanmadi" } }
                attempt = 0
                continue
            }
            if (closeReason == CloseReason.NORMAL) return

            val wait = Backoff.delayMs(attempt)
            LKLog.i { "realtime: qayta ulanish ${wait}ms dan keyin (urinish=$attempt)" }
            sleep(wait)
            attempt++
        }
    }

    private enum class CloseReason { NORMAL, ERROR, UNAUTHORIZED }

    /** Bitta ulanish: ochadi, hodisalarni uzatadi va yopilgunicha kutadi. */
    private suspend fun connectAndWait(token: String): CloseReason {
        val url = baseUrl.trimEnd('/')
            .replaceFirst("http", "ws") + "/api/v1/ws?token=" + URLEncoder.encode(token, "UTF-8")

        var reason = CloseReason.ERROR
        val done = kotlinx.coroutines.CompletableDeferred<CloseReason>()

        val ws = client.newWebSocket(
            Request.Builder().url(url).build(),
            object : WebSocketListener() {
                override fun onOpen(webSocket: WebSocket, response: Response) {
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
                    webSocket.close(NORMAL_CLOSURE, null)
                    done.complete(if (code == NORMAL_CLOSURE) CloseReason.NORMAL else CloseReason.ERROR)
                }

                override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                    val code = response?.code
                    reason = if (code == 401 || code == 403) CloseReason.UNAUTHORIZED else CloseReason.ERROR
                    LKLog.w(t) { "realtime: uzildi (kod=$code)" }
                    done.complete(reason)
                }
            },
        )
        socket = ws
        val result = done.await()
        socket = null
        return result
    }

    /** Korutina hali tirikmi (bekor qilinmaganmi). */
    private suspend fun currentScopeActive(): Boolean =
        kotlin.coroutines.coroutineContext[Job]?.isActive ?: true

    companion object {
        private const val NORMAL_CLOSURE = 1000

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
