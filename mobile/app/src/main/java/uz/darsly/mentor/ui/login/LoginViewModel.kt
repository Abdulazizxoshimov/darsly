package uz.darsly.mentor.ui.login

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.AuthRepository

data class LoginUiState(
    val loading: Boolean = false,
    val error: String? = null,
    val success: Boolean = false,
)

class LoginViewModel : ViewModel() {

    private val _state = MutableStateFlow(LoginUiState())
    val state: StateFlow<LoginUiState> = _state.asStateFlow()

    fun login(email: String, password: String) {
        if (_state.value.loading) return
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            // AuthRepository tokenlarni shifrlangan saqlagichga yozadi (M1).
            runCatching { AuthRepository.login(email, password) }
                .onSuccess { _state.update { it.copy(loading = false, success = true) } }
                .onFailure { t ->
                    _state.update { it.copy(loading = false, error = ApiErrors.humanError(t)) }
                }
        }
    }
}
