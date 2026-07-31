package uz.darsly.mentor.ui.blocklist

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.BlocklistEntry
import uz.darsly.mentor.data.repo.BlocklistRepository

data class BlocklistUiState(
    val loading: Boolean = false,
    val refreshing: Boolean = false,
    val items: List<BlocklistEntry> = emptyList(),
    /** Ro'yxat o'rnida ko'rsatiladigan xato (birinchi yuklash muvaffaqiyatsiz). */
    val error: String? = null,
    /** Ro'yxat bor, lekin yangilash tarmoqsizlikdan o'tmadi. */
    val offline: Boolean = false,
    /** Hozir o'chirilayotgan yozuv ID'si — o'sha qator bloklanadi. */
    val removingId: String? = null,
    val notice: String? = null,
)

/**
 * Doimiy qora ro'yxat — ko'rish va olib tashlash (№4).
 *
 * ## Nega o'chirish OPTIMISTIK emas
 * Ban'ni olib tashlash — xavfsizlikka tegishli amal. Qator ekrandan darhol
 * yo'qolib, so'rov esa serverda muvaffaqiyatsiz tugasa, ustoz "qaytardim" deb
 * o'ylab yurardi, o'quvchi esa hamon kira olmasdi. Shuning uchun qator faqat
 * server 204 qaytargandan keyin olib tashlanadi.
 */
@HiltViewModel
class BlocklistViewModel @Inject constructor(
    private val repo: BlocklistRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(BlocklistUiState())
    val state: StateFlow<BlocklistUiState> = _state.asStateFlow()

    private var started = false

    /** Ekran birinchi ko'rinishida chaqiriladi (qayta kompozitsiyada takrorlanmaydi). */
    fun start() {
        if (started) return
        started = true
        refresh(userInitiated = false)
    }

    fun refresh(userInitiated: Boolean = true) {
        val current = _state.value
        if (current.loading || current.refreshing) return
        val hasContent = current.items.isNotEmpty()
        _state.update {
            it.copy(loading = !hasContent, refreshing = hasContent && userInitiated, error = null)
        }
        viewModelScope.launch {
            repo.list()
                .onSuccess { items ->
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            items = BlocklistFormat.sortForDisplay(items),
                            error = null,
                            offline = false,
                        )
                    }
                }
                .onFailure { t ->
                    val message = ApiErrors.humanError(t)
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            offline = BlocklistRepository.isOffline(t) && it.items.isNotEmpty(),
                            // Ro'yxat bo'sh bo'lsa xato ekran o'rtasida; aks holda
                            // eski ro'yxat qoladi va xato snackbar bo'lib o'tadi.
                            error = if (it.items.isEmpty()) message else null,
                            notice = if (it.items.isEmpty()) null else message,
                        )
                    }
                }
        }
    }

    fun unblock(entry: BlocklistEntry) {
        if (_state.value.removingId != null) return
        _state.update { it.copy(removingId = entry.id) }
        val name = BlocklistFormat.nameOf(entry)
        viewModelScope.launch {
            repo.unblock(entry.id)
                .onSuccess {
                    _state.update { s ->
                        s.copy(
                            removingId = null,
                            items = s.items.filterNot { it.id == entry.id },
                            notice = "$name qora ro'yxatdan chiqarildi",
                        )
                    }
                }
                .onFailure { t ->
                    _state.update {
                        it.copy(removingId = null, notice = ApiErrors.humanError(t))
                    }
                }
        }
    }

    fun noticeShown() = _state.update { it.copy(notice = null) }
}
