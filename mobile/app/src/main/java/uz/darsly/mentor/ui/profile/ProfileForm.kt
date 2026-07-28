package uz.darsly.mentor.ui.profile

import uz.darsly.mentor.data.api.UpdateProfileReq
import uz.darsly.mentor.data.api.User

/**
 * Shaxsiy kabinet formalarining **sof** mantiqi (Android'siz, JVM testida).
 *
 * Chegaralar `backend/internal/entity/user.go` dagi `validate` teglaridan aynan
 * ko'chirilgan:
 *   `UpdateUserReq.FullName` → `min=2,max=255` · `Timezone` → `max=64` ·
 *   `Language` → `max=8` · `ChangePasswordReq.NewPassword` → `min=8,max=72`
 *
 * Nega klientda takrorlanadi (B-7 bilan bir xil sabab): server xatosi inglizcha
 * va maydon darajasida keladi ("full_name: must be at least 2 characters").
 * Bu yerda xato o'zbekcha, maydon yonida va tarmoqqa chiqmasdan ko'rsatiladi.
 */
object ProfileForm {

    const val NAME_MIN = 2
    const val NAME_MAX = 255
    const val TIMEZONE_MAX = 64
    const val LANGUAGE_MAX = 8

    /**
     * Parol uzunligi — backend `min=8,max=72`.
     *
     * 72 tasodifiy son emas: bcrypt **72 baytdan** keyingisini jimgina tashlaydi,
     * shuning uchun backend ham shu chegarani qo'ygan. Klient uzunroq parolni
     * qabul qilsa, foydalanuvchi kiritgan qismning bir bo'lagi hech qachon
     * tekshirilmasdi.
     */
    const val PASSWORD_MIN = 8
    const val PASSWORD_MAX = 72

    /**
     * TIL VA VAQT MINTAQASI — SOZLAMA EMAS, QAT'IY QIYMAT.
     *
     * Avval ikkalasi ham profilda tanlanardi (tillar ro'yxati, 10 ta mintaqa).
     * Amalda ilova butunlay o'zbekcha va ustozlar O'zbekistonda — ya'ni bu
     * maydonlar hech qachon o'zgarmasdi, lekin har bir ustozga "buni sozlash
     * kerakmi?" degan savol tug'dirardi va noto'g'ri tanlov dars vaqtlarini
     * boshqa mintaqada ko'rsatib qo'yish xavfini yaratardi.
     *
     * Endi qiymatlar shu yerda qotirilgan va profil saqlanganda **har safar**
     * shular yuboriladi: eski hisoblarda qolgan `UTC` ham birinchi saqlashda
     * o'z-o'zidan tuzaladi.
     */
    const val DEFAULT_LANGUAGE = "uz"
    const val DEFAULT_TIMEZONE = "Asia/Tashkent"

    // ─── Profil ma'lumotlari ──────────────────────────────────────────────────

    data class Input(
        val fullName: String = "",
        val timezone: String = "",
        val language: String = "",
    )

    data class Errors(
        val fullName: String? = null,
        val timezone: String? = null,
        val language: String? = null,
    ) {
        val isValid: Boolean get() = fullName == null && timezone == null && language == null
    }

    fun fromUser(user: User): Input = Input(
        fullName = user.fullName,
        timezone = DEFAULT_TIMEZONE,
        language = DEFAULT_LANGUAGE,
    )

    fun validate(input: Input): Errors = Errors(
        fullName = when {
            input.fullName.trim().length < NAME_MIN -> "Ism kamida $NAME_MIN belgidan iborat bo'lsin"
            input.fullName.trim().length > NAME_MAX -> "Ism $NAME_MAX belgidan oshmasin"
            else -> null
        },
        timezone = if (input.timezone.trim().length > TIMEZONE_MAX) {
            "Vaqt mintaqasi $TIMEZONE_MAX belgidan oshmasin"
        } else {
            null
        },
        language = if (input.language.trim().length > LANGUAGE_MAX) {
            "Til kodi $LANGUAGE_MAX belgidan oshmasin"
        } else {
            null
        },
    )

