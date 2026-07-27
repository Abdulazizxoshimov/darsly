package uz.darsly.mentor.data.repo

import com.squareup.moshi.Moshi
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import okhttp3.mockwebserver.SocketPolicy
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import uz.darsly.mentor.data.api.CreateLessonReq
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.data.store.CachedLessons
import uz.darsly.mentor.data.store.LessonsCache
import java.io.IOException

/**
 * M6 · M7 — repozitoriy: tarmoq + offline kesh.
 *
 * ENG MUHIM MEZON (B-4): **tarmoq yiqilganda kesh o'chmasligi kerak**. Ustoz
 * koridorda ro'yxatni ko'ra olishi shart. Shu sabab bu yerda mock obyekt emas,
 * haqiqiy HTTP (MockWebServer) va haqiqiy kesh implementatsiyasi ishlatiladi.
 */
class LessonsRepositoryTest {

    private lateinit var server: MockWebServer
    private lateinit var api: DarslyApi

    /** Xotiradagi kesh — yozuvlar sonini ham sanaydi. */
    private class RecordingCache(private var value: CachedLessons? = null) : LessonsCache {
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

    private companion object {
        const val LIST_BODY = """
            {"data":[
              {"id":"l1","title":"Algebra","status":"scheduled","join_slug":"abc","duration_min":60},
              {"id":"l2","title":"Geometriya","status":"live","join_slug":"def","duration_min":45}
            ],"total":2,"page":1,"limit":50,"total_pages":1}
        """

        const val CREATED_BODY = """
            {"data":{"id":"new1","title":"Yangi dars","status":"scheduled",
                     "join_slug":"xyz789","duration_min":90,"has_passcode":true}}
        """
    }

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
    fun tearDown() {
        server.shutdown()
    }

    private fun repo(cache: LessonsCache, now: Long = 1_000L) =
        LessonsRepository(api = api, cache = cache, now = { now })

    @Test
    fun `royxat yuklanadi va keshga yoziladi`() = runBlocking {
        server.enqueue(MockResponse().setResponseCode(200).setBody(LIST_BODY))
        val cache = RecordingCache()

        val result = repo(cache, now = 777L).refresh()

        assertEquals(listOf("l1", "l2"), result.getOrThrow().lessons.map { it.id })
        assertEquals(1, cache.writes)
        assertEquals(777L, cache.read()!!.savedAtMillis)
        assertEquals("abc", cache.read()!!.lessons[0].joinSlug)
    }

    @Test
    fun `tarmoq uzilsa kesh TEGILMAYDI va xato offline deb belgilanadi`() = runBlocking {
        val cache = RecordingCache(CachedLessons(listOf(lesson("eski")), savedAtMillis = 5L))
        server.enqueue(MockResponse().setSocketPolicy(SocketPolicy.DISCONNECT_AT_START))

        val result = repo(cache).refresh()

        assertTrue(result.isFailure)
        assertTrue(
            "IOException offline deb tanilishi kerak",
            LessonsRepository.isOffline(result.exceptionOrNull()!!),
        )
        assertEquals("kesh ustiga yozilmasligi kerak", 0, cache.writes)
        assertEquals(listOf("eski"), cache.read()!!.lessons.map { it.id })
    }

    @Test
    fun `server xatosi offline emas`() = runBlocking {
        // 500 — internet BOR, server yiqilgan. "Internet yo'q" deyish yolg'on bo'lardi.
        server.enqueue(
            MockResponse().setResponseCode(500)
                .setBody("""{"code":"INTERNAL_ERROR","message":"boom"}"""),
        )
        val cache = RecordingCache()

        val result = repo(cache).refresh()

        assertTrue(result.isFailure)
        assertFalse(LessonsRepository.isOffline(result.exceptionOrNull()!!))
        assertEquals(0, cache.writes)
    }

