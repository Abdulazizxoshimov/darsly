package uz.darsly.mentor.data.api

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import retrofit2.HttpException
import retrofit2.Response

/**
 * 🟡D — maydon darajasidagi validatsiya xatolari.
 *
 * MANBA (taxmin emas): `backend/internal/pkg/validator/validator.go`
 *  · `ValidationError.Error()` → `"field: message"`, ko'pi `"; "` bilan qo'shiladi
 *  · `fieldMessage(e)` → quyidagi inglizcha shablonlar
 *
 * Avval bu tafsilot yo'qolardi va ustoz faqat "Kiritilgan ma'lumotlar noto'g'ri"
 * ni ko'rardi — qaysi maydon, nima uchun ekani ma'lum emas edi.
 */
class ApiFieldErrorsTest {

    private fun httpError(code: Int, body: String): HttpException =
        HttpException(Response.error<Any>(code, body.toResponseBody("application/json".toMediaType())))

    @Test
    fun `maydon nomi va qoida ozbekchaga ogiriladi`() {
        assertEquals(
            "Dars nomi: kamida 2 belgi bo'lishi kerak",
            ApiErrors.fieldError("title: must be at least 2 characters"),
        )
        assertEquals(
            "Tavsif: 2000 belgidan oshmasligi kerak",
            ApiErrors.fieldError("description: must be at most 2000 characters"),
        )
        assertEquals(
            "Parol: kamida 4 belgi bo'lishi kerak",
            ApiErrors.fieldError("passcode: must be at least 4 characters"),
        )
        assertEquals(
            "Dars nomi: to'ldirilishi shart",
            ApiErrors.fieldError("title: field is required"),
        )
        assertEquals(
            "Email: email formati noto'g'ri",
            ApiErrors.fieldError("email: invalid email format"),
        )
    }

    @Test
    fun `raqamli maydonda belgi soz ishlatilmaydi`() {
        // Backend "characters" deydi, lekin `duration_min` — daqiqa, belgi emas.
        val text = ApiErrors.fieldError("duration_min: must be at least 5 characters")!!
        assertEquals("Davomiylik: kamida 5 bo'lishi kerak", text)
        assertTrue("'belgi' so'zi bo'lmasin", !text.contains("belgi"))
        assertEquals(
            "Davomiylik: 1440 dan oshmasligi kerak",
            ApiErrors.fieldError("duration_min: must be at most 1440 characters"),
        )
    }

    @Test
    fun `bir nechta maydon xatosi birga korsatiladi`() {
        assertEquals(
            "Dars nomi: kamida 2 belgi bo'lishi kerak; Parol: kamida 4 belgi bo'lishi kerak",
            ApiErrors.fieldError("title: must be at least 2 characters; passcode: must be at least 4 characters"),
        )
    }

    @Test
    fun `notanish maydon yoki qoida tarjima qilinmaydi`() {
        // Chala tarjima ko'rsatishdan ko'ra umumiy matnga tushish xavfsizroq.
        assertNull(ApiErrors.fieldError("recurrence: something weird"))
        assertNull(ApiErrors.fieldError("title: must smell nice"))
        assertNull(ApiErrors.fieldError("umuman ikki nuqtasiz matn"))
        assertNull(ApiErrors.fieldError(""))
        assertNull(ApiErrors.fieldError(null))
        // Bitta bo'lak tanilmasa — HAMMASI rad etiladi (yarim inglizcha chiqmasin).
        assertNull(ApiErrors.fieldError("title: must be at least 2 characters; xyz: bar"))
    }

    @Test
    fun `humanError endi aniq maydonni korsatadi`() {
        // Jonli serverdan olingan haqiqiy javob (2026-07-26, `POST /lessons`).
        val msg = ApiErrors.humanError(
            httpError(400, """{"code":"BAD_REQUEST","message":"title: must be at least 2 characters"}"""),
        )
        assertEquals("Dars nomi: kamida 2 belgi bo'lishi kerak", msg)
    }

    @Test
    fun `tarjima qilib bolmasa umumiy ozbekcha matn qaytadi`() {
        val msg = ApiErrors.humanError(
            httpError(422, """{"code":"VALIDATION_ERROR","message":"weird_field: something"}"""),
        )
        assertEquals(ApiErrors.uz("VALIDATION_ERROR"), msg)
        assertTrue("inglizcha xom matn chiqmasin", !msg.contains("weird_field"))
    }

    @Test
    fun `boshqa kodlar oldingidek ishlaydi`() {
        // Regressiya: 🟡D tuzatishi 1-blok xulqini buzmasligi kerak.
        assertEquals(
            ApiErrors.uz("RATE_LIMITED"),
            ApiErrors.humanError(httpError(429, """{"code":"RATE_LIMITED","message":"slow down"}""")),
        )
        assertEquals(
            ApiErrors.uz("FORBIDDEN"),
            ApiErrors.humanError(httpError(403, """{"code":"FORBIDDEN","message":"nope"}""")),
        )
    }
}