    /**
     * Faqat **o'zgargan** maydonlardan so'rov yasaydi; o'zgarish bo'lmasa `null`.
     *
     * `PUT /users/me` bo'lsa ham semantikasi PATCH'ga o'xshaydi: backend
     * `UpdateUserReq` da har maydon ko'rsatkich va `!= nil` bo'lganda qo'llanadi
     * (`usecase/user`). Ya'ni yuborilmagan maydon tegilmaydi va biz avatar/rangni
     * bexosdan nolga tushirmaymiz.
     */
    fun toRequest(user: User, input: Input): UpdateProfileReq? {
        val fullName = input.fullName.trim().takeIf { it != user.fullName }
        val timezone = input.timezone.trim().takeIf { it != user.timezone.orEmpty() }
        val language = input.language.trim().takeIf { it != user.language.orEmpty() }
        if (fullName == null && timezone == null && language == null) return null
        return UpdateProfileReq(fullName = fullName, timezone = timezone, language = language)
    }

    // ─── Ko'rsatish uchun formatlash ──────────────────────────────────────────

    /**
     * "Ali Valiyev" → "AV"; bitta so'z bo'lsa bitta harf; bo'sh bo'lsa "?".
     *
     * Avatar rasmi yo'q (backend'da fayl yuklash endpoint'i yo'q), shuning uchun
     * harfli doira ishlatiladi — web'dagi bilan bir xil naqsh.
     */
    fun initials(fullName: String): String {
        val parts = fullName.trim().split(Regex("\\s+")).filter { it.isNotBlank() }
        return when {
            parts.isEmpty() -> "?"
            parts.size == 1 -> parts[0].take(1).uppercase()
            else -> (parts[0].take(1) + parts[1].take(1)).uppercase()
        }
    }

    /** Rol kodi → o'zbekcha nom. Notanish rol o'zi ko'rsatiladi (yolg'on tarjima yo'q). */
    fun roleLabel(role: String): String = when (role) {
        "mentor" -> "O'qituvchi"
        "admin" -> "Administrator"
        "student" -> "O'quvchi"
        "guest" -> "Mehmon"
        else -> role
    }

    // ─── Parolni o'zgartirish ─────────────────────────────────────────────────

    data class PasswordInput(
        val current: String = "",
        val new: String = "",
        val confirm: String = "",
    )

    data class PasswordErrors(
        val current: String? = null,
        val new: String? = null,
        val confirm: String? = null,
    ) {
        val isValid: Boolean get() = current == null && new == null && confirm == null
    }

    /**
     * Parol formasi.
     *
     * "Yangi parol joriysi bilan bir xil" tekshiruvi ATAYLAB bor: backend buni
     * rad etmaydi (texnik jihatdan to'g'ri so'rov), lekin foydalanuvchi uchun bu
     * deyarli har doim xato — u parolni o'zgartirmoqchi bo'lgan.
     */
    fun validatePassword(input: PasswordInput): PasswordErrors = PasswordErrors(
        current = if (input.current.isEmpty()) "Joriy parolni kiriting" else null,
        new = when {
            input.new.length < PASSWORD_MIN -> "Yangi parol kamida $PASSWORD_MIN belgidan iborat bo'lsin"
            input.new.length > PASSWORD_MAX -> "Yangi parol $PASSWORD_MAX belgidan oshmasin"
            input.current.isNotEmpty() && input.new == input.current ->
                "Yangi parol joriysidan farq qilsin"
            else -> null
        },
        confirm = when {
            input.confirm.isEmpty() -> "Yangi parolni takrorlang"
            input.confirm != input.new -> "Parollar mos kelmadi"
            else -> null
        },
    )
}
