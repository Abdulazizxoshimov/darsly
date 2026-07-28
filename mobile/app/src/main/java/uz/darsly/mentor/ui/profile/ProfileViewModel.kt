package uz.darsly.mentor.ui.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.Session
import uz.darsly.mentor.data.api.AuthRepository
import uz.darsly.mentor.data.api.User
import uz.darsly.mentor.data.repo.ProfileRepository

data class ProfileUiState(
    val loading: Boolean = false,
    val user: User? = null,
    /** To'liq ekranli xato — ko'rsatadigan hech narsa bo'lmaganda. */
    val error: String? = null,

    val input: ProfileForm.Input = ProfileForm.Input(),
    val errors: ProfileForm.Errors = ProfileForm.Errors(),
    val showErrors: Boolean = false,
    val saving: Boolean = false,

    val passwordInput: ProfileForm.PasswordInput = ProfileForm.PasswordInput(),
    val passwordErrors: ProfileForm.PasswordErrors = ProfileForm.PasswordErrors(),
    val showPasswordErrors: Boolean = false,
    val changingPassword: Boolean = false,
    /** Parol formasi ostidagi server xatosi (masalan "joriy parol noto'g'ri"). */
    val passwordError: String? = null,
    /**
     * Bir martalik signal: parol muvaffaqiyatli o'zgardi → oyna yopiladi.
     *
     * Ekran buni holatni "taxmin qilish" (masalan "forma bo'shab qoldi") orqali
     * aniqlashi mumkin emas edi: foydalanuvchi maydonlarni o'zi tozalasa ham
     * xuddi shunday ko'rinardi va oyna sababsiz yopilardi.
     */
    val passwordChanged: Boolean = false,

    val loggingOut: Boolean = false,
    /** Bir martalik xabar (snackbar). */
    val notice: String? = null,
) {
    /** Saqlash faqat o'zgarish bo'lganda va forma to'g'ri bo'lganda. */
    val canSave: Boolean
        get() = user != null && errors.isValid && !saving &&
            ProfileForm.toRequest(user, input) != null

    val canChangePassword: Boolean
        get() = !changingPassword && passwordErrors.isValid
}

/**
 * Shaxsiy kabinet: profil ma'lumotlari, parolni o'zgartirish, tizimdan chiqish.
 *
 * `ViewModel` (`AndroidViewModel` emas): bu ekranga `Application` kerak emas —
 * profil keshlanmaydi va `SharedPreferences` bilan ishlamaydi. Shu sabab
 * `@JvmOverloads` hiylasi ham kerak emas (u `(Application)` konstruktori
 * reflektsiya bilan qidirilgani uchun zarur edi).
 */
