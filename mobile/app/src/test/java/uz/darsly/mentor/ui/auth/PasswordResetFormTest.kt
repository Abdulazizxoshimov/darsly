package uz.darsly.mentor.ui.auth

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Parolni tiklash formasi.
 *
 * Eng qimmatli qism — [PasswordResetForm.extractToken]: ustoz emaildagi
 * **havolani** yopishtiradi, sof tokenni emas. Ajratish noto'g'ri ishlasa
 * server "yaroqsiz token" deydi va ustoz nima qilishni bilmay qoladi.
 */
class PasswordResetFormTest {

    private val token = "9f1c4e2a-7b3d-4f5e-8a9c-1d2e3f4a5b6c"

    // ─── Tokenni ajratish ─────────────────────────────────────────────────────

    @Test
    fun `sof token ozgarishsiz qaytadi`() {
        assertEquals(token, PasswordResetForm.extractToken(token))
        assertEquals(token, PasswordResetForm.extractToken("  $token  "))
    }

    @Test
    fun `toliq havoladan token ajratiladi`() {
        val url = "https://app.194.163.139.242.sslip.io/reset-password?token=$token"
        assertEquals(token, PasswordResetForm.extractToken(url))
    }

    @Test
    fun `qoshimcha parametrli havoladan token ajratiladi`() {
        val url = "https://app.example/reset-password?token=$token&lang=uz"
        assertEquals(token, PasswordResetForm.extractToken(url))
    }

    @Test
    fun `token birinchi parametr bolmasa ham topiladi`() {
        val url = "https://app.example/reset-password?lang=uz&token=$token"
        assertEquals(token, PasswordResetForm.extractToken(url))
    }

    @Test
    fun `fragment tokendan keyin kesiladi`() {
        val url = "https://app.example/reset-password?token=$token#top"
        assertEquals(token, PasswordResetForm.extractToken(url))
    }

    @Test
    fun `tokensiz havola null qaytaradi`() {
        assertNull(PasswordResetForm.extractToken("https://app.example/reset-password"))
        // Bo'sh `token=` — server uni baribir rad etardi, lekin foydalanuvchi
        // "yaroqsiz token" o'rniga aniqroq "havolani kiriting" xabarini oladi.
        assertNull(PasswordResetForm.extractToken("https://app.example/x?token="))
    }

    @Test
    fun `bosh yoki manosiz kiritma null qaytaradi`() {
        assertNull(PasswordResetForm.extractToken(""))
        assertNull(PasswordResetForm.extractToken("   "))
        // Bo'shliqli matn — foydalanuvchi boshqa narsa yopishtirgan.
        assertNull(PasswordResetForm.extractToken("salom dunyo"))
    }

    // ─── Email ────────────────────────────────────────────────────────────────

    @Test
    fun `email tekshiruvi`() {
        assertNull(PasswordResetForm.emailError("ali@darsly.uz"))
        assertNotNull(PasswordResetForm.emailError(""))
        assertNotNull(PasswordResetForm.emailError("   "))
        assertNotNull(PasswordResetForm.emailError("alidarsly.uz"))
        assertNotNull(PasswordResetForm.emailError("@darsly.uz"))
        assertNotNull(PasswordResetForm.emailError("ali@"))
    }

    // ─── Yangi parol formasi ──────────────────────────────────────────────────

    @Test
    fun `togri forma xatosiz`() {
        val errors = PasswordResetForm.validate(
            PasswordResetForm.Input(token = token, password = "yangi12345", confirm = "yangi12345"),
        )
        assertTrue(errors.isValid)
    }

    @Test
    fun `havola yopishtirilsa ham forma togri`() {
        // Foydalanuvchi tokenni qo'lda ajratib olishi SHART emas.
        val errors = PasswordResetForm.validate(
            PasswordResetForm.Input(
                token = "https://app.example/reset-password?token=$token",
                password = "yangi12345",
                confirm = "yangi12345",
            ),
        )
        assertNull(errors.token)
    }

    @Test
    fun `tokensiz forma rad etiladi`() {
        val errors = PasswordResetForm.validate(
            PasswordResetForm.Input(token = "", password = "yangi12345", confirm = "yangi12345"),
        )
        assertNotNull(errors.token)
    }

    @Test
    fun `qisqa parol rad etiladi`() {
        val errors = PasswordResetForm.validate(
            PasswordResetForm.Input(token = token, password = "1234567", confirm = "1234567"),
        )
        assertNotNull(errors.password)
    }

    @Test
    fun `mos kelmagan takror rad etiladi`() {
        val errors = PasswordResetForm.validate(
            PasswordResetForm.Input(token = token, password = "yangi12345", confirm = "boshqa12345"),
        )
        assertNotNull(errors.confirm)
    }
}