    @Test
    fun `bosh data maydonli javob bosh royxat beradi`() = runBlocking {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"data":null,"total":0}"""))
        val cache = RecordingCache()

        assertTrue(repo(cache).refresh().getOrThrow().lessons.isEmpty())
        assertEquals(1, cache.writes)
    }

    // ── Dars yaratish (M7) ────────────────────────────────────────────────────

    @Test
    fun `dars yaratiladi va kesh boshiga qoyiladi`() = runBlocking {
        server.enqueue(MockResponse().setResponseCode(201).setBody(CREATED_BODY))
        val cache = RecordingCache(CachedLessons(listOf(lesson("eski")), savedAtMillis = 5L))

        val created = repo(cache, now = 999L).create(
            CreateLessonReq(title = "Yangi dars", durationMin = 90, passcode = "1234"),
        ).getOrThrow()

        assertEquals("new1", created.id)
        assertEquals("xyz789", created.joinSlug)
        assertTrue(created.hasPasscode)
        assertEquals(listOf("new1", "eski"), cache.read()!!.lessons.map { it.id })
        assertEquals(999L, cache.read()!!.savedAtMillis)

        val body = server.takeRequest().body.readUtf8()
        assertTrue("sarlavha yuborilsin", body.contains(""""title":"Yangi dars""""))
        assertTrue("parol yuborilsin", body.contains(""""passcode":"1234""""))
    }

    @Test
    fun `yaratish yoli va metodi togri`() = runBlocking {
        server.enqueue(MockResponse().setResponseCode(201).setBody(CREATED_BODY))
        repo(RecordingCache()).create(CreateLessonReq(title = "X")).getOrThrow()

        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/lessons", request.path)
    }

    @Test
    fun `yaratishda validatsiya xatosi Result failure beradi`() = runBlocking {
        server.enqueue(
            MockResponse().setResponseCode(422)
                .setBody("""{"code":"VALIDATION_ERROR","message":"title: too short"}"""),
        )
        val cache = RecordingCache()

        val result = repo(cache).create(CreateLessonReq(title = "A"))

        assertTrue(result.isFailure)
        assertNull("muvaffaqiyatsiz yaratish keshga tegmasin", cache.read())
    }

    @Test
    fun `bosh data bilan 201 xato deb qaraladi`() = runBlocking {
        server.enqueue(MockResponse().setResponseCode(201).setBody("""{"data":null}"""))
        val result = repo(RecordingCache()).create(CreateLessonReq(title = "X"))
        assertTrue(result.isFailure)
        assertFalse(
            "bu tarmoq xatosi emas",
            result.exceptionOrNull() is IOException,
        )
    }

    // ── Sahifalash (🟡E) ──────────────────────────────────────────────────────

    @Test
    fun `50 tadan kop dars boр bolsa hamma sahifa olinadi`() = runBlocking {
        // Server 5 ta dars, sahifa hajmi 2 → 3 ta so'rov kutiladi.
        server.dispatcher = pagingDispatcher(total = 5, pageSize = 2)
        val cache = RecordingCache()

        val page = repo(cache).refresh(pageSize = 2).getOrThrow()

        assertEquals(5, page.lessons.size)
        assertEquals(5, page.total)
        assertFalse("chegara oshmagan", page.truncated)
        assertEquals(listOf("p1-0", "p1-1", "p2-0", "p2-1", "p3-0"), page.lessons.map { it.id })
        assertEquals(3, server.requestCount)
        // Kesh — TO'LIQ ro'yxat, birinchi sahifa emas.
        assertEquals(5, cache.read()!!.lessons.size)
    }

    @Test
    fun `chegaradan oshsa jim qirqilmaydi truncated bayrogi qoyiladi`() = runBlocking {
        // Server "1000 ta bor" deydi va har sahifada to'la chunk qaytaradi.
        server.dispatcher = pagingDispatcher(total = 1000, pageSize = 2)

        val page = repo(RecordingCache()).refresh(pageSize = 2).getOrThrow()

        assertTrue("foydalanuvchi qirqilganini bilishi kerak", page.truncated)
        assertEquals(LessonsRepository.MAX_PAGES * 2, page.lessons.size)
        assertEquals(LessonsRepository.MAX_PAGES, server.requestCount)
    }

    @Test
    fun `total notogri katta bolsa ham cheksiz sikl yoq`() = runBlocking {
        // Server "100 ta bor" deydi, lekin ikkinchi sahifada bo'sh qaytaradi.
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                val page = request.requestUrl?.queryParameter("page")?.toInt() ?: 1
                val body = if (page == 1) {
                    """{"data":[{"id":"a","title":"A","status":"scheduled"}],"total":100}"""
                } else {
                    """{"data":[],"total":100}"""
                }
                return MockResponse().setResponseCode(200).setBody(body)
            }
        }

        val page = repo(RecordingCache()).refresh(pageSize = 1).getOrThrow()

        assertEquals(1, page.lessons.size)
        assertEquals(2, server.requestCount)
    }

    @Test
    fun `ikkinchi sahifa xato bersa kesh tegilmaydi`() = runBlocking {
        val cache = RecordingCache(CachedLessons(listOf(lesson("eski")), savedAtMillis = 5L))
        server.enqueue(
            MockResponse().setResponseCode(200)
                .setBody("""{"data":[{"id":"a","title":"A","status":"scheduled"}],"total":9}"""),
        )
        server.enqueue(MockResponse().setResponseCode(500).setBody("""{"code":"INTERNAL_ERROR"}"""))

        val result = repo(cache).refresh(pageSize = 1)

        assertTrue(result.isFailure)
        assertEquals("chala ro'yxat keshni buzmasin", 0, cache.writes)
        assertEquals(listOf("eski"), cache.read()!!.lessons.map { it.id })
    }

    /** `total` ta darsni `pageSize` bo'yicha bo'lib beruvchi server. */
    private fun pagingDispatcher(total: Int, pageSize: Int) = object : Dispatcher() {
        override fun dispatch(request: RecordedRequest): MockResponse {
            val page = request.requestUrl?.queryParameter("page")?.toInt() ?: 1
            val start = (page - 1) * pageSize
            val count = minOf(pageSize, maxOf(0, total - start))
            val items = (0 until count).joinToString(",") { i ->
                """{"id":"p$page-$i","title":"Dars $start$i","status":"scheduled"}"""
            }
            return MockResponse().setResponseCode(200)
                .setBody("""{"data":[$items],"total":$total,"page":$page,"limit":$pageSize}""")
        }
    }

    private fun lesson(id: String) = Lesson(id = id, title = "Eski dars", status = "ended")
}
