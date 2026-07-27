package uz.darsly.mentor.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * M42 — semver solishtirish.
 *
 * ENG MUHIM TEST: `1.10.0 > 1.9.0`. Satr solishtirishda `"1.10.0" < "1.9.0"` chiqadi
 * ("1" < "9") va majburiy yangilanish butun ustozlar guruhini noto'g'ri bloklaydi.
 */
class SemverTest {

    private fun cmp(a: String?, b: String?) = Semver.compare(a, b)

    @Test
    fun `1_10_0 katta 1_9_0 dan`() {
        assertTrue(cmp("1.10.0", "1.9.0")!! > 0)
        assertTrue(cmp("1.9.0", "1.10.0")!! < 0)
    }

    @Test
    fun `raqamli solishtirish satr emas`() {
        assertTrue(cmp("2.0.0", "1.99.99")!! > 0)
        assertTrue(cmp("1.0.10", "1.0.9")!! > 0)
        assertTrue(cmp("10.0.0", "9.999.999")!! > 0)
    }

    @Test
    fun `teng versiyalar`() {
        assertEquals(0, cmp("1.2.3", "1.2.3"))
        // Yetishmayotgan qismlar nol bilan to'ldiriladi.
        assertEquals(0, cmp("1.0", "1.0.0"))
        assertEquals(0, cmp("1", "1.0.0"))
        // `v` prefiksi va bo'shliqlar e'tiborga olinmaydi.
        assertEquals(0, cmp(" v1.2.3 ", "1.2.3"))
        // Build metadata solishtirishga ta'sir qilmaydi.
        assertEquals(0, cmp("1.2.3+build7", "1.2.3"))
    }

    @Test
    fun `pre-release relizdan past`() {
        // R0 spike build'i (0.1.0-spike) min_version=1.0.0 da bloklanishi kerak.
        assertTrue(cmp("0.1.0-spike", "1.0.0")!! < 0)
        assertTrue(cmp("1.0.0-rc1", "1.0.0")!! < 0)
        assertTrue(cmp("1.0.0", "1.0.0-rc1")!! > 0)
        assertEquals(0, cmp("1.0.0-rc1", "1.0.0-rc1"))
    }

    @Test
    fun `bosh yoki buzuq qiymat null qaytaradi`() {
        assertNull(cmp("", "1.0.0"))
        assertNull(cmp(null, "1.0.0"))
        assertNull(cmp("1.0.0", ""))
        assertNull(cmp("1.0.0", null))
        assertNull(cmp("abc", "1.0.0"))
        assertNull(cmp("1.x.0", "1.0.0"))
        assertNull(cmp("1..0", "1.0.0"))
        assertNull(cmp("1.0.0.0.0", "1.0.0"))
        assertNull(cmp("-1.0.0", "1.0.0"))
    }

    @Test
    fun `isOlder buzuq qiymatda fail-open`() {
        assertTrue(Semver.isOlder("1.0.0", "1.1.0"))
        assertFalse(Semver.isOlder("1.1.0", "1.0.0"))
        assertFalse(Semver.isOlder("1.0.0", "1.0.0"))
        // Solishtirib bo'lmasa BLOKLAMAYMIZ.
        assertFalse(Semver.isOlder("1.0.0", "buzuq"))
        assertFalse(Semver.isOlder("", ""))
    }
}
