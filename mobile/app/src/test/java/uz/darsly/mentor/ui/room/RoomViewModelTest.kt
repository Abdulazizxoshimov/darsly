package uz.darsly.mentor.ui.room

import androidx.lifecycle.viewModelScope
import com.squareup.moshi.Moshi
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.setMain
import kotlinx.coroutines.withTimeout
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import org.junit.After
import org.junit.AfterClass
import org.junit.BeforeClass
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.data.livekit.FakeLessonSession
import uz.darsly.mentor.data.livekit.FakeSessionFactory
import uz.darsly.mentor.data.livekit.LessonSessionStore
import uz.darsly.mentor.data.livekit.TransportSource
import uz.darsly.mentor.data.repo.ChatAttachment
import uz.darsly.mentor.data.repo.ChatAttachments
import uz.darsly.mentor.data.repo.LessonsRepository
import uz.darsly.mentor.data.repo.LocalRecordingRepository
import uz.darsly.mentor.data.repo.ModerationRepository
import uz.darsly.mentor.data.repo.RoomChatRepository
import uz.darsly.mentor.data.repo.RoomRepository
import uz.darsly.mentor.data.store.CachedLessons
import uz.darsly.mentor.data.store.LessonsCache
import uz.darsly.mentor.data.store.UiPrefs
import uz.darsly.mentor.service.FakeLessonPlatform
import java.util.concurrent.CopyOnWriteArrayList

/**
 * ★ Xona ViewModel'ining birinchi JVM testi.
 *
 * Avval bu sinf `Context` va `LessonSessionHolder` (process `object`) ga
 * bog'langani uchun testda yaratib bo'lmasdi — va aynan shu sinfda C1
 * (yakunlashda yarim qolgan bo'shatish) va H5 (o'lik trekka bog'langan
 * yozuv) yashardi. Endi tizim chegarasi [FakeLessonPlatform], sessiya
 * [FakeLessonSession], tarmoq — MockWebServer.
 */
@OptIn(ExperimentalCoroutinesApi::class)
class RoomViewModelTest {

    private lateinit var server: MockWebServer
    private val requests = CopyOnWriteArrayList<String>()
    private val platform = FakeLessonPlatform()
    private val factory = FakeSessionFactory()
    private val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private lateinit var store: LessonSessionStore

    private var recordingEnabled = true

    companion object {
        /**
         * `Main` SINF uchun bir marta: har testda `setMain` qilinsa oldingi testning
         * OkHttp oqimida kechikib kelgan callback'i "Dispatchers.Main is used
         * concurrently with setting it" bilan keyingi testni yiqitadi.
         */
        @JvmStatic @BeforeClass fun mainOn() = Dispatchers.setMain(UnconfinedTestDispatcher())
        @JvmStatic @AfterClass fun mainOff() = Dispatchers.resetMain()
    }

    private val vms = mutableListOf<RoomViewModel>()

