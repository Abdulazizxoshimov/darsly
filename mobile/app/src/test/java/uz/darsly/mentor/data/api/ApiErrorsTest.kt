package uz.darsly.mentor.data.api

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import retrofit2.HttpException
import retrofit2.Response
import java.io.IOException
import java.net.UnknownHostException

/**
 * Xato kodi → o'zbekcha matn.
 *
 * Matnlar `frontend/src/api/api.jsx` dagi `ERROR_UZ` bilan **aynan** bir xil bo'lishi
 * shart (o'sha faylga tegilmagan). Ustoz web'da va ilovada bir xil matnni ko'radi.
 */
class ApiErrorsTest {

    private fun httpError(code: Int, body: String): HttpException =
        HttpException(
            Response.error<Any>(code, body.toResponseBody("application/json".toMediaType())),
        )

    @Test
    fun `barcha backend kodlari xaritalangan`() {
        // Ro'yxat backend manbasidan olingan: internal/pkg/errors/errors.go +
        // api/middleware/{auth,rbac,rate_limit,ratelimit_redis}.go
        val codes = listOf(
            "UNAUTHORIZED", "FORBIDDEN", "NOT_FOUND", "BAD_REQUEST", "CONFLICT",
            "INTERNAL_ERROR", "VALIDATION_ERROR", "RATE_LIMITED", "AUTHZ_UNAVAILABLE",
            "TOKEN_EXPIRED", "TOKEN_INVALID",
        )
        codes.forEach { code ->
            val text = ApiErrors.uz(code)
            org.junit.Assert.assertNotNull("`$code` uchun o'zbekcha matn yo'q", text)
        }
    }

    @Test
    fun `web frontend bilan bir xil matnlar`() {
        assertEquals("Email yoki parol noto‘g‘ri", ApiErrors.uz("UNAUTHORIZED"))
        assertEquals("Bu amalga ruxsatingiz yo‘q", ApiErrors.uz("FORBIDDEN"))
        assertEquals("Topilmadi", ApiErrors.uz("NOT_FOUND"))
        assertEquals("So‘rovda xatolik", ApiErrors.uz("BAD_REQUEST"))
        assertEquals("Bu ma’lumot allaqachon mavjud", ApiErrors.uz("CONFLICT"))
        assertEquals("Serverda xatolik yuz berdi", ApiErrors.uz("INTERNAL_ERROR"))
        assertEquals("Kiritilgan ma’lumotlar noto‘g‘ri", ApiErrors.uz("VALIDATION_ERROR"))
        assertEquals("Juda ko‘p urinish — biroz kuting", ApiErrors.uz("RATE_LIMITED"))
        assertEquals(
            "Xizmat vaqtincha ishlamayapti — birozdan so‘ng urinib ko‘ring",
            ApiErrors.uz("AUTHZ_UNAVAILABLE"),
        )
        assertEquals("Sessiya tugadi — qaytadan kiring", ApiErrors.uz("TOKEN_EXPIRED"))
        assertEquals("Sessiya yaroqsiz — qaytadan kiring", ApiErrors.uz("TOKEN_INVALID"))
    }

    @Test
    fun `nomalum kod null`() {
        assertNull(ApiErrors.uz("NIMADIR_YANGI"))
        assertNull(ApiErrors.uz(null))
    }

    @Test
    fun `konvertdagi kod HTTP statusdan ustun`() {
        // 401 + TOKEN_EXPIRED → "parol xato" emas, "sessiya tugadi".
        val e = httpError(401, """{"code":"TOKEN_EXPIRED","message":"token expired"}""")
        assertEquals("Sessiya tugadi — qaytadan kiring", ApiErrors.humanError(e))
    }

    @Test
    fun `429 rate limit`() {
        val e = httpError(429, """{"code":"RATE_LIMITED","message":"too many requests"}""")
        assertEquals("Juda ko‘p urinish — biroz kuting", ApiErrors.humanError(e))
    }

    @Test
    fun `503 rbac ishlamayapti`() {
        val e = httpError(503, """{"code":"AUTHZ_UNAVAILABLE","message":"authorization unavailable"}""")
        assertEquals(
            "Xizmat vaqtincha ishlamayapti — birozdan so‘ng urinib ko‘ring",
            ApiErrors.humanError(e),
        )
    }

    @Test
    fun `nomalum kodda server xabari korsatiladi`() {
        val e = httpError(400, """{"code":"YANGI_KOD","message":"dars allaqachon boshlangan"}""")
        assertEquals("dars allaqachon boshlangan", ApiErrors.humanError(e))
    }

    @Test
    fun `bosh yoki buzuq tanada HTTP status boyicha zaxira`() {
        assertEquals("Topilmadi", ApiErrors.humanError(httpError(404, "")))
        assertEquals("Serverda xatolik yuz berdi", ApiErrors.humanError(httpError(500, "<html>502</html>")))
    }

    @Test
    fun `tarmoq xatosi alohida matn oladi`() {
        assertEquals(ApiErrors.NETWORK, ApiErrors.humanError(UnknownHostException("app.example")))
        assertEquals(ApiErrors.NETWORK, ApiErrors.humanError(IOException("timeout")))
        assertEquals("Serverga ulanib bo‘lmadi", ApiErrors.NETWORK)
    }

    @Test
    fun `notanish istisno fallback oladi`() {
        assertEquals("Xatolik yuz berdi", ApiErrors.humanError(IllegalStateException()))
    }
}
