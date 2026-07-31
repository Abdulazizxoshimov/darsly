package uz.darsly.mentor.ui.lessons

import com.squareup.moshi.Moshi
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.setMain
import kotlinx.coroutines.withTimeout
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.SocketPolicy
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.data.repo.LessonsRepository
import uz.darsly.mentor.data.store.CachedLessons
import uz.darsly.mentor.data.store.LessonsCache

/**
 * ★ Loyihadagi BIRINCHI ViewModel testi.
 *
 * ## Nega u ilgari yo'q edi
 *
 * `LessonsViewModel` repozitoriyni O'ZI yasardi:
 * `repo: LessonsRepository = LessonsRepository.create(app)`. `create()` esa
 * ichida global `Net.api` va `PrefsLessonsCache` (Android `SharedPreferences`)
 * ga murojaat qilardi — ya'ni ViewModel'ni JVM testida yaratib bo'lmasdi:
 * Android konteksti va tarmoq talab qilinardi.
 *
 * Natijada 37 test faylida bitta ham ViewModel testi yo'q edi. Bu tasodif emas,
 * arxitektura oqibati: DI bo'lmagani uchun **butun taqdim etish qatlami**
 * sinovdan tashqarida qolgan edi.
 *
 * Hilt kiritilgach konstruktor bog'liqliklarni tashqaridan oladi va quyidagi
 * holatlarni oddiy JVM testida tekshirish mumkin bo'ldi.
 */
@OptIn(ExperimentalCoroutinesApi::class)
class LessonsViewModelTest {

    private lateinit var server: MockWebServer

    @Before
    fun setUp() {
        // `UnconfinedTestDispatcher` — `viewModelScope` korutinasi DARHOL
        // boshlansin. Tarmoq qismi (MockWebServer) esa haqiqiy IO da bajariladi,
        // shuning uchun natija `state` oqimini kutish bilan olinadi — virtual
        // vaqtni surish (`advanceUntilIdle`) bu yerda ishlamaydi.
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer().also { it.start() }
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
        server.shutdown()
    }

    // ── Yordamchilar ────────────────────────────────────────────────────────

    private fun api(): DarslyApi = Retrofit.Builder()
        .baseUrl(server.url("/"))
        .client(OkHttpClient())
        .addConverterFactory(MoshiConverterFactory.create(Moshi.Builder().build()))
        .build()
        .create(DarslyApi::class.java)

    /** Xotiradagi kesh — diskka ham, Android'ga ham bog'liq emas. */
    private class FakeCache(private var stored: CachedLessons? = null) : LessonsCache {
        override suspend fun read(): CachedLessons? = stored
        override suspend fun write(lessons: List<Lesson>, savedAtMillis: Long) {
            stored = CachedLessons(lessons, savedAtMillis)
        }
        override suspend fun clear() { stored = null }
    }

    private fun vm(cache: LessonsCache = FakeCache()) = LessonsViewModel(
        LessonsRepository(api = api(), cache = cache, now = { 1_000L }),
    )

    private fun jsonList(vararg titles: String): String {
        val items = titles.mapIndexed { i, t ->
            """{"id":"l$i","title":"$t","status":"scheduled","duration_min":60}"""
        }
        return """{"data":[${items.joinToString(",")}],"total":${titles.size},"page":1,"limit":50,"total_pages":1}"""
    }

    private fun lesson(id: String, title: String) =
        Lesson(id = id, title = title, status = "scheduled", durationMin = 60)

    // ── Testlar ─────────────────────────────────────────────────────────────

    @Test
    fun `serverdan royxat kelsa holatga tushadi`() = runBlocking {
        server.enqueue(MockResponse().setBody(jsonList("Algebra")))
        val vm = vm()

        vm.start()
        vm.state.awaitUntil { it.lessons.isNotEmpty() }

        assertEquals(1, vm.state.value.lessons.size)
        assertEquals("Algebra", vm.state.value.lessons.first().title)
        assertFalse(vm.state.value.loading)
    }

    /**
     * ⭐ ENG MUHIM MEZON: internet uzilganda ustozning ro'yxati EKRANDA QOLISHI
     * kerak (kesh ko'rsatiladi) va offline belgisi yonishi shart.
     *
     * Bu xulq repozitoriy darajasida sinalgan edi; endi u ViewModel orqali —
     * ya'ni foydalanuvchi HAQIQATAN ko'radigan holat orqali — tasdiqlanadi.
     */
    @Test
    fun `tarmoq yiqilsa kesh korsatiladi va offline belgilanadi`() = runBlocking {
        server.enqueue(MockResponse().setSocketPolicy(SocketPolicy.DISCONNECT_AT_START))
        val cached = CachedLessons(listOf(lesson("l9", "Keshdagi dars")), savedAtMillis = 500L)
        val vm = vm(FakeCache(cached))

        vm.start()
        vm.state.awaitUntil { it.offline }

        val st = vm.state.value
        assertEquals("kesh ro'yxati yo'qolmasligi kerak", 1, st.lessons.size)
        assertEquals("Keshdagi dars", st.lessons.first().title)
        assertTrue("offline belgisi yonishi kerak", st.offline)
    }

    /**
     * Ekran qayta kompozitsiyaga kirganda `start()` yana chaqiriladi (masalan
     * darsdan qaytganda). Qorovul bo'lmasa har qaytishda ortiqcha tarmoq
     * so'rovi ketardi.
     */
    @Test
    fun `yuklanish davom etayotganda takroriy start sorov yubormaydi`() = runBlocking {
        server.enqueue(MockResponse().setBody(jsonList("A")))
        val vm = vm()

        vm.start()
        vm.start() // darhol, birinchisi tugamasdan
        vm.state.awaitUntil { !it.loading && !it.refreshing }

        assertEquals("ikkinchi start ortiqcha so'rov yubormasligi kerak", 1, server.requestCount)
    }

    @Test
    fun `bosh royxat xatosiz korsatiladi`() = runBlocking {
        server.enqueue(MockResponse().setBody("""{"data":[],"total":0,"page":1,"limit":50,"total_pages":0}"""))
        val vm = vm()

        vm.start()
        vm.state.awaitUntil { !it.loading && !it.refreshing }

        assertTrue(vm.state.value.lessons.isEmpty())
        assertFalse(vm.state.value.loading)
    }

    /**
     * Holat oqimida kutilgan shart bajarilguncha kutadi.
     *
     * Kerak, chunki tarmoq qismi HAQIQIY (MockWebServer) — u `runTest` ning
     * virtual soatiga bo'ysunmaydi. Timeout bo'lsa test osilib qolmaydi,
     * aniq xato beradi.
     */
    private suspend fun <T> kotlinx.coroutines.flow.StateFlow<T>.awaitUntil(
        predicate: (T) -> Boolean,
    ): T = withTimeout(5_000) { first(predicate) }
}
