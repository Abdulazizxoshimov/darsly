package uz.darsly.mentor.ui.room

import android.content.Intent
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.livekit.FakeLessonSession
import uz.darsly.mentor.service.FakeLessonPlatform

/**
 * Ekran ulashish controller'i — LiveKit'siz, Android'siz (M4 va tiklash yo'llari).
 *
 * `Intent` — android.jar stub'i (`isReturnDefaultValues=true`), unga tegilmaydi.
 */
class ScreenShareControllerTest {

    private var state = RoomUiState()
    private val log = mutableListOf<String>()
    private val platform = FakeLessonPlatform()
    private val session = FakeLessonSession()

    private fun controller(sdkInt: Int = 33) = ScreenShareController(
        platform = platform,
        onState = { t -> state = t(state) },
        onLog = { log += it },
        sdkInt = sdkInt,
    )

    @Test
    fun `boshlash — avval FGS, keyin capture, keyin ekran audiosi`() = runBlocking {
        controller().start(session, Intent())
        assertEquals("startService(true)", platform.calls.first())
        assertEquals(1, session.startShareCalls)
        assertTrue(session.screenAudioStarted)
        assertTrue(session.screenShareOn.value)
        assertNull(state.error)
    }

    @Test
    fun `M4 — FGS otsa korutina olmaydi, ustoz xato koradi, capture boshlanmaydi`() = runBlocking {
        // Android 12+ fon rejimi: `ForegroundServiceStartNotAllowedException`.
        platform.startFailure = IllegalStateException("not allowed to start foreground service")
        controller().start(session, Intent())
        assertEquals("FGS'siz capture boshlanmasligi kerak (Android 14+ SecurityException)", 0, session.startShareCalls)
        assertNotNull(state.error)
        assertFalse(state.restoreShare)
    }

    @Test
    fun `tiklash urinishi yiqilsa xato emas — bir bosishlik taklif`() = runBlocking {
        val c = controller()
        c.start(session, Intent()) // niyat = true
        platform.startFailure = IllegalStateException("bg")
        c.start(session, Intent())
        assertTrue("ustoz ulashayotgan edi — restoreShare", state.restoreShare)
        assertNull(state.error)
    }

    @Test
    fun `toxtatish niyatni ochiradi va FGS tipini qaytaradi`() = runBlocking {
        val c = controller()
        c.start(session, Intent())
        c.stop(session)
        assertFalse(c.wanted)
        assertEquals(1, session.stopShareCalls)
        assertEquals("startService(false)", platform.calls.last())
    }

    @Test
    fun `qayta ulanish — trek bor bolsa hech nima`() = runBlocking {
        val c = controller()
        c.start(session, Intent())
        session.reconcileResult = true
        assertNull(c.planAfterReconnect(session))
        assertFalse(state.restoreShare)
    }

    @Test
    fun `qayta ulanish — Android 13 da rozilik qayta ishlatiladi`() = runBlocking {
        val c = controller(sdkInt = 33)
        val consent = Intent()
        c.start(session, consent)
        session.reconcileResult = false
        assertNotNull("saqlangan rozilik qaytadi", c.planAfterReconnect(session))
    }

    @Test
    fun `qayta ulanish — Android 14 da rozilik qayta soraladi va bildirishnoma chiqadi`() = runBlocking {
        val c = controller(sdkInt = 34)
        c.start(session, Intent())
        session.reconcileResult = false
        assertNull(c.planAfterReconnect(session))
        assertTrue(state.restoreShare)
        assertEquals(1, platform.alerts.size)
    }

    @Test
    fun `niyat yoq bolsa qayta ulanishda hech nima`() {
        val c = controller(sdkInt = 34)
        assertNull(c.planAfterReconnect(session))
        assertTrue(platform.alerts.isEmpty())
    }
}
