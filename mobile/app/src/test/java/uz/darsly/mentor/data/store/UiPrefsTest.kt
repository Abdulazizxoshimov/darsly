package uz.darsly.mentor.data.store

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * B-1 — "Butun ekran" tushuntirishi bir marta o'chirilsa, **qaytib chiqmasligi** kerak.
 *
 * `PrefsTokenStore` testidagi kabi ishlab chiqarish klassining aynan o'zi sinaladi
 * ([FakeSharedPreferences] bilan).
 */
class UiPrefsTest {

    @Test
    fun `sukut boyicha tushuntirish korsatiladi`() {
        assertTrue(PrefsUiPrefs(FakeSharedPreferences()).screenShareTipEnabled)
    }

    @Test
    fun `ochirilgan tanlov ilova qayta ochilganda ham saqlanadi`() {
        val prefs = FakeSharedPreferences()
        PrefsUiPrefs(prefs).screenShareTipEnabled = false

        // Yangi obyekt = "ilova qayta ishga tushdi".
        assertFalse(PrefsUiPrefs(prefs.reopen()).screenShareTipEnabled)
    }

    @Test
    fun `qayta yoqish ham ishlaydi`() {
        val prefs = FakeSharedPreferences()
        val ui = PrefsUiPrefs(prefs)
        ui.screenShareTipEnabled = false
        ui.screenShareTipEnabled = true
        assertTrue(PrefsUiPrefs(prefs.reopen()).screenShareTipEnabled)
    }
}
