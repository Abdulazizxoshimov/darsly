package uz.darsly.mentor.e2e

import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.ComposeTestRule
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.espresso.Espresso
import androidx.test.ext.junit.runners.AndroidJUnit4
import dagger.hilt.android.testing.HiltAndroidRule
import dagger.hilt.android.testing.HiltAndroidTest
import org.junit.Assert.fail
import org.junit.FixMethodOrder
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.junit.runners.MethodSorters
import uz.darsly.mentor.MainActivity

/**
 * TO'LIQ-FLOW E2E (mobil UI -> real backend -> DB), ulangan qurilmada.
 * Talab: backend ishlab turishi (BuildConfig.API_BASE_URL) + seed admin.
 * Metodlar NOMLANISH tartibida ishlaydi (holat sessiya saqlanishi orqali oqadi):
 * t1 xato login -> t2 login -> t3 tablar -> t4 dars+xona -> t5 chiqish.
 */
@HiltAndroidTest
@RunWith(AndroidJUnit4::class)
@FixMethodOrder(MethodSorters.NAME_ASCENDING)
class MentorJourneyTest {

    @get:Rule(order = 0)
    val hilt = HiltAndroidRule(this)

    @get:Rule(order = 1)
    val compose = createAndroidComposeRule<MainActivity>()

    companion object {
        // t2 yaratadi, keyingi metodlar ishlatadi (sessiya storage orqali ham oqadi).
        lateinit var mentor: TestBackend.Creds
        var lessonTitle: String = ""
    }

    // ── yordamchilar ─────────────────────────────────────────────────────────
    private fun waitText(text: String, timeoutMs: Long = 20_000) {
        compose.waitUntil(timeoutMs) {
            compose.onAllNodesWithText(text, substring = true).fetchSemanticsNodes().isNotEmpty()
        }
    }

    private fun textGone(text: String, timeoutMs: Long = 20_000) {
        compose.waitUntil(timeoutMs) {
            compose.onAllNodesWithText(text, substring = true).fetchSemanticsNodes().isEmpty()
        }
    }

    private fun typeInto(placeholder: String, value: String) {
        compose.onNode(hasSetTextAction() and hasText(placeholder)).performTextInput(value)
    }

    private fun clickText(text: String, index: Int = 0) {
        compose.onAllNodesWithText(text)[index].performClick()
    }

    private fun uiLogin(email: String, password: String) {
        waitText("Kirish")
        typeInto("Email", email)
        typeInto("Parol", password)
        clickText("Kirish")
    }

    private fun ComposeTestRule.has(text: String): Boolean =
        onAllNodesWithText(text, substring = true).fetchSemanticsNodes().isNotEmpty()

    // ── oqim ─────────────────────────────────────────────────────────────────

    @Test
    fun t1_notogriParol_kirolmaydi() {
        waitText("Kirish") // login ekrani (ilova toza holatdan boshlanadi)
        typeInto("Email", "yoq+user@darsly.uz")
        typeInto("Parol", "notogri999")
        clickText("Kirish")
        // Dashboard OCHILMASLIGI kerak.
        try {
            waitText("Hali dars yo'q", 6_000)
            fail("Noto'g'ri parol bilan kirib bo'ldi!")
        } catch (_: androidx.compose.ui.test.ComposeTimeoutException) {
            // kutilgan: login ekranida qolamiz
        }
    }

    @Test
    fun t2_login_dashboardOchiladi() {
        mentor = TestBackend.createMentor()
        uiLogin(mentor.email, mentor.password)
        waitText("Darslar")          // dashboard sarlavhasi
        waitText("Hali dars yo'q")   // yangi mentor — bo'sh holat (GET /lessons -> DB)
    }

    @Test
    fun t3_tablar_jadvalArxivKabinet() {
        waitText("Darslar")
        clickText("Jadval"); waitText("Jadval")
        clickText("Arxiv"); waitText("Arxiv")
        clickText("Kabinet")
        waitText("Tizimdan chiqish") // profil ekrani belgilari
        waitText(mentor.email)       // GET /users/me -> DB (email ko'rinadi)
        clickText("Darslar")
    }

    @Test
    fun t4_darsYaratish_xona_yakunlash() {
        waitText("Darslar")
        lessonTitle = "E2E dars ${System.currentTimeMillis().toString(36)}"
        clickText("Dars yaratish")
        waitText("Dars nomi")
        typeInto("Dars nomi", lessonTitle)
        clickText("Boshlash")
        // POST /lessons (DB) -> xona ekrani.
        waitText("Hali hech kim qo'shilmadi", 30_000)
        // Chiqish -> "Darsni yakunlaysizmi?" -> Yakunlash.
        clickText("Chiqish")
        waitText("Darsni yakunlaysizmi?")
        clickText("Yakunlash")
        // Dashboardga qaytadi, dars ro'yxatda (DB dan qayta o'qiladi).
        waitText(lessonTitle, 30_000)
    }

    @Test
    fun t5_tizimdanChiqish() {
        waitText("Darslar")
        clickText("Kabinet")
        waitText("Tizimdan chiqish")
        clickText("Tizimdan chiqish")
        waitText("Tizimdan chiqasizmi?")
        clickText("Chiqish") // dialog tasdiqlash
        waitText("Kirish")   // login ekraniga qaytdi
    }
}