class ProfileViewModel(
    private val repo: ProfileRepository = ProfileRepository.create(),
) : ViewModel() {

    private val _state = MutableStateFlow(ProfileUiState())
    val state: StateFlow<ProfileUiState> = _state.asStateFlow()

    /** Ekran ochilganda. Qayta chaqirilsa kiritilgan (saqlanmagan) matn yo'qolmaydi. */
    fun start() {
        if (_state.value.loading || _state.value.user != null) return
        load()
    }

    fun load() {
        if (_state.value.loading) return
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            repo.load()
                .onSuccess { user ->
                    val input = ProfileForm.fromUser(user)
                    _state.update {
                        it.copy(
                            loading = false,
                            user = user,
                            input = input,
                            errors = ProfileForm.validate(input),
                            error = null,
                        )
                    }
                }
                .onFailure { t ->
                    // ⚠️ 404 — SESSIYA O'LIK, xato emas.
                    //
                    // Qurilma sinovida (2026-07-28) uchradi: token hali yaroqli
                    // (Redis'da sessiya bor), lekin foydalanuvchi DB'dan
                    // o'chirilgan. `GET /users/me` → 404 va ekranda "Topilmadi /
                    // Qayta urinish" degan BOSHI BERK ko'cha qolardi: qayta
                    // urinish har safar aynan shu 404 ni qaytaradi, chiqish
                    // tugmasi esa shu ekranda yo'q. Yagona chora — ilova
                    // ma'lumotini tozalash edi.
                    //
                    // Endi bunday holatda avtomatik chiqiladi va kirish ekrani
                    // ko'rsatiladi — foydalanuvchi qayta kira oladi.
                    if (t is HttpException && t.code() == 404) {
                        Session.forceLogout()
                        return@onFailure
                    }
                    _state.update { it.copy(loading = false, error = ApiErrors.humanError(t)) }
                }
        }
    }

    fun edit(transform: (ProfileForm.Input) -> ProfileForm.Input) {
        _state.update { st ->
            val input = transform(st.input)
            st.copy(input = input, errors = ProfileForm.validate(input))
        }
    }

    fun save() {
        val current = _state.value
        val user = current.user ?: return
        if (current.saving) return

        val errors = ProfileForm.validate(current.input)
        if (!errors.isValid) {
            _state.update { it.copy(errors = errors, showErrors = true) }
            return
        }
        val req = ProfileForm.toRequest(user, current.input) ?: return

        _state.update { it.copy(saving = true, showErrors = true) }
        viewModelScope.launch {
            repo.update(req)
                .onSuccess { updated ->
                    val input = ProfileForm.fromUser(updated)
                    _state.update {
                        it.copy(
                            saving = false,
                            user = updated,
                            input = input,
                            errors = ProfileForm.validate(input),
                            notice = "Profil saqlandi",
                        )
                    }
                }
                .onFailure { t ->
                    _state.update { it.copy(saving = false, notice = ApiErrors.humanError(t)) }
                }
        }
    }

    // ─── Parolni o'zgartirish ─────────────────────────────────────────────────

    fun editPassword(transform: (ProfileForm.PasswordInput) -> ProfileForm.PasswordInput) {
        _state.update { st ->
            val input = transform(st.passwordInput)
            st.copy(
                passwordInput = input,
                passwordErrors = ProfileForm.validatePassword(input),
                passwordError = null,
            )
        }
    }

    fun changePassword() {
        val current = _state.value
        if (current.changingPassword) return

        val errors = ProfileForm.validatePassword(current.passwordInput)
        if (!errors.isValid) {
            _state.update { it.copy(passwordErrors = errors, showPasswordErrors = true) }
            return
        }

        _state.update { it.copy(changingPassword = true, showPasswordErrors = true, passwordError = null) }
        viewModelScope.launch {
            repo.changePassword(current.passwordInput.current, current.passwordInput.new)
                .onSuccess {
                    _state.update {
                        it.copy(
                            changingPassword = false,
                            passwordInput = ProfileForm.PasswordInput(),
                            passwordErrors = ProfileForm.PasswordErrors(),
                            showPasswordErrors = false,
                            passwordChanged = true,
                            notice = "Parol o'zgartirildi",
                        )
                    }
                }
                .onFailure { t ->
                    _state.update {
                        it.copy(changingPassword = false, passwordError = passwordFailure(t))
                    }
                }
        }
    }

    /**
     * Parol xatosini aniqroq matnga aylantiradi.
     *
     * Backend joriy parol noto'g'ri bo'lganda `UNAUTHORIZED` beradi va umumiy
     * xarita uni **"Email yoki parol noto'g'ri"** deb tarjima qiladi — bu ekranda
     * chalg'ituvchi: bu yerda email umuman so'ralmagan, ustoz esa "email'im
     * noto'g'rimi?" deb o'ylardi. Boshqa barcha kodlar odatiy xaritaga tushadi.
     */
    private fun passwordFailure(t: Throwable): String =
        if (t is HttpException && t.code() == 401) {
            "Joriy parol noto'g'ri"
        } else {
            ApiErrors.humanError(t)
        }

    // ─── Chiqish ──────────────────────────────────────────────────────────────

    /**
     * M3 — chiqish. Serverga xabar beramiz, lekin natijadan qat'i nazar lokal
     * tozalash bajariladi; navigatsiyani `Session.loggedIn` signali qo'zg'atadi.
     */
    fun logout() {
        if (_state.value.loggingOut) return
        _state.update { it.copy(loggingOut = true) }
        viewModelScope.launch {
            AuthRepository.logout()
            _state.update { it.copy(loggingOut = false) }
        }
    }

    /** Parol oynasi yopilgach — signal iste'mol qilindi. */
    fun passwordChangeHandled() = _state.update { it.copy(passwordChanged = false) }

    /** Oyna bekor qilinganda — kiritilgan parollar xotirada qolmasin. */
    fun resetPasswordForm() = _state.update {
        it.copy(
            passwordInput = ProfileForm.PasswordInput(),
            passwordErrors = ProfileForm.PasswordErrors(),
            showPasswordErrors = false,
            passwordError = null,
        )
    }

    fun noticeShown() = _state.update { it.copy(notice = null) }
}
