package uz.darsly.mentor.data.ws

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import okhttp3.OkHttpClient
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.TimeUnit

/**
 * Real-time kanalning qayta ulanish sikli (H1 · M1 · S1) — haqiqiy WS
 * handshake bilan (MockWebServer), kutish esa almashtirilgan.
 */
class RealtimeClientTest {

    private lateinit var server: MockWebServer
    private val sleeps = CopyOnWriteArrayList<Long>()
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private var refreshes = 0

    /** Berilsa kutish shu darvoza ochilguncha to'xtaydi — sikl qadamini ushlab turish uchun. */
    private var sleepGate: CompletableDeferred<Unit>? = null

    @Before fun setUp() { server = MockWebServer().also { it.start() } }

    @After fun tearDown() {
        scope.cancel()
        server.shutdown()
    }

    private fun client(token: () -> String? = { "jwt" }) = RealtimeClient(
        client = OkHttpClient.Builder().readTimeout(0, TimeUnit.MILLISECONDS).build(),
        baseUrl = server.url("/").toString(),
        sleep = { ms -> sleeps += ms; sleepGate?.await() ?: delay(1) },
        jitter = { 0.0 },
        accessToken = token,
        refreshToken = { refreshes++ },
    )

    /** Server: ulanishni qabul qilib, darhol [code] bilan yopadi (deploy'dagi `Stop`). */
    private fun serverCloses(code: Int, reason: String = "server shutdown") = MockResponse().withWebSocketUpgrade(
        object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: okhttp3.Response) {
                webSocket.close(code, reason)
            }
        },
    )

    private suspend fun awaitRequests(n: Int) = withTimeout(5_000) {
        while (server.requestCount < n) delay(10)
    }

    @Test
    fun `H1 — server 1000 bilan yopsa ham qayta ulanadi`() = runBlocking {
        // Backend deploy'da `CloseNormalClosure` (1000) yuboradi. Avval klient
        // buni "tugadi" deb siklni tark etardi — mentor qayta login qilgunча
        // kutish xonasi so'rovlarini ko'rmasdi.
        server.enqueue(serverCloses(1000))
        server.enqueue(serverCloses(1000))
        server.enqueue(serverCloses(1000))

        client().start(scope)
        awaitRequests(3)

        assertTrue("ikkinchi ulanishdan oldin backoff bo'lishi kerak", sleeps.isNotEmpty())
        // Ulanish OCHILGAN edi → hisob nolga tushadi → har safar bazaviy kechikish.
        assertEquals(Backoff.BASE_MS, sleeps.first())
    }

    @Test
    fun `stop dan keyin qayta ulanmaydi`() = runBlocking {
        server.enqueue(serverCloses(1000))
        server.enqueue(serverCloses(1000))
        sleepGate = CompletableDeferred()
        val c = client()
        c.start(scope)
        awaitRequests(1)
        withTimeout(5_000) { while (sleeps.isEmpty()) delay(10) } // sikl backoff'da turibdi
        c.stop()
        sleepGate?.complete(Unit) // kutish tugadi — lekin sikl allaqachon bekor
        delay(150)
        assertEquals("logoutdan keyin yangi handshake yo'q", 1, server.requestCount)
        assertEquals(RealtimeState.DISCONNECTED, c.state.value)
    }

    @Test
    fun `S1 — token header da, query da emas`() = runBlocking {
        server.enqueue(serverCloses(1000))
        client { "secret-jwt" }.start(scope)
        awaitRequests(1)

        val req = server.takeRequest()
        assertEquals("/api/v1/ws", req.path)
        assertEquals("Bearer secret-jwt", req.getHeader("Authorization"))
        assertNull(req.requestUrl?.queryParameter("token"))
    }

    @Test
    fun `M1 — 401 da ham backoff va token yangilanadi`() = runBlocking {
        // Avval 401 zero-delay issiq sikl edi: `attempt = 0; continue`.
        server.enqueue(MockResponse().setResponseCode(401))
        server.enqueue(MockResponse().setResponseCode(401))
        server.enqueue(MockResponse().setResponseCode(401))

        client().start(scope)
        awaitRequests(3)

        assertTrue("har 401 dan keyin refresh urinishi", refreshes >= 2)
        assertTrue(sleeps.size >= 2)
        // Ulanish OCHILMAGAN → hisob o'sadi → kechikish eksponensial.
        assertEquals(Backoff.delayMs(0), sleeps[0])
        assertEquals(Backoff.delayMs(1), sleeps[1])
        assertFalse("hech qachon nol kechikish", sleeps.any { it <= 0 })
    }

    @Test
    fun `tokensiz tarmoqqa chiqmaydi`() = runBlocking {
        client { null }.start(scope)
        delay(80)
        assertEquals(0, server.requestCount)
        assertTrue(sleeps.all { it == Backoff.BASE_MS })
    }
}
