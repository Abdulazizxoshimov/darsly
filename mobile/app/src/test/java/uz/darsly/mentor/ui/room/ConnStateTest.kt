package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/** M7 — SDK holat nomi enum'ga o'giriladi; notanish nom ilovani yiqitmaydi. */
class ConnStateTest {

    @Test
    fun `SDK nomlari ogiriladi`() {
        assertEquals(ConnState.CONNECTED, ConnState.fromSdk("CONNECTED"))
        assertEquals(ConnState.CONNECTING, ConnState.fromSdk("CONNECTING"))
        assertEquals(ConnState.RECONNECTING, ConnState.fromSdk("RECONNECTING"))
        assertEquals(ConnState.DISCONNECTED, ConnState.fromSdk("DISCONNECTED"))
    }

    @Test
    fun `notanish yoki bosh nom — uzilgan`() {
        assertEquals(ConnState.DISCONNECTED, ConnState.fromSdk("MIGRATING"))
        assertEquals(ConnState.DISCONNECTED, ConnState.fromSdk(null))
    }

    @Test
    fun `xonada bolish`() {
        assertTrue(ConnState.CONNECTED.inRoom)
        assertTrue(ConnState.RECONNECTING.inRoom)
        assertFalse(ConnState.CONNECTING.inRoom)
        assertFalse(ConnState.DISCONNECTED.inRoom)
    }
}
