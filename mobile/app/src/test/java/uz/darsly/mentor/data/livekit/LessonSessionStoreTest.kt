package uz.darsly.mentor.data.livekit

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.service.FakeLessonPlatform

/**
 * Sessiyaning yagona egasi (M2) — C1/C2/M6 ning poydevori.
 *
 * Bu yerdagi har test avval qurilmada ko'ringan xatoni qotiradi: bekor
 * qilingan qamrovda yarim qolgan bo'shatish, eski sessiyaning kechikkan
 * tozalashi yangisini o'ldirishi, ikki marta bo'shatish.
 */
class LessonSessionStoreTest {

    private val platform = FakeLessonPlatform()
    private val factory = FakeSessionFactory()
    private val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val store = LessonSessionStore(factory, platform, appScope)

    private fun start(id: String = "l1") = runBlocking { store.start(id, FakeLessonSession.token(id)) as FakeLessonSession }

    @Test
    fun `start sessiyani oqimga qoyadi`() {
        val s = start()
        assertSame(s, store.session.value)
        assertFalse(s.released)
    }

    @Test
    fun `release sessiyani boshatadi va servisni toxtatadi`() = runBlocking {
        val s = start()
        store.release(s)
        assertTrue("session.release() chaqirilishi kerak", s.released)
        assertNull(store.session.value)
        assertEquals(listOf("hideFrame", "stopService"), platform.calls)
    }

    @Test
    fun `C1 — chaqiruvchi qamrovi bekor qilinsa ham boshatish tugaydi`() {
        // Aynan C1 stsenariysi: UI "Yakunlash" dan keyin darhol ekranni yopadi
        // va `viewModelScope` bekor bo'ladi. Bo'shatish ILOVA qamrovida
        // bo'lgani uchun baribir yetib boradi.
        val s = start()
        val callerScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val requested = CompletableDeferred<Unit>()
        callerScope.launch {
            store.releaseAsync(s)
            requested.complete(Unit)
            awaitCancellation() // "ekran hali ochiq" — to navigatsiya uni bekor qilguncha
        }
        runBlocking { requested.await() }
        callerScope.cancel()

        runBlocking { withTimeout(2_000) { while (!s.released) kotlinx.coroutines.delay(10) } }
        assertNull(store.session.value)
        assertTrue(platform.stopServiceCalls >= 1)
    }

    @Test
    fun `takroriy release zararsiz`() = runBlocking {
        val s = start()
        store.release(s)
        store.release(s)
        store.releaseAll()
        assertEquals("sessiya BIR marta bo'shatiladi", 1, s.releaseCalls.get())
        assertEquals(1, platform.stopServiceCalls)
    }

    @Test
    fun `M6 — eski sessiyaning kechikkan release i yangisiga tegmaydi`() = runBlocking {
        val old = start("l1")
        val fresh = start("l2") // eskisi shu yerda bo'shatiladi
        assertTrue(old.released)

        store.release(old) // kechikib kelgan tozalash (masalan eski onDisconnected)
        assertSame("yangi sessiya joyida qolishi kerak", fresh, store.session.value)
        assertFalse(fresh.released)
    }

    @Test
    fun `releaseAllAsync sessiyani SINXRON ushlaydi`() = runBlocking {
        // Servisning kechikkan `onDestroy` si: "joriy"ni keyin emas, chaqiruv
        // paytida aniqlaydi — sessiya yo'q bo'lsa hech nima bo'lmaydi.
        assertNull(store.releaseAllAsync())
        val s = start()
        store.releaseAllAsync()?.join()
        assertTrue(s.released)
    }

    @Test
    fun `release otsa ham servis toxtaydi`() = runBlocking {
        val f = FakeSessionFactory { id, t -> FakeLessonSession(id, t, releaseFailure = IllegalStateException("SDK yiqildi")) }
        val st = LessonSessionStore(f, platform, appScope)
        st.start("l1", FakeLessonSession.token("l1"))
        st.releaseAll()
        assertNull(st.session.value)
        assertEquals(1, platform.stopServiceCalls)
    }
}
