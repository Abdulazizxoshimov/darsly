package uz.darsly.mentor.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Chatda fayl ulashish qoidalari (№15).
 *
 * Server baribir tekshiradi, lekin bu tekshiruvlar 20 MB'ni sekin mobil
 * internetda YUKLAB BO'LGANDAN keyin emas, TANLASH paytida ishlaydi — ustoz
 * uchun farqi bir necha daqiqa.
 */
class ChatUploadTest {

    // ─── Nom tozalash ─────────────────────────────────────────────────────────

    @Test
    fun `nomdagi yol tashlanadi`() {
        // Server ham shunday qiladi (`filepath.Base`), lekin nom tasdiqlash
        // oynasida DARHOL ko'rsatiladi — u yerda ham toza bo'lishi kerak.
        assertEquals("x.png", ChatUpload.sanitizeName("../../etc/x.png"))
        assertEquals("x.png", ChatUpload.sanitizeName("""C:\Users\ali\x.png"""))
        assertEquals("uy_ishi.pdf", ChatUpload.sanitizeName("  uy_ishi.pdf  "))
    }

    @Test
    fun `bosh nom zaxira nom bilan almashtiriladi`() {
        // Ba'zi provayderlar `DISPLAY_NAME` bermaydi — bo'sh nom bilan
        // multipart yuborilsa server "file name is required" berardi.
        assertEquals(ChatUpload.FALLBACK_NAME, ChatUpload.sanitizeName(null))
        assertEquals(ChatUpload.FALLBACK_NAME, ChatUpload.sanitizeName(""))
        assertEquals(ChatUpload.FALLBACK_NAME, ChatUpload.sanitizeName("   "))
        assertEquals(ChatUpload.FALLBACK_NAME, ChatUpload.sanitizeName("/"))
    }

    // ─── Kengaytma ────────────────────────────────────────────────────────────

    @Test
    fun `kengaytma kichik harfga keltiriladi`() {
        assertEquals("pdf", ChatUpload.extensionOf("Konspekt.PDF"))
        assertEquals("jpeg", ChatUpload.extensionOf("photo.JPEG"))
    }

    @Test
    fun `kengaytmasiz nom bosh satr beradi`() {
        assertEquals("", ChatUpload.extensionOf("README"))
        // Yashirin fayl (`.gitignore`) — "gitignore" kengaytma EMAS.
        assertEquals("", ChatUpload.extensionOf(".gitignore"))
        assertEquals("", ChatUpload.extensionOf("nuqta."))
    }

    @Test
    fun `ruxsat etilgan turlar backend royxati bilan bir xil`() {
        // Manba: `backend/internal/usecase/chat/file.go: allowedChatFiles`.
        val expected = setOf(
            "jpg", "jpeg", "png", "gif", "webp", "pdf",
            "docx", "xlsx", "pptx", "doc", "xls", "ppt", "txt", "csv",
        )
        assertEquals(expected, ChatUpload.ALLOWED_EXTENSIONS)
    }

    @Test
    fun `ruxsat etilmagan turlar rad etiladi`() {
        // Aynan bular xavfli/ma'nosiz: `.apk`/`.exe` — tarqatish, `.html` —
        // XSS urinishi, `.mp4` — chatga emas, yozuvlar bo'limiga tegishli.
        for (name in listOf("virus.apk", "setup.exe", "page.html", "dars.mp4", "arxiv.zip")) {
            assertFalse(name, ChatUpload.isAllowed(name))
            assertNotNull(name, ChatUpload.validate(name, 1024))
        }
    }

    // ─── Hajm ─────────────────────────────────────────────────────────────────

    @Test
    fun `chegara 20 MB`() {
        assertEquals(20L * 1024 * 1024, ChatUpload.MAX_BYTES)
    }

    @Test
    fun `aynan chegaradagi fayl otadi`() {
        // "≤ 20 MB" — serverdagi shart ham aynan shunday (`size > Max` → xato).
        // Bitta bayt farq bilan yaroqli faylni rad etish eng bezovta qiluvchi
        // xato turi bo'lardi.
        assertNull(ChatUpload.validate("dars.pdf", ChatUpload.MAX_BYTES))
    }

    @Test
    fun `chegaradan katta fayl HAJMI bilan rad etiladi`() {
        val message = ChatUpload.validate("dars.pdf", ChatUpload.MAX_BYTES + 1)
        assertNotNull(message)
        // Xabar HAQIQIY hajmni aytishi kerak: "juda katta" o'zi ustozga nima
        // qilishni aytmaydi, "24,0 MB · eng ko'pi 20 MB" esa aytadi.
        assertTrue(message!!, message.contains("20 MB"))
        assertTrue(message, message.contains("MB)"))
    }

    @Test
    fun `bosh fayl rad etiladi`() {
        assertNotNull(ChatUpload.validate("dars.pdf", 0))
        assertNotNull(ChatUpload.validate("dars.pdf", -1))
    }

    @Test
    fun `tekshiruv tartibi - avval hajm keyin tur`() {
        // Bir vaqtning o'zida ikki muammo bo'lsa bitta ANIQ jumla ko'rsatiladi.
        // Hajm birinchi: u aniqroq va tez tuzatiladigan sabab.
        val message = ChatUpload.validate("virus.apk", ChatUpload.MAX_BYTES + 1)
        assertTrue(message!!, message.contains("20 MB"))
    }

    @Test
    fun `yarokli fayl otkaziladi`() {
        assertNull(ChatUpload.validate("uy_ishi.pdf", 184_320))
        assertNull(ChatUpload.validate("rasm.PNG", 1))
        assertNull(ChatUpload.validate("jadval.xlsx", 5_000_000))
    }

    // ─── Ko'rinish ────────────────────────────────────────────────────────────

    @Test
    fun `hajm yorligi hech qachon bosh emas`() {
        // Kartochkada bo'sh joy "hajm noma'lum" degan savol tug'dirardi.
        assertEquals("1 MB", ChatUpload.sizeLabel(1024L * 1024))
        assertEquals("0 B", ChatUpload.sizeLabel(0))
    }

    @Test
    fun `belgi fayl turiga qarab tanlanadi`() {
        assertEquals("🖼", ChatUpload.icon("rasm.png"))
        assertEquals("📕", ChatUpload.icon("kitob.pdf"))
        assertEquals("📊", ChatUpload.icon("jadval.xlsx"))
        assertEquals("📊", ChatUpload.icon("royxat.csv"))
        assertEquals("📎", ChatUpload.icon("nomalum.txt"))
    }
}
