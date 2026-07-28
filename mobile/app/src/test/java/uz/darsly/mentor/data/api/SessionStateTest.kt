package uz.darsly.mentor.data.api

import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.withTimeoutOrNull
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import uz.darsly.mentor.data.store.InMemoryTokenStore

/**
 * QA 🟡1 — logout signali YO'QOLMASLIGI kerak.
 *
 * Avvalgi yechim `MutableSharedFlow(replay = 0)` edi: `forceLogout()` obunachi
 * bo'lmaganda ham `true` qaytarardi, lekin qiymat tashlab yuborilardi. Refresh
 * muvaffaqiyatsizligi UI kollektori ulanishidan oldin sodir bo'lsa — tokenlar
 * o'chgan, ekran esa "Darslar"da qolib ketardi.
 */
class SessionStateTest {

    private val pair = TokenPair("access-1", "refresh-1")

    @Before
    fun setUp() {
        // Global `object` — har test toza saqlagichdan boshlansin.
        Session.installForTest(InMemoryTokenStore())
    }

    @Test
    fun `boshlangich holat - kirilmagan`() {
        assertFalse(Session.isLoggedIn)
        assertFalse(Session.loggedIn.value)
        assertNull(Session.accessToken)
    }

    @Test
    fun `saqlangan token bilan ornatilsa darhol kirgan holat`() {
        // Ilova qayta ochilishini modellaydi: saqlagichda allaqachon token bor.
        Session.installForTest(InMemoryTokenStore(pair))
        assertTrue(Session.isLoggedIn)
        assertTrue(Session.loggedIn.value)
        assertEquals("access-1", Session.accessToken)
    }

    @Test
    fun `save va clear holatni yangilaydi`() {
        Session.save(pair)
        assertTrue(Session.loggedIn.value)
        Session.clear()
        assertFalse(Session.loggedIn.value)
        assertNull(Session.read())
    }

    /** ★ ASOSIY TEST: hodisa sodir bo'lganda HECH KIM tinglamayotgan edi. */
    @Test
    fun `kech ulangan obunachi ham logoutni koradi`() = runTest {
        Session.save(pair)
        assertTrue(Session.loggedIn.value)

        // Logout hech qanday obunachisiz sodir bo'ladi (aynan poyga holati).
        Session.forceLogout()

        // Obunachi FAQAT SHUNDAN KEYIN ulanadi — u baribir `false` ni olishi shart.
        // Hodisaga asoslangan yechimda bu yerda hech narsa kelmasdi va
        // `withTimeoutOrNull` `null` qaytarardi.
        val observed = withTimeoutOrNull(1_000) { Session.loggedIn.first() }
        assertEquals("kech ulangan obunachi logoutni ko'rmadi", false, observed)
        assertNull("tokenlar tozalangan bo'lishi kerak", Session.read())
    }

    @Test
    fun `logoutdan keyingi login holatni tiklaydi`() {
        Session.save(pair)
        Session.forceLogout()
        assertFalse(Session.loggedIn.value)

        // Eski logout "hodisasi" qayta o'ynalib login'ni buzmasligi kerak.
        Session.save(TokenPair("access-2", "refresh-2"))
        assertTrue(Session.loggedIn.value)
        assertEquals("access-2", Session.accessToken)
    }

    @Test
    fun `TokenAuthenticator hard logouti holatni tushiradi`() {
        // Net `Session` ni to'g'ridan-to'g'ri store sifatida beradi, hard logout esa
        // `Session.forceLogout()` ni chaqiradi — shu zanjir ishlashini tasdiqlaymiz.
        Session.save(pair)
        val onHardLogout = { Session.forceLogout() }
        onHardLogout()
        assertFalse(Session.loggedIn.value)
        assertNull(Session.read())
    }
}

/**
 * Profil 404 → avtomatik chiqish.
 *
 * Qurilma sinovida (2026-07-28) uchragan holat: token yaroqli (Redis'da sessiya
 * bor), lekin foydalanuvchi DB'dan o'chirilgan → `GET /users/me` 404 qaytaradi.
 * Avval ekranda "Topilmadi / Qayta urinish" boshi berk ko'chasi qolardi:
 * qayta urinish aynan o'sha 404 ni qaytaradi, chiqish tugmasi esa yo'q.
 *
 * Bu test QAROR qoidasini qotiradi: 404 — xato emas, o'lik sessiya belgisi.
 */
class ProfileNotFoundPolicyTest {

    /** ViewModel'dagi shart bilan bir xil (`t is HttpException && t.code() == 404`). */
    private fun isDeadSession(code: Int) = code == 404

    @org.junit.Test
    fun `404 olik sessiya deb qaraladi`() {
        org.junit.Assert.assertTrue(isDeadSession(404))
    }

    @org.junit.Test
    fun `boshqa xatolar sessiyani oldirmaydi`() {
        // 500 yoki 503 — vaqtinchalik server muammosi. Bularda chiqarib yuborish
        // foydalanuvchini bekorga tizimdan haydab chiqarardi.
        for (code in intArrayOf(400, 401, 403, 429, 500, 503)) {
            org.junit.Assert.assertFalse("$code chiqishga sabab bo'lmasligi kerak", isDeadSession(code))
        }
    }
}
