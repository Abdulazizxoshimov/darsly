package uz.darsly.mentor.data.api

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * `SESSION_REVOKED` — «boshqa qurilmada kirildi».
 *
 * NEGA ALOHIDA TEST: bu kod TOKEN_EXPIRED'dan farqli o'laroq **qayta urinib
 * bo'lmaydigan** holat, va uni umumiy "sessiya tugadi" ga qo'shib yuborish
 * ustozni noto'g'ri yo'ldan boshlardi — u parolni yoki internetni ayblab,
 * hisobi ulashilganini bilmay qolardi (bitta akkaunt = bitta faol sessiya,
 * PRODUCT.md №1).
 */
class LogoutReasonTest {

    @Test
    fun `session revoked kodi tanib olinadi`() {
        assertEquals(LogoutReason.REVOKED, LogoutReason.ofCode("SESSION_REVOKED"))
    }

    @Test
    fun `token kodlari muddat tugashi deb qaraladi`() {
        assertEquals(LogoutReason.EXPIRED, LogoutReason.ofCode("TOKEN_EXPIRED"))
        assertEquals(LogoutReason.EXPIRED, LogoutReason.ofCode("TOKEN_INVALID"))
    }

    @Test
    fun `notanish va bosh kod sababsiz qoladi`() {
        assertEquals(LogoutReason.UNKNOWN, LogoutReason.ofCode(null))
        assertEquals(LogoutReason.UNKNOWN, LogoutReason.ofCode(""))
        assertEquals(LogoutReason.UNKNOWN, LogoutReason.ofCode("YANGI_KOD"))
    }

    @Test
    fun `xato konvertidan sabab oqiladi`() {
        val body = """{"code":"SESSION_REVOKED","message":"session revoked"}"""
        assertEquals(LogoutReason.REVOKED, LogoutReason.ofBody(body))
    }

    @Test
    fun `buzuq yoki bosh tana ilovani yiqitmaydi`() {
        // Bu yo'l HAR 401 da bosiladi — parse xatosi butun ilovani o'ldirardi.
        assertEquals(LogoutReason.UNKNOWN, LogoutReason.ofBody(null))
        assertEquals(LogoutReason.UNKNOWN, LogoutReason.ofBody(""))
        assertEquals(LogoutReason.UNKNOWN, LogoutReason.ofBody("   "))
        assertEquals(LogoutReason.UNKNOWN, LogoutReason.ofBody("<html>502 Bad Gateway</html>"))
        assertEquals(LogoutReason.UNKNOWN, LogoutReason.ofBody("""{"code":}"""))
        assertEquals(LogoutReason.UNKNOWN, LogoutReason.ofBody("""{}"""))
    }

    @Test
    fun `sababsiz holatda login ekranida hech nima yozilmaydi`() {
        assertNull(LogoutReason.UNKNOWN.message)
    }

    @Test
    fun `revoked matni boshqa qurilma haqida gapiradi`() {
        // Matn web bilan bir xil bo'lishi kerak (`frontend/src/lib/logoutReason.js`).
        // "Sessiya tugadi" deyish ADASHTIRARDI — sabab butunlay boshqa.
        val text = LogoutReason.REVOKED.message
        assertNotNull(text)
        assertTrue("sabab aytilmagan: $text", text!!.contains("Boshqa qurilmada"))
    }

    @Test
    fun `ApiErrors ham bu kodni ozbekchaga ogiradi`() {
        // ViewModel'lar `humanError` orqali ishlaydi: agar u bu kodni tanimasa,
        // ekranda serverning inglizcha matni chiqib qolardi.
        assertEquals("Boshqa qurilmada kirildi — qaytadan kiring", ApiErrors.uz("SESSION_REVOKED"))
    }
}

/**
 * [SessionManager] sabab bilan chiqarish.
 *
 * Sabab HOLAT bo'lishi shart: sessiya tugatilishi va login ekranining paydo
 * bo'lishi orasida bir necha kadr o'tadi va hodisa aynan shu oraliqda
 * yo'qolardi (`SessionStateTest` dagi `loggedIn` bilan bir xil sabab).
 */
class SessionLogoutReasonTest {

    private val pair = TokenPair("access-1", "refresh-1")

    @Test
    fun `boshlangich holatda sabab yoq`() {
        val session = SessionManager(uz.darsly.mentor.data.store.InMemoryTokenStore())
        assertEquals(LogoutReason.UNKNOWN, session.logoutReason.value)
    }

    @Test
    fun `majburiy chiqishda sabab saqlanadi`() {
        val session = SessionManager(uz.darsly.mentor.data.store.InMemoryTokenStore(pair))
        session.forceLogout(LogoutReason.REVOKED)
        assertEquals(LogoutReason.REVOKED, session.logoutReason.value)
        assertNull("tokenlar tozalanishi shart", session.read())
    }

    @Test
    fun `foydalanuvchi ozi chiqsa sabab korsatilmaydi`() {
        // "Chiqish" tugmasini bosgan ustozga "nega chiqdingiz" deb tushuntirish
        // ortiqcha shovqin bo'lardi — u nima qilganini biladi.
        val session = SessionManager(uz.darsly.mentor.data.store.InMemoryTokenStore(pair))
        session.forceLogout()
        assertEquals(LogoutReason.UNKNOWN, session.logoutReason.value)
    }

    @Test
    fun `yangi login eski sababni tozalaydi`() {
        val session = SessionManager(uz.darsly.mentor.data.store.InMemoryTokenStore(pair))
        session.forceLogout(LogoutReason.REVOKED)
        session.save(TokenPair("a2", "r2"))
        // Aks holda sabab keyingi chiqishda qayta ko'rinib, ustozni chalg'itardi.
        assertEquals(LogoutReason.UNKNOWN, session.logoutReason.value)
    }

    @Test
    fun `korsatilgan sabab takror chiqmaydi`() {
        val session = SessionManager(uz.darsly.mentor.data.store.InMemoryTokenStore(pair))
        session.forceLogout(LogoutReason.EXPIRED)
        session.logoutReasonShown()
        assertEquals(LogoutReason.UNKNOWN, session.logoutReason.value)
    }
}
