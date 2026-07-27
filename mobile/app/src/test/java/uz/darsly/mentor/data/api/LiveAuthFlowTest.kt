package uz.darsly.mentor.data.api

import com.squareup.moshi.Moshi
import okhttp3.Interceptor
import okhttp3.OkHttpClient
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assume.assumeTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Call
import retrofit2.Callback
import retrofit2.Response
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import uz.darsly.mentor.data.store.InMemoryTokenStore
import java.util.Collections
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

/**
 * JONLI integratsiya testi — HAQIQIY test serveriga qarshi (M1 · M2 · M3).
 *
 * Odatiy `./gradlew testDebugUnitTest` da **o'tkazib yuboriladi** (JUnit `Assume`):
 * CI va offline qurish tarmoqqa bog'liq bo'lmasligi kerak. Yoqish:
 *
 * ```bash
 * DARSLY_LIVE=1 \
 * DARSLY_LIVE_EMAIL=... DARSLY_LIVE_PASSWORD=... \
 *   ./gradlew testDebugUnitTest --tests '*LiveAuthFlowTest'
 * ```
 *
 * Sirlar kodda emas — faqat muhit o'zgaruvchilaridan o'qiladi.
 */
class LiveAuthFlowTest {

    private interface LiveApi {
        @GET("api/v1/auth/me")
        fun me(): Call<Envelope<User>>

        @GET("api/v1/app-config")
        fun appConfig(): Call<Envelope<AppConfig>>

        @POST("api/v1/auth/login")
        fun login(@Body req: LoginReq): Call<Envelope<TokenPair>>

        @POST("api/v1/auth/logout")
        fun logout(@Body req: LogoutReq): Call<Void>
    }

    private val baseUrl = System.getenv("DARSLY_BASE_URL")?.takeIf { it.isNotBlank() }
        ?: "https://app.194.163.139.242.sslip.io"
    private val email: String? = System.getenv("DARSLY_LIVE_EMAIL")
    private val password: String? = System.getenv("DARSLY_LIVE_PASSWORD")

    /** Simda HAQIQATAN ketgan `/auth/refresh` so'rovlari soni. */
    private val refreshOnWire = AtomicInteger()

    private lateinit var store: InMemoryTokenStore
    private lateinit var api: LiveApi
    private lateinit var bareApi: LiveApi
    private val logoutCount = AtomicInteger()

    private val moshi: Moshi = Moshi.Builder().build()

    @Before
    fun requireLiveEnabled() {
        assumeTrue(
            "Jonli test o'chirilgan — DARSLY_LIVE=1 bilan yoqiladi",
            System.getenv("DARSLY_LIVE") == "1",
        )
        assumeTrue(
            "DARSLY_LIVE_EMAIL / DARSLY_LIVE_PASSWORD berilmagan",
            !email.isNullOrBlank() && !password.isNullOrBlank(),
        )

        val countingRefresh = Interceptor { chain ->
            if (chain.request().url.encodedPath.endsWith("/auth/refresh")) {
                refreshOnWire.incrementAndGet()
            }
            chain.proceed(chain.request())
        }

        val bare = OkHttpClient.Builder()
            .addInterceptor(countingRefresh)
            .connectTimeout(20, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .build()

        fun retrofit(client: OkHttpClient) = Retrofit.Builder()
            .baseUrl(baseUrl.trimEnd('/') + "/")
            .client(client)
            .addConverterFactory(MoshiConverterFactory.create(moshi))
            .build()

        bareApi = retrofit(bare).create(LiveApi::class.java)
        val refreshApi = retrofit(bare).create(AuthRefreshApi::class.java)

        store = InMemoryTokenStore()

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
            .addInterceptor(countingRefresh)
            .authenticator(
                TokenAuthenticator(store, refreshApi) { logoutCount.incrementAndGet(); store.clear() },
            )
            .connectTimeout(20, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .build()
        client.dispatcher.maxRequestsPerHost = 32

        api = retrofit(client).create(LiveApi::class.java)
    }

    @Test
    fun `jonli oqim login - eskirgan access - single-flight refresh - logout`() {
        // ── 1) LOGIN ────────────────────────────────────────────────────────────
        val login = bareApi.login(LoginReq(email!!, password!!)).execute()
        assertEquals("login muvaffaqiyatsiz", 200, login.code())
        val pair = login.body()?.data
        assertNotNull("login token qaytarmadi", pair)
        store.save(pair!!)

        // ── 2) HIMOYALANGAN ENDPOINT ishlaydi ───────────────────────────────────
        val me = api.me().execute()
        assertEquals(200, me.code())
        assertEquals(0, refreshOnWire.get())

        // ── 3) ACCESS TOKENNI "eskirgan" qilamiz ────────────────────────────────
        // (15 daqiqa kutmaslik uchun ataylab yaroqsiz qilinadi — server 401 beradi,
        //  bu bizga kerak bo'lgan aynan o'sha holat.)
        val realRefresh = store.read()!!.refreshToken
        store.save(TokenPair(accessToken = "eskirgan.access.token", refreshToken = realRefresh))

        // ── 4) 5 ta PARALLEL so'rov → serverga BITTA refresh ketishi kerak ──────
        val n = 5
        val latch = CountDownLatch(n)
        val codes = Collections.synchronizedList(mutableListOf<Int>())
        repeat(n) {
            api.me().enqueue(object : Callback<Envelope<User>> {
                override fun onResponse(call: Call<Envelope<User>>, response: Response<Envelope<User>>) {
                    codes += response.code(); response.errorBody()?.close(); latch.countDown()
                }

                override fun onFailure(call: Call<Envelope<User>>, t: Throwable) {
                    codes += -1; latch.countDown()
                }
            })
        }
        assertTrue("jonli so'rovlar 60s ichida tugamadi", latch.await(60, TimeUnit.SECONDS))

        assertEquals("hamma so'rov jim yangilangandan keyin muvaffaqiyatli bo'lsin", List(n) { 200 }, codes)
        assertEquals("SINGLE-FLIGHT: jonli serverga faqat 1 ta refresh ketishi kerak", 1, refreshOnWire.get())
        assertEquals(0, logoutCount.get())
        assertTrue("token yangilangan bo'lishi kerak", store.read()!!.accessToken != "eskirgan.access.token")

        // ── 5) app-config (M42) ochiq va o'qiladi ───────────────────────────────
        val cfg = bareApi.appConfig().execute()
        assertEquals(200, cfg.code())
        assertNotNull("app-config `android` bloki yo'q", cfg.body()?.data?.android)

        // ── 6) LOGOUT (M3) ──────────────────────────────────────────────────────
        val current = store.read()!!
        val out = api.logout(LogoutReq(current.refreshToken)).execute()
        assertEquals("logout 204 qaytarishi kerak", 204, out.code())
        store.clear()

        // ── 7) Logoutdan keyin refresh o'lgan bo'lishi kerak → toza logout ──────
        store.save(TokenPair("yana.eskirgan.access", current.refreshToken))
        val refreshBefore = refreshOnWire.get()
        val after = api.me().execute()
        after.errorBody()?.close()

        assertEquals("sessiya o'lgach 401 qaytishi kerak", 401, after.code())
        assertEquals("faqat BIR marta refresh urinilsin", refreshBefore + 1, refreshOnWire.get())
        assertEquals("toza logout aynan bir marta", 1, logoutCount.get())
        assertNull("tokenlar tozalanishi shart", store.read())
    }
}
