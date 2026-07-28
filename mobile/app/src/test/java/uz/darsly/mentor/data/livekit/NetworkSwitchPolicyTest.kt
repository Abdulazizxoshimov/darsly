package uz.darsly.mentor.data.livekit

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * C-11: Wi-Fi ↔ LTE almashuvi.
 *
 * Bu testlar QARORNI qotiradi: qachon majburiy qayta ulanamiz va qachon
 * tegmaymiz. Qurilmadagi haqiqiy almashuv bularning o'rnini bosmaydi — u
 * alohida qo'lda sinaladi (`docs/MOBILE-DEVICE-TEST.md`, C-11).
 */
class NetworkSwitchPolicyTest {

    @Test
    fun `Wi-Fi dan LTE ga otishda majburiy qayta ulanadi`() {
        // Ayni C-11 stsenariysi: eski interfeys "half-open" qoladi va SDK uni
        // faqat timeout orqali sezadi — biz esa signalni darhol olamiz.
        assertTrue(
            NetworkSwitchPolicy.shouldForceReconnect(Transport.WIFI, Transport.CELLULAR, connected = true),
        )
    }

    @Test
    fun `LTE dan Wi-Fi ga otishda ham qayta ulanadi`() {
        assertTrue(
            NetworkSwitchPolicy.shouldForceReconnect(Transport.CELLULAR, Transport.WIFI, connected = true),
        )
    }

    @Test
    fun `bir xil transport ichidagi ozgarish darsni uzmaydi`() {
        // Bir Wi-Fi nuqtadan boshqasiga o'tish WebRTC uchun odatda shaffof.
        // Bu yerda majburiy uzib-ulash foydadan ko'ra zarar keltiradi.
        assertFalse(
            NetworkSwitchPolicy.shouldForceReconnect(Transport.WIFI, Transport.WIFI, connected = true),
        )
    }

    @Test
    fun `birinchi olchov almashuv deb hisoblanmaydi`() {
        // Ilova endi ishga tushdi — bu boshlang'ich holat, almashuv emas.
        assertFalse(
            NetworkSwitchPolicy.shouldForceReconnect(null, Transport.WIFI, connected = true),
        )
    }

    @Test
    fun `xonada bolmasak qayta ulanmaymiz`() {
        assertFalse(
            NetworkSwitchPolicy.shouldForceReconnect(Transport.WIFI, Transport.CELLULAR, connected = false),
        )
    }

    @Test
    fun `internet umuman yoqolganda kutamiz`() {
        // Majburiy urinish shu zahoti yiqiladi va faqat backoff'ni sarflaydi.
        // Tarmoq qaytganda qaror qaytadan hisoblanadi.
        assertFalse(
            NetworkSwitchPolicy.shouldForceReconnect(Transport.WIFI, Transport.NONE, connected = true),
        )
    }

    @Test
    fun `tarmoq qaytganda qayta ulanadi`() {
        assertTrue(
            NetworkSwitchPolicy.shouldForceReconnect(Transport.NONE, Transport.CELLULAR, connected = true),
        )
    }

    @Test
    fun `izoh foydalanuvchi tilida va faqat kerak bolganda chiqadi`() {
        assertNull("boshlang'ich holat izohsiz", NetworkSwitchPolicy.label(null, Transport.WIFI))
        assertNull("o'zgarish yo'q — izoh yo'q", NetworkSwitchPolicy.label(Transport.WIFI, Transport.WIFI))
        assertEquals("Mobil internetga o'tildi", NetworkSwitchPolicy.label(Transport.WIFI, Transport.CELLULAR))
        assertEquals("Wi-Fi'ga o'tildi", NetworkSwitchPolicy.label(Transport.CELLULAR, Transport.WIFI))
        assertEquals("Internet yo'q", NetworkSwitchPolicy.label(Transport.WIFI, Transport.NONE))
    }
}
