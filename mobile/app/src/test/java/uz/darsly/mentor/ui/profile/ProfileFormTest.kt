package uz.darsly.mentor.ui.profile

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.api.User

/**
 * Shaxsiy kabinet formalari.
 *
 * Chegaralar `backend/internal/entity/user.go` bilan bir xil bo'lishi shart:
 * klient yumshoqroq bo'lsa ustoz inglizcha server xatosini ko'radi, qattiqroq
 * bo'lsa haqiqatda ruxsat etilgan qiymatni kirita olmaydi.
 */
class ProfileFormTest {

    private fun user(
        fullName: String = "Ali Valiyev",
        timezone: String? = "Asia/Tashkent",
        language: String? = "uz",
    ) = User(
        id = "u1",
        email = "ali@darsly.uz",
        fullName = fullName,
        role = "mentor",
        timezone = timezone,
        language = language,
    )

    // ─── Validatsiya ──────────────────────────────────────────────────────────

    @Test
    fun `togri forma xatosiz`() {
        assertTrue(ProfileForm.validate(ProfileForm.fromUser(user())).isValid)
    }

    @Test
    fun `qisqa ism rad etiladi`() {
        val errors = ProfileForm.validate(ProfileForm.Input(fullName = "A"))
        assertNotNull(errors.fullName)
        assertFalse(errors.isValid)
    }

    @Test
    fun `chegaradagi ism qabul qilinadi`() {
        // AYNAN `min=2` — chegara bir belgiga surilsa (`< 3`) ismi "Ma" bo'lgan
        // foydalanuvchi profilini saqlay olmasdi.
        assertNull(ProfileForm.validate(ProfileForm.Input(fullName = "Ma")).fullName)
        assertNull(ProfileForm.validate(ProfileForm.Input(fullName = "x".repeat(255))).fullName)
        assertNotNull(ProfileForm.validate(ProfileForm.Input(fullName = "x".repeat(256))).fullName)
    }

    @Test
    fun `ism atrofidagi boshliqlar hisobga olinmaydi`() {
        assertNotNull(ProfileForm.validate(ProfileForm.Input(fullName = "  A  ")).fullName)
        assertNull(ProfileForm.validate(ProfileForm.Input(fullName = "  Ali  ")).fullName)
    }

    @Test
    fun `uzun mintaqa va til rad etiladi`() {
        assertNotNull(ProfileForm.validate(ProfileForm.Input("Ali", timezone = "x".repeat(65))).timezone)
        assertNotNull(ProfileForm.validate(ProfileForm.Input("Ali", language = "x".repeat(9))).language)
    }

    // ─── Diff ─────────────────────────────────────────────────────────────────

    @Test
    fun `ozgarish bolmasa sorov yasalmaydi`() {
        val u = user()
        assertNull(ProfileForm.toRequest(u, ProfileForm.fromUser(u)))
    }

    @Test
    fun `faqat ozgargan maydon yuboriladi`() {
        val u = user()
        val input = ProfileForm.fromUser(u).copy(language = "ru")
        val req = ProfileForm.toRequest(u, input)!!
        assertEquals("ru", req.language)
        // Avatar/rang so'rovda umuman yo'q (modelda ham yo'q), ism va mintaqa
        // esa `null` — server ularga tegmaydi.
        assertNull(req.fullName)
        assertNull(req.timezone)
    }

    // ─── Til va mintaqa endi sozlama emas ─────────────────────────────────────

    @Test
    fun `mintaqa va til har doim Toshkent va ozbekcha`() {
        // Sozlamadan olib tashlangani uchun forma serverdagi qiymatga
        // QARAMAYDI: eski hisobdagi "UTC" ham shu yerdan tuzaladi.
        val input = ProfileForm.fromUser(user(timezone = "Europe/Berlin", language = "en"))
        assertEquals("Asia/Tashkent", input.timezone)
        assertEquals("uz", input.language)
    }

    @Test
    fun `eski hisobdagi UTC birinchi saqlashda tuzatiladi`() {
        val u = user(timezone = "UTC", language = "en")
        val req = ProfileForm.toRequest(u, ProfileForm.fromUser(u))
        assertEquals("Asia/Tashkent", req!!.timezone)
        assertEquals("uz", req.language)
    }

    // ─── Parol formasi ────────────────────────────────────────────────────────

    @Test
    fun `togri parol formasi xatosiz`() {
        val errors = ProfileForm.validatePassword(
            ProfileForm.PasswordInput(current = "eski123456", new = "yangi12345", confirm = "yangi12345"),
        )
        assertTrue(errors.isValid)
    }

    @Test
    fun `qisqa yangi parol rad etiladi`() {
        val errors = ProfileForm.validatePassword(
            ProfileForm.PasswordInput(current = "eski123456", new = "1234567", confirm = "1234567"),
        )
        assertNotNull(errors.new)
    }

    @Test
    fun `72 belgidan uzun parol rad etiladi`() {
        // bcrypt 72 baytdan keyingisini jimgina tashlaydi — backend ham shu
        // chegarani qo'ygan. Klient uzunroqni qabul qilsa, foydalanuvchi
        // kiritgan qismning bir bo'lagi hech qachon tekshirilmasdi.
        val long = "x".repeat(73)
        assertNotNull(ProfileForm.validatePassword(
            ProfileForm.PasswordInput(current = "eski123456", new = long, confirm = long),
        ).new)
    }

    @Test
    fun `mos kelmagan takror rad etiladi`() {
        val errors = ProfileForm.validatePassword(
            ProfileForm.PasswordInput(current = "eski123456", new = "yangi12345", confirm = "yangi1234"),
        )
        assertNotNull(errors.confirm)
    }

    @Test
    fun `yangi parol joriysi bilan bir xil bolsa rad etiladi`() {
        // Backend buni rad etmaydi (texnik jihatdan to'g'ri so'rov), lekin
        // foydalanuvchi uchun bu deyarli har doim xato.
        val errors = ProfileForm.validatePassword(
            ProfileForm.PasswordInput(current = "birxil12345", new = "birxil12345", confirm = "birxil12345"),
        )
        assertNotNull(errors.new)
    }

    @Test
    fun `bosh joriy parol rad etiladi`() {
        val errors = ProfileForm.validatePassword(
            ProfileForm.PasswordInput(current = "", new = "yangi12345", confirm = "yangi12345"),
        )
        assertNotNull(errors.current)
    }

    // ─── Ko'rsatish ───────────────────────────────────────────────────────────

    @Test
    fun `bosh harflar togri hisoblanadi`() {
        assertEquals("AV", ProfileForm.initials("Ali Valiyev"))
        assertEquals("A", ProfileForm.initials("Ali"))
        assertEquals("AV", ProfileForm.initials("  ali   valiyev  "))
        assertEquals("?", ProfileForm.initials("   "))
        // Uch so'zli ismda faqat birinchi ikkitasi — doiraga uchta harf sig'maydi.
        assertEquals("AB", ProfileForm.initials("Ali Bek Valiyev"))
    }

    @Test
    fun `notanish rol ozi korsatiladi`() {
        assertEquals("O'qituvchi", ProfileForm.roleLabel("mentor"))
        assertEquals("supervisor", ProfileForm.roleLabel("supervisor"))
    }
}
