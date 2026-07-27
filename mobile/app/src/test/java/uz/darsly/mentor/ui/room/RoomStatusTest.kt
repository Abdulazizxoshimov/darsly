package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * M16 (aloqa indikatori) va B-6 (dars tugashi) mantiqi.
 *
 * SDK enum'lari **nomi bo'yicha** o'giriladi — shu tufayli LiveKit'siz sinaladi.
 * Nomlar `livekit-android 2.27.0` dan `javap` bilan olingan, taxmin emas.
 */
class RoomStatusTest {

    // ── Aloqa sifati (M16) ────────────────────────────────────────────────────

    @Test
    fun `SDK sifat nomlari togri ogiriladi`() {
        assertEquals(RoomStatus.LinkQuality.EXCELLENT, RoomStatus.qualityOf("EXCELLENT"))
        assertEquals(RoomStatus.LinkQuality.GOOD, RoomStatus.qualityOf("GOOD"))
        assertEquals(RoomStatus.LinkQuality.POOR, RoomStatus.qualityOf("POOR"))
        assertEquals(RoomStatus.LinkQuality.LOST, RoomStatus.qualityOf("LOST"))
        assertEquals(RoomStatus.LinkQuality.UNKNOWN, RoomStatus.qualityOf("UNKNOWN"))
    }

    @Test
    fun `notanish yoki bosh sifat nomi ilovani yiqitmaydi`() {
        // SDK yangi qiymat qo'shsa eski ilova ishlashda davom etishi kerak.
        assertEquals(RoomStatus.LinkQuality.UNKNOWN, RoomStatus.qualityOf("SUPER_DUPER"))
        assertEquals(RoomStatus.LinkQuality.UNKNOWN, RoomStatus.qualityOf(null))
    }

    @Test
    fun `aloqa yaxshi bolganda indikator korinmaydi`() {
        // Zoom xulqi: hammasi joyida bo'lganda ustozni bezovta qilmaymiz.
        assertNull(RoomStatus.linkLabel(RoomStatus.LinkQuality.EXCELLENT, reconnecting = false))
        assertNull(RoomStatus.linkLabel(RoomStatus.LinkQuality.GOOD, reconnecting = false))
        assertNull(RoomStatus.linkLabel(RoomStatus.LinkQuality.UNKNOWN, reconnecting = false))
        assertFalse(RoomStatus.isWarning(RoomStatus.LinkQuality.GOOD, reconnecting = false))
    }

    @Test
    fun `sifat pasayganda ogohlantiradi`() {
        assertNotNull(RoomStatus.linkLabel(RoomStatus.LinkQuality.POOR, reconnecting = false))
        assertTrue(RoomStatus.isWarning(RoomStatus.LinkQuality.POOR, reconnecting = false))
        assertEquals("Aloqa yo'q", RoomStatus.linkLabel(RoomStatus.LinkQuality.LOST, false))
    }

    @Test
    fun `qayta ulanish sifatdan qatiy nazar korsatiladi`() {
        // Eng muhimi: dars jim to'xtaganda ustoz "buzildi" deb o'ylab telefonni
        // qayta ishga tushirmasligi kerak — nima bo'layotgani yozilib turadi.
        assertEquals(
            "Qayta ulanmoqda…",
            RoomStatus.linkLabel(RoomStatus.LinkQuality.EXCELLENT, reconnecting = true),
        )
        assertTrue(RoomStatus.isWarning(RoomStatus.LinkQuality.EXCELLENT, reconnecting = true))
    }

    // ── Dars tugashi (B-6) ────────────────────────────────────────────────────

    @Test
    fun `server xonani yopgani taniladi`() {
        assertEquals(RoomStatus.EndReason.HOST_ENDED, RoomStatus.endReasonOf("ROOM_DELETED"))
        assertEquals(RoomStatus.EndReason.HOST_ENDED, RoomStatus.endReasonOf("ROOM_CLOSED"))
        assertTrue(RoomStatus.endMessage(RoomStatus.EndReason.HOST_ENDED)!!.contains("yakunlandi"))
    }

    @Test
    fun `ozimiz chiqqanda ortiqcha xabar chiqmaydi`() {
        assertEquals(RoomStatus.EndReason.SELF, RoomStatus.endReasonOf("CLIENT_INITIATED"))
        assertNull(
            "\"Chiqish\" bosgan ustozga \"dars uzildi\" deyish yolg'on bo'lardi",
            RoomStatus.endMessage(RoomStatus.EndReason.SELF),
        )
    }

    @Test
    fun `tarmoq va server sabablari ajratiladi`() {
        listOf("CONNECTION_TIMEOUT", "MEDIA_FAILURE", "SIGNAL_CLOSE").forEach {
            assertEquals("$it → tarmoq", RoomStatus.EndReason.NETWORK, RoomStatus.endReasonOf(it))
        }
        listOf("SERVER_SHUTDOWN", "JOIN_FAILURE", "STATE_MISMATCH", "DUPLICATE_IDENTITY").forEach {
            assertEquals("$it → server", RoomStatus.EndReason.SERVER, RoomStatus.endReasonOf(it))
        }
        // Matnlar bir-biridan farq qilishi kerak: ustoz nima qilishni bilishi uchun.
        assertTrue(
            RoomStatus.endMessage(RoomStatus.EndReason.NETWORK) !=
                RoomStatus.endMessage(RoomStatus.EndReason.SERVER),
        )
        assertTrue(RoomStatus.endMessage(RoomStatus.EndReason.NETWORK)!!.contains("Internet"))
    }

    @Test
    fun `chiqarib yuborilganda alohida xabar`() {
        assertEquals(RoomStatus.EndReason.REMOVED, RoomStatus.endReasonOf("PARTICIPANT_REMOVED"))
        assertTrue(RoomStatus.endMessage(RoomStatus.EndReason.REMOVED)!!.contains("chiqarildingiz"))
    }

    @Test
    fun `notanish sabab ham bosh bolmagan xabar beradi`() {
        // Xabar bo'sh bo'lsa UI kartani ko'rsatmaydi va ustoz nima bo'lganini bilmaydi.
        listOf("UNKNOWN_REASON", "MIGRATION", "SIP_TRUNK_FAILURE", null, "").forEach { name ->
            val reason = RoomStatus.endReasonOf(name)
            if (reason != RoomStatus.EndReason.SELF) {
                assertTrue(
                    "sabab '$name' uchun xabar bo'sh",
                    !RoomStatus.endMessage(reason).isNullOrBlank(),
                )
            }
        }
    }
}
