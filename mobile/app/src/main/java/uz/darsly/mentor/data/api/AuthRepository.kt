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
}
