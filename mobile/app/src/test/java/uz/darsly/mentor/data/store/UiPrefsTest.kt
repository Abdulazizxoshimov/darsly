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

    // ─── №25: ulashishda ilova fonga o'tsinmi ────────────────────────────────

    @Test
    fun `fonga otish sukut boyicha YONIQ`() {
        // Mahsulot qarori: 2026-08-01 yozuvida aynan bu yo'qligi tufayli
        // o'quvchilar dars materiali o'rniga Jonly interfeysini ko'rgan.
        assertTrue(PrefsUiPrefs(FakeSharedPreferences()).autoBackgroundOnShare)
    }

    @Test
    fun `ochirilgan bolsa ilova qayta ochilganda ham ochiq qoladi`() {
        val prefs = FakeSharedPreferences()
        PrefsUiPrefs(prefs).autoBackgroundOnShare = false
        assertFalse(PrefsUiPrefs(prefs.reopen()).autoBackgroundOnShare)
    }

    @Test
    fun `ikki sozlama bir-biriga aralashmaydi`() {
        // Bitta `SharedPreferences` faylida ikki bayroq — kalitlar chalkashsa
        // "eslatmang" tanlovi jimgina fonga o'tishni ham o'chirib qo'yardi.
        val prefs = FakeSharedPreferences()
        val ui = PrefsUiPrefs(prefs)
        ui.screenShareTipEnabled = false
        assertTrue(ui.autoBackgroundOnShare)

        ui.autoBackgroundOnShare = false
        ui.screenShareTipEnabled = true
        assertFalse(ui.autoBackgroundOnShare)
    }
}
