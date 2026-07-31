package uz.darsly.mentor.data.api

import com.squareup.moshi.Moshi
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import okhttp3.mockwebserver.SocketPolicy
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Call
import retrofit2.Callback
import retrofit2.Response
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import retrofit2.http.GET
import retrofit2.http.POST
import uz.darsly.mentor.data.store.InMemoryTokenStore
import java.io.IOException
import java.util.Collections
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

/**
 * M2 — jim token yangilash, **single-flight**.
 *
 * BU BLOKNING ENG MUHIM TESTI: bir vaqtda kelgan N ta 401 uchun serverga
 * ATIGI BITTA `POST /auth/refresh` ketishi shart. Backend'da 60 soniyalik grace
 * oynasi bor, lekin u klient poygasini oqlamaydi — poyga bu yerda prinsipial
 * ravishda yo'q qilinadi.
 *
 * Test haqiqiy HTTP orqali ishlaydi (MockWebServer), ya'ni so'rovlar **sanaladi**,
 * mock obyekt xulqi emas.
 */
class TokenAuthenticatorTest {

    private companion object {
        const val OLD_ACCESS = "access-eskirgan"
        const val OLD_REFRESH = "refresh-eski"
        const val NEW_ACCESS = "access-yangi"
        const val NEW_REFRESH = "refresh-yangi"

        /** Parallel so'rovlar soni. */
        const val N = 8

        /**
         * Refresh javobini ataylab sekinlashtiramiz — single-flight bo'lmasa
         * qolgan 7 ta oqim ham refresh yuborishga ULGURADI (test vakuum emas).
         */
        const val REFRESH_DELAY_MS = 200L

        const val EXPIRED_BODY = """{"code":"TOKEN_EXPIRED","message":"token expired"}"""
        const val INVALID_BODY = """{"code":"TOKEN_INVALID","message":"invalid refresh token"}"""
        const val REVOKED_BODY = """{"code":"SESSION_REVOKED","message":"session revoked"}"""
        const val ME_BODY =
            """{"data":{"id":"u1","email":"a@darsly.uz","full_name":"Ali","role":"mentor"}}"""
    }

    /** Testda `Call` qulayroq — parallel `enqueue` uchun. */
    private interface TestApi {
        @GET("api/v1/auth/me")
        fun me(): Call<Envelope<User>>

        @POST("api/v1/auth/login")
        fun login(): Call<Envelope<TokenPair>>
    }

    private lateinit var server: MockWebServer
    private lateinit var api: TestApi
    private lateinit var store: InMemoryTokenStore

    private val refreshCount = AtomicInteger()
    private val protectedCount = AtomicInteger()
    private val logoutCount = AtomicInteger()

    /** Oxirgi chiqarish sababi — login ekrani AYNAN shuni ko'rsatadi. */
    @Volatile private var lastLogoutReason: LogoutReason = LogoutReason.UNKNOWN

    /** Himoyalangan endpointning 401 tanasi (holatga qarab almashadi). */
    @Volatile private var unauthorizedBody = EXPIRED_BODY

    /** Refresh javobini test o'zgartiradi (muvaffaqiyat / 401 / 500). */
    @Volatile private var refreshResponse: () -> MockResponse = {
        MockResponse().setResponseCode(200)
            .setBody("""{"data":{"access_token":"$NEW_ACCESS","refresh_token":"$NEW_REFRESH"}}""")
    }

    /** `true` bo'lsa himoyalangan endpoint yangi token bilan ham 401 beradi. */
    @Volatile private var alwaysUnauthorized = false

