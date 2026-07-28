package uz.darsly.mentor.data.repo

import com.squareup.moshi.Moshi
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.UpdateLessonReq
import uz.darsly.mentor.data.store.CachedLessons
import uz.darsly.mentor.data.store.LessonsCache
import uz.darsly.mentor.data.api.Lesson

/**
 * Yangi repozitoriylar — **haqiqiy HTTP** bilan (MockWebServer), mock obyektsiz.
 *
 * Nega shunday: bu qatlamdagi xatolarning aksariyati JSON shakli va HTTP
 * status kodlari bilan bog'liq (bo'sh konvert, 204, 409). Mock interfeys
 * bularning hech birini ushlamasdi — u faqat bizning taxminimizni takrorlardi.
 */
class NewRepositoriesTest {

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

    /** Xotiradagi kesh — yozuvlar sonini ham sanaydi. */
    private class RecordingCache(var value: CachedLessons? = null) : LessonsCache {
        var writes = 0
            private set

        override fun read(): CachedLessons? = value
        override fun write(lessons: List<Lesson>, savedAtMillis: Long) {
            writes++
            value = CachedLessons(lessons, savedAtMillis)
        }

        override fun clear() {
            value = null
        }
    }

    // ─── Darslar: tahrirlash va o'chirish ─────────────────────────────────────

    @Test
    fun `tahrirlash keshdagi nusxani almashtiradi`() {
        val cache = RecordingCache(
            CachedLessons(
                listOf(
                    Lesson(id = "l1", title = "Eski nom"),
                    Lesson(id = "l2", title = "Boshqa dars"),
                ),
                savedAtMillis = 1L,
            ),
        )
        val repo = LessonsRepository(api, cache) { 42L }
        enqueue(200, """{"data":{"id":"l1","title":"Yangi nom","status":"scheduled"}}""")

        val result = runBlocking { repo.update("l1", UpdateLessonReq(title = "Yangi nom")) }

        assertTrue(result.isSuccess)
        assertEquals("Yangi nom", result.getOrNull()?.title)
        // Kesh o'rnida yangilandi — boshqa dars joyida qoldi.
        val cached = cache.read()!!.lessons
        assertEquals(listOf("Yangi nom", "Boshqa dars"), cached.map { it.title })
        assertEquals(42L, cache.read()!!.savedAtMillis)
    }

    @Test
    fun `tahrirlash PATCH metodi bilan yuboriladi`() {
        val repo = LessonsRepository(api, RecordingCache())
        enqueue(200, """{"data":{"id":"l1","title":"X","status":"scheduled"}}""")

        runBlocking { repo.update("l1", UpdateLessonReq(title = "X")) }

        val request = server.takeRequest()
        // PUT bo'lsa server butun obyektni kutardi va tegilmagan maydonlar
        // nolga tushardi.
        assertEquals("PATCH", request.method)
        assertEquals("/api/v1/lessons/l1", request.path)
    }

    @Test
    fun `tahrirlash xatosida kesh tegilmaydi`() {
        val cache = RecordingCache(CachedLessons(listOf(Lesson(id = "l1", title = "Eski")), 1L))
        val repo = LessonsRepository(api, cache)
        enqueue(403, """{"code":"FORBIDDEN","message":"not yours"}""")

        val result = runBlocking { repo.update("l1", UpdateLessonReq(title = "Yangi")) }

        assertTrue(result.isFailure)
        assertEquals(0, cache.writes)
        assertEquals("Eski", cache.read()!!.lessons.first().title)
    }

    @Test
    fun `ochirish keshdan olib tashlaydi`() {
        val cache = RecordingCache(
            CachedLessons(listOf(Lesson(id = "l1", title = "A"), Lesson(id = "l2", title = "B")), 1L),
        )
        val repo = LessonsRepository(api, cache)
        enqueue(204)

        val result = runBlocking { repo.delete("l1") }

        assertTrue(result.isSuccess)
        assertEquals(listOf("l2"), cache.read()!!.lessons.map { it.id })
        assertEquals("DELETE", server.takeRequest().method)
    }

    @Test
    fun `ochirish xatosida dars keshda qoladi`() {
        // ⭐ Optimistik o'chirish ATAYLAB ishlatilmagan: so'rov rad etilsa dars
        // ekrandan yo'qolib, serverda tirik qolardi — ustoz uni o'chdi deb
        // o'ylardi va havola ishlashda davom etardi.
        val cache = RecordingCache(CachedLessons(listOf(Lesson(id = "l1", title = "A")), 1L))
        val repo = LessonsRepository(api, cache)
        enqueue(403, """{"code":"FORBIDDEN","message":"nope"}""")

        assertTrue(runBlocking { repo.delete("l1") }.isFailure)
        assertEquals(0, cache.writes)
        assertEquals(1, cache.read()!!.lessons.size)
    }

