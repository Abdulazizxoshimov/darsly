package uz.darsly.mentor.data.repo

import com.squareup.moshi.Moshi
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import uz.darsly.mentor.data.api.CreatePollReq
import uz.darsly.mentor.data.api.DarslyApi

/**
 * Chat moderatsiyasi (№6), fayl ulashish (№15) va so'rovnoma (№7) —
 * **haqiqiy HTTP** bilan (MockWebServer), mock obyektsiz.
 *
 * Bu qatlamdagi xatolarning aksariyati yo'l/metod/status kodi bilan bog'liq
 * (`/chat/upload` statik segmenti `/chat/:messageID` dan OLDIN turishi kerak,
 * 204 tanasiz keladi, 404 esa xato emas) — mock interfeys bularning hech
 * birini ushlamasdi.
 */
class ChatPollRepositoryTest {

    private lateinit var server: MockWebServer
    private lateinit var api: DarslyApi

    @Before
    fun setUp() {
        server = MockWebServer().apply { start() }
        api = Retrofit.Builder()
            .baseUrl(server.url("/"))
            .client(OkHttpClient())
            .addConverterFactory(MoshiConverterFactory.create(Moshi.Builder().build()))
            .build()
            .create(DarslyApi::class.java)
    }

    @After
    fun tearDown() = server.shutdown()

    private fun enqueue(code: Int, body: String = "") {
        server.enqueue(MockResponse().setResponseCode(code).setBody(body))
    }

    private fun chatRepo() = RoomChatRepository(api, attachments = FakeAttachments())

    // ─── Chat: xabarni o'chirish ─────────────────────────────────────────────

    @Test
    fun `xabar ochirish DELETE yuboradi`() {
        enqueue(204)

        val result = runBlocking { chatRepo().delete("l1", "m1") }

        assertTrue(result.isSuccess)
        val request = server.takeRequest()
        assertEquals("DELETE", request.method)
        assertEquals("/api/v1/lessons/l1/chat/m1", request.path)
    }

    @Test
    fun `404 xato emas — xabar allaqachon ochirilgan`() {
        // ⭐ Server atomik ishlaydi (`WHERE deleted_at IS NULL`): ikkinchi
        // o'chirish 404 oladi. Bu poyga natijasi (ikki marta bosildi yoki ikki
        // qurilmadan) va foydalanuvchi nuqtai nazaridan ish BAJARILGAN —
        // "topilmadi" deb ko'rsatish "o'chmadi" degan yolg'on bo'lardi.
        enqueue(404, """{"code":"NOT_FOUND","message":"message not found"}""")
        assertTrue(runBlocking { chatRepo().delete("l1", "m1") }.isSuccess)
    }

    @Test
    fun `403 yutilmaydi — begona darsning xabari`() {
        // 404 ni yutish qoidasi FAQAT 404 ga tegishli: 403 jim o'tsa ustoz
        // "o'chirdim" deb o'ylab qolardi, xabar esa xonada turaverardi.
        enqueue(403, """{"code":"FORBIDDEN","message":"not your lesson"}""")
        assertTrue(runBlocking { chatRepo().delete("l1", "m1") }.isFailure)
    }

    // ─── Chat: fayl ─────────────────────────────────────────────────────────

    @Test
    fun `fayl bilan kelgan xabar tarixdan oqiladi`() {
        enqueue(
            200,
            """{"data":[{"id":"m1","sender_identity":"g1","sender_name":"Ali","body":"Uy ishi",""" +
                """"file":{"name":"uy_ishi.pdf","size":184320,"mime":"application/pdf",""" +
                """"url":"https://minio/x?sig=1","expires_in_s":3600}}]}""",
        )

        val items = runBlocking { chatRepo().history("l1", "room-token") }.getOrThrow()

        assertEquals("uy_ishi.pdf", items.first().file?.name)
        assertEquals(184_320L, items.first().file?.size)
        assertEquals(3600, items.first().file?.expiresInS)
    }

    @Test
    fun `faylsiz xabarda file maydoni null`() {
        enqueue(200, """{"data":[{"id":"m2","sender_name":"Ali","body":"Salom"}]}""")
        val items = runBlocking { chatRepo().history("l1", "t") }.getOrThrow()
        assertNull(items.first().file)
    }

    // ─── So'rovnoma ─────────────────────────────────────────────────────────

