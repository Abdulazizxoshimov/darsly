package uz.darsly.mentor.ui.login

import uz.darsly.mentor.data.api.AdminNotAllowedException
import uz.darsly.mentor.data.api.ApiErrors

/**
 * Login xatosini foydalanuvchi matniga aylantirish — **sof** (JVM testida).
 *
 * ## Nega ajratildi
 * Admin bloki (mobil FAQAT mentorlar uchun) aniq, o'zbekcha sabab ko'rsatishi
 * kerak — umumiy "xato" emas. Bu shart `LoginViewModel` ichida edi va agar
 * u tushib qolsa yoki `ApiErrors.humanError` ni admin holatida ham chaqirsa,
 * admin "parolim noto'g'ri" deb o'ylab qayta-qayta urinardi. Ajratilib
 * qulflandi.
 */
object LoginError {

    fun messageFor(t: Throwable): String =
        if (t is AdminNotAllowedException) t.message ?: ApiErrors.humanError(t)
        else ApiErrors.humanError(t)
}