    // ─── Bildirishnomalar ─────────────────────────────────────────────────────

    @Test
    fun `bildirishnomalar sahifalanadi`() {
        val repo = NotificationsRepository(api)
        enqueue(200, """{"data":[{"id":"n1","title":"A"},{"id":"n2","title":"B"}],"total":3,"page":1,"limit":2}""")
        enqueue(200, """{"data":[{"id":"n3","title":"C"}],"total":3,"page":2,"limit":2}""")

        val page = runBlocking { repo.refresh(pageSize = 2) }.getOrThrow()

        assertEquals(listOf("n1", "n2", "n3"), page.items.map { it.id })
        assertEquals(3, page.total)
        assertFalse(page.truncated)
        assertEquals(2, server.requestCount)
    }

    @Test
    fun `bosh sahifa cheksiz siklga olib kelmaydi`() {
        // Server `total` ni noto'g'ri (haqiqatdan katta) qaytarsa ham to'xtaymiz.
        val repo = NotificationsRepository(api)
        enqueue(200, """{"data":[{"id":"n1"}],"total":99,"page":1,"limit":50}""")
        enqueue(200, """{"data":[],"total":99,"page":2,"limit":50}""")

        val page = runBlocking { repo.refresh() }.getOrThrow()

        assertEquals(1, page.items.size)
        assertEquals(2, server.requestCount)
    }

    @Test
    fun `oqilmaganlar soni bosh konvertda nolga tushadi`() {
        val repo = NotificationsRepository(api)
        enqueue(200, """{"data":{"count":7}}""")
        assertEquals(7, runBlocking { repo.unreadCount() }.getOrThrow())

        // `data` bo'lmasa ilova yiqilmasligi kerak: bu qiymat faqat nishon uchun.
        enqueue(200, """{}""")
        assertEquals(0, runBlocking { repo.unreadCount() }.getOrThrow())
    }

    @Test
    fun `oqilgan deb belgilash POST yuboradi`() {
        val repo = NotificationsRepository(api)
        enqueue(204)
        assertTrue(runBlocking { repo.markRead("n1") }.isSuccess)
        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/notifications/n1/read", request.path)
    }

    // ─── Yozuvlar ─────────────────────────────────────────────────────────────

    @Test
    fun `yozuvlar royxati sahifalanmagan konvertdan oqiladi`() {
        // ⚠️ Backend `hs.Success(items)` ishlatadi — `total` YO'Q. Bu yerda
        // `ListEnvelope` kutilsa ro'yxat jimgina bo'sh chiqardi.
        val repo = RecordingsRepository(api)
        enqueue(200, """{"data":[{"id":"r1","status":"ready","size_bytes":1048576,"duration_sec":600}]}""")

        val items = runBlocking { repo.list("l1") }.getOrThrow()

        assertEquals(1, items.size)
        assertEquals(1_048_576L, items.first().sizeBytes)
        assertEquals("/api/v1/lessons/l1/recordings", server.takeRequest().path)
    }

    @Test
    fun `yuklab olish havolasi va muddati oqiladi`() {
        val repo = RecordingsRepository(api)
        enqueue(200, """{"data":{"url":"https://files.example/rec.mp4?sig=x","expires_in_s":900}}""")

        val dl = runBlocking { repo.downloadUrl("r1") }.getOrThrow()

        assertEquals("https://files.example/rec.mp4?sig=x", dl.url)
        assertEquals(900, dl.expiresInS)
    }

    @Test
    fun `bosh konvert xato sifatida qaytadi`() {
        // Server 200 berib, `data` ni bermasa — bu ilova uchun xato holat va
        // `null` bilan davom etish `NullPointerException` bo'lardi.
        val repo = RecordingsRepository(api)
        enqueue(200, """{}""")
        assertTrue(runBlocking { repo.downloadUrl("r1") }.isFailure)
    }

    // ─── Kutish xonasi ────────────────────────────────────────────────────────

    @Test
    fun `kutayotganlar royxati oqiladi`() {
        val repo = WaitingRoomRepository(api)
        enqueue(200, """{"data":[{"id":"w1","lesson_id":"l1","requester_name":"Ali","status":"pending"}]}""")

        val items = runBlocking { repo.pending("l1") }.getOrThrow()

        assertEquals("Ali", items.first().requesterName)
        assertEquals("/api/v1/lessons/l1/waitingroom", server.takeRequest().path)
    }

