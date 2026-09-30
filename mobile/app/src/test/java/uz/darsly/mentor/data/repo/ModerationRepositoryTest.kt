package uz.darsly.mentor.data.repo

import com.squareup.moshi.Moshi
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.HttpException
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.DarslyApi

/**
 * H2 — moderatsiya amallari HTTP xatosini yutmasligi kerak.
 *
 * `Response<Unit>` bilan Retrofit 4xx/5xx da ham "muvaffaqiyat" qaytaradi;
 * avval repozitoriy `isSuccessful` ni tekshirmasdi va 403 (begona dars) yoki
 * 500 ham "mute qilindi" deb ko'rsatilardi.
 */
class ModerationRepositoryTest {

    private lateinit var server: MockWebServer
    private lateinit var repo: ModerationRepository

    @Before
    fun setUp() {
        server = MockWebServer().apply { start() }
        val api = Retrofit.Builder()
            .baseUrl(server.url("/"))
            .client(OkHttpClient())
            .addConverterFactory(MoshiConverterFactory.create(Moshi.Builder().build()))
            .build()
            .create(DarslyApi::class.java)
        repo = ModerationRepository(api)
    }

    @After fun tearDown() = server.shutdown()

    private fun enqueue(code: Int, body: String = "") {
        server.enqueue(MockResponse().setResponseCode(code).setBody(body))
    }

    @Test
    fun `204 muvaffaqiyat`() {
        enqueue(204)
        assertTrue(runBlocking { repo.mute("l1", "s1") }.isSuccess)
        assertEquals("/api/v1/lessons/l1/participants/s1/mute", server.takeRequest().path)
    }

    @Test
    fun `403 xato — begona dars`() {
        enqueue(403, """{"code":"FORBIDDEN","message":"not your lesson"}""")
        val r = runBlocking { repo.mute("l1", "s1") }
        assertTrue(r.isFailure)
        val t = r.exceptionOrNull()
        assertTrue("HttpException bo'lishi kerak (ApiErrors tanish shakli)", t is HttpException)
        assertEquals(ApiErrors.uz("FORBIDDEN"), ApiErrors.humanError(t!!))
    }

    @Test
    fun `har yettala amal statusni tekshiradi`() = runBlocking {
        val calls: List<suspend () -> Result<Unit>> = listOf(
            { repo.muteAll("l1") },
            { repo.mute("l1", "s1") },
            { repo.remove("l1", "s1") },
            { repo.allowSpeak("l1", "s1") },
            { repo.revokeSpeak("l1", "s1") },
            { repo.lowerHand("l1", "s1") },
            { repo.lowerAllHands("l1") },
        )
        for ((i, call) in calls.withIndex()) {
            enqueue(500, """{"code":"INTERNAL_ERROR","message":"boom"}""")
            assertTrue("amal #$i 500 ni yutmasligi kerak", call().isFailure)
        }
        for ((i, call) in calls.withIndex()) {
            enqueue(204)
            assertTrue("amal #$i 204 da muvaffaqiyat", call().isSuccess)
        }
    }

    @Test
    fun `tarmoq xatosi ham failure`() {
        server.shutdown()
        assertTrue(runBlocking { repo.lowerAllHands("l1") }.isFailure)
    }
}
