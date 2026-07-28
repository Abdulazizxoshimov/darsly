package uz.darsly.mentor.ui.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.AuthRepository

/** Tiklash oqimining qadamlari. */
enum class ResetStep {
    /** Email so'raladi → serverga xat yuborish so'rovi. */
    REQUEST,

    /** Xat yuborildi (yoki email topilmadi — farqi bilinmaydi) → token kutilmoqda. */
    SENT,

    /** Token + yangi parol kiritiladi. */
    CONFIRM,

    /** Parol o'zgardi → login ekraniga qaytish. */
    DONE,
}

data class PasswordResetUiState(
    val step: ResetStep = ResetStep.REQUEST,
    val email: String = "",
    val emailError: String? = null,
    val input: PasswordResetForm.Input = PasswordResetForm.Input(),
    val errors: PasswordResetForm.Errors = PasswordResetForm.Errors(),
    val showErrors: Boolean = false,
    val submitting: Boolean = false,
    /** Server xatosi (o'zbekcha). */
    val error: String? = null,
)

/**
 * Parolni tiklash.
 *
 * ## Xavfsizlik jihati, u interfeysga ham ta'sir qiladi
 * `POST /auth/forgot-password` **har doim** 204 qaytaradi — email ro'yxatdan
 * o'tganmi yoki yo'qmi, javob bir xil (`usecase/auth/auth.go:194-198`). Bu
 * ataylab: aks holda endpoint "bu email tizimda bormi?" degan savolga javob
 * beruvchi asbobga aylanardi.
 *
 * Shuning uchun ilova ham **"topilmadi" demaydi**: xabar matni har doim
 * "agar bunday email ro'yxatdan o'tgan bo'lsa, xat yuborildi". Bu chalg'itish
 * emas — bu rost va to'liq: biz ham bilmaymiz.
 */
class PasswordResetViewModel(
    private val auth: AuthRepository = AuthRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(PasswordResetUiState())
    val state: StateFlow<PasswordResetUiState> = _state.asStateFlow()

    fun setEmail(value: String) {
        _state.update { it.copy(email = value, emailError = null, error = null) }
    }

    /** 1-qadam: tiklash xatini so'rash. */
    fun requestReset() {
        val current = _state.value
        if (current.submitting) return

        val emailError = PasswordResetForm.emailError(current.email)
        if (emailError != null) {
            _state.update { it.copy(emailError = emailError) }
            return
        }

        _state.update { it.copy(submitting = true, error = null) }
        viewModelScope.launch {
            auth.forgotPassword(current.email.trim())
                .onSuccess {
                    _state.update { it.copy(submitting = false, step = ResetStep.SENT) }
                }
                .onFailure { t ->
                    // Bu yerga faqat TARMOQ yoki rate-limit xatosi olib keladi:
                    // "email topilmadi" backend'da xato emas.
                    _state.update { it.copy(submitting = false, error = ApiErrors.humanError(t)) }
                }
        }
    }

    /** "Kodni kiritdim" — token formasiga o'tish. */
    fun goToConfirm() {
        _state.update { it.copy(step = ResetStep.CONFIRM, error = null) }
    }

    /** Email formasiga qaytish (masalan email xato yozilgan bo'lsa). */
    fun goToRequest() {
        _state.update { it.copy(step = ResetStep.REQUEST, error = null) }
    }

    fun edit(transform: (PasswordResetForm.Input) -> PasswordResetForm.Input) {
        _state.update { st ->
            val input = transform(st.input)
            st.copy(input = input, errors = PasswordResetForm.validate(input), error = null)
        }
    }

    /** 2-qadam: token + yangi parol. */
    fun confirm() {
        val current = _state.value
        if (current.submitting) return

        val errors = PasswordResetForm.validate(current.input)
        if (!errors.isValid) {
            _state.update { it.copy(errors = errors, showErrors = true) }
            return
        }
        // Validatsiya o'tgan bo'lsa token albatta ajratiladi (aynan shu shart
        // tekshirilgan), lekin `!!` o'rniga aniq tekshiruv: kelajakda qoida
        // o'zgarsa ilova yiqilmasin.
        val token = PasswordResetForm.extractToken(current.input.token)
        if (token == null) {
            _state.update { it.copy(errors = errors, showErrors = true) }
            return
        }

        _state.update { it.copy(submitting = true, showErrors = true, error = null) }
        viewModelScope.launch {
            auth.resetPassword(token, current.input.password)
                .onSuccess {
                    _state.update {
                        it.copy(
                            submitting = false,
                            step = ResetStep.DONE,
                            // Parollar xotirada qolmasin.
                            input = PasswordResetForm.Input(),
                        )
                    }
                }
                .onFailure { t ->
                    _state.update { it.copy(submitting = false, error = ApiErrors.humanError(t)) }
                }
        }
    }
}
