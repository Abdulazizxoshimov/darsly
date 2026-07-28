package uz.darsly.mentor.data.repo

import uz.darsly.mentor.data.api.ChangePasswordReq
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.Net
import uz.darsly.mentor.data.api.UpdateProfileReq
import uz.darsly.mentor.data.api.User

/**
 * Shaxsiy kabinet: profilni o'qish/tahrirlash va parolni o'zgartirish.
 *
 * KESH YO'Q — ataylab. Profil kamdan-kam ochiladi va **eskirgan** ma'lumot
 * ko'rsatish bu yerda zararli: ustoz "ismim o'zgardi" deb o'ylab yurishi mumkin,
 * aslida so'rov yiqilgan bo'lsa. Darslar ro'yxatidan farqi shu — u ustozga
 * koridorda kerak, profil esa yo'q.
 */
class ProfileRepository(private val api: DarslyApi) {

    suspend fun load(): Result<User> = runCatching {
        api.profile().data ?: throw IllegalStateException("Server bo'sh javob qaytardi")
    }

    /**
     * Profilni yangilaydi va serverning **qaytargan** obyektini beradi.
     *
     * Lokal nusxa emas, aynan server javobi ishlatiladi: backend qiymatlarni
     * normallashtirishi mumkin (masalan bo'sh `timezone` ni default bilan
     * to'ldirishi), va ekranda haqiqiy saqlangan holat turishi kerak.
     */
    suspend fun update(req: UpdateProfileReq): Result<User> = runCatching {
        api.updateProfile(req).data ?: throw IllegalStateException("Server bo'sh javob qaytardi")
    }

    /**
     * Parolni o'zgartiradi. Joriy parol noto'g'ri bo'lsa server **401** beradi
     * (`UNAUTHORIZED` → "Email yoki parol noto'g'ri") — ViewModel bu holatni
     * maydon yonidagi aniqroq matnga aylantiradi.
     */
    suspend fun changePassword(currentPassword: String, newPassword: String): Result<Unit> =
        runCatching { api.changePassword(ChangePasswordReq(currentPassword, newPassword)) }

    companion object {
        fun create(): ProfileRepository = ProfileRepository(Net.api)
    }
}
