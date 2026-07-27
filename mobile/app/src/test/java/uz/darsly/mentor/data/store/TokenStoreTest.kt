package uz.darsly.mentor.data.store

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import uz.darsly.mentor.data.api.TokenPair

/** M1 — token saqlash / o'qish / tozalash. */
class TokenStoreTest {

    private val pair = TokenPair(accessToken = "access-1", refreshToken = "refresh-1")

    @Test
    fun `bosh saqlagich null qaytaradi`() {
        assertNull(PrefsTokenStore(FakeSharedPreferences()).read())
    }

    @Test
    fun `saqlangan juftlik oqiladi`() {
        val store = PrefsTokenStore(FakeSharedPreferences())
        store.save(pair)
        assertEquals(pair, store.read())
    }

    @Test
    fun `ilova qayta ochilganda sessiya saqlanadi`() {
        // M1 ning asosiy va'dasi: ilova yopilib qayta ochilsa ustoz kirgan holatda bo'ladi.
        val prefs = FakeSharedPreferences()
        PrefsTokenStore(prefs).save(pair)

        val afterRestart = PrefsTokenStore(prefs.reopen())
        assertEquals(pair, afterRestart.read())
    }

    @Test
    fun `refresh natijasi eskisini almashtiradi`() {
        val store = PrefsTokenStore(FakeSharedPreferences())
        store.save(pair)
        val rotated = TokenPair("access-2", "refresh-2")
        store.save(rotated)
        assertEquals(rotated, store.read())
    }

    @Test
    fun `clear toliq tozalaydi va qayta ochilganda ham bosh`() {
        val prefs = FakeSharedPreferences()
        val store = PrefsTokenStore(prefs)
        store.save(pair)

        store.clear()

        assertNull(store.read())
        assertNull(PrefsTokenStore(prefs.reopen()).read())
        // Diskda ham token qolmasligi kerak.
        assertNull(prefs.getString("access_token", null))
        assertNull(prefs.getString("refresh_token", null))
    }

    @Test
    fun `yarim yozilgan holat null hisoblanadi`() {
        // Faqat access bor, refresh yo'q → sessiya yaroqsiz.
        val prefs = FakeSharedPreferences()
        prefs.edit().putString("access_token", "only-access").apply()
        assertNull(PrefsTokenStore(prefs).read())
    }

    @Test
    fun `InMemoryTokenStore ham ayni kontraktga amal qiladi`() {
        val store = InMemoryTokenStore()
        assertNull(store.read())
        store.save(pair)
        assertEquals(pair, store.read())
        store.clear()
        assertNull(store.read())
    }
}