    @Test
    fun `sorovnomalar sahifalanmagan konvertdan oqiladi`() {
        // ⚠️ Backend `hs.Success(items)` ishlatadi — `total` YO'Q. Bu yerda
        // `ListEnvelope` kutilsa ro'yxat jimgina bo'sh chiqardi.
        enqueue(
            200,
            """{"data":[{"id":"p1","lesson_id":"l1","question":"2+2?","options":["3","4"],""" +
                """"is_active":true,"results_visibility":"public"}]}""",
        )

        val items = runBlocking { PollsRepository(api).list("l1") }.getOrThrow()

        assertEquals(1, items.size)
        assertEquals(listOf("3", "4"), items.first().options)
        assertEquals("public", items.first().resultsVisibility)
        assertEquals("/api/v1/lessons/l1/polls", server.takeRequest().path)
    }

    @Test
    fun `yaratishda results_visibility JSONga tushadi`() {
        // Maydon yuborilmasa server `mentor_only` qo'yadi — ya'ni "hammaga
        // ko'rinsin" tanlovi JIMGINA yo'qolardi va ustoz keyin natijani e'lon
        // qila olmasdi (rejim o'zgarmas).
        enqueue(201, """{"data":{"id":"p1","question":"2+2?","options":["3","4"]}}""")

        runBlocking {
            PollsRepository(api).create("l1", CreatePollReq("2+2?", listOf("3", "4"), "public"))
        }

        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/lessons/l1/polls", request.path)
        assertTrue(request.body.readUtf8().contains(""""results_visibility":"public""""))
    }

    @Test
    fun `elon qilish natijani qaytaradi`() {
        enqueue(
            200,
            """{"data":{"poll":{"id":"p1","question":"Q","options":["A","B"],""" +
                """"results_visibility":"public","results_published_at":"2026-07-30T10:00:00Z"},""" +
                """"counts":[3,7],"total":10}}""",
        )

        val res = runBlocking { PollsRepository(api).publish("l1", "p1") }.getOrThrow()

        assertEquals(listOf(3, 7), res.counts)
        assertEquals(10, res.total)
        assertEquals("2026-07-30T10:00:00Z", res.poll?.resultsPublishedAt)
        assertEquals("/api/v1/lessons/l1/polls/p1/publish", server.takeRequest().path)
    }

    @Test
    fun `mentor_only sorovnomani elon qilish 400 beradi`() {
        // Server qoidasi: rejim yaratishda tanlangan va o'zgarmas. UI tugmani
        // ko'rsatmaydi, lekin xato baribir yutilmasligi kerak.
        enqueue(400, """{"code":"BAD_REQUEST","message":"poll results are mentor-only"}""")
        assertTrue(runBlocking { PollsRepository(api).publish("l1", "p1") }.isFailure)
    }

    @Test
    fun `yopish alohida yolda`() {
        enqueue(200, """{"data":{"poll":{"id":"p1","is_active":false},"counts":[1],"total":1}}""")

        val res = runBlocking { PollsRepository(api).close("p1") }.getOrThrow()

        assertEquals(false, res.poll?.isActive)
        // `/lessons/:id/polls/...` EMAS — bu boshqa yo'l (`protected` guruhda).
        assertEquals("/api/v1/polls/p1/close", server.takeRequest().path)
    }

    @Test
    fun `natija room-token bilan soraladi`() {
        // JWT emas: endpoint ochiq va "xonada bo'lganlik isboti"ni LiveKit
        // tokeni orqali talab qiladi. Tokensiz so'rov 401 olardi.
        enqueue(200, """{"data":{"poll":{"id":"p1"},"counts":[2,4],"total":6}}""")

        val res = runBlocking { PollsRepository(api).results("p1", "lk-token") }.getOrThrow()

        assertEquals(6, res.total)
        assertEquals("/api/v1/polls/p1/results?token=lk-token", server.takeRequest().path)
    }

    @Test
    fun `bosh konvert xato sifatida qaytadi`() {
        // Server 200 berib `data` ni bermasa, `null` bilan davom etish
        // `NullPointerException` bo'lardi.
        enqueue(200, """{}""")
        assertTrue(runBlocking { PollsRepository(api).close("p1") }.isFailure)
    }
}

/**
 * Fayl o'qishning JVM'dagi o'rinbosari.
 *
 * URI o'qish Android `ContentResolver` iga bog'langan va bu testlarda
 * sinalmaydi — bu yerda repozitoriyning HTTP tomoni tekshiriladi. Chegara
 * qoidalari esa sof [uz.darsly.mentor.util.ChatUpload] da va `ChatUploadTest`
 * bilan alohida qoplangan.
 */
private class FakeAttachments(
    private val result: Result<ChatAttachment> = Result.success(
        ChatAttachment("uy_ishi.pdf", 4, byteArrayOf(1, 2, 3, 4)),
    ),
) : ChatAttachments {
    override fun read(uri: android.net.Uri): Result<ChatAttachment> = result
}