    @Before
    fun setUp() {
        server = MockWebServer().also { it.start() }
        // Xonaga kirishda bir necha so'rov PARALLEL ketadi (token, dars, holat,
        // chat) — navbat emas, yo'l bo'yicha javob.
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                val path = request.path.orEmpty()
                requests += "${request.method} $path"
                return when {
                    path.endsWith("/token") -> json(
                        """{"data":{"token":"t","ws_url":"wss://x","room_name":"lesson_l1","identity":"mentor","role":"host","lesson_id":"l1"}}""",
                    )
                    path.startsWith("/api/v1/lessons?") -> json(
                        """{"data":[{"id":"l1","title":"Algebra","status":"live","duration_min":60,"is_recording_enabled":$recordingEnabled}],"total":1,"page":1,"limit":50,"total_pages":1}""",
                    )
                    path.contains("/rooms/l1/state") -> json("""{"data":{"hands":[]}}""")
                    path.contains("/rooms/l1/chat") -> json("""{"data":[],"total":0,"page":1,"limit":50}""")
                    path.endsWith("/end") -> MockResponse().setResponseCode(204)
                    path.endsWith("/recording/local-start") -> json("""{"data":{"id":"rec1","status":"recording"}}""")
                    path.endsWith("/upload-url") -> json("""{"data":{"url":"${server.url("/put")}"}}""")
                    path == "/put" -> MockResponse().setResponseCode(200)
                    path.endsWith("/complete") -> MockResponse().setResponseCode(204)
                    else -> MockResponse().setResponseCode(404)
                }
            }
        }
        store = LessonSessionStore(factory, platform, appScope)
    }

    @After
    fun tearDown() {
        vms.forEach { it.viewModelScope.cancel() }
        appScope.cancel()
        server.shutdown()
    }

    private fun json(body: String) = MockResponse().setResponseCode(200).setBody(body)

    private class FakePrefs : UiPrefs {
        override var screenShareTipEnabled = true
        override var autoBackgroundOnShare = true
    }

    private class FakeCache : LessonsCache {
        private var stored: CachedLessons? = null
        override suspend fun read(): CachedLessons? = stored
        override suspend fun write(lessons: List<Lesson>, savedAtMillis: Long) { stored = CachedLessons(lessons, savedAtMillis) }
        override suspend fun clear() { stored = null }
    }

    private object NoAttachments : ChatAttachments {
        override fun read(uri: android.net.Uri): Result<ChatAttachment> = Result.failure(IllegalStateException("test"))
    }

    private fun vm(): RoomViewModel {
        val api = Retrofit.Builder()
            .baseUrl(server.url("/"))
            .client(OkHttpClient())
            .addConverterFactory(MoshiConverterFactory.create(Moshi.Builder().build()))
            .build()
            .create(DarslyApi::class.java)
        return RoomViewModel(
            rooms = RoomRepository(api),
            moderation = ModerationRepository(api),
            chatRepo = RoomChatRepository(api, NoAttachments),
            lessons = LessonsRepository(api, FakeCache()) { 1L },
            localRec = LocalRecordingRepository(api, OkHttpClient()),
            uiPrefs = FakePrefs(),
            sessions = store,
            platform = platform,
            transports = TransportSource { emptyFlow() },
            appScope = appScope,
        ).also { vms += it }
    }

    private suspend fun RoomViewModel.awaitJoined(): FakeLessonSession {
        withTimeout(5_000) { state.first { !it.connecting && it.identity != null } }
        return factory.created.single()
    }

    private suspend fun await(what: String, cond: () -> Boolean) = withTimeout(5_000) {
        while (!cond()) delay(10)
    }

    @Test
    fun `join — sessiya store dan, servis sessiyadan KEYIN, ulanadi`() = runBlocking {
        val vm = vm()
        vm.join("l1")
        val s = vm.awaitJoined()

        assertEquals(1, s.connectCalls.get())
        assertEquals(ConnState.DISCONNECTED, vm.state.value.connState) // Connected hodisasi soxta sessiyada yo'q
        assertTrue(vm.state.value.lessonLive)
        assertEquals("Algebra", vm.state.value.lessonTitle)
        // Servis sessiyadan KEYIN (store eskisini bo'shatib servisni to'xtatishi mumkin).
        assertEquals("startService(false)", platform.calls.first { it.startsWith("startService") })
        assertEquals(0, platform.stopServiceCalls)
        assertTrue(store.session.value === s)
    }

    @Test
    fun `C1 — Yakunlash dan keyin viewModelScope bekor bolsa ham boshatish tugaydi`() = runBlocking {
        val vm = vm()
        vm.join("l1")
        val s = vm.awaitJoined()

        vm.endLesson("l1")
        // UI `onLeave()` bilan ekranni darhol yopadi → qamrov bekor.
        vm.viewModelScope.cancel()

        await("release") { s.released }
        assertTrue("server `end` chaqirilishi kerak", requests.any { it == "POST /api/v1/lessons/l1/end" })
        assertNull(store.session.value)
        assertTrue("foreground servis to'xtashi kerak", platform.stopServiceCalls >= 1)
        assertFalse(vm.state.value.lessonActive)
    }

    @Test
    fun `C1 — yakunlashda ketayotgan yozuv yakunlanadi va yuklanadi`() = runBlocking {
        val vm = vm()
        vm.join("l1")
        val s = vm.awaitJoined()
        // Ulashish boshlandi → avto-yozuv (sozlama yoniq).
        s.startScreenShare(android.content.Intent(), null)
        await("recording") { vm.state.value.isRecording }
        assertEquals(1, s.recordingStarted)

        vm.endLesson("l1")
        vm.viewModelScope.cancel()

        await("release") { s.released }
        await("upload") { requests.any { it == "POST /api/v1/recordings/rec1/complete" } }
        assertEquals("yozuv AVVAL yakunlanadi, keyin sessiya", 1, s.stopRecordingCalls.get())
        assertTrue(requests.any { it == "PUT /put" })
    }

    @Test
    fun `H5 — trek ketsa yozuv segmenti yakunlanadi, qaytsa yangisi boshlanadi`() = runBlocking {
        val vm = vm()
        vm.join("l1")
        val s = vm.awaitJoined()

        s.startScreenShare(android.content.Intent(), null)
        await("rec1") { vm.state.value.isRecording }

        // Qayta ulanish trekni oldi (yoki tizim "Stop sharing").
        s.simulateShareLost(byUser = false)
        await("stopped") { !vm.state.value.isRecording && s.stopRecordingCalls.get() == 1 }
        assertFalse("REC indikatori o'chishi kerak", vm.state.value.isRecording)

        // Trek qaytdi — ustoz yozuvni xohlagan edi → yangi segment.
        s.startScreenShare(android.content.Intent(), null)
        await("rec2") { s.recordingStarted == 2 }
        assertTrue(vm.state.value.isRecording)
    }

    @Test
    fun `H5 — ustoz yozuvni ozi toxtatgan bolsa trek qaytganda tiklanmaydi`() = runBlocking {
        val vm = vm()
        vm.join("l1")
        val s = vm.awaitJoined()
        s.startScreenShare(android.content.Intent(), null)
        await("rec") { vm.state.value.isRecording }

        vm.toggleRecording() // ustoz to'xtatdi
        await("stopped") { s.stopRecordingCalls.get() == 1 }

        s.simulateShareLost(byUser = false)
        s.startScreenShare(android.content.Intent(), null)
        delay(200)
        assertEquals("maxfiylik: jimgina qayta boshlanmaydi", 1, s.recordingStarted)
        assertFalse(vm.state.value.isRecording)
    }

    @Test
    fun `avto-yozuv ochiq darsda ulashish yozuvni boshlamaydi`() = runBlocking {
        recordingEnabled = false
        val vm = vm()
        vm.join("l1")
        val s = vm.awaitJoined()
        s.startScreenShare(android.content.Intent(), null)
        delay(200)
        assertEquals(0, s.recordingStarted)
        assertFalse(vm.state.value.isRecording)
    }

    @Test
    fun `leave — sessiya boshatiladi, holat tozalanadi`() = runBlocking {
        val vm = vm()
        vm.join("l1")
        val s = vm.awaitJoined()
        vm.leave()
        await("release") { s.released }
        assertNull(store.session.value)
        assertEquals(RoomUiState(), vm.state.value)
        assertFalse(requests.any { it.endsWith("/end") })
    }

    @Test
    fun `ulanish yiqilsa sessiya boshatiladi va qayta urinish mumkin`() = runBlocking {
        val failing = FakeSessionFactory { id, t -> FakeLessonSession(id, t, connectFailure = { IllegalStateException("ws") }) }
        store = LessonSessionStore(failing, platform, appScope)
        val vm = vm()
        vm.join("l1")
        withTimeout(5_000) { vm.state.first { it.error != null } }
        val s = failing.created.single()
        await("release") { s.released }
        assertNull(store.session.value)

        vm.retryJoin("l1")
        withTimeout(5_000) { vm.state.first { it.error != null } }
        assertEquals("qorovul IDLE ga qaytgan — ikkinchi urinish haqiqatan ketadi", 2, failing.created.size)
    }
}
