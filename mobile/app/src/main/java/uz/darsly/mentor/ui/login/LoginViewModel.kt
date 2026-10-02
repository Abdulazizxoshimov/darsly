package uz.darsly.mentor.ui.login

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import uz.darsly.mentor.data.api.AuthRepository
import uz.darsly.mentor.data.api.SessionManager

data class LoginUiState(
    val loading: Boolean = false,
    val error: String? = null,
    val success: Boolean = false,
)

@HiltViewModel
class LoginViewModel @Inject constructor(
    private val auth: AuthRepository,
    private val session: SessionManager,
) : ViewModel() {

    private val _state = MutableStateFlow(LoginUiState())
    val state: StateFlow<LoginUiState> = _state.asStateFlow()

    /**
     * NEGA chiqarilgan edik — forma USTIDA ko'rsatiladigan izoh.
     *
     * Eng ko'p uchraydigani `SESSION_REVOKED`: kimdir shu akkaunt bilan boshqa
     * qurilmadan kirgan (bitta akkaunt = bitta sessiya). Bunga umumiy "sessiya
     * tugadi" deb javob berish ustozni parolni yoki internetni ayblashga
     * majburlardi — haqiqiy sabab esa butunlay boshqa.
     */
    val notice: StateFlow<String?> = session.logoutReason
        .map { it.message }
        .stateIn(viewModelScope, SharingStarted.Eagerly, session.logoutReason.value.message)

    /** Izoh ko'rsatildi/yopildi — takror chiqmasin. */
    fun noticeShown() = session.logoutReasonShown()

    fun login(email: String, password: String) {
        if (_state.value.loading) return
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            // AuthRepository tokenlarni shifrlangan saqlagichga yozadi (M1).
            runCatching { auth.login(email, password) }
                .onSuccess { _state.update { it.copy(loading = false, success = true) } }
                .onFailure { t ->
                    // Admin bloki — aniq xabar (generic "xato" emas). Qaror sof [LoginError] da.
                    _state.update { it.copy(loading = false, error = LoginError.messageFor(t)) }
                }
        }
    }
}
