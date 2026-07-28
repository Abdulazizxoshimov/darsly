package uz.darsly.mentor.ui.auth

import uz.darsly.mentor.ui.profile.ProfileForm

/**
 * Parolni tiklash formalarining **sof** mantiqi.
 *
 * ## Oqim (backend `usecase/auth/auth.go`)
 * 1. `POST /auth/forgot-password` → serverda tasodifiy token yaratiladi, uning
 *    SHA-256 hash'i bazaga yoziladi, **xom** token esa emaildagi havolaga
 *    qo'yiladi: `<FRONTEND_BASE_URL>/reset-password?token=<token>`.
 * 2. `POST /auth/reset-password` → token + yangi parol.
 *
 * ## Nega mobil ilovada token qo'lda kiritiladi
 * Havola **web sahifasiga** ishora qiladi (`frontend`), ilovaga emas: backend
 * uni `FRONTEND_BASE_URL` dan quradi va bu qiymat serverda. Ilovaga chuqur
 * havola (deep link) qo'shish backend'ni o'zgartirishni talab qilardi, mobil
 * uchun esa **zarur emas** — ustoz web'da parolni tiklab, ilovada yangi parol
 * bilan kirishi mumkin.
 *
 * Shunga qaramay ilovada tiklash ekrani bor va u havolaning **o'zini** ham
 * qabul qiladi ([extractToken]): ustoz emaildagi havolani nusxalab yopishtirsa
 * ham ishlaydi, tokenni qo'lda ajratib olishi shart emas.
 */
object PasswordResetForm {

    /**
     * Emaildagi qiymatdan tokenni ajratadi.
     *
     * Uch xil kiritmani qabul qiladi, chunki foydalanuvchi uchalasini ham
     * yopishtirishi mumkin:
     *  · sof token — `9f1c…`
     *  · to'liq havola — `https://app.example/reset-password?token=9f1c…`
     *  · qo'shimcha parametrli havola — `…?token=9f1c…&lang=uz`
     *
     * Bo'sh yoki tokensiz havola uchun `null`.
     */
    fun extractToken(raw: String): String? {
        val text = raw.trim()
        if (text.isEmpty()) return null

        val marker = "token="
        val idx = text.indexOf(marker)
        if (idx < 0) {
            // Havola emas — sof token deb qabul qilamiz. Ichida bo'shliq bo'lsa
            // bu token emas (foydalanuvchi boshqa narsa yopishtirgan).
            return text.takeIf { !it.contains(' ') && !it.contains('/') }
        }
        val after = text.substring(idx + marker.length)
        // Keyingi parametrgacha (`&`) yoki fragmentgacha (`#`).
        val end = after.indexOfFirst { it == '&' || it == '#' }
        val token = if (end < 0) after else after.substring(0, end)
        return token.trim().takeIf { it.isNotEmpty() }
    }

    // ─── 1-qadam: emailni so'rash ─────────────────────────────────────────────

    /**
     * Email shaklini yuzaki tekshiradi.
     *
     * ATAYLAB soddalashtirilgan: to'liq RFC 5322 tekshiruvi klientda ma'nosiz —
     * haqiqiy tasdiq baribir emailning yetib borishi bilan bo'ladi. Maqsad
     * faqat aniq xatoni (bo'sh maydon, `@` yo'q) tarmoqqa chiqmasdan ushlash.
     */
    fun emailError(email: String): String? {
        val value = email.trim()
        return when {
            value.isEmpty() -> "Emailni kiriting"
            !value.contains('@') || value.startsWith('@') || value.endsWith('@') ->
                "Email formati noto'g'ri"
            else -> null
        }
    }

    // ─── 2-qadam: token + yangi parol ─────────────────────────────────────────

    data class Input(
        val token: String = "",
        val password: String = "",
        val confirm: String = "",
    )

    data class Errors(
        val token: String? = null,
        val password: String? = null,
        val confirm: String? = null,
    ) {
        val isValid: Boolean get() = token == null && password == null && confirm == null
    }

    /**
     * Chegaralar `entity.ResetPasswordReq` dan: `token` majburiy,
     * `new_password` — `min=8,max=72` ([ProfileForm.PASSWORD_MIN] bilan bir xil
     * manba, shuning uchun u qayta yozilmaydi).
     */
    fun validate(input: Input): Errors = Errors(
        token = if (extractToken(input.token) == null) {
            "Emaildagi havolani yoki kodni kiriting"
        } else {
            null
        },
        password = when {
            input.password.length < ProfileForm.PASSWORD_MIN ->
                "Parol kamida ${ProfileForm.PASSWORD_MIN} belgidan iborat bo'lsin"
            input.password.length > ProfileForm.PASSWORD_MAX ->
                "Parol ${ProfileForm.PASSWORD_MAX} belgidan oshmasin"
            else -> null
        },
        confirm = when {
            input.confirm.isEmpty() -> "Parolni takrorlang"
            input.confirm != input.password -> "Parollar mos kelmadi"
            else -> null
        },
    )
}