    @Test
    fun `qabul qilish guest tokenini tashlaydi`() {
        val repo = WaitingRoomRepository(api)
        enqueue(200, """{"data":{"token":"jwt","ws_url":"wss://lk","room_name":"r","identity":"g","role":"guest"}}""")

        val result = runBlocking { repo.admit("w1") }

        assertTrue(result.isSuccess)
        // Token ustozga kerak emas — u o'quvchiga WS orqali server tomonidan
        // yuboriladi. Uni qaytarish uni tasodifan ishlatish xavfini tug'dirardi.
        assertEquals(Unit, result.getOrNull())
    }

    @Test
    fun `409 xato emas — sorov allaqachon hal qilingan`() {
        // ⭐ Backend `TransitionFromPending` atomik: ikkinchi qaror 409 oladi.
        // Bu poyga natijasi (ikki marta bosildi yoki ikki qurilmadan), va
        // foydalanuvchi nuqtai nazaridan ish BAJARILGAN. Xato deb ko'rsatilsa
        // hal qilingan so'rov ekranda "xato" bilan osilib qolardi.
        val repo = WaitingRoomRepository(api)
        enqueue(409, """{"code":"CONFLICT","message":"already decided"}""")
        assertTrue(runBlocking { repo.admit("w1") }.isSuccess)

        enqueue(409, """{"code":"CONFLICT","message":"already decided"}""")
        assertTrue(runBlocking { repo.reject("w1") }.isSuccess)
    }

    @Test
    fun `boshqa xatolar yutilmaydi`() {
        // 409 ni yutish qoidasi FAQAT 409 ga tegishli bo'lishi kerak: 403
        // (boshqa ustozning darsi) jim o'tsa, so'rov ro'yxatdan yo'qolib,
        // o'quvchi eshik ortida qolib ketardi.
        val repo = WaitingRoomRepository(api)
        enqueue(403, """{"code":"FORBIDDEN","message":"not your lesson"}""")
        assertTrue(runBlocking { repo.admit("w1") }.isFailure)

        enqueue(404, """{"code":"NOT_FOUND","message":"gone"}""")
        assertTrue(runBlocking { repo.reject("w1") }.isFailure)
    }

    // ─── Profil ───────────────────────────────────────────────────────────────

    @Test
    fun `profil oqiladi va yangilanadi`() {
        val repo = ProfileRepository(api)
        enqueue(200, """{"data":{"id":"u1","email":"a@b.uz","full_name":"Ali","role":"mentor","timezone":"Asia/Tashkent"}}""")

        val user = runBlocking { repo.load() }.getOrThrow()

        assertEquals("Ali", user.fullName)
        assertEquals("Asia/Tashkent", user.timezone)
        assertEquals("/api/v1/users/me", server.takeRequest().path)
    }

    @Test
    fun `parol ozgartirish PUT yuboradi`() {
        val repo = ProfileRepository(api)
        enqueue(204)

        assertTrue(runBlocking { repo.changePassword("eski", "yangi12345") }.isSuccess)

        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/users/me/password", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("current_password"))
        assertTrue(body.contains("new_password"))
    }

    @Test
    fun `tegilmagan profil maydonlari JSONga tushmaydi`() {
        // Moshi `null` maydonlarni yozmaydi — shu tufayli `PUT /users/me`
        // avatar/rangni nolga tushirmaydi. Bu xulq test bilan qotirilgan:
        // `serializeNulls(true)` qo'shilsa bu yerda ko'rinadi.
        val repo = ProfileRepository(api)
        enqueue(200, """{"data":{"id":"u1","email":"a@b.uz","full_name":"Yangi","role":"mentor"}}""")

        runBlocking {
            repo.update(uz.darsly.mentor.data.api.UpdateProfileReq(fullName = "Yangi"))
        }

        val body = server.takeRequest().body.readUtf8()
        assertTrue(body.contains("full_name"))
        assertFalse(body.contains("timezone"))
        assertFalse(body.contains("language"))
    }

    @Test
    fun `bosh profil konverti xato sifatida qaytadi`() {
        val repo = ProfileRepository(api)
        enqueue(200, """{}""")
        val result = runBlocking { repo.load() }
        assertTrue(result.isFailure)
        assertNull(result.getOrNull())
    }
}