    @Before
    fun setUp() {
        server = MockWebServer()
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                val path = request.path.orEmpty()
                return when {
                    path.endsWith("/auth/refresh") -> {
                        refreshCount.incrementAndGet()
                        Thread.sleep(REFRESH_DELAY_MS)
                        refreshResponse()
                    }

                    path.endsWith("/auth/login") -> MockResponse()
                        .setResponseCode(401)
                        .setBody("""{"code":"UNAUTHORIZED","message":"invalid credentials"}""")

                    else -> {
                        protectedCount.incrementAndGet()
                        val auth = request.getHeader("Authorization")
                        if (!alwaysUnauthorized && auth == "Bearer $NEW_ACCESS") {
                            MockResponse().setResponseCode(200).setBody(ME_BODY)
                        } else {
                            MockResponse().setResponseCode(401).setBody(unauthorizedBody)
                        }
                    }
                }
            }
        }
        server.start()

        val moshi = Moshi.Builder().build()
        val base = server.url("/").toString()

        // Refresh klienti — authenticator'siz (rekursiya bo'lmasin).
        val refreshApi = Retrofit.Builder()
            .baseUrl(base)
            .client(OkHttpClient())
            .addConverterFactory(MoshiConverterFactory.create(moshi))
            .build()
            .create(AuthRefreshApi::class.java)

        store = InMemoryTokenStore(TokenPair(OLD_ACCESS, OLD_REFRESH))

        val client = OkHttpClient.Builder()
            .addInterceptor { chain ->
                val token = store.read()?.accessToken
                val req = if (token.isNullOrBlank()) {
                    chain.request()
                } else {
                    chain.request().newBuilder().header("Authorization", "Bearer $token").build()
                }
                chain.proceed(req)
            }
            .authenticator(
                TokenAuthenticator(
                    store = store,
                    refreshApi = refreshApi,
                    onHardLogout = { reason ->
                        logoutCount.incrementAndGet()
                        lastLogoutReason = reason
                        store.clear()
                    },
                ),
            )
            .build()
        // Standart limit 5 — N=8 ta so'rov HAQIQATAN parallel ketsin.
        client.dispatcher.maxRequests = 64
        client.dispatcher.maxRequestsPerHost = 64

        api = Retrofit.Builder()
            .baseUrl(base)
            .client(client)
            .addConverterFactory(MoshiConverterFactory.create(moshi))
            .build()
            .create(TestApi::class.java)
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    /** N ta parallel `me()` chaqiradi va HTTP kodlarini qaytaradi. */
    private fun runParallel(count: Int = N): List<Int> {
        val latch = CountDownLatch(count)
        val codes = Collections.synchronizedList(mutableListOf<Int>())
        repeat(count) {
            api.me().enqueue(object : Callback<Envelope<User>> {
                override fun onResponse(call: Call<Envelope<User>>, response: Response<Envelope<User>>) {
                    codes += response.code()
                    response.errorBody()?.close()
                    latch.countDown()
                }

                override fun onFailure(call: Call<Envelope<User>>, t: Throwable) {
                    codes += -1
                    latch.countDown()
                }
            })
        }
        assertTrue("so'rovlar 30s ichida tugamadi", latch.await(30, TimeUnit.SECONDS))
        return codes
    }

    // ─────────────────────────────────────────────────────────────────────────

    @Test
    fun `N ta parallel 401 uchun serverga ATIGI BITTA refresh ketadi`() {
        val codes = runParallel()

        assertEquals("serverga bittadan ortiq refresh ketdi", 1, refreshCount.get())
        assertEquals("hamma so'rov muvaffaqiyatli qayta urilishi kerak", List(N) { 200 }, codes)
        // Har so'rov: 1 ta 401 + 1 ta muvaffaqiyatli qayta urinish.
        assertEquals(2 * N, protectedCount.get())
        assertEquals(0, logoutCount.get())
        assertEquals(TokenPair(NEW_ACCESS, NEW_REFRESH), store.read())
    }

    @Test
    fun `yangilangandan keyin keyingi 401 yana refresh yubormaydi`() {
        runParallel()
        assertEquals(1, refreshCount.get())

        // Ikkinchi to'lqin: token allaqachon yangi, 401 umuman bo'lmaydi.
        val codes = runParallel()
        assertEquals(List(N) { 200 }, codes)
        assertEquals("token yangi bo'lsa refresh kerak emas", 1, refreshCount.get())
    }

    @Test
    fun `refresh muvaffaqiyatsiz bolsa toza logout va sikl yoq`() {
        refreshResponse = { MockResponse().setResponseCode(401).setBody(INVALID_BODY) }

        val codes = runParallel()

        assertEquals("faqat bitta refresh urinishi bo'lishi kerak", 1, refreshCount.get())
        assertEquals("logout aynan bir marta chaqirilsin", 1, logoutCount.get())
        assertNull("tokenlar tozalanishi shart", store.read())
        assertEquals("401 chaqiruvchiga qaytarilsin", List(N) { 401 }, codes)
        // Qayta urinish YO'Q — aks holda sonlar 2*N bo'lardi (sikl belgisi).
        assertEquals(N, protectedCount.get())
    }

    @Test
    fun `logoutdan keyin yangi sorov refresh yubormaydi`() {
        refreshResponse = { MockResponse().setResponseCode(401).setBody(INVALID_BODY) }
        runParallel(1)
        assertEquals(1, refreshCount.get())
        assertEquals(1, logoutCount.get())

        // Sessiya o'ldi → token yo'q → refresh urinishi ham yo'q (cheksiz sikl himoyasi).
        val codes = runParallel(1)
        assertEquals(listOf(401), codes)
        assertEquals(1, refreshCount.get())
        assertEquals(1, logoutCount.get())
    }

    @Test
    fun `token yangilangan bolsa ham 401 davom etsa sikl boshlanmaydi`() {
        alwaysUnauthorized = true

        val codes = runParallel(1)

        assertEquals(listOf(401), codes)
        assertEquals("faqat bir marta yangilanadi", 1, refreshCount.get())
        // 1 ta boshlang'ich + 1 ta qayta urinish, xolos.
        assertEquals(2, protectedCount.get())
    }

    @Test
    fun `server xatosi 500 sessiyani oldirmaydi`() {
        refreshResponse = { MockResponse().setResponseCode(500).setBody("""{"code":"INTERNAL_ERROR"}""") }

        val codes = runParallel(1)

        assertEquals(listOf(401), codes)
        assertEquals(1, refreshCount.get())
        assertEquals("5xx da logout qilinmasin — server tiklanishi mumkin", 0, logoutCount.get())
        assertNotNull(store.read())
        assertEquals(OLD_ACCESS, store.read()!!.accessToken)
    }

    @Test
    fun `refreshda tarmoq uzilsa tarmoq xatosi qaytadi sessiya saqlanadi`() {
        // QA 🟢8: avval bu holatda asl 401 chaqiruvchiga qaytardi va UI
        // "Sessiya tugadi — qaytadan kiring" deb yozardi, holbuki foydalanuvchi
        // chiqarilmagan — aybdor tarmoq.
        refreshResponse = {
            MockResponse().apply { socketPolicy = SocketPolicy.DISCONNECT_AT_START }
        }

        val thrown = try {
            api.me().execute()
            null
        } catch (e: IOException) {
            e
        }

        assertNotNull("tarmoq uzilishi IOException bo'lib chiqishi kerak", thrown)
        assertEquals(
            "UI tarmoq xatosi matnini ko'rsatsin",
            ApiErrors.NETWORK,
            ApiErrors.humanError(thrown!!),
        )
        assertEquals("tarmoq xatosida logout QILINMASIN", 0, logoutCount.get())
        assertNotNull("sessiya saqlanishi shart", store.read())
        assertEquals(OLD_ACCESS, store.read()!!.accessToken)
    }

    @Test
    fun `login 401 refreshni qozgatmaydi`() {
        val resp = api.login().execute()
        assertEquals(401, resp.code())
        resp.errorBody()?.close()
        assertEquals("auth endpointlari chetlab o'tilishi kerak", 0, refreshCount.get())
        assertEquals(0, logoutCount.get())
    }

    @Test
    fun `tokensiz sorov uchun refresh yuborilmaydi`() {
        store.clear()
        val codes = runParallel(1)
        assertEquals(listOf(401), codes)
        assertEquals(0, refreshCount.get())
        assertEquals(0, logoutCount.get())
    }

    // ─── SESSION_REVOKED (bitta akkaunt = bitta sessiya · PRODUCT.md №1) ─────

    @Test
    fun `SESSION_REVOKED da refresh UMUMAN yuborilmaydi`() {
        // ⭐ Boshqa qurilmadan kirilgan: sessiya SERVERDA tugatilgan va refresh
        // ham aynan shu kod bilan qaytadi (grace oynasi tugatilgan sessiyani
        // tiriltirmaydi). Ya'ni refresh so'rovi — bekorga ketgan aylanish
        // vaqti va bekorga sarflangan trafik.
        unauthorizedBody = REVOKED_BODY

        val codes = runParallel(1)

        assertEquals(listOf(401), codes)
        assertEquals("refresh yuborilmasligi kerak", 0, refreshCount.get())
        assertEquals(1, logoutCount.get())
        assertNull("tokenlar tozalanishi shart", store.read())
        // Qayta urinish YO'Q — bitta so'rov, xolos.
        assertEquals(1, protectedCount.get())
    }

    @Test
    fun `SESSION_REVOKED sababi login ekraniga yetkaziladi`() {
        unauthorizedBody = REVOKED_BODY
        runParallel(1)
        // "Sessiya tugadi" EMAS: ustoz hisobi ulashilganini bilishi kerak.
        assertEquals(LogoutReason.REVOKED, lastLogoutReason)
    }

    @Test
    fun `refresh SESSION_REVOKED bersa ham sabab uzatiladi`() {
        // Boshqa yo'l: access hali `TOKEN_EXPIRED` beradi (odatiy eskirish),
        // lekin refresh urinishida sessiya allaqachon tugatilgan bo'lib chiqadi.
        refreshResponse = { MockResponse().setResponseCode(401).setBody(REVOKED_BODY) }

        runParallel(1)

        assertEquals(1, refreshCount.get())
        assertEquals(1, logoutCount.get())
        assertEquals(LogoutReason.REVOKED, lastLogoutReason)
    }

    @Test
    fun `oddiy eskirishda sabab muddat tugashi deb qoladi`() {
        // `TOKEN_INVALID` — "boshqa qurilma" EMAS; noto'g'ri sabab ko'rsatish
        // ustozni yo'q muammoni qidirishga majburlardi.
        refreshResponse = { MockResponse().setResponseCode(401).setBody(INVALID_BODY) }

        runParallel(1)

        assertEquals(1, logoutCount.get())
        assertEquals(LogoutReason.EXPIRED, lastLogoutReason)
    }

    @Test
    fun `SESSION_REVOKED tekshiruvi javob tanasini buzmaydi`() {
        // `peekBody` ishlatilgani MUHIM: `response.body` bir marta o'qilsa
        // chaqiruvchiga bo'sh tana yetib borardi va UI xato matnini
        // ko'rsata olmasdi.
        unauthorizedBody = REVOKED_BODY

        val resp = api.me().execute()

        assertEquals(401, resp.code())
        val body = resp.errorBody()?.string()
        assertNotNull(body)
        assertEquals(
            "chaqiruvchi ham kodni ko'rishi kerak",
            LogoutReason.REVOKED,
            LogoutReason.ofBody(body),
        )
    }
}
