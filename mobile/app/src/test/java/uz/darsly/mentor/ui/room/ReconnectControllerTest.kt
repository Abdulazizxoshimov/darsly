package uz.darsly.mentor.ui.room

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.livekit.FakeLessonSession
import uz.darsly.mentor.data.livekit.Transport
import uz.darsly.mentor.data.livekit.TransportSource
import java.io.IOException

/**
 * Majburiy qayta ulanish sikli (C-11) — LiveKit'siz, virtual vaqt bilan.
 */
@OptIn(ExperimentalCoroutinesApi::class)
class ReconnectControllerTest {

    private val state = MutableStateFlow(RoomUiState(connState = ConnState.CONNECTED))
    private val log = mutableListOf<String>()
    private val transports = MutableSharedFlow<Transport>()
    private var reconnected = 0
    private val sleeps = mutableListOf<Long>()

    /** Har kutishda `intentional` qiymati — sikl ICHIDA bayroq ko'tarilganini tekshirish uchun. */
    private val intentionalDuringSleep = mutableListOf<Boolean>()
    private lateinit var current: ReconnectController

    private fun TestScope.controller() = ReconnectController(
        transports = TransportSource { transports },
        scope = this,
        onState = { t -> state.value = t(state.value) },
        onLog = { log += it },
        onReconnected = { reconnected++ },
        sleep = { sleeps += it; intentionalDuringSleep += current.intentional },
    ).also { current = it }

    @Test
    fun `birinchi urinish yiqilsa ham keyingisi ulaydi`() = runTest(StandardTestDispatcher()) {
        // Qurilma sinovi (2026-07-28): LTE'da birinchi urinish DNS bilan yiqiladi.
        var failures = 2
        val s = FakeLessonSession(connectFailure = { if (failures-- > 0) IOException("dns") else null })
        val c = controller()
        c.force(s)
        advanceUntilIdle()
        assertTrue("uzilish paytida intentional=true", intentionalDuringSleep.all { it })

        assertEquals(3, s.connectCalls.get())
        assertEquals(1, reconnected)
        assertFalse(state.value.reconnecting)
        assertNull(state.value.networkNote)
        assertFalse("bayroq HAR holatda tushadi", c.intentional)
        assertEquals(ReconnectController.RECONNECT_DELAYS.take(3), sleeps)
    }

    @Test
    fun `urinishlar tugasa rost xabar`() = runTest(StandardTestDispatcher()) {
        val s = FakeLessonSession(connectFailure = { IOException("down") })
        controller().force(s)
        advanceUntilIdle()
        assertEquals(ReconnectController.RECONNECT_DELAYS.size, s.connectCalls.get())
        assertEquals("Qayta ulanib bo'lmadi", state.value.networkNote)
        assertEquals(0, reconnected)
    }

    @Test
    fun `sessiya almashsa sikl toxtaydi`() = runTest(StandardTestDispatcher()) {
        val s = FakeLessonSession(connectFailure = { IOException("down") })
        controller().force(s, isCurrent = { false })
        advanceUntilIdle()
        assertEquals("almashgan sessiyaga ulanmaydi", 0, s.connectCalls.get())
    }

    @Test
    fun `bir vaqtda bitta sikl`() = runTest(StandardTestDispatcher()) {
        val s = FakeLessonSession()
        val c = controller()
        c.force(s); c.force(s)
        advanceUntilIdle()
        assertEquals(1, s.connectCalls.get())
    }

    @Test
    fun `tarmoq almashsa majburiy qayta ulanadi, bir xil transportda emas`() = runTest(StandardTestDispatcher()) {
        val s = FakeLessonSession()
        val c = controller()
        val job = c.observeNetwork(state) { s }
        advanceUntilIdle() // obuna o'rnatilsin — aks holda birinchi emit tashlanadi
        transports.emit(Transport.WIFI)
        transports.emit(Transport.WIFI)
        advanceUntilIdle()
        assertEquals(0, s.connectCalls.get())

        transports.emit(Transport.CELLULAR)
        advanceUntilIdle()
        assertEquals(1, s.connectCalls.get())
        assertTrue(log.any { it.contains("majburiy qayta ulanish") })
        job.cancel()
    }
}
