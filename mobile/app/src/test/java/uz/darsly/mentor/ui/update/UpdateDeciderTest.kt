package uz.darsly.mentor.ui.update

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.api.AndroidConfig

/** M42 — `/app-config` javobidan qaror chiqarish. */
class UpdateDeciderTest {

    private fun cfg(
        min: String = "1.0.0",
        latest: String = "1.0.0",
        force: Boolean = false,
        url: String = "https://example.test/app.apk",
    ) = AndroidConfig(
        minVersion = min,
        latestVersion = latest,
        apkUrl = url,
        forceUpdate = force,
        releaseNotes = "izoh",
    )

    @Test
    fun `eski versiya bloklanadi`() {
        val s = UpdateDecider.decide("0.9.0", cfg(min = "1.0.0"))
        assertTrue(s is UpdateState.Required)
        assertEquals("1.0.0", (s as UpdateState.Required).minVersion)
    }

    @Test
    fun `1_9_0 versiya 1_10_0 min bilan bloklanadi`() {
        // Satr solishtirishda "1.9.0" > "1.10.0" bo'lib bloklanmay qolardi.
        assertTrue(UpdateDecider.decide("1.9.0", cfg(min = "1.10.0")) is UpdateState.Required)
    }

    @Test
    fun `yangi versiya bloklanmaydi`() {
        assertEquals(UpdateState.None, UpdateDecider.decide("1.10.0", cfg(min = "1.9.0", latest = "1.10.0")))
        assertEquals(UpdateState.None, UpdateDecider.decide("1.0.0", cfg(min = "1.0.0", latest = "1.0.0")))
    }

    @Test
    fun `yangiroq mavjud bolsa yumshoq eslatma`() {
        val s = UpdateDecider.decide("1.0.0", cfg(min = "1.0.0", latest = "1.2.0"))
        assertTrue(s is UpdateState.Optional)
        assertEquals("1.2.0", (s as UpdateState.Optional).latest)
    }

    @Test
    fun `force_update versiyadan qat'i nazar bloklaydi`() {
        val s = UpdateDecider.decide("9.9.9", cfg(min = "1.0.0", latest = "1.0.0", force = true))
        assertTrue(s is UpdateState.Required)
    }

    @Test
    fun `endpoint javob bermasa ilova ishlashda davom etadi`() {
        // FAIL-OPEN: server nosozligi hamma ustozni bloklamasin.
        assertEquals(UpdateState.None, UpdateDecider.decide("1.0.0", null))
    }

    @Test
    fun `buzuq yoki bosh versiya qiymatlari bloklamaydi`() {
        assertEquals(UpdateState.None, UpdateDecider.decide("1.0.0", cfg(min = "", latest = "")))
        assertEquals(UpdateState.None, UpdateDecider.decide("1.0.0", cfg(min = "buzuq", latest = "yana-buzuq")))
        assertEquals(UpdateState.None, UpdateDecider.decide("", cfg(min = "2.0.0")))
    }
}
