package uz.darsly.mentor.data.api

import io.livekit.android.util.LKLog

/** Kirish/chiqish oqimi (M1 · M3). */
object AuthRepository {

    /** `POST /auth/login` → tokenlarni saqlaydi (shifrlangan preferens). */
    suspend fun login(email: String, password: String): TokenPair {
        val pair = Net.api.login(LoginReq(email, password)).data
            ?: throw IllegalStateException("Server bo'sh javob qaytardi")
        Session.save(pair)
        return pair
    }

    /**
     * `POST /auth/logout` → keyin **har doim** lokal tozalash (M3).
     *
     * Server xato bersa yoki internet bo'lmasa ham foydalanuvchi chiqishi kerak:
     * aks holda "Chiqish" tugmasi ishlamaydigan tugmaga aylanadi. Serverdagi
     * sessiya baribir refresh TTL bilan o'ladi.
     */
    suspend fun logout() {
        val refresh = Session.refreshToken
        if (!refresh.isNullOrBlank()) {
            runCatching { Net.api.logout(LogoutReq(refresh)) }
                .onFailure { LKLog.w(it) { "logout serverda bajarilmadi — lokal tozalanadi" } }
        }
        Session.forceLogout()
    }

    /**
     * Parolni tiklash xatini so'raydi.
     *
     * Backend **har doim 204** qaytaradi — hisob bor-yo'qligidan qat'i nazar
     * (`usecase/auth/auth.go:194-198`: foydalanuvchi topilmasa ham `nil`).
     * Bu ataylab: aks holda kimdir email ro'yxatini shu endpoint orqali
     * tekshirib chiqishi mumkin bo'lardi. Shuning uchun ilova ham
     * "topilmadi" demaydi — "agar bunday email ro'yxatdan o'tgan bo'lsa,
     * xat yuborildi" deydi.
     */
    suspend fun forgotPassword(email: String): Result<Unit> =
        runCatching { Net.api.forgotPassword(ForgotPasswordReq(email)) }

    /**
     * Emaildagi token bilan yangi parol o'rnatadi.
     *
     * Muvaffaqiyatda server **barcha sessiyalarni bekor qiladi** (DB va Redis) —
     * o'g'irlangan sessiya tirik qolmasligi uchun. Shu sabab bu yerda lokal
     * tokenlar ham tozalanadi: aks holda ilova o'lgan token bilan qolib,
     * keyingi so'rovda 401 ko'rardi.
     */
    suspend fun resetPassword(token: String, newPassword: String): Result<Unit> = runCatching {
        Net.api.resetPassword(ResetPasswordReq(token, newPassword))
        // Serverdagi sessiyalar o'ldi — lokal nusxani ham tashlaymiz.
        // `isLoggedIn` tekshiruvi: chiqmagan holatda `forceLogout` navigatsiyani
        // login ekraniga majburlab, tiklash oqimini uzib qo'yardi.
        if (Session.isLoggedIn) Session.forceLogout()
    }
}
