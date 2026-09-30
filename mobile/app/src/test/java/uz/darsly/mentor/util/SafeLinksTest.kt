package uz.darsly.mentor.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/** S2 — serverdan kelgan havola `ACTION_VIEW` ga tekshirilmasdan ketmaydi. */
class SafeLinksTest {

    private val hosts = SafeLinks.hostsOf("https://app.example.uz", "https://app.example.uz/")

    @Test
    fun `faqat https`() {
        assertTrue(SafeLinks.isAllowed("https://files.example.uz/rec.mp4"))
        assertFalse(SafeLinks.isAllowed("http://files.example.uz/rec.mp4"))
        assertFalse(SafeLinks.isAllowed("intent://scan/#Intent;scheme=zxing;end"))
        assertFalse(SafeLinks.isAllowed("file:///sdcard/x.apk"))
        assertFalse(SafeLinks.isAllowed("content://com.android.providers/x"))
        assertFalse(SafeLinks.isAllowed("javascript:alert(1)"))
        assertFalse(SafeLinks.isAllowed(""))
        assertFalse(SafeLinks.isAllowed("not a url"))
    }

    @Test
    fun `debug da http ruxsat`() {
        assertTrue(SafeLinks.isAllowed("http://localhost:8087/x", allowCleartext = true))
        assertFalse(SafeLinks.isAllowed("ftp://localhost/x", allowCleartext = true))
    }

    @Test
    fun `apk faqat oz hostimizdan`() {
        assertEquals(setOf("app.example.uz"), hosts)
        assertTrue(SafeLinks.isAllowed("https://app.example.uz/download/app.apk", hosts))
        assertTrue(SafeLinks.isAllowed("https://APP.example.UZ/download/app.apk", hosts))
        assertFalse(SafeLinks.isAllowed("https://evil.example.com/app.apk", hosts))
        assertFalse(SafeLinks.isAllowed("https://app.example.uz.evil.com/app.apk", hosts))
        assertFalse("http bilan host to'g'ri bo'lsa ham rad", SafeLinks.isAllowed("http://app.example.uz/app.apk", hosts))
    }

    @Test
    fun `yaroqsiz baza host royxatini buzmaydi`() {
        assertEquals(emptySet<String>(), SafeLinks.hostsOf("", "::not a url::"))
    }
}
